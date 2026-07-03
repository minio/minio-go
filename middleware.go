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
// NOTE: This is the safe phase for one-shot logging — it runs once per API call
// before the retry loop (unlike Finalize/Deserialize which fire on every retry).
type SerializeMiddleware interface {
	Middleware
	Serialize(ctx context.Context, execCtx ExecutionContext, req *http.Request) error
}

// FinalizeMiddleware runs AFTER the request is signed and ready to go out.
// Ideal for logging request sizes, outbound traffic, or tracing.
//
// NOTE: This fires on EVERY retry attempt inside the retry loop, not once per
// top-level API call. If you only want to log/logic once, use SerializeMiddleware
// instead.
type FinalizeMiddleware interface {
	Middleware
	Finalize(ctx context.Context, execCtx ExecutionContext, req *http.Request) error
}

// DeserializeMiddleware runs AFTER the HTTP response is received from the server.
// It can inspect status codes, headers, or modify/log errors.
//
// NOTE: This fires on EVERY retry attempt inside the retry loop. Errors from all
// Deserialize middleware are stacked via errors.Join with the transport error.
// If any middleware returns a non-nil error, the request is aborted even on 2xx
// responses. Use this for response validation.
type DeserializeMiddleware interface {
	Middleware
	Deserialize(ctx context.Context, execCtx ExecutionContext, resp *http.Response, err error) error
}
