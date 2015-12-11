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
	"errors"
	"io"
	"net/http"
	"time"
)

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

/// Bucket operations

// MakeBucket makes a new bucket.
//
// Optional arguments are acl and location - by default all buckets are created
// with ``private`` acl and in US Standard region.
//
// ACL valid values
//
//  private - owner gets full access [default].
//  public-read - owner gets full access, all others get read access.
//  public-read-write - owner gets full access, all others get full access too.
//  authenticated-read - owner gets full access, authenticated users get read access.
//
// Region valid values.
// ------------------
// [ us-west-1 | us-west-2 | eu-west-1 | eu-central-1 | ap-southeast-1 | ap-northeast-1 | ap-southeast-2 | sa-east-1 ]
// Defaults to US standard
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
	return a.putBucket(bucketName, string(acl), region)
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
	return a.putBucketACL(bucketName, string(acl))
}

// GetBucketACL get the permissions on an existing bucket.
//
// Returned values are:
//
//  private - owner gets full access.
//  public-read - owner gets full access, others get read access.
//  public-read-write - owner gets full access, others get full access too.
//  authenticated-read - owner gets full access, authenticated users get read access.
func (a API) GetBucketACL(bucketName string) (BucketACL, error) {
	if err := isValidBucketName(bucketName); err != nil {
		return "", err
	}
	policy, err := a.getBucketACL(bucketName)
	if err != nil {
		return "", err
	}
	// boolean cues to indentify right canned acls.
	var publicRead, publicWrite bool

	// Handle grants.
	grants := policy.AccessControlList.Grant
	for _, g := range grants {
		if g.Grantee.URI == "" && g.Permission == "FULL_CONTROL" {
			continue
		}
		if g.Grantee.URI == "http://acs.amazonaws.com/groups/global/AuthenticatedUsers" && g.Permission == "READ" {
			return BucketACL("authenticated-read"), nil
		} else if g.Grantee.URI == "http://acs.amazonaws.com/groups/global/AllUsers" && g.Permission == "WRITE" {
			publicWrite = true
		} else if g.Grantee.URI == "http://acs.amazonaws.com/groups/global/AllUsers" && g.Permission == "READ" {
			publicRead = true
		}
	}

	// public write and not enabled. return.
	if !publicWrite && !publicRead {
		return BucketACL("private"), nil
	}
	// public write not enabled but public read is. return.
	if !publicWrite && publicRead {
		return BucketACL("public-read"), nil
	}
	// public read and public write are enabled return.
	if publicRead && publicWrite {
		return BucketACL("public-read-write"), nil
	}

	return "", ErrorResponse{
		Code:       "NoSuchBucketPolicy",
		Message:    "The specified bucket does not have a bucket policy.",
		BucketName: bucketName,
		RequestID:  "minio",
	}
}

// BucketExists verify if bucket exists and you have permission to access it.
func (a API) BucketExists(bucketName string) error {
	if err := isValidBucketName(bucketName); err != nil {
		return err
	}
	return a.headBucket(bucketName)
}

// RemoveBucket deletes the bucket name.
//
//  All objects (including all object versions and delete markers).
//  in the bucket must be deleted before successfully attempting this request.
func (a API) RemoveBucket(bucketName string) error {
	if err := isValidBucketName(bucketName); err != nil {
		return err
	}
	return a.deleteBucket(bucketName)
}

// ListBuckets list of all buckets owned by the authenticated sender of the request.
//
// This call requires explicit authentication, no anonymous requests are
// allowed for listing buckets.
//
//   api := client.New(....)
//   for message := range api.ListBuckets() {
//       fmt.Println(message)
//   }
//
func (a API) ListBuckets() <-chan BucketStat {
	ch := make(chan BucketStat, 100)
	go a.listBucketsInRoutine(ch)
	return ch
}

// ListObjects - (List Objects) - List some objects or all recursively.
//
// ListObjects is a channel based API implemented to facilitate ease
// of usage of S3 API ListObjects() by automatically recursively
// traversing all objects on a given bucket if specified.
//
// Your input paramters are just bucketName, prefix and recursive. If you
// enable recursive as 'true' this function will return back all the
// objects in a given bucket name.
//
//   api := client.New(....)
//   recursive := true
//   for message := range api.ListObjects("mytestbucket", "starthere", recursive) {
//       fmt.Println(message)
//   }
//
func (a API) ListObjects(bucketName string, prefix string, recursive bool) <-chan ObjectStat {
	ch := make(chan ObjectStat, 1000)
	go a.listObjectsInRoutine(bucketName, prefix, recursive, ch)
	return ch
}

// ListIncompleteUploads - List incompletely uploaded multipart objects.
//
// ListIncompleteUploads is a channel based API implemented to facilitate
// ease of usage of S3 API ListMultipartUploads() by automatically
// recursively traversing all multipart objects on a given bucket if specified.
//
// Your input paramters are just bucketName, prefix and recursive.
// If you enable recursive as 'true' this function will return back all
// the multipart objects in a given bucket name.
//
//   api := client.New(....)
//   recursive := true
//   for message := range api.ListIncompleteUploads("mytestbucket", "starthere", recursive) {
//       fmt.Println(message)
//   }
//
func (a API) ListIncompleteUploads(bucketName, prefix string, recursive bool) <-chan ObjectMultipartStat {
	return a.listIncompleteUploads(bucketName, prefix, recursive)
}

// GetObject retrieve object. retrieves full object, if you need ranges use GetPartialObject.
func (a API) GetObject(bucketName, objectName string) (io.ReadSeeker, error) {
	if err := isValidBucketName(bucketName); err != nil {
		return nil, err
	}
	if err := isValidObjectName(objectName); err != nil {
		return nil, err
	}
	// get object.
	return newObjectReadSeeker(a, bucketName, objectName), nil
}

