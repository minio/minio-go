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
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/encrypt"
)

func TestPutObjectOptionsValidate(t *testing.T) {
	testCases := []struct {
		name, value string
		shouldPass  bool
	}{
		// Invalid cases.
		{"X-Amz-Matdesc", "blah", false},
		{"x-amz-meta-X-Amz-Iv", "blah", false},
		{"x-amz-meta-X-Amz-Key", "blah", false},
		{"x-amz-meta-X-Amz-Matdesc", "blah", false},
		{"It has spaces", "v", false},
		{"It,has@illegal=characters", "v", false},
		{"X-Amz-Iv", "blah", false},
		{"X-Amz-Key", "blah", false},
		{"X-Amz-Key-prefixed-header", "blah", false},
		{"Content-Type", "custom/content-type", false},
		{"content-type", "custom/content-type", false},
		{"Content-Encoding", "gzip", false},
		{"Cache-Control", "blah", false},
		{"Content-Disposition", "something", false},
		{"Content-Language", "somelanguage", false},

		// Valid metadata names.
		{"my-custom-header", "blah", true},
		{"custom-X-Amz-Key-middle", "blah", true},
		{"my-custom-header-X-Amz-Key", "blah", true},
		{"blah-X-Amz-Matdesc", "blah", true},
		{"X-Amz-MatDesc-suffix", "blah", true},
		{"It-Is-Fine", "v", true},
		{"Numbers-098987987-Should-Work", "v", true},
		{"Crazy-!#$%&'*+-.^_`|~-Should-193832-Be-Fine", "v", true},
	}
	for i, testCase := range testCases {
		err := PutObjectOptions{UserMetadata: map[string]string{
			testCase.name: testCase.value,
		}}.validate(nil)
		if testCase.shouldPass && err != nil {
			t.Errorf("Test %d - output did not match with reference results, %s", i+1, err)
		}
	}
}

type InterceptRouteTripper struct {
	request *http.Request
}

func (i *InterceptRouteTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	i.request = request
	return &http.Response{StatusCode: 200}, nil
}

func Test_SSEHeaders(t *testing.T) {
	rt := &InterceptRouteTripper{}
	c, err := New("s3.amazonaws.com", &Options{
		Transport: rt,
	})
	if err != nil {
		t.Error(err)
	}

	testCases := map[string]struct {
		sse                            func() encrypt.ServerSide
		initiateMultipartUploadHeaders http.Header
		headerNotAllowedAfterInit      []string
	}{
		"noEncryption": {
			sse:                            func() encrypt.ServerSide { return nil },
			initiateMultipartUploadHeaders: http.Header{},
		},
		"sse": {
			sse: func() encrypt.ServerSide {
				s, err := encrypt.NewSSEKMS("keyId", nil)
				if err != nil {
					t.Error(err)
				}
				return s
			},
			initiateMultipartUploadHeaders: http.Header{
				encrypt.SseGenericHeader: []string{"aws:kms"},
				encrypt.SseKmsKeyID:      []string{"keyId"},
			},
			headerNotAllowedAfterInit: []string{encrypt.SseGenericHeader, encrypt.SseKmsKeyID, encrypt.SseEncryptionContext},
		},
		"sse with context": {
			sse: func() encrypt.ServerSide {
				s, err := encrypt.NewSSEKMS("keyId", "context")
				if err != nil {
					t.Error(err)
				}
				return s
			},
			initiateMultipartUploadHeaders: http.Header{
				encrypt.SseGenericHeader:     []string{"aws:kms"},
				encrypt.SseKmsKeyID:          []string{"keyId"},
				encrypt.SseEncryptionContext: []string{base64.StdEncoding.EncodeToString([]byte("\"context\""))},
			},
			headerNotAllowedAfterInit: []string{encrypt.SseGenericHeader, encrypt.SseKmsKeyID, encrypt.SseEncryptionContext},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			opts := PutObjectOptions{
				ServerSideEncryption: tc.sse(),
			}
			c.bucketLocCache.Set("test", "region")
			c.initiateMultipartUpload(context.Background(), "test", "test", opts)
			for s, vls := range tc.initiateMultipartUploadHeaders {
				if !reflect.DeepEqual(rt.request.Header[s], vls) {
					t.Errorf("Header %v are not equal, want: %v got %v", s, vls, rt.request.Header[s])
				}
			}

			_, err := c.uploadPart(context.Background(), uploadPartParams{
				bucketName: "test",
				objectName: "test",
				partNumber: 1,
				uploadID:   "upId",
				sse:        opts.ServerSideEncryption,
			})
			if err != nil {
				t.Error(err)
			}

			for _, k := range tc.headerNotAllowedAfterInit {
				if rt.request.Header.Get(k) != "" {
					t.Errorf("header %v should not be set", k)
				}
			}

			c.completeMultipartUpload(context.Background(), "test", "test", "upId", completeMultipartUpload{}, opts)

			for _, k := range tc.headerNotAllowedAfterInit {
				if rt.request.Header.Get(k) != "" {
					t.Errorf("header %v should not be set", k)
				}
			}
		})
	}
}

