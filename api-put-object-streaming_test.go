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
	"bytes"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
)

// eofReaderAt returns io.EOF together with the last byte, which io.ReaderAt allows.
type eofReaderAt struct{ *bytes.Reader }

func (r eofReaderAt) ReadAt(p []byte, off int64) (int, error) {
	n, err := r.Reader.ReadAt(p, off)
	if err == nil && off+int64(n) == r.Size() {
		err = io.EOF
	}
	return n, err
}

func TestPutObjectMultipartStreamFromReadAtShortRead(t *testing.T) {
	var requests atomic.Int32
	clnt := newTestStatClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		switch {
		case r.URL.Query().Has("uploads"):
			w.Write([]byte(`<InitiateMultipartUploadResult><UploadId>id</UploadId></InitiateMultipartUploadResult>`))
		case r.Method == http.MethodPut:
			io.Copy(io.Discard, r.Body)
			w.Header().Set("ETag", `"etag"`)
		case r.Method == http.MethodPost:
			w.Write([]byte(`<CompleteMultipartUploadResult><Bucket>bucket</Bucket></CompleteMultipartUploadResult>`))
		default:
			w.WriteHeader(http.StatusNoContent)
		}
	})

	const size = 12 << 20 // 5, 5 and 2 MiB parts
	for _, tc := range []struct {
		name  string
		avail int
		ok    bool
	}{
		{"exact", size, true},
		{"one byte short", size - 1, false},
		{"short in middle part", 7 << 20, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requests.Store(0)
			// eofReaderAt implements io.ReaderAt, so PutObject reads the parts with ReadAt.
			r := eofReaderAt{bytes.NewReader(make([]byte, tc.avail))}
			_, err := clnt.PutObject(t.Context(), "bucket", "object", r, size, PutObjectOptions{PartSize: 5 << 20})
			if tc.ok {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if code := ToErrorResponse(err).Code; code != UnexpectedEOF {
				t.Fatalf("error code = %q, want %q (err: %v)", code, UnexpectedEOF, err)
			}
			if n := requests.Load(); n != 0 {
				t.Fatalf("%d requests sent, want 0", n)
			}
		})
	}
}
