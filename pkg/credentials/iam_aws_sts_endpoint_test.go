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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// credsRespStsImpl is an AssumeRoleWithWebIdentity response. It lives here,
// in a file without a build constraint, because iam_aws_test.go (which
// also uses it) is excluded on Windows.
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

// stsEndpointEnv lists every variable that influences the web identity
// STS endpoint, so each test starts from a clean environment.
var stsEndpointEnv = []string{
	"AWS_ENDPOINT_URL_STS",
	"AWS_ENDPOINT_URL",
	"AWS_IGNORE_CONFIGURED_ENDPOINT_URLS",
	"AWS_CA_BUNDLE",
	"AWS_REGION",
}

// setupWebIdentityEnv clears the endpoint variables and points the
// web identity variables at a fresh token file.
func setupWebIdentityEnv(t *testing.T) {
	t.Helper()
	for _, k := range stsEndpointEnv {
		t.Setenv(k, "")
	}
	tokenFile := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tokenFile, []byte("token"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", tokenFile)
	t.Setenv("AWS_ROLE_ARN", "arn:aws:iam::123456789012:role/example")
}

// stsRecorder is an STS test server that counts the requests it
// receives and records the path of the last one.
type stsRecorder struct {
	*httptest.Server
	hits atomic.Int32
	path atomic.Value
}

func newSTSRecorder(t *testing.T, tlsServer bool) *stsRecorder {
	t.Helper()
	rec := &stsRecorder{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.hits.Add(1)
		rec.path.Store(r.URL.Path)
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if r.Form.Get("Action") != "AssumeRoleWithWebIdentity" {
			http.Error(w, "unexpected action", http.StatusBadRequest)
			return
		}
		fmt.Fprintf(w, credsRespStsImpl, "2099-01-01T00:00:00Z")
	})
	if tlsServer {
		rec.Server = httptest.NewTLSServer(handler)
	} else {
		rec.Server = httptest.NewServer(handler)
	}
	t.Cleanup(rec.Close)
	return rec
}

func (r *stsRecorder) lastPath() string {
	p, _ := r.path.Load().(string)
	return p
}

// localOnlyClient allows requests to loopback test servers only and fails
// any other request immediately, so a precedence regression can never
// reach a real STS endpoint.
func localOnlyClient() *CredContext {
	return &CredContext{Client: &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if ip := net.ParseIP(r.URL.Hostname()); ip == nil || !ip.IsLoopback() {
			return nil, fmt.Errorf("test request to non-local host %q", r.URL.Host)
		}
		return http.DefaultTransport.RoundTrip(r)
	})}}
}

func TestIAMWebIdentitySTSEndpointPrecedence(t *testing.T) {
	testCases := []struct {
		name string
		// set returns the explicit IAM.Endpoint and sets the environment
		// for the case. It receives the server that must be called and
		// a server that must not be called.
		set func(t *testing.T, want, other string) string
	}{
		{
			name: "explicit Endpoint wins over AWS_ENDPOINT_URL_STS and AWS_ENDPOINT_URL",
			set: func(t *testing.T, want, other string) string {
				t.Setenv("AWS_ENDPOINT_URL_STS", other)
				t.Setenv("AWS_ENDPOINT_URL", other)
				return want
			},
		},
		{
			name: "AWS_ENDPOINT_URL_STS wins over AWS_ENDPOINT_URL",
			set: func(t *testing.T, want, other string) string {
				t.Setenv("AWS_ENDPOINT_URL_STS", want)
				t.Setenv("AWS_ENDPOINT_URL", other)
				return ""
			},
		},
		{
			name: "AWS_ENDPOINT_URL_STS wins over AWS_REGION",
			set: func(t *testing.T, want, _ string) string {
				t.Setenv("AWS_REGION", "us-west-2")
				t.Setenv("AWS_ENDPOINT_URL_STS", want)
				return ""
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			setupWebIdentityEnv(t)
			want := newSTSRecorder(t, false)
			other := newSTSRecorder(t, false)

			p := &IAM{Endpoint: tc.set(t, want.URL, other.URL)}
			creds, err := p.RetrieveWithCredContext(localOnlyClient())
			if err != nil {
				t.Fatalf("Unexpected failure %s", err)
			}
			if creds.AccessKeyID != "accessKey" {
				t.Errorf("Expected \"accessKey\", got %s", creds.AccessKeyID)
			}
			if got := want.hits.Load(); got != 1 {
				t.Errorf("Expected 1 request to the selected endpoint, got %d", got)
			}
			if got := other.hits.Load(); got != 0 {
				t.Errorf("Expected no request to the lower-precedence endpoint, got %d", got)
			}
		})
	}
}

