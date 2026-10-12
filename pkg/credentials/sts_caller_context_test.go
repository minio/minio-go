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
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// TestSTSProvidersCallerContextCanceled verifies that each remaining STS
// provider threads the caller context into its request: an already-canceled
// caller context aborts the retrieval before anything reaches the server.
func TestSTSProvidersCallerContextCanceled(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// The certificate provider clones the transport to attach its client
	// certificate, so the shared client needs a TLS-configured transport.
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{}}}

	cases := []struct {
		name     string
		retrieve func(*CredContext) (Value, error)
	}{
		{"client-grants", (&STSClientGrants{
			STSEndpoint: server.URL,
			GetClientGrantsTokenExpiry: func() (*ClientGrantsToken, error) {
				return &ClientGrantsToken{Token: "token", Expiry: 60}, nil
			},
		}).RetrieveWithCredContext},
		{"web-identity", (&STSWebIdentity{
			STSEndpoint: server.URL,
			GetWebIDTokenExpiry: func() (*WebIdentityToken, error) {
				return &WebIdentityToken{Token: "token"}, nil
			},
		}).RetrieveWithCredContext},
		{"ldap", (&LDAPIdentity{
			STSEndpoint:  server.URL,
			LDAPUsername: "user",
			LDAPPassword: "pass",
		}).RetrieveWithCredContext},
		{"custom-token", (&CustomTokenIdentity{
			STSEndpoint: server.URL,
			Token:       "token",
			RoleArn:     "arn:minio:iam:::role/x",
		}).RetrieveWithCredContext},
		{"certificate", (&STSCertificateIdentity{
			STSEndpoint: server.URL,
		}).RetrieveWithCredContext},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := retrieveWithin(t, "the "+tc.name+" retrieval", func() error {
				_, err := tc.retrieve(&CredContext{Client: client, Context: ctx})
				return err
			})
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("Expected context.Canceled, got %v", err)
			}
		})
	}
	if n := requests.Load(); n != 0 {
		t.Fatalf("Expected no requests with a canceled caller context, got %d", n)
	}
}
