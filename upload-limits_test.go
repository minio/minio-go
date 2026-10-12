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
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
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
		// totalPartsCount would not survive the float64 round trip.
		{"parts count at the int64 ceiling", UploadLimits{MinPartSize: 1, MaxPartSize: 1, MaxPartsCount: math.MaxInt64}, true},
		{"parts count above the float64 exact range", UploadLimits{MinPartSize: 1, MaxPartSize: 1, MaxPartsCount: 1<<53 + 1}, true},
		{"parts count at the float64 exact range", UploadLimits{MinPartSize: 1, MaxPartSize: 1, MaxPartsCount: 1 << 53}, false},
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

// The automatically chosen part size must never fall below MinPartSize, or the
// remote rejects every non-final part.
func TestOptimalPartInfoRaisedMinPartSize(t *testing.T) {
	for _, tc := range []struct {
		name       string
		limits     UploadLimits
		objectSize int64
	}{
		// 1GiB/10000 rounds to 16MiB on the internal threshold alone.
		{"1GiB at 64MiB minimum", UploadLimits{MinPartSize: 64 * 1024 * 1024}, 1024 * 1024 * 1024},
		{"100 parts at 64MiB minimum", UploadLimits{MinPartSize: 64 * 1024 * 1024}, 100 * 64 * 1024 * 1024},
		// Below maxPartsCount the division rounds down to a zero part size.
		{"object smaller than the parts count", UploadLimits{}, 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			totalParts, partSize, lastPartSize, err := tc.limits.optimalPartInfo(tc.objectSize, 0)
			if err != nil {
				t.Fatal(err)
			}
			if partSize < tc.limits.minPartSize() {
				t.Errorf("partSize %d is below MinPartSize %d", partSize, tc.limits.minPartSize())
			}
			if totalParts < 0 || int64(totalParts) > tc.limits.maxPartsCount() {
				t.Errorf("totalPartsCount = %d, want within [0, %d]", totalParts, tc.limits.maxPartsCount())
			}
			if lastPartSize > partSize {
				t.Errorf("lastPartSize %d exceeds partSize %d", lastPartSize, partSize)
			}
		})
	}
}