func TestIAMWebIdentitySTSEndpointPreservesPath(t *testing.T) {
	const stsPath = "/sts/example-namespace/example-name"
	for _, env := range []string{"AWS_ENDPOINT_URL_STS"} {
		t.Run(env, func(t *testing.T) {
			setupWebIdentityEnv(t)
			srv := newSTSRecorder(t, false)
			t.Setenv(env, srv.URL+stsPath)

			p := &IAM{}
			if _, err := p.RetrieveWithCredContext(localOnlyClient()); err != nil {
				t.Fatalf("Unexpected failure %s", err)
			}
			if got := srv.lastPath(); got != stsPath {
				t.Errorf("Expected request path %q, got %q", stsPath, got)
			}
		})
	}
}

func TestSTSEndpointFromEnv(t *testing.T) {
	testCases := []struct {
		name   string
		env    map[string]string
		expect string
	}{
		{
			name:   "nothing set",
			expect: "",
		},
		{
			name:   "service-specific",
			env:    map[string]string{"AWS_ENDPOINT_URL_STS": "https://sts.example.com/sts/a/b"},
			expect: "https://sts.example.com/sts/a/b",
		},
		{
			name:   "global variable is not read",
			env:    map[string]string{"AWS_ENDPOINT_URL": "https://example.com"},
			expect: "",
		},
		{
			name: "service-specific wins",
			env: map[string]string{
				"AWS_ENDPOINT_URL_STS": "https://sts.example.com",
				"AWS_ENDPOINT_URL":     "https://example.com",
			},
			expect: "https://sts.example.com",
		},
		{
			name: "ignored",
			env: map[string]string{
				"AWS_ENDPOINT_URL_STS":                "https://sts.example.com",
				"AWS_ENDPOINT_URL":                    "https://example.com",
				"AWS_IGNORE_CONFIGURED_ENDPOINT_URLS": "true",
			},
			expect: "",
		},
		{
			name: "ignore flag is case-insensitive",
			env: map[string]string{
				"AWS_ENDPOINT_URL_STS":                "https://sts.example.com",
				"AWS_IGNORE_CONFIGURED_ENDPOINT_URLS": "TRUE",
			},
			expect: "",
		},
		{
			name: "ignore flag false",
			env: map[string]string{
				"AWS_ENDPOINT_URL_STS":                "https://sts.example.com",
				"AWS_IGNORE_CONFIGURED_ENDPOINT_URLS": "false",
			},
			expect: "https://sts.example.com",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for _, k := range stsEndpointEnv {
				t.Setenv(k, "")
			}
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			if got := stsEndpointFromEnv(); got != tc.expect {
				t.Errorf("Expected %q, got %q", tc.expect, got)
			}
		})
	}
}

