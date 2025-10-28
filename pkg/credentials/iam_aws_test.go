//go:build !windows
// +build !windows

/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2017 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package credentials

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"
)

const credsRespTmpl = `{
  "Code": "Success",
  "Type": "AWS-HMAC",
  "AccessKeyId" : "accessKey",
  "SecretAccessKey" : "secret",
  "Token" : "token",
  "Expiration" : "%s",
  "LastUpdated" : "2009-11-23T0:00:00Z"
}`

const credsFailRespTmpl = `{
  "Code": "ErrorCode",
  "Message": "ErrorMsg",
  "LastUpdated": "2009-11-23T0:00:00Z"
}`

const credsRespEcsTaskTmpl = `{
	"AccessKeyId" : "accessKey",
	"SecretAccessKey" : "secret",
	"Token" : "token",
	"Expiration" : "%s"
}`

const credsRespStsImpl = `<AssumeRoleWithWebIdentityResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/">
<AssumeRoleWithWebIdentityResult>
  <SubjectFromWebIdentityToken>amzn1.account.AF6RHO7KZU5XRVQJGXK6HB56KR2A</SubjectFromWebIdentityToken>
  <Audience>client.5498841531868486423.1548@apps.example.com</Audience>
  <AssumedRoleUser>
	<Arn>arn:aws:sts::123456789012:assumed-role/FederatedWebIdentityRole/app1</Arn>
	<AssumedRoleId>AROACLKWSDQRAOEXAMPLE:app1</AssumedRoleId>
  </AssumedRoleUser>
  <Credentials>
	<SessionToken>token</SessionToken>
	<SecretAccessKey>secret</SecretAccessKey>
	<Expiration>%s</Expiration>
	<AccessKeyId>accessKey</AccessKeyId>
  </Credentials>
  <Provider>www.amazon.com</Provider>
</AssumeRoleWithWebIdentityResult>
<ResponseMetadata>
  <RequestId>ad4156e9-bce1-11e2-82e6-6b6efEXAMPLE</RequestId>
</ResponseMetadata>
</AssumeRoleWithWebIdentityResponse>`

func initTestFailServer() *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Not allowed", http.StatusBadRequest)
	}))
	return server
}

func initTestServerNoRoles() *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(""))
	}))
	return server
}

// Instance Metadata Service with V1 disabled.
func initIMDSv2Server(expireOn string, failAssume bool) *httptest.Server {
	imdsToken := "IMDSTokenabc123=="
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(r.URL.Path)
		fmt.Println(r.Method)
		if r.URL.Path == "/latest/api/token" && r.Method == "PUT" {
			ttlHeader := r.Header.Get("X-aws-ec2-metadata-token-ttl-seconds")
			ttl, err := strconv.ParseInt(ttlHeader, 10, 32)
			if err != nil || ttl < 0 || ttl > 21600 {
				http.Error(w, "", http.StatusBadRequest)
				return
			}
			w.Header().Set("X-Aws-Ec2-Metadata-Token-Ttl-Seconds", ttlHeader)
			w.Write([]byte(imdsToken))
			return
		}
		token := r.Header.Get("X-aws-ec2-metadata-token")
		if token != imdsToken {
			http.Error(w, r.URL.Path, http.StatusUnauthorized)
			return
		}

		switch r.URL.Path {
		case "/latest/meta-data/iam/security-credentials/":
			fmt.Fprintln(w, "RoleName")
		case "/latest/meta-data/iam/security-credentials/RoleName":
			if failAssume {
				fmt.Fprint(w, credsFailRespTmpl)
			} else {
				fmt.Fprintf(w, credsRespTmpl, expireOn)
			}
		default:
			http.Error(w, "bad request", http.StatusBadRequest)
		}
	}))
	return server
}

func initEcsTaskTestServer(expireOn string) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, credsRespEcsTaskTmpl, expireOn)
	}))

	return server
}

