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
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestFileAWS(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("\"/bin/cat\": file does not exist")
	}
	os.Clearenv()

	creds := NewFileAWSCredentials("credentials.sample", "")
	credValues, err := creds.GetWithContext(defaultCredContext)
	if err != nil {
		t.Fatal(err)
	}

	if credValues.AccessKeyID != "accessKey" {
		t.Errorf("Expected 'accessKey', got %s'", credValues.AccessKeyID)
	}
	if credValues.SecretAccessKey != "secret" {
		t.Errorf("Expected 'secret', got %s'", credValues.SecretAccessKey)
	}
	if credValues.SessionToken != "token" {
		t.Errorf("Expected 'token', got %s'", credValues.SessionToken)
	}

	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "credentials.sample")
	creds = NewFileAWSCredentials("", "")
	credValues, err = creds.GetWithContext(defaultCredContext)
	if err != nil {
		t.Fatal(err)
	}

	if credValues.AccessKeyID != "accessKey" {
		t.Errorf("Expected 'accessKey', got %s'", credValues.AccessKeyID)
	}
	if credValues.SecretAccessKey != "secret" {
		t.Errorf("Expected 'secret', got %s'", credValues.SecretAccessKey)
	}
	if credValues.SessionToken != "token" {
		t.Errorf("Expected 'token', got %s'", credValues.SessionToken)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(wd, "credentials.sample"))
	creds = NewFileAWSCredentials("", "")
	credValues, err = creds.GetWithContext(defaultCredContext)
	if err != nil {
		t.Fatal(err)
	}

	if credValues.AccessKeyID != "accessKey" {
		t.Errorf("Expected 'accessKey', got %s'", credValues.AccessKeyID)
	}
	if credValues.SecretAccessKey != "secret" {
		t.Errorf("Expected 'secret', got %s'", credValues.SecretAccessKey)
	}
	if credValues.SessionToken != "token" {
		t.Errorf("Expected 'token', got %s'", credValues.SessionToken)
	}

	os.Clearenv()
	t.Setenv("AWS_PROFILE", "no_token")

	creds = NewFileAWSCredentials("credentials.sample", "")
	credValues, err = creds.GetWithContext(defaultCredContext)
	if err != nil {
		t.Fatal(err)
	}

	if credValues.AccessKeyID != "accessKey" {
		t.Errorf("Expected 'accessKey', got %s'", credValues.AccessKeyID)
	}
	if credValues.SecretAccessKey != "secret" {
		t.Errorf("Expected 'secret', got %s'", credValues.SecretAccessKey)
	}

	os.Clearenv()

	creds = NewFileAWSCredentials("credentials.sample", "no_token")
	credValues, err = creds.GetWithContext(defaultCredContext)
	if err != nil {
		t.Fatal(err)
	}

	if credValues.AccessKeyID != "accessKey" {
		t.Errorf("Expected 'accessKey', got %s'", credValues.AccessKeyID)
	}
	if credValues.SecretAccessKey != "secret" {
		t.Errorf("Expected 'secret', got %s'", credValues.SecretAccessKey)
	}

	creds = NewFileAWSCredentials("credentials-non-existent.sample", "no_token")
	_, err = creds.GetWithContext(defaultCredContext)
	if !os.IsNotExist(err) {
		t.Errorf("Expected open non-existent.json: no such file or directory, got %s", err)
	}
	if !creds.IsExpired() {
		t.Error("Should be expired if not loaded")
	}

	os.Clearenv()

	creds = NewFileAWSCredentials("credentials.sample", "with_process")
	credValues, err = creds.GetWithContext(defaultCredContext)
	if err != nil {
		t.Fatal(err)
	}

	if credValues.AccessKeyID != "accessKey" {
		t.Errorf("Expected 'accessKey', got %s'", credValues.AccessKeyID)
	}
	if credValues.SecretAccessKey != "secret" {
		t.Errorf("Expected 'secret', got %s'", credValues.SecretAccessKey)
	}
	if credValues.SessionToken != "token" {
		t.Errorf("Expected 'token', got %s'", credValues.SessionToken)
	}
	if creds.IsExpired() {
		t.Error("Should not be expired")
	}
}

