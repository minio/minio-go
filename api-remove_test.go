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

func TestRemoveObjectSendsTheConditionItWasGiven(t *testing.T) {
	for _, test := range []struct {
		name  string
		etag  string
		unset bool
		want  string
	}{
		{name: "no condition", unset: true},
		{name: "an etag as ObjectInfo carries it", etag: "d41d8cd98f00b204e9800998ecf8427e", want: `"d41d8cd98f00b204e9800998ecf8427e"`},
		{name: "an etag already quoted", etag: `"d41d8cd98f00b204e9800998ecf8427e"`, want: `"d41d8cd98f00b204e9800998ecf8427e"`},
		{name: "a multipart etag", etag: "d41d8cd98f00b204e9800998ecf8427e-7", want: `"d41d8cd98f00b204e9800998ecf8427e-7"`},
		{name: "whatever etag the object has", etag: "*", want: "*"},
		{name: "a weak etag, which carries its own quotes", etag: `W/"d41d8cd98f00b204e9800998ecf8427e"`, want: `W/"d41d8cd98f00b204e9800998ecf8427e"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			var got http.Header
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = r.Header.Clone()
				w.WriteHeader(http.StatusNoContent)
			}))
			defer srv.Close()

			clnt, err := New(srv.Listener.Addr().String(), &Options{Region: "us-east-1"})
			if err != nil {
				t.Fatal(err)
			}

			opts := RemoveObjectOptions{}
			if !test.unset {
				opts.SetMatchETag(test.etag)
			}
			if err := clnt.RemoveObject(context.Background(), "bucket", "object", opts); err != nil {
				t.Fatalf("RemoveObject: %v", err)
			}
			if value := got.Get("If-Match"); value != test.want {
				t.Errorf("If-Match = %q, want %q", value, test.want)
			}
		})
	}
}

func TestRemoveObjectReportsAConditionThatDidNotHold(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusPreconditionFailed)
	}))
	defer srv.Close()

	clnt, err := New(srv.Listener.Addr().String(), &Options{Region: "us-east-1"})
	if err != nil {
		t.Fatal(err)
	}

	opts := RemoveObjectOptions{}
	opts.SetMatchETag("d41d8cd98f00b204e9800998ecf8427e")
	err = clnt.RemoveObject(context.Background(), "bucket", "object", opts)
	if err == nil {
		t.Fatal("a removal the server refused was reported as made")
	}
	if code := ToErrorResponse(err).Code; code != "PreconditionFailed" {
		t.Errorf("code = %q, want %q", code, "PreconditionFailed")
	}
}

// TestRemoveObjectsWithIterReportsErrorIn200Response checks that a bulk delete
// which the server answers with 200 OK and an error document in the body is
// reported as failed for every object in the batch.
func TestRemoveObjectsWithIterReportsErrorIn200Response(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>` +
			`<Error><Code>InternalError</Code><Message>We encountered an internal error. Please try again.</Message>` +
			`<BucketName>bucket</BucketName><RequestId>1</RequestId><HostId>h</HostId></Error>`))
	}))
	defer srv.Close()

	clnt, err := New(srv.Listener.Addr().String(), &Options{Region: "us-east-1", MaxRetries: 1})
	if err != nil {
		t.Fatal(err)
	}

	keys := []string{"a", "b"}
	objects := func(yield func(ObjectInfo) bool) {
		for _, key := range keys {
			if !yield(ObjectInfo{Key: key}) {
				return
			}
		}
	}

	results, err := clnt.RemoveObjectsWithIter(context.Background(), "bucket", objects, RemoveObjectsOptions{})
	if err != nil {
		t.Fatal(err)
	}

	var got []string
	for res := range results {
		if code := ToErrorResponse(res.Err).Code; code != "InternalError" {
			t.Errorf("object %q: err = %v, want code %q", res.ObjectName, res.Err, "InternalError")
		}
		got = append(got, res.ObjectName)
	}
	if len(got) != len(keys) {
		t.Fatalf("failed objects = %q, want %q", got, keys)
	}
}
