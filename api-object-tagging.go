/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2020 MinIO, Inc.
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

package minio

import (
	"bytes"
	"context"
	"encoding/xml"
	"net/http"
	"net/url"

	"github.com/minio/minio-go/v7/pkg/s3utils"
	"github.com/minio/minio-go/v7/pkg/tags"
)

// PutObjectTaggingOptions holds an object version id
// to update tag(s) of a specific object version
type PutObjectTaggingOptions struct {
	VersionID string
	Internal  AdvancedObjectTaggingOptions
}

// AdvancedObjectTaggingOptions for internal use by MinIO server - not intended for client use.
type AdvancedObjectTaggingOptions struct {
	ReplicationProxyRequest string
}

// PutObjectTagging replaces or creates object tag(s) and can target a specific object version
// in a versioned bucket.
//
// Parameters:
//   - ctx: Context for request cancellation and timeout
//   - bucketName: Name of the bucket
//   - objectName: Name of the object
//   - otags: Tags to apply to the object
//   - opts: Options including VersionID to target a specific version
//
// Returns an error if the operation fails.
func (c *Client) PutObjectTagging(ctx context.Context, bucketName, objectName string, otags *tags.Tags, opts PutObjectTaggingOptions) error {
	// Input validation.
	if err := s3utils.CheckValidBucketName(bucketName); err != nil {
		return err
	}

	// Get resources properly escaped and lined up before
	// using them in http request.
	urlValues := make(url.Values)
	urlValues.Set("tagging", "")

	if opts.VersionID != "" {
		urlValues.Set("versionId", opts.VersionID)
	}
	headers := make(http.Header, 0)
	if opts.Internal.ReplicationProxyRequest != "" {
		headers.Set(minIOBucketReplicationProxyRequest, opts.Internal.ReplicationProxyRequest)
	}
	reqBytes, err := xml.Marshal(otags)
	if err != nil {
		return err
	}

	reqMetadata := requestMetadata{
		bucketName:       bucketName,
		objectName:       objectName,
		queryValues:      urlValues,
		contentBody:      bytes.NewReader(reqBytes),
		contentLength:    int64(len(reqBytes)),
		contentMD5Base64: sumMD5Base64(reqBytes),
		customHeader:     headers,
	}

	// Execute PUT to set a object tagging.
	resp, err := c.executeMethod(ctx, http.MethodPut, reqMetadata)
	defer closeResponse(resp)
	if err != nil {
		return err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return httpRespToErrorResponse(resp, bucketName, objectName)
		}
	}
	return nil
}

// PutObjectTaggingIfChanged adds or replaces object tag(s) only if they differ
// from the existing tags. This avoids unnecessary overwrites and minimizes
// redundant API calls.
//
// Parameters:
//   - ctx: Context for request cancellation and timeout
//   - bucketName: Name of the bucket
//   - objectName: Name of the object
//   - newTags: New tags to apply
//   - opts: Options including VersionID to target a specific object version
//
// Returns an error if the operation fails.
func (c *Client) PutObjectTaggingIfChanged(ctx context.Context, bucketName, objectName string, newTags *tags.Tags, opts PutObjectTaggingOptions) error {
	// Validate bucket name
	if err := s3utils.CheckValidBucketName(bucketName); err != nil {
		return err
	}

	// Attempt to get current tags
	currentTags, err := c.GetObjectTagging(ctx, bucketName, objectName, GetObjectTaggingOptions{VersionID: opts.VersionID})
	if err != nil {
		// If no existing tags or 404, continue with PUT
		// Other errors (network, auth, etc.) should still fail
		if respErr, ok := err.(ErrorResponse); ok {
			if respErr.StatusCode != http.StatusNotFound {
				return err
			}
		} else {
			return err
		}
	}

	// Compare if we have current tags
	if currentTags != nil && tagMapsEqual(currentTags.ToMap(), newTags.ToMap()) {
		// No difference, skip update
		return nil
	}

	// Perform tagging update
	return c.PutObjectTagging(ctx, bucketName, objectName, newTags, opts)
}

// tagMapsEqual compares two tag maps for equality.
func tagMapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// GetObjectTaggingOptions holds the object version ID
// to fetch the tagging key/value pairs
type GetObjectTaggingOptions struct {
	VersionID string
	Internal  AdvancedObjectTaggingOptions
}

// GetObjectTagging retrieves object tag(s) with options to target a specific object version
// in a versioned bucket.
//
// Parameters:
//   - ctx: Context for request cancellation and timeout
//   - bucketName: Name of the bucket
//   - objectName: Name of the object
//   - opts: Options including VersionID to target a specific version
//
// Returns the object's tags or an error if the operation fails.
func (c *Client) GetObjectTagging(ctx context.Context, bucketName, objectName string, opts GetObjectTaggingOptions) (*tags.Tags, error) {
	// Get resources properly escaped and lined up before
	// using them in http request.
	urlValues := make(url.Values)
	urlValues.Set("tagging", "")

	if opts.VersionID != "" {
		urlValues.Set("versionId", opts.VersionID)
	}
	headers := make(http.Header, 0)
	if opts.Internal.ReplicationProxyRequest != "" {
		headers.Set(minIOBucketReplicationProxyRequest, opts.Internal.ReplicationProxyRequest)
	}
	// Execute GET on object to get object tag(s)
	resp, err := c.executeMethod(ctx, http.MethodGet, requestMetadata{
		bucketName:   bucketName,
		objectName:   objectName,
		queryValues:  urlValues,
		customHeader: headers,
	})

	defer closeResponse(resp)
	if err != nil {
		return nil, err
	}

	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, httpRespToErrorResponse(resp, bucketName, objectName)
		}
	}

	return tags.ParseObjectXML(resp.Body)
}

// RemoveObjectTaggingOptions holds the version id of the object to remove
type RemoveObjectTaggingOptions struct {
	VersionID string
	Internal  AdvancedObjectTaggingOptions
}

// RemoveObjectTagging removes object tag(s) with options to target a specific object version
// in a versioned bucket.
//
// Parameters:
//   - ctx: Context for request cancellation and timeout
//   - bucketName: Name of the bucket
//   - objectName: Name of the object
//   - opts: Options including VersionID to target a specific version
//
// Returns an error if the operation fails.
func (c *Client) RemoveObjectTagging(ctx context.Context, bucketName, objectName string, opts RemoveObjectTaggingOptions) error {
	// Get resources properly escaped and lined up before
	// using them in http request.
	urlValues := make(url.Values)
	urlValues.Set("tagging", "")

	if opts.VersionID != "" {
		urlValues.Set("versionId", opts.VersionID)
	}
	headers := make(http.Header, 0)
	if opts.Internal.ReplicationProxyRequest != "" {
		headers.Set(minIOBucketReplicationProxyRequest, opts.Internal.ReplicationProxyRequest)
	}
	// Execute DELETE on object to remove object tag(s)
	resp, err := c.executeMethod(ctx, http.MethodDelete, requestMetadata{
		bucketName:   bucketName,
		objectName:   objectName,
		queryValues:  urlValues,
		customHeader: headers,
	})

	defer closeResponse(resp)
	if err != nil {
		return err
	}

	if resp != nil {
		// S3 returns "204 No content" after Object tag deletion.
		if resp.StatusCode != http.StatusNoContent {
			return httpRespToErrorResponse(resp, bucketName, objectName)
		}
	}
	return err
}