func initStsTestServer(expireOn string) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		required := []string{"RoleArn", "RoleSessionName", "WebIdentityToken", "Version"}
		for _, field := range required {
			if _, ok := r.Form[field]; !ok {
				http.Error(w, fmt.Sprintf("%s missing", field), http.StatusBadRequest)
				return
			}
		}

		fmt.Fprintf(w, credsRespStsImpl, expireOn)
	}))

	return server
}

func TestIAMMalformedEndpoint(t *testing.T) {
	creds := NewIAM("%%%%")
	_, err := creds.GetWithContext(defaultCredContext)
	if err == nil {
		t.Fatal("Unexpected should fail here")
	}
}

func TestIAMFailServer(t *testing.T) {
	server := initTestFailServer()
	defer server.Close()

	creds := NewIAM(server.URL)

	_, err := creds.GetWithContext(defaultCredContext)
	if err == nil {
		t.Fatal("Unexpected should fail here")
	}
	if err.Error() != "400 Bad Request" {
		t.Fatalf("Expected '400 Bad Request', got %s", err)
	}
}

func TestIAMNoRoles(t *testing.T) {
	server := initTestServerNoRoles()
	defer server.Close()

	creds := NewIAM(server.URL)
	_, err := creds.GetWithContext(defaultCredContext)
	if err == nil {
		t.Fatal("Unexpected should fail here")
	}
	if err.Error() != "No IAM roles attached to this EC2 service" {
		t.Fatalf("Expected 'No IAM roles attached to this EC2 service', got %s", err)
	}
}

func TestIAM(t *testing.T) {
	server := initIMDSv2Server("2014-12-16T01:51:37Z", false)
	defer server.Close()

	p := &IAM{
		Endpoint: server.URL,
	}

	creds, err := p.RetrieveWithCredContext(defaultCredContext)
	if err != nil {
		t.Fatal(err)
	}

	if creds.AccessKeyID != "accessKey" {
		t.Errorf("Expected \"accessKey\", got %s", creds.AccessKeyID)
	}

	if creds.SecretAccessKey != "secret" {
		t.Errorf("Expected \"secret\", got %s", creds.SecretAccessKey)
	}

	if creds.SessionToken != "token" {
		t.Errorf("Expected \"token\", got %s", creds.SessionToken)
	}

	if !p.IsExpired() {
		t.Error("Expected creds to be expired.")
	}
}

func TestIAMFailAssume(t *testing.T) {
	server := initIMDSv2Server("2014-12-16T01:51:37Z", true)
	defer server.Close()

	p := &IAM{
		Endpoint: server.URL,
	}

	_, err := p.RetrieveWithCredContext(defaultCredContext)
	if err == nil {
		t.Fatal("Unexpected success, should fail")
	}
	if err.Error() != "ErrorMsg" {
		t.Errorf("Expected \"ErrorMsg\", got %s", err)
	}
}

func TestIAMIsExpired(t *testing.T) {
	server := initIMDSv2Server("2014-12-16T01:51:37Z", false)
	defer server.Close()

	p := &IAM{
		Endpoint: server.URL,
	}
	p.CurrentTime = func() time.Time {
		return time.Date(2014, 12, 15, 21, 26, 0, 0, time.UTC)
	}

	if !p.IsExpired() {
		t.Error("Expected creds to be expired before retrieve.")
	}

	_, err := p.RetrieveWithCredContext(defaultCredContext)
	if err != nil {
		t.Fatal(err)
	}

	if p.IsExpired() {
		t.Error("Expected creds to not be expired after retrieve.")
	}

	p.CurrentTime = func() time.Time {
		return time.Date(3014, 12, 15, 21, 26, 0, 0, time.UTC)
	}

	if !p.IsExpired() {
		t.Error("Expected creds to be expired when curren time has changed")
	}
}

