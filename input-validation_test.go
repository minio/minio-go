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
	"testing"
)

// TestNilReaderIsRejected ensures APIs taking an io.Reader return an error
// instead of panicking in the HTTP transport when the reader is nil.
func TestNilReaderIsRejected(t *testing.T) {
	rt := &InterceptRouteTripper{}
	c, err := New("s3.amazonaws.com", &Options{Transport: rt})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	if _, err := c.PutObject(ctx, "bucket", "object", nil, 10, PutObjectOptions{}); err == nil {
		t.Fatal("PutObject: expected an error for a nil reader")
	}
	if _, err := c.PutObject(ctx, "bucket", "object", nil, -1, PutObjectOptions{}); err == nil {
		t.Fatal("PutObject: expected an error for a nil reader with unknown size")
	}
	if _, err := c.AppendObject(ctx, "bucket", "object", nil, 10, AppendObjectOptions{}); err == nil {
		t.Fatal("AppendObject: expected an error for a nil reader")
	}
	if _, err := c.PutObjectFanOut(ctx, "bucket", nil, PutObjectFanOutRequest{
		Entries: []PutObjectFanOutEntry{{Key: "key"}},
	}); err == nil {
		t.Fatal("PutObjectFanOut: expected an error for nil fan-out data")
	}
	if rt.request != nil {
		t.Fatal("a nil reader must be rejected before any request is issued")
	}

	core, err := NewCore("s3.amazonaws.com", &Options{Transport: rt})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := core.PutObject(ctx, "bucket", "object", nil, 10, "", "", PutObjectOptions{}); err == nil {
		t.Fatal("Core.PutObject: expected an error for nil data")
	}
	if _, err := core.PutObjectPart(ctx, "bucket", "object", "upload-id", 1, nil, 10, PutObjectPartOptions{}); err == nil {
		t.Fatal("Core.PutObjectPart: expected an error for nil data")
	}
}

// TestSnowballNilContent ensures a nil object content is rejected instead of
// panicking while writing the archive.
func TestSnowballNilContent(t *testing.T) {
	c, err := New("s3.amazonaws.com", &Options{})
	if err != nil {
		t.Fatal(err)
	}
	objs := make(chan SnowballObject, 1)
	objs <- SnowballObject{Key: "object", Size: 0, Content: nil}
	close(objs)

	if err := c.PutObjectsSnowball(context.Background(), "bucket", SnowballOptions{InMemory: true}, objs); err == nil {
		t.Fatal("expected an error for a nil object content")
	}
}

// TestGetNumThreads verifies the worker count is capped so a caller cannot
// make the client allocate an unbounded number of part buffers.
func TestGetNumThreads(t *testing.T) {
	tests := []struct {
		in   uint
		want int
	}{
		{0, totalWorkers},
		{1, 1},
		{4, 4},
		{maxNumThreads, maxNumThreads},
		{maxNumThreads + 1, maxNumThreads},
		{1 << 40, maxNumThreads},
	}
	for _, tt := range tests {
		got := PutObjectOptions{NumThreads: tt.in}.getNumThreads()
		if got != tt.want {
			t.Errorf("PutObjectOptions{NumThreads: %d}.getNumThreads() = %d, want %d", tt.in, got, tt.want)
		}
		got = CopyDestOptions{NumThreads: tt.in}.getNumThreads()
		want := tt.want
		if tt.in == 0 {
			// CopyDestOptions treats 0 as a single worker.
			want = 1
		}
		if got != want {
			t.Errorf("CopyDestOptions{NumThreads: %d}.getNumThreads() = %d, want %d", tt.in, got, want)
		}
	}
}
