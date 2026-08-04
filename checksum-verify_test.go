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
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/minio/minio-go/v7/pkg/credentials"
)

func testCRC32CBase64(b []byte) string {
	sum := crc32.Checksum(b, crc32.MakeTable(crc32.Castagnoli))
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], sum)
	return base64.StdEncoding.EncodeToString(buf[:])
}

// checksumStub mimics server checksum-header behavior: HEAD and un-ranged GET
// responses carry x-amz-checksum-crc32c and x-amz-checksum-type, ranged (206)
// responses carry neither.
type checksumStub struct {
	body     []byte
	checksum string
	mode     string
}

func (s *checksumStub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("ETag", `"checksum-stub-etag"`)
	h.Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
	h.Set("Accept-Ranges", "bytes")
	body := s.body
	ranged := false
	if rng := r.Header.Get("Range"); rng != "" {
		var off int64
		if _, err := fmt.Sscanf(rng, "bytes=%d-", &off); err != nil || off < 0 || off > int64(len(body)) {
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}
		h.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", off, len(s.body)-1, len(s.body)))
		body = body[off:]
		ranged = true
	} else {
		if s.checksum != "" {
			h.Set("x-amz-checksum-crc32c", s.checksum)
		}
		if s.mode != "" {
			h.Set("x-amz-checksum-type", s.mode)
		}
	}
	h.Set("Content-Length", strconv.Itoa(len(body)))
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	if ranged {
		w.WriteHeader(http.StatusPartialContent)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	_, _ = w.Write(body)
}

func checksumStubClient(t *testing.T, s *checksumStub) *Client {
	t.Helper()
	srv := httptest.NewServer(s)
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	c, err := New(u.Host, &Options{
		Creds:  credentials.NewStaticV4("stub", "stubsecret", ""),
		Secure: false,
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func testChecksumBody(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i * 11)
	}
	return b
}

func TestGetObjectChecksumVerification(t *testing.T) {
	body := testChecksumBody(1 << 20)
	good := testCRC32CBase64(body)
	corrupt := append([]byte{}, body...)
	corrupt[4321] ^= 0xFF

	cases := []struct {
		name    string
		served  []byte
		sum     string
		mode    string
		wantErr string
	}{
		{name: "match", served: body, sum: good, mode: "FULL_OBJECT"},
		{name: "mismatch", served: corrupt, sum: good, mode: "FULL_OBJECT", wantErr: "checksum mismatch"},
		{name: "no checksum headers", served: corrupt, sum: "", mode: ""},
		{name: "composite mode not verified", served: corrupt, sum: good, mode: "COMPOSITE"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := checksumStubClient(t, &checksumStub{body: tc.served, checksum: tc.sum, mode: tc.mode})
			obj, err := c.GetObject(context.Background(), "bkt", "obj", GetObjectOptions{Checksum: true})
			if err != nil {
				t.Fatalf("GetObject: %v", err)
			}
			defer obj.Close()
			_, err = io.ReadAll(obj)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("ReadAll: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("ReadAll error = %v, want %q", err, tc.wantErr)
			}
		})
	}
}

func TestFGetObjectChecksumResume(t *testing.T) {
	body := testChecksumBody(1 << 20)
	good := testCRC32CBase64(body)
	prefix := body[:512<<10]
	corruptPrefix := make([]byte, len(prefix))
	for i := range corruptPrefix {
		corruptPrefix[i] = ^prefix[i]
	}

	cases := []struct {
		name    string
		prefix  []byte
		wantErr string
	}{
		{name: "full download", prefix: nil},
		{name: "resume correct prefix", prefix: prefix},
		{name: "resume corrupted prefix", prefix: corruptPrefix, wantErr: "checksum mismatch"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := checksumStubClient(t, &checksumStub{body: body, checksum: good, mode: "FULL_OBJECT"})
			dir := t.TempDir()
			dst := filepath.Join(dir, "out.bin")
			if tc.prefix != nil {
				partPath := filepath.Join(dir, sum256Hex([]byte("out.bin"+"checksum-stub-etag"))+".part.minio")
				if err := os.WriteFile(partPath, tc.prefix, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			err := c.FGetObject(context.Background(), "bkt", "obj", dst, GetObjectOptions{Checksum: true})
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("FGetObject error = %v, want %q", err, tc.wantErr)
				}
				if _, statErr := os.Stat(dst); !os.IsNotExist(statErr) {
					t.Fatal("destination committed despite checksum mismatch")
				}
				return
			}
			if err != nil {
				t.Fatalf("FGetObject: %v", err)
			}
			got, err := os.ReadFile(dst)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, body) {
				t.Fatalf("downloaded content mismatch: %d bytes", len(got))
			}
		})
	}
}