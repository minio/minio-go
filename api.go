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
	"net/url"
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

func (a *api) listObjectsRequest(bucket string, maxkeys int, marker, prefix, delimiter string) (*Request, error) {
	type inputResources struct {
		maxkeys   int
		marker    string
		prefix    string
		delimiter string
	}
	var resource inputResources
	resource.maxkeys = maxkeys
	resource.marker = marker
	resource.prefix = prefix
	resource.delimiter = delimiter
	// resourceQuery - get resources properly escaped and lined up before
	// using them in http request
	resourceQuery := func(input inputResources) string {
		var maxkeys, marker, prefix, delimiter string
		maxkeys = fmt.Sprintf("?max-keys=%d", resource.maxkeys)
		if resource.marker != "" {
			marker = fmt.Sprintf("&marker=%s", url.QueryEscape(resource.marker))
		}
		if resource.prefix != "" {
			prefix = fmt.Sprintf("&prefix=%s", url.QueryEscape(resource.prefix))
		}
		if resource.delimiter != "" {
			delimiter = fmt.Sprintf("&delimiter=%s", url.QueryEscape(resource.delimiter))
		}
		return maxkeys + marker + prefix + delimiter
	}
	op := &Operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "GET",
		HTTPPath:   "/" + bucket + resourceQuery(resource),
	}
	req, err := NewRequest(op, a.config, nil)
	if err != nil {
		return nil, err
	}
	return req, nil
}

/// Bucket Read Operations

// ListObjects - (List Objects) - List some or all (up to 1000) of the objects in a bucket.
//
// You can use the request parameters as selection criteria to return a subset of the objects in a bucket.
// request paramters :-
// ---------
// ?marker - Specifies the key to start with when listing objects in a bucket.
// ?delimiter - A delimiter is a character you use to group keys.
// ?prefix - Limits the response to keys that begin with the specified prefix.
// ?max-keys - Sets the maximum number of keys returned in the response body.
func (a *api) ListObjects(bucket string, maxkeys int, marker, prefix, delimiter string) (*ListObjects, error) {
	req, err := a.listObjectsRequest(bucket, maxkeys, marker, prefix, delimiter)
	if err != nil {
		return nil, err
	}
	resp, err := req.Do()
	if err != nil {
		return nil, err
	}
	listobjects := new(ListObjects)
	decoder := xml.NewDecoder(resp.Body)
	err = decoder.Decode(listobjects)
	if err != nil {
		return nil, err
	}
	return listobjects, nil
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

func (a *api) deleteBucketRequest(bucket string) (*Request, error) {
	op := &Operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "DELETE",
		HTTPPath:   "/" + bucket,
	}
	return NewRequest(op, a.config, nil)
}

// DeleteBucket - deletes the bucket named in the URI
// NOTE: - All objects (including all object versions and delete markers) in the bucket must be deleted
func (a *api) DeleteBucket(bucket string) error {
	req, err := a.deleteBucketRequest(bucket)
	if err != nil {
		return err
	}
	resp, err := req.Do()
	if err != nil {
		return err
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
	md5Sum, err := contentMD5(body, size)
	if err != nil {
		return nil, err
	}
	req, err := NewRequest(op, a.config, ioutil.NopCloser(body))
	if err != nil {
		return nil, err
	}
	req.Set("Content-MD5", md5Sum)
	req.Set("Content-Length", strconv.FormatInt(size, 10))
	return req, nil
}

// PutObject - add an object to a bucket
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
	// TODO - fix this to support full - http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html
	switch {
	case length > 0:
		req.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, offset+length-1))
	default:
		req.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	return req, nil
}

// GetObject - retrieve object from Object Storage
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

// HeadObject - retrieves metadata from an object without returning the object itself
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

func (a *api) deleteObjectRequest(bucket, object string) (*Request, error) {
	op := &Operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "DELETE",
		HTTPPath:   "/" + bucket + "/" + object,
	}
	return NewRequest(op, a.config, nil)
}

// DeleteObject - removes the object
func (a *api) DeleteObject(bucket, object string) error {
	req, err := a.deleteObjectRequest(bucket, object)
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
func (a *api) ListBuckets() (*ListBuckets, error) {
	req, err := a.listBucketsRequest()
	if err != nil {
		return nil, err
	}
	resp, err := req.Do()
	if err != nil {
		return nil, err
	}
	listbuckets := new(ListBuckets)
	decoder := xml.NewDecoder(resp.Body)
	err = decoder.Decode(listbuckets)
	if err != nil {
		return nil, err
	}
	return listbuckets, nil
}
