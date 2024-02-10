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

		if r.URL.Path == "/latest/meta-data/iam/security-credentials/" {
			fmt.Fprintln(w, "RoleName")
		} else if r.URL.Path == "/latest/meta-data/iam/security-credentials/RoleName" {
			if failAssume {
				fmt.Fprint(w, credsFailRespTmpl)
			} else {
				fmt.Fprintf(w, credsRespTmpl, expireOn)
			}
		} else {
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
	_, err := creds.Get()
	if err == nil {
		t.Fatal("Unexpected should fail here")
	}
}

func TestIAMFailServer(t *testing.T) {
	server := initTestFailServer()
	defer server.Close()

	creds := NewIAM(server.URL)

	_, err := creds.Get()
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
	_, err := creds.Get()
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
		Client:   http.DefaultClient,
		Endpoint: server.URL,
	}

	creds, err := p.Retrieve()
	if err != nil {
		t.Fatal(err)
	}

	if "accessKey" != creds.AccessKeyID {
		t.Errorf("Expected \"accessKey\", got %s", creds.AccessKeyID)
	}

	if "secret" != creds.SecretAccessKey {
		t.Errorf("Expected \"secret\", got %s", creds.SecretAccessKey)
	}

	if "token" != creds.SessionToken {
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
		Client:   http.DefaultClient,
		Endpoint: server.URL,
	}

	_, err := p.Retrieve()
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
		Client:   http.DefaultClient,
		Endpoint: server.URL,
	}
	p.CurrentTime = func() time.Time {
		return time.Date(2014, 12, 15, 21, 26, 0, 0, time.UTC)
	}

	if !p.IsExpired() {
		t.Error("Expected creds to be expired before retrieve.")
	}

	_, err := p.Retrieve()
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
		Client:   http.DefaultClient,
		Endpoint: server.URL,
	}
	os.Setenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI", "/v2/credentials?id=task_credential_id")
	creds, err := p.Retrieve()
	os.Unsetenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI")
	if err != nil {
		t.Errorf("Unexpected failure %s", err)
	}
	if "accessKey" != creds.AccessKeyID {
		t.Errorf("Expected \"accessKey\", got %s", creds.AccessKeyID)
	}

	if "secret" != creds.SecretAccessKey {
		t.Errorf("Expected \"secret\", got %s", creds.SecretAccessKey)
	}

	if "token" != creds.SessionToken {
		t.Errorf("Expected \"token\", got %s", creds.SessionToken)
	}

	if !p.IsExpired() {
		t.Error("Expected creds to be expired.")
	}
}

func TestEcsTaskFullURI(t *testing.T) {
	server := initEcsTaskTestServer("2014-12-16T01:51:37Z")
	defer server.Close()
	p := &IAM{
		Client: http.DefaultClient,
	}
	os.Setenv("AWS_CONTAINER_CREDENTIALS_FULL_URI",
		fmt.Sprintf("%s%s", server.URL, "/v2/credentials?id=task_credential_id"))
	creds, err := p.Retrieve()
	os.Unsetenv("AWS_CONTAINER_CREDENTIALS_FULL_URI")
	if err != nil {
		t.Errorf("Unexpected failure %s", err)
	}
	if "accessKey" != creds.AccessKeyID {
		t.Errorf("Expected \"accessKey\", got %s", creds.AccessKeyID)
	}

	if "secret" != creds.SecretAccessKey {
		t.Errorf("Expected \"secret\", got %s", creds.SecretAccessKey)
	}

	if "token" != creds.SessionToken {
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
		Client:   http.DefaultClient,
		Endpoint: server.URL,
	}

	f, err := os.CreateTemp("", "minio-go")
	if err != nil {
		t.Errorf("Unexpected failure %s", err)
	}
	defer os.Remove(f.Name())
	f.Write([]byte("token"))
	f.Close()

	os.Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", f.Name())
	os.Setenv("AWS_ROLE_ARN", "arn:aws:sts::123456789012:assumed-role/FederatedWebIdentityRole/app1")
	creds, err := p.Retrieve()
	os.Unsetenv("AWS_WEB_IDENTITY_TOKEN_FILE")
	os.Unsetenv("AWS_ROLE_ARN")
	if err != nil {
		t.Errorf("Unexpected failure %s", err)
	}
	if "accessKey" != creds.AccessKeyID {
		t.Errorf("Expected \"accessKey\", got %s", creds.AccessKeyID)
	}

	if "secret" != creds.SecretAccessKey {
		t.Errorf("Expected \"secret\", got %s", creds.SecretAccessKey)
	}

	if "token" != creds.SessionToken {
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
		Client:   http.DefaultClient,
		Endpoint: server.URL,
	}

	f, err := os.CreateTemp("", "minio-go")
	if err != nil {
		t.Errorf("Unexpected failure %s", err)
	}
	defer os.Remove(f.Name())
	f.Write([]byte("token"))
	f.Close()

	os.Setenv("AWS_REGION", "cn-northwest-1")
	os.Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", f.Name())
	os.Setenv("AWS_ROLE_ARN", "arn:aws:sts::123456789012:assumed-role/FederatedWebIdentityRole/app1")
	creds, err := p.Retrieve()
	os.Unsetenv("AWS_WEB_IDENTITY_TOKEN_FILE")
	os.Unsetenv("AWS_ROLE_ARN")
	if err != nil {
		t.Errorf("Unexpected failure %s", err)
	}
	if "accessKey" != creds.AccessKeyID {
		t.Errorf("Expected \"accessKey\", got %s", creds.AccessKeyID)
	}

	if "secret" != creds.SecretAccessKey {
		t.Errorf("Expected \"secret\", got %s", creds.SecretAccessKey)
	}

	if "token" != creds.SessionToken {
		t.Errorf("Expected \"token\", got %s", creds.SessionToken)
	}

	if !p.IsExpired() {
		t.Error("Expected creds to be expired.")
	}
}

func TestIMDSv1Blocked(t *testing.T) {
	server := initIMDSv2Server("2014-12-16T01:51:37Z", false)
	p := &IAM{
		Client:   http.DefaultClient,
		Endpoint: server.URL,
	}
	_, err := p.Retrieve()
	if err != nil {
		t.Errorf("Unexpected IMDSv2 failure %s", err)
	}
}
