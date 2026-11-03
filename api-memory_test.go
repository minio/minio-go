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

package minio

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/minio/minio-go/v7/pkg/credentials"
)

func newMemoryTestClient(t *testing.T, handler http.HandlerFunc, opts ...func(*Options)) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	o := &Options{Creds: credentials.NewStaticV4("AKIDEXAMPLE", "SECRETEXAMPLE", "")}
	for _, f := range opts {
		f(o)
	}
	c, err := New(u.Host, o)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func writeMemoryError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"Code": code, "Message": message, "Resource": "/_mem/v1/cortexes", "RequestId": "REQ1", "HostId": "HOST1",
	})
}

func TestCreateCortexSignsTheRequest(t *testing.T) {
	var gotBody []byte
	var gotReq *http.Request
	c := newMemoryTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotReq = r
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"memories","bucket":"memories","encryption":true,"format":"1"}`))
	})

	cortex, err := c.CreateCortex(context.Background(), "memories", CreateCortexOptions{KMSKeyID: "key-1"})
	if err != nil {
		t.Fatal(err)
	}
	if cortex.Name != "memories" || !cortex.Encryption || cortex.Format != "1" {
		t.Fatalf("unexpected cortex %+v", cortex)
	}
	if gotReq.Method != http.MethodPost || gotReq.URL.Path != "/_mem/v1/cortexes" {
		t.Fatalf("got %s %s", gotReq.Method, gotReq.URL.Path)
	}
	var sent map[string]any
	if err := json.Unmarshal(gotBody, &sent); err != nil {
		t.Fatal(err)
	}
	if sent["name"] != "memories" || sent["kmsKeyId"] != "key-1" {
		t.Fatalf("unexpected body %s", gotBody)
	}
	sum := sha256.Sum256(gotBody)
	if got := gotReq.Header.Get("X-Amz-Content-Sha256"); got != hex.EncodeToString(sum[:]) {
		t.Fatalf("payload hash %q does not match the body", got)
	}
	auth := gotReq.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "AWS4-HMAC-SHA256 Credential=AKIDEXAMPLE/") || !strings.Contains(auth, "/us-east-1/s3/aws4_request") {
		t.Fatalf("unexpected Authorization %q", auth)
	}
	if ct := gotReq.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type %q", ct)
	}
	if got, want := authField(auth, "Signature"), expectedSigV4(t, gotReq, "SECRETEXAMPLE"); got != want {
		t.Fatalf("Signature %q, want %q derived from the received request", got, want)
	}
}

// authField returns the value of field in a SigV4 Authorization header.
func authField(auth, field string) string {
	_, params, _ := strings.Cut(auth, " ")
	for _, p := range strings.Split(params, ",") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(p), field+"="); ok {
			return v
		}
	}
	return ""
}

// expectedSigV4 derives the SigV4 signature of a received request from its
// method, raw path, query, signed headers and X-Amz-Date, independently of
// the client's signer, so a change in how the signer canonicalizes the
// request fails the test.
func expectedSigV4(t *testing.T, r *http.Request, secret string) string {
	t.Helper()
	auth := r.Header.Get("Authorization")
	signed := authField(auth, "SignedHeaders")
	scope := strings.SplitN(authField(auth, "Credential"), "/", 2)
	if signed == "" || len(scope) != 2 {
		t.Fatalf("malformed Authorization %q", auth)
	}
	amzDate := r.Header.Get("X-Amz-Date")

	var headers strings.Builder
	for _, name := range strings.Split(signed, ";") {
		var values []string
		switch name {
		case "host":
			values = []string{r.Host}
		case "content-length":
			values = []string{strconv.FormatInt(r.ContentLength, 10)}
		default:
			values = r.Header.Values(name)
		}
		for i, v := range values {
			values[i] = strings.Join(strings.Fields(v), " ")
		}
		headers.WriteString(name + ":" + strings.Join(values, ",") + "\n")
	}

	query := r.URL.Query()
	keys := make([]string, 0, len(query))
	for k := range query {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var pairs []string
	for _, k := range keys {
		vs := append([]string(nil), query[k]...)
		sort.Strings(vs)
		for _, v := range vs {
			pairs = append(pairs, url.QueryEscape(k)+"="+strings.ReplaceAll(url.QueryEscape(v), "+", "%20"))
		}
	}

	canonical := strings.Join([]string{
		r.Method,
		r.URL.EscapedPath(),
		strings.Join(pairs, "&"),
		headers.String(),
		signed,
		r.Header.Get("X-Amz-Content-Sha256"),
	}, "\n")
	canonicalSum := sha256.Sum256([]byte(canonical))
	stringToSign := "AWS4-HMAC-SHA256\n" + amzDate + "\n" + scope[1] + "\n" + hex.EncodeToString(canonicalSum[:])

	mac := func(key []byte, data string) []byte {
		h := hmac.New(sha256.New, key)
		h.Write([]byte(data))
		return h.Sum(nil)
	}
	parts := strings.Split(scope[1], "/") // date/region/service/aws4_request
	key := []byte("AWS4" + secret)
	for _, p := range parts {
		key = mac(key, p)
	}
	return hex.EncodeToString(mac(key, stringToSign))
}

func TestMemoryRequestsStayPathStyle(t *testing.T) {
	var path, host string
	c := newMemoryTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		path, host = r.URL.Path, r.Host
		_, _ = w.Write([]byte(`{"name":"memories"}`))
	}, func(o *Options) { o.BucketLookup = BucketLookupDNS })

	if _, err := c.GetCortex(context.Background(), "memories"); err != nil {
		t.Fatal(err)
	}
	if path != "/_mem/v1/cortexes/memories" || strings.HasPrefix(host, "memories.") {
		t.Fatalf("got host %q path %q", host, path)
	}
}

func TestMemoryErrorIsDecoded(t *testing.T) {
	c := newMemoryTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeMemoryError(w, http.StatusConflict, CortexAlreadyExists, "cortex already exists")
	})

	_, err := c.CreateCortex(context.Background(), "memories", CreateCortexOptions{})
	errResp := ToErrorResponse(err)
	if errResp.Code != CortexAlreadyExists || errResp.StatusCode != http.StatusConflict ||
		errResp.Message != "cortex already exists" || errResp.RequestID != "REQ1" {
		t.Fatalf("unexpected error %#v", errResp)
	}
}

func TestMemoryErrorFromAServerWithoutTheAPI(t *testing.T) {
	c := newMemoryTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`<Error><Code>InvalidBucketName</Code></Error>`))
	})

	_, err := c.CreateCortex(context.Background(), "memories", CreateCortexOptions{})
	errResp := ToErrorResponse(err)
	if errResp.Code != "" || errResp.StatusCode != http.StatusBadRequest || !strings.Contains(errResp.Message, "InvalidBucketName") {
		t.Fatalf("unexpected error %#v", errResp)
	}
}

func TestMemoryRetriesOnlyRepeatableRequests(t *testing.T) {
	var calls atomic.Int32
	c := newMemoryTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			writeMemoryError(w, http.StatusServiceUnavailable, "SlowDown", "slow down")
			return
		}
		if r.Method == http.MethodGet {
			w.Header().Set("ETag", `"e1"`)
		}
		_, _ = w.Write([]byte(`{"id":"alpha"}`))
	})

	if _, _, err := c.GetCortexAgent(context.Background(), "memories", "alpha"); err != nil {
		t.Fatalf("a read is retried: %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("read sent %d times, want 2", calls.Load())
	}

	calls.Store(0)
	_, err := c.CreateCortexAgent(context.Background(), "memories", "alpha", []byte(`{"id":"alpha"}`))
	if ToErrorResponse(err).Code != "SlowDown" {
		t.Fatalf("got %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("agent create sent %d times, want 1", calls.Load())
	}
}

func TestPutCortexSecretIsSentOnce(t *testing.T) {
	var calls atomic.Int32
	c := newMemoryTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		writeMemoryError(w, http.StatusServiceUnavailable, "SlowDown", "slow down")
	})

	_, err := c.PutCortexSecret(context.Background(), "memories", "a", []byte("v"))
	if ToErrorResponse(err).Code != "SlowDown" {
		t.Fatalf("got %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("secret put sent %d times, want 1, since each put adds a version", calls.Load())
	}
}

func TestGetCortexSecretDoesNotDrainAnOversizedBody(t *testing.T) {
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	c := newMemoryTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(make([]byte, 2*maxSecretValueBytes))
		w.(http.Flusher).Flush()
		// Hold the body open: a client draining it to EOF would wait here.
		select {
		case <-r.Context().Done():
		case <-release:
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _, err := c.GetCortexSecret(ctx, "memories", "big")
	if err == nil || ctx.Err() != nil {
		t.Fatalf("an oversized secret fails at once, got %v (context: %v)", err, ctx.Err())
	}
}

func TestMemoryNilContext(t *testing.T) {
	c := newMemoryTestClient(t, func(http.ResponseWriter, *http.Request) {
		t.Error("no request is sent without a context")
	})
	//nolint:staticcheck // a nil context is what is tested
	if _, err := c.GetCortex(nil, "memories"); ToErrorResponse(err).Code != InvalidArgument {
		t.Fatalf("got %v", err)
	}
}

func TestCortexSecrets(t *testing.T) {
	store := map[string][]byte{}
	c := newMemoryTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/_mem/v1/cortexes/memories":
			_, _ = w.Write([]byte(`{"name":"memories"}`))
			return
		case "/_mem/v1/cortexes/gone":
			writeMemoryError(w, http.StatusNotFound, NoSuchCortex, "no such cortex")
			return
		case "/_mem/v1/cortexes/gone/secrets/a":
			w.WriteHeader(http.StatusNotFound)
			return
		}
		name, ok := strings.CutPrefix(r.URL.Path, "/_mem/v1/cortexes/memories/secrets/")
		if !ok {
			t.Errorf("unexpected path %q", r.URL.Path)
			return
		}
		switch r.Method {
		case http.MethodPut:
			store[name], _ = io.ReadAll(r.Body)
			w.Header().Set(secretVersionHeader, "3")
			w.Header().Set(secretUpdatedHeader, "2026-11-02T10:00:00Z")
		case http.MethodGet, http.MethodHead:
			v, ok := store[name]
			if !ok {
				if r.Method == http.MethodHead {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				writeMemoryError(w, http.StatusNotFound, NoSuchSecret, "no such secret")
				return
			}
			w.Header().Set(secretVersionHeader, "3")
			if r.Method == http.MethodGet {
				_, _ = w.Write(v)
			}
		case http.MethodDelete:
			delete(store, name)
			w.WriteHeader(http.StatusNoContent)
		}
	})
	ctx := context.Background()

	info, err := c.PutCortexSecret(ctx, "memories", "a b", []byte("s3cr3t"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Version != 3 || info.Updated.IsZero() {
		t.Fatalf("unexpected info %+v", info)
	}
	value, info, err := c.GetCortexSecret(ctx, "memories", "a b")
	if err != nil || string(value) != "s3cr3t" || info.Version != 3 {
		t.Fatalf("got %q %+v %v", value, info, err)
	}
	if err := c.RemoveCortexSecret(ctx, "memories", "a b"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.GetCortexSecret(ctx, "memories", "a b"); ToErrorResponse(err).Code != NoSuchSecret {
		t.Fatalf("got %v", err)
	}
	if _, err := c.StatCortexSecret(ctx, "memories", "a b"); ToErrorResponse(err).Code != NoSuchSecret {
		t.Fatalf("a HEAD 404 is NoSuchSecret, got %v", err)
	}
	if _, err := c.StatCortexSecret(ctx, "gone", "a"); ToErrorResponse(err).Code != NoSuchCortex {
		t.Fatalf("a HEAD 404 in a missing cortex is NoSuchCortex, got %v", err)
	}
	if _, err := c.PutCortexSecret(ctx, "memories", "../x", nil); err == nil {
		t.Fatal("a name leaving the route is refused")
	}
	if _, err := c.PutCortexSecret(ctx, "memories", "big", make([]byte, maxSecretValueBytes+1)); err == nil {
		t.Fatal("an oversized value is refused")
	}
}

func TestListCortexSecretsFollowsPages(t *testing.T) {
	pages := map[string]listCortexSecretsPage{
		"":    {Secrets: []string{"p.a", "p.b"}, NextMarker: "p.b", IsTruncated: true},
		"p.b": {Secrets: []string{"p.c"}},
	}
	c := newMemoryTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_mem/v1/cortexes/memories/secrets" || r.URL.Query().Get("prefix") != "p." {
			t.Errorf("unexpected request %s", r.URL)
		}
		_ = json.NewEncoder(w).Encode(pages[r.URL.Query().Get("marker")])
	})

	var names []string
	for name, err := range c.ListCortexSecrets(context.Background(), "memories", ListCortexSecretsOptions{Prefix: "p."}) {
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, name)
	}
	if strings.Join(names, ",") != "p.a,p.b,p.c" {
		t.Fatalf("got %v", names)
	}
}

func TestListCortexSecretsRefusesAStuckMarker(t *testing.T) {
	c := newMemoryTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(listCortexSecretsPage{Secrets: []string{"a"}, IsTruncated: true})
	})

	var last error
	for _, err := range c.ListCortexSecrets(context.Background(), "memories", ListCortexSecretsOptions{}) {
		last = err
	}
	if last == nil {
		t.Fatal("a truncated page without a marker ends the listing with an error")
	}
}

func TestListCortexSecretsRefusesAMarkerCycle(t *testing.T) {
	pages := map[string]listCortexSecretsPage{
		"":  {Secrets: []string{"a"}, NextMarker: "a", IsTruncated: true},
		"a": {Secrets: []string{"b"}, NextMarker: "b", IsTruncated: true},
		"b": {Secrets: []string{"c"}, NextMarker: "a", IsTruncated: true},
	}
	var calls atomic.Int32
	c := newMemoryTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) > 10 {
			t.Error("the listing does not end")
		}
		_ = json.NewEncoder(w).Encode(pages[r.URL.Query().Get("marker")])
	})

	var last error
	for _, err := range c.ListCortexSecrets(context.Background(), "memories", ListCortexSecretsOptions{}) {
		last = err
	}
	if last == nil || calls.Load() != 3 {
		t.Fatalf("a marker seen before ends the listing with an error, got %v after %d pages", last, calls.Load())
	}
}

func TestCreateCortexAgentWithoutAKeyPair(t *testing.T) {
	for _, body := range []string{
		`{"credentials":{"accessKey":"AK"}}`,
		`{"credentials":{"secretKey":"SK"}}`,
		`null`,
	} {
		c := newMemoryTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(body))
		})
		if _, err := c.CreateCortexAgent(context.Background(), "memories", "alpha", []byte(`{"id":"alpha"}`)); err == nil {
			t.Fatalf("a create response %s without a full key pair is an error", body)
		}
	}
}

func TestListCortexSecretsRefusesANullPage(t *testing.T) {
	c := newMemoryTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("null\n"))
	})

	var last error
	for _, err := range c.ListCortexSecrets(context.Background(), "memories", ListCortexSecretsOptions{}) {
		last = err
	}
	if last == nil {
		t.Fatal("a null listing page is an error, not an empty listing")
	}

	// A null secrets field in a page object is still an empty page.
	c = newMemoryTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"secrets":null,"isTruncated":false}`))
	})
	for _, err := range c.ListCortexSecrets(context.Background(), "memories", ListCortexSecretsOptions{}) {
		t.Fatalf("got %v", err)
	}
}

