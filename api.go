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
	"net/http"
	"net/url"
	"time"
)

// API implements Amazon S3 compatible methods.
type API struct {
	// User supplied.
	userAgent   string
	credentials *clientCredentials
	endpointURL *url.URL

	// This http transport is usually needed for debugging OR to add your own
	// custom TLS certificates on the client transport, for custom CA's and
	// certs which are not part of standard certificate authority.
	httpTransport http.RoundTripper

	// Needs allocation.
	bucketRgnC *bucketRegionCache
}

// NewV2 - instantiate minio client with Amazon S3 signature version '2' compatiblity.
func NewV2(endpoint string, accessKeyID, secretAccessKey string, inSecure bool) (API, error) {
	// construct endpoint.
	endpointURL, err := getEndpointURL(endpoint, inSecure)
	if err != nil {
		return API{}, err
	}

	// create a new client Config.
	credentials := &clientCredentials{}
	credentials.AccessKeyID = accessKeyID
	credentials.SecretAccessKey = secretAccessKey
	credentials.Signature = SignatureV2

	// instantiate new API.
	api := API{
		// Save for lower level calls.
		userAgent:   libraryUserAgent,
		credentials: credentials,
		endpointURL: endpointURL,
		// Allocate.
		bucketRgnC: newBucketRegionCache(),
	}
	return api, nil
}

// NewV4 - instantiate minio client with Amazon S3 signature version '4' compatibility.
func NewV4(endpoint string, accessKeyID, secretAccessKey string, inSecure bool) (API, error) {
	// construct endpoint.
	endpointURL, err := getEndpointURL(endpoint, inSecure)
	if err != nil {
		return API{}, err
	}

	// create a new client Config.
	credentials := &clientCredentials{}
	credentials.AccessKeyID = accessKeyID
	credentials.SecretAccessKey = secretAccessKey
	credentials.Signature = SignatureV4

	// instantiate new API.
	api := API{
		// Save for lower level calls.
		userAgent:   libraryUserAgent,
		credentials: credentials,
		endpointURL: endpointURL,
		// Allocate.
		bucketRgnC: newBucketRegionCache(),
	}
	return api, nil
}

// New - instantiate minio client API, adds automatic verification of signature.
func New(endpoint string, accessKeyID, secretAccessKey string, inSecure bool) (API, error) {
	// construct endpoint.
	endpointURL, err := getEndpointURL(endpoint, inSecure)
	if err != nil {
		return API{}, err
	}

	// create a new client Config.
	credentials := &clientCredentials{}
	credentials.AccessKeyID = accessKeyID
	credentials.SecretAccessKey = secretAccessKey

	// Google cloud storage should be set to signature V2, force it if not.
	if isGoogleEndpoint(endpointURL) {
		credentials.Signature = SignatureV2
	}
	// If Amazon S3 set to signature v2.
	if isAmazonEndpoint(endpointURL) {
		credentials.Signature = SignatureV4
	}

	// instantiate new API.
	api := API{
		// Save for lower level calls.
		userAgent:   libraryUserAgent,
		credentials: credentials,
		endpointURL: endpointURL,
		// Allocate.
		bucketRgnC: newBucketRegionCache(),
	}
	return api, nil
}

// SetAppInfo - add application details to user agent.
func (a *API) SetAppInfo(appName string, appVersion string) {
	// if app name and version is not set, we do not a new user agent.
	if appName != "" && appVersion != "" {
		appUserAgent := appName + "/" + appVersion
		a.userAgent = libraryUserAgent + " " + appUserAgent
	}
}

// SetCustomTransport - set new custom transport.
func (a *API) SetCustomTransport(customHTTPTransport http.RoundTripper) {
	// Set this to override default transport ``http.DefaultTransport``.
	//
	// This transport is usually needed for debugging OR to add your own
	// custom TLS certificates on the client transport, for custom CA's and
	// certs which are not part of standard certificate authority follow this
	// example :-
	//
	//   tr := &http.Transport{
	//           TLSClientConfig:    &tls.Config{RootCAs: pool},
	//           DisableCompression: true,
	//   }
	//   api.SetTransport(tr)
	//
	a.httpTransport = customHTTPTransport
}

// CloudStorageAPI - Cloud Storage API interface.
type CloudStorageAPI interface {
	// Bucket Read/Write/Stat operations.
	MakeBucket(bucket string, cannedACL BucketACL, location string) error
	BucketExists(bucket string) error
	RemoveBucket(bucket string) error
	SetBucketACL(bucket string, cannedACL BucketACL) error
	GetBucketACL(bucket string) (BucketACL, error)

	ListBuckets() <-chan BucketStat
	ListObjects(bucket, prefix string, recursive bool) <-chan ObjectStat
	ListIncompleteUploads(bucket, prefix string, recursive bool) <-chan ObjectMultipartStat

	// Object Read/Write/Stat operations.
	GetObject(bucket, object string) (io.ReadSeeker, error)
	GetPartialObject(bucket, object string, offset, length int64) (io.ReadSeeker, error)
	PutObject(bucket, object string, data io.ReadSeeker, size int64, contentType string) (int64, error)
	StatObject(bucket, object string) (ObjectStat, error)
	RemoveObject(bucket, object string) error
	RemoveIncompleteUpload(bucket, object string) <-chan error

	// Presigned operations.
	PresignedGetObject(bucket, object string, expires time.Duration) (string, error)
	PresignedPutObject(bucket, object string, expires time.Duration) (string, error)
	PresignedPostPolicy(*PostPolicy) (map[string]string, error)

	// Application info.
	SetAppInfo(appName, appVersion string)

	// Set custom transport.
	SetCustomTransport(customTransport http.RoundTripper)
}
