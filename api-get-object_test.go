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
	"context"
	"crypto/rand"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
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

// TestObjectSeekAtObjectSizeAllowsSubsequentReadEOF verifies that seeking to (or
// past) the object size succeeds for every whence per the io.Seeker contract,
// and that io.EOF is reported by the subsequent Read rather than by Seek. See
// https://github.com/minio/minio-go/issues/2155 and #2166.
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

// TestObjectSeekCurrentNegativeOffset verifies that a negative offset with
// io.SeekCurrent is honoured when the resulting absolute position is within the
// object, and rejected only when it would move before the start of the object.
// Regression test for https://github.com/minio/minio-go/issues/2155.
func TestObjectSeekCurrentNegativeOffset(t *testing.T) {
	o := &Object{
		mutex:         &sync.Mutex{},
		objectInfo:    ObjectInfo{Size: 10},
		objectInfoSet: true,
		isStarted:     true,
		currOffset:    5,
	}

	n, err := o.Seek(-2, io.SeekCurrent)
	if err != nil {
		t.Fatalf("expected SeekCurrent with negative offset to succeed, got %v", err)
	}
	if n != 3 {
		t.Fatalf("expected offset 3, got %d", n)
	}

	_, err = o.Seek(-10, io.SeekCurrent)
	if err == nil {
		t.Fatal("expected error when SeekCurrent would move before start of object")
	}
}
