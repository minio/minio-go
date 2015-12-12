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
	"io"
	"io/ioutil"
	"math"
	"runtime"
	"sort"
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

// putNoChecksum special function used for anonymous uploads and Google Cloud Storage.
// This special function is necessary since Amazon S3 doesn't allow multipart uploads
// for anonymous requests. This special function is also used for Google Cloud Storage
// since multipart API is not S3 compatible.
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
