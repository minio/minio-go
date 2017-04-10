/*
 * Minio Go Library for Amazon S3 Compatible Cloud Storage (C) 2016 Minio, Inc.
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
	"strings"

	"github.com/minio/minio-go/pkg/s3utils"
)

// CopyObject - copy a source object into a new object with the
// provided name in the provided bucket
func (c Client) CopyObject(bucketName string, objectName string, objectSource string, cpCond CopyConditions) error {
	// Input validation.
	if err := isValidBucketName(bucketName); err != nil {
		return err
	}
	if err := isValidObjectName(objectName); err != nil {
		return err
	}
	srcBucket, srcObject, err := getObjectSource(objectSource)
	if err != nil {
		return err
	}

	// Get info about the source object
	srcInfo, err := c.StatObject(srcBucket, srcObject)
	if err != nil {
		return err
	}
	srcByteRangeSize := cpCond.getRangeSize()
	if srcByteRangeSize > srcInfo.Size ||
		(srcByteRangeSize > 0 && cpCond.byteRangeEnd >= srcInfo.Size) {
		return ErrInvalidArgument(fmt.Sprintf(
			"Specified byte range (%d, %d) does not fit within source object (size = %d)",
			cpCond.byteRangeStart, cpCond.byteRangeEnd, srcInfo.Size))
	}

	copySize := srcByteRangeSize
	if copySize == 0 {
		copySize = srcInfo.Size
	}

	// customHeaders apply headers.
	customHeaders := make(http.Header)
	for key, value := range cpCond.conditions {
		customHeaders.Set(key, value)
	}

	// Set copy source.
	customHeaders.Set("x-amz-copy-source", s3utils.EncodePath(objectSource))

	// Check if single part copy suffices. Multipart is required when:
	// 1. source-range-offset does not refer to full source object, or
	// 2. size of copied object > 5gb
	if copySize > maxPartSize ||
		(srcByteRangeSize > 0 && srcByteRangeSize != srcInfo.Size) {
		return c.multipartCopyObject(bucketName, objectName,
			objectSource, cpCond, customHeaders, copySize)
	}

	// Execute PUT on objectName.
	resp, err := c.executeMethod("PUT", requestMetadata{
		bucketName:   bucketName,
		objectName:   objectName,
		customHeader: customHeaders,
	})
	defer closeResponse(resp)
	if err != nil {
		return err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return httpRespToErrorResponse(resp, bucketName, objectName)
		}
	}

	// Decode copy response on success.
	cpObjRes := copyObjectResult{}
	err = xmlDecoder(resp.Body, &cpObjRes)
	if err != nil {
		return err
	}

	// Return nil on success.
	return nil
}

func getObjectSource(src string) (bucket string, object string, err error) {
	parts := strings.Split(src, "/")
	if len(parts) != 3 || parts[0] != "" || parts[1] == "" || parts[2] == "" {
		return "", "", ErrInvalidArgument("Object source should be formatted as '/bucketName/objectName'")
	}
	return parts[1], parts[2], nil
}

func (c Client) multipartCopyObject(bucketName string, objectName string,
	objectSource string, cpCond CopyConditions, headers http.Header,
	copySize int64) error {

	// Compute split sizes for multipart copy.
	partsCount, partSize, lastPartSize, err := optimalPartInfo(copySize)
	if err != nil {
		return err
	}

	// It is not possible to resume a multipart copy object
	// operation, so we just create new uploadID and proceed with
	// the copy operations.
	uid, err := c.newUploadID(bucketName, objectSource, nil)
	if err != nil {
		return err
	}

	queryParams := url.Values{}
	queryParams.Set("uploadId", uid)

	var complMultipartUpload completeMultipartUpload

	// Initiate copy object operations.
	for partNumber := 1; partNumber <= partsCount; partNumber++ {
		pCond := cpCond.duplicate()
		pCond.byteRangeStart = partSize * (int64(partNumber) - 1)
		if partNumber < partsCount {
			pCond.byteRangeEnd = pCond.byteRangeStart + partSize - 1
		} else {
			pCond.byteRangeEnd = pCond.byteRangeStart + lastPartSize - 1
		}

		// Update the source range header value.
		headers.Set("x-amz-copy-source-range",
			fmt.Sprintf("bytes:%d-%d", pCond.byteRangeStart,
				pCond.byteRangeEnd))

		// Update part number in the query parameters.
		queryParams.Set("partNumber", fmt.Sprintf("%d", partNumber))

		// Perform part-copy.
		resp, err := c.executeMethod("PUT", requestMetadata{
			bucketName:   bucketName,
			objectName:   objectName,
			customHeader: headers,
			queryValues:  queryParams,
		})
		defer closeResponse(resp)
		if err != nil {
			return err
		}
		if resp != nil {
			if resp.StatusCode != http.StatusOK {
				return httpRespToErrorResponse(resp, bucketName,
					objectName)
			}
		}

		// Decode copy response on success.
		cpObjRes := copyObjectResult{}
		err = xmlDecoder(resp.Body, &cpObjRes)
		if err != nil {
			return err
		}

		// append part info for complete multipart request
		complMultipartUpload.Parts = append(complMultipartUpload.Parts,
			CompletePart{
				PartNumber: partNumber,
				ETag:       cpObjRes.ETag,
			})
	}

	// Complete the multipart upload.
	_, err = c.completeMultipartUpload(bucketName, objectName, uid,
		complMultipartUpload)
	return err
}
