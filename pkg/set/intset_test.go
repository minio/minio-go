/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2015-2026 MinIO, Inc.
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

package set

import (
	"reflect"
	"testing"
)

// IntSet.MarshalJSON() is called with series of cases for valid and erroneous inputs and the result is validated.
func TestIntSetMarshalJSON(t *testing.T) {
	testCases := []struct {
		set            IntSet
		expectedResult string
	}{
		// Test set with values.
		{CreateIntSet(1, 2, 3), `[1,2,3]`},
		// Test empty set.
		{NewIntSet(), "[]"},
	}

	for _, testCase := range testCases {
		if result, _ := testCase.set.MarshalJSON(); string(result) != testCase.expectedResult {
			t.Fatalf("expected: %s, got: %s", testCase.expectedResult, string(result))
		}
	}
}

// IntSet.UnmarshalJSON() is called with series of cases for valid and erroneous inputs and the result is validated.
func TestIntSetUnmarshalJSON(t *testing.T) {
	testCases := []struct {
		data           []byte
		expectedResult string
	}{
		// Test to convert JSON array to set.
		{[]byte(`[1,2,3]`), `[1 2 3]`},
		// Test to convert JSON empty array to set.
		{[]byte(`[]`), `[]`},
	}

	for _, testCase := range testCases {
		var set IntSet
		set.UnmarshalJSON(testCase.data)
		if result := set.String(); result != testCase.expectedResult {
			t.Fatalf("expected: %s, got: %s", testCase.expectedResult, result)
		}
	}
}

// IntSet.String() is called with series of cases for valid and erroneous inputs and the result is validated.
func TestIntSetString(t *testing.T) {
	testCases := []struct {
		set            IntSet
		expectedResult string
	}{
		// Test empty set.
		{NewIntSet(), `[]`},
		// Test set with value.
		{CreateIntSet(42), `[42]`},
		// Test set with multiple values.
		{CreateIntSet(1, 2, 3), `[1 2 3]`},
	}

	for _, testCase := range testCases {
		if str := testCase.set.String(); str != testCase.expectedResult {
			t.Fatalf("expected: %s, got: %s", testCase.expectedResult, str)
		}
	}
}

// IntSet.ToSlice() is called with series of cases for valid and erroneous inputs and the result is validated.
func TestIntSetToSlice(t *testing.T) {
	testCases := []struct {
		set            IntSet
		expectedResult []int
	}{
		// Test empty set.
		{NewIntSet(), []int{}},
		// Test set with value.
		{CreateIntSet(42), []int{42}},
		// Test set with multiple values (should be sorted).
		{CreateIntSet(3, 1, 2), []int{1, 2, 3}},
	}

	for _, testCase := range testCases {
		islice := testCase.set.ToSlice()
		if !reflect.DeepEqual(islice, testCase.expectedResult) {
			t.Fatalf("expected: %v, got: %v", testCase.expectedResult, islice)
		}
	}
}

func TestIntSet_UnmarshalJSON(t *testing.T) {
	type args struct {
		data         []byte
		expectResult []int
	}
	tests := []struct {
		name    string
		set     IntSet
		args    args
		wantErr bool
	}{
		{
			name: "test ints",
			set:  NewIntSet(),
			args: args{
				data:         []byte(`[1,2,3]`),
				expectResult: []int{1, 2, 3},
			},
			wantErr: false,
		},
		{
			name: "test negative ints",
			set:  NewIntSet(),
			args: args{
				data:         []byte(`[-1,-2,0,1,2]`),
				expectResult: []int{-2, -1, 0, 1, 2},
			},
			wantErr: false,
		},
		{
			name: "test empty array",
			set:  NewIntSet(),
			args: args{
				data:         []byte(`[]`),
				expectResult: []int{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.set.UnmarshalJSON(tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			slice := tt.set.ToSlice()
			if !reflect.DeepEqual(slice, tt.args.expectResult) {
				t.Errorf("IntSet() get %v, want %v", slice, tt.args.expectResult)
			}
		})
	}
}
