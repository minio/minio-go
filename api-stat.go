package minio

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// BucketExists verify if bucket exists and you have permission to access it.
func (a API) BucketExists(bucketName string) error {
	if err := isValidBucketName(bucketName); err != nil {
		return err
	}
	req, err := a.bucketExistsRequest(bucketName)
	if err != nil {
		return err
	}
	resp, err := req.Do()
	defer closeResponse(resp)
	if err != nil {
		return err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			// Head has no response body, handle it.
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

// bucketExistsRequest constructs a new request for BucketExists.
func (a API) bucketExistsRequest(bucketName string) (*Request, error) {
	targetURL, err := getTargetURL(a.endpointURL, bucketName, "", url.Values{})
	if err != nil {
		return nil, err
	}

	// get bucket region.
	region, err := a.getRegion(bucketName)
	if err != nil {
		return nil, err
	}

	return newRequest("HEAD", targetURL, requestMetadata{
		credentials:      a.credentials,
		userAgent:        a.userAgent,
		bucketRegion:     region,
		contentTransport: a.httpTransport,
	})
}

// StatObject verifies if object exists and you have permission to access.
func (a API) StatObject(bucketName, objectName string) (ObjectStat, error) {
	if err := isValidBucketName(bucketName); err != nil {
		return ObjectStat{}, err
	}
	if err := isValidObjectName(objectName); err != nil {
		return ObjectStat{}, err
	}
	req, err := a.statObjectRequest(bucketName, objectName)
	if err != nil {
		return ObjectStat{}, err
	}
	resp, err := req.Do()
	defer closeResponse(resp)
	if err != nil {
		return ObjectStat{}, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			var errorResponse ErrorResponse
			switch resp.StatusCode {
			case http.StatusNotFound:
				errorResponse = ErrorResponse{
					Code:            "NoSuchKey",
					Message:         "The specified key does not exist.",
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
					Message:         resp.Status,
					BucketName:      bucketName,
					Key:             objectName,
					RequestID:       resp.Header.Get("x-amz-request-id"),
					HostID:          resp.Header.Get("x-amz-id-2"),
					AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
				}

			}
			return ObjectStat{}, errorResponse
		}
	}
	md5sum := strings.Trim(resp.Header.Get("ETag"), "\"") // trim off the odd double quotes
	size, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return ObjectStat{}, ErrorResponse{
			Code:            "InternalError",
			Message:         "Content-Length is invalid, please report this issue at https://github.com/minio/minio-go/issues.",
			BucketName:      bucketName,
			Key:             objectName,
			RequestID:       resp.Header.Get("x-amz-request-id"),
			HostID:          resp.Header.Get("x-amz-id-2"),
			AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
		}
	}
	date, err := time.Parse(http.TimeFormat, resp.Header.Get("Last-Modified"))
	if err != nil {
		return ObjectStat{}, ErrorResponse{
			Code:            "InternalError",
			Message:         "Last-Modified time format is invalid, please report this issue at https://github.com/minio/minio-go/issues.",
			BucketName:      bucketName,
			Key:             objectName,
			RequestID:       resp.Header.Get("x-amz-request-id"),
			HostID:          resp.Header.Get("x-amz-id-2"),
			AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
		}
	}
	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	// Save object metadata info.
	var objectStat ObjectStat
	objectStat.ETag = md5sum
	objectStat.Key = objectName
	objectStat.Size = size
	objectStat.LastModified = date
	objectStat.ContentType = contentType
	return objectStat, nil
}

// statObjectRequest wrapper creates a new headObject request.
func (a API) statObjectRequest(bucketName, objectName string) (*Request, error) {
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
	req, err := newRequest("HEAD", targetURL, requestMetadata{
		credentials:      a.credentials,
		userAgent:        a.userAgent,
		bucketRegion:     region,
		contentTransport: a.httpTransport,
	})
	if err != nil {
		return nil, err
	}

	// Return new request.
	return req, nil
}
