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

	// Header name for bucket type
	bucketTypeHeader = "x-minio-bucket-type"
)

// BucketTypeInfo contains bucket type information
type BucketTypeInfo struct {
	Type string
}

// IsWarehouse returns true if bucket type is warehouse
func (b BucketTypeInfo) IsWarehouse() bool {
	return b.Type == BucketTypeWarehouse
}

// IsStandard returns true if bucket type is standard
func (b BucketTypeInfo) IsStandard() bool {
	return b.Type == BucketTypeStandard
}

// GetBucketType gets the bucket type by performing a HEAD request
// and reading the x-minio-bucket-type header.
// This is a MinIO extension API.
func (c *Client) GetBucketType(ctx context.Context, bucketName string) (BucketTypeInfo, error) {
	// Input validation.
	if err := s3utils.CheckValidBucketName(bucketName); err != nil {
		return BucketTypeInfo{}, err
	}

	// Execute HEAD on bucket to get the type from header.
	resp, err := c.executeMethod(ctx, http.MethodHead, requestMetadata{
		bucketName: bucketName,
	})

	defer closeResponse(resp)
	if err != nil {
		return BucketTypeInfo{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return BucketTypeInfo{}, httpRespToErrorResponse(resp, bucketName, "")
	}

	// Read bucket type from header, default to standard if not present
	bucketType := resp.Header.Get(bucketTypeHeader)
	if bucketType == "" {
		bucketType = BucketTypeStandard
	}

	return BucketTypeInfo{Type: bucketType}, nil
}
