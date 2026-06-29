/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2015-2026 MinIO, Inc.
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
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/minio/minio-go/v7/pkg/credentials"
)

// TestUploadPartCopyChecksum5924 validates that CopyObjectPart surfaces the
// per-part checksum returned in the UploadPartCopy CopyPartResult (AIStor
// #5924) on the returned CompletePart. It uses a mock endpoint so it runs
// deterministically in CI without a live server.
func TestUploadPartCopyChecksum5924(t *testing.T) {
	const wantCRC32C = "yZRlqg=="
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// UploadPartCopy: PUT with uploadId + partNumber + copy-source header.
		if r.Method == http.MethodPut && r.URL.Query().Get("uploadId") != "" {
			w.Header().Set("Content-Type", "application/xml")
			io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>`+
				`<CopyPartResult>`+
				`<ETag>&quot;3858f62230ac3c915f300c664312c11f&quot;</ETag>`+
				`<LastModified>2026-01-01T00:00:00.000Z</LastModified>`+
				`<ChecksumCRC32C>`+wantCRC32C+`</ChecksumCRC32C>`+
				`</CopyPartResult>`)
			return
		}
		// Bucket location lookup: answer with a valid (us-east-1) constraint.
		if r.Method == http.MethodGet && r.URL.Query().Has("location") {
			w.Header().Set("Content-Type", "application/xml")
			io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>`+
				`<LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/"></LocationConstraint>`)
			return
		}
		// Anything else: succeed with an empty body.
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	core, err := NewCore(u.Host, &Options{
		Creds:  credentials.NewStaticV4("ak", "sk", ""),
		Secure: false,
	})
	if err != nil {
		t.Fatalf("NewCore: %v", err)
	}

	part, err := core.CopyObjectPart(context.Background(),
		"src-bucket", "src", "dst-bucket", "dst", "upload-id", 1, 0, -1, nil)
	if err != nil {
		t.Fatalf("CopyObjectPart: %v", err)
	}

	// The crux of #5924: the part checksum must be surfaced on CompletePart so
	// CompleteMultipartUpload can echo it (pre-fix it was silently dropped).
	if part.ChecksumCRC32C != wantCRC32C {
		t.Fatalf("ChecksumCRC32C = %q, want %q", part.ChecksumCRC32C, wantCRC32C)
	}
	if part.PartNumber != 1 {
		t.Fatalf("PartNumber = %d, want 1", part.PartNumber)
	}
}

// TestCopyObjectResultSetChecksums verifies setChecksums copies every checksum
// flavour from the parsed CopyPartResult onto the CompletePart.
func TestCopyObjectResultSetChecksums(t *testing.T) {
	r := copyObjectResult{
		ChecksumCRC32:     "crc32",
		ChecksumCRC32C:    "crc32c",
		ChecksumSHA1:      "sha1",
		ChecksumSHA256:    "sha256",
		ChecksumCRC64NVME: "crc64nvme",
		ChecksumMD5:       "md5",
		ChecksumSHA512:    "sha512",
		ChecksumXXHash64:  "xxh64",
		ChecksumXXHash3:   "xxh3",
		ChecksumXXHash128: "xxh128",
	}
	var p CompletePart
	r.setChecksums(&p)

	for _, tc := range []struct {
		name      string
		got, want string
	}{
		{"CRC32", p.ChecksumCRC32, r.ChecksumCRC32},
		{"CRC32C", p.ChecksumCRC32C, r.ChecksumCRC32C},
		{"SHA1", p.ChecksumSHA1, r.ChecksumSHA1},
		{"SHA256", p.ChecksumSHA256, r.ChecksumSHA256},
		{"CRC64NVME", p.ChecksumCRC64NVME, r.ChecksumCRC64NVME},
		{"MD5", p.ChecksumMD5, r.ChecksumMD5},
		{"SHA512", p.ChecksumSHA512, r.ChecksumSHA512},
		{"XXHash64", p.ChecksumXXHash64, r.ChecksumXXHash64},
		{"XXHash3", p.ChecksumXXHash3, r.ChecksumXXHash3},
		{"XXHash128", p.ChecksumXXHash128, r.ChecksumXXHash128},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}
