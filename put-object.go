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
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"runtime"
	"sort"
	"strconv"
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

// putParts - fully managed multipart uploader, resumes where its left off at `uploadID`
func (a API) putParts(bucketName, objectName, uploadID string, data io.ReadSeeker, size int64) error {
	// Cleanup any previously left stale files, as the function exits.
	defer cleanupStaleTempfiles("multiparts$")

	var seekOffset int64
	partNumber := 1
	completeMultipartUpload := completeMultipartUpload{}
	for objPart := range a.listObjectPartsRecursive(bucketName, objectName, uploadID) {
		if objPart.Err != nil {
			return objPart.Err
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
		seekOffset += objPart.Size // Add seek Offset for future Seek to skip entries.
		// Increment lexically to verify holes in next iteration.
		partNumber++
	}

	// Calculate the optimal part size for a given size.
	partSize := calculatePartSize(size)

	type erroredPart struct {
		err    error
		closer io.ReadCloser
	}
	// Allocate bufferred error channel for maximum parts.
	errCh := make(chan erroredPart, maxParts)

	// Allocate bufferred upload part channel.
	uploadedPartsCh := make(chan completePart, maxParts)

	// Limit multipart queue size to max concurrent queue, defaults to NCPUs - 1.
	mpQueueCh := make(chan struct{}, maxConcurrentQueue)

	// Close all our channels.
	defer close(errCh)
	defer close(mpQueueCh)
	defer close(uploadedPartsCh)

	// Allocate a new wait group.
	wg := new(sync.WaitGroup)

	// Seek to the new offset if greater than '0'
	if seekOffset > 0 {
		if _, err := data.Seek(seekOffset, 0); err != nil {
			return err
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
		go func(mpQueueCh <-chan struct{}, part partMetadata, wg *sync.WaitGroup,
			errCh chan<- erroredPart, uploadedPartsCh chan<- completePart) {
			defer wg.Done()
			defer func() {
				<-mpQueueCh
			}()
			if part.Err != nil {
				errCh <- erroredPart{
					err:    part.Err,
					closer: part.ReadCloser,
				}
				return
			}
			complPart, err := a.uploadPart(bucketName, objectName, uploadID, part)
			if err != nil {
				errCh <- erroredPart{
					err:    err,
					closer: part.ReadCloser,
				}
				return
			}
			uploadedPartsCh <- complPart
			errCh <- erroredPart{
				err: nil,
			}
		}(mpQueueCh, part, wg, errCh, uploadedPartsCh)
		// If any errors return right here.
		if erroredPrt, ok := <-errCh; ok {
			if erroredPrt.err != nil {
				// Close the part to remove it from disk.
				erroredPrt.closer.Close()
				return erroredPrt.err
			}
		}
		// If success fully uploaded, save them in Parts.
		if uploadedPart, ok := <-uploadedPartsCh; ok {
			completeMultipartUpload.Parts = append(completeMultipartUpload.Parts, uploadedPart)
		}
		partNumber++
	}
	wg.Wait()
	sort.Sort(completedParts(completeMultipartUpload.Parts))
	_, err := a.completeMultipartUpload(bucketName, objectName, uploadID, completeMultipartUpload)
	if err != nil {
		return err
	}
	return nil
}

// putNoChecksum special function used for anonymous uploads and Google Cloud Storage.
// This special function is necessary since Amazon S3 doesn't allow multipart uploads
// for anonymous requests. This special function is also used for Google Cloud Storage
// since multipart API is not S3 compatible.
func (a API) putNoChecksum(bucketName, objectName string, data io.ReadSeeker, size int64, contentType string) error {
	if size > maxPartSize {
		return ErrorResponse{
			Code:       "EntityTooLarge",
			Message:    "Your proposed upload exceeds the maximum allowed object size '5GB' for single PUT operation.",
			BucketName: bucketName,
			Key:        objectName,
		}
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
		return err
	}
	return nil
}

// putSmallObject uploads files smaller than 5 mega bytes.
func (a API) putSmallObject(bucketName, objectName string, data io.ReadSeeker, size int64, contentType string) error {
	dataBytes, err := ioutil.ReadAll(data)
	if err != nil {
		return err
	}
	if int64(len(dataBytes)) != size {
		msg := fmt.Sprintf("Data read ‘%s’ is not equal to expected size ‘%s’",
			strconv.FormatInt(int64(len(dataBytes)), 10), strconv.FormatInt(size, 10))
		return ErrorResponse{
			Code:       "UnexpectedShortRead",
			Message:    msg,
			BucketName: bucketName,
			Key:        objectName,
		}
	}
	putObjMetadata := putObjectMetadata{
		MD5Sum:      sumMD5(dataBytes),
		Sha256Sum:   sum256(dataBytes),
		ReadCloser:  ioutil.NopCloser(bytes.NewReader(dataBytes)),
		Size:        size,
		ContentType: contentType,
	}
	// Single part use case, use putObject directly.
	if _, err = a.putObject(bucketName, objectName, putObjMetadata); err != nil {
		return err
	}
	return nil
}

// putLargeObject uploads files bigger than 5 mega bytes.
func (a API) putLargeObject(bucketName, objectName string, data io.ReadSeeker, size int64, contentType string) error {
	var uploadID string
	isRecursive := true
	for mpUpload := range a.listIncompleteUploads(bucketName, objectName, isRecursive) {
		if mpUpload.Err != nil {
			return mpUpload.Err
		}
		if mpUpload.Key == objectName {
			uploadID = mpUpload.UploadID
			break
		}
	}
	if uploadID == "" {
		initMultipartUploadResult, err := a.initiateMultipartUpload(bucketName, objectName, contentType)
		if err != nil {
			return err
		}
		uploadID = initMultipartUploadResult.UploadID
	}
	// Initiate multipart upload.
	return a.putParts(bucketName, objectName, uploadID, data, size)
}
