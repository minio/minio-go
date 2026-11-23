/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2022 MinIO, Inc.
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

package tags

import (
	"fmt"
	"testing"
)

func TestParseTags(t *testing.T) {
	testCases := []struct {
		tags        string
		expectedErr bool
		count       int
	}{
		{
			"key1=value1&key2=value2",
			false,
			2,
		},
		{
			"store+forever=false&factory=true",
			false,
			2,
		},
		{
			" store forever =false&factory=true",
			false,
			2,
		},
		{
			"key=value=",
			false,
			1,
		},
		{
			fmt.Sprintf("%0128d=%0256d", 1, 1),
			false,
			1,
		},
		// Failure cases - https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/Using_Tags.html#tag-restrictions
		{
			"key1=value1&key1=value2",
			true,
			0,
		},
		{
			"key$=value1",
			true,
			0,
		},
		{
			"key1=value$",
			true,
			0,
		},
		{
			// An invalid escape must not be forgotten when a later pair is valid.
			"key1=%zz&key2=value2",
			true,
			0,
		},
		{
			fmt.Sprintf("%0128d=%0257d", 1, 1),
			true,
			0,
		},
		{
			fmt.Sprintf("%0129d=%0256d", 1, 1),
			true,
			0,
		},
		{
			fmt.Sprintf("%0129d=%0257d", 1, 1),
			true,
			0,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.tags, func(t *testing.T) {
			tt, err := ParseObjectTags(testCase.tags)
			if !testCase.expectedErr && err != nil {
				t.Errorf("Expected success but failed with %v", err)
			}
			if testCase.expectedErr && err == nil {
				t.Error("Expected failure but found success")
			}
			if err == nil {
				if tt.Count() != testCase.count {
					t.Errorf("Expected count %d, got %d", testCase.count, tt.Count())
				} else {
					t.Logf("%s", tt)
				}
			}
		})
	}
}

func TestSetExistingKeyAtLimit(t *testing.T) {
	testCases := []struct {
		name     string
		isObject bool
		limit    int
	}{
		{"object", true, maxObjectTagCount},
		{"bucket", false, maxTagCount},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			tagMap := make(map[string]string, testCase.limit)
			for i := 0; i < testCase.limit; i++ {
				tagMap[fmt.Sprintf("key%d", i)] = "value"
			}
			tt, err := NewTags(tagMap, testCase.isObject)
			if err != nil {
				t.Fatalf("Expected success but failed with %v", err)
			}
			if err = tt.Set("key0", "updated"); err != nil {
				t.Errorf("Expected updating an existing key to succeed but failed with %v", err)
			}
			if got := tt.ToMap()["key0"]; got != "updated" {
				t.Errorf("Expected value %q, got %q", "updated", got)
			}
			if tt.Count() != testCase.limit {
				t.Errorf("Expected count %d, got %d", testCase.limit, tt.Count())
			}
			if err = tt.Set("another", "value"); err == nil {
				t.Error("Expected adding a tag beyond the limit to fail but found success")
			}
		})
	}
}

func BenchmarkParseTags(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ParseObjectTags("key1=value1&key2=value2")
	}
}
