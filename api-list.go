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
	"fmt"
	"net/http"
	"net/url"
)

// ListBuckets list all buckets owned by this authenticated user.
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

// listBucketsInRoutine goroutine based iterator for listBuckets.
func (a API) listBucketsInRoutine(ch chan<- BucketStat) {
	defer close(ch)
	req, err := a.listBucketsRequest()
	if err != nil {
		ch <- BucketStat{
			Err: err,
		}
		return
	}
	resp, err := req.Do()
	defer closeResponse(resp)
	if err != nil {
		ch <- BucketStat{
			Err: err,
		}
		return
	}
	if resp != nil {
		// for un-authenticated requests, amazon sends a redirect handle it.
		if resp.StatusCode == http.StatusTemporaryRedirect {
			ch <- BucketStat{
				Err: ErrorResponse{
					Code:            "AccessDenied",
					Message:         "Anonymous access is forbidden for this operation.",
					RequestID:       resp.Header.Get("x-amz-request-id"),
					HostID:          resp.Header.Get("x-amz-id-2"),
					AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
				},
			}
			return
		}
		if resp.StatusCode != http.StatusOK {
			ch <- BucketStat{
				Err: BodyToErrorResponse(resp.Body),
			}
			return
		}
	}
	listAllMyBucketsResult := listAllMyBucketsResult{}
	err = xmlDecoder(resp.Body, &listAllMyBucketsResult)
	if err != nil {
		ch <- BucketStat{
			Err: err,
		}
		return
	}

	for _, bucket := range listAllMyBucketsResult.Buckets.Bucket {
		ch <- bucket
	}
}

// listBucketRequest wrapper creates a new listBuckets request.
func (a API) listBucketsRequest() (*Request, error) {
	// List buckets is directly on the endpoint URL.
	targetURL := a.endpointURL
	targetURL.Path = "/"

	// Instantiate a new request.
	req, err := newRequest("GET", targetURL, requestMetadata{
		credentials:      a.credentials,
		userAgent:        a.userAgent,
		bucketRegion:     "us-east-1",
		contentTransport: a.httpTransport,
	})
	if err != nil {
		return nil, err
	}
	return req, nil
}

// ListObjects - (List Objects) - List some objects or all recursively.
//
// ListObjects lists all objects matching the objectPrefix from
// the specified bucket. If recursion is enabled it would list
// all subdirectories and all its contents.
//
// Your input paramters are just bucketName, objectPrefix and recursive. If you
// enable recursive as 'true' this function will return back all the
// objects in a given bucket name and object prefix.
//
//   api := client.New(....)
//   recursive := true
//   for message := range api.ListObjects("mytestbucket", "starthere", recursive) {
//       fmt.Println(message)
//   }
//
func (a API) ListObjects(bucketName string, objectPrefix string, recursive bool) <-chan ObjectStat {
	ch := make(chan ObjectStat, 1000)
	go a.listObjectsInRoutine(bucketName, objectPrefix, recursive, ch)
	return ch
}

// listObjectsRecursive lists all objects recursively matching a prefix.
func (a API) listObjectsRecursive(bucketName, objectPrefix string, ch chan<- ObjectStat) {
	var objectMarker string
	for {
		result, err := a.listObjects(bucketName, objectPrefix, objectMarker, "", 1000)
		if err != nil {
			ch <- ObjectStat{
				Err: err,
			}
			return
		}
		for _, object := range result.Contents {
			ch <- object
			objectMarker = object.Key
		}
		if !result.IsTruncated {
			break
		}
	}
}

// listObjectsNonRecursive lists objects delimited with "/" matching a prefix.
func (a API) listObjectsNonRecursive(bucketName, objectPrefix string, ch chan<- ObjectStat) {
	// Non recursive delimit with "/".
	var objectMarker string
	for {
		result, err := a.listObjects(bucketName, objectPrefix, objectMarker, "/", 1000)
		if err != nil {
			ch <- ObjectStat{
				Err: err,
			}
			return
		}
		objectMarker = result.NextMarker
		for _, object := range result.Contents {
			ch <- object
		}
		for _, obj := range result.CommonPrefixes {
			object := ObjectStat{}
			object.Key = obj.Prefix
			object.Size = 0
			ch <- object
		}
		if !result.IsTruncated {
			break
		}
	}
}

