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
	"strings"
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

// putBucketRequest wrapper creates a new PutBucket request
func (a *api) putBucketRequest(bucket string) (*Request, error) {
	op := &Operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "PUT",
		HTTPPath:   "/" + bucket,
	}
	return NewRequest(op, a.config, nil)
}

/// Bucket Write Operations

// PutBucket create a new bucket
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

// putBucketRequestACL wrapper creates a new PutBucketACL request
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

// PutBucketACL set the permissions on an existing bucket using access control lists (ACL)
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

// listObjectsRequest wrapper creates a new ListObjects request
func (a *api) listObjectsRequest(bucket string, maxkeys int, marker, prefix, delimiter string) (*Request, error) {
	// resourceQuery - get resources properly escaped and lined up before using them in http request
	resourceQuery := func() string {
		switch {
		case marker != "":
			marker = fmt.Sprintf("&marker=%s", url.QueryEscape(marker))
		case prefix != "":
			prefix = fmt.Sprintf("&prefix=%s", url.QueryEscape(prefix))
		case delimiter != "":
			delimiter = fmt.Sprintf("&delimiter=%s", url.QueryEscape(delimiter))
		}
		return fmt.Sprintf("?max-keys=%d", maxkeys) + marker + prefix + delimiter
	}
	op := &Operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "GET",
		HTTPPath:   "/" + bucket + resourceQuery(),
	}
	r, err := NewRequest(op, a.config, nil)
	if err != nil {
		return nil, err
	}
	return r, nil
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
func (a *api) ListObjects(bucket string, maxkeys int, marker, prefix, delimiter string) (*ListBucketResult, error) {
	req, err := a.listObjectsRequest(bucket, maxkeys, marker, prefix, delimiter)
	if err != nil {
		return nil, err
	}
	resp, err := req.Do()
	if err != nil {
		return nil, err
	}

	listBucketResult := new(ListBucketResult)
	decoder := xml.NewDecoder(resp.Body)
	err = decoder.Decode(listBucketResult)
	if err != nil {
		return nil, err
	}

	// close body while returning, along with any error
	return listBucketResult, resp.Body.Close()
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
			// Head has no response body, handle it
			return fmt.Errorf("%s", resp.Status)
		}
	}
	return resp.Body.Close()
}

// deleteBucketRequest wrapper creates a new DeleteBucket request
func (a *api) deleteBucketRequest(bucket string) (*Request, error) {
	op := &Operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "DELETE",
		HTTPPath:   "/" + bucket,
	}
	return NewRequest(op, a.config, nil)
}

// DeleteBucket - deletes the bucket named in the URI
// NOTE: -
//  All objects (including all object versions and delete markers)
//  in the bucket must be deleted before successfully attempting this request
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

// putObjectRequest wrapper creates a new PutObject request
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
	r, err := NewRequest(op, a.config, ioutil.NopCloser(body))
	if err != nil {
		return nil, err
	}
	r.Set("Content-MD5", md5Sum)
	r.req.ContentLength = size
	return r, nil
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

// getObjectRequest wrapper creates a new GetObject request
func (a *api) getObjectRequest(bucket, object string, offset, length uint64) (*Request, error) {
	op := &Operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "GET",
		HTTPPath:   "/" + bucket + "/" + object,
	}
	r, err := NewRequest(op, a.config, nil)
	if err != nil {
		return nil, err
	}
	// TODO - fix this to support full - http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html
	switch {
	case length > 0:
		r.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, offset+length-1))
	default:
		r.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	return r, nil
}

// GetObject - retrieve object from Object Storage
//
// Additionally it also takes range arguments to download the specified range bytes of an object.
// For more information about the HTTP Range header, go to http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.35.
func (a *api) GetObject(bucket, object string, offset, length uint64) (io.ReadCloser, int64, string, error) {
	req, err := a.getObjectRequest(bucket, object, offset, length)
	if err != nil {
		return nil, 0, "", err
	}
	resp, err := req.Do()
	if err != nil {
		return nil, 0, "", err
	}
	md5sum := strings.Trim(resp.Header.Get("ETag"), "\"") // trim off the odd double quotes
	// do not close body here, caller will close
	return resp.Body, resp.ContentLength, md5sum, nil
}

// headObjectRequest wrapper creates a new HeadObject request
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
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return ResponseToError(resp)
		}
	}
	return resp.Body.Close()
}

// deleteObjectRequest wrapper creates a new DeleteObject request
func (a *api) deleteObjectRequest(bucket, object string) (*Request, error) {
	op := &Operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "DELETE",
		HTTPPath:   "/" + bucket + "/" + object,
	}
	return NewRequest(op, a.config, nil)
}

// DeleteObject removes the object
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

// listBucketRequest wrapper creates a new ListBuckets request
func (a *api) listBucketsRequest() (*Request, error) {
	op := &Operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "GET",
		HTTPPath:   "/",
	}
	return NewRequest(op, a.config, nil)
}

// ListBuckets list of all buckets owned by the authenticated sender of the request
func (a *api) ListBuckets() (*ListAllMyBucketsResult, error) {
	req, err := a.listBucketsRequest()
	if err != nil {
		return nil, err
	}
	resp, err := req.Do()
	if err != nil {
		return nil, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, ResponseToError(resp)
		}
	}
	listAllMyBucketsResult := new(ListAllMyBucketsResult)
	decoder := xml.NewDecoder(resp.Body)
	err = decoder.Decode(listAllMyBucketsResult)
	if err != nil {
		return nil, err
	}
	return listAllMyBucketsResult, resp.Body.Close()
}
