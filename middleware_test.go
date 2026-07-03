package minio

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/minio/minio-go/v7/pkg/credentials"
)

type mockMiddleware struct {
	mu           sync.Mutex
	initialized  []string
	serialized   []string
	finalized    []string
	deserialized []string
}

func (m *mockMiddleware) ID() string {
	return "mock-middleware"
}

func (m *mockMiddleware) Initialize(ctx context.Context, execCtx ExecutionContext) (context.Context, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.initialized = append(m.initialized, execCtx.Operation)
	return ctx, nil
}

func (m *mockMiddleware) Serialize(ctx context.Context, execCtx ExecutionContext, req *http.Request) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.serialized = append(m.serialized, execCtx.Operation)
	req.Header.Set("X-Mock-Header", "Hello")
	return nil
}

func (m *mockMiddleware) Finalize(ctx context.Context, execCtx ExecutionContext, req *http.Request) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.finalized = append(m.finalized, execCtx.Operation)
	return nil
}

func (m *mockMiddleware) Deserialize(ctx context.Context, execCtx ExecutionContext, resp *http.Response, err error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deserialized = append(m.deserialized, execCtx.Operation)
	return nil
}

func TestMiddlewareIntegration(t *testing.T) {
	// Start local mock S3 server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify custom header was injected by Serialize middleware
		if r.Header.Get("X-Mock-Header") != "Hello" {
			t.Errorf("Expected X-Mock-Header: 'Hello', got '%s'", r.Header.Get("X-Mock-Header"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	mw := &mockMiddleware{}

	// Instantiate client with middleware options
	client, err := New(server.URL[7:], &Options{
		Creds:        credentials.NewStaticV4("mockkey", "mocksecret", ""),
		Secure:       false,
		Region:       "us-east-1",
		Middlewares:  []Middleware{mw},
	})
	if err != nil {
		t.Fatalf("Failed to create minio client: %v", err)
	}

	// Make a GetBucketPolicy request (which uses executeMethod under the hood)
	_, err = client.GetBucketPolicy(context.Background(), "mybucket")
	// Since the mock server returns 200 OK but empty payload, GetBucketPolicy may return XML parsing error or nil, but HTTP request was triggered.

	mw.mu.Lock()
	defer mw.mu.Unlock()

	// Verify the S3 Operation was correctly resolved by resolveS3Operation (GetBucketPolicy)
	if len(mw.initialized) != 1 || mw.initialized[0] != "GetBucketPolicy" {
		t.Errorf("Expected initialized operation to be 'GetBucketPolicy', got %v", mw.initialized)
	}

	if len(mw.serialized) != 1 || mw.serialized[0] != "GetBucketPolicy" {
		t.Errorf("Expected serialized operation to be 'GetBucketPolicy', got %v", mw.serialized)
	}

	if len(mw.finalized) != 1 || mw.finalized[0] != "GetBucketPolicy" {
		t.Errorf("Expected finalized operation to be 'GetBucketPolicy', got %v", mw.finalized)
	}

	if len(mw.deserialized) != 1 || mw.deserialized[0] != "GetBucketPolicy" {
		t.Errorf("Expected deserialized operation to be 'GetBucketPolicy', got %v", mw.deserialized)
	}
}
