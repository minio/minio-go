/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2015-2017 MinIO, Inc.
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
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/minio/minio-go/v7/pkg/credentials"
)

func TestListObjectVersionsHonorsStartAfter(t *testing.T) {
	startAfter := "b.txt"

	var capturedQuery url.Values
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<ListVersionsResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><IsTruncated>false</IsTruncated></ListVersionsResult>`))
	}))
	defer ts.Close()

	srv, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	client, err := New(srv.Host, &Options{
		Creds:  credentials.NewStaticV4("accesskey", "secretkey", ""),
		Secure: false,
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	for range client.ListObjects(t.Context(), "test-bucket", ListObjectsOptions{
		WithVersions: true,
		StartAfter:   startAfter,
		Recursive:    true,
	}) {
	}

	if capturedQuery.Get("key-marker") != startAfter {
		t.Fatalf("expected key-marker=%q, got %q", startAfter, capturedQuery.Get("key-marker"))
	}
}

// TestListObjectsUserMetadataStripped verifies that listing with WithMetadata
// keeps UserMetadata exactly as returned by the server while
// UserMetadataStripped carries the prefix-stripped keys StatObject would
// return, with values passed through verbatim — an RFC 2047-looking value
// must NOT be MIME-decoded, since list responses carry the stored values.
// Regression test for https://github.com/minio/minio-go/issues/2054.
func TestListObjectsUserMetadataStripped(t *testing.T) {
	const userMetadataXML = `<UserMetadata>` +
		`<X-Amz-Meta-Hello>World</X-Amz-Meta-Hello>` +
		`<X-Amz-Meta-Encoded>=?UTF-8?q?ren=C3=A9?=</X-Amz-Meta-Encoded>` +
		`<content-type>application/octet-stream</content-type>` +
		`<expires>Mon, 01 Jan 0001 00:00:00 GMT</expires>` +
		`</UserMetadata>`

	wantRaw := StringMap{
		"X-Amz-Meta-Hello":   "World",
		"X-Amz-Meta-Encoded": "=?UTF-8?q?ren=C3=A9?=",
		"content-type":       "application/octet-stream",
		"expires":            "Mon, 01 Jan 0001 00:00:00 GMT",
	}
	wantStripped := StringMap{
		"Hello":   "World",
		"Encoded": "=?UTF-8?q?ren=C3=A9?=",
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		if _, versioned := r.URL.Query()["versions"]; versioned {
			w.Write([]byte(`<ListVersionsResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">` +
				`<IsTruncated>false</IsTruncated>` +
				`<Version><Key>hello.txt</Key><LastModified>2025-01-01T00:00:00.000Z</LastModified>` +
				`<IsLatest>true</IsLatest><VersionId>null</VersionId>` + userMetadataXML + `</Version>` +
				`</ListVersionsResult>`))
			return
		}
		w.Write([]byte(`<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">` +
			`<IsTruncated>false</IsTruncated>` +
			`<Contents><Key>hello.txt</Key><LastModified>2025-01-01T00:00:00.000Z</LastModified>` + userMetadataXML + `</Contents>` +
			`</ListBucketResult>`))
	}))
	defer ts.Close()

	srv, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	client, err := New(srv.Host, &Options{
		Creds:  credentials.NewStaticV4("accesskey", "secretkey", ""),
		Secure: false,
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, withVersions := range []bool{false, true} {
		var seen int
		for obj := range client.ListObjects(t.Context(), "test-bucket", ListObjectsOptions{
			WithMetadata: true,
			WithVersions: withVersions,
			Recursive:    true,
		}) {
			if obj.Err != nil {
				t.Fatalf("withVersions=%v: %v", withVersions, obj.Err)
			}
			seen++
			if !maps.Equal(obj.UserMetadata, wantRaw) {
				t.Errorf("withVersions=%v: UserMetadata changed, got %v, want %v", withVersions, obj.UserMetadata, wantRaw)
			}
			if !maps.Equal(obj.UserMetadataStripped, wantStripped) {
				t.Errorf("withVersions=%v: UserMetadataStripped got %v, want %v", withVersions, obj.UserMetadataStripped, wantStripped)
			}
		}
		if seen != 1 {
			t.Fatalf("withVersions=%v: expected 1 object, got %d", withVersions, seen)
		}
	}
}
