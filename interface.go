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

import "io"

// API - minimalist object storage API interface
type API interface {
	/// Bucket Write Operations
	PutBucket(bucket string) error
	PutBucketACL(bucket, acl string) error

	/// Bucket Read Operations
	ListObjects(bucket string) (ListObjects, error)
	HeadBucket(bucket string) error

	/// Object Read/Write/Stat Operations
	PutObject(bucket, object string, size int64, body io.ReadSeeker) error
	GetObject(bucket, object string, offset, length uint64) (io.ReadCloser, error)
	HeadObject(bucket, object string) error

	/// Service Operations
	ListBuckets() (ListBuckets, error)
}