// TestCompleteUploadOpts checks that the options sent on
// CompleteMultipartUpload keep opts.Internal. MinIO reads
// x-minio-source-mtime and x-minio-source-replication-request from the
// complete request and from nowhere else, so dropping Internal silently
// discards a preserved modification time on every multipart upload.
func TestCompleteUploadOpts(t *testing.T) {
	mtime := time.Date(2026, 8, 21, 15, 42, 50, 543000000, time.UTC)
	sse, err := encrypt.NewSSEC(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}

	opts := PutObjectOptions{
		ContentType:          "text/plain",
		StorageClass:         "REDUCED_REDUNDANCY",
		UserMetadata:         map[string]string{"Key": "value"},
		ServerSideEncryption: sse,
		AutoChecksum:         ChecksumCRC32C,
	}
	opts.Internal.SourceMTime = mtime
	opts.Internal.SourceETag = "0d2e0d0e0f0a0b0c"
	opts.Internal.ReplicationRequest = true

	got := completeUploadOpts(opts)

	if !got.Internal.SourceMTime.Equal(mtime) {
		t.Errorf("SourceMTime = %v, want %v", got.Internal.SourceMTime, mtime)
	}
	if got.Internal.SourceETag != opts.Internal.SourceETag {
		t.Errorf("SourceETag = %q, want %q", got.Internal.SourceETag, opts.Internal.SourceETag)
	}
	if !got.Internal.ReplicationRequest {
		t.Error("ReplicationRequest = false, want true")
	}
	if got.ServerSideEncryption != sse {
		t.Error("ServerSideEncryption was dropped")
	}
	if got.AutoChecksum != opts.AutoChecksum {
		t.Errorf("AutoChecksum = %v, want %v", got.AutoChecksum, opts.AutoChecksum)
	}

	// Options that already applied at NewMultipartUpload must not be repeated.
	if got.ContentType != "" || got.StorageClass != "" || got.UserMetadata != nil {
		t.Errorf("upload-wide options leaked into the complete request: %+v", got)
	}

	header := got.Header()
	if v := header.Get(minIOBucketSourceMTime); v != mtime.Format(time.RFC3339Nano) {
		t.Errorf("%s header = %q, want %q", minIOBucketSourceMTime, v, mtime.Format(time.RFC3339Nano))
	}
	if v := header.Get(minIOBucketReplicationRequest); v != "true" {
		t.Errorf("%s header = %q, want %q", minIOBucketReplicationRequest, v, "true")
	}
	if v := header.Get(amzStorageClass); v != "" {
		t.Errorf("%s header = %q, want it absent", amzStorageClass, v)
	}
}

// errReadFailed is the error a reader returns in place of io.EOF.
var errReadFailed = errors.New("read failed")

