package minio

import (
	"net/http"
	"net/url"
)

// RemoveBucket deletes the bucket name.
//
//  All objects (including all object versions and delete markers).
//  in the bucket must be deleted before successfully attempting this request.
func (a API) RemoveBucket(bucketName string) error {
	if err := isValidBucketName(bucketName); err != nil {
		return err
	}
	req, err := a.removeBucketRequest(bucketName)
	if err != nil {
		return err
	}
	resp, err := req.Do()
	defer closeResponse(resp)
	if err != nil {
		return err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusNoContent {
			var errorResponse ErrorResponse
			switch resp.StatusCode {
			case http.StatusNotFound:
				errorResponse = ErrorResponse{
					Code:            "NoSuchBucket",
					Message:         "The specified bucket does not exist.",
					BucketName:      bucketName,
					RequestID:       resp.Header.Get("x-amz-request-id"),
					HostID:          resp.Header.Get("x-amz-id-2"),
					AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
				}
			case http.StatusForbidden:
				errorResponse = ErrorResponse{
					Code:            "AccessDenied",
					Message:         "Access Denied.",
					BucketName:      bucketName,
					RequestID:       resp.Header.Get("x-amz-request-id"),
					HostID:          resp.Header.Get("x-amz-id-2"),
					AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
				}
			case http.StatusConflict:
				errorResponse = ErrorResponse{
					Code:            "Conflict",
					Message:         "Bucket not empty.",
					BucketName:      bucketName,
					RequestID:       resp.Header.Get("x-amz-request-id"),
					HostID:          resp.Header.Get("x-amz-id-2"),
					AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
				}
			default:
				errorResponse = ErrorResponse{
					Code:            resp.Status,
					Message:         resp.Status,
					BucketName:      bucketName,
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

// removeBucketRequest constructs a new request for RemoveBucket.
func (a API) removeBucketRequest(bucketName string) (*Request, error) {
	targetURL, err := getTargetURL(a.endpointURL, bucketName, "", url.Values{})
	if err != nil {
		return nil, err
	}

	// get bucket region.
	region, err := a.getRegion(bucketName)
	if err != nil {
		return nil, err
	}

	return newRequest("DELETE", targetURL, requestMetadata{
		credentials:      a.credentials,
		userAgent:        a.userAgent,
		bucketRegion:     region,
		contentTransport: a.httpTransport,
	})
}

// RemoveObject remove an object from a bucket.
func (a API) RemoveObject(bucketName, objectName string) error {
	if err := isValidBucketName(bucketName); err != nil {
		return err
	}
	if err := isValidObjectName(objectName); err != nil {
		return err
	}
	req, err := a.removeObjectRequest(bucketName, objectName)
	if err != nil {
		return err
	}
	resp, err := req.Do()
	defer closeResponse(resp)
	if err != nil {
		return err
	}
	// DeleteObject always responds with http '204' even for
	// objects which do not exist. So no need to handle them
	// specifically.
	return nil
}

// removeObjectRequest constructs a request for RemoveObject.
func (a API) removeObjectRequest(bucketName, objectName string) (*Request, error) {
	// get targetURL.
	targetURL, err := getTargetURL(a.endpointURL, bucketName, objectName, url.Values{})
	if err != nil {
		return nil, err
	}

	// get bucket region.
	region, err := a.getRegion(bucketName)
	if err != nil {
		return nil, err
	}

	// Instantiate a new request.
	req, err := newRequest("DELETE", targetURL, requestMetadata{
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

// RemoveIncompleteUpload aborts an partially uploaded object.
// Requires explicit authentication, no anonymous requests are allowed for multipart API.
func (a API) RemoveIncompleteUpload(bucketName, objectName string) <-chan error {
	errorCh := make(chan error)
	go a.removeIncompleteUploadInRoutine(bucketName, objectName, errorCh)
	return errorCh
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

// abortMultipartUpload aborts a multipart upload for the given uploadID, all parts are deleted.
func (a API) abortMultipartUpload(bucketName, objectName, uploadID string) error {
	req, err := a.abortMultipartUploadRequest(bucketName, objectName, uploadID)
	if err != nil {
		return err
	}
	resp, err := req.Do()
	defer closeResponse(resp)
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