func TestMemoryReadWithoutRetriesIsSentOnce(t *testing.T) {
	var calls atomic.Int32
	c := newMemoryTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = w.Write([]byte(`{"name":"memories"}`))
	})
	c.maxRetries = 0

	cortex, err := c.GetCortex(context.Background(), "memories")
	if err != nil || cortex.Name != "memories" || calls.Load() != 1 {
		t.Fatalf("got %+v %v after %d calls", cortex, err, calls.Load())
	}
}

func TestGetObjectBioDoesNotWaitOnAnOpenBody(t *testing.T) {
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	c := newMemoryTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"type":"end","error":"disk"}`+"\n")
		w.(http.Flusher).Flush()
		// Hold the body open: a client draining it would wait here.
		select {
		case <-r.Context().Done():
		case <-release:
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := c.GetObjectBio(ctx, "memories", "notes/a.json", ObjectBioOptions{})
	if err == nil || ctx.Err() != nil {
		t.Fatalf("an end frame reporting a failure returns at once, got %v (context: %v)", err, ctx.Err())
	}
}

func TestCortexAgentLifecycle(t *testing.T) {
	c := newMemoryTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_mem/v1/cortexes/memories/agents/alpha" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		switch {
		case r.Method == http.MethodPut && r.URL.Query().Has("update"):
			if r.Header.Get("If-Match") != `"e1"` {
				writeMemoryError(w, http.StatusPreconditionFailed, "PreconditionFailed", "stale")
				return
			}
			w.Header().Set("ETag", `"e2"`)
			_, _ = w.Write([]byte(`{"id":"alpha","v":2}`))
		case r.Method == http.MethodPut:
			if r.Header.Get("Content-Type") != "application/json" {
				writeMemoryError(w, http.StatusBadRequest, "InvalidAgentContentType", "json only")
				return
			}
			_, _ = w.Write([]byte(`{"agent":{"id":"alpha"},"credentials":{"accessKey":"AK","secretKey":"SK"},"telemetry":{"warehouse":"memories","namespace":"agent_alpha"}}`))
		case r.Method == http.MethodGet:
			w.Header().Set("ETag", `"e1"`)
			w.Header().Set("Last-Modified", "Mon, 02 Nov 2026 10:00:00 GMT")
			_, _ = w.Write([]byte(`{"id":"alpha"}`))
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		}
	})
	ctx := context.Background()

	created, err := c.CreateCortexAgent(ctx, "memories", "alpha", []byte(`{"id":"alpha"}`))
	if err != nil {
		t.Fatal(err)
	}
	if created.Credentials.SecretKey != "SK" || created.Telemetry.Namespace != "agent_alpha" || !bytes.Contains(created.Agent, []byte("alpha")) {
		t.Fatalf("unexpected result %+v", created)
	}
	record, info, err := c.GetCortexAgent(ctx, "memories", "alpha")
	if err != nil || info.ETag != "e1" || info.LastModified.IsZero() || !bytes.Contains(record, []byte(`"id":"alpha"`)) {
		t.Fatalf("got %s %+v %v", record, info, err)
	}
	record, info, err = c.UpdateCortexAgent(ctx, "memories", "alpha", []byte(`{"v":2}`), UpdateCortexAgentOptions{IfMatch: info.ETag})
	if err != nil || info.ETag != "e2" || !bytes.Contains(record, []byte(`"v":2`)) {
		t.Fatalf("got %s %+v %v", record, info, err)
	}
	if _, _, err := c.UpdateCortexAgent(ctx, "memories", "alpha", []byte(`{}`), UpdateCortexAgentOptions{}); err == nil {
		t.Fatal("an update without If-Match is refused")
	}
	if err := c.RemoveCortexAgent(ctx, "memories", "alpha"); err != nil {
		t.Fatal(err)
	}
}

func TestSearchCortex(t *testing.T) {
	stream := `{"type":"begin","name":"notes/a.md"}
{"type":"match","name":"notes/a.md","line":3,"column":1,"text":"hello","before":["x"]}
{"type":"end","name":"notes/a.md","matches":1}
{"type":"summary","stats":{"filesScanned":4,"filesMatched":1,"matchesReturned":1},"truncated":true}
`
	var body CortexSearchRequest
	c := newMemoryTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/_mem/v1/cortexes/memories/search" || r.URL.Query().Get("prefix") != "notes/" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = io.WriteString(w, stream)
	})

	res, err := c.SearchCortex(context.Background(), "memories", CortexSearchRequest{Prefix: "notes/", Patterns: []string{"hello"}}, CortexSearchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if body.Prefix != "notes/" || len(body.Patterns) != 1 {
		t.Fatalf("unexpected body %+v", body)
	}
	if len(res.Objects) != 1 || res.Objects[0].Matches[0].Line != 3 || res.Stats.FilesScanned != 4 || !res.Partial() {
		t.Fatalf("unexpected result %+v", res)
	}

	if _, err := readCortexSearch(strings.NewReader(`{"type":"begin","name":"a"}` + "\n")); err == nil {
		t.Fatal("a stream without its summary is incomplete")
	}
	if _, err := c.SearchCortex(context.Background(), "memories", CortexSearchRequest{Prefix: "/", Patterns: []string{"x"}}, CortexSearchOptions{}); err == nil {
		t.Fatal("a search of the whole cortex is refused")
	}
}

func TestGetObjectBio(t *testing.T) {
	stream := `{"type":"begin","key":"notes/a.json"}
{"type":"version","version":{"version":"v2","writtenAt":1793613600000,"principal":"alice","current":true,"meta":{"Author":"u1"}}}
{"type":"version","version":{"version":"v1","writtenAt":1793610000000,"principal":"alice"}}
{"type":"end","returned":2,"totalVersions":5,"truncated":true,"nextAfter":"v1"}
`
	c := newMemoryTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.URL.Path != "/_mem/v1/cortexes/memories/bio" || q.Get("key") != "notes/a.json" || q.Get("limit") != "2" || q.Get("after") != "v3" {
			t.Errorf("unexpected request %s", r.URL)
		}
		_, _ = io.WriteString(w, stream)
	})

	bio, err := c.GetObjectBio(context.Background(), "memories", "notes/a.json", ObjectBioOptions{Limit: 2, After: "v3"})
	if err != nil {
		t.Fatal(err)
	}
	if bio.Key != "notes/a.json" || len(bio.Entries) != 2 || bio.TotalVersions != 5 || !bio.Truncated || bio.NextAfter != "v1" {
		t.Fatalf("unexpected bio %+v", bio)
	}
	if e := bio.Entries[0]; e.VersionID != "v2" || !e.Current || e.Meta["Author"] != "u1" || e.WrittenAt.UnixMilli() != 1793613600000 {
		t.Fatalf("unexpected entry %+v", e)
	}

	if _, err := readObjectBio(strings.NewReader(`{"type":"version","version":{"version":"v1"}}` + "\n")); err == nil {
		t.Fatal("a stream without its end frame is incomplete")
	}
	if _, err := readObjectBio(strings.NewReader(`{"type":"end","error":"disk"}` + "\n")); err == nil {
		t.Fatal("an end frame reporting a failure is an error")
	}
	if _, err := readObjectBio(strings.NewReader(`{"type":"version","version":null}` + "\n" + `{"type":"end","totalVersions":1}` + "\n")); err == nil {
		t.Fatal("a version frame without a version is an error")
	}
	if _, err := readObjectBio(strings.NewReader(`{"type":"end","totalVersions":5,"truncated":true}` + "\n")); err == nil {
		t.Fatal("a truncated end frame without a nextAfter cursor is an error")
	}
}

func TestRawPathRequestNamesNoBucket(t *testing.T) {
	c := newMemoryTestClient(t, func(http.ResponseWriter, *http.Request) {})
	_, err := c.newRequest(context.Background(), http.MethodGet, requestMetadata{rawPath: "/_mem/v1/cortexes", bucketName: "b"})
	if err == nil {
		t.Fatal("a raw path request naming a bucket is refused")
	}
}
