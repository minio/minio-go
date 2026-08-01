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
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/minio/minio-go/v7/pkg/credentials"
)

// blockingProvider is a credentials.Provider whose retrieval blocks until
// released, honoring the CredContext caller context like real providers do.
type blockingProvider struct {
	startedOnce sync.Once
	started     chan struct{}
	release     chan struct{}
}

func (p *blockingProvider) RetrieveWithCredContext(cc *credentials.CredContext) (credentials.Value, error) {
	p.startedOnce.Do(func() { close(p.started) })
	var done <-chan struct{}
	if cc != nil && cc.Context != nil {
		done = cc.Context.Done()
	}
	select {
	case <-p.release:
	case <-done:
		return credentials.Value{}, cc.Context.Err()
	}
	return credentials.Value{
		AccessKeyID:     "accessKey",
		SecretAccessKey: "secret",
		SignerType:      credentials.SignatureV4,
	}, nil
}

func (p *blockingProvider) Retrieve() (credentials.Value, error) {
	return p.RetrieveWithCredContext(nil)
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

	<-provider.started

	waiterErr := make(chan error, 1)
	go func() {
		_, err := clnt.BucketExists(context.Background(), "test-bucket")
		waiterErr <- err
	}()

	// Cancel the first caller mid-retrieval; the waiter must not inherit
	// that cancellation, whether it blocks on the credentials mutex or
	// joins the de-dup group.
	cancel()

	if err := <-canceledErr; !errors.Is(err, context.Canceled) {
		t.Fatalf("Expected context.Canceled for the canceled caller, got %v", err)
	}

	close(provider.release)

	if err := <-waiterErr; err != nil {
		t.Fatalf("Expected the concurrent waiter to succeed, got %v", err)
	}
}

// expressSessionTransport records the first request's query and blocks
// every request until its context ends.
type expressSessionTransport struct {
	started    chan struct{}
	once       sync.Once
	firstQuery string
}

func (tr *expressSessionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
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
	go func() {
		<-tr.started
		cancel()
	}()

	_, err = clnt.BucketExists(ctx, "mybucket--use1-az4--x-s3")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Expected context.Canceled from the express session retrieval, got %v", err)
	}
	if !strings.Contains(tr.firstQuery, "session") {
		t.Fatalf("Expected the first request to be the CreateSession call (query %q lacks \"session\")", tr.firstQuery)
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

	_, _, err = clnt.PresignedPostPolicy(ctx, policy)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Expected context.Canceled from the credential provider, got %v", err)
	}
}
