/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2021 MinIO, Inc.
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
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

func TestGetObjectReturnSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
		w.Header().Set("Content-Length", "5")

		// Write less bytes than the content length.
		w.Write([]byte("12345"))
	}))
	defer srv.Close()

	// New - instantiate minio client with options
	clnt, err := New(srv.Listener.Addr().String(), &Options{
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	obj, err := clnt.GetObject(context.Background(), "bucketName", "objectName", GetObjectOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// We expect an error when reading back.
	buf, err := io.ReadAll(obj)
	if err != nil {
		t.Fatalf("Expected 'nil', got %v", err)
	}

	if len(buf) != 5 {
		t.Fatalf("Expected read bytes '5', got %v", len(buf))
	}
}

func TestGetObjectReturnErrorIfServerTruncatesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
		w.Header().Set("Content-Length", "100")

		// Write less bytes than the content length.
		w.Write([]byte("12345"))
	}))
	defer srv.Close()

	// New - instantiate minio client with options
	clnt, err := New(srv.Listener.Addr().String(), &Options{
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	obj, err := clnt.GetObject(context.Background(), "bucketName", "objectName", GetObjectOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// We expect an error when reading back.
	if _, err = io.ReadAll(obj); err != io.ErrUnexpectedEOF {
		t.Fatalf("Expected %v, got %v", io.ErrUnexpectedEOF, err)
	}
}

func TestGetObjectReturnErrorIfServerTruncatesResponseDouble(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
		w.Header().Set("Content-Length", "1024")

		// Write less bytes than the content length.
		io.Copy(w, io.LimitReader(rand.Reader, 1023))
	}))
	defer srv.Close()

	// New - instantiate minio client with options
	clnt, err := New(srv.Listener.Addr().String(), &Options{
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	obj, err := clnt.GetObject(context.Background(), "bucketName", "objectName", GetObjectOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// We expect an error when reading back.
	if _, err = io.ReadAll(obj); err != io.ErrUnexpectedEOF {
		t.Fatalf("Expected %v, got %v", io.ErrUnexpectedEOF, err)
	}
}

func TestGetObjectReturnErrorIfServerSendsMore(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
		w.Header().Set("Content-Length", "1")

		// Write less bytes than the content length.
		w.Write([]byte("12345"))
	}))
	defer srv.Close()

	// New - instantiate minio client with options
	clnt, err := New(srv.Listener.Addr().String(), &Options{
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	obj, err := clnt.GetObject(context.Background(), "bucketName", "objectName", GetObjectOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// We expect an error when reading back.
	if _, err = io.ReadAll(obj); err != io.ErrUnexpectedEOF {
		t.Fatalf("Expected %v, got %v", io.ErrUnexpectedEOF, err)
	}
}

// --- Parallel GET tests ---

// newParallelGetTestServer creates an httptest server that serves a known
// payload with support for HEAD (stat) and range-GET requests.
func newParallelGetTestServer(t *testing.T, data []byte) *httptest.Server {
	t.Helper()
	etag := `"test-etag-12345"`
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			w.Header().Set("Content-Length", strconv.Itoa(len(data)))
			w.Header().Set("ETag", etag)
			w.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)

		case http.MethodGet:
			rangeHeader := r.Header.Get("Range")
			if rangeHeader == "" {
				w.Header().Set("Content-Length", strconv.Itoa(len(data)))
				w.Header().Set("ETag", etag)
				w.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
				w.WriteHeader(http.StatusOK)
				w.Write(data)
				return
			}

			var start, end int64
			_, err := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end)
			if err != nil {
				http.Error(w, "bad range", http.StatusBadRequest)
				return
			}
			if start < 0 || end >= int64(len(data)) || start > end {
				http.Error(w, "range not satisfiable", http.StatusRequestedRangeNotSatisfiable)
				return
			}

			chunk := data[start : end+1]
			w.Header().Set("Content-Length", strconv.Itoa(len(chunk)))
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(data)))
			w.Header().Set("ETag", etag)
			w.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
			w.WriteHeader(http.StatusPartialContent)
			w.Write(chunk)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))
}