func TestEcsTask(t *testing.T) {
	server := initEcsTaskTestServer("2014-12-16T01:51:37Z")
	defer server.Close()
	p := &IAM{
		Endpoint: server.URL,
	}
	t.Setenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI", "/v2/credentials?id=task_credential_id")
	creds, err := p.RetrieveWithCredContext(defaultCredContext)
	os.Unsetenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI")
	if err != nil {
		t.Errorf("Unexpected failure %s", err)
	}
	if creds.AccessKeyID != "accessKey" {
		t.Errorf("Expected \"accessKey\", got %s", creds.AccessKeyID)
	}

	if creds.SecretAccessKey != "secret" {
		t.Errorf("Expected \"secret\", got %s", creds.SecretAccessKey)
	}

	if creds.SessionToken != "token" {
		t.Errorf("Expected \"token\", got %s", creds.SessionToken)
	}

	if !p.IsExpired() {
		t.Error("Expected creds to be expired.")
	}
}

func TestEcsTaskFullURI(t *testing.T) {
	server := initEcsTaskTestServer("2014-12-16T01:51:37Z")
	defer server.Close()
	p := &IAM{}
	t.Setenv("AWS_CONTAINER_CREDENTIALS_FULL_URI",
		fmt.Sprintf("%s%s", server.URL, "/v2/credentials?id=task_credential_id"))
	creds, err := p.RetrieveWithCredContext(defaultCredContext)
	os.Unsetenv("AWS_CONTAINER_CREDENTIALS_FULL_URI")
	if err != nil {
		t.Errorf("Unexpected failure %s", err)
	}
	if creds.AccessKeyID != "accessKey" {
		t.Errorf("Expected \"accessKey\", got %s", creds.AccessKeyID)
	}

	if creds.SecretAccessKey != "secret" {
		t.Errorf("Expected \"secret\", got %s", creds.SecretAccessKey)
	}

	if creds.SessionToken != "token" {
		t.Errorf("Expected \"token\", got %s", creds.SessionToken)
	}

	if !p.IsExpired() {
		t.Error("Expected creds to be expired.")
	}
}

func TestSts(t *testing.T) {
	server := initStsTestServer("2014-12-16T01:51:37Z")
	defer server.Close()
	p := &IAM{
		Endpoint: server.URL,
	}

	f, err := os.CreateTemp(t.TempDir(), "minio-go")
	if err != nil {
		t.Errorf("Unexpected failure %s", err)
	}
	defer os.Remove(f.Name())
	f.Write([]byte("token"))
	f.Close()

	t.Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", f.Name())
	t.Setenv("AWS_ROLE_ARN", "arn:aws:sts::123456789012:assumed-role/FederatedWebIdentityRole/app1")
	creds, err := p.RetrieveWithCredContext(defaultCredContext)
	os.Unsetenv("AWS_WEB_IDENTITY_TOKEN_FILE")
	os.Unsetenv("AWS_ROLE_ARN")
	if err != nil {
		t.Errorf("Unexpected failure %s", err)
	}
	if creds.AccessKeyID != "accessKey" {
		t.Errorf("Expected \"accessKey\", got %s", creds.AccessKeyID)
	}

	if creds.SecretAccessKey != "secret" {
		t.Errorf("Expected \"secret\", got %s", creds.SecretAccessKey)
	}

	if creds.SessionToken != "token" {
		t.Errorf("Expected \"token\", got %s", creds.SessionToken)
	}

	if !p.IsExpired() {
		t.Error("Expected creds to be expired.")
	}
}

