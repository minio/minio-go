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
	"reflect"
	"testing"

	"github.com/minio/minio-go/v7/pkg/credentials"
)

// newTestStatClient returns a Client pointed at an httptest server that
// serves handler; the server is closed via t.Cleanup.
func newTestStatClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	clnt, err := New(srv.Listener.Addr().String(), &Options{
		Creds:  credentials.NewStaticV4("foo", "foo12345", ""),
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	return clnt
}

// Tests that StatObject returns the delete-marker ObjectInfo fields
// (VersionID and IsDeleteMarker — ReplicationReady is deliberately not
// merged into this return) and the MethodNotAllowed error code when a
// versioned HEAD hits a delete marker (HTTP 405).
func TestStatObjectDeleteMarker(t *testing.T) {
	clnt := newTestStatClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(amzDeleteMarker, "true")
		w.Header().Set(amzVersionID, "test-version-id")
		w.Header().Set(minioTgtReplicationReady, "true")
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	objInfo, err := clnt.StatObject(context.Background(), "bucket-name", "object-name",
		StatObjectOptions{VersionID: "test-version-id"})
	if err == nil {
		t.Fatal("expected error for delete marker, got nil")
	}
	errResp := ToErrorResponse(err)
	if errResp.Code != MethodNotAllowed {
		t.Errorf("error code = %q, want %q", errResp.Code, MethodNotAllowed)
	}
	if errResp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("error status = %d, want %d", errResp.StatusCode, http.StatusMethodNotAllowed)
	}
	if errResp.BucketName != "bucket-name" || errResp.Key != "object-name" {
		t.Errorf("error bucket/key = %q/%q, want %q/%q",
			errResp.BucketName, errResp.Key, "bucket-name", "object-name")
	}
	if !objInfo.IsDeleteMarker {
		t.Error("expected IsDeleteMarker to be true")
	}
	if objInfo.VersionID != "test-version-id" {
		t.Errorf("VersionID = %q, want %q", objInfo.VersionID, "test-version-id")
	}
	if objInfo.ReplicationReady {
		t.Error("expected ReplicationReady to stay false on the delete-marker return")
	}
}

// Tests that a 405 response missing either half of the delete-marker
// shape (the x-amz-delete-marker header, or a version-targeted stat)
// falls through to the generic error path with the raw status code.
func TestStatObjectMethodNotAllowedGeneric(t *testing.T) {
	const wantCode = "405 Method Not Allowed"
	tests := []struct {
		name         string
		deleteMarker bool
		versionID    string
	}{
		{"no delete-marker header", false, "test-version-id"},
		{"no version id", true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clnt := newTestStatClient(t, func(w http.ResponseWriter, _ *http.Request) {
				if tt.deleteMarker {
					w.Header().Set(amzDeleteMarker, "true")
				}
				w.Header().Set(amzVersionID, "test-version-id")
				w.WriteHeader(http.StatusMethodNotAllowed)
			})

			objInfo, err := clnt.StatObject(context.Background(), "bucket-name", "object-name",
				StatObjectOptions{VersionID: tt.versionID})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if errResp := ToErrorResponse(err); errResp.Code != wantCode {
				t.Errorf("error code = %q, want %q", errResp.Code, wantCode)
			}
			if objInfo.IsDeleteMarker != tt.deleteMarker {
				t.Errorf("IsDeleteMarker = %v, want %v", objInfo.IsDeleteMarker, tt.deleteMarker)
			}
			if objInfo.VersionID != "test-version-id" {
				t.Errorf("VersionID = %q, want %q", objInfo.VersionID, "test-version-id")
			}
		})
	}
}

