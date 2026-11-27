/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2017 MinIO, Inc.
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
	"encoding/xml"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	gb1    = 1024 * 1024 * 1024
	gb5    = 5 * gb1
	gb5p1  = gb5 + 1
	gb10p1 = 2*gb5 + 1
	gb10p2 = 2*gb5 + 2

	// oldPartSize is the legacy part size calculation for backward compatibility testing
	// It was: maxMultipartPutObjectSize / (maxPartsCount - 1)
	oldPartSize = maxMultipartPutObjectSize / (defaultMaxPartsCount - 1)
)

func TestPartsRequired(t *testing.T) {
	testCases := []struct {
		size     int64
		partSize int64
		ref      int64
	}{
		{0, 0, 0},
		{1, 0, 1},
		{gb5, 0, 1},                      // 5 GiB / 5 GiB = 1 part
		{gb5p1, 0, 2},                    // 5 GiB + 1 byte needs 2 parts
		{2 * gb5, 0, 2},                  // 10 GiB / 5 GiB = 2 parts
		{gb10p1, 0, 3},                   // 10 GiB + 1 byte needs 3 parts
		{gb10p2, 0, 3},                   // 10 GiB + 2 bytes needs 3 parts
		{gb10p1 + gb10p2, 0, 5},          // 20 GiB + 3 bytes needs 5 parts
		{defaultMaxPartSize * 10, 0, 10}, // exactly 10 parts
		// Custom part sizes
		{gb5, gb1, 5},      // 5 GiB / 1 GiB = 5 parts
		{gb5p1, gb1, 6},    // 5 GiB + 1 byte / 1 GiB = 6 parts
		{2 * gb5, gb1, 10}, // 10 GiB / 1 GiB = 10 parts
		// Legacy behavior with oldPartSize
		{gb5, oldPartSize, 10},                          // matches old behavior
		{gb5p1, oldPartSize, 10},                        // matches old behavior
		{2 * gb5, oldPartSize, 20},                      // matches old behavior
		{gb10p1, oldPartSize, 20},                       // matches old behavior
		{gb10p2, oldPartSize, 20},                       // matches old behavior
		{gb10p1 + gb10p2, oldPartSize, 40},              // matches old behavior
		{maxMultipartPutObjectSize, oldPartSize, 10000}, // matches old behavior
	}

	for i, testCase := range testCases {
		res := partsRequired(testCase.size, testCase.partSize)
		if res != testCase.ref {
			t.Errorf("Test %d - output did not match with reference results, Expected %d, got %d", i+1, testCase.ref, res)
		}
	}
}

func TestCalculateEvenSplits(t *testing.T) {
	testCases := []struct {
		// input size, source object, and part size
		size     int64
		src      CopySrcOptions
		partSize int64

		// output part-indexes
		starts, ends []int64
	}{
		// Empty and minimal cases with default part size (0 = maxPartSize)
		{0, CopySrcOptions{Start: -1}, 0, nil, nil},
		{1, CopySrcOptions{Start: -1}, 0, []int64{0}, []int64{0}},
		{1, CopySrcOptions{Start: 0}, 0, []int64{0}, []int64{0}},

		// With default part size (5 GiB), these fit in 1 part
		{gb1, CopySrcOptions{Start: -1}, 0, []int64{0}, []int64{gb1 - 1}},
		{gb5, CopySrcOptions{Start: -1}, 0, []int64{0}, []int64{gb5 - 1}},

		// 5 GiB + 1 byte needs 2 parts with default part size
		{gb5p1, CopySrcOptions{Start: -1}, 0, []int64{0, 2684354561}, []int64{2684354560, gb5p1 - 1}},

		// 10 GiB + 1 byte needs 3 parts with default part size
		{gb10p1, CopySrcOptions{Start: -1}, 0, []int64{0, 3579139414, 7158278828}, []int64{3579139413, 7158278827, gb10p1 - 1}},

		// With custom 1 GiB part size, 5 GiB needs 5 parts
		{
			gb5,
			CopySrcOptions{Start: -1},
			gb1,
			[]int64{0, gb1, 2 * gb1, 3 * gb1, 4 * gb1},
			[]int64{gb1 - 1, 2*gb1 - 1, 3*gb1 - 1, 4*gb1 - 1, 5*gb1 - 1},
		},

		// With custom 1 GiB part size, 1 GiB needs 1 part
		{gb1, CopySrcOptions{Start: -1}, gb1, []int64{0}, []int64{gb1 - 1}},

		// With start offset
		{gb1, CopySrcOptions{Start: 100}, gb1, []int64{100}, []int64{gb1 + 99}},

		// Legacy behavior with oldPartSize - 5 GiB splits into 10 parts
		{
			gb5,
			CopySrcOptions{Start: -1},
			oldPartSize,
			[]int64{
				0, 536870912, 1073741824, 1610612736, 2147483648, 2684354560,
				3221225472, 3758096384, 4294967296, 4831838208,
			},
			[]int64{
				536870911, 1073741823, 1610612735, 2147483647, 2684354559, 3221225471,
				3758096383, 4294967295, 4831838207, 5368709119,
			},
		},

		// Legacy behavior with oldPartSize - 5 GiB + 1 splits into 10 parts
		{
			gb5p1,
			CopySrcOptions{Start: -1},
			oldPartSize,
			[]int64{
				0, 536870913, 1073741825, 1610612737, 2147483649, 2684354561,
				3221225473, 3758096385, 4294967297, 4831838209,
			},
			[]int64{
				536870912, 1073741824, 1610612736, 2147483648, 2684354560, 3221225472,
				3758096384, 4294967296, 4831838208, 5368709120,
			},
		},
	}

	for i, testCase := range testCases {
		resStart, resEnd := calculateEvenSplits(testCase.size, testCase.src, testCase.partSize)
		if !reflect.DeepEqual(testCase.starts, resStart) || !reflect.DeepEqual(testCase.ends, resEnd) {
			t.Errorf("Test %d - output did not match with reference results, Expected %v/%v, got %v/%v", i+1, testCase.starts, testCase.ends, resStart, resEnd)
		}
	}
}

