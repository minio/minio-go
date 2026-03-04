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
	"cmp"
	"reflect"
	"testing"
)

// Test New() creates an empty set
func TestNew(t *testing.T) {
	s := New[int]()
	if !s.IsEmpty() {
		t.Fatalf("expected empty set, got: %v", s)
	}
}

// Test Create() with initial values
func TestCreate(t *testing.T) {
	s := Create(1, 2, 3)
	if len(s) != 3 {
		t.Fatalf("expected length 3, got: %d", len(s))
	}
	if !s.Contains(1) || !s.Contains(2) || !s.Contains(3) {
		t.Fatalf("expected set to contain 1, 2, 3")
	}
}

// Test Add() and Contains()
func TestIntSetAdd(t *testing.T) {
	s := New[int]()
	s.Add(42)
	if !s.Contains(42) {
		t.Fatalf("expected set to contain 42")
	}
	if s.Contains(99) {
		t.Fatalf("expected set not to contain 99")
	}
}

// Test Remove()
func TestIntSetRemove(t *testing.T) {
	s := Create(1, 2, 3)
	s.Remove(2)
	if s.Contains(2) {
		t.Fatalf("expected set not to contain 2 after removal")
	}
	if !s.Contains(1) || !s.Contains(3) {
		t.Fatalf("expected set to still contain 1 and 3")
	}
}

// Test Equals()
func TestIntSetEquals(t *testing.T) {
	s1 := Create(1, 2, 3)
	s2 := Create(3, 2, 1)
	s3 := Create(1, 2)

	if !s1.Equals(s2) {
		t.Fatalf("expected s1 to equal s2")
	}
	if s1.Equals(s3) {
		t.Fatalf("expected s1 not to equal s3")
	}
}

// Test Intersection()
func TestIntSetIntersection(t *testing.T) {
	s1 := Create(1, 2, 3, 4)
	s2 := Create(3, 4, 5, 6)
	result := s1.Intersection(s2)

	expected := Create(3, 4)
	if !result.Equals(expected) {
		t.Fatalf("expected intersection {3, 4}, got: %v", result)
	}
}

// Test Difference()
func TestIntSetDifference(t *testing.T) {
	s1 := Create(1, 2, 3, 4)
	s2 := Create(3, 4, 5, 6)
	result := s1.Difference(s2)

	expected := Create(1, 2)
	if !result.Equals(expected) {
		t.Fatalf("expected difference {1, 2}, got: %v", result)
	}
}

// Test Union()
func TestIntSetUnion(t *testing.T) {
	s1 := Create(1, 2, 3)
	s2 := Create(3, 4, 5)
	result := s1.Union(s2)

	expected := Create(1, 2, 3, 4, 5)
	if !result.Equals(expected) {
		t.Fatalf("expected union {1, 2, 3, 4, 5}, got: %v", result)
	}
}

// Test Copy()
func TestIntSetCopy(t *testing.T) {
	s1 := Create(1, 2, 3)
	s2 := Copy(s1)

	if !s1.Equals(s2) {
		t.Fatalf("expected copy to equal original")
	}

	// Modify copy and ensure original is unchanged
	s2.Add(4)
	if s1.Contains(4) {
		t.Fatalf("expected original set not to be modified")
	}
}

// Test ToSliceOrdered() with integers
func TestToSliceOrdered(t *testing.T) {
	s := Create(3, 1, 4, 1, 5, 9, 2, 6)
	slice := ToSliceOrdered(s)

	expected := []int{1, 2, 3, 4, 5, 6, 9}
	if !reflect.DeepEqual(slice, expected) {
		t.Fatalf("expected sorted slice %v, got: %v", expected, slice)
	}
}

// Test ToSlice() with custom comparison function
func TestToSliceWithCustomCompare(t *testing.T) {
	s := Create(3, 1, 4, 1, 5, 9, 2, 6)

	// Reverse sort
	slice := s.ToSlice(func(a, b int) int {
		return cmp.Compare(b, a)
	})

	expected := []int{9, 6, 5, 4, 3, 2, 1}
	if !reflect.DeepEqual(slice, expected) {
		t.Fatalf("expected reverse sorted slice %v, got: %v", expected, slice)
	}
}

// Test FuncMatch()
func TestIntSetFuncMatch(t *testing.T) {
	s := Create(1, 2, 3, 4, 5, 6)

	// Find all even numbers
	result := s.FuncMatch(func(val, _ int) bool {
		return val%2 == 0
	}, 0)

	expected := Create(2, 4, 6)
	if !result.Equals(expected) {
		t.Fatalf("expected even numbers {2, 4, 6}, got: %v", result)
	}
}

// Test ApplyFunc()
func TestIntSetApplyFunc(t *testing.T) {
	s := Create(1, 2, 3)

	// Double each value
	result := s.ApplyFunc(func(val int) int {
		return val * 2
	})

	expected := Create(2, 4, 6)
	if !result.Equals(expected) {
		t.Fatalf("expected doubled values {2, 4, 6}, got: %v", result)
	}
}

// Test with different comparable types
func TestFloat64Set(t *testing.T) {
	s := Create(1.5, 2.7, 3.14)
	if !s.Contains(3.14) {
		t.Fatalf("expected set to contain 3.14")
	}
}

func TestBoolSet(t *testing.T) {
	s := Create(true, false)
	if len(s) != 2 {
		t.Fatalf("expected length 2, got: %d", len(s))
	}
}

// Custom comparable type
type Point struct {
	X, Y int
}

func TestCustomTypeSet(t *testing.T) {
	p1 := Point{1, 2}
	p2 := Point{3, 4}
	p3 := Point{1, 2} // same as p1

	s := Create(p1, p2, p3)
	// p1 and p3 are the same, so only 2 unique points
	if len(s) != 2 {
		t.Fatalf("expected length 2, got: %d", len(s))
	}

	if !s.Contains(p1) || !s.Contains(p2) {
		t.Fatalf("expected set to contain both points")
	}
}