// listObjectsInRoutine goroutine based iterator for listObjects.
func (a API) listObjectsInRoutine(bucketName, objectPrefix string, recursive bool, ch chan<- ObjectStat) {
	defer close(ch)
	// Validate bucket name.
	if err := isValidBucketName(bucketName); err != nil {
		ch <- ObjectStat{
			Err: err,
		}
		return
	}
	// Validate incoming object prefix.
	if err := isValidObjectPrefix(objectPrefix); err != nil {
		ch <- ObjectStat{
			Err: err,
		}
		return
	}
	// Recursive do not delimit.
	if recursive {
		a.listObjectsRecursive(bucketName, objectPrefix, ch)
		return
	}
	a.listObjectsNonRecursive(bucketName, objectPrefix, ch)
	return
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
	req, err := newRequest("GET", targetURL, requestMetadata{
		credentials:      a.credentials,
		userAgent:        a.userAgent,
		bucketRegion:     region,
		contentTransport: a.httpTransport,
	})
	if err != nil {
		return nil, err
	}
	return req, nil
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
	defer closeResponse(resp)
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

// ListIncompleteUploads - List incompletely uploaded multipart objects.
//
// ListIncompleteUploads lists all incompleted objects matching the
// objectPrefix from the specified bucket. If recursion is enabled
// it would list all subdirectories and all its contents.
//
// Your input paramters are just bucketName, objectPrefix and recursive.
// If you enable recursive as 'true' this function will return back all
// the multipart objects in a given bucket name.
//
//   api := client.New(....)
//   recursive := true
//   for message := range api.ListIncompleteUploads("mytestbucket", "starthere", recursive) {
//       fmt.Println(message)
//   }
//
func (a API) ListIncompleteUploads(bucketName, objectPrefix string, recursive bool) <-chan ObjectMultipartStat {
	return a.listIncompleteUploads(bucketName, objectPrefix, recursive)
}

// listIncompleteUploads lists all incomplete uploads.
func (a API) listIncompleteUploads(bucketName, objectName string, recursive bool) <-chan ObjectMultipartStat {
	ch := make(chan ObjectMultipartStat, 1000)
	go a.listIncompleteUploadsInRoutine(bucketName, objectName, recursive, ch)
	return ch
}

// listIncompleteUploadsRecursive list incomplete uploads matching a prefix recursively.
func (a API) listIncompleteUploadsRecursive(bucketName, objectPrefix string, ch chan<- ObjectMultipartStat) {
	var objectMarker string
	var uploadIDMarker string
	for {
		result, err := a.listMultipartUploads(bucketName, objectMarker, uploadIDMarker, objectPrefix, "", 1000)
		if err != nil {
			ch <- ObjectMultipartStat{
				Err: err,
			}
			return
		}
		for _, objectSt := range result.Uploads {
			// NOTE: getTotalMultipartSize can make listing incomplete uploads slower.
			objectSt.Size, err = a.getTotalMultipartSize(bucketName, objectSt.Key, objectSt.UploadID)
			if err != nil {
				ch <- ObjectMultipartStat{
					Err: err,
				}
			}
			ch <- objectSt
			objectMarker = result.NextKeyMarker
			uploadIDMarker = result.NextUploadIDMarker
		}
		if !result.IsTruncated {
			break
		}
	}
	return
}

// listIncompleteUploadsNonRecursive list incomplete uploads delimited at "/" matching a prefix.
func (a API) listIncompleteUploadsNonRecursive(bucketName, objectPrefix string, ch chan<- ObjectMultipartStat) {
	// Non recursive with "/" delimiter.
	var objectMarker string
	var uploadIDMarker string
	for {
		result, err := a.listMultipartUploads(bucketName, objectMarker, uploadIDMarker, objectPrefix, "/", 1000)
		if err != nil {
			ch <- ObjectMultipartStat{
				Err: err,
			}
			return
		}
		objectMarker = result.NextKeyMarker
		uploadIDMarker = result.NextUploadIDMarker
		for _, objectSt := range result.Uploads {
			objectSt.Size, err = a.getTotalMultipartSize(bucketName, objectSt.Key, objectSt.UploadID)
			if err != nil {
				ch <- ObjectMultipartStat{
					Err: err,
				}
			}
			ch <- objectSt
		}
		for _, obj := range result.CommonPrefixes {
			object := ObjectMultipartStat{}
			object.Key = obj.Prefix
			object.Size = 0
			ch <- object
		}
		if !result.IsTruncated {
			break
		}
	}
}

// listIncompleteUploadsInRoutine goroutine based iterator for listing all incomplete uploads.
func (a API) listIncompleteUploadsInRoutine(bucketName, objectPrefix string, recursive bool, ch chan<- ObjectMultipartStat) {
	defer close(ch)
	// Validate incoming bucket name.
	if err := isValidBucketName(bucketName); err != nil {
		ch <- ObjectMultipartStat{
			Err: err,
		}
		return
	}
	// Validate incoming object prefix.
	if err := isValidObjectPrefix(objectPrefix); err != nil {
		ch <- ObjectMultipartStat{
			Err: err,
		}
		return
	}
	// Recursive with no delimiter.
	if recursive {
		a.listIncompleteUploadsRecursive(bucketName, objectPrefix, ch)
		return
	}
	a.listIncompleteUploadsNonRecursive(bucketName, objectPrefix, ch)
	return
}

// listMultipartUploadsRequest wrapper creates a new listMultipartUploads request.
func (a API) listMultipartUploadsRequest(bucketName, keyMarker, uploadIDMarker,
	prefix, delimiter string, maxUploads int) (*Request, error) {
	// Get resources properly escaped and lined up before using them in http request.
	urlValues := make(url.Values)
	// Set uploads.
	urlValues.Set("uploads", "")
	// Set object key marker.
	urlValues.Set("key-marker", urlEncodePath(keyMarker))
	// Set upload id marker.
	urlValues.Set("upload-id-marker", uploadIDMarker)
	// Set prefix marker.
	urlValues.Set("prefix", urlEncodePath(prefix))
	// Set delimiter.
	urlValues.Set("delimiter", delimiter)
	// Set max-uploads.
	urlValues.Set("max-uploads", fmt.Sprintf("%d", maxUploads))

	// get targetURL.
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
	return newRequest("GET", targetURL, requestMetadata{
		credentials:      a.credentials,
		userAgent:        a.userAgent,
		bucketRegion:     region,
		contentTransport: a.httpTransport,
	})
}

// listMultipartUploads - (List Multipart Uploads).
//   - Lists some or all (up to 1000) in-progress multipart uploads in a bucket.
//
// You can use the request parameters as selection criteria to return a subset of the uploads in a bucket.
// request paramters. :-
// ---------
// ?key-marker - Specifies the multipart upload after which listing should begin.
// ?upload-id-marker - Together with key-marker specifies the multipart upload after which listing should begin.
// ?delimiter - A delimiter is a character you use to group keys.
// ?prefix - Limits the response to keys that begin with the specified prefix.
// ?max-uploads - Sets the maximum number of multipart uploads returned in the response body.
func (a API) listMultipartUploads(bucketName, keyMarker,
	uploadIDMarker, prefix, delimiter string, maxUploads int) (listMultipartUploadsResult, error) {
	req, err := a.listMultipartUploadsRequest(bucketName,
		keyMarker, uploadIDMarker, prefix, delimiter, maxUploads)
	if err != nil {
		return listMultipartUploadsResult{}, err
	}
	resp, err := req.Do()
	defer closeResponse(resp)
	if err != nil {
		return listMultipartUploadsResult{}, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return listMultipartUploadsResult{}, BodyToErrorResponse(resp.Body)
		}
	}
	listMultipartUploadsResult := listMultipartUploadsResult{}
	err = xmlDecoder(resp.Body, &listMultipartUploadsResult)
	if err != nil {
		return listMultipartUploadsResult, err
	}
	return listMultipartUploadsResult, nil
}

