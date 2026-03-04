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
	"bytes"
	"reflect"
	"testing"

	"github.com/tinylib/msgp/msgp"
)

// TestStringSetMsgpRoundtrip tests msgp serialization/deserialization for StringSet
func TestStringSetMsgpRoundtrip(t *testing.T) {
	testCases := []struct {
		name string
		set  StringSet
	}{
		{
			name: "empty set",
			set:  NewStringSet(),
		},
		{
			name: "single element",
			set:  CreateStringSet("foo"),
		},
		{
			name: "multiple elements",
			set:  CreateStringSet("foo", "bar", "baz"),
		},
		{
			name: "with special characters",
			set:  CreateStringSet("hello world", "test@example.com", "path/to/file"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test MarshalMsg/UnmarshalMsg
			data, err := tc.set.MarshalMsg(nil)
			if err != nil {
				t.Fatalf("MarshalMsg() error = %v", err)
			}

			var decoded StringSet
			_, err = decoded.UnmarshalMsg(data)
			if err != nil {
				t.Fatalf("UnmarshalMsg() error = %v", err)
			}

			if !tc.set.Equals(decoded) {
				t.Errorf("Roundtrip failed: original=%v, decoded=%v", tc.set.ToSlice(), decoded.ToSlice())
			}
		})
	}
}

// TestStringSetMsgpEncodeDecodeMsg tests EncodeMsg/DecodeMsg for StringSet
func TestStringSetMsgpEncodeDecodeMsg(t *testing.T) {
	testCases := []struct {
		name string
		set  StringSet
	}{
		{
			name: "empty set",
			set:  NewStringSet(),
		},
		{
			name: "populated set",
			set:  CreateStringSet("alpha", "beta", "gamma"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			writer := msgp.NewWriter(&buf)

			// Encode
			err := tc.set.EncodeMsg(writer)
			if err != nil {
				t.Fatalf("EncodeMsg() error = %v", err)
			}
			writer.Flush()

			// Decode
			reader := msgp.NewReader(&buf)
			var decoded StringSet
			err = decoded.DecodeMsg(reader)
			if err != nil {
				t.Fatalf("DecodeMsg() error = %v", err)
			}

			if !tc.set.Equals(decoded) {
				t.Errorf("EncodeMsg/DecodeMsg roundtrip failed: original=%v, decoded=%v", tc.set.ToSlice(), decoded.ToSlice())
			}
		})
	}
}

// TestStringSetMsgpBinary tests MarshalBinary/UnmarshalBinary for StringSet
func TestStringSetMsgpBinary(t *testing.T) {
	original := CreateStringSet("one", "two", "three")

	data, err := original.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v", err)
	}

	var decoded StringSet
	err = decoded.UnmarshalBinary(data)
	if err != nil {
		t.Fatalf("UnmarshalBinary() error = %v", err)
	}

	if !original.Equals(decoded) {
		t.Errorf("Binary roundtrip failed: original=%v, decoded=%v", original.ToSlice(), decoded.ToSlice())
	}
}

// TestStringSetMsgpAppendBinary tests AppendBinary for StringSet
func TestStringSetMsgpAppendBinary(t *testing.T) {
	set1 := CreateStringSet("foo", "bar")
	set2 := CreateStringSet("baz", "qux")

	// Append set1
	data, err := set1.AppendBinary(nil)
	if err != nil {
		t.Fatalf("AppendBinary(set1) error = %v", err)
	}

	// Append set2 to existing data
	data, err = set2.AppendBinary(data)
	if err != nil {
		t.Fatalf("AppendBinary(set2) error = %v", err)
	}

	// Decode both sets
	var decoded1 StringSet
	remaining, err := decoded1.UnmarshalMsg(data)
	if err != nil {
		t.Fatalf("UnmarshalMsg(set1) error = %v", err)
	}

	var decoded2 StringSet
	_, err = decoded2.UnmarshalMsg(remaining)
	if err != nil {
		t.Fatalf("UnmarshalMsg(set2) error = %v", err)
	}

	if !set1.Equals(decoded1) {
		t.Errorf("AppendBinary failed for set1: original=%v, decoded=%v", set1.ToSlice(), decoded1.ToSlice())
	}

	if !set2.Equals(decoded2) {
		t.Errorf("AppendBinary failed for set2: original=%v, decoded=%v", set2.ToSlice(), decoded2.ToSlice())
	}
}

