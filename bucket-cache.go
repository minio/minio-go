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
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
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
		bucketRegion:     "us-east-1",
		credentials:      a.credentials,
		contentTransport: a.httpTransport,
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