func TestGetObjectParallelSmallFallback(t *testing.T) {
	data := []byte("this is a small object that fits in one chunk")

	srv := newParallelGetTestServer(t, data)
	defer srv.Close()

	clnt, err := New(srv.Listener.Addr().String(), &Options{Region: "us-east-1"})
	if err != nil {
		t.Fatal(err)
	}

	obj, err := clnt.GetObject(context.Background(), "bucket", "key", GetObjectOptions{
		ParallelChunkSize: 4096,
		ParallelWorkers:   4,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer obj.Close()

	got, err := io.ReadAll(obj)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("data mismatch: got %d bytes, want %d", len(got), len(data))
	}
}

func TestGetObjectParallelLarge(t *testing.T) {
	const chunkSize = 1024
	const totalSize = chunkSize*3 + 256
	data := make([]byte, totalSize)
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}

	srv := newParallelGetTestServer(t, data)
	defer srv.Close()

	clnt, err := New(srv.Listener.Addr().String(), &Options{Region: "us-east-1"})
	if err != nil {
		t.Fatal(err)
	}

	obj, err := clnt.GetObject(context.Background(), "bucket", "key", GetObjectOptions{
		ParallelChunkSize: chunkSize,
		ParallelWorkers:   3,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer obj.Close()

	got, err := io.ReadAll(obj)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("data mismatch via GetObject parallel: got %d bytes, want %d", len(got), len(data))
	}
}

func TestGetObjectParallelConcurrency(t *testing.T) {
	const chunkSize = 512
	const totalSize = chunkSize * 4
	data := make([]byte, totalSize)
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}

	var concurrentRequests atomic.Int32
	var maxConcurrent atomic.Int32
	etag := `"test-etag-12345"`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			w.Header().Set("Content-Length", strconv.Itoa(len(data)))
			w.Header().Set("ETag", etag)
			w.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
			w.WriteHeader(http.StatusOK)

		case http.MethodGet:
			cur := concurrentRequests.Add(1)
			defer concurrentRequests.Add(-1)

			for {
				old := maxConcurrent.Load()
				if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
					break
				}
			}

			rangeHeader := r.Header.Get("Range")
			var start, end int64
			fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end)

			chunk := data[start : end+1]
			w.Header().Set("Content-Length", strconv.Itoa(len(chunk)))
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(data)))
			w.Header().Set("ETag", etag)
			w.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
			w.WriteHeader(http.StatusPartialContent)
			w.Write(chunk)
		}
	}))
	defer srv.Close()

	clnt, err := New(srv.Listener.Addr().String(), &Options{Region: "us-east-1"})
	if err != nil {
		t.Fatal(err)
	}

	obj, err := clnt.GetObject(context.Background(), "bucket", "key", GetObjectOptions{
		ParallelChunkSize: chunkSize,
		ParallelWorkers:   4,
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := io.ReadAll(obj)
	obj.Close()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatal("data mismatch")
	}
}

func TestGetObjectParallelETagPinning(t *testing.T) {
	const chunkSize = 512
	data := make([]byte, chunkSize*2)
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}

	etag := `"pinned-etag-abc"`
	var etagChecks atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			w.Header().Set("Content-Length", strconv.Itoa(len(data)))
			w.Header().Set("ETag", etag)
			w.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
			w.WriteHeader(http.StatusOK)

		case http.MethodGet:
			if ifMatch := r.Header.Get("If-Match"); ifMatch != "" {
				if strings.Contains(ifMatch, "pinned-etag-abc") {
					etagChecks.Add(1)
				}
			}

			rangeHeader := r.Header.Get("Range")
			var start, end int64
			fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end)

			chunk := data[start : end+1]
			w.Header().Set("Content-Length", strconv.Itoa(len(chunk)))
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(data)))
			w.Header().Set("ETag", etag)
			w.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
			w.WriteHeader(http.StatusPartialContent)
			w.Write(chunk)
		}
	}))
	defer srv.Close()

	clnt, err := New(srv.Listener.Addr().String(), &Options{Region: "us-east-1"})
	if err != nil {
		t.Fatal(err)
	}

	obj, err := clnt.GetObject(context.Background(), "bucket", "key", GetObjectOptions{
		ParallelChunkSize: chunkSize,
		ParallelWorkers:   2,
	})
	if err != nil {
		t.Fatal(err)
	}
	io.ReadAll(obj)
	obj.Close()

	if etagChecks.Load() != 2 {
		t.Fatalf("expected 2 ETag-pinned requests, got %d", etagChecks.Load())
	}
}

func TestGetObjectParallelOptionsDefaults(t *testing.T) {
	opts := GetObjectOptions{}
	if parallelGetChunkSize(opts) != minPartSize {
		t.Fatalf("expected default chunk size %d, got %d", minPartSize, parallelGetChunkSize(opts))
	}

	opts = GetObjectOptions{ParallelChunkSize: 1 << 20, ParallelWorkers: 4}
	if parallelGetChunkSize(opts) != 1<<20 {
		t.Fatalf("expected chunk size %d, got %d", 1<<20, parallelGetChunkSize(opts))
	}
}
