/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2015-2025 MinIO, Inc.
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
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/minio/minio-go/v7/pkg/credentials"
)

// A zero UploadLimits must reproduce Amazon S3's limits.
func TestUploadLimitsDefaults(t *testing.T) {
	var l UploadLimits
	for _, tc := range []struct {
		name string
		got  int64
		want int64
	}{
		{"minPartSize", l.minPartSize(), 5 * 1024 * 1024},
		{"maxPartSize", l.maxPartSize(), 5 * 1024 * 1024 * 1024},
		{"maxPartsCount", l.maxPartsCount(), 10000},
		{"maxSinglePutObjectSize", l.maxSinglePutObjectSize(), 5 * 1024 * 1024 * 1024},
		{"maxObjectSize", l.maxObjectSize(), 5 * 1024 * 1024 * 1024 * 10000},
	} {
		if tc.got != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, tc.got, tc.want)
		}
	}
}

// Setting one field must not disturb the others.
func TestUploadLimitsPartialOverride(t *testing.T) {
	l := UploadLimits{MaxPartsCount: 100000}
	if got := l.maxPartsCount(); got != 100000 {
		t.Errorf("maxPartsCount: got %d, want 100000", got)
	}
	if got := l.maxPartSize(); got != defaultMaxPartSize {
		t.Errorf("maxPartSize: got %d, want %d", got, int64(defaultMaxPartSize))
	}
	if got := l.minPartSize(); got != defaultMinPartSize {
		t.Errorf("minPartSize: got %d, want %d", got, int64(defaultMinPartSize))
	}
	if got, want := l.maxObjectSize(), int64(defaultMaxPartSize)*100000; got != want {
		t.Errorf("maxObjectSize: got %d, want %d", got, want)
	}
}

// A Client built outside privateNew (as several tests in this package do) must
// keep behaving like the defaults rather than dividing by a zero limit.
func TestUploadLimitsZeroValueClient(t *testing.T) {
	c := &Client{}
	for _, size := range []int64{-1, 1, 5243928576, defaultMaxPartSize * 10} {
		wantParts, wantPart, wantLast, wantErr := OptimalPartInfo(size, 0)
		gotParts, gotPart, gotLast, gotErr := c.optimalPartInfo(size, 0)
		if gotParts != wantParts || gotPart != wantPart || gotLast != wantLast || (gotErr == nil) != (wantErr == nil) {
			t.Errorf("size %d: got (%d, %d, %d, %v), want (%d, %d, %d, %v)",
				size, gotParts, gotPart, gotLast, gotErr, wantParts, wantPart, wantLast, wantErr)
		}
	}
	if got := c.limits.maxObjectSize(); got != int64(defaultMaxPartSize)*defaultMaxPartsCount {
		t.Errorf("maxObjectSize: got %d, want %d", got, int64(defaultMaxPartSize)*defaultMaxPartsCount)
	}
}

func TestUploadLimitsValidate(t *testing.T) {
	testCases := []struct {
		name    string
		limits  UploadLimits
		wantErr bool
	}{
		{"zero value", UploadLimits{}, false},
		{"raised parts count only", UploadLimits{MaxPartsCount: 100000}, false},
		{"raised part size only", UploadLimits{MaxPartSize: 64 * 1024 * 1024 * 1024}, false},
		{"all raised", UploadLimits{
			MinPartSize:            1024,
			MaxPartSize:            64 * 1024 * 1024 * 1024,
			MaxPartsCount:          100000,
			MaxSinglePutObjectSize: 64 * 1024 * 1024 * 1024,
		}, false},
		{"negative MinPartSize", UploadLimits{MinPartSize: -1}, true},
		{"negative MaxPartSize", UploadLimits{MaxPartSize: -1}, true},
		{"negative MaxPartsCount", UploadLimits{MaxPartsCount: -1}, true},
		{"negative MaxSinglePutObjectSize", UploadLimits{MaxSinglePutObjectSize: -1}, true},
		// MaxPartSize below the default 5MiB floor.
		{"max part size under default min", UploadLimits{MaxPartSize: 1024}, true},
		{"min above max", UploadLimits{MinPartSize: 1024 * 1024 * 1024, MaxPartSize: 1024 * 1024}, true},
		// maxObjectSize() would wrap negative.
		{"max object size overflows", UploadLimits{MaxPartSize: math.MaxInt64 / 2, MaxPartsCount: 3}, true},
		{"max object size at the int64 ceiling", UploadLimits{MaxPartSize: math.MaxInt64 / 10000, MaxPartsCount: 10000}, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.limits.validate(); (err != nil) != tc.wantErr {
				t.Fatalf("validate() error = %v, wantErr %v", err, tc.wantErr)
			}

			limits := tc.limits
			c, err := New("play.min.io", &Options{
				Creds:        credentials.NewStaticV4("id", "secret", ""),
				UploadLimits: &limits,
			})
			if (err != nil) != tc.wantErr {
				t.Fatalf("New() error = %v, wantErr %v", err, tc.wantErr)
			}
			if err == nil && c.limits != tc.limits {
				t.Fatalf("client limits = %+v, want %+v", c.limits, tc.limits)
			}
		})
	}
}

