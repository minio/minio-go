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
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestSTSCertificateIdentityNilTLSClientConfig verifies that a transport
// without a TLSClientConfig does not panic when the client certificate is set.
func TestSTSCertificateIdentityNilTLSClientConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`<AssumeRoleWithCertificateResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/">
<AssumeRoleWithCertificateResult><Credentials>
<AccessKeyId>accessKey</AccessKeyId><SecretAccessKey>secret</SecretAccessKey>
<SessionToken>token</SessionToken><Expiration>2026-01-01T00:00:00Z</Expiration>
</Credentials></AssumeRoleWithCertificateResult>
</AssumeRoleWithCertificateResponse>`))
	}))
	defer server.Close()

	// A custom dialer keeps net/http from enabling HTTP/2, which would
	// otherwise allocate the TLSClientConfig when the transport is cloned.
	tr := &http.Transport{DialContext: (&net.Dialer{}).DialContext}
	creds, err := NewSTSCertificateIdentity(server.URL, tls.Certificate{},
		CertificateIdentityWithTransport(tr))
	if err != nil {
		t.Fatal(err)
	}
	value, err := creds.GetWithContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	if value.AccessKeyID != "accessKey" {
		t.Errorf("Expected access key %q, got %q", "accessKey", value.AccessKeyID)
	}
}