// Tests that 202 and 204 responses, which executeMethod treats as
// success, are parsed like a 200 instead of being converted into errors.
func TestStatObjectNoContentSuccess(t *testing.T) {
	for _, status := range []int{http.StatusAccepted, http.StatusNoContent} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			clnt := newTestStatClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Last-Modified", "Thu, 30 Jul 2026 00:00:00 GMT")
				w.Header().Set("ETag", `"deadbeef"`)
				w.Header().Set(amzVersionID, "test-version-id")
				w.WriteHeader(status)
			})

			objInfo, err := clnt.StatObject(context.Background(), "bucket-name", "object-name", StatObjectOptions{})
			if err != nil {
				t.Fatalf("expected nil error for %d, got %v", status, err)
			}
			if objInfo.ETag != "deadbeef" {
				t.Errorf("ETag = %q, want %q", objInfo.ETag, "deadbeef")
			}
			if objInfo.VersionID != "test-version-id" {
				t.Errorf("VersionID = %q, want %q", objInfo.VersionID, "test-version-id")
			}
		})
	}
}

// Tests that StatObject returns a zero ObjectInfo when the request fails
// before any response is received.
func TestStatObjectNoResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	addr := srv.Listener.Addr().String()
	srv.Close()

	clnt, err := New(addr, &Options{
		Creds:      credentials.NewStaticV4("foo", "foo12345", ""),
		Region:     "us-east-1",
		MaxRetries: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	objInfo, err := clnt.StatObject(context.Background(), "bucket-name", "object-name", StatObjectOptions{})
	if err == nil {
		t.Fatal("expected error for unreachable endpoint, got nil")
	}
	if !reflect.DeepEqual(objInfo, ObjectInfo{}) {
		t.Errorf("expected zero ObjectInfo, got %+v", objInfo)
	}
}

// Tests that StatObject surfaces the delete-marker and replication-ready
// headers on a generic error response, e.g. HEAD on an object whose
// latest version is a delete marker (HTTP 404) — and that the
// IsReplicationReadyForDeleteMarker option puts the matching check
// header on the request.
func TestStatObjectErrorHeaders(t *testing.T) {
	var gotReplicationReadyCheck string
	clnt := newTestStatClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotReplicationReadyCheck = r.Header.Get(isMinioTgtReplicationReady)
		w.Header().Set(amzDeleteMarker, "true")
		w.Header().Set(amzVersionID, "test-version-id")
		w.Header().Set(minioTgtReplicationReady, "true")
		w.WriteHeader(http.StatusNotFound)
	})

	objInfo, err := clnt.StatObject(context.Background(), "bucket-name", "object-name",
		StatObjectOptions{Internal: AdvancedGetOptions{IsReplicationReadyForDeleteMarker: true}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errResp := ToErrorResponse(err); errResp.Code != NoSuchKey {
		t.Errorf("error code = %q, want %q", errResp.Code, NoSuchKey)
	}
	if !objInfo.IsDeleteMarker {
		t.Error("expected IsDeleteMarker to be true")
	}
	if objInfo.VersionID != "test-version-id" {
		t.Errorf("VersionID = %q, want %q", objInfo.VersionID, "test-version-id")
	}
	if !objInfo.ReplicationReady {
		t.Error("expected ReplicationReady to be true")
	}
	if gotReplicationReadyCheck != "true" {
		t.Errorf("request header %s = %q, want %q",
			isMinioTgtReplicationReady, gotReplicationReadyCheck, "true")
	}
}

// Tests that the delete-marker branch is gated on the 405 status: a
// non-405 error carrying the same delete-marker and version shape must
// still take the generic path.
func TestStatObjectDeleteMarkerNon405(t *testing.T) {
	clnt := newTestStatClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(amzDeleteMarker, "true")
		w.Header().Set(amzVersionID, "test-version-id")
		w.WriteHeader(http.StatusNotFound)
	})

	_, err := clnt.StatObject(context.Background(), "bucket-name", "object-name",
		StatObjectOptions{VersionID: "test-version-id"})
	if errResp := ToErrorResponse(err); errResp.Code != NoSuchKey {
		t.Errorf("error code = %q, want %q", errResp.Code, NoSuchKey)
	}
}
