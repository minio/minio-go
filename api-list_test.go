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
	"time"

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
// return, with values passed through verbatim: an RFC 2047-looking value
// must NOT be MIME-decoded.
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

// TestListObjectsWithRestoreStatus verifies that WithRestoreStatus asks for
// the optional attribute on every list API and that the reply lands in
// ObjectInfo.Restore, the field StatObject fills from x-amz-restore.
func TestListObjectsWithRestoreStatus(t *testing.T) {
	const restoreStatusXML = `<RestoreStatus>` +
		`<IsRestoreInProgress>false</IsRestoreInProgress>` +
		`<RestoreExpiryDate>2026-09-11T12:00:00.000Z</RestoreExpiryDate>` +
		`</RestoreStatus>`
	wantExpiry := time.Date(2026, time.September, 11, 12, 0, 0, 0, time.UTC)

	var gotAttributes, gotCustom string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAttributes = r.Header.Get("x-amz-optional-object-attributes")
		gotCustom = r.Header.Get("x-custom")
		var entry string
		if gotAttributes == "RestoreStatus" {
			entry = restoreStatusXML
		}
		w.Header().Set("Content-Type", "application/xml")
		const key = `<Key>archived.txt</Key><LastModified>2026-09-10T00:00:00.000Z</LastModified>`
		if _, versioned := r.URL.Query()["versions"]; versioned {
			w.Write([]byte(`<ListVersionsResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">` +
				`<IsTruncated>false</IsTruncated>` +
				`<Version>` + key + `<IsLatest>true</IsLatest><VersionId>null</VersionId>` + entry + `</Version>` +
				`</ListVersionsResult>`))
			return
		}
		w.Write([]byte(`<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">` +
			`<IsTruncated>false</IsTruncated>` +
			`<Contents>` + key + entry + `</Contents>` +
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

	for _, base := range []ListObjectsOptions{
		{Recursive: true},
		{Recursive: true, UseV1: true},
		{Recursive: true, WithVersions: true},
	} {
		for _, want := range []bool{false, true} {
			opts := base
			opts.WithRestoreStatus = want
			opts.Set("x-custom", "kept")

			var seen int
			for obj := range client.ListObjects(t.Context(), "test-bucket", opts) {
				if obj.Err != nil {
					t.Fatalf("%+v: %v", opts, obj.Err)
				}
				seen++
				switch {
				case !want && obj.Restore != nil:
					t.Errorf("%+v: Restore set without asking: %+v", opts, obj.Restore)
				case want && obj.Restore == nil:
					t.Errorf("%+v: Restore not parsed", opts)
				case want && (obj.Restore.OngoingRestore || !obj.Restore.ExpiryTime.Equal(wantExpiry)):
					t.Errorf("%+v: Restore = %+v, want expiry %v", opts, obj.Restore, wantExpiry)
				}
			}
			if seen != 1 {
				t.Fatalf("%+v: expected 1 object, got %d", opts, seen)
			}

			wantAttributes := ""
			if want {
				wantAttributes = "RestoreStatus"
			}
			if gotAttributes != wantAttributes {
				t.Errorf("%+v: x-amz-optional-object-attributes = %q, want %q", opts, gotAttributes, wantAttributes)
			}
			if gotCustom != "kept" {
				t.Errorf("%+v: caller header dropped", opts)
			}
			// The caller's headers must not pick up the attribute.
			if got := opts.headers.Get(amzOptionalObjectAttributes); got != "" {
				t.Errorf("%+v: caller headers mutated with %q", opts, got)
			}
		}
	}
}

// TestListObjectsChecksums verifies that every list API reports checksums in
// the same ObjectInfo fields. AWS returns <ChecksumAlgorithm> and
// <ChecksumType> for every object that has a checksum, and never a value;
// AiStor adds the values when listing with WithMetadata. When only a value is
// returned the algorithm is derived from it.
func TestListObjectsChecksums(t *testing.T) {
	const sha256Value = "uU0nuZNNPgilLlLX2n2r+sSE7+N6U4DukIj3rOLvzek="

	// ObjectInfo is not comparable, so only the checksum fields are compared.
	type checksums struct {
		CRC32, CRC32C, SHA1, SHA256, CRC64NVME    string
		MD5, SHA512, XXHash64, XXHash3, XXHash128 string
		Algorithm, Mode                           string
	}
	of := func(o ObjectInfo) checksums {
		return checksums{
			CRC32: o.ChecksumCRC32, CRC32C: o.ChecksumCRC32C, SHA1: o.ChecksumSHA1,
			SHA256: o.ChecksumSHA256, CRC64NVME: o.ChecksumCRC64NVME, MD5: o.ChecksumMD5,
			SHA512: o.ChecksumSHA512, XXHash64: o.ChecksumXXHash64, XXHash3: o.ChecksumXXHash3,
			XXHash128: o.ChecksumXXHash128, Algorithm: o.ChecksumAlgorithm, Mode: o.ChecksumMode,
		}
	}

	tests := []struct {
		name  string
		entry string
		want  checksums
	}{
		{
			name:  "algorithm and mode only",
			entry: `<ChecksumAlgorithm>CRC32C</ChecksumAlgorithm><ChecksumType>FULL_OBJECT</ChecksumType>`,
			want:  checksums{Algorithm: "CRC32C", Mode: "FULL_OBJECT"},
		},
		{
			name: "value included",
			entry: `<ChecksumCRC64NVME>uWTmruzHkQI=</ChecksumCRC64NVME>` +
				`<ChecksumAlgorithm>CRC64NVME</ChecksumAlgorithm><ChecksumType>FULL_OBJECT</ChecksumType>`,
			want: checksums{CRC64NVME: "uWTmruzHkQI=", Algorithm: "CRC64NVME", Mode: "FULL_OBJECT"},
		},
		{
			name:  "algorithm derived from value",
			entry: `<ChecksumSHA256>` + sha256Value + `</ChecksumSHA256>`,
			want:  checksums{SHA256: sha256Value, Algorithm: "SHA256"},
		},
		{
			// The xxhash elements are spelled in all caps by the server.
			name:  "xxhash element names",
			entry: `<ChecksumXXHASH3>SqjOZg2iBAI=</ChecksumXXHASH3><ChecksumType>COMPOSITE</ChecksumType>`,
			want:  checksums{XXHash3: "SqjOZg2iBAI=", Algorithm: "XXHASH3", Mode: "COMPOSITE"},
		},
		{
			name:  "no checksum",
			entry: ``,
			want:  checksums{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/xml")
				const key = `<Key>hello.txt</Key><LastModified>2025-01-01T00:00:00.000Z</LastModified>`
				if _, versioned := r.URL.Query()["versions"]; versioned {
					w.Write([]byte(`<ListVersionsResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">` +
						`<IsTruncated>false</IsTruncated>` +
						`<Version>` + key + `<IsLatest>true</IsLatest><VersionId>null</VersionId>` + test.entry + `</Version>` +
						`</ListVersionsResult>`))
					return
				}
				w.Write([]byte(`<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">` +
					`<IsTruncated>false</IsTruncated>` +
					`<Contents>` + key + test.entry + `</Contents>` +
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

			for _, opts := range []ListObjectsOptions{
				{Recursive: true},
				{Recursive: true, UseV1: true},
				{Recursive: true, WithVersions: true},
			} {
				var seen int
				for obj := range client.ListObjects(t.Context(), "test-bucket", opts) {
					if obj.Err != nil {
						t.Fatalf("%+v: %v", opts, obj.Err)
					}
					seen++
					if got := of(obj); got != test.want {
						t.Errorf("%+v: got %+v, want %+v", opts, got, test.want)
					}
				}
				if seen != 1 {
					t.Fatalf("%+v: expected 1 object, got %d", opts, seen)
				}
			}
		})
	}
}
