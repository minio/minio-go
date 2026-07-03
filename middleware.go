package minio

import (
	"context"
	"net/http"
)

// ExecutionContext provides context metadata about the S3 operation currently running.
type ExecutionContext struct {
	Operation  string // e.g. "GetObject", "PutObject", "ListBuckets"
	BucketName string
	ObjectName string
}

// Middleware is the base interface for all pipeline interceptors.
type Middleware interface {
	ID() string
}

// InitializeMiddleware runs BEFORE the HTTP request is built.
// It can mutate the context or alter metadata.
//
// NOTE: This runs once per API call (outside the retry loop), safe for
// one-shot setup or pre-validation.
type InitializeMiddleware interface {
	Middleware
	Initialize(ctx context.Context, execCtx ExecutionContext) (context.Context, error)
}

// SerializeMiddleware runs AFTER the request is created, but BEFORE it is signed.
// It can mutate headers, query parameters, etc.
//
// NOTE: This fires on EVERY retry attempt (new request per attempt).
// Only InitializeMiddleware runs once per API call.
type SerializeMiddleware interface {
	Middleware
	Serialize(ctx context.Context, execCtx ExecutionContext, req *http.Request) error
}

// FinalizeMiddleware runs AFTER the request is signed and ready to go out.
// Ideal for logging request sizes, outbound traffic, or tracing.
//
// NOTE: This fires on EVERY retry attempt inside the retry loop, not once per
// top-level API call. Only InitializeMiddleware runs once.
type FinalizeMiddleware interface {
	Middleware
	Finalize(ctx context.Context, execCtx ExecutionContext, req *http.Request) error
}

// DeserializeMiddleware runs AFTER the HTTP response is received from the server.
// It can inspect status codes, headers, or modify/log errors.
//
// NOTE: This fires on EVERY retry attempt inside the retry loop.
// Only InitializeMiddleware runs once per API call.
// Errors from all Deserialize middleware stack via errors.Join with the transport
// error. If any middleware returns a non-nil error, the request aborts even on
// 2xx responses. Use this for response validation.
type DeserializeMiddleware interface {
	Middleware
	Deserialize(ctx context.Context, execCtx ExecutionContext, resp *http.Response, err error) error
}