func writeSSOCachedToken(t *testing.T, dir, cacheKey, body string) {
	t.Helper()
	hash := sha1.Sum([]byte(cacheKey))
	name := hex.EncodeToString(hash[:]) + ".json"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestFileAWSSSO(t *testing.T) {
	os.Clearenv()

	// 2020-01-10; cached tokens below expire 2020-01-11, role credentials
	// expire 2023-12-11 (1702317362000 ms).
	testNow := func() time.Time { return time.Date(2020, time.January, 10, 1, 1, 1, 1, time.UTC) }
	credsExpiration := time.Unix(0, 1702317362000*int64(time.Millisecond))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/federation/credentials" {
			t.Errorf("Expected path /federation/credentials, got %s", r.URL.Path)
		}
		wantRole := map[string]string{"123456789": "myrole", "987654321": "legacyrole", "222222222": "noregionrole", "333333333": "ghostrole"}
		accountID := r.URL.Query().Get("account_id")
		if _, ok := wantRole[accountID]; !ok {
			t.Errorf("Unexpected account ID %s", accountID)
		}
		if roleName := r.URL.Query().Get("role_name"); roleName != wantRole[accountID] {
			t.Errorf("Expected role name %s, got %s", wantRole[accountID], roleName)
		}
		if token := r.Header.Get("x-amz-sso_bearer_token"); token != "my-access-token" {
			t.Errorf("Expected bearer token my-access-token, got %s", token)
		}
		fmt.Fprintln(w, `{"roleCredentials": {"accessKeyId": "accessKey", "secretAccessKey": "secret", "sessionToken": "token", "expiration": 1702317362000}}`)
	}))
	defer ts.Close()

	newSSOCredsAt := func(profile, cacheDir, portalURL string) *Credentials {
		return New(&FileAWSCredentials{
			Expiry:               Expiry{CurrentTime: testNow},
			Filename:             "credentials-sso.sample",
			Profile:              profile,
			overrideSSOCacheDir:  cacheDir,
			overrideSSOPortalURL: portalURL,
		})
	}
	newSSOCreds := func(profile, cacheDir string) *Credentials {
		return newSSOCredsAt(profile, cacheDir, ts.URL)
	}

	checkValues := func(t *testing.T, creds *Credentials) {
		t.Helper()
		credValues, err := creds.GetWithContext(defaultCredContext)
		if err != nil {
			t.Fatal(err)
		}
		if credValues.AccessKeyID != "accessKey" {
			t.Errorf("Expected 'accessKey', got %s", credValues.AccessKeyID)
		}
		if credValues.SecretAccessKey != "secret" {
			t.Errorf("Expected 'secret', got %s", credValues.SecretAccessKey)
		}
		if credValues.SessionToken != "token" {
			t.Errorf("Expected 'token', got %s", credValues.SessionToken)
		}
		if !credValues.Expiration.Equal(credsExpiration) {
			t.Errorf("Expected expiration %v, got %v", credsExpiration, credValues.Expiration)
		}
		if creds.IsExpired() {
			t.Error("Should not be expired")
		}
	}

	t.Run("sso-session", func(t *testing.T) {
		cacheDir := t.TempDir()
		// The cached token file is named after the SHA1 of the sso_session
		// name ("main" in credentials-sso.sample).
		writeSSOCachedToken(t, cacheDir, "main",
			`{"startUrl": "https://testacct.awsapps.com/start", "region": "us-test-2", "accessToken": "my-access-token", "expiresAt": "2020-01-11T00:00:00Z"}`)
		checkValues(t, newSSOCreds("p1", cacheDir))
	})

	t.Run("legacy-start-url", func(t *testing.T) {
		cacheDir := t.TempDir()
		// Without sso_session, the cached token file is named after the
		// SHA1 of the profile's sso_start_url.
		writeSSOCachedToken(t, cacheDir, "https://legacy.awsapps.com/start",
			`{"startUrl": "https://legacy.awsapps.com/start", "region": "us-test-1", "accessToken": "my-access-token", "expiresAt": "2020-01-11T00:00:00Z"}`)
		checkValues(t, newSSOCreds("p2-legacy", cacheDir))
	})

	t.Run("expired-cached-token", func(t *testing.T) {
		cacheDir := t.TempDir()
		writeSSOCachedToken(t, cacheDir, "main",
			`{"startUrl": "https://testacct.awsapps.com/start", "region": "us-test-2", "accessToken": "my-access-token", "expiresAt": "2020-01-09T00:00:00Z"}`)
		_, err := newSSOCreds("p1", cacheDir).GetWithContext(defaultCredContext)
		if err == nil {
			t.Fatal("Expected error for expired cached SSO token")
		}
	})

	t.Run("config-region-without-token-region", func(t *testing.T) {
		cacheDir := t.TempDir()
		// No region in the cached token: the sso-session's sso_region must
		// be used.
		writeSSOCachedToken(t, cacheDir, "main",
			`{"startUrl": "https://testacct.awsapps.com/start", "accessToken": "my-access-token", "expiresAt": "2020-01-11T00:00:00Z"}`)
		checkValues(t, newSSOCreds("p1", cacheDir))
	})

	t.Run("token-region-fallback", func(t *testing.T) {
		cacheDir := t.TempDir()
		// Legacy profile without sso_region: the cached token's region must
		// be used.
		writeSSOCachedToken(t, cacheDir, "https://noregion.awsapps.com/start",
			`{"startUrl": "https://noregion.awsapps.com/start", "region": "us-test-3", "accessToken": "my-access-token", "expiresAt": "2020-01-11T00:00:00Z"}`)
		checkValues(t, newSSOCreds("p4-noregion", cacheDir))
	})

	t.Run("missing-session-section", func(t *testing.T) {
		cacheDir := t.TempDir()
		// sso_session names a section that does not exist: region resolution
		// falls through to the cached token's region.
		writeSSOCachedToken(t, cacheDir, "ghost",
			`{"startUrl": "https://ghost.awsapps.com/start", "region": "us-test-4", "accessToken": "my-access-token", "expiresAt": "2020-01-11T00:00:00Z"}`)
		checkValues(t, newSSOCreds("p5-ghost-session", cacheDir))
	})

	t.Run("portal-error-status", func(t *testing.T) {
		cacheDir := t.TempDir()
		writeSSOCachedToken(t, cacheDir, "main",
			`{"startUrl": "https://testacct.awsapps.com/start", "region": "us-test-2", "accessToken": "my-access-token", "expiresAt": "2020-01-11T00:00:00Z"}`)
		tsErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprintln(w, `{"message": "UnauthorizedException"}`)
		}))
		defer tsErr.Close()
		_, err := newSSOCredsAt("p1", cacheDir, tsErr.URL).GetWithContext(defaultCredContext)
		if err == nil || !strings.Contains(err.Error(), "UnauthorizedException") {
			t.Fatalf("Expected portal error containing body detail, got %v", err)
		}
	})

	t.Run("empty-role-credentials", func(t *testing.T) {
		cacheDir := t.TempDir()
		writeSSOCachedToken(t, cacheDir, "main",
			`{"startUrl": "https://testacct.awsapps.com/start", "region": "us-test-2", "accessToken": "my-access-token", "expiresAt": "2020-01-11T00:00:00Z"}`)
		tsEmpty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprintln(w, `{}`)
		}))
		defer tsEmpty.Close()
		_, err := newSSOCredsAt("p1", cacheDir, tsEmpty.URL).GetWithContext(defaultCredContext)
		if err == nil || !strings.Contains(err.Error(), "empty role credentials") {
			t.Fatalf("Expected empty-role-credentials error, got %v", err)
		}
	})

	t.Run("malformed-cached-token", func(t *testing.T) {
		cacheDir := t.TempDir()
		writeSSOCachedToken(t, cacheDir, "main", `{not json`)
		_, err := newSSOCreds("p1", cacheDir).GetWithContext(defaultCredContext)
		if err == nil {
			t.Fatal("Expected error for malformed cached SSO token")
		}
	})

	t.Run("missing-cached-token", func(t *testing.T) {
		_, err := newSSOCreds("p1", t.TempDir()).GetWithContext(defaultCredContext)
		if err == nil || !strings.Contains(err.Error(), "aws sso login") {
			t.Fatalf("Expected missing-cache error advising `aws sso login`, got %v", err)
		}
	})

	t.Run("missing-sso-config", func(t *testing.T) {
		_, err := newSSOCreds("p3-broken", t.TempDir()).GetWithContext(defaultCredContext)
		if err == nil || !strings.Contains(err.Error(), "neither sso_session nor sso_start_url") {
			t.Fatalf("Expected missing-sso-config error, got %v", err)
		}
	})

	t.Run("incomplete-sso-no-role", func(t *testing.T) {
		// SSO configuration without sso_role_name and without static keys
		// must error instead of yielding empty anonymous credentials.
		_, err := newSSOCreds("p6-norole", t.TempDir()).GetWithContext(defaultCredContext)
		if err == nil || !strings.Contains(err.Error(), "no sso_role_name") {
			t.Fatalf("Expected incomplete-SSO error naming sso_role_name, got %v", err)
		}
	})

	t.Run("missing-account-id", func(t *testing.T) {
		_, err := newSSOCreds("p7-noaccount", t.TempDir()).GetWithContext(defaultCredContext)
		if err == nil || !strings.Contains(err.Error(), "no sso_account_id") {
			t.Fatalf("Expected missing-account-id error, got %v", err)
		}
	})

	t.Run("mixed-static-fallback", func(t *testing.T) {
		// Incomplete SSO configuration alongside static keys: the static
		// keys are used, matching aws-sdk-go-v2's static-first resolution.
		credValues, err := newSSOCreds("p8-mixed", t.TempDir()).GetWithContext(defaultCredContext)
		if err != nil {
			t.Fatal(err)
		}
		if credValues.AccessKeyID != "mixedAccessKey" {
			t.Errorf("Expected 'mixedAccessKey', got %s", credValues.AccessKeyID)
		}
	})

	t.Run("mixed-complete-static-precedence", func(t *testing.T) {
		// Complete SSO configuration alongside a complete static key pair:
		// the static keys win, matching aws-sdk-go-v2's static-first
		// resolution. The empty cache dir would make the SSO flow fail with
		// a missing-token error, so success proves it was never engaged.
		credValues, err := newSSOCreds("p9-mixed-complete", t.TempDir()).GetWithContext(defaultCredContext)
		if err != nil {
			t.Fatal(err)
		}
		if credValues.AccessKeyID != "completeAccessKey" {
			t.Errorf("Expected 'completeAccessKey', got %s", credValues.AccessKeyID)
		}
		if credValues.SecretAccessKey != "completeSecret" {
			t.Errorf("Expected 'completeSecret', got %s", credValues.SecretAccessKey)
		}
	})

	t.Run("incomplete-sso-partial-static-key", func(t *testing.T) {
		// SSO markers plus only an access key must error rather than
		// return half a key pair.
		_, err := newSSOCreds("p10-partial-key", t.TempDir()).GetWithContext(defaultCredContext)
		if err == nil || !strings.Contains(err.Error(), "no sso_role_name") {
			t.Fatalf("Expected incomplete-SSO error for partial static key, got %v", err)
		}
	})

	t.Run("incomplete-sso-partial-static-secret", func(t *testing.T) {
		// SSO markers plus only a secret must error rather than return
		// half a key pair.
		_, err := newSSOCreds("p11-partial-secret", t.TempDir()).GetWithContext(defaultCredContext)
		if err == nil || !strings.Contains(err.Error(), "no sso_role_name") {
			t.Fatalf("Expected incomplete-SSO error for partial static secret, got %v", err)
		}
	})

	t.Run("region-indeterminable", func(t *testing.T) {
		cacheDir := t.TempDir()
		// Legacy profile without sso_region and a cached token without
		// region: the region cannot be resolved.
		writeSSOCachedToken(t, cacheDir, "https://noregion.awsapps.com/start",
			`{"startUrl": "https://noregion.awsapps.com/start", "accessToken": "my-access-token", "expiresAt": "2020-01-11T00:00:00Z"}`)
		_, err := newSSOCreds("p4-noregion", cacheDir).GetWithContext(defaultCredContext)
		if err == nil || !strings.Contains(err.Error(), "unable to determine AWS SSO region") {
			t.Fatalf("Expected region-indeterminable error, got %v", err)
		}
	})
}