func TestStsCn(t *testing.T) {
	server := initStsTestServer("2014-12-16T01:51:37Z")
	defer server.Close()
	p := &IAM{
		Endpoint: server.URL,
	}

	f, err := os.CreateTemp(t.TempDir(), "minio-go")
	if err != nil {
		t.Errorf("Unexpected failure %s", err)
	}
	defer os.Remove(f.Name())
	f.Write([]byte("token"))
	f.Close()

	t.Setenv("AWS_REGION", "cn-northwest-1")
	t.Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", f.Name())
	t.Setenv("AWS_ROLE_ARN", "arn:aws:sts::123456789012:assumed-role/FederatedWebIdentityRole/app1")
	creds, err := p.RetrieveWithCredContext(defaultCredContext)
	os.Unsetenv("AWS_WEB_IDENTITY_TOKEN_FILE")
	os.Unsetenv("AWS_ROLE_ARN")
	if err != nil {
		t.Errorf("Unexpected failure %s", err)
	}
	if creds.AccessKeyID != "accessKey" {
		t.Errorf("Expected \"accessKey\", got %s", creds.AccessKeyID)
	}

	if creds.SecretAccessKey != "secret" {
		t.Errorf("Expected \"secret\", got %s", creds.SecretAccessKey)
	}

	if creds.SessionToken != "token" {
		t.Errorf("Expected \"token\", got %s", creds.SessionToken)
	}

	if !p.IsExpired() {
		t.Error("Expected creds to be expired.")
	}
}

func TestIMDSv1Blocked(t *testing.T) {
	server := initIMDSv2Server("2014-12-16T01:51:37Z", false)
	p := &IAM{
		Endpoint: server.URL,
	}
	_, err := p.RetrieveWithCredContext(defaultCredContext)
	if err != nil {
		t.Errorf("Unexpected IMDSv2 failure %s", err)
	}
}

func TestIAMCustomExpiryWindow(t *testing.T) {
	server := initIMDSv2Server("2014-12-16T01:51:37Z", false)
	defer server.Close()

	// Test with custom expiry window of 5 minutes
	customWindow := 5 * time.Minute
	p := &IAM{
		Endpoint:     server.URL,
		ExpiryWindow: customWindow,
	}

	// Set a known current time for predictable testing
	p.CurrentTime = func() time.Time {
		return time.Date(2014, 12, 15, 21, 0, 0, 0, time.UTC)
	}

	// retrieve credentials - triggers initial expiration calculation
	creds, err := p.RetrieveWithCredContext(defaultCredContext)
	if err != nil {
		t.Fatal(err)
	}

	if creds.AccessKeyID != "accessKey" {
		t.Errorf("Expected \"accessKey\", got %s", creds.AccessKeyID)
	}

	// Verify that the custom expiry window was used
	// The expiration time should be: original expiration - custom window
	// Original: 2014-12-16T01:51:37Z
	// Custom window: 5 minutes
	// Expected expiry: 2014-12-16T01:46:37Z
	expectedExpiry := time.Date(2014, 12, 16, 1, 46, 37, 0, time.UTC)
	if !p.expiration.Equal(expectedExpiry) {
		t.Errorf("Expected expiration %v, got %v", expectedExpiry, p.expiration)
	}

	// Credentials should not be expired at current time (2014-12-15 21:00:00)
	if p.IsExpired() {
		t.Error("Expected creds to not be expired with custom window.")
	}

	// Move time forward to just before expiry
	p.CurrentTime = func() time.Time {
		return time.Date(2014, 12, 16, 1, 46, 0, 0, time.UTC)
	}
	if p.IsExpired() {
		t.Error("Expected creds to not be expired yet.")
	}

	// Move time forward past the custom expiry window
	p.CurrentTime = func() time.Time {
		return time.Date(2014, 12, 16, 1, 47, 0, 0, time.UTC)
	}
	if !p.IsExpired() {
		t.Error("Expected creds to be expired after custom window.")
	}
}

