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
	"net/url"
	"testing"

	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Tests that StatObject returns the delete-marker ObjectInfo fields
// (VersionID, IsDeleteMarker) and the MethodNotAllowed error code when a
// versioned HEAD hits a delete marker (HTTP 405).
func TestStatObjectDeleteMarker(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(amzDeleteMarker, "true")
		w.Header().Set(amzVersionID, "test-version-id")
		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	clnt, err := New(u.Host, &Options{
		Creds:  credentials.NewStaticV4("foo", "foo12345", ""),
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}

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
	if !objInfo.IsDeleteMarker {
		t.Error("expected IsDeleteMarker to be true")
	}
	if objInfo.VersionID != "test-version-id" {
		t.Errorf("VersionID = %q, want %q", objInfo.VersionID, "test-version-id")
	}
}

// Tests that a 405 response missing either half of the delete-marker
// shape (the x-amz-delete-marker header, or a version-targeted stat)
// falls through to the generic error path with the raw status code.
func TestStatObjectMethodNotAllowedGeneric(t *testing.T) {
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
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if tt.deleteMarker {
					w.Header().Set(amzDeleteMarker, "true")
				}
				w.Header().Set(amzVersionID, "test-version-id")
				w.WriteHeader(http.StatusMethodNotAllowed)
			}))
			defer srv.Close()

			u, err := url.Parse(srv.URL)
			if err != nil {
				t.Fatal(err)
			}

			clnt, err := New(u.Host, &Options{
				Creds:  credentials.NewStaticV4("foo", "foo12345", ""),
				Region: "us-east-1",
			})
			if err != nil {
				t.Fatal(err)
			}

			objInfo, err := clnt.StatObject(context.Background(), "bucket-name", "object-name",
				StatObjectOptions{VersionID: tt.versionID})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if errResp := ToErrorResponse(err); errResp.Code != "405 Method Not Allowed" {
				t.Errorf("error code = %q, want %q", errResp.Code, "405 Method Not Allowed")
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
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Last-Modified", "Thu, 30 Jul 2026 00:00:00 GMT")
				w.Header().Set("ETag", `"deadbeef"`)
				w.Header().Set(amzVersionID, "test-version-id")
				w.WriteHeader(status)
			}))
			defer srv.Close()

			u, err := url.Parse(srv.URL)
			if err != nil {
				t.Fatal(err)
			}

			clnt, err := New(u.Host, &Options{
				Creds:  credentials.NewStaticV4("foo", "foo12345", ""),
				Region: "us-east-1",
			})
			if err != nil {
				t.Fatal(err)
			}

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
	if objInfo.IsDeleteMarker || objInfo.VersionID != "" || objInfo.ETag != "" {
		t.Errorf("expected zero ObjectInfo, got %+v", objInfo)
	}
}

// Tests that StatObject surfaces the delete-marker and replication-ready
// headers on a generic error response, e.g. HEAD on an object whose
// latest version is a delete marker (HTTP 404).
func TestStatObjectErrorHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(amzDeleteMarker, "true")
		w.Header().Set(amzVersionID, "test-version-id")
		w.Header().Set(minioTgtReplicationReady, "true")
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	clnt, err := New(u.Host, &Options{
		Creds:  credentials.NewStaticV4("foo", "foo12345", ""),
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	objInfo, err := clnt.StatObject(context.Background(), "bucket-name", "object-name", StatObjectOptions{})
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
}
