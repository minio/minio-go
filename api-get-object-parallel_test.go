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
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const parallelTestETag = "parallel-etag"

type parallelGetServer struct {
	*httptest.Server
	data []byte

	mu        sync.Mutex
	gets      []http.Header
	inFlight  atomic.Int32
	maxFlight atomic.Int32

	// shortAt truncates the body of the range GET starting at this offset.
	shortAt int64
}

func newParallelGetServer(t *testing.T, size int) *parallelGetServer {
	t.Helper()
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}
	s := &parallelGetServer{data: data, shortAt: -1}
	s.Server = httptest.NewServer(http.HandlerFunc(s.handle))
	t.Cleanup(s.Close)
	return s
}

func (s *parallelGetServer) handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("ETag", `"`+parallelTestETag+`"`)
	w.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
	if r.Method == http.MethodHead {
		w.Header().Set("Content-Length", strconv.Itoa(len(s.data)))
		return
	}

	n := s.inFlight.Add(1)
	defer s.inFlight.Add(-1)
	for {
		m := s.maxFlight.Load()
		if n <= m || s.maxFlight.CompareAndSwap(m, n) {
			break
		}
	}
	s.mu.Lock()
	s.gets = append(s.gets, r.Header.Clone())
	s.mu.Unlock()

	rng := r.Header.Get("Range")
	if r.URL.Query().Get("partNumber") != "" && rng != "" {
		http.Error(w, "Cannot specify both Range header and partNumber query parameter", http.StatusBadRequest)
		return
	}
	if rng == "" {
		w.Header().Set("Content-Length", strconv.Itoa(len(s.data)))
		w.Write(s.data)
		return
	}
	var start, end int64
	if _, err := fmt.Sscanf(rng, "bytes=%d-%d", &start, &end); err != nil || end >= int64(len(s.data)) {
		http.Error(w, "bad range", http.StatusRequestedRangeNotSatisfiable)
		return
	}
	body := s.data[start : end+1]
	if start == s.shortAt {
		body = body[:len(body)-1]
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(s.data)))
	w.WriteHeader(http.StatusPartialContent)
	w.Write(body)
}

func (s *parallelGetServer) getHeaders() []http.Header {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.gets)
}

func (s *parallelGetServer) rangeGets() (n int) {
	for _, h := range s.getHeaders() {
		if h.Get("Range") != "" {
			n++
		}
	}
	return n
}

func (s *parallelGetServer) getObject(t *testing.T, opts GetObjectOptions) *Object {
	t.Helper()
	clnt, err := New(s.Listener.Addr().String(), &Options{Region: "us-east-1"})
	if err != nil {
		t.Fatal(err)
	}
	obj, err := clnt.GetObject(context.Background(), "bucket", "object", opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { obj.Close() })
	return obj
}

func TestGetObjectParallel(t *testing.T) {
	s := newParallelGetServer(t, 3*minPartSize+123)
	got, err := io.ReadAll(s.getObject(t, GetObjectOptions{ParallelWorkers: 3}))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, s.data) {
		t.Fatalf("got %d bytes, want %d", len(got), len(s.data))
	}
	if n := s.rangeGets(); n != 4 {
		t.Fatalf("expected 4 range GETs, got %d", n)
	}
	for _, h := range s.getHeaders() {
		if h.Get("If-Match") != `"`+parallelTestETag+`"` {
			t.Fatalf("range GET without If-Match: %q", h.Get("If-Match"))
		}
	}
}

func TestGetObjectParallelSmallObject(t *testing.T) {
	s := newParallelGetServer(t, 1024)
	got, err := io.ReadAll(s.getObject(t, GetObjectOptions{ParallelWorkers: 4, ParallelChunkSize: 1}))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, s.data) {
		t.Fatalf("got %d bytes, want %d", len(got), len(s.data))
	}
	if n := s.rangeGets(); n != 0 {
		t.Fatalf("expected a single un-ranged GET, got %d range GETs", n)
	}
}

func TestGetObjectParallelAfterStat(t *testing.T) {
	s := newParallelGetServer(t, 2*minPartSize+1)
	obj := s.getObject(t, GetObjectOptions{ParallelWorkers: 3})
	data := make([]byte, (len(s.data)+1)/2)
	if _, err := io.ReadFull(obj, data); err != nil {
		t.Fatal(err)
	}
	info, err := obj.Stat()
	if err != nil || info.Size != int64(len(s.data)-len(data)) {
		t.Fatalf("Stat: got size %d, error %v; want size %d, nil", info.Size, err, len(s.data)-len(data))
	}
	rest, err := io.ReadAll(obj)
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, rest...)
	if !bytes.Equal(data, s.data) {
		t.Fatalf("got %d bytes after Stat, want %d", len(data), len(s.data))
	}
	if n := s.rangeGets(); n != 3 {
		t.Fatalf("expected 3 range GETs, got %d", n)
	}
}

func TestGetObjectParallelCallerRange(t *testing.T) {
	s := newParallelGetServer(t, 2*minPartSize+1)
	opts := GetObjectOptions{ParallelWorkers: 4}
	if err := opts.SetRange(100, 199); err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(s.getObject(t, opts))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, s.data[100:200]) {
		t.Fatalf("got %d bytes, want the 100 requested", len(got))
	}
	if len(s.getHeaders()) != 1 {
		t.Fatalf("expected 1 GET, got %d", len(s.getHeaders()))
	}
}

func TestGetObjectParallelPartNumber(t *testing.T) {
	s := newParallelGetServer(t, 2*minPartSize+1)
	got, err := io.ReadAll(s.getObject(t, GetObjectOptions{ParallelWorkers: 4, PartNumber: 1}))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, s.data) {
		t.Fatalf("got %d bytes, want %d", len(got), len(s.data))
	}
	if n := s.rangeGets(); n != 0 {
		t.Fatalf("expected no range GETs with PartNumber, got %d", n)
	}
}

func TestGetObjectParallelShortChunk(t *testing.T) {
	s := newParallelGetServer(t, 3*minPartSize)
	s.shortAt = minPartSize
	_, err := io.ReadAll(s.getObject(t, GetObjectOptions{ParallelWorkers: 2}))
	if err == nil {
		t.Fatal("expected an error for a short chunk")
	}
}

func TestGetObjectParallelSlowReader(t *testing.T) {
	const workers = 2
	s := newParallelGetServer(t, 5*minPartSize)
	obj := s.getObject(t, GetObjectOptions{ParallelWorkers: workers})

	if _, err := io.ReadFull(obj, make([]byte, 1)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	if n := s.rangeGets(); n > workers {
		t.Fatalf("a stalled reader let %d range GETs start, want at most %d", n, workers)
	}

	rest, err := io.ReadAll(obj)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rest, s.data[1:]) {
		t.Fatal("data mismatch")
	}
	if m := s.maxFlight.Load(); m > workers {
		t.Fatalf("%d concurrent GETs, want at most %d", m, workers)
	}
}

func TestGetObjectParallelClose(t *testing.T) {
	const workers = 2
	s := newParallelGetServer(t, 5*minPartSize)
	obj := s.getObject(t, GetObjectOptions{ParallelWorkers: workers})
	if _, err := io.ReadFull(obj, make([]byte, 1)); err != nil {
		t.Fatal(err)
	}
	obj.Close()
	time.Sleep(200 * time.Millisecond)
	if n := s.rangeGets(); n > workers {
		t.Fatalf("%d range GETs after Close, want at most %d", n, workers)
	}
}
