/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2015-2020 MinIO, Inc.
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
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/minio/minio-go/v7/pkg/s3utils"
)

// ApplyServerSideEncryptionByDefault defines default encryption configuration, KMS or SSE. To activate
// KMS, SSEAlgoritm needs to be set to "aws:kms"
// Minio currently does not support Kms.
type ApplyServerSideEncryptionByDefault struct {
	KmsMasterKeyID string `xml:"KMSMasterKeyID,omitempty"`
	SSEAlgorithm   string `xml:"SSEAlgorithm"`
}

// Rule layer encapsulates default encryption configuration
type Rule struct {
	Apply ApplyServerSideEncryptionByDefault `xml:"ApplyServerSideEncryptionByDefault"`
}

// ServerSideEncryptionConfiguration is the default encryption configuration structure
type ServerSideEncryptionConfiguration struct {
	XMLName xml.Name `xml:"ServerSideEncryptionConfiguration"`
	Rules   []Rule   `xml:"Rule"`
}

/// Bucket operations

func (c Client) makeBucket(ctx context.Context, bucketName string, opts MakeBucketOptions) (err error) {
	// Validate the input arguments.
	if err := s3utils.CheckValidBucketNameStrict(bucketName); err != nil {
		return err
	}

	err = c.doMakeBucket(ctx, bucketName, opts.Region, opts.ObjectLocking)
	if err != nil && (opts.Region == "" || opts.Region == "us-east-1") {
		if resp, ok := err.(ErrorResponse); ok && resp.Code == "AuthorizationHeaderMalformed" && resp.Region != "" {
			err = c.doMakeBucket(ctx, bucketName, resp.Region, opts.ObjectLocking)
		}
	}
	return err
}

func (c Client) doMakeBucket(ctx context.Context, bucketName string, location string, objectLockEnabled bool) (err error) {
	defer func() {
		// Save the location into cache on a successful makeBucket response.
		if err == nil {
			c.bucketLocCache.Set(bucketName, location)
		}
	}()

	// If location is empty, treat is a default region 'us-east-1'.
	if location == "" {
		location = "us-east-1"
		// For custom region clients, default
		// to custom region instead not 'us-east-1'.
		if c.region != "" {
			location = c.region
		}
	}
	// PUT bucket request metadata.
	reqMetadata := requestMetadata{
		bucketName:     bucketName,
		bucketLocation: location,
	}

	if objectLockEnabled {
		headers := make(http.Header)
		headers.Add("x-amz-bucket-object-lock-enabled", "true")
		reqMetadata.customHeader = headers
	}

	// If location is not 'us-east-1' create bucket location config.
	if location != "us-east-1" && location != "" {
		createBucketConfig := createBucketConfiguration{}
		createBucketConfig.Location = location
		var createBucketConfigBytes []byte
		createBucketConfigBytes, err = xml.Marshal(createBucketConfig)
		if err != nil {
			return err
		}
		reqMetadata.contentMD5Base64 = sumMD5Base64(createBucketConfigBytes)
		reqMetadata.contentSHA256Hex = sum256Hex(createBucketConfigBytes)
		reqMetadata.contentBody = bytes.NewReader(createBucketConfigBytes)
		reqMetadata.contentLength = int64(len(createBucketConfigBytes))
	}

	// Execute PUT to create a new bucket.
	resp, err := c.executeMethod(ctx, "PUT", reqMetadata)
	defer closeResponse(resp)
	if err != nil {
		return err
	}

	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return httpRespToErrorResponse(resp, bucketName, "")
		}
	}

	// Success.
	return nil
}

// MakeBucketOptions holds all options to tweak bucket creation
type MakeBucketOptions struct {
	// Bucket location
	Region string
	// Enable object locking
	ObjectLocking bool
}

// MakeBucket creates a new bucket with bucketName with a context to control cancellations and timeouts.
//
// Location is an optional argument, by default all buckets are
// created in US Standard Region.
//
// For Amazon S3 for more supported regions - http://docs.aws.amazon.com/general/latest/gr/rande.html
// For Google Cloud Storage for more supported regions - https://cloud.google.com/storage/docs/bucket-locations
func (c Client) MakeBucket(ctx context.Context, bucketName string, opts MakeBucketOptions) (err error) {
	return c.makeBucket(ctx, bucketName, opts)
}

// SetBucketPolicy sets the access permissions on an existing bucket.
func (c Client) SetBucketPolicy(ctx context.Context, bucketName, policy string) error {
	// Input validation.
	if err := s3utils.CheckValidBucketName(bucketName); err != nil {
		return err
	}

	// If policy is empty then delete the bucket policy.
	if policy == "" {
		return c.removeBucketPolicy(ctx, bucketName)
	}

	// Save the updated policies.
	return c.putBucketPolicy(ctx, bucketName, policy)
}

