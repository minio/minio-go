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

// listBucketsInRoutine goroutine based iterator for listBuckets.
func (a API) listBucketsInRoutine(ch chan<- BucketStat) {
	defer close(ch)
	listAllMyBucketListResults, err := a.listBuckets()
	if err != nil {
		ch <- BucketStat{
			Err: err,
		}
		return
	}
	for _, bucket := range listAllMyBucketListResults.Buckets.Bucket {
		ch <- bucket
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
		return
	}
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

// listIncompleteUploads lists all incomplete uploads.
func (a API) listIncompleteUploads(bucketName, objectName string, recursive bool) <-chan ObjectMultipartStat {
	ch := make(chan ObjectMultipartStat, 1000)
	go a.listIncompleteUploadsInRoutine(bucketName, objectName, recursive, ch)
	return ch
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

// removeIncompleteUploadInRoutine iterates over all incomplete uploads
// and removes only input object name.
func (a API) removeIncompleteUploadInRoutine(bucketName, objectName string, errorCh chan<- error) {
	defer close(errorCh)
	// Validate incoming bucket name.
	if err := isValidBucketName(bucketName); err != nil {
		errorCh <- err
		return
	}
	// Validate incoming object name.
	if err := isValidObjectName(objectName); err != nil {
		errorCh <- err
		return
	}
	// List all incomplete uploads recursively.
	for mpUpload := range a.listIncompleteUploads(bucketName, objectName, true) {
		if objectName == mpUpload.Key {
			err := a.abortMultipartUpload(bucketName, mpUpload.Key, mpUpload.UploadID)
			if err != nil {
				errorCh <- err
				return
			}
			return
		}
	}
}