func TestIAMDefaultExpiryWindow(t *testing.T) {
	server := initIMDSv2Server("2014-12-16T01:51:37Z", false)
	defer server.Close()

	// Test with default expiry window (should use 80% rule)
	p := &IAM{
		Endpoint:     server.URL,
		ExpiryWindow: DefaultExpiryWindow,
	}

	p.CurrentTime = func() time.Time {
		return time.Date(2014, 12, 15, 21, 0, 0, 0, time.UTC)
	}

	// retrieve credentials - triggers initial expiration calculation
	creds, err := p.RetrieveWithCredContext(defaultCredContext)
	if err != nil {
		t.Fatal(err)
	}

	if creds.AccessKeyID != "accessKey" {
		t.Errorf("Expected \"accessKey\", got %s", creds.AccessKeyID)
	}

	// With default window, expiry should be calculated as:
	// expiration - (80% of time until expiration)
	// Time from current (2014-12-15 21:00:00) to expiration (2014-12-16 01:51:37) = 4h 51m 37s = 17497s
	// 80% of that = 13997.6s ≈ 3h 53m 17.6s
	// So expiry should be around: 2014-12-16 01:51:37 - 3h 53m 17.6s = 2014-12-15 21:58:19.4
	// We'll check it's expired before the actual expiration time
	originalExpiration := time.Date(2014, 12, 16, 1, 51, 37, 0, time.UTC)
	if !p.expiration.Before(originalExpiration) {
		t.Errorf("Expected expiration to be before original expiration time with default window")
	}

	// Credentials should not be expired at current time
	if p.IsExpired() {
		t.Error("Expected creds to not be expired initially.")
	}
}

func TestNewIAMWithConfig(t *testing.T) {
	server := initIMDSv2Server("2014-12-16T01:51:37Z", false)
	defer server.Close()

	// Test NewIAMWithConfig with custom expiry window
	customWindow := 10 * time.Minute
	config := IAMConfig{
		ExpiryWindow: customWindow,
	}

	creds := NewIAMWithConfig(server.URL, config)
	if creds == nil {
		t.Fatal("Expected non-nil credentials")
	}

	// Verify the provider is properly configured
	provider, ok := creds.provider.(*IAM)
	if !ok {
		t.Fatal("Expected provider to be *IAM")
	}

	if provider.Endpoint != server.URL {
		t.Errorf("Expected endpoint %s, got %s", server.URL, provider.Endpoint)
	}

	if provider.ExpiryWindow != customWindow {
		t.Errorf("Expected expiry window %v, got %v", customWindow, provider.ExpiryWindow)
	}

	// Set a known current time
	provider.CurrentTime = func() time.Time {
		return time.Date(2014, 12, 15, 21, 0, 0, 0, time.UTC)
	}

	// Retrieve credentials and verify custom window is applied
	value, err := creds.GetWithContext(defaultCredContext)
	if err != nil {
		t.Fatal(err)
	}

	if value.AccessKeyID != "accessKey" {
		t.Errorf("Expected \"accessKey\", got %s", value.AccessKeyID)
	}

	// Verify expiration is set with custom window
	expectedExpiry := time.Date(2014, 12, 16, 1, 41, 37, 0, time.UTC)
	if !provider.expiration.Equal(expectedExpiry) {
		t.Errorf("Expected expiration %v, got %v", expectedExpiry, provider.expiration)
	}
}

func TestIAMZeroExpiryWindowUsesDefault(t *testing.T) {
	server := initIMDSv2Server("2014-12-16T01:51:37Z", false)
	defer server.Close()

	// Test that zero expiry window falls back to default
	p := &IAM{
		Endpoint:     server.URL,
		ExpiryWindow: 0, // Explicitly set to zero
	}

	p.CurrentTime = func() time.Time {
		return time.Date(2014, 12, 15, 21, 0, 0, 0, time.UTC)
	}

	_, err := p.RetrieveWithCredContext(defaultCredContext)
	if err != nil {
		t.Fatal(err)
	}

	// After retrieve, ExpiryWindow should be set to DefaultExpiryWindow
	if p.ExpiryWindow != DefaultExpiryWindow {
		t.Errorf("Expected ExpiryWindow to be DefaultExpiryWindow, got %v", p.ExpiryWindow)
	}

	// Verify default behavior (80% rule) is applied
	originalExpiration := time.Date(2014, 12, 16, 1, 51, 37, 0, time.UTC)
	if !p.expiration.Before(originalExpiration) {
		t.Error("Expected expiration to be before original expiration time with default window")
	}
}