// GetPartialObject retrieve partial object.
//
// Takes range arguments to download the specified range bytes of an object.
// Setting offset and length = 0 will download the full object.
// For more information about the HTTP Range header,
// go to http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.35
func (a API) GetPartialObject(bucketName, objectName string, offset, length int64) (io.ReadSeeker, error) {
	if err := isValidBucketName(bucketName); err != nil {
		return nil, err
	}
	if err := isValidObjectName(objectName); err != nil {
		return nil, err
	}
	// get partial object.
	return newObjectReadSeeker(a, bucketName, objectName), nil
}

// PutObject create an object in a bucket.
//
// You must have WRITE permissions on a bucket to create an object.
//
//  - For size lesser than 5MB PutObject automatically does single Put operation.
//  - For size equal to 0Bytes PutObject automatically does single Put operation.
//  - For size larger than 5MB PutObject automatically does resumable multipart operation.
//  - For size input as -1 PutObject treats it as a stream and does multipart operation until
//    input stream reaches EOF. Maximum object size that can be uploaded through this operation
//    will be 5TB.
//
// NOTE: if you are using Google Cloud Storage. Then there is no resumable multipart
// upload support yet. Currently PutObject will behave like a single PUT operation and would
// only upload for file sizes upto maximum 5GB. (maximum limit for single PUT operation).
//
// For un-authenticated requests S3 doesn't allow multipart upload, so we fall back to single PUT operation.
func (a API) PutObject(bucketName, objectName string, data io.ReadSeeker, size int64, contentType string) error {
	if err := isValidBucketName(bucketName); err != nil {
		return err
	}
	if err := isValidObjectName(objectName); err != nil {
		return err
	}
	// NOTE: S3 doesn't allow anonymous multipart requests.
	if isAmazonEndpoint(a.endpointURL) && isAnonymousCredentials(*a.credentials) {
		if size <= -1 {
			return ErrorResponse{
				Code:       "NotImplemented",
				Message:    "For anonymous requests Content-Length cannot be negative.",
				Key:        objectName,
				BucketName: bucketName,
			}
		}
		// Do not compute MD5 for anonymous requests to Amazon S3. Uploads upto 5GB in size.
		return a.putNoChecksum(bucketName, objectName, data, size, contentType)
	}
	// FIXME: we should remove this in future when we fully implement
	// resumable object upload for Google Cloud Storage.
	if isGoogleEndpoint(a.endpointURL) {
		if size <= -1 {
			return ErrorResponse{
				Code:       "NotImplemented",
				Message:    "Content-Length cannot be negative for file uploads to Google Cloud Storage.",
				Key:        objectName,
				BucketName: bucketName,
			}
		}
		// Do not compute MD5 for Google Cloud Storage. Uploads upto 5GB in size.
		return a.putNoChecksum(bucketName, objectName, data, size, contentType)
	}
	// Large file upload is initiated for uploads for input data size
	// if its greater than 5MB or data size is negative.
	if size >= minimumPartSize || size < 0 {
		return a.putLargeObject(bucketName, objectName, data, size, contentType)
	}
	return a.putSmallObject(bucketName, objectName, data, size, contentType)
}

// StatObject verify if object exists and you have permission to access it.
func (a API) StatObject(bucketName, objectName string) (ObjectStat, error) {
	if err := isValidBucketName(bucketName); err != nil {
		return ObjectStat{}, err
	}
	if err := isValidObjectName(objectName); err != nil {
		return ObjectStat{}, err
	}
	return a.headObject(bucketName, objectName)
}

// RemoveObject remove an object from a bucket.
func (a API) RemoveObject(bucketName, objectName string) error {
	if err := isValidBucketName(bucketName); err != nil {
		return err
	}
	if err := isValidObjectName(objectName); err != nil {
		return err
	}
	return a.deleteObject(bucketName, objectName)
}

// RemoveIncompleteUpload - abort a specific in progress active multipart upload.
// Requires explicit authentication, no anonymous requests are allowed for multipart API.
func (a API) RemoveIncompleteUpload(bucketName, objectName string) <-chan error {
	errorCh := make(chan error)
	go a.removeIncompleteUploadInRoutine(bucketName, objectName, errorCh)
	return errorCh
}

// PresignedGetObject get a presigned URL to get an object for third party apps.
func (a API) PresignedGetObject(bucketName, objectName string, expires time.Duration) (string, error) {
	if err := isValidExpiry(expires); err != nil {
		return "", err
	}
	expireSeconds := int64(expires / time.Second)
	return a.presignedGetObject(bucketName, objectName, expireSeconds, 0, 0)
}

// PresignedPutObject get a presigned URL to upload an object.
// Expires maximum is 7days - ie. 604800 and minimum is 1.
func (a API) PresignedPutObject(bucketName, objectName string, expires time.Duration) (string, error) {
	if err := isValidExpiry(expires); err != nil {
		return "", err
	}
	expireSeconds := int64(expires / time.Second)
	return a.presignedPutObject(bucketName, objectName, expireSeconds)
}

// PresignedPostPolicy return POST form data that can be used for object upload.
func (a API) PresignedPostPolicy(p *PostPolicy) (map[string]string, error) {
	if p.expiration.IsZero() {
		return nil, errors.New("Expiration time must be specified")
	}
	if _, ok := p.formData["key"]; !ok {
		return nil, errors.New("object key must be specified")
	}
	if _, ok := p.formData["bucket"]; !ok {
		return nil, errors.New("bucket name must be specified")
	}
	return a.presignedPostPolicy(p)
}
