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

package set_test

import (
	"fmt"

	"github.com/minio/minio-go/v7/pkg/set"
)

// Example demonstrates basic usage of generic Set with integers
func Example() {
	// Create a new set of integers
	s := set.Create(1, 2, 3, 4, 5)

	// Add an element
	s.Add(6)

	// Check if element exists
	fmt.Println("Contains 3:", s.Contains(3))
	fmt.Println("Contains 10:", s.Contains(10))

	// Remove an element
	s.Remove(2)

	// Get sorted slice
	fmt.Println("Elements:", set.ToSliceOrdered(s))

	// Output:
	// Contains 3: true
	// Contains 10: false
	// Elements: [1 3 4 5 6]
}

// ExampleSet_intSet demonstrates set operations with integers
func ExampleSet_intSet() {
	setA := set.Create(1, 2, 3, 4)
	setB := set.Create(3, 4, 5, 6)

	// Union
	union := setA.Union(setB)
	fmt.Println("Union:", set.ToSliceOrdered(union))

	// Intersection
	intersection := setA.Intersection(setB)
	fmt.Println("Intersection:", set.ToSliceOrdered(intersection))

	// Difference
	difference := setA.Difference(setB)
	fmt.Println("Difference:", set.ToSliceOrdered(difference))

	// Output:
	// Union: [1 2 3 4 5 6]
	// Intersection: [3 4]
	// Difference: [1 2]
}

// ExampleSet_stringSet demonstrates using generic Set with strings
func ExampleSet_stringSet() {
	fruits := set.Create("apple", "banana", "cherry")

	fruits.Add("date")
	fruits.Add("banana") // duplicate, won't be added

	fmt.Println("Has apple:", fruits.Contains("apple"))
	fmt.Println("Count:", len(fruits))
	fmt.Println("Fruits:", set.ToSliceOrdered(fruits))

	// Output:
	// Has apple: true
	// Count: 4
	// Fruits: [apple banana cherry date]
}

// ExampleSet_float64Set demonstrates using Set with floating point numbers
func ExampleSet_float64Set() {
	temps := set.Create(98.6, 100.4, 99.1, 98.6)

	fmt.Println("Unique temperatures:", len(temps))
	fmt.Println("Has 100.4:", temps.Contains(100.4))
	fmt.Println("Temperatures:", set.ToSliceOrdered(temps))

	// Output:
	// Unique temperatures: 3
	// Has 100.4: true
	// Temperatures: [98.6 99.1 100.4]
}

// ExampleSet_ApplyFunc demonstrates transforming set elements
func ExampleSet_ApplyFunc() {
	numbers := set.Create(1, 2, 3, 4, 5)

	// Square each number
	squared := numbers.ApplyFunc(func(n int) int {
		return n * n
	})

	fmt.Println("Original:", set.ToSliceOrdered(numbers))
	fmt.Println("Squared:", set.ToSliceOrdered(squared))

	// Output:
	// Original: [1 2 3 4 5]
	// Squared: [1 4 9 16 25]
}

// ExampleSet_FuncMatch demonstrates filtering set elements
func ExampleSet_FuncMatch() {
	numbers := set.Create(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)

	// Find all even numbers
	evens := numbers.FuncMatch(func(n, _ int) bool {
		return n%2 == 0
	}, 0)

	// Find all numbers greater than 5
	greaterThanFive := numbers.FuncMatch(func(n, threshold int) bool {
		return n > threshold
	}, 5)

	fmt.Println("Even numbers:", set.ToSliceOrdered(evens))
	fmt.Println("Greater than 5:", set.ToSliceOrdered(greaterThanFive))

	// Output:
	// Even numbers: [2 4 6 8 10]
	// Greater than 5: [6 7 8 9 10]
}

// ExampleCopy demonstrates copying a set
func ExampleCopy() {
	original := set.Create(1, 2, 3)
	copied := set.Copy(original)

	// Modify the copy
	copied.Add(4)
	copied.Remove(1)

	fmt.Println("Original:", set.ToSliceOrdered(original))
	fmt.Println("Modified copy:", set.ToSliceOrdered(copied))

	// Output:
	// Original: [1 2 3]
	// Modified copy: [2 3 4]
}

// ExampleSet_Equals demonstrates checking set equality
func ExampleSet_Equals() {
	set1 := set.Create(1, 2, 3)
	set2 := set.Create(3, 2, 1) // same elements, different order
	set3 := set.Create(1, 2, 3, 4)

	fmt.Println("set1 equals set2:", set1.Equals(set2))
	fmt.Println("set1 equals set3:", set1.Equals(set3))

	// Output:
	// set1 equals set2: true
	// set1 equals set3: false
}

// ExampleSet_customType demonstrates using Set with custom types
func ExampleSet_customType() {
	type UserID int

	activeUsers := set.Create(UserID(101), UserID(102), UserID(103))
	premiumUsers := set.Create(UserID(102), UserID(103), UserID(104))

	// Find users that are both active and premium
	activePremium := activeUsers.Intersection(premiumUsers)
	fmt.Println("Active premium users:", set.ToSliceOrdered(activePremium))

	// Find active users that are not premium
	freeUsers := activeUsers.Difference(premiumUsers)
	fmt.Println("Free users:", set.ToSliceOrdered(freeUsers))

	// Output:
	// Active premium users: [102 103]
	// Free users: [101]
}

