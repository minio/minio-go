/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2015-2017 MinIO, Inc.
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
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/minio/minio-go/v7/pkg/credentials"
)

type recordedPhase struct {
	method      string
	bucketName  string
	objectName  string
	queryValues url.Values
}

type mockMiddleware struct {
	mu           sync.Mutex
	initialized  []recordedPhase
	serialized   []recordedPhase
	finalized    []recordedPhase
	deserialized []recordedPhase
}

func (m *mockMiddleware) ID() string {
	return "mock-middleware"
}

func (m *mockMiddleware) Initialize(ctx context.Context, execCtx ExecutionContext) (context.Context, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.initialized = append(m.initialized, recordedPhase{method: execCtx.Method, bucketName: execCtx.BucketName, objectName: execCtx.ObjectName, queryValues: execCtx.QueryValues})
	return ctx, nil
}

func (m *mockMiddleware) Serialize(ctx context.Context, execCtx ExecutionContext, req *http.Request) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.serialized = append(m.serialized, recordedPhase{method: execCtx.Method, bucketName: execCtx.BucketName, objectName: execCtx.ObjectName, queryValues: execCtx.QueryValues})
	req.Header.Set("X-Mock-Header", "Hello")
	return nil
}

func (m *mockMiddleware) Finalize(ctx context.Context, execCtx ExecutionContext, req *http.Request) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.finalized = append(m.finalized, recordedPhase{method: execCtx.Method, bucketName: execCtx.BucketName, objectName: execCtx.ObjectName, queryValues: execCtx.QueryValues})
	return nil
}

func (m *mockMiddleware) Deserialize(ctx context.Context, execCtx ExecutionContext, resp *http.Response) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deserialized = append(m.deserialized, recordedPhase{method: execCtx.Method, bucketName: execCtx.BucketName, objectName: execCtx.ObjectName, queryValues: execCtx.QueryValues})
	return nil
}

func TestMiddlewareIntegration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Mock-Header") != "Hello" {
			t.Errorf("Expected X-Mock-Header: 'Hello', got '%s'", r.Header.Get("X-Mock-Header"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	mw := &mockMiddleware{}

	client, err := New(strings.TrimPrefix(server.URL, "http://"), &Options{
		Creds:       credentials.NewStaticV4("mockkey", "mocksecret", ""),
		Secure:      false,
		Region:      "us-east-1",
		Middlewares: []Middleware{mw},
	})
	if err != nil {
		t.Fatalf("Failed to create minio client: %v", err)
	}

	_, err = client.GetBucketPolicy(context.Background(), "mybucket")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	mw.mu.Lock()
	defer mw.mu.Unlock()

	if len(mw.initialized) != 1 {
		t.Fatalf("Expected 1 initialized call, got %d", len(mw.initialized))
	}
	if mw.initialized[0].method != "GET" {
		t.Errorf("Expected method GET, got %s", mw.initialized[0].method)
	}
	if !mw.initialized[0].queryValues.Has("policy") {
		t.Errorf("Expected queryValues to have 'policy', got %v", mw.initialized[0].queryValues)
	}

	if len(mw.serialized) != 1 {
		t.Fatalf("Expected 1 serialized call, got %d", len(mw.serialized))
	}
	if mw.serialized[0].method != "GET" {
		t.Errorf("Expected method GET, got %s", mw.serialized[0].method)
	}

	if len(mw.finalized) != 1 {
		t.Fatalf("Expected 1 finalized call, got %d", len(mw.finalized))
	}

	if len(mw.deserialized) != 1 {
		t.Fatalf("Expected 1 deserialized call, got %d", len(mw.deserialized))
	}
}
