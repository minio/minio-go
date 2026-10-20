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
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestAbortFailedUploadIgnoresCanceledContext(t *testing.T) {
	var aborts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && r.URL.Query().Get("uploadId") == "upload-1" {
			aborts.Add(1)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	clnt, err := New(srv.Listener.Addr().String(), &Options{Region: "us-east-1"})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	clnt.abortFailedUpload(ctx, "bucket", "object", "upload-1")

	if n := aborts.Load(); n != 1 {
		t.Fatalf("expected 1 abort request, got %d", n)
	}
}

// TestRemoveObjectsInvalidXMLCharFailureReportedOnce checks that a bulk delete
// reports one result per object for keys that fall back to a single DELETE
// request because they cannot be carried in the multi-delete XML body.
func TestRemoveObjectsInvalidXMLCharFailureReportedOnce(t *testing.T) {
	var deletes atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes.Add(1)
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`<Error><Code>AccessDenied</Code><Message>Access Denied.</Message>` +
				`<BucketName>bucket</BucketName><Key>bad</Key><RequestId>1</RequestId><HostId>h</HostId></Error>`))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	clnt, err := New(srv.Listener.Addr().String(), &Options{Region: "us-east-1"})
	if err != nil {
		t.Fatal(err)
	}

	// A control character cannot appear in an XML 1.0 document, so this key is
	// deleted with its own DELETE request instead of the multi-delete API.
	const key = "bad\x01key"

	t.Run("RemoveObjects", func(t *testing.T) {
		objectsCh := make(chan ObjectInfo, 1)
		objectsCh <- ObjectInfo{Key: key}
		close(objectsCh)

		var failures []RemoveObjectError
		for rmErr := range clnt.RemoveObjects(context.Background(), "bucket", objectsCh, RemoveObjectsOptions{}) {
			failures = append(failures, rmErr)
		}
		if len(failures) != 1 {
			t.Fatalf("got %d error(s) for one failed object, want 1: %v", len(failures), failures)
		}
		if failures[0].Err == nil {
			t.Fatal("got a nil error for a failed delete")
		}
	})

	t.Run("RemoveObjectsWithResult", func(t *testing.T) {
		objectsCh := make(chan ObjectInfo, 1)
		objectsCh <- ObjectInfo{Key: key}
		close(objectsCh)

		var results []RemoveObjectResult
		for res := range clnt.RemoveObjectsWithResult(context.Background(), "bucket", objectsCh, RemoveObjectsOptions{}) {
			results = append(results, res)
		}
		if len(results) != 1 {
			t.Fatalf("got %d result(s) for one failed object, want 1: %v", len(results), results)
		}
		if results[0].Err == nil {
			t.Fatal("got a nil error for a failed delete")
		}
	})

	// One DELETE per object: the duplicate results are not retried requests.
	if n := deletes.Load(); n != 2 {
		t.Fatalf("expected 2 DELETE requests (one per object), got %d", n)
	}
}
