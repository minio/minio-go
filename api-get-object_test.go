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
	"testing"
)

func TestGetObjectReturnSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
