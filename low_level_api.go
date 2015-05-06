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

// config - main configuration struct used by all to set endpoint, credentials, and other options for requests.
type config struct {
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string
	ContentType     string
	UserAgent       string
}

type lowLevelAPI struct {
	config *config
}

// putBucketRequest wrapper creates a new PutBucket request
func (a *lowLevelAPI) putBucketRequest(bucket, acl string) (*request, error) {
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "PUT",
		HTTPPath:   "/" + bucket,
	}
	req, err := newRequest(op, a.config, nil)
	if err != nil {
		return nil, err
	}
	req.Set("x-amz-acl", acl)
	return req, nil
}

/// Bucket Write Operations

// putBucket create a new bucket
//
// Requires valid AWS Access Key ID to authenticate requests
// Anonymous requests are never allowed to create buckets
func (a *lowLevelAPI) putBucket(bucket, acl string) error {
	req, err := a.putBucketRequest(bucket, acl)
	if err != nil {
		return err
	}
	resp, err := req.Do()
	if err != nil {
		return err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return responseToError(resp)
		}
	}
	return resp.Body.Close()
}

// putBucketRequestACL wrapper creates a new PutBucketACL request
func (a *lowLevelAPI) putBucketRequestACL(bucket, acl string) (*request, error) {
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "PUT",
		HTTPPath:   "/" + bucket + "?acl",
	}
	req, err := newRequest(op, a.config, nil)
	if err != nil {
		return nil, err
	}
	req.Set("x-amz-acl", acl)
	return req, nil
}

// putBucketACL set the permissions on an existing bucket using access control lists (ACL)
func (a *lowLevelAPI) putBucketACL(bucket, acl string) error {
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
			return responseToError(resp)
		}
	}
	return resp.Body.Close()
}

// listObjectsRequest wrapper creates a new ListObjects request
func (a *lowLevelAPI) listObjectsRequest(bucket string, maxkeys int, marker, prefix, delimiter string) (*request, error) {
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
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "GET",
		HTTPPath:   "/" + bucket + resourceQuery(),
	}
	r, err := newRequest(op, a.config, nil)
	if err != nil {
		return nil, err
	}
	return r, nil
}

/// Bucket Read Operations

// listObjects - (List Objects) - List some or all (up to 1000) of the objects in a bucket.
//
// You can use the request parameters as selection criteria to return a subset of the objects in a bucket.
// request paramters :-
// ---------
// ?marker - Specifies the key to start with when listing objects in a bucket.
// ?delimiter - A delimiter is a character you use to group keys.
// ?prefix - Limits the response to keys that begin with the specified prefix.
// ?max-keys - Sets the maximum number of keys returned in the response body.
func (a *lowLevelAPI) listObjects(bucket string, maxkeys int, marker, prefix, delimiter string) (*ListBucketResult, error) {
	req, err := a.listObjectsRequest(bucket, maxkeys, marker, prefix, delimiter)
	if err != nil {
		return nil, err
	}
	resp, err := req.Do()
	if err != nil {
		return nil, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, responseToError(resp)
		}
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

func (a *lowLevelAPI) headBucketRequest(bucket string) (*request, error) {
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "HEAD",
		HTTPPath:   "/" + bucket,
	}
	return newRequest(op, a.config, nil)
}

// headBucket - useful to determine if a bucket exists and you have permission to access it.
func (a *lowLevelAPI) headBucket(bucket string) error {
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
func (a *lowLevelAPI) deleteBucketRequest(bucket string) (*request, error) {
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "DELETE",
		HTTPPath:   "/" + bucket,
	}
	return newRequest(op, a.config, nil)
}

// deleteBucket - deletes the bucket named in the URI
// NOTE: -
//  All objects (including all object versions and delete markers)
//  in the bucket must be deleted before successfully attempting this request
func (a *lowLevelAPI) deleteBucket(bucket string) error {
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
func (a *lowLevelAPI) putObjectRequest(bucket, object string, size int64, body io.ReadSeeker) (*request, error) {
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "PUT",
		HTTPPath:   "/" + bucket + "/" + object,
	}
	md5Sum, err := contentMD5(body, size)
	if err != nil {
		return nil, err
	}
	r, err := newRequest(op, a.config, ioutil.NopCloser(body))
	if err != nil {
		return nil, err
	}
	r.Set("Content-MD5", md5Sum)
	r.req.ContentLength = size
	return r, nil
}

// putObject - add an object to a bucket
//
// You must have WRITE permissions on a bucket to add an object to it.
func (a *lowLevelAPI) putObject(bucket, object string, size int64, body io.ReadSeeker) error {
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
			return responseToError(resp)
		}
	}
	return resp.Body.Close()
}

// getObjectRequest wrapper creates a new GetObject request
func (a *lowLevelAPI) getObjectRequest(bucket, object string, offset, length uint64) (*request, error) {
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "GET",
		HTTPPath:   "/" + bucket + "/" + object,
	}
	r, err := newRequest(op, a.config, nil)
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

// getObject - retrieve object from Object Storage
//
// Additionally it also takes range arguments to download the specified range bytes of an object.
// For more information about the HTTP Range header, go to http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.35.
func (a *lowLevelAPI) getObject(bucket, object string, offset, length uint64) (io.ReadCloser, int64, string, error) {
	req, err := a.getObjectRequest(bucket, object, offset, length)
	if err != nil {
		return nil, 0, "", err
	}
	resp, err := req.Do()
	if err != nil {
		return nil, 0, "", err
	}
	if resp != nil {
		switch resp.StatusCode {
		case http.StatusOK:
		case http.StatusPartialContent:
		default:
			return nil, 0, "", responseToError(resp)
		}
	}
	md5sum := strings.Trim(resp.Header.Get("ETag"), "\"") // trim off the odd double quotes
	// do not close body here, caller will close
	return resp.Body, resp.ContentLength, md5sum, nil
}

// headObjectRequest wrapper creates a new HeadObject request
func (a *lowLevelAPI) headObjectRequest(bucket, object string) (*request, error) {
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "HEAD",
		HTTPPath:   "/" + bucket + "/" + object,
	}
	return newRequest(op, a.config, nil)
}

// headObject - retrieves metadata from an object without returning the object itself
func (a *lowLevelAPI) headObject(bucket, object string) error {
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
			return responseToError(resp)
		}
	}
	return resp.Body.Close()
}

// deleteObjectRequest wrapper creates a new DeleteObject request
func (a *lowLevelAPI) deleteObjectRequest(bucket, object string) (*request, error) {
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "DELETE",
		HTTPPath:   "/" + bucket + "/" + object,
	}
	return newRequest(op, a.config, nil)
}

// deleteObject removes the object
func (a *lowLevelAPI) deleteObject(bucket, object string) error {
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
func (a *lowLevelAPI) listBucketsRequest() (*request, error) {
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "GET",
		HTTPPath:   "/",
	}
	return newRequest(op, a.config, nil)
}

// listBuckets list of all buckets owned by the authenticated sender of the request
func (a *lowLevelAPI) listBuckets() (*ListAllMyBucketsResult, error) {
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
			return nil, responseToError(resp)
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
