package minio

import (
	"bytes"
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"net/url"
)

/// Bucket operations

// MakeBucket makes a new bucket.
//
// Optional arguments are acl and location - by default all buckets are created
// with ``private`` acl and in US Standard region.
//
// ACL valid values - http://docs.aws.amazon.com/AmazonS3/latest/dev/acl-overview.html
//
//  private - owner gets full access [default].
//  public-read - owner gets full access, all others get read access.
//  public-read-write - owner gets full access, all others get full access too.
//  authenticated-read - owner gets full access, authenticated users get read access.
//
// For Amazon S3 for more supported regions - http://docs.aws.amazon.com/general/latest/gr/rande.html
// For Google Cloud Storage for more supported regions - https://cloud.google.com/storage/docs/bucket-locations
func (a API) MakeBucket(bucketName string, acl BucketACL, region string) error {
	if err := isValidBucketName(bucketName); err != nil {
		return err
	}
	if !acl.isValidBucketACL() {
		return ErrInvalidArgument("Unrecognized ACL " + acl.String())
	}

	if region == "" {
		region = "us-east-1"
	}

	req, err := a.makeBucketRequest(bucketName, string(acl), region)
	if err != nil {
		return err
	}

	// Initiate the request.
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return err
	}

	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return BodyToErrorResponse(resp.Body)
		}
	}

	return nil
}

// makeBucketRequest constructs request for makeBucket.
func (a API) makeBucketRequest(bucketName, acl, region string) (*Request, error) {
	// get target URL.
	targetURL, err := getTargetURL(a.endpointURL, bucketName, "", url.Values{})
	if err != nil {
		return nil, err
	}

	// Initialize request metadata.
	var reqMetadata requestMetadata
	reqMetadata = requestMetadata{
		userAgent:        a.userAgent,
		credentials:      a.credentials,
		bucketRegion:     region,
		contentTransport: a.httpTransport,
	}

	// If region is set use to create bucket location config.
	if region != "" {
		createBucketConfig := new(createBucketConfiguration)
		createBucketConfig.Location = region
		var createBucketConfigBytes []byte
		createBucketConfigBytes, err = xml.Marshal(createBucketConfig)
		if err != nil {
			return nil, err
		}
		createBucketConfigBuffer := bytes.NewBuffer(createBucketConfigBytes)
		reqMetadata.contentBody = ioutil.NopCloser(createBucketConfigBuffer)
		reqMetadata.contentLength = int64(createBucketConfigBuffer.Len())
		reqMetadata.contentSha256Bytes = sum256(createBucketConfigBuffer.Bytes())
	}

	// Initialize new request.
	req, err := newRequest("PUT", targetURL, reqMetadata)
	if err != nil {
		return nil, err
	}

	// by default bucket acl is set to private.
	req.Set("x-amz-acl", "private")
	if acl != "" {
		req.Set("x-amz-acl", acl)
	}

	return req, nil
}

// SetBucketACL set the permissions on an existing bucket using access control lists (ACL).
//
// For example
//
//  private - owner gets full access [default].
//  public-read - owner gets full access, all others get read access.
//  public-read-write - owner gets full access, all others get full access too.
//  authenticated-read - owner gets full access, authenticated users get read access.
func (a API) SetBucketACL(bucketName string, acl BucketACL) error {
	if err := isValidBucketName(bucketName); err != nil {
		return err
	}
	if !acl.isValidBucketACL() {
		return ErrInvalidArgument("Unrecognized ACL " + acl.String())
	}

	// Initialize a new request.
	req, err := a.setBucketACLRequest(bucketName, string(acl))
	if err != nil {
		return err
	}

	// Initiate the request.
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return err
	}

	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return BodyToErrorResponse(resp.Body)
		}
	}
	return nil
}

// setBucketRequestACL constructs request for SetBucketACL.
func (a API) setBucketACLRequest(bucketName, acl string) (*Request, error) {
	// Set acl query.
	urlValues := make(url.Values)
	urlValues.Set("acl", "")

	// get target URL.
	targetURL, err := getTargetURL(a.endpointURL, bucketName, "", urlValues)
	if err != nil {
		return nil, err
	}

	// get bucket region.
	region, err := a.getRegion(bucketName)
	if err != nil {
		return nil, err
	}

	// Instantiate a new request.
	req, err := newRequest("PUT", targetURL, requestMetadata{
		credentials:      a.credentials,
		userAgent:        a.userAgent,
		bucketRegion:     region,
		contentTransport: a.httpTransport,
	})
	if err != nil {
		return nil, err
	}

	// Set relevant acl.
	if acl != "" {
		req.Set("x-amz-acl", acl)
	} else {
		req.Set("x-amz-acl", "private")
	}

	// Return.
	return req, nil
}