// Saves a new bucket policy.
func (c Client) putBucketPolicy(ctx context.Context, bucketName, policy string) error {
	// Input validation.
	if err := s3utils.CheckValidBucketName(bucketName); err != nil {
		return err
	}

	// Get resources properly escaped and lined up before
	// using them in http request.
	urlValues := make(url.Values)
	urlValues.Set("policy", "")

	// Content-length is mandatory for put policy request
	policyReader := strings.NewReader(policy)
	b, err := ioutil.ReadAll(policyReader)
	if err != nil {
		return err
	}

	reqMetadata := requestMetadata{
		bucketName:    bucketName,
		queryValues:   urlValues,
		contentBody:   policyReader,
		contentLength: int64(len(b)),
	}

	// Execute PUT to upload a new bucket policy.
	resp, err := c.executeMethod(ctx, "PUT", reqMetadata)
	defer closeResponse(resp)
	if err != nil {
		return err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusNoContent {
			return httpRespToErrorResponse(resp, bucketName, "")
		}
	}
	return nil
}

// Removes all policies on a bucket.
func (c Client) removeBucketPolicy(ctx context.Context, bucketName string) error {
	// Input validation.
	if err := s3utils.CheckValidBucketName(bucketName); err != nil {
		return err
	}
	// Get resources properly escaped and lined up before
	// using them in http request.
	urlValues := make(url.Values)
	urlValues.Set("policy", "")

	// Execute DELETE on objectName.
	resp, err := c.executeMethod(ctx, "DELETE", requestMetadata{
		bucketName:       bucketName,
		queryValues:      urlValues,
		contentSHA256Hex: emptySHA256Hex,
	})
	defer closeResponse(resp)
	if err != nil {
		return err
	}
	return nil
}

// SetBucketLifecycle set the lifecycle on an existing bucket.
func (c Client) SetBucketLifecycle(ctx context.Context, bucketName, lifecycle string) error {
	// Input validation.
	if err := s3utils.CheckValidBucketName(bucketName); err != nil {
		return err
	}

	// If lifecycle is empty then delete it.
	if lifecycle == "" {
		return c.removeBucketLifecycle(ctx, bucketName)
	}

	// Save the updated lifecycle.
	return c.putBucketLifecycle(ctx, bucketName, lifecycle)
}

// Saves a new bucket lifecycle.
func (c Client) putBucketLifecycle(ctx context.Context, bucketName, lifecycle string) error {
	// Input validation.
	if err := s3utils.CheckValidBucketName(bucketName); err != nil {
		return err
	}

	// Get resources properly escaped and lined up before
	// using them in http request.
	urlValues := make(url.Values)
	urlValues.Set("lifecycle", "")

	// Content-length is mandatory for put lifecycle request
	lifecycleReader := strings.NewReader(lifecycle)
	b, err := ioutil.ReadAll(lifecycleReader)
	if err != nil {
		return err
	}

	reqMetadata := requestMetadata{
		bucketName:       bucketName,
		queryValues:      urlValues,
		contentBody:      lifecycleReader,
		contentLength:    int64(len(b)),
		contentMD5Base64: sumMD5Base64(b),
	}

	// Execute PUT to upload a new bucket lifecycle.
	resp, err := c.executeMethod(ctx, "PUT", reqMetadata)
	defer closeResponse(resp)
	if err != nil {
		return err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return httpRespToErrorResponse(resp, bucketName, "")
		}
	}
	return nil
}

// Remove lifecycle from a bucket.
func (c Client) removeBucketLifecycle(ctx context.Context, bucketName string) error {
	// Input validation.
	if err := s3utils.CheckValidBucketName(bucketName); err != nil {
		return err
	}
	// Get resources properly escaped and lined up before
	// using them in http request.
	urlValues := make(url.Values)
	urlValues.Set("lifecycle", "")

	// Execute DELETE on objectName.
	resp, err := c.executeMethod(ctx, "DELETE", requestMetadata{
		bucketName:       bucketName,
		queryValues:      urlValues,
		contentSHA256Hex: emptySHA256Hex,
	})
	defer closeResponse(resp)
	if err != nil {
		return err
	}
	return nil
}

// SetBucketEncryption sets the default encryption configuration on an existing bucket.
func (c Client) SetBucketEncryption(ctx context.Context, bucketName string, configuration ServerSideEncryptionConfiguration) error {
	// Input validation.
	if err := s3utils.CheckValidBucketName(bucketName); err != nil {
		return err
	}

	buf, err := xml.Marshal(&configuration)
	if err != nil {
		return err
	}

	// Get resources properly escaped and lined up before
	// using them in http request.
	urlValues := make(url.Values)
	urlValues.Set("encryption", "")

	// Content-length is mandatory to set a default encryption configuration
	reqMetadata := requestMetadata{
		bucketName:       bucketName,
		queryValues:      urlValues,
		contentBody:      bytes.NewReader(buf),
		contentLength:    int64(len(buf)),
		contentMD5Base64: sumMD5Base64(buf),
	}

	// Execute PUT to upload a new bucket default encryption configuration.
	resp, err := c.executeMethod(ctx, http.MethodPut, reqMetadata)
	defer closeResponse(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return httpRespToErrorResponse(resp, bucketName, "")
	}
	return nil
}