func TestPutObjectMultipartStreamShortRead(t *testing.T) {
	var puts, completes, aborts atomic.Int32
	clnt := newTestStatClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Query().Has("uploads"):
			w.Write([]byte(`<InitiateMultipartUploadResult><UploadId>id</UploadId></InitiateMultipartUploadResult>`))
		case r.Method == http.MethodPut:
			puts.Add(1)
			io.Copy(io.Discard, r.Body)
			w.Header().Set("ETag", `"etag"`)
		case r.Method == http.MethodPost:
			completes.Add(1)
			w.Write([]byte(`<CompleteMultipartUploadResult><Bucket>bucket</Bucket></CompleteMultipartUploadResult>`))
		case r.Method == http.MethodDelete:
			aborts.Add(1)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNoContent)
		}
	})

	const size = 12 << 20 // 5, 5 and 2 MiB parts
	for _, tc := range []struct {
		name    string
		size    int64 // declared size, size when zero
		avail   int
		readErr error // returned in place of io.EOF
		ok      bool
		puts    int32 // full parts sent
	}{
		{"exact", 0, size, nil, true, 3},
		{"short in middle part", 0, 7 << 20, nil, false, 1},
		{"short in last part", 0, 11 << 20, nil, false, 2},
		{"one byte short", 0, size - 1, nil, false, 2},
		{"ends on part boundary", 0, 10 << 20, nil, false, 2},
		{"empty reader", 0, 0, nil, false, 0},
		{"one byte longer", 0, size + 1, nil, false, 2},
		{"5 MiB longer", 0, size + 5<<20, nil, false, 2},
		{"longer on part boundary", 10 << 20, 10<<20 + 1, nil, false, 1},
		{"read error at the end", 0, size, errReadFailed, false, 2},
		{"read error at the end on part boundary", 10 << 20, 10 << 20, errReadFailed, false, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			declared := tc.size
			if declared == 0 {
				declared = size
			}
			// Hide ReadAt so PutObject streams the parts sequentially.
			var r io.Reader = bytes.NewReader(make([]byte, tc.avail))
			if tc.readErr != nil {
				r = &failAtEOFReader{Reader: r, err: tc.readErr}
			}
			r = struct{ io.Reader }{r}
			puts.Store(0)
			completes.Store(0)
			aborts.Store(0)
			_, err := clnt.PutObject(t.Context(), "bucket", "object", r, declared, PutObjectOptions{PartSize: 5 << 20})
			if n := puts.Load(); n != tc.puts {
				t.Fatalf("PUT requests = %d, want %d", n, tc.puts)
			}
			if tc.ok {
				if err != nil {
					t.Fatal(err)
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
			if n := completes.Load(); n != 0 {
				t.Errorf("complete requests = %d, want 0", n)
			}
			if n := aborts.Load(); n != 1 {
				t.Errorf("abort requests = %d, want 1", n)
			}
		})
	}
}

// A reader that holds more than the declared size must fail with
// errUnexpectedEOF, not with net/http's content length mismatch, and the
// server must never receive a complete body.
func TestPutObjectSingleLongerReader(t *testing.T) {
	// Below the multipart threshold, so PutObject sends a single PUT.
	const size = 1 << 20
	for _, tc := range []struct {
		name    string
		opts    PutObjectOptions
		readErr error // when set, the reader holds size bytes and then returns it in place of io.EOF
		started int32 // requests the server sees
	}{
		{"plain reader", PutObjectOptions{}, nil, 1},
		{"plain reader with Content-MD5", PutObjectOptions{SendContentMd5: true}, nil, 0},
		{"read error at the end", PutObjectOptions{}, errReadFailed, 1},
		{"read error at the end with Content-MD5", PutObjectOptions{SendContentMd5: true}, errReadFailed, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var started, complete atomic.Int32
			// TLS, so the body is sent without the streaming signature: there, a
			// failed read also drops the closing chunk, and the handler would see a
			// cut body even when every object byte was handed out.
			srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				started.Add(1)
				if _, err := io.Copy(io.Discard, r.Body); err == nil {
					complete.Add(1)
				}
				w.Header().Set("ETag", `"etag"`)
			}))
			clnt, err := New(srv.Listener.Addr().String(), &Options{
				Creds:     credentials.NewStaticV4("foo", "foo12345", ""),
				Region:    "us-east-1",
				Secure:    true,
				Transport: srv.Client().Transport,
			})
			if err != nil {
				srv.Close()
				t.Fatal(err)
			}

			// Hide ReadAt and Seek so PutObject sends the reader as is.
			var src io.Reader = bytes.NewReader(make([]byte, size+1))
			if tc.readErr != nil {
				src = &failAtEOFReader{Reader: bytes.NewReader(make([]byte, size)), err: tc.readErr}
			}
			r := struct{ io.Reader }{src}
			_, err = clnt.PutObject(t.Context(), "bucket", "object", r, size, tc.opts)

			// Close waits for the handlers still reading the aborted body.
			srv.Close()
			if tc.readErr != nil {
				if !errors.Is(err, tc.readErr) {
					t.Fatalf("error = %v, want %v", err, tc.readErr)
				}
			} else if code := ToErrorResponse(err).Code; code != UnexpectedEOF {
				t.Fatalf("error code = %q, want %q (err: %v)", code, UnexpectedEOF, err)
			}
			if n := complete.Load(); n != 0 {
				t.Errorf("requests with a complete body = %d, want 0", n)
			}
			if n := started.Load(); n > tc.started {
				t.Errorf("requests = %d, want at most %d", n, tc.started)
			}
		})
	}
}
