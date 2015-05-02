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

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
)

type api struct {
	config *Config
}

// Config - main configuration struct used by all to set endpoint, credentials, and other options for requests.
type Config struct {
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string
	ContentType     string
	UserAgent       string
}

// New - instantiate a new minio api client
func New(config *Config) API {
	return &api{config}
}

func (a *api) putBucketRequest(bucket string) (*Request, error) {
	op := &Operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "PUT",
		HTTPPath:   "/" + bucket,
	}
	return NewRequest(op, a.config, nil)
}

/// Bucket Write Operations

// PutBucket - create a new bucket
//
// Requires valid AWS Access Key ID to authenticate requests
// Anonymous requests are never allowed to create buckets
func (a *api) PutBucket(bucket string) error {
	req, err := a.putBucketRequest(bucket)
	if err != nil {
		return err
	}
	resp, err := req.Do()
	if err != nil {
		return err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return ResponseToError(resp)
		}
	}
	return resp.Body.Close()
}

func (a *api) putBucketRequestACL(bucket, acl string) (*Request, error) {
	op := &Operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "PUT",
		HTTPPath:   "/" + bucket + "?acl",
	}
	req, err := NewRequest(op, a.config, nil)
	if err != nil {
		return nil, err
	}
	req.Set("x-amz-acl", acl)
	return req, nil
}

// PutBucketACL - set the permissions on an existing bucket using access control lists (ACL)
//
// Currently supported are
//    - "private"
//    - "public-read"
//    - "public-read-write"
func (a *api) PutBucketACL(bucket, acl string) error {
	req, err := a.putBucketRequestACL(bucket, acl)
	if err != nil {
		return err
	}
	resp, err := req.Do()
	if err != nil {
		return err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return ResponseToError(resp)
		}
	}
	return resp.Body.Close()
}

/// Bucket Read Operations

// ListObjects - (List Objects) - List some or all (up to 1000) of the objects in a bucket.
//
// You can use the request parameters as selection criteria to return a subset of the objects in a bucket.
// request paramters :-
// ---------
// ?delimiter - A delimiter is a character you use to group keys.
// ?marker - Specifies the key to start with when listing objects in a bucket.
// ?max-keys - Sets the maximum number of keys returned in the response body.
// ?prefix - Limits the response to keys that begin with the specified prefix.
func (a *api) ListObjects(bucket string) (ListObjects, error) {
	return ListObjects{}, nil
}

func (a *api) headBucketRequest(bucket string) (*Request, error) {
	op := &Operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "HEAD",
		HTTPPath:   "/" + bucket,
	}
	return NewRequest(op, a.config, nil)
}

// HeadBucket - useful to determine if a bucket exists and you have permission to access it.
func (a *api) HeadBucket(bucket string) error {
	req, err := a.headBucketRequest(bucket)
	if err != nil {
		return err
	}
	resp, err := req.Do()
	if err != nil {
		return err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return ResponseToError(resp)
		}
	}
	return resp.Body.Close()
}

/// Object Read/Write/Stat Operations

func (a *api) putObjectRequest(bucket, object string, size int64, body io.ReadSeeker) (*Request, error) {
	op := &Operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "PUT",
		HTTPPath:   "/" + bucket + "/" + object,
	}
	req, err := NewRequest(op, a.config, ioutil.NopCloser(body))
	if err != nil {
		return nil, err
	}
	req.Set("Content-Length", strconv.FormatInt(size, 10))
	return req, nil
}

// Put - add an object to a bucket
//
// You must have WRITE permissions on a bucket to add an object to it.
func (a *api) PutObject(bucket, object string, size int64, body io.ReadSeeker) error {
	req, err := a.putObjectRequest(bucket, object, size, body)
	if err != nil {
		return err
	}
	resp, err := req.Do()
	if err != nil {
		return err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return ResponseToError(resp)
		}
	}
	return resp.Body.Close()
}

func (a *api) getObjectRequest(bucket, object string, offset, length uint64) (*Request, error) {
	op := &Operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "GET",
		HTTPPath:   "/" + bucket + "/" + object,
	}
	req, err := NewRequest(op, a.config, nil)
	if err != nil {
		return nil, err
	}
	// TODO fix this to support full - http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html
	switch {
	case length > 0:
		req.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, offset+length-1))
	default:
		req.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	return req, nil
}

// Get - retrieve object from Object Storage
//
// Additionally it also takes range arguments to download the specified range bytes of an object.
// For more information about the HTTP Range header, go to http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.35.
func (a *api) GetObject(bucket, object string, offset, length uint64) (io.ReadCloser, error) {
	req, err := a.getObjectRequest(bucket, object, offset, length)
	if err != nil {
		return nil, err
	}
	resp, err := req.Do()
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func (a *api) headObjectRequest(bucket, object string) (*Request, error) {
	op := &Operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "HEAD",
		HTTPPath:   "/" + bucket + "/" + object,
	}
	return NewRequest(op, a.config, nil)
}

// Head - retrieves metadata from an object without returning the object itself
func (a *api) HeadObject(bucket, object string) error {
	req, err := a.headObjectRequest(bucket, object)
	if err != nil {
		return err
	}
	resp, err := req.Do()
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

/// Service Operations

func (a *api) listBucketsRequest() (*Request, error) {
	op := &Operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "GET",
		HTTPPath:   "/",
	}
	return NewRequest(op, a.config, nil)
}

// ListBuckets - (List Buckets) - list of all buckets owned by the authenticated sender of the request
func (a *api) ListBuckets() (ListBuckets, error) {
	req, err := a.listBucketsRequest()
	if err != nil {
		return ListBuckets{}, err
	}
	resp, err := req.Do()
	if err != nil {
		return ListBuckets{}, err
	}
	var listbuckets ListBuckets
	decoder := xml.NewDecoder(resp.Body)
	err = decoder.Decode(&listbuckets)
	if err != nil {
		return ListBuckets{}, err
	}
	return listbuckets, nil
}
