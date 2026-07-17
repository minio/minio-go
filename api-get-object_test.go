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
	"sync"
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

func TestObjectSeekAtObjectSizeAllowsSubsequentReadEOF(t *testing.T) {
	o := &Object{
		mutex:         &sync.Mutex{},
		objectInfo:    ObjectInfo{Size: 10},
		objectInfoSet: true,
		isStarted:     true,
	}

	n, err := o.Seek(10, io.SeekStart)
	if err != nil {
		t.Fatalf("expected seeking to object size to succeed, got %v", err)
	}
	if n != 10 {
		t.Fatalf("expected offset 10, got %d", n)
	}
	if _, err = o.Read(make([]byte, 1)); err != io.EOF {
		t.Fatalf("expected read at object size to return io.EOF, got %v", err)
	}

	o.prevErr = nil
	o.currOffset = 9
	n, err = o.Seek(1, io.SeekCurrent)
	if err != nil {
		t.Fatalf("expected seeking current to object size to succeed, got %v", err)
	}
	if n != 10 {
		t.Fatalf("expected offset 10, got %d", n)
	}
	if _, err = o.Read(make([]byte, 1)); err != io.EOF {
		t.Fatalf("expected read at object size to return io.EOF, got %v", err)
	}

	o.prevErr = nil
	n, err = o.Seek(0, io.SeekEnd)
	if err != nil {
		t.Fatalf("expected seeking to object end to succeed, got %v", err)
	}
	if n != 10 {
		t.Fatalf("expected offset 10, got %d", n)
	}
	if _, err = o.Read(make([]byte, 1)); err != io.EOF {
		t.Fatalf("expected read at object end to return io.EOF, got %v", err)
	}
}

// eofRangeTestServer mimics a server answering HEAD with the object size,
// plain GET with the full payload, and range GET with 206 for satisfiable
// ranges or 416 InvalidRange when the range starts at or beyond the size,
// as captured in issue #2166. It counts GET requests.
func eofRangeTestServer(t *testing.T, payload []byte, getCount *int32) *httptest.Server {
	t.Helper()
	size := len(payload)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"0123456789abcdef0123456789abcdef"`)
		w.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
		w.Header().Set("Accept-Ranges", "bytes")

		if r.Method == http.MethodHead {
			w.Header().Set("Content-Length", strconv.Itoa(size))
			return
		}
		atomic.AddInt32(getCount, 1)
		rng := r.Header.Get("Range")
		if rng == "" {
			w.Header().Set("Content-Length", strconv.Itoa(size))
			w.Write(payload)
			return
		}
		var start, end int
		spec := strings.TrimPrefix(rng, "bytes=")
		if i := strings.Index(spec, "-"); i >= 0 {
			start, _ = strconv.Atoi(spec[:i])
			end = size - 1
			if i+1 < len(spec) {
				end, _ = strconv.Atoi(spec[i+1:])
			}
		}
		if start >= size {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?><Error><Code>InvalidRange</Code><Message>The requested range 'bytes=%d--1' is not satisfiable</Message><Key>objectName</Key><BucketName>bucketName</BucketName><ActualObjectSize>%d</ActualObjectSize><RangeRequested>%s</RangeRequested></Error>`, start, size, spec)
			return
		}
		if end >= size {
			end = size - 1
		}
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, size))
		w.Header().Set("Content-Length", strconv.Itoa(end-start+1))
		w.WriteHeader(http.StatusPartialContent)
		w.Write(payload[start : end+1])
	}))
}

func TestGetObjectReadAtEOFReturnsEOF(t *testing.T) {
	payload := []byte("0123456789")
	var gets int32
	srv := eofRangeTestServer(t, payload, &gets)
	defer srv.Close()

	clnt, err := New(srv.Listener.Addr().String(), &Options{
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Seek to the object size, then Read: io.EOF without any GET request.
	obj, err := clnt.GetObject(context.Background(), "bucketName", "objectName", GetObjectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	n64, err := obj.Seek(int64(len(payload)), io.SeekStart)
	if err != nil || n64 != int64(len(payload)) {
		t.Fatalf("Seek(size, SeekStart): expected (%d, nil), got (%d, %v)", len(payload), n64, err)
	}
	buf := make([]byte, 4)
	if _, err = obj.Read(buf); err != io.EOF {
		t.Fatalf("Read at object size: expected io.EOF, got %v", err)
	}
	if got := atomic.LoadInt32(&gets); got != 0 {
		t.Fatalf("Read at object size issued %d GET request(s), expected 0", got)
	}

	// The object must stay usable: seek back and read real data.
	if _, err = obj.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("Seek(0, SeekStart) after EOF read: expected success, got %v", err)
	}
	n, err := obj.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Read after re-seek: expected data, got %v", err)
	}
	if n != len(buf) || !bytes.Equal(buf, payload[:n]) {
		t.Fatalf("Read after re-seek: expected %q, got %q", payload[:len(buf)], buf[:n])
	}
	obj.Close()

	// ReadAt in chunks up to and past the end: the object size is never
	// known client-side on the ReadAt path, so the at-EOF request is issued
	// and the server's 416 InvalidRange must surface as io.EOF.
	obj, err = clnt.GetObject(context.Background(), "bucketName", "objectName", GetObjectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer obj.Close()
	n, err = obj.ReadAt(buf, 0)
	if err != nil && err != io.EOF {
		t.Fatalf("ReadAt(0): expected data, got %v", err)
	}
	if n != len(buf) || !bytes.Equal(buf, payload[:n]) {
		t.Fatalf("ReadAt(0): expected %q, got %q", payload[:len(buf)], buf[:n])
	}
	if _, err = obj.ReadAt(buf, int64(len(payload))); err != io.EOF {
		t.Fatalf("ReadAt(size): expected io.EOF, got %v", err)
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
