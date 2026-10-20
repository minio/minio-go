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
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestPutObjectMultipartStreamParallelShortRead(t *testing.T) {
	const partSize = 5 << 20

	var mu sync.Mutex
	var maxPart int
	var puts, aborts, completed atomic.Int32
	clnt := newTestStatClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Query().Has("uploads"):
			w.Write([]byte(`<InitiateMultipartUploadResult><UploadId>id</UploadId></InitiateMultipartUploadResult>`))
		case r.Method == http.MethodPut:
			n, _ := strconv.Atoi(r.URL.Query().Get("partNumber"))
			puts.Add(1)
			mu.Lock()
			maxPart = max(maxPart, n)
			mu.Unlock()
			io.Copy(io.Discard, r.Body)
			w.Header().Set("ETag", `"etag"`)
		case r.Method == http.MethodPost:
			body, _ := io.ReadAll(r.Body)
			completed.Store(int32(strings.Count(string(body), "<Part>")))
			w.Write([]byte(`<CompleteMultipartUploadResult><Bucket>bucket</Bucket></CompleteMultipartUploadResult>`))
		case r.Method == http.MethodDelete:
			aborts.Add(1)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNoContent)
		}
	})

	for _, tc := range []struct {
		name    string
		size    int64 // declared size, -1 for unknown
		avail   int   // bytes the reader holds
		readErr error // returned in place of io.EOF
		parts   int32 // parts in the completed object, 0 when the upload must fail
	}{
		{"exact", 12 << 20, 12 << 20, nil, 3},
		{"exact on part boundary", 10 << 20, 10 << 20, nil, 2},
		{"short in middle part", 12 << 20, 7 << 20, nil, 0},
		{"short in last part", 12 << 20, 11 << 20, nil, 0},
		{"one byte short", 12 << 20, 12<<20 - 1, nil, 0},
		{"ends on part boundary", 12 << 20, 10 << 20, nil, 0},
		{"empty reader", 12 << 20, 0, nil, 0},
		{"one byte longer", 12 << 20, 12<<20 + 1, nil, 0},
		{"longer past the last part", 12 << 20, 17 << 20, nil, 0},
		{"longer on part boundary", 10 << 20, 10<<20 + 1, nil, 0},
		{"read error at the end", 12 << 20, 12 << 20, errReadFailed, 0},
		{"unknown size", -1, 7 << 20, nil, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// ConcurrentStreamParts sends any reader, with or without ReadAt, to
			// the parallel uploader.
			var r io.Reader = bytes.NewReader(make([]byte, tc.avail))
			if tc.readErr != nil {
				r = &failAtEOFReader{Reader: r, err: tc.readErr}
			}
			puts.Store(0)
			mu.Lock()
			maxPart = 0
			mu.Unlock()
			aborts.Store(0)
			completed.Store(0)
			info, err := clnt.PutObject(t.Context(), "bucket", "object", r, tc.size, PutObjectOptions{
				PartSize:              partSize,
				ConcurrentStreamParts: true,
				NumThreads:            3,
			})
			if tc.parts > 0 {
				if err != nil {
					t.Fatal(err)
				}
				if info.Size != int64(tc.avail) {
					t.Errorf("Size = %d, want %d", info.Size, tc.avail)
				}
				if n := puts.Load(); n != tc.parts {
					t.Errorf("PUT requests = %d, want %d", n, tc.parts)
				}
				if n := completed.Load(); n != tc.parts {
					t.Errorf("parts completed = %d, want %d", n, tc.parts)
				}
				if n := aborts.Load(); n != 0 {
					t.Errorf("abort requests = %d, want 0", n)
				}
				return
			}
			if tc.readErr != nil {
				if !errors.Is(err, tc.readErr) {
					t.Fatalf("error = %v, want %v", err, tc.readErr)
				}
			} else if code := ToErrorResponse(err).Code; code != UnexpectedEOF {
				t.Fatalf("error code = %q, want %q (err: %v)", code, UnexpectedEOF, err)
			}
			// In-flight parts may be canceled, but a part the reader could not
			// fill must never be sent, nor the last part of a reader that holds
			// more than the declared size or fails after it.
			mu.Lock()
			got := maxPart
			mu.Unlock()
			if full := min(tc.avail/partSize, int((tc.size-1)/partSize)); got > full {
				t.Errorf("highest part sent = %d, want at most %d", got, full)
			}
			if n := completed.Load(); n != 0 {
				t.Errorf("parts completed = %d, want 0", n)
			}
			if n := aborts.Load(); n != 1 {
				t.Errorf("abort requests = %d, want 1", n)
			}
		})
	}
}

// A reader that ends exactly where the upload limits stop the part layout is
// one byte short of a declared size the layout cannot hold. Every part is
// full, so only a layout computed from the declared size catches it.
func TestPutObjectMultipartStreamParallelSizeBeyondLayout(t *testing.T) {
	const partSize = 5 << 20

	var requests, completes atomic.Int32
	clnt := newTestStatClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		switch {
		case r.URL.Query().Has("uploads"):
			w.Write([]byte(`<InitiateMultipartUploadResult><UploadId>id</UploadId></InitiateMultipartUploadResult>`))
		case r.Method == http.MethodPut:
			io.Copy(io.Discard, r.Body)
			w.Header().Set("ETag", `"etag"`)
		case r.Method == http.MethodPost:
			completes.Add(1)
			w.Write([]byte(`<CompleteMultipartUploadResult><Bucket>bucket</Bucket></CompleteMultipartUploadResult>`))
		default:
			w.WriteHeader(http.StatusNoContent)
		}
	})
	clnt.limits = UploadLimits{MaxPartsCount: 3}

	r := bytes.NewReader(make([]byte, 3*partSize))
	_, err := clnt.PutObject(t.Context(), "bucket", "object", r, 3*partSize+1, PutObjectOptions{
		PartSize:              partSize,
		ConcurrentStreamParts: true,
		NumThreads:            3,
	})
	if code := ToErrorResponse(err).Code; code != InvalidArgument {
		t.Errorf("error code = %q, want %q (err: %v)", code, InvalidArgument, err)
	}
	if n := completes.Load(); n != 0 {
		t.Errorf("complete requests = %d, want 0", n)
	}
	if n := requests.Load(); n != 0 {
		t.Errorf("requests = %d, want 0: the layout must be rejected before the upload starts", n)
	}
}