// listObjectPartsRecursive list all object parts recursively.
func (a API) listObjectPartsRecursive(bucketName, objectName, uploadID string) <-chan objectPartMetadata {
	objectPartCh := make(chan objectPartMetadata, 1000)
	go a.listObjectPartsRecursiveInRoutine(bucketName, objectName, uploadID, objectPartCh)
	return objectPartCh
}

// listObjectPartsRecursiveInRoutine gorountine based iterator for listing all object parts.
func (a API) listObjectPartsRecursiveInRoutine(bucketName, objectName, uploadID string, ch chan<- objectPartMetadata) {
	defer close(ch)
	listObjPartsResult, err := a.listObjectParts(bucketName, objectName, uploadID, 0, 1000)
	if err != nil {
		ch <- objectPartMetadata{
			Err: err,
		}
		return
	}
	for _, uploadedObjectPart := range listObjPartsResult.ObjectParts {
		ch <- uploadedObjectPart
	}
	// listObject parts.
	for {
		if !listObjPartsResult.IsTruncated {
			break
		}
		nextPartNumberMarker := listObjPartsResult.NextPartNumberMarker
		listObjPartsResult, err = a.listObjectParts(bucketName, objectName, uploadID, nextPartNumberMarker, 1000)
		if err != nil {
			ch <- objectPartMetadata{
				Err: err,
			}
			return
		}
		for _, uploadedObjectPart := range listObjPartsResult.ObjectParts {
			ch <- uploadedObjectPart
		}
	}
}

