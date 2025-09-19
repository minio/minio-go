/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2025 MinIO, Inc.
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
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFGetObjectReturnSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
		w.Header().Set("Content-Length", "5")
		w.Header().Set("Etag", "abc123")
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

	localFilePath := filepath.Join(t.TempDir(), "minio_test_fgetobject_file")

	err = clnt.FGetObject(context.Background(), "bucketName", "objectName", localFilePath, GetObjectOptions{})
	if err != nil {
		t.Fatal(err)
	}

	buf, err := os.ReadFile(localFilePath)
	if err != nil {
		t.Fatalf("Expected 'nil', got %v", err)
	}

	if len(buf) != 5 {
		t.Fatalf("Expected read bytes '5', got %v", len(buf))
	}
}

func TestFGetObjectReturnSuccessIfFileNameLengthIs255(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
		w.Header().Set("Content-Length", "5")
		w.Header().Set("Etag", "abc123")
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

	localFilePath := filepath.Join(t.TempDir(), strings.Repeat("a", 255))
	if len(filepath.Base(localFilePath)) != 255 {
		t.Fatalf("Expected file name length 255, got %v", len(filepath.Base(localFilePath)))
	}

	err = clnt.FGetObject(context.Background(), "bucketName", "objectName", localFilePath, GetObjectOptions{})
	if err != nil {
		t.Fatal(err)
	}

	buf, err := os.ReadFile(localFilePath)
	if err != nil {
		t.Fatalf("Expected 'nil', got %v", err)
	}

	if len(buf) != 5 {
		t.Fatalf("Expected read bytes '5', got %v", len(buf))
	}
}
