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
	"net/http"
	"reflect"
	"strings"
	"testing"
)

const (
	gb1    = 1024 * 1024 * 1024
	gb5    = 5 * gb1
	gb5p1  = gb5 + 1
	gb10p1 = 2*gb5 + 1
	gb10p2 = 2*gb5 + 2
)

func TestPartsRequired(t *testing.T) {
	testCases := []struct {
		size, ref int64
	}{
		{0, 0},
		{1, 1},
		{gb5, 10},
		{gb5p1, 10},
		{2 * gb5, 20},
		{gb10p1, 20},
		{gb10p2, 20},
		{gb10p1 + gb10p2, 40},
		{maxMultipartPutObjectSize, 10000},
	}

	for i, testCase := range testCases {
		res := partsRequired(testCase.size)
		if res != testCase.ref {
			t.Errorf("Test %d - output did not match with reference results, Expected %d, got %d", i+1, testCase.ref, res)
		}
	}
}

func TestCalculateEvenSplits(t *testing.T) {
	testCases := []struct {
		// input size and source object
		size int64
		src  CopySrcOptions

		// output part-indexes
		starts, ends []int64
	}{
		{0, CopySrcOptions{Start: -1}, nil, nil},
		{1, CopySrcOptions{Start: -1}, []int64{0}, []int64{0}},
		{1, CopySrcOptions{Start: 0}, []int64{0}, []int64{0}},

		{gb1, CopySrcOptions{Start: -1}, []int64{0, 536870912}, []int64{536870911, 1073741823}},
		{
			gb5,
			CopySrcOptions{Start: -1},
			[]int64{
				0, 536870912, 1073741824, 1610612736, 2147483648, 2684354560,
				3221225472, 3758096384, 4294967296, 4831838208,
			},
			[]int64{
				536870911, 1073741823, 1610612735, 2147483647, 2684354559, 3221225471,
				3758096383, 4294967295, 4831838207, 5368709119,
			},
		},

		// 2 part splits
		{
			gb5p1,
			CopySrcOptions{Start: -1},
			[]int64{
				0, 536870913, 1073741825, 1610612737, 2147483649, 2684354561,
				3221225473, 3758096385, 4294967297, 4831838209,
			},
			[]int64{
				536870912, 1073741824, 1610612736, 2147483648, 2684354560, 3221225472,
				3758096384, 4294967296, 4831838208, 5368709120,
			},
		},
		{
			gb5p1,
			CopySrcOptions{Start: -1},
			[]int64{
				0, 536870913, 1073741825, 1610612737, 2147483649, 2684354561,
				3221225473, 3758096385, 4294967297, 4831838209,
			},
			[]int64{
				536870912, 1073741824, 1610612736, 2147483648, 2684354560, 3221225472,
				3758096384, 4294967296, 4831838208, 5368709120,
			},
		},

		// 3 part splits
		{
			gb10p1,
			CopySrcOptions{Start: -1},
			[]int64{
				0, 536870913, 1073741825, 1610612737, 2147483649, 2684354561,
				3221225473, 3758096385, 4294967297, 4831838209, 5368709121,
				5905580033, 6442450945, 6979321857, 7516192769, 8053063681,
				8589934593, 9126805505, 9663676417, 10200547329,
			},
			[]int64{
				536870912, 1073741824, 1610612736, 2147483648, 2684354560,
				3221225472, 3758096384, 4294967296, 4831838208, 5368709120,
				5905580032, 6442450944, 6979321856, 7516192768, 8053063680,
				8589934592, 9126805504, 9663676416, 10200547328, 10737418240,
			},
		},
		{
			gb10p2,
			CopySrcOptions{Start: -1},
			[]int64{
				0, 536870913, 1073741826, 1610612738, 2147483650, 2684354562,
				3221225474, 3758096386, 4294967298, 4831838210, 5368709122,
				5905580034, 6442450946, 6979321858, 7516192770, 8053063682,
				8589934594, 9126805506, 9663676418, 10200547330,
			},
			[]int64{
				536870912, 1073741825, 1610612737, 2147483649, 2684354561,
				3221225473, 3758096385, 4294967297, 4831838209, 5368709121,
				5905580033, 6442450945, 6979321857, 7516192769, 8053063681,
				8589934593, 9126805505, 9663676417, 10200547329, 10737418241,
			},
		},
	}

	for i, testCase := range testCases {
		resStart, resEnd := calculateEvenSplits(testCase.size, testCase.src)
		if !reflect.DeepEqual(testCase.starts, resStart) || !reflect.DeepEqual(testCase.ends, resEnd) {
			t.Errorf("Test %d - output did not match with reference results, Expected %d/%d, got %d/%d", i+1, testCase.starts, testCase.ends, resStart, resEnd)
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
