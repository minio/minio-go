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
