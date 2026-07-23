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
	"net/url"
)

// ExecutionContext provides context metadata about the S3 operation currently running.
type ExecutionContext struct {
	Method      string // e.g. "GET", "PUT", "DELETE"
	BucketName  string
	ObjectName  string
	QueryValues url.Values // e.g. ?policy, ?tagging, ?versioning
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
// NOTE: This fires on every retry attempt. Only Initialize runs once.
type SerializeMiddleware interface {
	Middleware
	Serialize(ctx context.Context, execCtx ExecutionContext, req *http.Request) error
}

// FinalizeMiddleware runs AFTER the request is signed and ready to go out.
// Ideal for logging request sizes, outbound traffic, or tracing.
//
// NOTE: This fires on every retry attempt. Only Initialize runs once.
type FinalizeMiddleware interface {
	Middleware
	Finalize(ctx context.Context, execCtx ExecutionContext, req *http.Request) error
}

// DeserializeMiddleware runs AFTER the HTTP response is received from the server.
// It can inspect status codes, headers, or modify/log errors.
//
// NOTE: This fires on every retry attempt. Only Initialize runs once.
// Errors from all Deserialize middleware stack via errors.Join with the transport
// error. If any middleware returns a non-nil error, the request aborts even on
// 2xx responses.
type DeserializeMiddleware interface {
	Middleware
	Deserialize(ctx context.Context, execCtx ExecutionContext, resp *http.Response) error
}
