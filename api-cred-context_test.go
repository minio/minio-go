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
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/minio/minio-go/v7/pkg/credentials"
)

// testWaitTimeout bounds every cross-goroutine wait in this file so a
// regression fails in seconds instead of hanging to the package timeout.
const testWaitTimeout = 10 * time.Second

// recvSignal waits for ch to close, failing t if it does not in time.
func recvSignal(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(testWaitTimeout):
		t.Fatalf("timed out waiting for %s", what)
	}
}

// recvErr receives from ch, failing t if nothing arrives in time.
func recvErr(t *testing.T, ch <-chan error, what string) error {
	t.Helper()
	select {
	case err := <-ch:
		return err
	case <-time.After(testWaitTimeout):
		t.Fatalf("timed out waiting for %s", what)
		return nil
	}
}

// blockingProvider is a credentials.Provider whose retrieval blocks until
// released, honoring the CredContext caller context like real providers do.
type blockingProvider struct {
	startedOnce sync.Once
	started     chan struct{}
	release     chan struct{}
}

func (p *blockingProvider) RetrieveWithCredContext(cc *credentials.CredContext) (credentials.Value, error) {
	p.startedOnce.Do(func() { close(p.started) })
	select {
	case <-p.release:
	case <-cc.Context.Done():
		return credentials.Value{}, cc.Context.Err()
	}
	return credentials.Value{
		AccessKeyID:     "accessKey",
		SecretAccessKey: "secret",
		SignerType:      credentials.SignatureV4,
	}, nil
}

func (p *blockingProvider) Retrieve() (credentials.Value, error) {
	return p.RetrieveWithCredContext(&credentials.CredContext{Context: context.Background()})
}

func (p *blockingProvider) IsExpired() bool { return true }