func TestSSOPortalBaseURL(t *testing.T) {
	cases := []struct {
		region string
		want   string
	}{
		{"us-east-1", "https://portal.sso.us-east-1.amazonaws.com"},
		{"us-gov-west-1", "https://portal.sso.us-gov-west-1.amazonaws.com"},
		{"cn-north-1", "https://portal.sso.cn-north-1.amazonaws.com.cn"},
		{"us-iso-east-1", "https://portal.sso.us-iso-east-1.c2s.ic.gov"},
		{"us-isob-east-1", "https://portal.sso.us-isob-east-1.sc2s.sgov.gov"},
		{"eu-isoe-west-1", "https://portal.sso.eu-isoe-west-1.cloud.adc-e.uk"},
		{"us-isof-south-1", "https://portal.sso.us-isof-south-1.csp.hci.ic.gov"},
		{"eusc-de-east-1", "https://portal.sso.eusc-de-east-1.amazonaws.eu"},
	}
	for _, tc := range cases {
		if got := ssoPortalBaseURL(tc.region); got != tc.want {
			t.Errorf("ssoPortalBaseURL(%s) = %s, want %s", tc.region, got, tc.want)
		}
	}
}

// TestFileAWSDefaultDiscovery covers the empty-Filename path: the AWS config
// file and the shared credentials file are merged with credentials-file
// precedence, so profiles written by `aws configure sso` are found without
// pointing Filename anywhere.
func TestFileAWSDefaultDiscovery(t *testing.T) {
	os.Clearenv()

	testNow := func() time.Time { return time.Date(2020, time.January, 10, 1, 1, 1, 1, time.UTC) }

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, `{"roleCredentials": {"accessKeyId": "ssoAccessKey", "secretAccessKey": "ssoSecret", "sessionToken": "ssoToken", "expiration": 1702317362000}}`)
	}))
	defer ts.Close()

	writeAWSFile := func(t *testing.T, home, name, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(home, ".aws"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(home, ".aws", name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	setHome := func(t *testing.T, home string) {
		t.Helper()
		t.Setenv("HOME", home)        // os.UserHomeDir on Unix
		t.Setenv("USERPROFILE", home) // os.UserHomeDir on Windows
	}

	t.Run("sso-profile-from-config-file", func(t *testing.T) {
		home := t.TempDir()
		writeAWSFile(t, home, "config", `[profile dev]
sso_session = main
sso_account_id = 123456789
sso_role_name = myrole

[sso-session main]
sso_region = us-test-2
sso_start_url = https://testacct.awsapps.com/start
`)
		setHome(t, home)
		cacheDir := t.TempDir()
		writeSSOCachedToken(t, cacheDir, "main",
			`{"startUrl": "https://testacct.awsapps.com/start", "region": "us-test-2", "accessToken": "my-access-token", "expiresAt": "2020-01-11T00:00:00Z"}`)
		creds := New(&FileAWSCredentials{
			Expiry:               Expiry{CurrentTime: testNow},
			Profile:              "dev",
			overrideSSOCacheDir:  cacheDir,
			overrideSSOPortalURL: ts.URL,
		})
		credValues, err := creds.GetWithContext(defaultCredContext)
		if err != nil {
			t.Fatal(err)
		}
		if credValues.AccessKeyID != "ssoAccessKey" {
			t.Errorf("Expected 'ssoAccessKey', got %s", credValues.AccessKeyID)
		}
	})

	t.Run("aws-profile-env", func(t *testing.T) {
		// The issue #1930 shape: an empty FileAWSCredentials plus only
		// AWS_PROFILE in the environment.
		home := t.TempDir()
		writeAWSFile(t, home, "config", `[profile fromenv]
aws_access_key_id = envKey
aws_secret_access_key = envSecret
`)
		setHome(t, home)
		t.Setenv("AWS_PROFILE", "fromenv")
		credValues, err := New(&FileAWSCredentials{}).GetWithContext(defaultCredContext)
		if err != nil {
			t.Fatal(err)
		}
		if credValues.AccessKeyID != "envKey" {
			t.Errorf("Expected 'envKey', got %s", credValues.AccessKeyID)
		}
	})

	t.Run("credentials-file-precedence", func(t *testing.T) {
		home := t.TempDir()
		writeAWSFile(t, home, "config", `[profile dev]
aws_access_key_id = configKey
aws_secret_access_key = configSecret
`)
		writeAWSFile(t, home, "credentials", `[dev]
aws_access_key_id = credsKey
aws_secret_access_key = credsSecret
`)
		setHome(t, home)
		credValues, err := New(&FileAWSCredentials{Profile: "dev"}).GetWithContext(defaultCredContext)
		if err != nil {
			t.Fatal(err)
		}
		if credValues.AccessKeyID != "credsKey" {
			t.Errorf("Expected 'credsKey', got %s", credValues.AccessKeyID)
		}
		if credValues.SecretAccessKey != "credsSecret" {
			t.Errorf("Expected 'credsSecret', got %s", credValues.SecretAccessKey)
		}
	})

	t.Run("profile-merged-across-files", func(t *testing.T) {
		// SSO markers in the config file and a complete static pair in the
		// credentials file merge into one profile; the static pair wins,
		// matching aws-sdk-go-v2's static-first resolution.
		home := t.TempDir()
		writeAWSFile(t, home, "config", `[profile dev]
sso_start_url = https://merged.awsapps.com/start
`)
		writeAWSFile(t, home, "credentials", `[dev]
aws_access_key_id = credsKey
aws_secret_access_key = credsSecret
`)
		setHome(t, home)
		credValues, err := New(&FileAWSCredentials{Profile: "dev"}).GetWithContext(defaultCredContext)
		if err != nil {
			t.Fatal(err)
		}
		if credValues.AccessKeyID != "credsKey" {
			t.Errorf("Expected 'credsKey', got %s", credValues.AccessKeyID)
		}
	})

	t.Run("aws-config-file-env", func(t *testing.T) {
		home := t.TempDir()
		setHome(t, home)
		configPath := filepath.Join(t.TempDir(), "custom-config")
		if err := os.WriteFile(configPath, []byte(`[profile envprof]
aws_access_key_id = customKey
aws_secret_access_key = customSecret
`), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("AWS_CONFIG_FILE", configPath)
		credValues, err := New(&FileAWSCredentials{Profile: "envprof"}).GetWithContext(defaultCredContext)
		if err != nil {
			t.Fatal(err)
		}
		if credValues.AccessKeyID != "customKey" {
			t.Errorf("Expected 'customKey', got %s", credValues.AccessKeyID)
		}
	})

	t.Run("partial-pair-in-credentials-does-not-clobber", func(t *testing.T) {
		// A credentials-file section carrying only half the static key
		// pair must not overwrite half of the config file's complete
		// pair; the SDK merges the pair atomically.
		home := t.TempDir()
		writeAWSFile(t, home, "config", `[profile dev]
aws_access_key_id = configKey
aws_secret_access_key = configSecret
`)
		writeAWSFile(t, home, "credentials", `[dev]
aws_access_key_id = loneKey
`)
		setHome(t, home)
		credValues, err := New(&FileAWSCredentials{Profile: "dev"}).GetWithContext(defaultCredContext)
		if err != nil {
			t.Fatal(err)
		}
		if credValues.AccessKeyID != "configKey" {
			t.Errorf("Expected 'configKey', got %s", credValues.AccessKeyID)
		}
		if credValues.SecretAccessKey != "configSecret" {
			t.Errorf("Expected 'configSecret', got %s", credValues.SecretAccessKey)
		}
	})

	t.Run("isolated-session-token-in-credentials-ignored", func(t *testing.T) {
		// An aws_session_token in a credentials-file section lacking the
		// complete key pair is dropped, matching the SDK.
		home := t.TempDir()
		writeAWSFile(t, home, "config", `[profile dev]
aws_access_key_id = configKey
aws_secret_access_key = configSecret
`)
		writeAWSFile(t, home, "credentials", `[dev]
aws_session_token = loneToken
`)
		setHome(t, home)
		credValues, err := New(&FileAWSCredentials{Profile: "dev"}).GetWithContext(defaultCredContext)
		if err != nil {
			t.Fatal(err)
		}
		if credValues.SessionToken != "" {
			t.Errorf("Expected empty session token, got %s", credValues.SessionToken)
		}
	})

	t.Run("bare-profile-in-config-ignored", func(t *testing.T) {
		// A non-default config-file section without the "profile " prefix
		// is an invalid profile name; the AWS SDK ignores it.
		home := t.TempDir()
		writeAWSFile(t, home, "config", `[dev]
aws_access_key_id = bareKey
aws_secret_access_key = bareSecret
`)
		setHome(t, home)
		_, err := New(&FileAWSCredentials{Profile: "dev"}).GetWithContext(defaultCredContext)
		if err == nil {
			t.Fatal("Expected error: unprefixed config-file profile must be ignored")
		}
	})

	t.Run("prefixed-profile-in-credentials-ignored", func(t *testing.T) {
		// A "profile "-prefixed section is invalid in the credentials
		// file; the AWS SDK ignores it.
		home := t.TempDir()
		writeAWSFile(t, home, "credentials", `[profile dev]
aws_access_key_id = prefixedKey
aws_secret_access_key = prefixedSecret
`)
		setHome(t, home)
		_, err := New(&FileAWSCredentials{Profile: "dev"}).GetWithContext(defaultCredContext)
		if err == nil {
			t.Fatal("Expected error: prefixed credentials-file profile must be ignored")
		}
	})

	t.Run("profile-default-replaces-bare-default", func(t *testing.T) {
		// When one config file defines both [default] and
		// [profile default], the prefixed form wins wholesale: no key from
		// the bare section may leak into the resolved profile.
		home := t.TempDir()
		writeAWSFile(t, home, "config", `[default]
aws_access_key_id = bareKey
aws_secret_access_key = bareSecret
aws_session_token = staleToken

[profile default]
aws_access_key_id = prefixedKey
aws_secret_access_key = prefixedSecret
`)
		setHome(t, home)
		credValues, err := New(&FileAWSCredentials{}).GetWithContext(defaultCredContext)
		if err != nil {
			t.Fatal(err)
		}
		if credValues.AccessKeyID != "prefixedKey" {
			t.Errorf("Expected 'prefixedKey', got %s", credValues.AccessKeyID)
		}
		if credValues.SessionToken != "" {
			t.Errorf("Expected empty session token, got %s", credValues.SessionToken)
		}
	})

	t.Run("neither-file", func(t *testing.T) {
		home := t.TempDir()
		setHome(t, home)
		_, err := New(&FileAWSCredentials{Profile: "nope"}).GetWithContext(defaultCredContext)
		if err == nil {
			t.Fatal("Expected error when neither AWS file exists")
		}
		if !os.IsNotExist(err) {
			t.Errorf("Expected os.IsNotExist to hold for missing default files, got %v", err)
		}
	})
}

func TestFileMinioClient(t *testing.T) {
	os.Clearenv()

	creds := NewFileMinioClient("config.json.sample", "")
	credValues, err := creds.GetWithContext(defaultCredContext)
	if err != nil {
		t.Fatal(err)
	}

	if credValues.AccessKeyID != "accessKey" {
		t.Errorf("Expected 'accessKey', got %s'", credValues.AccessKeyID)
	}
	if credValues.SecretAccessKey != "secret" {
		t.Errorf("Expected 'secret', got %s'", credValues.SecretAccessKey)
	}
	if credValues.SignerType != SignatureV4 {
		t.Errorf("Expected 'S3v4', got %s'", credValues.SignerType)
	}

	os.Clearenv()
	t.Setenv("MINIO_ALIAS", "play")

	creds = NewFileMinioClient("config.json.sample", "")
	credValues, err = creds.GetWithContext(defaultCredContext)
	if err != nil {
		t.Fatal(err)
	}

	if credValues.AccessKeyID != "Q3AM3UQ867SPQQA43P2F" {
		t.Errorf("Expected 'Q3AM3UQ867SPQQA43P2F', got %s'", credValues.AccessKeyID)
	}
	if credValues.SecretAccessKey != "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG" {
		t.Errorf("Expected 'zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG', got %s'", credValues.SecretAccessKey)
	}
	if credValues.SignerType != SignatureV2 {
		t.Errorf("Expected 'S3v2', got %s'", credValues.SignerType)
	}

	os.Clearenv()

	creds = NewFileMinioClient("config.json.sample", "play")
	credValues, err = creds.GetWithContext(defaultCredContext)
	if err != nil {
		t.Fatal(err)
	}

	if credValues.AccessKeyID != "Q3AM3UQ867SPQQA43P2F" {
		t.Errorf("Expected 'Q3AM3UQ867SPQQA43P2F', got %s'", credValues.AccessKeyID)
	}
	if credValues.SecretAccessKey != "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG" {
		t.Errorf("Expected 'zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG', got %s'", credValues.SecretAccessKey)
	}
	if credValues.SignerType != SignatureV2 {
		t.Errorf("Expected 'S3v2', got %s'", credValues.SignerType)
	}

	creds = NewFileMinioClient("non-existent.json", "play")
	_, err = creds.GetWithContext(defaultCredContext)
	if !os.IsNotExist(err) {
		t.Errorf("Expected open non-existent.json: no such file or directory, got %s", err)
	}
	if !creds.IsExpired() {
		t.Error("Should be expired if not loaded")
	}
}