// writeServerCA writes the certificate of a TLS test server as a PEM
// bundle and returns its path.
func writeServerCA(t *testing.T, srv *httptest.Server) string {
	t.Helper()
	bundle := filepath.Join(t.TempDir(), "ca.pem")
	data := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	if err := os.WriteFile(bundle, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return bundle
}

func TestIAMWebIdentityCABundle(t *testing.T) {
	t.Run("trusted with AWS_CA_BUNDLE", func(t *testing.T) {
		setupWebIdentityEnv(t)
		srv := newSTSRecorder(t, true)
		t.Setenv("AWS_ENDPOINT_URL_STS", srv.URL)
		t.Setenv("AWS_CA_BUNDLE", writeServerCA(t, srv.Server))

		p := &IAM{}
		if _, err := p.RetrieveWithCredContext(defaultCredContext); err != nil {
			t.Fatalf("Unexpected failure %s", err)
		}
		if got := srv.hits.Load(); got != 1 {
			t.Errorf("Expected 1 request, got %d", got)
		}
	})

	t.Run("caller-supplied transport is an error and is not modified", func(t *testing.T) {
		setupWebIdentityEnv(t)
		srv := newSTSRecorder(t, true)
		t.Setenv("AWS_ENDPOINT_URL_STS", srv.URL)
		t.Setenv("AWS_CA_BUNDLE", writeServerCA(t, srv.Server))

		var used atomic.Bool
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.Proxy = func(*http.Request) (*url.URL, error) {
			used.Store(true)
			return nil, nil
		}
		cc := &CredContext{Client: &http.Client{Transport: transport}}

		p := &IAM{}
		_, err := p.RetrieveWithCredContext(cc)
		if err == nil || !strings.Contains(err.Error(), "AWS_CA_BUNDLE") || !strings.Contains(err.Error(), "IAM.Client") {
			t.Fatalf("Expected an AWS_CA_BUNDLE error pointing to IAM.Client, got %v", err)
		}
		if used.Load() || srv.hits.Load() != 0 {
			t.Error("Expected no request through the caller's transport")
		}
		if transport.TLSClientConfig != nil && transport.TLSClientConfig.RootCAs != nil {
			t.Error("Expected the caller's transport to be left unmodified")
		}
	})

	t.Run("RegisterProtocol policy on a caller transport is never bypassed", func(t *testing.T) {
		setupWebIdentityEnv(t)
		srv := newSTSRecorder(t, true)
		t.Setenv("AWS_ENDPOINT_URL_STS", srv.URL)
		t.Setenv("AWS_CA_BUNDLE", writeServerCA(t, srv.Server))

		// An egress policy registered as the https handler rejects everything.
		var policyCalls atomic.Int32
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.RegisterProtocol("https", roundTripperFunc(func(*http.Request) (*http.Response, error) {
			policyCalls.Add(1)
			return nil, errors.New("rejected by egress policy")
		}))
		cc := &CredContext{Client: &http.Client{Transport: transport}}

		p := &IAM{}
		_, err := p.RetrieveWithCredContext(cc)
		if err == nil || !strings.Contains(err.Error(), "IAM.Client") {
			t.Fatalf("Expected an error pointing to IAM.Client, got %v", err)
		}
		if got := srv.hits.Load(); got != 0 {
			t.Errorf("Expected no STS request to reach the server, got %d", got)
		}
		if got := policyCalls.Load(); got != 0 {
			t.Errorf("Expected no request at all, got %d through the policy", got)
		}
	})

	t.Run("ignored with an explicit IAM.Endpoint", func(t *testing.T) {
		setupWebIdentityEnv(t)
		srv := newSTSRecorder(t, false)
		t.Setenv("AWS_CA_BUNDLE", filepath.Join(t.TempDir(), "does-not-exist.pem"))

		// The bundle is not read at all, so a missing file is no error.
		p := &IAM{Endpoint: srv.URL}
		if _, err := p.RetrieveWithCredContext(localOnlyClient()); err != nil {
			t.Fatalf("Unexpected failure %s", err)
		}
	})

	t.Run("ignored for the AWS default endpoint", func(t *testing.T) {
		setupWebIdentityEnv(t)
		t.Setenv("AWS_REGION", "us-west-2")
		t.Setenv("AWS_CA_BUNDLE", filepath.Join(t.TempDir(), "does-not-exist.pem"))

		rec := &hostRecorder{}
		p := &IAM{}
		_, err := p.RetrieveWithCredContext(&CredContext{Client: &http.Client{Transport: rec}})
		if err == nil || strings.Contains(err.Error(), "AWS_CA_BUNDLE") {
			t.Fatalf("Expected only the blocked request to fail, got %v", err)
		}
		if hosts := rec.all(); len(hosts) != 1 || hosts[0] != "sts.us-west-2.amazonaws.com" {
			t.Errorf("Expected one request to sts.us-west-2.amazonaws.com, got %v", hosts)
		}
	})

	t.Run("untrusted without AWS_CA_BUNDLE", func(t *testing.T) {
		setupWebIdentityEnv(t)
		srv := newSTSRecorder(t, true)
		t.Setenv("AWS_ENDPOINT_URL_STS", srv.URL)

		p := &IAM{}
		_, err := p.RetrieveWithCredContext(defaultCredContext)
		var verr *tls.CertificateVerificationError
		if !errors.As(err, &verr) {
			t.Fatalf("Expected a certificate verification error, got %v", err)
		}
		if got := srv.hits.Load(); got != 0 {
			t.Errorf("Expected no request, got %d", got)
		}
	})

	t.Run("explicit IAM.Client ignores AWS_CA_BUNDLE", func(t *testing.T) {
		setupWebIdentityEnv(t)
		srv := newSTSRecorder(t, true)
		t.Setenv("AWS_ENDPOINT_URL_STS", srv.URL)
		t.Setenv("AWS_CA_BUNDLE", filepath.Join(t.TempDir(), "does-not-exist.pem"))

		p := &IAM{Client: srv.Client()}
		if _, err := p.RetrieveWithCredContext(defaultCredContext); err != nil {
			t.Fatalf("Unexpected failure %s", err)
		}
	})

	t.Run("missing bundle file", func(t *testing.T) {
		setupWebIdentityEnv(t)
		srv := newSTSRecorder(t, true)
		t.Setenv("AWS_ENDPOINT_URL_STS", srv.URL)
		t.Setenv("AWS_CA_BUNDLE", filepath.Join(t.TempDir(), "does-not-exist.pem"))

		// The library-owned client trusts the server, so only bundle
		// processing can make this retrieval fail.
		p := &IAM{}
		_, err := p.RetrieveWithCredContext(&CredContext{Client: srv.Client(), LibraryOwnedClient: true})
		if err == nil || !strings.Contains(err.Error(), "AWS_CA_BUNDLE") {
			t.Fatalf("Expected an AWS_CA_BUNDLE error, got %v", err)
		}
		if got := srv.hits.Load(); got != 0 {
			t.Errorf("Expected no request, got %d", got)
		}
	})

	t.Run("bundle without certificates", func(t *testing.T) {
		setupWebIdentityEnv(t)
		srv := newSTSRecorder(t, true)
		t.Setenv("AWS_ENDPOINT_URL_STS", srv.URL)
		bundle := filepath.Join(t.TempDir(), "empty.pem")
		if err := os.WriteFile(bundle, []byte("not a certificate"), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("AWS_CA_BUNDLE", bundle)

		p := &IAM{}
		_, err := p.RetrieveWithCredContext(&CredContext{Client: srv.Client(), LibraryOwnedClient: true})
		if err == nil || !strings.Contains(err.Error(), "AWS_CA_BUNDLE") {
			t.Fatalf("Expected an AWS_CA_BUNDLE error, got %v", err)
		}
		if got := srv.hits.Load(); got != 0 {
			t.Errorf("Expected no request, got %d", got)
		}
	})

	t.Run("wrapped transport is an error", func(t *testing.T) {
		setupWebIdentityEnv(t)
		srv := newSTSRecorder(t, true)
		t.Setenv("AWS_ENDPOINT_URL_STS", srv.URL)
		t.Setenv("AWS_CA_BUNDLE", writeServerCA(t, srv.Server))
		var wrapperCalls atomic.Int32
		wrapped := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			wrapperCalls.Add(1)
			return http.DefaultTransport.RoundTrip(r)
		})
		cc := &CredContext{Client: &http.Client{Transport: wrapped}, LibraryOwnedClient: true}

		p := &IAM{}
		_, err := p.RetrieveWithCredContext(cc)
		if err == nil || !strings.Contains(err.Error(), "AWS_CA_BUNDLE") || !strings.Contains(err.Error(), "IAM.Client") {
			t.Fatalf("Expected an AWS_CA_BUNDLE error pointing to IAM.Client, got %v", err)
		}
		if got := srv.hits.Load(); got != 0 {
			t.Errorf("Expected no request, got %d", got)
		}
		if got := wrapperCalls.Load(); got != 0 {
			t.Errorf("Expected the wrapper not to be called, got %d calls", got)
		}
	})

	t.Run("wrapped transport works through IAM.Client", func(t *testing.T) {
		setupWebIdentityEnv(t)
		srv := newSTSRecorder(t, true)
		t.Setenv("AWS_ENDPOINT_URL_STS", srv.URL)
		t.Setenv("AWS_CA_BUNDLE", writeServerCA(t, srv.Server))
		var wrapperCalls atomic.Int32
		inner := srv.Client().Transport
		wrapped := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			wrapperCalls.Add(1)
			return inner.RoundTrip(r)
		})

		p := &IAM{Client: &http.Client{Transport: wrapped}}
		if _, err := p.RetrieveWithCredContext(defaultCredContext); err != nil {
			t.Fatalf("Unexpected failure %s", err)
		}
		if got := wrapperCalls.Load(); got != 1 {
			t.Errorf("Expected the STS request to go through the wrapper once, got %d", got)
		}
	})
}

