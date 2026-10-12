/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2026 MinIO, Inc.
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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestKubernetesIdentityWithWebIdentityTokenFile(t *testing.T) {
	var gotToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		gotToken = r.PostForm.Get("WebIdentityToken")
		w.Write([]byte(`<AssumeRoleWithWebIdentityResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/">
<AssumeRoleWithWebIdentityResult><Credentials>
<AccessKeyId>access</AccessKeyId><SecretAccessKey>secret</SecretAccessKey><SessionToken>session</SessionToken>
<Expiration>2099-01-01T00:00:00Z</Expiration>
</Credentials></AssumeRoleWithWebIdentityResult></AssumeRoleWithWebIdentityResponse>`))
	}))
	defer srv.Close()

	tokenFile := filepath.Join(t.TempDir(), "token")
	creds, err := NewKubernetesIdentity(srv.URL, WithWebIdentityTokenFile(tokenFile))
	if err != nil {
		t.Fatal(err)
	}

	for _, token := range []string{"first-token", "rotated-token"} {
		if err := os.WriteFile(tokenFile, []byte(token), 0o600); err != nil {
			t.Fatal(err)
		}
		creds.Expire()
		v, err := creds.Get()
		if err != nil {
			t.Fatal(err)
		}
		if gotToken != token {
			t.Fatalf("expected token %q, got %q", token, gotToken)
		}
		if v.AccessKeyID != "access" {
			t.Fatalf("expected access key %q, got %q", "access", v.AccessKeyID)
		}
	}
}
