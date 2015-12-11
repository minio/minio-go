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
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

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
		credentials:  a.credentials,
		userAgent:    a.userAgent,
		bucketRegion: region,
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
	defer closeResp(resp)
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

// initiateMultipartRequest wrapper creates a new initiateMultiPart request.
func (a API) initiateMultipartRequest(bucketName, objectName, contentType string) (*Request, error) {
	// Initialize url queries.
	urlValues := make(url.Values)
	urlValues.Set("uploads", "")

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

	if contentType == "" {
		contentType = "application/octet-stream"
	}
	// set ContentType header.
	multipartHeader := make(http.Header)
	multipartHeader.Set("Content-Type", contentType)

	rmetadata := requestMetadata{
		credentials:   a.credentials,
		userAgent:     a.userAgent,
		bucketRegion:  region,
		contentHeader: multipartHeader,
	}
	return newRequest("POST", targetURL, rmetadata)
}

// initiateMultipartUpload initiates a multipart upload and returns an upload ID.
func (a API) initiateMultipartUpload(bucketName, objectName, contentType string) (initiateMultipartUploadResult, error) {
	req, err := a.initiateMultipartRequest(bucketName, objectName, contentType)
	if err != nil {
		return initiateMultipartUploadResult{}, err
	}
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return initiateMultipartUploadResult{}, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return initiateMultipartUploadResult{}, BodyToErrorResponse(resp.Body)
		}
	}
	initiateMultipartUploadResult := initiateMultipartUploadResult{}
	err = xmlDecoder(resp.Body, &initiateMultipartUploadResult)
	if err != nil {
		return initiateMultipartUploadResult, err
	}
	return initiateMultipartUploadResult, nil
}

// completeMultipartUploadRequest wrapper creates a new CompleteMultipartUpload request.
func (a API) completeMultipartUploadRequest(bucketName, objectName, uploadID string,
	complete completeMultipartUpload) (*Request, error) {
	// Initialize url queries.
	urlValues := make(url.Values)
	urlValues.Set("uploadId", uploadID)

	// get targetURL.
	targetURL, err := getTargetURL(a.endpointURL, bucketName, objectName, urlValues)
	if err != nil {
		return nil, err
	}
	completeMultipartUploadBytes, err := xml.Marshal(complete)
	if err != nil {
		return nil, err
	}

	// get bucket region.
	region, err := a.getRegion(bucketName)
	if err != nil {
		return nil, err
	}

	completeMultipartUploadBuffer := bytes.NewBuffer(completeMultipartUploadBytes)
	rmetadata := requestMetadata{
		credentials:        a.credentials,
		userAgent:          a.userAgent,
		bucketRegion:       region,
		contentBody:        ioutil.NopCloser(completeMultipartUploadBuffer),
		contentLength:      int64(completeMultipartUploadBuffer.Len()),
		contentSha256Bytes: sum256(completeMultipartUploadBuffer.Bytes()),
	}
	req, err := newRequest("POST", targetURL, rmetadata)
	if err != nil {
		return nil, err
	}
	return req, nil
}

// completeMultipartUpload completes a multipart upload by assembling previously uploaded parts.
func (a API) completeMultipartUpload(bucketName, objectName, uploadID string,
	c completeMultipartUpload) (completeMultipartUploadResult, error) {
	req, err := a.completeMultipartUploadRequest(bucketName, objectName, uploadID, c)
	if err != nil {
		return completeMultipartUploadResult{}, err
	}
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return completeMultipartUploadResult{}, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return completeMultipartUploadResult{}, BodyToErrorResponse(resp.Body)
		}
	}
	completeMultipartUploadResult := completeMultipartUploadResult{}
	err = xmlDecoder(resp.Body, &completeMultipartUploadResult)
	if err != nil {
		return completeMultipartUploadResult, err
	}
	return completeMultipartUploadResult, nil
}

// abortMultipartUploadRequest wrapper creates a new AbortMultipartUpload request.
func (a API) abortMultipartUploadRequest(bucketName, objectName, uploadID string) (*Request, error) {
	// Initialize url queries.
	urlValues := make(url.Values)
	urlValues.Set("uploadId", uploadID)

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

// abortMultipartUpload aborts a multipart upload for the given uploadID, all parts are deleted.
func (a API) abortMultipartUpload(bucketName, objectName, uploadID string) error {
	req, err := a.abortMultipartUploadRequest(bucketName, objectName, uploadID)
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
			// Abort has no response body, handle it.
			var errorResponse ErrorResponse
			switch resp.StatusCode {
			case http.StatusNotFound:
				errorResponse = ErrorResponse{
					Code:            "NoSuchUpload",
					Message:         "The specified multipart upload does not exist.",
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
					Message:         "Unknown error, please report this at https://github.com/minio/minio-go-legacy/issues.",
					BucketName:      bucketName,
					Key:             objectName,
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
		credentials:  a.credentials,
		userAgent:    a.userAgent,
		bucketRegion: region,
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
	defer closeResp(resp)
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

// uploadPartRequest wrapper creates a new UploadPart request.
func (a API) uploadPartRequest(bucketName, objectName, uploadID string, uploadingPart partMetadata) (*Request, error) {
	// Get resources properly escaped and lined up before using them in http request.
	urlValues := make(url.Values)
	// Set part number.
	urlValues.Set("partNumber", strconv.Itoa(uploadingPart.Number))
	// Set upload id.
	urlValues.Set("uploadId", uploadID)

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

	rmetadata := requestMetadata{
		credentials:        a.credentials,
		userAgent:          a.userAgent,
		bucketRegion:       region,
		contentBody:        uploadingPart.ReadCloser,
		contentLength:      uploadingPart.Size,
		contentSha256Bytes: uploadingPart.Sha256Sum,
		contentMD5Bytes:    uploadingPart.MD5Sum,
	}
	req, err := newRequest("PUT", targetURL, rmetadata)
	if err != nil {
		return nil, err
	}
	return req, nil
}

// uploadPart uploads a part in a multipart upload.
func (a API) uploadPart(bucketName, objectName, uploadID string, uploadingPart partMetadata) (completePart, error) {
	req, err := a.uploadPartRequest(bucketName, objectName, uploadID, uploadingPart)
	if err != nil {
		return completePart{}, err
	}
	// Initiate the request.
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return completePart{}, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return completePart{}, BodyToErrorResponse(resp.Body)
		}
	}
	cPart := completePart{}
	cPart.PartNumber = uploadingPart.Number
	cPart.ETag = resp.Header.Get("ETag")
	return cPart, nil
}