// DeleteBucketEncryption removes the default encryption configuration on a bucket with a context to control cancellations and timeouts.
func (c Client) DeleteBucketEncryption(ctx context.Context, bucketName string) error {
	// Input validation.
	if err := s3utils.CheckValidBucketName(bucketName); err != nil {
		return err
	}

	// Get resources properly escaped and lined up before
	// using them in http request.
	urlValues := make(url.Values)
	urlValues.Set("encryption", "")

	// DELETE default encryption configuration on a bucket.
	resp, err := c.executeMethod(ctx, http.MethodDelete, requestMetadata{
		bucketName:       bucketName,
		queryValues:      urlValues,
		contentSHA256Hex: emptySHA256Hex,
	})
	defer closeResponse(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return httpRespToErrorResponse(resp, bucketName, "")
	}
	return nil
}

// SetBucketNotification saves a new bucket notification with a context to control cancellations
// and timeouts.
func (c Client) SetBucketNotification(ctx context.Context, bucketName string, bucketNotification BucketNotification) error {
	// Input validation.
	if err := s3utils.CheckValidBucketName(bucketName); err != nil {
		return err
	}

	// Get resources properly escaped and lined up before
	// using them in http request.
	urlValues := make(url.Values)
	urlValues.Set("notification", "")

	notifBytes, err := xml.Marshal(bucketNotification)
	if err != nil {
		return err
	}

	notifBuffer := bytes.NewReader(notifBytes)
	reqMetadata := requestMetadata{
		bucketName:       bucketName,
		queryValues:      urlValues,
		contentBody:      notifBuffer,
		contentLength:    int64(len(notifBytes)),
		contentMD5Base64: sumMD5Base64(notifBytes),
		contentSHA256Hex: sum256Hex(notifBytes),
	}

	// Execute PUT to upload a new bucket notification.
	resp, err := c.executeMethod(ctx, "PUT", reqMetadata)
	defer closeResponse(resp)
	if err != nil {
		return err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return httpRespToErrorResponse(resp, bucketName, "")
		}
	}
	return nil
}

// RemoveAllBucketNotification - Remove bucket notification clears all previously specified config
func (c Client) RemoveAllBucketNotification(ctx context.Context, bucketName string) error {
	return c.SetBucketNotification(ctx, bucketName, BucketNotification{})
}

var (
	versionEnableConfig       = []byte("<VersioningConfiguration xmlns=\"http://s3.amazonaws.com/doc/2006-03-01/\"><Status>Enabled</Status></VersioningConfiguration>")
	versionEnableConfigLen    = int64(len(versionEnableConfig))
	versionEnableConfigMD5Sum = sumMD5Base64(versionEnableConfig)
	versionEnableConfigSHA256 = sum256Hex(versionEnableConfig)

	versionDisableConfig       = []byte("<VersioningConfiguration xmlns=\"http://s3.amazonaws.com/doc/2006-03-01/\"><Status>Suspended</Status></VersioningConfiguration>")
	versionDisableConfigLen    = int64(len(versionDisableConfig))
	versionDisableConfigMD5Sum = sumMD5Base64(versionDisableConfig)
	versionDisableConfigSHA256 = sum256Hex(versionDisableConfig)
)

func (c Client) setVersioning(ctx context.Context, bucketName string, config []byte, length int64, md5sum, sha256sum string) error {
	// Input validation.
	if err := s3utils.CheckValidBucketName(bucketName); err != nil {
		return err
	}

	// Get resources properly escaped and lined up before
	// using them in http request.
	urlValues := make(url.Values)
	urlValues.Set("versioning", "")

	reqMetadata := requestMetadata{
		bucketName:       bucketName,
		queryValues:      urlValues,
		contentBody:      bytes.NewReader(config),
		contentLength:    length,
		contentMD5Base64: md5sum,
		contentSHA256Hex: sha256sum,
	}

	// Execute PUT to set a bucket versioning.
	resp, err := c.executeMethod(ctx, "PUT", reqMetadata)
	defer closeResponse(resp)
	if err != nil {
		return err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return httpRespToErrorResponse(resp, bucketName, "")
		}
	}
	return nil
}

type VersioningStatus int

const (
	VersioningEnabled = iota
	VersioningSuspended
)

type VersioningOpts int

const (
	VersioningMFA = iota
)

// SetBucketVersioning - Enable/Suspend object versioning in given bucket.
func (c Client) SetBucketVersioning(ctx context.Context, bucketName string, status VersioningStatus, opts ...VersioningOpts) error {
	switch status {
	case VersioningEnabled:
		return c.setVersioning(ctx, bucketName, versionEnableConfig, versionEnableConfigLen, versionEnableConfigMD5Sum, versionEnableConfigSHA256)
	case VersioningSuspended:
		return c.setVersioning(ctx, bucketName, versionDisableConfig, versionDisableConfigLen, versionDisableConfigMD5Sum, versionDisableConfigSHA256)
	}

	return ErrInvalidArgument("invalid versioning status")
}
