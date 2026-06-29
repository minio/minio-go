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
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"hash/crc32"
	"os"
	"testing"

	"github.com/minio/minio-go/v7/pkg/credentials"
)

// TestUploadPartCopyChecksum5924 validates that UploadPartCopy returns the
// copied part's checksum in CopyPartResult (AIStor issue #5924), so a
// checksummed multipart upload built from copied parts can echo the part
// checksum on CompleteMultipartUpload. Runs only against a live server
// configured via SERVER_ENDPOINT/ACCESS_KEY/SECRET_KEY.
func TestUploadPartCopyChecksum5924(t *testing.T) {
	endpoint := os.Getenv("SERVER_ENDPOINT")
	if endpoint == "" {
		t.Skip("SERVER_ENDPOINT not set; skipping live-server validation")
	}

	core, err := NewCore(endpoint, &Options{
		Creds:  credentials.NewStaticV4(os.Getenv("ACCESS_KEY"), os.Getenv("SECRET_KEY"), ""),
		Secure: os.Getenv("ENABLE_HTTPS") == "1",
	})
	if err != nil {
		t.Fatalf("NewCore: %v", err)
	}

	ctx := context.Background()
	bucket := "val5924-" + randomSuffix(t)
	if err := core.MakeBucket(ctx, bucket, MakeBucketOptions{}); err != nil {
		t.Fatalf("MakeBucket: %v", err)
	}
	defer func() {
		core.RemoveObject(ctx, bucket, "dst", RemoveObjectOptions{})
		core.RemoveObject(ctx, bucket, "src", RemoveObjectOptions{})
		core.RemoveBucket(ctx, bucket)
	}()

	srcData := make([]byte, 5*1024*1024)
	if _, err := rand.Read(srcData); err != nil {
		t.Fatalf("rand: %v", err)
	}
	if _, err := core.PutObject(ctx, bucket, "src", bytes.NewReader(srcData),
		int64(len(srcData)), "", "", PutObjectOptions{}); err != nil {
		t.Fatalf("PutObject(src): %v", err)
	}

	uploadID, err := core.NewMultipartUpload(ctx, bucket, "dst", PutObjectOptions{
		UserMetadata: map[string]string{"x-amz-checksum-algorithm": "CRC32C"},
	})
	if err != nil {
		t.Fatalf("NewMultipartUpload: %v", err)
	}

	part, err := core.CopyObjectPart(ctx, bucket, "src", bucket, "dst", uploadID, 1, 0, -1, nil)
	if err != nil {
		t.Fatalf("CopyObjectPart: %v", err)
	}

	// The crux of #5924: the part checksum must come back in CopyPartResult.
	want := crc32cBase64(srcData)
	if part.ChecksumCRC32C == "" {
		t.Fatalf("CopyPartResult carried no ChecksumCRC32C - server and/or minio-go fix for #5924 missing")
	}
	if part.ChecksumCRC32C != want {
		t.Fatalf("part ChecksumCRC32C = %q, want %q", part.ChecksumCRC32C, want)
	}

	// Completing with the echoed checksum must succeed (pre-fix: InvalidPart).
	if _, err := core.CompleteMultipartUpload(ctx, bucket, "dst", uploadID,
		[]CompletePart{part}, PutObjectOptions{}); err != nil {
		t.Fatalf("CompleteMultipartUpload: %v", err)
	}
}

// TestComposeObjectChecksum5924 exercises the high-level ComposeObject path:
// CopyDestOptions.ChecksumType is now carried onto the multipart upload, and
// UploadPartCopy returns each part checksum, so a checksummed compose completes
// (pre-fix it failed with InvalidPart). Runs only against a live server.
func TestComposeObjectChecksum5924(t *testing.T) {
	endpoint := os.Getenv("SERVER_ENDPOINT")
	if endpoint == "" {
		t.Skip("SERVER_ENDPOINT not set; skipping live-server validation")
	}

	c, err := New(endpoint, &Options{
		Creds:  credentials.NewStaticV4(os.Getenv("ACCESS_KEY"), os.Getenv("SECRET_KEY"), ""),
		Secure: os.Getenv("ENABLE_HTTPS") == "1",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	bucket := "val5924c-" + randomSuffix(t)
	if err := c.MakeBucket(ctx, bucket, MakeBucketOptions{}); err != nil {
		t.Fatalf("MakeBucket: %v", err)
	}
	defer func() {
		c.RemoveObject(ctx, bucket, "dst", RemoveObjectOptions{})
		c.RemoveObject(ctx, bucket, "src", RemoveObjectOptions{})
		c.RemoveBucket(ctx, bucket)
	}()

	srcData := make([]byte, 5*1024*1024)
	if _, err := rand.Read(srcData); err != nil {
		t.Fatalf("rand: %v", err)
	}
	if _, err := c.PutObject(ctx, bucket, "src", bytes.NewReader(srcData),
		int64(len(srcData)), PutObjectOptions{}); err != nil {
		t.Fatalf("PutObject(src): %v", err)
	}

	dst := CopyDestOptions{Bucket: bucket, Object: "dst", ChecksumType: ChecksumCRC32C}
	src := CopySrcOptions{Bucket: bucket, Object: "src"}
	if _, err := c.ComposeObject(ctx, dst, src); err != nil {
		t.Fatalf("ComposeObject (checksummed): %v", err)
	}

	attr, err := c.GetObjectAttributes(ctx, bucket, "dst", ObjectAttributesOptions{})
	if err != nil {
		t.Fatalf("GetObjectAttributes: %v", err)
	}
	if attr.Checksum.ChecksumCRC32C == "" {
		t.Fatalf("composed object carries no CRC32C checksum")
	}
}

// crc32cBase64 returns the base64-encoded CRC32C (Castagnoli) of b.
func crc32cBase64(b []byte) string {
	sum := crc32.Checksum(b, crc32.MakeTable(crc32.Castagnoli))
	var raw [4]byte
	binary.BigEndian.PutUint32(raw[:], sum)
	return base64.StdEncoding.EncodeToString(raw[:])
}

// randomSuffix returns a short lower-case hex suffix for unique bucket names.
func randomSuffix(t *testing.T) string {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatalf("rand: %v", err)
	}
	const hex = "0123456789abcdef"
	out := make([]byte, 0, 12)
	for _, x := range b {
		out = append(out, hex[x>>4], hex[x&0xf])
	}
	return string(out)
}