// hostRecorder is a RoundTripper that records the host of each request
// and fails it, so tests can assert the destination without network
// access.
type hostRecorder struct {
	mu    sync.Mutex
	hosts []string
}

func (h *hostRecorder) RoundTrip(r *http.Request) (*http.Response, error) {
	h.mu.Lock()
	h.hosts = append(h.hosts, r.URL.Hostname())
	h.mu.Unlock()
	return nil, errors.New("request blocked by test")
}

func (h *hostRecorder) all() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.hosts...)
}

func TestIAMWebIdentityIgnoresGlobalEndpointURL(t *testing.T) {
	setupWebIdentityEnv(t)
	srv := newSTSRecorder(t, false)
	t.Setenv("AWS_ENDPOINT_URL", srv.URL)
	t.Setenv("AWS_REGION", "us-west-2")

	rec := &hostRecorder{}
	p := &IAM{}
	if _, err := p.RetrieveWithCredContext(&CredContext{Client: &http.Client{Transport: rec}}); err == nil {
		t.Fatal("Expected the blocked request to fail")
	}
	if got := srv.hits.Load(); got != 0 {
		t.Errorf("Expected AWS_ENDPOINT_URL to be ignored, got %d requests to it", got)
	}
	if hosts := rec.all(); len(hosts) != 1 || hosts[0] != "sts.us-west-2.amazonaws.com" {
		t.Errorf("Expected one request to sts.us-west-2.amazonaws.com, got %v", hosts)
	}
}