// ExampleToSliceOrdered demonstrates sorting different ordered types
func ExampleToSliceOrdered() {
	// Works with any ordered type (int, float, string, etc.)
	ints := set.Create(5, 2, 8, 1, 9)
	fmt.Println("Sorted ints:", set.ToSliceOrdered(ints))

	strings := set.Create("zebra", "apple", "mango", "banana")
	fmt.Println("Sorted strings:", set.ToSliceOrdered(strings))

	floats := set.Create(3.14, 2.71, 1.41, 1.73)
	fmt.Println("Sorted floats:", set.ToSliceOrdered(floats))

	// Output:
	// Sorted ints: [1 2 5 8 9]
	// Sorted strings: [apple banana mango zebra]
	// Sorted floats: [1.41 1.73 2.71 3.14]
}

// ExampleSet_ToSlice demonstrates custom sorting with comparison function
func ExampleSet_ToSlice() {
	words := set.Create("go", "rust", "python", "javascript")

	// Sort by length (shortest first)
	byLength := words.ToSlice(func(a, b string) int {
		return len(a) - len(b)
	})

	fmt.Println("By length:", byLength)

	// Output:
	// By length: [go rust python javascript]
}

// ExampleNew demonstrates creating an empty set and adding elements
func ExampleNew() {
	// Create an empty set of strings
	tags := set.New[string]()

	// Add elements one by one
	tags.Add("go")
	tags.Add("generics")
	tags.Add("set")
	tags.Add("go") // duplicate, ignored

	fmt.Println("Tags:", set.ToSliceOrdered(tags))
	fmt.Println("Count:", len(tags))

	// Output:
	// Tags: [generics go set]
	// Count: 3
}

// ExampleCreate demonstrates creating a set with initial values
func ExampleCreate() {
	// Create set with initial values
	primes := set.Create(2, 3, 5, 7, 11, 13)

	fmt.Println("Is 7 prime:", primes.Contains(7))
	fmt.Println("Is 9 prime:", primes.Contains(9))
	fmt.Println("Prime count:", len(primes))

	// Output:
	// Is 7 prime: true
	// Is 9 prime: false
	// Prime count: 6
}

// ExampleSet_complexCustomType demonstrates using Set with a complex comparable struct
func ExampleSet_complexCustomType() {
	// Define a complex type representing a network connection
	type Connection struct {
		Host     string
		Port     int
		Protocol string
		Secure   bool
	}

	// Track unique connections
	connections := set.New[Connection]()

	// Add connections
	connections.Add(Connection{"api.example.com", 443, "https", true})
	connections.Add(Connection{"db.example.com", 5432, "postgres", true})
	connections.Add(Connection{"api.example.com", 443, "https", true}) // duplicate
	connections.Add(Connection{"cache.example.com", 6379, "redis", false})

	fmt.Println("Unique connections:", len(connections))

	// Check if specific connection exists
	apiConn := Connection{"api.example.com", 443, "https", true}
	fmt.Println("Has API connection:", connections.Contains(apiConn))

	// Filter secure connections
	secureConns := connections.FuncMatch(func(conn, _ Connection) bool {
		return conn.Secure
	}, Connection{})

	fmt.Println("Secure connections:", len(secureConns))

	// Output:
	// Unique connections: 3
	// Has API connection: true
	// Secure connections: 2
}

// ExampleSet_structWithMultipleFields demonstrates deduplication with complex structs
func ExampleSet_structWithMultipleFields() {
	// Define a struct representing an S3 object key with metadata
	type S3Object struct {
		Bucket  string
		Key     string
		Version string
		ETag    string
	}

	objects := set.New[S3Object]()

	// Add objects
	objects.Add(S3Object{"my-bucket", "docs/file1.txt", "v1", "abc123"})
	objects.Add(S3Object{"my-bucket", "docs/file2.txt", "v1", "def456"})
	objects.Add(S3Object{"my-bucket", "docs/file1.txt", "v2", "ghi789"})
	objects.Add(S3Object{"my-bucket", "docs/file1.txt", "v1", "abc123"}) // duplicate

	fmt.Println("Unique objects:", len(objects))

	// Check for specific object
	obj := S3Object{"my-bucket", "docs/file1.txt", "v1", "abc123"}
	fmt.Println("Contains object:", objects.Contains(obj))

	// Get all objects from a specific bucket
	bucketObjects := objects.FuncMatch(func(o, filter S3Object) bool {
		return o.Bucket == filter.Bucket && o.Key == filter.Key
	}, S3Object{Bucket: "my-bucket", Key: "docs/file1.txt"})

	fmt.Println("Versions of file1.txt:", len(bucketObjects))

	// Output:
	// Unique objects: 3
	// Contains object: true
	// Versions of file1.txt: 2
}

// ExampleSet_nestedComparable demonstrates using Set with nested comparable structs
func ExampleSet_nestedComparable() {
	// Define nested comparable types
	type Coordinate struct {
		Lat, Lon float64
	}

	type Location struct {
		Name   string
		Coords Coordinate
	}

	places := set.New[Location]()

	places.Add(Location{"Eiffel Tower", Coordinate{48.8584, 2.2945}})
	places.Add(Location{"Statue of Liberty", Coordinate{40.6892, -74.0445}})
	places.Add(Location{"Eiffel Tower", Coordinate{48.8584, 2.2945}}) // duplicate

	fmt.Println("Unique places:", len(places))

	// Check if location exists
	eiffel := Location{"Eiffel Tower", Coordinate{48.8584, 2.2945}}
	fmt.Println("Has Eiffel Tower:", places.Contains(eiffel))

	// Output:
	// Unique places: 2
	// Has Eiffel Tower: true
}