func TestDestOptions(t *testing.T) {
	userMetadata := map[string]string{
		"test":                "test",
		"x-amz-acl":           "public-read-write",
		"content-type":        "application/binary",
		"X-Amz-Storage-Class": "rrs",
		"x-amz-grant-write":   "test@exo.ch",
	}

	r := make(http.Header)

	dst := CopyDestOptions{
		Bucket:          "bucket",
		Object:          "object",
		ReplaceMetadata: true,
		UserMetadata:    userMetadata,
	}
	dst.Marshal(r)

	if v := r.Get("x-amz-metadata-directive"); v != "REPLACE" {
		t.Errorf("Test - metadata directive was expected but is missing")
	}

	for k := range r {
		if strings.HasSuffix(k, "test") && !strings.HasPrefix(k, "x-amz-meta-") {
			t.Errorf("Test meta %q was expected as an x amz meta", k)
		}

		if !strings.HasSuffix(k, "test") && strings.HasPrefix(k, "x-amz-meta-") {
			t.Errorf("Test an amz/standard/storageClass Header was expected but got an x amz meta data")
		}
	}
}

type composeTestServer struct {
	srcSize  int64
	failPart int

	mu        sync.Mutex
	copied    int
	completed []int
	aborted   bool

	inflight, maxInflight atomic.Int32
}