func TestIAMWebIdentityIgnoreConfiguredEndpointURLs(t *testing.T) {
	setupWebIdentityEnv(t)
	srv := newSTSRecorder(t, false)
	t.Setenv("AWS_ENDPOINT_URL_STS", srv.URL)
	t.Setenv("AWS_ENDPOINT_URL", srv.URL)
	t.Setenv("AWS_IGNORE_CONFIGURED_ENDPOINT_URLS", "true")
	t.Setenv("AWS_REGION", "us-west-2")

	rec := &hostRecorder{}
	p := &IAM{}
	if _, err := p.RetrieveWithCredContext(&CredContext{Client: &http.Client{Transport: rec}}); err == nil {
		t.Fatal("Expected the blocked request to fail")
	}
	if got := srv.hits.Load(); got != 0 {
		t.Errorf("Expected no request to the configured endpoint, got %d", got)
	}
	if hosts := rec.all(); len(hosts) != 1 || hosts[0] != "sts.us-west-2.amazonaws.com" {
		t.Errorf("Expected one request to sts.us-west-2.amazonaws.com, got %v", hosts)
	}
}

func TestIAMEndpointEnvIgnoredOutsideWebIdentity(t *testing.T) {
	setupWebIdentityEnv(t)
	for _, k := range []string{
		"AWS_WEB_IDENTITY_TOKEN_FILE",
		"AWS_CONTAINER_CREDENTIALS_RELATIVE_URI",
		"AWS_CONTAINER_CREDENTIALS_FULL_URI",
		"AWS_CONTAINER_AUTHORIZATION_TOKEN",
		"AWS_CONTAINER_AUTHORIZATION_TOKEN_FILE",
	} {
		t.Setenv(k, "")
	}
	srv := newSTSRecorder(t, false)
	t.Setenv("AWS_ENDPOINT_URL_STS", srv.URL)
	t.Setenv("AWS_ENDPOINT_URL", srv.URL)

	rec := &hostRecorder{}
	p := &IAM{}
	if _, err := p.RetrieveWithCredContext(&CredContext{Client: &http.Client{Transport: rec}}); err == nil {
		t.Fatal("Expected the blocked request to fail")
	}
	if got := srv.hits.Load(); got != 0 {
		t.Errorf("Expected no request to the configured endpoint, got %d", got)
	}
	hosts := rec.all()
	if len(hosts) == 0 {
		t.Fatal("Expected at least one IMDS request")
	}
	for _, host := range hosts {
		if host != "169.254.169.254" {
			t.Errorf("Expected only IMDS requests, got a request to %s", host)
		}
	}
}

