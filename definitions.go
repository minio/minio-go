/*
 * Minimal object storage library (C) 2015 Minio, Inc.
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

package objectstorage

import "time"

// ListBuckets -
type ListBuckets struct {
	Buckets []*Bucket
	Owner   Owner
}

// Owner -
type Owner struct {
	DisplayName string
	ID          string
}

// Bucket -
type Bucket struct {
	// Date the bucket was created.
	CreationDate time.Time
	// The name of the bucket.
	Name string
}

// CommonPrefix -
type CommonPrefix struct {
	Prefix string
}

// Object -
type Object struct {
	ETag         string
	Key          string
	LastModified time.Time
	Owner        Owner
	Size         int64
	// The class of storage used to store the object.
	StorageClass string
}

// ListObjects -
type ListObjects struct {
	CommonPrefixes []*CommonPrefix
	Contents       []*Object
	Delimiter      string
	// Encoding type used to encode object keys in the response.
	EncodingType string
	// A flag that indicates whether or not ListObjects returned all of the results
	// that satisfied the search criteria.
	IsTruncated bool
	Marker      string
	MaxKeys     int64
	Name        string
	// When response is truncated (the IsTruncated element value in the response
	// is true), you can use the key name in this field as marker in the subsequent
	// request to get next set of objects. Minio lists objects in alphabetical
	// order Note: This element is returned only if you have delimiter request parameter
	// specified. If response does not include the NextMaker and it is truncated,
	// you can use the value of the last Key in the response as the marker in the
	// subsequent request to get the next set of object keys.
	NextMarker string
	Prefix     string
}