func (s *composeTestServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	switch {
	case r.Method == http.MethodHead:
		w.Header().Set("Content-Length", strconv.FormatInt(s.srcSize, 10))
		w.Header().Set("ETag", `"src"`)
		w.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
	case r.Method == http.MethodPost && q.Has("uploads"):
		w.Write([]byte(`<InitiateMultipartUploadResult><UploadId>upload-1</UploadId></InitiateMultipartUploadResult>`))
	case r.Method == http.MethodPut && q.Has("partNumber"):
		n := s.inflight.Add(1)
		defer s.inflight.Add(-1)
		for {
			m := s.maxInflight.Load()
			if n <= m || s.maxInflight.CompareAndSwap(m, n) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		s.mu.Lock()
		s.copied++
		s.mu.Unlock()
		if q.Get("partNumber") == strconv.Itoa(s.failPart) {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`<Error><Code>AccessDenied</Code><Message>denied</Message></Error>`))
			return
		}
		w.Write([]byte(`<CopyPartResult><ETag>"part-` + q.Get("partNumber") + `"</ETag></CopyPartResult>`))
	case r.Method == http.MethodPost && q.Has("uploadId"):
		var body completeMultipartUpload
		if err := xml.NewDecoder(r.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		s.mu.Lock()
		for _, p := range body.Parts {
			s.completed = append(s.completed, p.PartNumber)
		}
		s.mu.Unlock()
		w.Write([]byte(`<CompleteMultipartUploadResult><Bucket>bucket</Bucket><Key>dst</Key><ETag>"dst"</ETag></CompleteMultipartUploadResult>`))
	case r.Method == http.MethodDelete && q.Has("uploadId"):
		s.mu.Lock()
		s.aborted = true
		s.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func composeTestObject(t *testing.T, s *composeTestServer, numThreads uint) error {
	t.Helper()
	srv := httptest.NewServer(s)
	defer srv.Close()

	clnt, err := New(srv.Listener.Addr().String(), &Options{Region: "us-east-1"})
	if err != nil {
		t.Fatal(err)
	}
	dst := CopyDestOptions{Bucket: "bucket", Object: "dst", PartSize: defaultMinPartSize, NumThreads: numThreads}
	_, err = clnt.ComposeObject(context.Background(), dst, CopySrcOptions{Bucket: "bucket", Object: "src"})
	return err
}

func TestComposeObjectParallel(t *testing.T) {
	const numParts = 16
	for _, tc := range []struct {
		numThreads  uint
		maxInflight int32
	}{
		{0, 1},
		{1, 1},
		{4, 4},
	} {
		s := &composeTestServer{srcSize: numParts * defaultMinPartSize}
		if err := composeTestObject(t, s, tc.numThreads); err != nil {
			t.Fatalf("NumThreads %d: %v", tc.numThreads, err)
		}
		if got := s.maxInflight.Load(); got != tc.maxInflight {
			t.Errorf("NumThreads %d: expected %d concurrent part copies, got %d", tc.numThreads, tc.maxInflight, got)
		}
		want := make([]int, numParts)
		for i := range want {
			want[i] = i + 1
		}
		if !reflect.DeepEqual(s.completed, want) {
			t.Errorf("NumThreads %d: expected parts %v, got %v", tc.numThreads, want, s.completed)
		}
	}
}

func TestComposeObjectParallelPartFailure(t *testing.T) {
	const numParts = 16
	s := &composeTestServer{srcSize: numParts * defaultMinPartSize, failPart: 3}
	err := composeTestObject(t, s, 4)
	var errResp ErrorResponse
	if !errors.As(err, &errResp) || errResp.Code != "AccessDenied" {
		t.Fatalf("expected AccessDenied, got %v", err)
	}
	if !s.aborted {
		t.Error("expected the multipart upload to be aborted")
	}
	if s.completed != nil {
		t.Errorf("expected no complete-multipart request, got parts %v", s.completed)
	}
	if s.copied >= numParts {
		t.Errorf("expected remaining parts to be canceled, but all %d were copied", s.copied)
	}
}

// TestDestOptionsKeepsUserMetaNamedLikeAHeader checks that a user metadata key
// whose own name is a header name survives a server-side copy.
//
// The key is classified as it was given. Taking the `x-amz-meta-` prefix off
// first made `x-amz-meta-cache-control` read as Cache-Control and go out as a
// real header, and `x-amz-meta-content-type` land on the Content-Type Marshal
// sets from its own field, where one silently took the other's place. Either
// way the object came back under a name the caller never asked for.
func TestDestOptionsKeepsUserMetaNamedLikeAHeader(t *testing.T) {
	header := make(http.Header)
	CopyDestOptions{
		Bucket:          "bucket",
		Object:          "object",
		ContentType:     "text/plain",
		ReplaceMetadata: true,
		UserMetadata: map[string]string{
			"x-amz-meta-cache-control": "no-store",
			"X-Amz-Meta-Content-Type":  "application/json",
			"x-amz-meta-expires":       "Thu, 01 Jan 2037 00:00:00 GMT",
			"x-amz-meta-owner":         "payments",
			"owner-team":               "payments",
			"x-amz-acl":                "public-read-write",
			"X-Amz-Storage-Class":      "rrs",
		},
	}.Marshal(header)

	for name, want := range map[string]string{
		// Kept as user metadata, prefix and all.
		"X-Amz-Meta-Cache-Control": "no-store",
		"X-Amz-Meta-Content-Type":  "application/json",
		"X-Amz-Meta-Expires":       "Thu, 01 Jan 2037 00:00:00 GMT",
		"X-Amz-Meta-Owner":         "payments",
		// A name without the prefix still gets one.
		"X-Amz-Meta-Owner-Team": "payments",
		// And a real header is still a real header.
		"Content-Type":        "text/plain",
		"X-Amz-Acl":           "public-read-write",
		"X-Amz-Storage-Class": "rrs",
	} {
		if got := header.Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}

	// The names a stripped key would have been sent under.
	for _, name := range []string{"Cache-Control", "Expires"} {
		if got := header.Get(name); got != "" {
			t.Errorf("user metadata went out as the header %s = %q", name, got)
		}
	}
}

// TestDestOptionsMatchesPutObject checks that a server-side copy sends a
// UserMetadata map under exactly the names PutObject would.
//
// The two classify by one shared rule now. They did not: the copy path took
// an `x-amz-meta-` prefix off first and then asked what the remainder was, so
// the same map reached the server one way on an upload and another on a copy.
func TestDestOptionsMatchesPutObject(t *testing.T) {
	userMetadata := map[string]string{
		"x-amz-meta-cache-control": "no-store",
		"X-Amz-Meta-Content-Type":  "application/json",
		"x-amz-meta-expires":       "Thu, 01 Jan 2037 00:00:00 GMT",
		"x-amz-meta-owner":         "payments",
		"owner-team":               "payments",
		"x-amz-acl":                "public-read-write",
		"X-Amz-Storage-Class":      "rrs",
	}

	// Named on both, so the comparison is about how UserMetadata keys are
	// classified and not about PutObject defaulting a Content-Type.
	const contentType = "text/plain"

	put := PutObjectOptions{ContentType: contentType, UserMetadata: userMetadata}.Header()

	copied := make(http.Header)
	CopyDestOptions{
		Bucket: "bucket", Object: "object", ContentType: contentType,
		ReplaceMetadata: true, UserMetadata: userMetadata,
	}.Marshal(copied)
	// Set by the copy alone, so not a difference in how names are classified.
	copied.Del("x-amz-metadata-directive")

	for name, want := range put {
		if got := copied[name]; !slices.Equal(got, want) {
			t.Errorf("copy sends %s = %q, PutObject sends %q", name, got, want)
		}
	}
	for name, got := range copied {
		if _, sent := put[name]; !sent {
			t.Errorf("copy sends %s = %q, which PutObject does not send at all", name, got)
		}
	}

	// And the names a stripped key used to be sent under.
	for _, name := range []string{"Cache-Control", "Expires"} {
		if got := copied.Get(name); got != "" {
			t.Errorf("user metadata went out as the header %s = %q", name, got)
		}
	}
}

// TestDestOptionsValidatesLikePutObject checks that a server-side copy refuses
// the same UserMetadata names PutObject refuses.
//
// Both send the map as headers, so both have the same names to reject: one
// that is not a valid header name at all, and one these options already carry
// in a field of their own. The copy path checked none of it, so a malformed
// name failed inside net/http with no mention of the key, and a name the
// server reserves failed only once the server saw it.
func TestDestOptionsValidatesLikePutObject(t *testing.T) {
	for _, test := range []struct {
		name    string
		meta    map[string]string
		wantErr bool
	}{
		{name: "a plain user name", meta: map[string]string{"owner": "payments"}},
		{name: "a prefixed user name", meta: map[string]string{"x-amz-meta-owner": "payments"}},
		{name: "an amz header", meta: map[string]string{"x-amz-acl": "public-read"}},
		{name: "a standard header", meta: map[string]string{"Cache-Control": "no-store"}, wantErr: true},
		{name: "an sse header", meta: map[string]string{"X-Amz-Server-Side-Encryption": "AES256"}, wantErr: true},
		{name: "a storage class header", meta: map[string]string{"X-Amz-Storage-Class": "rrs"}, wantErr: true},
		{name: "a minio header", meta: map[string]string{"X-Minio-Internal-Thing": "on"}, wantErr: true},
		{name: "a name that is no header name", meta: map[string]string{"bad name": "v"}, wantErr: true},
		{name: "a value no header can carry", meta: map[string]string{"owner": "line\nbreak"}, wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			copied := CopyDestOptions{
				Bucket: "bucket", Object: "object",
				ReplaceMetadata: true, UserMetadata: test.meta,
			}.validate()
			put := PutObjectOptions{UserMetadata: test.meta}.validate(nil)

			if (copied != nil) != test.wantErr {
				t.Errorf("copy validate() = %v, wantErr %v", copied, test.wantErr)
			}
			// Whatever the verdict, it has to be the same verdict.
			if (copied != nil) != (put != nil) {
				t.Errorf("copy validate() = %v, PutObject validate() = %v", copied, put)
			}
			if copied != nil && put != nil && copied.Error() != put.Error() {
				t.Errorf("copy says %q, PutObject says %q", copied, put)
			}

			// Without ReplaceMetadata the map is not sent at all, and the
			// copy preserves the source's metadata. Refusing it would fail a
			// request that has always been fine.
			ignored := CopyDestOptions{
				Bucket: "bucket", Object: "object", UserMetadata: test.meta,
			}.validate()
			if ignored != nil {
				t.Errorf("validate() = %v for a map ReplaceMetadata leaves unsent", ignored)
			}
		})
	}
}