// TestCredsCancelDoesNotPoisonWaiters verifies that canceling one caller
// during a de-duplicated credential retrieval fails only that caller: the
// shared retrieval is detached from any single caller's context, so a
// concurrent waiter on the same bucket still succeeds.
func TestCredsCancelDoesNotPoisonWaiters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	provider := &blockingProvider{started: make(chan struct{}), release: make(chan struct{})}
	clnt, err := New(srv.Listener.Addr().String(), &Options{
		Creds:  credentials.New(provider),
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	canceledErr := make(chan error, 1)
	go func() {
		_, err := clnt.BucketExists(ctx, "test-bucket")
		canceledErr <- err
	}()

	recvSignal(t, provider.started, "the provider retrieval to start")

	waiterErr := make(chan error, 1)
	go func() {
		_, err := clnt.BucketExists(context.Background(), "test-bucket")
		waiterErr <- err
	}()

	// Cancel the first caller mid-retrieval; the waiter must not inherit
	// that cancellation, whether it blocks on the credentials mutex or
	// joins the de-dup group.
	cancel()

	if err := recvErr(t, canceledErr, "the canceled caller"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Expected context.Canceled for the canceled caller, got %v", err)
	}

	close(provider.release)

	if err := recvErr(t, waiterErr, "the concurrent waiter"); err != nil {
		t.Fatalf("Expected the concurrent waiter to succeed, got %v", err)
	}
}

// expressSessionTransport records the first request's query and blocks
// every request until its context ends.
type expressSessionTransport struct {
	started    chan struct{}
	once       sync.Once
	firstQuery string
	requests   atomic.Int32
}

func (tr *expressSessionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tr.requests.Add(1)
	tr.once.Do(func() {
		tr.firstQuery = req.URL.RawQuery
		close(tr.started)
	})
	<-req.Context().Done()
	return nil, req.Context().Err()
}

// TestExpressCreateSessionCallerCancel verifies that the S3 Express session
// retrieval keeps the caller context: canceling the caller aborts the
// in-flight CreateSession request. The first-request query assertion pins
// that the express arm ran — only CreateSession sends "?session".
func TestExpressCreateSessionCallerCancel(t *testing.T) {
	tr := &expressSessionTransport{started: make(chan struct{})}
	clnt, err := New("s3.amazonaws.com", &Options{
		Creds:     credentials.NewStaticV4("k", "s", ""),
		Region:    "us-east-1",
		Transport: tr,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	callerErr := make(chan error, 1)
	go func() {
		_, err := clnt.BucketExists(ctx, "mybucket--use1-az4--x-s3")
		callerErr <- err
	}()
	recvSignal(t, tr.started, "the CreateSession request to arrive")
	cancel()

	if err := recvErr(t, callerErr, "the canceled caller"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Expected context.Canceled from the express session retrieval, got %v", err)
	}
	if !strings.Contains(tr.firstQuery, "session") {
		t.Fatalf("Expected the first request to be the CreateSession call (query %q lacks \"session\")", tr.firstQuery)
	}
}

// TestExpressWaiterOwnContextHonored verifies that a caller joining an
// in-flight de-duplicated S3 Express session retrieval stops waiting when
// its own context ends: the flight is parked in the transport for the whole
// test, so the waiter can only return through its own context.
func TestExpressWaiterOwnContextHonored(t *testing.T) {
	tr := &expressSessionTransport{started: make(chan struct{})}
	clnt, err := New("s3.amazonaws.com", &Options{
		Creds:     credentials.NewStaticV4("k", "s", ""),
		Region:    "us-east-1",
		Transport: tr,
	})
	if err != nil {
		t.Fatal(err)
	}

	const bucket = "mybucket--use1-az4--x-s3"
	winnerCtx, winnerCancel := context.WithCancel(context.Background())
	defer winnerCancel()
	winnerErr := make(chan error, 1)
	go func() {
		_, err := clnt.BucketExists(winnerCtx, bucket)
		winnerErr <- err
	}()
	recvSignal(t, tr.started, "the CreateSession request to arrive")

	waiterCtx, waiterCancel := context.WithCancel(context.Background())
	waiterCancel()
	waiterErr := make(chan error, 1)
	go func() {
		_, err := clnt.BucketExists(waiterCtx, bucket)
		waiterErr <- err
	}()
	if err := recvErr(t, waiterErr, "the waiter"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Expected the waiter's own context.Canceled, got %v", err)
	}

	winnerCancel()
	if err := recvErr(t, winnerErr, "the winner"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Expected context.Canceled for the canceled winner, got %v", err)
	}
	if n := tr.requests.Load(); n != 1 {
		t.Fatalf("Expected the waiter to join the winner's flight (1 CreateSession request), got %d", n)
	}
}

// recordingTransport serves every request with 200 OK and records the
// request queries it saw.
type recordingTransport struct {
	mu      sync.Mutex
	queries []string
}

func (tr *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tr.mu.Lock()
	tr.queries = append(tr.queries, req.URL.RawQuery)
	tr.mu.Unlock()
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       http.NoBody,
		Request:    req,
	}, nil
}

// TestExpressSessionCacheServedInline verifies that a still-fresh cached S3
// Express session skips the de-dup group and the CreateSession request
// entirely: the only request on the wire is the S3 operation itself.
func TestExpressSessionCacheServedInline(t *testing.T) {
	tr := &recordingTransport{}
	clnt, err := New("s3.amazonaws.com", &Options{
		Creds:     credentials.NewStaticV4("k", "s", ""),
		Region:    "us-east-1",
		Transport: tr,
	})
	if err != nil {
		t.Fatal(err)
	}

	const bucket = "mybucket--use1-az4--x-s3"
	clnt.bucketSessionCache.Set(bucket, credentials.Value{
		AccessKeyID:     "sessionKey",
		SecretAccessKey: "sessionSecret",
		SessionToken:    "sessionToken",
		SignerType:      credentials.SignatureV4,
		Expiration:      time.Now().Add(time.Hour),
	})

	exists, err := clnt.BucketExists(context.Background(), bucket)
	if err != nil || !exists {
		t.Fatalf("Expected the bucket probe to succeed from the cached session, got exists=%v err=%v", exists, err)
	}
	tr.mu.Lock()
	defer tr.mu.Unlock()
	if len(tr.queries) != 1 {
		t.Fatalf("Expected exactly one request (the S3 operation), got %d: %q", len(tr.queries), tr.queries)
	}
	if strings.Contains(tr.queries[0], "session") {
		t.Fatalf("Expected no CreateSession request for a fresh cached session, got query %q", tr.queries[0])
	}
}

// panicProvider is a credentials.Provider whose retrieval always panics.
type panicProvider struct{}

func (panicProvider) RetrieveWithCredContext(*credentials.CredContext) (credentials.Value, error) {
	panic("boom")
}

func (p panicProvider) Retrieve() (credentials.Value, error) {
	return p.RetrieveWithCredContext(nil)
}

func (panicProvider) IsExpired() bool { return true }

// TestCredsRetrievalPanicPropagates verifies that a provider panic inside
// the de-duplicated credential retrieval resumes on the caller's goroutine
// instead of crashing the process from the retrieval goroutine.
func TestCredsRetrievalPanicPropagates(t *testing.T) {
	clnt, err := New("s3.amazonaws.com", &Options{
		Creds:  credentials.New(panicProvider{}),
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if r := recover(); r != "boom" {
			t.Fatalf("Expected panic value %q on the caller goroutine, got %v", "boom", r)
		}
	}()
	exists, err := clnt.BucketExists(context.Background(), "test-bucket")
	t.Fatalf("Expected a panic before return, got exists=%v err=%v", exists, err)
}

// TestPresignCredsCallerContext verifies that a direct (non-de-duplicated)
// credential retrieval receives the caller context: a canceled caller
// context aborts presigning inside the provider.
func TestPresignCredsCallerContext(t *testing.T) {
	provider := &blockingProvider{started: make(chan struct{}), release: make(chan struct{})}
	clnt, err := New("s3.amazonaws.com", &Options{
		Creds:  credentials.New(provider),
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	policy := NewPostPolicy()
	if err := policy.SetBucket("test-bucket"); err != nil {
		t.Fatal(err)
	}
	if err := policy.SetKey("obj"); err != nil {
		t.Fatal(err)
	}
	if err := policy.SetExpires(time.Now().UTC().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	presignErr := make(chan error, 1)
	go func() {
		_, _, err := clnt.PresignedPostPolicy(ctx, policy)
		presignErr <- err
	}()
	if err := recvErr(t, presignErr, "the presign call"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Expected context.Canceled from the credential provider, got %v", err)
	}
}

// expressTraceProbeTransport serves CreateSession and S3 requests, recording
// whether each carried an httptrace.ClientTrace.
type expressTraceProbeTransport struct {
	mu     sync.Mutex
	traced []bool
}

func (tr *expressTraceProbeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tr.mu.Lock()
	tr.traced = append(tr.traced, httptrace.ContextClientTrace(req.Context()) != nil)
	tr.mu.Unlock()
	var body io.ReadCloser = http.NoBody
	if strings.Contains(req.URL.RawQuery, "session") {
		body = io.NopCloser(strings.NewReader(`<CreateSessionResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Credentials><AccessKeyId>k</AccessKeyId><SecretAccessKey>s</SecretAccessKey><SessionToken>tok</SessionToken><Expiration>2030-01-01T00:00:00Z</Expiration></Credentials></CreateSessionResult>`))
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       body,
		Request:    req,
	}, nil
}

// TestExpressSessionTraceStaysOff pins that with client tracing enabled the
// CreateSession request stays untraced while the S3 operation itself carries
// the trace: a de-duplicated session retrieval can serve several callers, so
// per-caller trace hooks must not fire for requests other callers own.
func TestExpressSessionTraceStaysOff(t *testing.T) {
	tr := &expressTraceProbeTransport{}
	clnt, err := New("s3.amazonaws.com", &Options{
		Creds:     credentials.NewStaticV4("k", "s", ""),
		Region:    "us-east-1",
		Transport: tr,
		Trace:     &httptrace.ClientTrace{},
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := clnt.BucketExists(context.Background(), "mybucket--use1-az4--x-s3"); err != nil {
		t.Fatal(err)
	}
	tr.mu.Lock()
	defer tr.mu.Unlock()
	if len(tr.traced) != 2 {
		t.Fatalf("Expected the CreateSession request and the S3 operation, got %d requests", len(tr.traced))
	}
	if tr.traced[0] {
		t.Fatal("Expected the CreateSession request to carry no client trace")
	}
	if !tr.traced[1] {
		t.Fatal("Expected the S3 operation to carry the client trace")
	}
}
