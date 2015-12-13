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
	"net/http"
	"net/url"
	"path/filepath"
	"sync"
)

// bucketRegionCache provides simple mechansim to hold bucket regions in memory.
type bucketRegionCache struct {
	// Mutex is used for handling the concurrent
	// read/write requests for cache
	sync.RWMutex

	// items holds the cached regions.
	items map[string]string
}

// newBucketRegionCache provides a new bucket region cache to be used
// internally with the client object.
func newBucketRegionCache() *bucketRegionCache {
	return &bucketRegionCache{
		items: make(map[string]string),
	}
}

// Get returns a value of a given key if it exists
func (r *bucketRegionCache) Get(bucketName string) (region string, ok bool) {
	r.RLock()
	defer r.RUnlock()
	region, ok = r.items[bucketName]
	return
}

// Set will persist a value to the cache
func (r *bucketRegionCache) Set(bucketName string, region string) {
	r.Lock()
	defer r.Unlock()
	r.items[bucketName] = region
}

// Delete deletes a bucket name.
func (r *bucketRegionCache) Delete(bucketName string) {
	r.Lock()
	defer r.Unlock()
	delete(r.items, bucketName)
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
	if region, ok := a.bucketRgnC.Get(bucketName); ok {
		return region, nil
	}

	// get bucket location.
	location, err := a.getBucketLocation(bucketName)
	if err != nil {
		return "", err
	}
	region := "us-east-1"
	// location is region in context of S3 API.
	if location != "" {
		region = location
	}
	a.bucketRgnC.Set(bucketName, region)
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
	defer closeResponse(resp)
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
