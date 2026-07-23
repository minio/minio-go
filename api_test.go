/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2015-2026 MinIO, Inc.
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
	"net/http"
	"runtime"
	"strings"
	"testing"
)

// TestDumpHTTPLargeBodyDoesNotAllocate tests that dumpHTTP does not allocate
func TestDumpHTTPLargeBodyDoesNotAllocate(t *testing.T) {
	var trace bytes.Buffer
	c := &Client{traceOutput: &trace, isTraceEnabled: true}
	req, err := http.NewRequest(http.MethodPut, "http://example.com/bucket/object", strings.NewReader("x"))
	if err != nil {
		t.Fatal(err)
	}
	req.ContentLength = 1 << 30 // pretend a 1 GiB upload was sent
	resp := &http.Response{StatusCode: http.StatusOK, Proto: "HTTP/1.1", ProtoMajor: 1, ProtoMinor: 1, Header: http.Header{}, Body: http.NoBody}

	var m0, m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m0)
	if err := c.dumpHTTP(req, resp,nil); err != nil {
		t.Fatal(err)
	}
	runtime.ReadMemStats(&m1)
	if got := m1.TotalAlloc - m0.TotalAlloc; got > 64<<20 {
		t.Fatalf("dumpHTTP allocated %d bytes for a 1 GiB Content-Length; the dump is buffering a body-sized dummy", got)
	}
	if !bytes.Contains(trace.Bytes(), []byte("PUT /bucket/object")) {
		t.Fatal("trace output missing the request line")
	}
}