// newTLSServerWithOwnCert starts a TLS server with a freshly generated
// self-signed certificate. httptest TLS servers all share one built-in
// certificate, so tests that need two distinct roots use this instead.
func newTLSServerWithOwnCert(t *testing.T) *httptest.Server {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "distinct test root"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	srv.TLS = &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}}
	srv.StartTLS()
	t.Cleanup(srv.Close)
	return srv
}

func TestIAMWebIdentityCABundleExtendsRootCAs(t *testing.T) {
	setupWebIdentityEnv(t)
	srv := newSTSRecorder(t, true)
	t.Setenv("AWS_ENDPOINT_URL_STS", srv.URL)
	t.Setenv("AWS_CA_BUNDLE", writeServerCA(t, srv.Server))

	// The caller already trusts a different server; that trust must
	// survive next to the bundle.
	other := newTLSServerWithOwnCert(t)
	callerPool := x509.NewCertPool()
	callerPool.AddCert(other.Certificate())
	want := callerPool.Clone()
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{RootCAs: callerPool, MinVersion: tls.VersionTLS12}

	p := &IAM{}
	cc := &CredContext{Client: &http.Client{Transport: transport}, LibraryOwnedClient: true}
	if _, err := p.RetrieveWithCredContext(cc); err != nil {
		t.Fatalf("Unexpected failure %s", err)
	}
	if transport.TLSClientConfig.RootCAs != callerPool || !callerPool.Equal(want) {
		t.Error("Expected the caller's root CA pool to be left unmodified")
	}
}

func TestClientWithCABundleKeepsExistingRoots(t *testing.T) {
	bundled := newSTSRecorder(t, true)
	existing := newTLSServerWithOwnCert(t)

	pool := x509.NewCertPool()
	pool.AddCert(existing.Certificate())
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}

	client, err := clientWithCABundle(&http.Client{Transport: transport}, writeServerCA(t, bundled.Server))
	if err != nil {
		t.Fatal(err)
	}
	defer client.CloseIdleConnections()

	untrusted := newTLSServerWithOwnCert(t)
	if _, err := client.Get(untrusted.URL); err == nil {
		t.Error("Expected a server trusted by neither pool to be rejected")
	} else if verr := (*tls.CertificateVerificationError)(nil); !errors.As(err, &verr) {
		t.Errorf("Expected a certificate verification error, got %v", err)
	}

	for name, srv := range map[string]*httptest.Server{"bundle": bundled.Server, "existing root": existing} {
		resp, err := client.Get(srv.URL)
		if err != nil {
			t.Errorf("Expected the %s certificate to be trusted: %v", name, err)
			continue
		}
		resp.Body.Close()
	}
}

