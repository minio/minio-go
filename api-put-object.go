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
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// maxConcurrentQueue - max concurrent upload queue, defaults to number of CPUs - 1.
var maxConcurrentQueue = int(math.Max(float64(runtime.NumCPU())-1, 1))

// completedParts is a wrapper to make parts sortable by their part numbers.
// multi part completion requires list of multi parts to be sorted.
type completedParts []completePart

func (a completedParts) Len() int           { return len(a) }
func (a completedParts) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a completedParts) Less(i, j int) bool { return a[i].PartNumber < a[j].PartNumber }

// PutObject creates an object in a bucket.
//
// You must have WRITE permissions on a bucket to create an object.
//
//  - For size smaller than 5MB PutObject automatically does a single atomic Put operation.
//  - For size larger than 5MB PutObject automatically does a resumable multipart Put operation.
//  - For size input as -1 PutObject does a multipart Put operation until input stream reaches EOF.
//    Maximum object size that can be uploaded through this operation will be 5TB.
//
// NOTE: Google Cloud Storage multipart Put is not compatible with Amazon S3 APIs.
// Current implementation will only upload a maximum of 5GB to Google Cloud Storage servers.
//
// NOTE: For anonymous requests Amazon S3 doesn't allow multipart upload,
// so we fall back to single PUT operation.
func (a API) PutObject(bucketName, objectName string, data io.ReadSeeker, size int64, contentType string) (int64, error) {
	// Input validation.
	if err := isValidBucketName(bucketName); err != nil {
		return 0, err
	}
	if err := isValidObjectName(objectName); err != nil {
		return 0, err
	}
	// NOTE: S3 doesn't allow anonymous multipart requests.
	if isAmazonEndpoint(a.endpointURL) && isAnonymousCredentials(*a.credentials) {
		if size <= -1 {
			return 0, ErrorResponse{
				Code:       "NotImplemented",
				Message:    "For anonymous requests Content-Length cannot be negative.",
				Key:        objectName,
				BucketName: bucketName,
			}
		}
		// Do not compute MD5 for anonymous requests to Amazon S3. Uploads upto 5GB in size.
		return a.putAnonymous(bucketName, objectName, data, size, contentType)
	}
	// NOTE: Google Cloud Storage multipart Put is not compatible with Amazon S3 APIs.
	// Current implementation will only upload a maximum of 5GB to Google Cloud Storage servers.
	if isGoogleEndpoint(a.endpointURL) {
		if size <= -1 {
			return 0, ErrorResponse{
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

// putNoChecksum special function used Google Cloud Storage. This special function
// is used for Google Cloud Storage since Google's multipart API is not S3 compatible.
func (a API) putNoChecksum(bucketName, objectName string, data io.ReadSeeker, size int64, contentType string) (int64, error) {
	if size > maxPartSize {
		return 0, ErrEntityTooLarge(size, bucketName, objectName)
	}
	// For anonymous requests, we will not calculate sha256 and md5sum.
	putObjMetadata := putObjectMetadata{
		MD5Sum:      nil,
		Sha256Sum:   nil,
		ReadCloser:  ioutil.NopCloser(data),
		Size:        size,
		ContentType: contentType,
	}
	if _, err := a.putObject(bucketName, objectName, putObjMetadata); err != nil {
		return 0, err
	}
	return size, nil
}

// putAnonymous is a special function for uploading content as anonymous request.
// This special function is necessary since Amazon S3 doesn't allow anonymous
// multipart uploads.
func (a API) putAnonymous(bucketName, objectName string, data io.ReadSeeker, size int64, contentType string) (int64, error) {
	return a.putNoChecksum(bucketName, objectName, data, size, contentType)
}

// putSmallObject uploads files smaller than 5 mega bytes.
func (a API) putSmallObject(bucketName, objectName string, data io.ReadSeeker, size int64, contentType string) (int64, error) {
	dataBytes, err := ioutil.ReadAll(data)
	if err != nil {
		return 0, err
	}
	if int64(len(dataBytes)) != size {
		return 0, ErrUnexpectedEOF(int64(len(dataBytes)), size, bucketName, objectName)
	}
	putObjMetadata := putObjectMetadata{
		MD5Sum:      sumMD5(dataBytes),
		Sha256Sum:   sum256(dataBytes),
		ReadCloser:  ioutil.NopCloser(bytes.NewReader(dataBytes)),
		Size:        size,
		ContentType: contentType,
	}
	// Single part use case, use putObject directly.
	if _, err := a.putObject(bucketName, objectName, putObjMetadata); err != nil {
		return 0, err
	}
	return size, nil
}

// putLargeObject uploads files bigger than 5 mega bytes.
func (a API) putLargeObject(bucketName, objectName string, data io.ReadSeeker, size int64, contentType string) (int64, error) {
	var uploadID string
	isRecursive := true
	for mpUpload := range a.listIncompleteUploads(bucketName, objectName, isRecursive) {
		if mpUpload.Err != nil {
			return 0, mpUpload.Err
		}
		if mpUpload.Key == objectName {
			uploadID = mpUpload.UploadID
			break
		}
	}
	if uploadID == "" {
		initMultipartUploadResult, err := a.initiateMultipartUpload(bucketName, objectName, contentType)
		if err != nil {
			return 0, err
		}
		uploadID = initMultipartUploadResult.UploadID
	}
	// Initiate multipart upload.
	return a.putParts(bucketName, objectName, uploadID, data, size)
}

// putParts - fully managed multipart uploader, resumes where its left off at `uploadID`
func (a API) putParts(bucketName, objectName, uploadID string, data io.ReadSeeker, size int64) (int64, error) {
	// Cleanup any previously left stale files, as the function exits.
	defer cleanupStaleTempfiles("multiparts$")

	// total data read and written to server. should be equal to 'size' at the end of the call.
	var totalWritten int64

	// Seek offset where the file will be seeked to.
	var seekOffset int64

	// Starting part number. Always part '1'.
	partNumber := 1
	completeMultipartUpload := completeMultipartUpload{}
	for objPart := range a.listObjectPartsRecursive(bucketName, objectName, uploadID) {
		if objPart.Err != nil {
			return 0, objPart.Err
		}
		// Verify if there is a hole i.e one of the parts is missing
		// Break and start uploading that part.
		if partNumber != objPart.PartNumber {
			break
		}
		var completedPart completePart
		completedPart.PartNumber = objPart.PartNumber
		completedPart.ETag = objPart.ETag
		completeMultipartUpload.Parts = append(completeMultipartUpload.Parts, completedPart)
		// Add seek Offset for future Seek to skip entries.
		seekOffset += objPart.Size
		// Save total written to verify later.
		totalWritten += objPart.Size
		// Increment lexically to verify holes in next iteration.
		partNumber++
	}

	// Calculate the optimal part size for a given size.
	partSize := calculatePartSize(size)

	// Error struct sent back upon error.
	type uploadedPart struct {
		Part   completePart
		Closer io.ReadCloser
		Error  error
	}

	// Allocate bufferred upload part channel.
	uploadedPartsCh := make(chan uploadedPart, maxParts)

	// Limit multipart queue size to max concurrent queue, defaults to NCPUs - 1.
	mpQueueCh := make(chan struct{}, maxConcurrentQueue)

	// Close all our channels.
	defer close(mpQueueCh)
	defer close(uploadedPartsCh)

	// Allocate a new wait group.
	wg := new(sync.WaitGroup)

	// Seek to the new offset if greater than '0'
	if seekOffset > 0 {
		if _, err := data.Seek(seekOffset, 0); err != nil {
			return 0, err
		}
	}

	var enableSha256Sum bool
	// if signature V4 - enable Sha256 calculation for individual parts.
	if a.credentials.Signature.isV4() {
		enableSha256Sum = true
	}

	// Chunk all parts at partSize and start uploading.
	for part := range partsManager(data, partSize, enableSha256Sum) {
		// Limit to NCPUs-1 parts at a given time.
		mpQueueCh <- struct{}{}
		// Account for all parts uploaded simultaneousy.
		wg.Add(1)
		part.Number = partNumber
		go func(mpQueueCh <-chan struct{}, part partMetadata, wg *sync.WaitGroup, uploadedPartsCh chan<- uploadedPart) {
			defer wg.Done()
			defer func() {
				<-mpQueueCh
			}()
			if part.Err != nil {
				uploadedPartsCh <- uploadedPart{
					Error:  part.Err,
					Closer: part.ReadCloser,
				}
				return
			}
			complPart, err := a.uploadPart(bucketName, objectName, uploadID, part)
			if err != nil {
				uploadedPartsCh <- uploadedPart{
					Error:  err,
					Closer: part.ReadCloser,
				}
				return
			}
			// On Success send through both the channels.
			uploadedPartsCh <- uploadedPart{
				Part:  complPart,
				Error: nil,
			}
		}(mpQueueCh, part, wg, uploadedPartsCh)
		// If any errors return right here.
		if uploadedPrt, ok := <-uploadedPartsCh; ok {
			// Uploading failed close the Reader and return error.
			if uploadedPrt.Error != nil {
				// Close the part to remove it from disk.
				if uploadedPrt.Closer != nil {
					uploadedPrt.Closer.Close()
				}
				return totalWritten, uploadedPrt.Error
			}
			// Save successfully uploaded size.
			totalWritten += part.Size
			// Save successfully uploaded part metadata.
			completeMultipartUpload.Parts = append(completeMultipartUpload.Parts, uploadedPrt.Part)
		}
		partNumber++
	}
	wg.Wait()
	// If size is greater than zero verify totalWritten.
	// if totalWritten is different than the input 'size', do not complete the request throw an error.
	if size > 0 {
		if totalWritten != size {
			return totalWritten, ErrUnexpectedEOF(totalWritten, size, bucketName, objectName)
		}
	}
	sort.Sort(completedParts(completeMultipartUpload.Parts))
	_, err := a.completeMultipartUpload(bucketName, objectName, uploadID, completeMultipartUpload)
	if err != nil {
		return totalWritten, err
	}
	return totalWritten, nil
}

// putObjectRequest wrapper creates a new PutObject request.
func (a API) putObjectRequest(bucketName, objectName string, putObjMetadata putObjectMetadata) (*Request, error) {
	targetURL, err := getTargetURL(a.endpointURL, bucketName, objectName, url.Values{})
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(putObjMetadata.ContentType) == "" {
		putObjMetadata.ContentType = "application/octet-stream"
	}

	// get bucket region.
	region, err := a.getRegion(bucketName)
	if err != nil {
		return nil, err
	}

	// Set headers.
	putObjMetadataHeader := make(http.Header)
	putObjMetadataHeader.Set("Content-Type", putObjMetadata.ContentType)

	// Populate request metadata.
	reqMetadata := requestMetadata{
		credentials:        a.credentials,
		userAgent:          a.userAgent,
		bucketRegion:       region,
		contentBody:        putObjMetadata.ReadCloser,
		contentLength:      putObjMetadata.Size,
		contentHeader:      putObjMetadataHeader,
		contentTransport:   a.httpTransport,
		contentSha256Bytes: putObjMetadata.Sha256Sum,
		contentMD5Bytes:    putObjMetadata.MD5Sum,
	}
	req, err := newRequest("PUT", targetURL, reqMetadata)
	if err != nil {
		return nil, err
	}
	return req, nil
}

// putObject - add an object to a bucket.
// NOTE: You must have WRITE permissions on a bucket to add an object to it.
func (a API) putObject(bucketName, objectName string, putObjMetadata putObjectMetadata) (ObjectStat, error) {
	req, err := a.putObjectRequest(bucketName, objectName, putObjMetadata)
	if err != nil {
		return ObjectStat{}, err
	}
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return ObjectStat{}, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return ObjectStat{}, BodyToErrorResponse(resp.Body)
		}
	}
	var metadata ObjectStat
	// Trim off the odd double quotes from ETag.
	metadata.ETag = strings.Trim(resp.Header.Get("ETag"), "\"")
	// A success here means data was written to server successfully.
	metadata.Size = putObjMetadata.Size
	return metadata, nil
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

	reqMetadata := requestMetadata{
		credentials:      a.credentials,
		userAgent:        a.userAgent,
		bucketRegion:     region,
		contentHeader:    multipartHeader,
		contentTransport: a.httpTransport,
	}
	return newRequest("POST", targetURL, reqMetadata)
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

	reqMetadata := requestMetadata{
		credentials:        a.credentials,
		userAgent:          a.userAgent,
		bucketRegion:       region,
		contentBody:        uploadingPart.ReadCloser,
		contentLength:      uploadingPart.Size,
		contentTransport:   a.httpTransport,
		contentSha256Bytes: uploadingPart.Sha256Sum,
		contentMD5Bytes:    uploadingPart.MD5Sum,
	}
	req, err := newRequest("PUT", targetURL, reqMetadata)
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
	reqMetadata := requestMetadata{
		credentials:        a.credentials,
		userAgent:          a.userAgent,
		bucketRegion:       region,
		contentBody:        ioutil.NopCloser(completeMultipartUploadBuffer),
		contentLength:      int64(completeMultipartUploadBuffer.Len()),
		contentTransport:   a.httpTransport,
		contentSha256Bytes: sum256(completeMultipartUploadBuffer.Bytes()),
	}
	req, err := newRequest("POST", targetURL, reqMetadata)
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
