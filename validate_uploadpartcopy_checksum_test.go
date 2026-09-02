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
	"slices"
	"strconv"
	"strings"
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
			if got := r.URL.Query().Get("partNumber"); got != "1" {
				http.Error(w, "unexpected partNumber", http.StatusBadRequest)
				return
			}
			if r.Header.Get("x-amz-copy-source") == "" {
				http.Error(w, "missing x-amz-copy-source", http.StatusBadRequest)
				return
			}
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
// flavor from the parsed CopyPartResult onto the CompletePart.
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

func TestCopyObjectResponseChecksums(t *testing.T) {
	want := copyObjectResult{
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
		ChecksumType:      "FULL_OBJECT",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("x-amz-copy-source") == "" {
			http.Error(w, "missing x-amz-copy-source", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>`+
			`<CopyObjectResult>`+
			`<ETag>&quot;3858f62230ac3c915f300c664312c11f&quot;</ETag>`+
			`<LastModified>2026-01-01T00:00:00.000Z</LastModified>`+
			`<ChecksumCRC32>`+want.ChecksumCRC32+`</ChecksumCRC32>`+
			`<ChecksumCRC32C>`+want.ChecksumCRC32C+`</ChecksumCRC32C>`+
			`<ChecksumSHA1>`+want.ChecksumSHA1+`</ChecksumSHA1>`+
			`<ChecksumSHA256>`+want.ChecksumSHA256+`</ChecksumSHA256>`+
			`<ChecksumCRC64NVME>`+want.ChecksumCRC64NVME+`</ChecksumCRC64NVME>`+
			`<ChecksumMD5>`+want.ChecksumMD5+`</ChecksumMD5>`+
			`<ChecksumSHA512>`+want.ChecksumSHA512+`</ChecksumSHA512>`+
			`<ChecksumXXHASH64>`+want.ChecksumXXHash64+`</ChecksumXXHASH64>`+
			`<ChecksumXXHASH3>`+want.ChecksumXXHash3+`</ChecksumXXHASH3>`+
			`<ChecksumXXHASH128>`+want.ChecksumXXHash128+`</ChecksumXXHASH128>`+
			`<ChecksumType>`+want.ChecksumType+`</ChecksumType>`+
			`</CopyObjectResult>`)
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	client, err := New(u.Host, &Options{
		Creds:  credentials.NewStaticV4("ak", "sk", ""),
		Secure: false,
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := client.CopyObject(context.Background(),
		CopyDestOptions{Bucket: "dst-bucket", Object: "dst-object"},
		CopySrcOptions{Bucket: "src-bucket", Object: "src-object"})
	if err != nil {
		t.Fatalf("CopyObject: %v", err)
	}

	for _, tc := range []struct {
		name      string
		got, want string
	}{
		{"CRC32", got.ChecksumCRC32, want.ChecksumCRC32},
		{"CRC32C", got.ChecksumCRC32C, want.ChecksumCRC32C},
		{"SHA1", got.ChecksumSHA1, want.ChecksumSHA1},
		{"SHA256", got.ChecksumSHA256, want.ChecksumSHA256},
		{"CRC64NVME", got.ChecksumCRC64NVME, want.ChecksumCRC64NVME},
		{"MD5", got.ChecksumMD5, want.ChecksumMD5},
		{"SHA512", got.ChecksumSHA512, want.ChecksumSHA512},
		{"XXHash64", got.ChecksumXXHash64, want.ChecksumXXHash64},
		{"XXHash3", got.ChecksumXXHash3, want.ChecksumXXHash3},
		{"XXHash128", got.ChecksumXXHash128, want.ChecksumXXHash128},
		{"ChecksumType", got.ChecksumMode, want.ChecksumType},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}

// TestComposeObjectChecksum5924 validates that ComposeObject sets the requested
// checksum algorithm on the multipart upload (so server-side copied parts are
// checksummed) and surfaces the composed object's checksum (AIStor #5924). A
// 10 MiB source with a 5 MiB part size forces the two-part multipart-copy path;
// a mock endpoint keeps it deterministic in CI without a live server.
func TestComposeObjectChecksum5924(t *testing.T) {
	const wantCRC32C = "yZRlqg=="
	srcSize := 10 * 1024 * 1024
	var (
		gotAlgo         string
		gotMode         string
		gotCompleteBody string
		gotRanges       []string
		initCount       int
		completeCount   int
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch {
		// Source stat (HEAD): report a size that needs two parts.
		case r.Method == http.MethodHead:
			w.Header().Set("Content-Length", strconv.Itoa(srcSize))
			w.Header().Set("Last-Modified", "Wed, 01 Jan 2026 00:00:00 GMT")
			w.Header().Set("ETag", `"3858f62230ac3c915f300c664312c11f"`)
			w.WriteHeader(http.StatusOK)
		// Initiate multipart upload (POST ?uploads): record the algorithm header.
		case r.Method == http.MethodPost && q.Has("uploads"):
			initCount++
			gotAlgo = r.Header.Get(amzChecksumAlgo)
			gotMode = r.Header.Get(amzChecksumMode)
			w.Header().Set("Content-Type", "application/xml")
			io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>`+
				`<InitiateMultipartUploadResult>`+
				`<Bucket>dst-bucket</Bucket><Key>dst</Key><UploadId>upload-id</UploadId>`+
				`</InitiateMultipartUploadResult>`)
		// UploadPartCopy (PUT ?uploadId&partNumber + copy-source).
		case r.Method == http.MethodPut && q.Get("uploadId") != "":
			if q.Get("partNumber") == "" {
				http.Error(w, "missing partNumber", http.StatusBadRequest)
				return
			}
			if r.Header.Get("x-amz-copy-source") == "" {
				http.Error(w, "missing x-amz-copy-source", http.StatusBadRequest)
				return
			}
			gotRanges = append(gotRanges, r.Header.Get("x-amz-copy-source-range"))
			w.Header().Set("Content-Type", "application/xml")
			io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>`+
				`<CopyPartResult>`+
				`<ETag>&quot;3858f62230ac3c915f300c664312c11f&quot;</ETag>`+
				`<LastModified>2026-01-01T00:00:00.000Z</LastModified>`+
				`<ChecksumCRC32C>`+wantCRC32C+`</ChecksumCRC32C>`+
				`</CopyPartResult>`)
		// Plain CopyObject (PUT + copy-source, no uploadId): the direct path an
		// empty or single small source takes.
		case r.Method == http.MethodPut && r.Header.Get("x-amz-copy-source") != "":
			w.Header().Set("Content-Type", "application/xml")
			io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>`+
				`<CopyObjectResult>`+
				`<ETag>&quot;3858f62230ac3c915f300c664312c11f&quot;</ETag>`+
				`<LastModified>2026-01-01T00:00:00.000Z</LastModified>`+
				`</CopyObjectResult>`)
		// CompleteMultipartUpload (POST ?uploadId): capture the part bodies and
		// echo the object checksum.
		case r.Method == http.MethodPost && q.Get("uploadId") != "":
			completeCount++
			body, _ := io.ReadAll(r.Body)
			gotCompleteBody = string(body)
			w.Header().Set("Content-Type", "application/xml")
			io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>`+
				`<CompleteMultipartUploadResult>`+
				`<Bucket>dst-bucket</Bucket><Key>dst</Key>`+
				`<ETag>&quot;3858f62230ac3c915f300c664312c11f-2&quot;</ETag>`+
				`<ChecksumCRC32C>`+wantCRC32C+`</ChecksumCRC32C>`+
				`</CompleteMultipartUploadResult>`)
		// Bucket location lookup: answer with a valid (us-east-1) constraint.
		case r.Method == http.MethodGet && q.Has("location"):
			w.Header().Set("Content-Type", "application/xml")
			io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>`+
				`<LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/"></LocationConstraint>`)
		// Anything else: succeed with an empty body.
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	client, err := New(u.Host, &Options{
		Creds:  credentials.NewStaticV4("ak", "sk", ""),
		Secure: false,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	info, err := client.ComposeObject(context.Background(),
		CopyDestOptions{Bucket: "dst-bucket", Object: "dst", ChecksumType: ChecksumCRC32C, PartSize: defaultMinPartSize},
		CopySrcOptions{Bucket: "src-bucket", Object: "src"})
	if err != nil {
		t.Fatalf("ComposeObject: %v", err)
	}

	// The compose half of #5924: the algorithm must be set on the MPU so the
	// server checksums the copied parts, and the composed object must carry it.
	if gotAlgo != "CRC32C" {
		t.Fatalf("multipart init checksum algorithm = %q, want %q", gotAlgo, "CRC32C")
	}
	if info.ChecksumCRC32C != wantCRC32C {
		t.Fatalf("ChecksumCRC32C = %q, want %q", info.ChecksumCRC32C, wantCRC32C)
	}
	// The per-part checksum parsed from CopyPartResult must reach the
	// CompleteMultipartUpload request body as <ChecksumCRC32C> on every part;
	// the 10 MiB source at a 5 MiB part size yields exactly two parts, so a
	// dropped second-part checksum would leave only one occurrence.
	want := "<ChecksumCRC32C>" + wantCRC32C + "</ChecksumCRC32C>"
	if got := strings.Count(gotCompleteBody, want); got != 2 {
		t.Fatalf("CompleteMultipartUpload body has %d %q, want 2; body %q", got, want, gotCompleteBody)
	}
	// Every generated copy range but the last must be at least MinPartSize,
	// otherwise the remote rejects the part.
	wantRanges := []string{"bytes=0-5242879", "bytes=5242880-10485759"}
	if !slices.Equal(gotRanges, wantRanges) {
		t.Fatalf("copy source ranges = %q, want %q", gotRanges, wantRanges)
	}
	// A composite (non-full-object) algorithm must not set the mode header.
	if gotMode != "" {
		t.Fatalf("composite checksum init mode = %q, want empty", gotMode)
	}

	// A full-object checksum type additionally sets the mode header on the MPU
	// init (the dst.ChecksumType.FullObjectRequested() branch).
	if _, err := client.ComposeObject(context.Background(),
		CopyDestOptions{Bucket: "dst-bucket", Object: "dst", ChecksumType: ChecksumFullObjectCRC32C, PartSize: defaultMinPartSize},
		CopySrcOptions{Bucket: "src-bucket", Object: "src"}); err != nil {
		t.Fatalf("ComposeObject (full object): %v", err)
	}
	if gotAlgo != "CRC32C" {
		t.Fatalf("full-object init checksum algorithm = %q, want %q", gotAlgo, "CRC32C")
	}
	if gotMode != "FULL_OBJECT" {
		t.Fatalf("full-object init checksum mode = %q, want %q", gotMode, "FULL_OBJECT")
	}

	// A 6 MiB source at a 5 MiB part size splits evenly into two 3 MiB ranges,
	// both below MinPartSize, so it must be rejected up front rather than
	// rejected by the remote mid-copy. Only the source stat may reach the wire.
	srcSize = 6 * 1024 * 1024
	gotRanges, initCount, completeCount = nil, 0, 0
	if _, err := client.ComposeObject(context.Background(),
		CopyDestOptions{Bucket: "dst-bucket", Object: "dst", PartSize: defaultMinPartSize},
		CopySrcOptions{Bucket: "src-bucket", Object: "src"}); err == nil {
		t.Fatal("ComposeObject: expected a rejection for ranges below the minimum part size")
	}
	if initCount != 0 || len(gotRanges) != 0 || completeCount != 0 {
		t.Fatalf("rejected compose issued %d initiations, %d copies and %d completions, want none",
			initCount, len(gotRanges), completeCount)
	}

	// A part size above the maximum is rejected too.
	if _, err := client.ComposeObject(context.Background(),
		CopyDestOptions{Bucket: "dst-bucket", Object: "dst", PartSize: uint64(defaultMaxPartSize) + 1},
		CopySrcOptions{Bucket: "src-bucket", Object: "src"}); err == nil {
		t.Fatal("ComposeObject: expected a rejection for a part size above the maximum")
	}

	// 2*MinPartSize-1 at a MinPartSize part size splits into a full range plus
	// a one-byte-short tail. That tail is the final range of the final source,
	// which S3 exempts from the minimum, so it must be accepted.
	srcSize = 2*defaultMinPartSize - 1
	gotRanges = nil
	if _, err := client.ComposeObject(context.Background(),
		CopyDestOptions{Bucket: "dst-bucket", Object: "dst", PartSize: defaultMinPartSize},
		CopySrcOptions{Bucket: "src-bucket", Object: "src"}); err != nil {
		t.Fatalf("ComposeObject (short final range): %v", err)
	}
	wantRanges = []string{"bytes=0-5242879", "bytes=5242880-10485758"}
	if !slices.Equal(gotRanges, wantRanges) {
		t.Fatalf("copy source ranges = %q, want %q", gotRanges, wantRanges)
	}

	// An empty source needs no ranges at all, so the split check must not run
	// against a zero part count. ComposeObject copies it directly, without
	// opening a multipart upload.
	srcSize = 0
	gotRanges, initCount, completeCount = nil, 0, 0
	if _, err := client.ComposeObject(context.Background(),
		CopyDestOptions{Bucket: "dst-bucket", Object: "dst", PartSize: defaultMinPartSize},
		CopySrcOptions{Bucket: "src-bucket", Object: "src"}); err != nil {
		t.Fatalf("ComposeObject (empty source): %v", err)
	}
	if len(gotRanges) != 0 {
		t.Fatalf("empty source produced copy ranges %q, want none", gotRanges)
	}
	if initCount != 0 || completeCount != 0 {
		t.Fatalf("empty source issued %d initiations and %d completions, want none (direct copy)",
			initCount, completeCount)
	}
}
