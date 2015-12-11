/*
 * Minio Go Library for Amazon S3 Compatible Cloud Storage (C) 2015 Minio, Inc.
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
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// closeResp close non nil response with any response Body.
// convenient wrapper to drain any remaining data on response body.
//
// Subsequently this allows golang http RoundTripper
// to re-use the same connection for future requests.
func closeResp(resp *http.Response) {
	// Callers should close resp.Body when done reading from it.
	// If resp.Body is not closed, the Client's underlying RoundTripper
	// (typically Transport) may not be able to re-use a persistent TCP
	// connection to the server for a subsequent "keep-alive" request.
	if resp != nil && resp.Body != nil {
		// Drain any remaining Body and then close the connection.
		// Without this closing connection would disallow re-using
		// the same connection for future uses.
		//  - http://stackoverflow.com/a/17961593/4465767
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}
}

// setRegion - set region for the bucketName in private region map cache.
func (a API) setRegion(bucketName string) (string, error) {
	// If signature version '2', no need to fetch bucket location.
	if a.credentials.Signature.isV2() {
		return "us-east-1", nil
	}
	if a.credentials.Signature.isV4() && !isAmazonEndpoint(a.endpointURL) {
		return "us-east-1", nil
	}
	// get bucket location.
	location, err := a.getBucketLocation(bucketName)
	if err != nil {
		return "", err
	}
	// location is region in context of S3 API.
	a.mutex.Lock()
	a.regionMap[bucketName] = location
	a.mutex.Unlock()
	return location, nil
}

// getRegion - get region for the bucketName from region map cache.
func (a API) getRegion(bucketName string) (string, error) {
	// If signature version '2', no need to fetch bucket location.
	if a.credentials.Signature.isV2() {
		return "us-east-1", nil
	}
	// If signature version '4' and latest and endpoint is not Amazon.
	// Return 'us-east-1'
	if a.credentials.Signature.isV4() || a.credentials.Signature.isLatest() {
		if !isAmazonEndpoint(a.endpointURL) {
			return "us-east-1", nil
		}
	}
	// Search through regionMap protected.
	a.mutex.Lock()
	region, ok := a.regionMap[bucketName]
	a.mutex.Unlock()
	// return if found.
	if ok {
		return region, nil
	}
	// Set region if no region was found for a bucket.
	region, err := a.setRegion(bucketName)
	if err != nil {
		return "us-east-1", err
	}
	return region, nil
}

// putBucketRequest wrapper creates a new putBucket request.
func (a API) putBucketRequest(bucketName, acl, region string) (*Request, error) {
	// get target URL.
	targetURL, err := getTargetURL(a.endpointURL, bucketName, "", url.Values{})
	if err != nil {
		return nil, err
	}

	// Initialize request metadata.
	var rmetadata requestMetadata
	rmetadata = requestMetadata{
		userAgent:    a.userAgent,
		credentials:  a.credentials,
		bucketRegion: region,
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
		rmetadata.contentBody = ioutil.NopCloser(createBucketConfigBuffer)
		rmetadata.contentLength = int64(createBucketConfigBuffer.Len())
		rmetadata.contentSha256Bytes = sum256(createBucketConfigBuffer.Bytes())
	}

	// Initialize new request.
	req, err := newRequest("PUT", targetURL, rmetadata)
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

/// Bucket Write Operations

// putBucket create a new bucket.
//
// Requires valid AWS Access Key ID to authenticate requests.
// Anonymous requests are never allowed to create buckets.
//
// optional arguments are acl and location - by default all buckets are created
// with ``private`` acl and location set to US Standard if one wishes to set
// different ACLs and Location one can set them properly.
//
// ACL valid values
// ------------------
// private - owner gets full access [DEFAULT].
// public-read - owner gets full access, others get read access.
// public-read-write - owner gets full access, others get full access too.
// authenticated-read - owner gets full access, authenticated users get read access.
// ------------------
//
// Region valid values.
// ------------------
// [ us-west-1 | us-west-2 | eu-west-1 | eu-central-1 | ap-southeast-1 | ap-northeast-1 | ap-southeast-2 | sa-east-1 ]
// Default - US standard
func (a API) putBucket(bucketName, acl, region string) error {
	// Initialize a new request.
	req, err := a.putBucketRequest(bucketName, acl, region)
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

// putBucketRequestACL wrapper creates a new putBucketACL request.
func (a API) putBucketACLRequest(bucketName, acl string) (*Request, error) {
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
		credentials:  a.credentials,
		userAgent:    a.userAgent,
		bucketRegion: region,
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

// putBucketACL set the permissions on an existing bucket using Canned ACL's.
func (a API) putBucketACL(bucketName, acl string) error {
	// Initialize a new request.
	req, err := a.putBucketACLRequest(bucketName, acl)
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

// getBucketACLRequest wrapper creates a new getBucketACL request.
func (a API) getBucketACLRequest(bucketName string) (*Request, error) {
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
	req, err := newRequest("GET", targetURL, requestMetadata{
		bucketRegion: region,
		credentials:  a.credentials,
	})
	if err != nil {
		return nil, err
	}
	return req, nil
}

// getBucketACL get the acl information on an existing bucket.
func (a API) getBucketACL(bucketName string) (accessControlPolicy, error) {
	// Initialize a new request.
	req, err := a.getBucketACLRequest(bucketName)
	if err != nil {
		return accessControlPolicy{}, err
	}

	// Initiate the request.
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return accessControlPolicy{}, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return accessControlPolicy{}, BodyToErrorResponse(resp.Body)
		}
	}

	// Decode access control policy.
	policy := accessControlPolicy{}
	err = xmlDecoder(resp.Body, &policy)
	if err != nil {
		return accessControlPolicy{}, err
	}

	// If Google private bucket policy doesn't have any Grant list.
	if isGoogleEndpoint(a.endpointURL) {
		return policy, nil
	}
	if policy.AccessControlList.Grant == nil {
		errorResponse := ErrorResponse{
			Code:            "InternalError",
			Message:         "Access control Grant list is empty, please report this at https://github.com/minio/minio-go/issues.",
			BucketName:      bucketName,
			RequestID:       resp.Header.Get("x-amz-request-id"),
			HostID:          resp.Header.Get("x-amz-id-2"),
			AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
		}
		return accessControlPolicy{}, errorResponse
	}
	return policy, nil
}

// getBucketLocationRequest wrapper creates a new getBucketLocation request.
func (a API) getBucketLocationRequest(bucketName string) (*Request, error) {
	// Set location query.
	urlValues := make(url.Values)
	urlValues.Set("location", "")

	// Set get bucket location always as path style.
	targetURL := a.endpointURL
	targetURL.Path = filepath.Join(bucketName, "")
	targetURL.RawQuery = urlValues.Encode()

	// Instantiate a new request.
	req, err := newRequest("GET", targetURL, requestMetadata{
		bucketRegion: "us-east-1",
		credentials:  a.credentials,
	})
	if err != nil {
		return nil, err
	}
	return req, nil
}

// getBucketLocation uses location subresource to return a bucket's region.
func (a API) getBucketLocation(bucketName string) (string, error) {
	// Initialize a new request.
	req, err := a.getBucketLocationRequest(bucketName)
	if err != nil {
		return "", err
	}

	// Initiate the request.
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return "", err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return "", BodyToErrorResponse(resp.Body)
		}
	}

	// Extract location.
	var locationConstraint string
	err = xmlDecoder(resp.Body, &locationConstraint)
	if err != nil {
		return "", err
	}

	// location is empty will be 'us-east-1'.
	if locationConstraint == "" {
		return "us-east-1", nil
	}

	// location can be 'EU' convert it to meaningful 'eu-west-1'.
	if locationConstraint == "EU" {
		return "eu-west-1", nil
	}

	// return location.
	return locationConstraint, nil
}

// listObjectsRequest wrapper creates a new listObjects request.
func (a API) listObjectsRequest(bucketName, objectPrefix, objectMarker, delimiter string, maxkeys int) (*Request, error) {
	// Get resources properly escaped and lined up before
	// using them in http request.
	urlValues := make(url.Values)
	// Set object prefix.
	urlValues.Set("prefix", urlEncodePath(objectPrefix))
	// Set object marker.
	urlValues.Set("marker", urlEncodePath(objectMarker))
	// Set delimiter.
	urlValues.Set("delimiter", delimiter)
	// Set max keys.
	urlValues.Set("max-keys", fmt.Sprintf("%d", maxkeys))

	// Get target url.
	targetURL, err := getTargetURL(a.endpointURL, bucketName, "", urlValues)
	if err != nil {
		return nil, err
	}

	// get bucket region.
	region, err := a.getRegion(bucketName)
	if err != nil {
		return nil, err
	}

	// Initialize a new request.
	r, err := newRequest("GET", targetURL, requestMetadata{
		credentials:  a.credentials,
		userAgent:    a.userAgent,
		bucketRegion: region,
	})
	if err != nil {
		return nil, err
	}
	return r, nil
}

/// Bucket Read Operations.

// listObjects - (List Objects) - List some or all (up to 1000) of the objects in a bucket.
//
// You can use the request parameters as selection criteria to return a subset of the objects in a bucket.
// request paramters :-
// ---------
// ?marker - Specifies the key to start with when listing objects in a bucket.
// ?delimiter - A delimiter is a character you use to group keys.
// ?prefix - Limits the response to keys that begin with the specified prefix.
// ?max-keys - Sets the maximum number of keys returned in the response body.
func (a API) listObjects(bucketName, objectPrefix, objectMarker, delimiter string, maxkeys int) (listBucketResult, error) {
	if err := isValidBucketName(bucketName); err != nil {
		return listBucketResult{}, err
	}
	if err := isValidObjectPrefix(objectPrefix); err != nil {
		return listBucketResult{}, err
	}
	req, err := a.listObjectsRequest(bucketName, objectPrefix, objectMarker, delimiter, maxkeys)
	if err != nil {
		return listBucketResult{}, err
	}
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return listBucketResult{}, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return listBucketResult{}, BodyToErrorResponse(resp.Body)
		}
	}
	listBucketResult := listBucketResult{}
	err = xmlDecoder(resp.Body, &listBucketResult)
	if err != nil {
		return listBucketResult, err
	}
	// close body while returning, along with any error.
	return listBucketResult, nil
}

// headBucketRequest wrapper creates a new headBucket request.
func (a API) headBucketRequest(bucketName string) (*Request, error) {
	targetURL, err := getTargetURL(a.endpointURL, bucketName, "", url.Values{})
	if err != nil {
		return nil, err
	}

	// get bucket region.
	region, err := a.getRegion(bucketName)
	if err != nil {
		return nil, err
	}

	return newRequest("HEAD", targetURL, requestMetadata{
		credentials:  a.credentials,
		userAgent:    a.userAgent,
		bucketRegion: region,
	})
}

// headBucket useful to determine if a bucket exists and you have permission to access it.
func (a API) headBucket(bucketName string) error {
	if err := isValidBucketName(bucketName); err != nil {
		return err
	}
	req, err := a.headBucketRequest(bucketName)
	if err != nil {
		return err
	}
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			// Head has no response body, handle it.
			var errorResponse ErrorResponse
			switch resp.StatusCode {
			case http.StatusNotFound:
				errorResponse = ErrorResponse{
					Code:            "NoSuchBucket",
					Message:         "The specified bucket does not exist.",
					BucketName:      bucketName,
					RequestID:       resp.Header.Get("x-amz-request-id"),
					HostID:          resp.Header.Get("x-amz-id-2"),
					AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
				}
			case http.StatusForbidden:
				errorResponse = ErrorResponse{
					Code:            "AccessDenied",
					Message:         "Access Denied.",
					BucketName:      bucketName,
					RequestID:       resp.Header.Get("x-amz-request-id"),
					HostID:          resp.Header.Get("x-amz-id-2"),
					AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
				}
			default:
				errorResponse = ErrorResponse{
					Code:            resp.Status,
					Message:         resp.Status,
					BucketName:      bucketName,
					RequestID:       resp.Header.Get("x-amz-request-id"),
					HostID:          resp.Header.Get("x-amz-id-2"),
					AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
				}
			}
			return errorResponse
		}
	}
	return nil
}

// deleteBucketRequest wrapper creates a new deleteBucket request.
func (a API) deleteBucketRequest(bucketName string) (*Request, error) {
	targetURL, err := getTargetURL(a.endpointURL, bucketName, "", url.Values{})
	if err != nil {
		return nil, err
	}

	// get bucket region.
	region, err := a.getRegion(bucketName)
	if err != nil {
		return nil, err
	}

	return newRequest("DELETE", targetURL, requestMetadata{
		credentials:  a.credentials,
		userAgent:    a.userAgent,
		bucketRegion: region,
	})
}

// deleteBucket deletes the bucket name.
//
// NOTE: -
//  All objects (including all object versions and delete markers)
//  in the bucket must be deleted before successfully attempting this request.
func (a API) deleteBucket(bucketName string) error {
	if err := isValidBucketName(bucketName); err != nil {
		return err
	}
	req, err := a.deleteBucketRequest(bucketName)
	if err != nil {
		return err
	}
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusNoContent {
			var errorResponse ErrorResponse
			switch resp.StatusCode {
			case http.StatusNotFound:
				errorResponse = ErrorResponse{
					Code:            "NoSuchBucket",
					Message:         "The specified bucket does not exist.",
					BucketName:      bucketName,
					RequestID:       resp.Header.Get("x-amz-request-id"),
					HostID:          resp.Header.Get("x-amz-id-2"),
					AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
				}
			case http.StatusForbidden:
				errorResponse = ErrorResponse{
					Code:            "AccessDenied",
					Message:         "Access Denied.",
					BucketName:      bucketName,
					RequestID:       resp.Header.Get("x-amz-request-id"),
					HostID:          resp.Header.Get("x-amz-id-2"),
					AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
				}
			case http.StatusConflict:
				errorResponse = ErrorResponse{
					Code:            "Conflict",
					Message:         "Bucket not empty.",
					BucketName:      bucketName,
					RequestID:       resp.Header.Get("x-amz-request-id"),
					HostID:          resp.Header.Get("x-amz-id-2"),
					AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
				}
			default:
				errorResponse = ErrorResponse{
					Code:            resp.Status,
					Message:         resp.Status,
					BucketName:      bucketName,
					RequestID:       resp.Header.Get("x-amz-request-id"),
					HostID:          resp.Header.Get("x-amz-id-2"),
					AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
				}
			}
			return errorResponse
		}
	}
	return nil
}

/// Object Read/Write/Stat Operations

// putObjectRequest wrapper creates a new PutObject request.
func (a API) putObjectRequest(bucketName, objectName string, putObjMetadata putObjectMetadata) (*Request, error) {
	targetURL, err := getTargetURL(a.endpointURL, bucketName, objectName, url.Values{})
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(putObjMetadata.ContentType) == "" {
		putObjMetadata.ContentType = "application/octet-stream"
	}

	// get bucket region.
	region, err := a.getRegion(bucketName)
	if err != nil {
		return nil, err
	}

	// Set headers.
	putObjMetadataHeader := make(http.Header)
	putObjMetadataHeader.Set("Content-Type", putObjMetadata.ContentType)

	// Populate request metadata.
	rmetadata := requestMetadata{
		credentials:        a.credentials,
		userAgent:          a.userAgent,
		bucketRegion:       region,
		contentBody:        putObjMetadata.ReadCloser,
		contentLength:      putObjMetadata.Size,
		contentHeader:      putObjMetadataHeader,
		contentSha256Bytes: putObjMetadata.Sha256Sum,
		contentMD5Bytes:    putObjMetadata.MD5Sum,
	}
	r, err := newRequest("PUT", targetURL, rmetadata)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// putObject - add an object to a bucket.
// NOTE: You must have WRITE permissions on a bucket to add an object to it.
func (a API) putObject(bucketName, objectName string, putObjMetadata putObjectMetadata) (ObjectStat, error) {
	req, err := a.putObjectRequest(bucketName, objectName, putObjMetadata)
	if err != nil {
		return ObjectStat{}, err
	}
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return ObjectStat{}, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return ObjectStat{}, BodyToErrorResponse(resp.Body)
		}
	}
	var metadata ObjectStat
	metadata.ETag = strings.Trim(resp.Header.Get("ETag"), "\"") // trim off the odd double quotes
	return metadata, nil
}

// presignedPostPolicy - generate post form data.
func (a API) presignedPostPolicy(p *PostPolicy) (map[string]string, error) {
	// get targetURL.
	targetURL, err := getTargetURL(a.endpointURL, p.formData["bucket"], "", url.Values{})
	if err != nil {
		return nil, err
	}

	// get bucket region.
	region, err := a.getRegion(p.formData["bucket"])
	if err != nil {
		return nil, err
	}

	// Instantiate a new request.
	req, err := newRequest("POST", targetURL, requestMetadata{
		credentials:  a.credentials,
		userAgent:    a.userAgent,
		bucketRegion: region,
	})
	if err != nil {
		return nil, err
	}

	// Keep time.
	t := time.Now().UTC()
	if req.credentials.Signature.isV2() {
		policyBase64 := p.base64()
		p.formData["policy"] = policyBase64
		// for all other regions set this value to be 'AWSAccessKeyId'.
		if isGoogleEndpoint(a.endpointURL) {
			p.formData["GoogleAccessId"] = req.credentials.AccessKeyID
		} else {
			p.formData["AWSAccessKeyId"] = req.credentials.AccessKeyID
		}
		p.formData["signature"] = req.PostPresignSignatureV2(policyBase64)
		return p.formData, nil
	}
	credential := getCredential(req.credentials.AccessKeyID, req.bucketRegion, t)
	p.addNewPolicy(policyCondition{
		matchType: "eq",
		condition: "$x-amz-date",
		value:     t.Format(iso8601DateFormat),
	})
	p.addNewPolicy(policyCondition{
		matchType: "eq",
		condition: "$x-amz-algorithm",
		value:     authHeader,
	})
	p.addNewPolicy(policyCondition{
		matchType: "eq",
		condition: "$x-amz-credential",
		value:     credential,
	})
	policyBase64 := p.base64()
	p.formData["policy"] = policyBase64
	p.formData["x-amz-algorithm"] = authHeader
	p.formData["x-amz-credential"] = credential
	p.formData["x-amz-date"] = t.Format(iso8601DateFormat)
	p.formData["x-amz-signature"] = req.PostPresignSignatureV4(policyBase64, t)
	return p.formData, nil
}

// presignedPutObject - generate presigned PUT url.
func (a API) presignedPutObject(bucketName, objectName string, expires int64) (string, error) {
	// get targetURL.
	targetURL, err := getTargetURL(a.endpointURL, bucketName, objectName, url.Values{})
	if err != nil {
		return "", err
	}

	// get bucket region.
	region, err := a.getRegion(bucketName)
	if err != nil {
		return "", err
	}

	// Instantiate a new request.
	req, err := newRequest("PUT", targetURL, requestMetadata{
		credentials:  a.credentials,
		expires:      expires,
		userAgent:    a.userAgent,
		bucketRegion: region,
	})
	if err != nil {
		return "", err
	}
	if req.credentials.Signature.isV2() {
		return req.PreSignV2()
	}
	return req.PreSignV4()
}

// presignedGetObject - generate presigned get object URL.
func (a API) presignedGetObject(bucketName, objectName string, expires, offset, length int64) (string, error) {
	// get targetURL.
	targetURL, err := getTargetURL(a.endpointURL, bucketName, objectName, url.Values{})
	if err != nil {
		return "", err
	}

	// get bucket region.
	region, err := a.getRegion(bucketName)
	if err != nil {
		return "", err
	}

	// Instantiate a new request.
	req, err := newRequest("GET", targetURL, requestMetadata{
		credentials:  a.credentials,
		expires:      expires,
		userAgent:    a.userAgent,
		bucketRegion: region,
	})
	if err != nil {
		return "", err
	}

	// Set ranges if length and offset are valid.
	if length > 0 && offset >= 0 {
		req.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, offset+length-1))
	} else if offset > 0 && length == 0 {
		req.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	} else if length > 0 && offset == 0 {
		req.Set("Range", fmt.Sprintf("bytes=-%d", length))
	}
	if req.credentials.Signature.isV2() {
		return req.PreSignV2()
	}
	return req.PreSignV4()
}

// getObjectRequest wrapper creates a new getObject request.
func (a API) getObjectRequest(bucketName, objectName string, offset, length int64) (*Request, error) {
	// get targetURL.
	targetURL, err := getTargetURL(a.endpointURL, bucketName, objectName, url.Values{})
	if err != nil {
		return nil, err
	}

	// get bucket region.
	region, err := a.getRegion(bucketName)
	if err != nil {
		return nil, err
	}

	// Instantiate a new request.
	req, err := newRequest("GET", targetURL, requestMetadata{
		credentials:  a.credentials,
		userAgent:    a.userAgent,
		bucketRegion: region,
	})
	if err != nil {
		return nil, err
	}

	// Set ranges if length and offset are valid.
	if length > 0 && offset >= 0 {
		req.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, offset+length-1))
	} else if offset > 0 && length == 0 {
		req.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	} else if length < 0 && offset == 0 {
		req.Set("Range", fmt.Sprintf("bytes=%d", length))
	}
	return req, nil
}

// getObject - retrieve object from Object Storage.
//
// Additionally this function also takes range arguments to download the specified
// range bytes of an object. Setting offset and length = 0 will download the full object.
//
// For more information about the HTTP Range header.
// go to http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.35.
func (a API) getObject(bucketName, objectName string, offset, length int64) (io.ReadCloser, ObjectStat, error) {
	if err := isValidBucketName(bucketName); err != nil {
		return nil, ObjectStat{}, err
	}
	if err := isValidObjectName(objectName); err != nil {
		return nil, ObjectStat{}, err
	}
	req, err := a.getObjectRequest(bucketName, objectName, offset, length)
	if err != nil {
		return nil, ObjectStat{}, err
	}
	resp, err := req.Do()
	if err != nil {
		return nil, ObjectStat{}, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
			return nil, ObjectStat{}, BodyToErrorResponse(resp.Body)
		}
	}
	md5sum := strings.Trim(resp.Header.Get("ETag"), "\"") // trim off the odd double quotes
	date, err := time.Parse(http.TimeFormat, resp.Header.Get("Last-Modified"))
	if err != nil {
		return nil, ObjectStat{}, ErrorResponse{
			Code:            "InternalError",
			Message:         "Last-Modified time format not recognized, please report this issue at https://github.com/minio/minio-go/issues.",
			RequestID:       resp.Header.Get("x-amz-request-id"),
			HostID:          resp.Header.Get("x-amz-id-2"),
			AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
		}
	}
	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	var objectstat ObjectStat
	objectstat.ETag = md5sum
	objectstat.Key = objectName
	objectstat.Size = resp.ContentLength
	objectstat.LastModified = date
	objectstat.ContentType = contentType

	// do not close body here, caller will close
	return resp.Body, objectstat, nil
}

// deleteObjectRequest wrapper creates a new deleteObject request.
func (a API) deleteObjectRequest(bucketName, objectName string) (*Request, error) {
	// get targetURL.
	targetURL, err := getTargetURL(a.endpointURL, bucketName, objectName, url.Values{})
	if err != nil {
		return nil, err
	}

	// get bucket region.
	region, err := a.getRegion(bucketName)
	if err != nil {
		return nil, err
	}

	// Instantiate a new request.
	req, err := newRequest("DELETE", targetURL, requestMetadata{
		credentials:  a.credentials,
		userAgent:    a.userAgent,
		bucketRegion: region,
	})
	if err != nil {
		return nil, err
	}
	return req, nil
}

// deleteObject deletes a given object from a bucket.
func (a API) deleteObject(bucketName, objectName string) error {
	if err := isValidBucketName(bucketName); err != nil {
		return err
	}
	if err := isValidObjectName(objectName); err != nil {
		return err
	}
	req, err := a.deleteObjectRequest(bucketName, objectName)
	if err != nil {
		return err
	}
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return err
	}
	// DeleteObject always responds with http '204' even for
	// objects which do not exist. So no need to handle them
	// specifically.
	return nil
}

// headObjectRequest wrapper creates a new headObject request.
func (a API) headObjectRequest(bucketName, objectName string) (*Request, error) {
	// get targetURL.
	targetURL, err := getTargetURL(a.endpointURL, bucketName, objectName, url.Values{})
	if err != nil {
		return nil, err
	}

	// get bucket region.
	region, err := a.getRegion(bucketName)
	if err != nil {
		return nil, err
	}

	// Instantiate a new request.
	req, err := newRequest("HEAD", targetURL, requestMetadata{
		credentials:  a.credentials,
		userAgent:    a.userAgent,
		bucketRegion: region,
	})
	if err != nil {
		return nil, err
	}

	// Return new request.
	return req, nil
}

// headObject retrieves metadata for an object without returning the object itself.
func (a API) headObject(bucketName, objectName string) (ObjectStat, error) {
	if err := isValidBucketName(bucketName); err != nil {
		return ObjectStat{}, err
	}
	if err := isValidObjectName(objectName); err != nil {
		return ObjectStat{}, err
	}
	req, err := a.headObjectRequest(bucketName, objectName)
	if err != nil {
		return ObjectStat{}, err
	}
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return ObjectStat{}, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			var errorResponse ErrorResponse
			switch resp.StatusCode {
			case http.StatusNotFound:
				errorResponse = ErrorResponse{
					Code:            "NoSuchKey",
					Message:         "The specified key does not exist.",
					BucketName:      bucketName,
					Key:             objectName,
					RequestID:       resp.Header.Get("x-amz-request-id"),
					HostID:          resp.Header.Get("x-amz-id-2"),
					AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
				}
			case http.StatusForbidden:
				errorResponse = ErrorResponse{
					Code:            "AccessDenied",
					Message:         "Access Denied.",
					BucketName:      bucketName,
					Key:             objectName,
					RequestID:       resp.Header.Get("x-amz-request-id"),
					HostID:          resp.Header.Get("x-amz-id-2"),
					AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
				}
			default:
				errorResponse = ErrorResponse{
					Code:            resp.Status,
					Message:         resp.Status,
					BucketName:      bucketName,
					Key:             objectName,
					RequestID:       resp.Header.Get("x-amz-request-id"),
					HostID:          resp.Header.Get("x-amz-id-2"),
					AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
				}

			}
			return ObjectStat{}, errorResponse
		}
	}
	md5sum := strings.Trim(resp.Header.Get("ETag"), "\"") // trim off the odd double quotes
	size, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return ObjectStat{}, ErrorResponse{
			Code:            "InternalError",
			Message:         "Content-Length not recognized, please report this issue at https://github.com/minio/minio-go/issues.",
			BucketName:      bucketName,
			Key:             objectName,
			RequestID:       resp.Header.Get("x-amz-request-id"),
			HostID:          resp.Header.Get("x-amz-id-2"),
			AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
		}
	}
	date, err := time.Parse(http.TimeFormat, resp.Header.Get("Last-Modified"))
	if err != nil {
		return ObjectStat{}, ErrorResponse{
			Code:            "InternalError",
			Message:         "Last-Modified time format not recognized, please report this issue at https://github.com/minio/minio-go/issues.",
			BucketName:      bucketName,
			Key:             objectName,
			RequestID:       resp.Header.Get("x-amz-request-id"),
			HostID:          resp.Header.Get("x-amz-id-2"),
			AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
		}
	}
	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	// Save object metadata info.
	var objectstat ObjectStat
	objectstat.ETag = md5sum
	objectstat.Key = objectName
	objectstat.Size = size
	objectstat.LastModified = date
	objectstat.ContentType = contentType
	return objectstat, nil
}

/// Service Operations.

// listBucketRequest wrapper creates a new listBuckets request.
func (a API) listBucketsRequest() (*Request, error) {
	// get targetURL.
	targetURL, err := getTargetURL(a.endpointURL, "", "", url.Values{})
	if err != nil {
		return nil, err
	}
	// Instantiate a new request.
	req, err := newRequest("GET", targetURL, requestMetadata{
		credentials:  a.credentials,
		userAgent:    a.userAgent,
		bucketRegion: "us-east-1",
	})
	if err != nil {
		return nil, err
	}
	return req, nil
}

// listBuckets list of all buckets owned by the authenticated sender of the request.
func (a API) listBuckets() (listAllMyBucketsResult, error) {
	req, err := a.listBucketsRequest()
	if err != nil {
		return listAllMyBucketsResult{}, err
	}
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return listAllMyBucketsResult{}, err
	}
	if resp != nil {
		// for un-authenticated requests, amazon sends a redirect handle it.
		if resp.StatusCode == http.StatusTemporaryRedirect {
			return listAllMyBucketsResult{}, ErrorResponse{
				Code:            "AccessDenied",
				Message:         "Anonymous access is forbidden for this operation.",
				RequestID:       resp.Header.Get("x-amz-request-id"),
				HostID:          resp.Header.Get("x-amz-id-2"),
				AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
			}
		}
		if resp.StatusCode != http.StatusOK {
			return listAllMyBucketsResult{}, BodyToErrorResponse(resp.Body)
		}
	}
	listAllMyBucketsResult := listAllMyBucketsResult{}
	err = xmlDecoder(resp.Body, &listAllMyBucketsResult)
	if err != nil {
		return listAllMyBucketsResult, err
	}
	return listAllMyBucketsResult, nil
}