// nil Options.UploadLimits leaves the client on the defaults.
func TestUploadLimitsUnset(t *testing.T) {
	c, err := New("play.min.io", &Options{Creds: credentials.NewStaticV4("id", "secret", "")})
	if err != nil {
		t.Fatal(err)
	}
	if (c.limits != UploadLimits{}) {
		t.Fatalf("client limits = %+v, want zero value", c.limits)
	}
}

// Raising MaxPartsCount must allow layouts with more than 10000 parts.
func TestOptimalPartInfoRaisedPartsCount(t *testing.T) {
	const size = 20 * 1024 * 1024 * 1024 * 1024 // 20TiB
	const partSize = 128 * 1024 * 1024          // 128MiB -> 163840 parts

	if _, _, _, err := OptimalPartInfo(size, partSize); err == nil {
		t.Fatal("default limits should reject a layout needing more than 10000 parts")
	}

	l := UploadLimits{MaxPartsCount: 1000000}
	totalParts, gotPartSize, lastPartSize, err := l.optimalPartInfo(size, partSize)
	if err != nil {
		t.Fatal(err)
	}
	if totalParts != size/partSize {
		t.Errorf("totalParts: got %d, want %d", totalParts, size/partSize)
	}
	if gotPartSize != partSize {
		t.Errorf("partSize: got %d, want %d", gotPartSize, int64(partSize))
	}
	if lastPartSize != partSize {
		t.Errorf("lastPartSize: got %d, want %d", lastPartSize, int64(partSize))
	}
}

// Raising MaxPartSize must allow a configured part size above 5GiB.
func TestOptimalPartInfoRaisedPartSize(t *testing.T) {
	const partSize = 10 * 1024 * 1024 * 1024 // 10GiB
	const size = partSize * 4

	if _, _, _, err := OptimalPartInfo(size, partSize); err == nil {
		t.Fatal("default limits should reject a part size above 5GiB")
	}

	l := UploadLimits{MaxPartSize: 64 * 1024 * 1024 * 1024}
	totalParts, gotPartSize, _, err := l.optimalPartInfo(size, partSize)
	if err != nil {
		t.Fatal(err)
	}
	if totalParts != 4 || gotPartSize != partSize {
		t.Errorf("got (%d parts, %d part size), want (4, %d)", totalParts, gotPartSize, int64(partSize))
	}
}