func TestClientWithCABundleKeepsBundleTrustOutOfCallerHTTP2Pool(t *testing.T) {
	// An HTTP/2-only server: it rejects any request that is not HTTP/2.
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor != 2 {
			http.Error(w, "HTTP/2 required", http.StatusHTTPVersionNotSupported)
			return
		}
		fmt.Fprint(w, r.Proto)
	}))
	srv.EnableHTTP2 = true
	srv.StartTLS()
	t.Cleanup(srv.Close)

	// A caller transport with its own HTTP/2 pool, installed the legacy way
	// (a TLSNextProto callback, as x/net/http2.ConfigureTransport does on
	// older Go versions). The fixture does not depend on how the installed
	// x/net version enables HTTP/2.
	caller := http.DefaultTransport.(*http.Transport).Clone()
	caller.TLSClientConfig = &tls.Config{NextProtos: []string{"h2", "http/1.1"}, MinVersion: tls.VersionTLS12}
	var callerPoolUses atomic.Int32
	caller.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{
		"h2": func(string, *tls.Conn) http.RoundTripper {
			callerPoolUses.Add(1)
			return roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return nil, errors.New("caller HTTP/2 pool used")
			})
		},
	}

	client, err := clientWithCABundle(&http.Client{Transport: caller}, writeServerCA(t, srv))
	if err != nil {
		t.Fatal(err)
	}
	defer client.CloseIdleConnections()

	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("Unexpected failure %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || resp.ProtoMajor != 2 {
		t.Errorf("Expected an HTTP/2 response from the HTTP/2-only server, got %s %s", resp.Proto, resp.Status)
	}
	if got := callerPoolUses.Load(); got != 0 {
		t.Errorf("Expected the bundle connection to stay out of the caller's HTTP/2 pool, got %d uses", got)
	}
	if _, ok := caller.TLSNextProto["h2"]; !ok {
		t.Error("Expected the caller's transport to keep its h2 callback")
	}
	if !slices.Contains(caller.TLSClientConfig.NextProtos, "h2") {
		t.Error("Expected the caller's TLS config to keep offering h2")
	}
}

func TestClientWithCABundleRejectsCustomProtocolHandlers(t *testing.T) {
	srv := newSTSRecorder(t, true)
	caller := http.DefaultTransport.(*http.Transport).Clone()
	caller.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{
		"example-proto": func(string, *tls.Conn) http.RoundTripper { return http.DefaultTransport },
	}

	_, err := clientWithCABundle(&http.Client{Transport: caller}, writeServerCA(t, srv.Server))
	if err == nil || !strings.Contains(err.Error(), "example-proto") || !strings.Contains(err.Error(), "IAM.Client") {
		t.Fatalf("Expected an error naming the custom protocol and IAM.Client, got %v", err)
	}
}

func TestClientWithCABundleRejectsTypedNilTransport(t *testing.T) {
	srv := newSTSRecorder(t, true)
	var nilTransport *http.Transport
	_, err := clientWithCABundle(&http.Client{Transport: nilTransport}, writeServerCA(t, srv.Server))
	if err == nil || !strings.Contains(err.Error(), "nil *http.Transport") {
		t.Fatalf("Expected an error for a typed-nil transport, got %v", err)
	}
}

func TestClientWithCABundleKeepsAutomaticHTTP2(t *testing.T) {
	for name, transport := range map[string]*http.Transport{
		"zero-value transport":             {},
		"unused clone of DefaultTransport": http.DefaultTransport.(*http.Transport).Clone(),
	} {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.ProtoMajor != 2 {
					http.Error(w, "HTTP/2 required", http.StatusHTTPVersionNotSupported)
				}
			}))
			srv.EnableHTTP2 = true
			srv.StartTLS()
			t.Cleanup(srv.Close)

			client, err := clientWithCABundle(&http.Client{Transport: transport}, writeServerCA(t, srv))
			if err != nil {
				t.Fatal(err)
			}
			defer client.CloseIdleConnections()
			resp, err := client.Get(srv.URL)
			if err != nil {
				t.Fatalf("Unexpected failure %s", err)
			}
			resp.Body.Close()
			if resp.ProtoMajor != 2 || resp.StatusCode != http.StatusOK {
				t.Errorf("Expected HTTP/2 to stay enabled on the clone, got %s %s", resp.Proto, resp.Status)
			}
		})
	}
}

func TestClientWithCABundleKeepsHTTP2Disabled(t *testing.T) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}
	client, err := clientWithCABundle(&http.Client{Transport: transport}, writeServerCA(t, newSTSRecorder(t, true).Server))
	if err != nil {
		t.Fatal(err)
	}
	if got := client.Transport.(*http.Transport).TLSNextProto; got == nil || len(got) != 0 {
		t.Errorf("Expected the clone to keep HTTP/2 disabled, got %v", got)
	}
}