// TestStringSetMsgsize tests Msgsize for StringSet
func TestStringSetMsgsize(t *testing.T) {
	testCases := []struct {
		name string
		set  StringSet
	}{
		{
			name: "empty set",
			set:  NewStringSet(),
		},
		{
			name: "small set",
			set:  CreateStringSet("a", "b"),
		},
		{
			name: "larger set",
			set:  CreateStringSet("alpha", "beta", "gamma", "delta", "epsilon"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			size := tc.set.Msgsize()
			data, err := tc.set.MarshalMsg(nil)
			if err != nil {
				t.Fatalf("MarshalMsg() error = %v", err)
			}

			// Msgsize should be >= actual size
			if size < len(data) {
				t.Errorf("Msgsize() = %d, but actual size = %d", size, len(data))
			}
		})
	}
}

// TestIntSetMsgpRoundtrip tests msgp serialization/deserialization for IntSet
func TestIntSetMsgpRoundtrip(t *testing.T) {
	testCases := []struct {
		name string
		set  IntSet
	}{
		{
			name: "empty set",
			set:  NewIntSet(),
		},
		{
			name: "single element",
			set:  CreateIntSet(42),
		},
		{
			name: "multiple elements",
			set:  CreateIntSet(1, 2, 3, 4, 5),
		},
		{
			name: "negative numbers",
			set:  CreateIntSet(-10, -5, 0, 5, 10),
		},
		{
			name: "large numbers",
			set:  CreateIntSet(1000000, 2000000, 3000000),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test MarshalMsg/UnmarshalMsg
			data, err := tc.set.MarshalMsg(nil)
			if err != nil {
				t.Fatalf("MarshalMsg() error = %v", err)
			}

			var decoded IntSet
			_, err = decoded.UnmarshalMsg(data)
			if err != nil {
				t.Fatalf("UnmarshalMsg() error = %v", err)
			}

			if !tc.set.Equals(decoded) {
				t.Errorf("Roundtrip failed: original=%v, decoded=%v", tc.set.ToSlice(), decoded.ToSlice())
			}
		})
	}
}

// TestIntSetMsgpEncodeDecodeMsg tests EncodeMsg/DecodeMsg for IntSet
func TestIntSetMsgpEncodeDecodeMsg(t *testing.T) {
	testCases := []struct {
		name string
		set  IntSet
	}{
		{
			name: "empty set",
			set:  NewIntSet(),
		},
		{
			name: "populated set",
			set:  CreateIntSet(100, 200, 300, 400),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			writer := msgp.NewWriter(&buf)

			// Encode
			err := tc.set.EncodeMsg(writer)
			if err != nil {
				t.Fatalf("EncodeMsg() error = %v", err)
			}
			writer.Flush()

			// Decode
			reader := msgp.NewReader(&buf)
			var decoded IntSet
			err = decoded.DecodeMsg(reader)
			if err != nil {
				t.Fatalf("DecodeMsg() error = %v", err)
			}

			if !tc.set.Equals(decoded) {
				t.Errorf("EncodeMsg/DecodeMsg roundtrip failed: original=%v, decoded=%v", tc.set.ToSlice(), decoded.ToSlice())
			}
		})
	}
}

// TestIntSetMsgpBinary tests MarshalBinary/UnmarshalBinary for IntSet
func TestIntSetMsgpBinary(t *testing.T) {
	original := CreateIntSet(10, 20, 30, 40, 50)

	data, err := original.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v", err)
	}

	var decoded IntSet
	err = decoded.UnmarshalBinary(data)
	if err != nil {
		t.Fatalf("UnmarshalBinary() error = %v", err)
	}

	if !original.Equals(decoded) {
		t.Errorf("Binary roundtrip failed: original=%v, decoded=%v", original.ToSlice(), decoded.ToSlice())
	}
}

