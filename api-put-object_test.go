/*
 * Minio Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2015-2017 Minio, Inc.
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
)

func TestPutObjectOptionsValidate(t *testing.T) {
	testCases := []struct {
		metadata   map[string]string
		shouldPass bool
	}{
		{map[string]string{"Content-Type": "custom/content-type"}, false},
		{map[string]string{"content-type": "custom/content-type"}, false},
		{map[string]string{"Content-Encoding": "gzip"}, false},
		{map[string]string{"Cache-Control": "blah"}, false},
		{map[string]string{"Content-Disposition": "something"}, false},
		{map[string]string{"my-custom-header": "blah"}, true},
		{map[string]string{"X-Amz-Iv": "blah"}, false},
		{map[string]string{"X-Amz-Key": "blah"}, false},
		{map[string]string{"X-Amz-Key-prefixed-header": "blah"}, false},
		{map[string]string{"custom-X-Amz-Key-middle": "blah"}, true},
		{map[string]string{"my-custom-header-X-Amz-Key": "blah"}, true},
		{map[string]string{"X-Amz-Matdesc": "blah"}, false},
		{map[string]string{"blah-X-Amz-Matdesc": "blah"}, true},
		{map[string]string{"X-Amz-MatDesc-suffix": "blah"}, true},
		{map[string]string{"x-amz-meta-X-Amz-Iv": "blah"}, false},
		{map[string]string{"x-amz-meta-X-Amz-Key": "blah"}, false},
		{map[string]string{"x-amz-meta-X-Amz-Matdesc": "blah"}, false},
	}
	for i, testCase := range testCases {
		err := PutObjectOptions{UserMetadata: testCase.metadata}.validate()

		if testCase.shouldPass && err != nil {
			t.Errorf("Test %d - output did not match with reference results", i+1)
		}
	}
}

// test that metadata is properly converted to headers
func TestPutObjectOptionsHeaderMetadata(t *testing.T) {

	// test that invalid header keys are not included
	testCases := []struct {
		metakey    string
		shouldPass bool
	}{
		{"It has spaces", false},
		{"It:has@illegal=characters", false},
		{"It-Is-Fine", true},
		{"Numbers-098987987-Should-Work", true},
		{"Crazy-!#$%&'*+-.^_`|~-Should-193832-Be-Fine", true},
	}

	// construct the meta map from test cases
	meta := make(map[string]string)
	for _, testCase := range testCases {
		meta[testCase.metakey] = "somevalue"
	}

	opt := PutObjectOptions{UserMetadata: meta}
	header := opt.Header()

	// iterate through test cases, ensure header is correct
	for _, testCase := range testCases {
		_, ok := header["X-Amz-Meta-"+testCase.metakey]
		if ok != testCase.shouldPass {
			t.Errorf("Test case %s should be in headers %t, but was %t", testCase.metakey, testCase.shouldPass, ok)
		}
	}
}
