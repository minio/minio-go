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

// ListAllMyBucketsResult container for ListBucets response
type ListAllMyBucketsResult struct {
	// Container for one or more buckets.
	Buckets struct {
		Bucket []*Bucket
	}
	Owner Owner
}

// Bucket container for bucket metadata
type Bucket struct {
	// The name of the bucket.
	Name string
	// Date the bucket was created.
	CreationDate time.Time
}

// Owner container for bucket owner information
type Owner struct {
	DisplayName string
	ID          string
}

// CommonPrefix container for prefix response in ListObjects
type CommonPrefix struct {
	Prefix string
}

// Object container for object metadata
type Object struct {
	ETag         string
	Key          string
	LastModified time.Time
	Size         int64

	Owner Owner

	// The class of storage used to store the object.
	StorageClass string
}

// ListBucketResult contaner for ListObjects response
type ListBucketResult struct {
	CommonPrefixes []*CommonPrefix // A response can contain CommonPrefixes only if you specify a delimiter
	Contents       []*Object       // Metadata about each object returned
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
	// request to get next set of objects. Object storage lists objects in alphabetical
	// order Note: This element is returned only if you have delimiter request parameter
	// specified. If response does not include the NextMaker and it is truncated,
	// you can use the value of the last Key in the response as the marker in the
	// subsequent request to get the next set of object keys.
	NextMarker string
	Prefix     string
}
