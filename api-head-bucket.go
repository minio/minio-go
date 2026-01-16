/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2025 MinIO, Inc.
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
	"context"
	"net/http"

	"github.com/minio/minio-go/v7/pkg/s3utils"
)

// Bucket type constants
const (
	BucketTypeStandard  = "standard"
	BucketTypeWarehouse = "warehouse"

	bucketTypeHeader = "x-minio-bucket-type"
)

// HeadBucketInfo contains bucket metadata returned by HeadBucket.
type HeadBucketInfo struct {
	// Type indicates the bucket type: "standard" or "warehouse"
	Type string
}

// IsWarehouse returns true if bucket type is warehouse
func (b HeadBucketInfo) IsWarehouse() bool {
	return b.Type == BucketTypeWarehouse
}

// IsStandard returns true if bucket type is standard
func (b HeadBucketInfo) IsStandard() bool {
	return b.Type == BucketTypeStandard
}

// HeadBucket performs a HEAD request on the bucket and returns bucket metadata.
// This can be used to check if a bucket exists and retrieve bucket properties.
func (c *Client) HeadBucket(ctx context.Context, bucketName string) (HeadBucketInfo, error) {
	// Input validation.
	if err := s3utils.CheckValidBucketName(bucketName); err != nil {
		return HeadBucketInfo{}, err
	}

	// Execute HEAD on bucket.
	resp, err := c.executeMethod(ctx, http.MethodHead, requestMetadata{
		bucketName: bucketName,
	})
	defer closeResponse(resp)
	if err != nil {
		return HeadBucketInfo{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return HeadBucketInfo{}, httpRespToErrorResponse(resp, bucketName, "")
	}

	// Read bucket type from header, default to standard if not present
	bucketType := resp.Header.Get(bucketTypeHeader)
	if bucketType == "" {
		bucketType = BucketTypeStandard
	}

	return HeadBucketInfo{Type: bucketType}, nil
}