// TestIntSetMsgpAppendBinary tests AppendBinary for IntSet
func TestIntSetMsgpAppendBinary(t *testing.T) {
	set1 := CreateIntSet(1, 2, 3)
	set2 := CreateIntSet(100, 200, 300)

	// Append set1
	data, err := set1.AppendBinary(nil)
	if err != nil {
		t.Fatalf("AppendBinary(set1) error = %v", err)
	}

	// Append set2 to existing data
	data, err = set2.AppendBinary(data)
	if err != nil {
		t.Fatalf("AppendBinary(set2) error = %v", err)
	}

	// Decode both sets
	var decoded1 IntSet
	remaining, err := decoded1.UnmarshalMsg(data)
	if err != nil {
		t.Fatalf("UnmarshalMsg(set1) error = %v", err)
	}

	var decoded2 IntSet
	_, err = decoded2.UnmarshalMsg(remaining)
	if err != nil {
		t.Fatalf("UnmarshalMsg(set2) error = %v", err)
	}

	if !set1.Equals(decoded1) {
		t.Errorf("AppendBinary failed for set1: original=%v, decoded=%v", set1.ToSlice(), decoded1.ToSlice())
	}

	if !set2.Equals(decoded2) {
		t.Errorf("AppendBinary failed for set2: original=%v, decoded=%v", set2.ToSlice(), decoded2.ToSlice())
	}
}

// TestIntSetMsgsize tests Msgsize for IntSet
func TestIntSetMsgsize(t *testing.T) {
	testCases := []struct {
		name string
		set  IntSet
	}{
		{
			name: "empty set",
			set:  NewIntSet(),
		},
		{
			name: "small set",
			set:  CreateIntSet(1, 2),
		},
		{
			name: "larger set",
			set:  CreateIntSet(1, 10, 100, 1000, 10000),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			size := tc.set.Msgsize()
			data, err := tc.set.MarshalMsg(nil)
			if err != nil {
				t.Fatalf("MarshalMsg() error = %v", err)
			}

			// Msgsize should be >= actual size
			if size < len(data) {
				t.Errorf("Msgsize() = %d, but actual size = %d", size, len(data))
			}
		})
	}
}

// TestMsgpSortedBehavior ensures that sorted variants maintain order
func TestMsgpSortedBehavior(t *testing.T) {
	t.Run("StringSet sorted", func(t *testing.T) {
		original := CreateStringSet("zebra", "apple", "banana", "cherry")
		data, err := original.MarshalMsg(nil)
		if err != nil {
			t.Fatalf("MarshalMsg() error = %v", err)
		}

		var decoded StringSet
		_, err = decoded.UnmarshalMsg(data)
		if err != nil {
			t.Fatalf("UnmarshalMsg() error = %v", err)
		}

		originalSlice := original.ToSlice()
		decodedSlice := decoded.ToSlice()

		// Both should be sorted
		expectedSlice := []string{"apple", "banana", "cherry", "zebra"}
		if !reflect.DeepEqual(originalSlice, expectedSlice) || !reflect.DeepEqual(decodedSlice, expectedSlice) {
			t.Errorf("Expected sorted slices to be %v, got original=%v, decoded=%v",
				expectedSlice, originalSlice, decodedSlice)
		}
	})

	t.Run("IntSet sorted", func(t *testing.T) {
		original := CreateIntSet(42, 1, 99, 7, 23)
		data, err := original.MarshalMsg(nil)
		if err != nil {
			t.Fatalf("MarshalMsg() error = %v", err)
		}

		var decoded IntSet
		_, err = decoded.UnmarshalMsg(data)
		if err != nil {
			t.Fatalf("UnmarshalMsg() error = %v", err)
		}

		originalSlice := original.ToSlice()
		decodedSlice := decoded.ToSlice()

		// Both should be sorted
		expectedSlice := []int{1, 7, 23, 42, 99}
		if !reflect.DeepEqual(originalSlice, expectedSlice) || !reflect.DeepEqual(decodedSlice, expectedSlice) {
			t.Errorf("Expected sorted slices to be %v, got original=%v, decoded=%v",
				expectedSlice, originalSlice, decodedSlice)
		}
	})
}