// Lowering limits must tighten what the client accepts.
func TestOptimalPartInfoLoweredLimits(t *testing.T) {
	l := UploadLimits{MaxPartsCount: 20}

	// 20 parts of 5GiB is all this allows.
	if got, want := l.maxObjectSize(), int64(defaultMaxPartSize)*20; got != want {
		t.Fatalf("maxObjectSize: got %d, want %d", got, want)
	}
	if _, _, _, err := l.optimalPartInfo(l.maxObjectSize()+1, 0); err == nil {
		t.Error("expected an error for an object above the lowered max object size")
	}
	// A part size that would need more than 20 parts.
	if _, _, _, err := l.optimalPartInfo(21*minPartSize, minPartSize); err == nil {
		t.Error("expected an error for a layout needing more than 20 parts")
	}

	// A lowered MaxPartSize must cap the part size chosen for us, even though
	// rounding up to a minPartSize multiple would overshoot it.
	small := UploadLimits{MaxPartSize: 20 * 1024 * 1024}
	_, partSize, _, err := small.optimalPartInfo(small.maxObjectSize(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if partSize > small.maxPartSize() {
		t.Errorf("partSize %d exceeds MaxPartSize %d", partSize, small.maxPartSize())
	}
}

// An unknown size must fall back to the resolved max object size when that is
// below the 5TiB memory cap, instead of failing outright.
func TestOptimalPartInfoUnknownSizeLoweredLimits(t *testing.T) {
	l := UploadLimits{MaxPartsCount: 20} // 100GiB
	totalParts, partSize, _, err := l.optimalPartInfo(-1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := int64(totalParts)*partSize, l.maxObjectSize(); got != want {
		t.Errorf("layout covers %d bytes, want %d", got, want)
	}
}

// A MinPartSize above the internal 16MiB threshold must be the floor for the
// automatically chosen part size, otherwise the remote rejects non-final parts.
func TestOptimalPartInfoRaisedMinPartSize(t *testing.T) {
	l := UploadLimits{MinPartSize: 64 * 1024 * 1024}
	_, partSize, _, err := l.optimalPartInfo(100*l.MinPartSize, 0)
	if err != nil {
		t.Fatal(err)
	}
	if partSize < l.minPartSize() {
		t.Errorf("partSize %d is below MinPartSize %d", partSize, l.minPartSize())
	}
}

// A PartSize above MaxSinglePutObjectSize must not turn into an oversized
// single PUT; it goes multipart, or errors when multipart is disabled.
func TestPutObjectPartSizeAboveSinglePutLimit(t *testing.T) {
	var singlePuts, partPuts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch {
		case r.Method == http.MethodPost && q.Has("uploads"):
			w.Header().Set("Content-Type", "application/xml")
			io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>`+
				`<InitiateMultipartUploadResult><Bucket>bucket</Bucket><Key>object</Key>`+
				`<UploadId>upload-id</UploadId></InitiateMultipartUploadResult>`)
		case r.Method == http.MethodPut && q.Get("uploadId") != "":
			partPuts++
			io.Copy(io.Discard, r.Body)
			w.Header().Set("ETag", `"3858f62230ac3c915f300c664312c11f"`)
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && q.Get("uploadId") != "":
			io.Copy(io.Discard, r.Body)
			w.Header().Set("Content-Type", "application/xml")
			io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>`+
				`<CompleteMultipartUploadResult><Bucket>bucket</Bucket><Key>object</Key>`+
				`<ETag>&quot;3858f62230ac3c915f300c664312c11f-1&quot;</ETag></CompleteMultipartUploadResult>`)
		case r.Method == http.MethodPut:
			singlePuts++
			io.Copy(io.Discard, r.Body)
			w.Header().Set("ETag", `"3858f62230ac3c915f300c664312c11f"`)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	limits := UploadLimits{MinPartSize: 1024, MaxSinglePutObjectSize: 4096}
	client, err := New(u.Host, &Options{
		Creds:        credentials.NewStaticV4("ak", "sk", ""),
		Secure:       false,
		Region:       "us-east-1",
		UploadLimits: &limits,
	})
	if err != nil {
		t.Fatal(err)
	}

	data := bytes.Repeat([]byte("a"), 8192)
	if _, err := client.PutObject(context.Background(), "bucket", "object",
		bytes.NewReader(data), int64(len(data)), PutObjectOptions{PartSize: uint64(len(data))}); err != nil {
		t.Fatalf("PutObject: %v", err)
	}
	if singlePuts != 0 {
		t.Errorf("%d single PUTs above the %d byte limit, want 0", singlePuts, limits.MaxSinglePutObjectSize)
	}
	if partPuts != 1 {
		t.Errorf("part uploads = %d, want 1", partPuts)
	}

	_, err = client.PutObject(context.Background(), "bucket", "object",
		bytes.NewReader(data), int64(len(data)),
		PutObjectOptions{PartSize: uint64(len(data)), DisableMultipart: true})
	if code := ToErrorResponse(err).Code; code != EntityTooLarge {
		t.Fatalf("PutObject error code = %q, want %q (err %v)", code, EntityTooLarge, err)
	}
}

// An unknown length stream that outlasts the part budget must fail instead of
// completing a truncated object.
func TestPutObjectUnknownLengthTruncation(t *testing.T) {
	var completes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch {
		case r.Method == http.MethodPost && q.Has("uploads"):
			w.Header().Set("Content-Type", "application/xml")
			io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>`+
				`<InitiateMultipartUploadResult><Bucket>bucket</Bucket><Key>object</Key>`+
				`<UploadId>upload-id</UploadId></InitiateMultipartUploadResult>`)
		case r.Method == http.MethodPost && q.Get("uploadId") != "":
			completes++
			io.Copy(io.Discard, r.Body)
			w.Header().Set("Content-Type", "application/xml")
			io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>`+
				`<CompleteMultipartUploadResult><Bucket>bucket</Bucket><Key>object</Key>`+
				`<ETag>&quot;3858f62230ac3c915f300c664312c11f-2&quot;</ETag></CompleteMultipartUploadResult>`)
		default:
			io.Copy(io.Discard, r.Body)
			w.Header().Set("ETag", `"3858f62230ac3c915f300c664312c11f"`)
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	// A budget of two 1KiB parts against a 4KiB stream.
	limits := UploadLimits{MinPartSize: 1024, MaxPartSize: 1024, MaxPartsCount: 2}

	for _, tc := range []struct {
		name  string
		creds *credentials.Credentials
		opts  PutObjectOptions
	}{
		{"stream no length", credentials.NewStaticV4("ak", "sk", ""), PutObjectOptions{}},
		{"stream parallel", credentials.NewStaticV4("ak", "sk", ""), PutObjectOptions{ConcurrentStreamParts: true, NumThreads: 2}},
		{"multipart no stream", credentials.NewStaticV2("ak", "sk", ""), PutObjectOptions{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			completes = 0
			l := limits
			client, err := New(u.Host, &Options{
				Creds:        tc.creds,
				Secure:       false,
				Region:       "us-east-1",
				UploadLimits: &l,
			})
			if err != nil {
				t.Fatal(err)
			}
			reader := bytes.NewReader(bytes.Repeat([]byte("a"), 4096))
			_, err = client.PutObject(context.Background(), "bucket", "object", reader, -1, tc.opts)
			if code := ToErrorResponse(err).Code; code != EntityTooLarge {
				t.Fatalf("PutObject error code = %q, want %q (err %v)", code, EntityTooLarge, err)
			}
			if completes != 0 {
				t.Fatalf("completed %d truncated uploads, want 0", completes)
			}
		})
	}
}