// getTotalMultipartSize - calculate total uploaded size for the a given multipart object.
func (a API) getTotalMultipartSize(bucketName, objectName, uploadID string) (int64, error) {
	var size int64
	// Iterate over all parts and aggregate the size.
	for part := range a.listObjectPartsRecursive(bucketName, objectName, uploadID) {
		if part.Err != nil {
			return 0, part.Err
		}
		size += part.Size
	}
	return size, nil
}

// listObjectPartsRequest wrapper creates a new ListObjectParts request.
func (a API) listObjectPartsRequest(bucketName, objectName, uploadID string, partNumberMarker, maxParts int) (*Request, error) {
	// Get resources properly escaped and lined up before using them in http request.
	urlValues := make(url.Values)
	// Set part number marker.
	urlValues.Set("part-number-marker", fmt.Sprintf("%d", partNumberMarker))
	// Set upload id.
	urlValues.Set("uploadId", uploadID)
	// Set max parts.
	urlValues.Set("max-parts", fmt.Sprintf("%d", maxParts))

	// get targetURL.
	targetURL, err := getTargetURL(a.endpointURL, bucketName, objectName, urlValues)
	if err != nil {
		return nil, err
	}

	// get bucket region.
	region, err := a.getRegion(bucketName)
	if err != nil {
		return nil, err
	}

	req, err := newRequest("GET", targetURL, requestMetadata{
		credentials:      a.credentials,
		userAgent:        a.userAgent,
		bucketRegion:     region,
		contentTransport: a.httpTransport,
	})
	if err != nil {
		return nil, err
	}
	return req, nil
}

// listObjectParts (List Parts)
//     - lists some or all (up to 1000) parts that have been uploaded for a specific multipart upload
//
// You can use the request parameters as selection criteria to return a subset of the uploads in a bucket.
// request paramters :-
// ---------
// ?part-number-marker - Specifies the part after which listing should begin.
func (a API) listObjectParts(bucketName, objectName, uploadID string, partNumberMarker, maxParts int) (listObjectPartsResult, error) {
	req, err := a.listObjectPartsRequest(bucketName, objectName, uploadID, partNumberMarker, maxParts)
	if err != nil {
		return listObjectPartsResult{}, err
	}
	resp, err := req.Do()
	defer closeResponse(resp)
	if err != nil {
		return listObjectPartsResult{}, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return listObjectPartsResult{}, BodyToErrorResponse(resp.Body)
		}
	}
	listObjectPartsResult := listObjectPartsResult{}
	err = xmlDecoder(resp.Body, &listObjectPartsResult)
	if err != nil {
		return listObjectPartsResult, err
	}
	return listObjectPartsResult, nil
}
