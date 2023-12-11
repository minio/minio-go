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
	"path"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestFileAWS(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("\"/bin/cat\": file does not exist")
	}
	os.Clearenv()

	creds := NewFileAWSCredentials("credentials.sample", "")
	credValues, err := creds.Get()
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

	os.Setenv("AWS_SHARED_CREDENTIALS_FILE", "credentials.sample")
	creds = NewFileAWSCredentials("", "")
	credValues, err = creds.Get()
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

	os.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(wd, "credentials.sample"))
	creds = NewFileAWSCredentials("", "")
	credValues, err = creds.Get()
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
	os.Setenv("AWS_PROFILE", "no_token")

	creds = NewFileAWSCredentials("credentials.sample", "")
	credValues, err = creds.Get()
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
	credValues, err = creds.Get()
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
	_, err = creds.Get()
	if !os.IsNotExist(err) {
		t.Errorf("Expected open non-existent.json: no such file or directory, got %s", err)
	}
	if !creds.IsExpired() {
		t.Error("Should be expired if not loaded")
	}

	os.Clearenv()

	creds = NewFileAWSCredentials("credentials.sample", "with_process")
	credValues, err = creds.Get()
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

func TestFileAWSSSO(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "minio-sso-")
	if err != nil {
		t.Errorf("Creating temp dir: %+v", err)
	}

	// the file path is the sso-profile, "main", sha1-ed
	os.WriteFile(
		path.Join(tmpDir, "b28b7af69320201d1cf206ebf28373980add1451.json"),
		[]byte(`{"startUrl": "https://testacct.awsapps.com/start", "region": "us-test-2", "accessToken": "my-access-token", "expiresAt": "2020-01-11T00:00:00Z"}`),
		0755,
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if urlPath := r.URL.Path; urlPath != "/federation/credentials" {
			t.Errorf("Expected path /federation/credentials, got %s", urlPath)
		}

		if accountID := r.URL.Query().Get("account_id"); accountID != "123456789" {
			t.Errorf("Expected account ID 123456789, got %s", accountID)
		}

		if roleName := r.URL.Query().Get("role_name"); roleName != "myrole" {
			t.Errorf("Expected role name myrole, got %s", roleName)
		}

		if xAuthHeader := r.Header.Get("x-amz-sso_bearer_token"); xAuthHeader != "my-access-token" {
			t.Errorf("Expected bearer token my-access-token, got %s", xAuthHeader)
		}

		fmt.Fprintln(w, `{"roleCredentials": {"accessKeyId": "accessKey", "secretAccessKey": "secret", "sessionToken": "token", "expiration":1702317362000}}`)
	}))
	defer ts.Close()

	creds := New(&FileAWSCredentials{
		Filename: "credentials-sso.sample",
		Profile:  "p1",

		overrideSSOPortalURL: ts.URL,
		overrideSSOCacheDir:  tmpDir,
		timeNow:              func() time.Time { return time.Date(2020, time.January, 10, 1, 1, 1, 1, time.UTC) },
	})
	credValues, err := creds.Get()
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

func TestFileMinioClient(t *testing.T) {
	os.Clearenv()

	creds := NewFileMinioClient("config.json.sample", "")
	credValues, err := creds.Get()
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
	os.Setenv("MINIO_ALIAS", "play")

	creds = NewFileMinioClient("config.json.sample", "")
	credValues, err = creds.Get()
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
	credValues, err = creds.Get()
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
	_, err = creds.Get()
	if !os.IsNotExist(err) {
		t.Errorf("Expected open non-existent.json: no such file or directory, got %s", err)
	}
	if !creds.IsExpired() {
		t.Error("Should be expired if not loaded")
	}
}