// An empty object has no parts, so the layout must be zero throughout rather
// than reporting a minimum-sized part and last part for a zero-part upload.
func TestOptimalPartInfoEmptyObject(t *testing.T) {
	totalParts, partSize, lastPartSize, err := OptimalPartInfo(0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if totalParts != 0 || partSize != 0 || lastPartSize != 0 {
		t.Errorf("got (%d parts, %d part size, %d last part size), want all zero",
			totalParts, partSize, lastPartSize)
	}
}

// A MaxPartSize at the int64 ceiling rounds through float64 to a value that
// converts back negative, which would panic the make() in the multipart paths.
func TestUploadLimitsRejectsUnrepresentableMaxPartSize(t *testing.T) {
	l := UploadLimits{MaxPartSize: math.MaxInt64, MaxPartsCount: 1}
	if err := l.validate(); err == nil {
		t.Fatal("validate() accepted a MaxPartSize that cannot round-trip through float64")
	}
	if _, err := New("play.min.io", &Options{
		Creds:        credentials.NewStaticV4("id", "secret", ""),
		UploadLimits: &l,
	}); err == nil {
		t.Fatal("New() accepted a MaxPartSize that cannot round-trip through float64")
	}

	// A part size one ulp below the boundary still round-trips, so the layout
	// stays positive and make() is safe.
	ok := UploadLimits{MaxPartSize: math.MaxInt64 - (1 << 11), MaxPartsCount: 1}
	if err := ok.validate(); err != nil {
		t.Fatalf("validate() rejected a representable MaxPartSize: %v", err)
	}
	_, partSize, _, err := ok.optimalPartInfo(ok.maxObjectSize(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if partSize <= 0 {
		t.Fatalf("partSize = %d, want positive (make would panic)", partSize)
	}

	// The same for the parts count: an accepted MaxPartsCount must still yield a
	// layout whose part count is a usable int.
	parts := UploadLimits{MinPartSize: 1, MaxPartSize: 1, MaxPartsCount: 1 << 53}
	if err := parts.validate(); err != nil {
		t.Fatalf("validate() rejected a representable MaxPartsCount: %v", err)
	}
	totalParts, _, _, err := parts.optimalPartInfo(parts.maxObjectSize(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if int64(totalParts) != parts.MaxPartsCount {
		t.Fatalf("totalPartsCount = %d, want %d", totalParts, parts.MaxPartsCount)
	}
}

// The automatic layout must never need more than maxPartsCount parts. Rounding
// a truncated objectSize/maxPartsCount leaves the part size one byte short
// whenever that quotient already sits on a rounding-unit multiple.
func TestOptimalPartInfoPartsCountCeiling(t *testing.T) {
	for _, tc := range []struct {
		name       string
		limits     UploadLimits
		objectSize int64
	}{
		{"one byte over the 16MiB unit", UploadLimits{}, int64(defaultMaxPartsCount)*minPartSize + 1},
		{"half a unit over", UploadLimits{}, int64(defaultMaxPartsCount)*minPartSize + minPartSize/2},
		{"one under the next unit", UploadLimits{}, int64(defaultMaxPartsCount)*minPartSize*2 - 1},
		{"raised minimum", UploadLimits{MinPartSize: 64 * 1024 * 1024}, int64(defaultMaxPartsCount)*64*1024*1024 + 1},
		{"lowered parts count", UploadLimits{MaxPartsCount: 20}, 20*minPartSize + 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			totalParts, partSize, lastPartSize, err := tc.limits.optimalPartInfo(tc.objectSize, 0)
			if err != nil {
				t.Fatal(err)
			}
			if int64(totalParts) > tc.limits.maxPartsCount() {
				t.Errorf("totalPartsCount = %d, exceeds MaxPartsCount %d (partSize %d)",
					totalParts, tc.limits.maxPartsCount(), partSize)
			}
			if got := int64(totalParts-1)*partSize + lastPartSize; got != tc.objectSize {
				t.Errorf("layout covers %d bytes, want %d", got, tc.objectSize)
			}
		})
	}
}

// The resolved limits must be readable off a built client without a round trip.
func TestClientUploadLimitsAccessor(t *testing.T) {
	// The shape AIStor configures for replication: the two size ceilings raised
	// to 5TiB, MinPartSize and MaxPartsCount left at the S3 defaults.
	const fiveTiB = int64(5) * 1024 * 1024 * 1024 * 1024
	limits := UploadLimits{MaxPartSize: fiveTiB, MaxSinglePutObjectSize: fiveTiB}
	c, err := New("play.min.io", &Options{
		Creds:        credentials.NewStaticV4("id", "secret", ""),
		UploadLimits: &limits,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := UploadLimits{
		MinPartSize:            defaultMinPartSize,
		MaxPartSize:            fiveTiB,
		MaxPartsCount:          defaultMaxPartsCount,
		MaxSinglePutObjectSize: fiveTiB,
	}
	if got := c.UploadLimits(); got != want {
		t.Errorf("UploadLimits() = %+v, want %+v", got, want)
	}

	// Core embeds *Client, so it reports the same limits.
	core, err := NewCore("play.min.io", &Options{
		Creds:        credentials.NewStaticV4("id", "secret", ""),
		UploadLimits: &limits,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := core.UploadLimits(); got != want {
		t.Errorf("Core.UploadLimits() = %+v, want %+v", got, want)
	}

	// Unset limits and a client not built by New both read back as S3.
	def := UploadLimits{
		MinPartSize:            defaultMinPartSize,
		MaxPartSize:            defaultMaxPartSize,
		MaxPartsCount:          defaultMaxPartsCount,
		MaxSinglePutObjectSize: defaultMaxSinglePutObjectSize,
	}
	plain, err := New("play.min.io", &Options{Creds: credentials.NewStaticV4("id", "secret", "")})
	if err != nil {
		t.Fatal(err)
	}
	if got := plain.UploadLimits(); got != def {
		t.Errorf("UploadLimits() = %+v, want %+v", got, def)
	}
	if got := (&Client{}).UploadLimits(); got != def {
		t.Errorf("zero-value Client UploadLimits() = %+v, want %+v", got, def)
	}
}

// A part size above MaxInt64 must not wrap negative and slip past the ceiling
// checks that are written in terms of int64.
func TestUploadLimitsUnsignedPartSizeGuards(t *testing.T) {
	const huge = uint64(math.MaxInt64) + 1

	if _, _, _, err := OptimalPartInfo(1024*1024*1024, huge); err == nil {
		t.Error("optimalPartInfo accepted a part size above MaxInt64")
	} else if msg := ToErrorResponse(err).Message; !strings.Contains(msg, "bigger than allowed maximum") {
		t.Errorf("optimalPartInfo error = %q, want it to report the maximum", msg)
	}

	c, err := New("play.min.io", &Options{
		Creds:           credentials.NewStaticV4("id", "secret", ""),
		TrailingHeaders: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := (AppendObjectOptions{ChunkSize: huge}).validate(c); err == nil {
		t.Error("AppendObjectOptions.validate accepted a chunk size above MaxInt64")
	}
}

// An oversized part must not be reported as a single PUT problem.
func TestPartTooLargeMessage(t *testing.T) {
	c, err := New("play.min.io", &Options{Creds: credentials.NewStaticV4("id", "secret", "")})
	if err != nil {
		t.Fatal(err)
	}
	maxPartSize := c.UploadLimits().MaxPartSize
	_, err = c.uploadPart(context.Background(), uploadPartParams{
		bucketName: "bucket", objectName: "object", uploadID: "upload-id",
		reader: bytes.NewReader(nil), partNumber: 1, size: maxPartSize + 1,
	})
	resp := ToErrorResponse(err)
	if resp.Code != EntityTooLarge {
		t.Fatalf("error code = %q, want %q (err %v)", resp.Code, EntityTooLarge, err)
	}
	if strings.Contains(resp.Message, "single PUT") {
		t.Errorf("part size error mentions a single PUT: %q", resp.Message)
	}
	if !strings.Contains(resp.Message, "part size") {
		t.Errorf("part size error does not mention the part size: %q", resp.Message)
	}
}

// The 5GiB default must not gate PutObject: remotes such as MinIO/AIStor accept
// single PUTs far above Amazon's, and PutObjectsSnowball sets DisableMultipart
// itself. An explicitly configured limit is enforced.
func TestPutObjectSinglePutLimitOnlyWhenConfigured(t *testing.T) {
	// Atomic: the aborted 6GiB request below leaves its handler running while
	// the test resets the counter.
	var singlePuts atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			singlePuts.Add(1)
		}
		io.Copy(io.Discard, r.Body)
		w.Header().Set("ETag", `"3858f62230ac3c915f300c664312c11f"`)
		w.WriteHeader(http.StatusOK)
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

	// The snowball shape at the real default: 6GiB with DisableMultipart. The
	// size gate runs before the request, so a canceled context separates the
	// two outcomes without putting 6GiB on the wire — EntityTooLarge means the
	// default was enforced, a context error means routing let it through.
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = client.PutObject(canceled, "bucket", "snowball.tar",
		bytes.NewReader(nil), int64(6)<<30, PutObjectOptions{DisableMultipart: true})
	if code := ToErrorResponse(err).Code; code == EntityTooLarge {
		t.Fatalf("PutObject refused a 6GiB single PUT client-side: %v", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("PutObject error = %v, want the canceled context to surface", err)
	}

	// A normal-sized object on default limits still goes out as one PUT.
	singlePuts.Store(0)
	data := bytes.Repeat([]byte("a"), 8192)
	if _, err := client.PutObject(context.Background(), "bucket", "object",
		bytes.NewReader(data), int64(len(data)), PutObjectOptions{DisableMultipart: true}); err != nil {
		t.Fatalf("PutObject: %v", err)
	}
	if got := singlePuts.Load(); got != 1 {
		t.Fatalf("single PUTs = %d, want 1 (upload was refused client-side)", got)
	}

	// Setting the limit explicitly opts in to enforcement.
	small, err := New(u.Host, &Options{
		Creds:        credentials.NewStaticV4("ak", "sk", ""),
		Secure:       false,
		Region:       "us-east-1",
		UploadLimits: &UploadLimits{MaxSinglePutObjectSize: 4096},
	})
	if err != nil {
		t.Fatal(err)
	}
	singlePuts.Store(0)
	_, err = small.PutObject(context.Background(), "bucket", "object",
		bytes.NewReader(data), int64(len(data)), PutObjectOptions{DisableMultipart: true})
	if code := ToErrorResponse(err).Code; code != EntityTooLarge {
		t.Fatalf("configured limit: error code = %q, want %q (err %v)", code, EntityTooLarge, err)
	}
	if got := singlePuts.Load(); got != 0 {
		t.Fatalf("configured limit: %d PUTs issued, want 0", got)
	}
}

// MaxSinglePutObjectSize is a Client.PutObject routing decision. Core.PutObject
// is the raw S3 call and sends the PUT as given, as its doc comment states.
func TestCorePutObjectIgnoresSinglePutLimit(t *testing.T) {
	var puts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			puts++
		}
		io.Copy(io.Discard, r.Body)
		w.Header().Set("ETag", `"3858f62230ac3c915f300c664312c11f"`)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	limits := UploadLimits{MinPartSize: 1024, MaxSinglePutObjectSize: 4096}
	opts := &Options{
		Creds:        credentials.NewStaticV4("ak", "sk", ""),
		Secure:       false,
		Region:       "us-east-1",
		UploadLimits: &limits,
	}
	core, err := NewCore(u.Host, opts)
	if err != nil {
		t.Fatal(err)
	}

	data := bytes.Repeat([]byte("a"), 8192)
	if _, err := core.PutObject(context.Background(), "bucket", "object",
		bytes.NewReader(data), int64(len(data)), "", "", PutObjectOptions{}); err != nil {
		t.Fatalf("Core.PutObject: %v", err)
	}
	if puts != 1 {
		t.Fatalf("Core.PutObject issued %d PUTs, want 1", puts)
	}

	// The limit was set explicitly, so Client.PutObject does refuse the same
	// oversized single PUT; only the raw Core call bypasses it.
	client, err := New(u.Host, opts)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.PutObject(context.Background(), "bucket", "object",
		bytes.NewReader(data), int64(len(data)), PutObjectOptions{DisableMultipart: true})
	if code := ToErrorResponse(err).Code; code != EntityTooLarge {
		t.Fatalf("Client.PutObject error code = %q, want %q (err %v)", code, EntityTooLarge, err)
	}
	if puts != 1 {
		t.Fatalf("PUTs issued = %d, want 1 (only the Core call)", puts)
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

	errBroken := errors.New("reader broke")

	for _, tc := range []struct {
		name string
		// readErr, when set, makes the reader deliver exactly the part budget
		// and then fail instead of reaching EOF.
		readErr error
		creds   *credentials.Credentials
		opts    PutObjectOptions
	}{
		{"stream no length", nil, credentials.NewStaticV4("ak", "sk", ""), PutObjectOptions{}},
		{"stream parallel", nil, credentials.NewStaticV4("ak", "sk", ""), PutObjectOptions{ConcurrentStreamParts: true, NumThreads: 2}},
		{"multipart no stream", nil, credentials.NewStaticV2("ak", "sk", ""), PutObjectOptions{}},
		{"failing trailing read", errBroken, credentials.NewStaticV4("ak", "sk", ""), PutObjectOptions{}},
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

			var reader io.Reader = bytes.NewReader(bytes.Repeat([]byte("a"), 4096))
			if tc.readErr != nil {
				reader = &failAtEOFReader{
					Reader: bytes.NewReader(bytes.Repeat([]byte("a"), 2048)),
					err:    tc.readErr,
				}
			}
			_, err = client.PutObject(context.Background(), "bucket", "object", reader, -1, tc.opts)
			switch {
			case tc.readErr != nil:
				if !errors.Is(err, tc.readErr) {
					t.Fatalf("PutObject error = %v, want %v", err, tc.readErr)
				}
			default:
				resp := ToErrorResponse(err)
				if resp.Code != EntityTooLarge {
					t.Fatalf("PutObject error code = %q, want %q (err %v)", resp.Code, EntityTooLarge, err)
				}
				// The part budget ran out; this is neither a single PUT nor an
				// object-size ceiling, and the bytes that fit are not a maximum.
				if strings.Contains(resp.Message, "single PUT") {
					t.Errorf("truncation error mentions a single PUT: %q", resp.Message)
				}
				if strings.Contains(resp.Message, "maximum allowed object size") {
					t.Errorf("truncation error reports an object-size maximum: %q", resp.Message)
				}
				// Two 1KiB parts were laid out and uploaded before the reader ran on.
				if !strings.Contains(resp.Message, "‘2’ parts") || !strings.Contains(resp.Message, "‘2048’ bytes") {
					t.Errorf("truncation error does not report the part budget and bytes uploaded: %q", resp.Message)
				}
			}
			if completes != 0 {
				t.Fatalf("completed %d truncated uploads, want 0", completes)
			}
		})
	}
}

// failAtEOFReader substitutes err for the io.EOF of the wrapped reader, so the
// trailing zero-byte read reports a failure rather than a drained stream.
type failAtEOFReader struct {
	io.Reader
	err error
}

func (r *failAtEOFReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	if err == io.EOF {
		err = r.err
	}
	return n, err
}
