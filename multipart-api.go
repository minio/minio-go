package objectstorage

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
)

// initiateMultipartRequest wrapper creates a new InitiateMultiPart request
func (a *api) initiateMultipartRequest(bucket, object string) (*Request, error) {
	op := &Operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "GET",
		HTTPPath:   "/" + bucket + "/" + object + "?uploads",
	}
	return NewRequest(op, a.config, nil)
}

// InitiateMultipartUpload initiates a multipart upload and returns an upload ID
func (a *api) InitiateMultipartUpload(bucket, object string) (*InitiateMultipartUploadResult, error) {
	req, err := a.initiateMultipartRequest(bucket, object)
	if err != nil {
		return nil, err
	}
	resp, err := req.Do()
	if err != nil {
		return nil, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, ResponseToError(resp)
		}
	}
	initiateMultipartUploadResult := new(InitiateMultipartUploadResult)
	decoder := xml.NewDecoder(resp.Body)
	err = decoder.Decode(initiateMultipartUploadResult)
	if err != nil {
		return nil, err
	}
	return initiateMultipartUploadResult, resp.Body.Close()
}

// complteMultipartUploadRequest wrapper creates a new CompleteMultipartUpload request
func (a *api) completeMultipartUploadRequest(bucket, object, uploadID string, complete *CompleteMultipartUpload) (*Request, error) {
	op := &Operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "POST",
		HTTPPath:   "/" + bucket + "/" + object + "?uploadId=" + uploadID,
	}
	completeMultipartUploadBytes, err := xml.Marshal(complete)
	if err != nil {
		return nil, err
	}
	completeMultipartUploadBuffer := bytes.NewBuffer(completeMultipartUploadBytes)
	r, err := NewRequest(op, a.config, ioutil.NopCloser(completeMultipartUploadBuffer))
	if err != nil {
		return nil, err
	}
	r.req.ContentLength = int64(completeMultipartUploadBuffer.Len())
	return r, nil
}

// CompleteMultipartUpload completes a multipart upload by assembling previously uploaded parts.
func (a *api) CompleteMultipartUpload(bucket, object, uploadID string, complete *CompleteMultipartUpload) (*CompleteMultipartUploadResult, error) {
	req, err := a.completeMultipartUploadRequest(bucket, object, uploadID, complete)
	if err != nil {
		return nil, err
	}
	resp, err := req.Do()
	if err != nil {
		return nil, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, ResponseToError(resp)
		}
	}
	completeMultipartUploadResult := new(CompleteMultipartUploadResult)
	decoder := xml.NewDecoder(resp.Body)
	err = decoder.Decode(completeMultipartUploadResult)
	if err != nil {
		return nil, err
	}
	return completeMultipartUploadResult, resp.Body.Close()
}

// abortMultipartUploadRequest wrapper creates a new AbortMultipartUpload request
func (a *api) abortMultipartUploadRequest(bucket, object, uploadID string) (*Request, error) {
	op := &Operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "DELETE",
		HTTPPath:   "/" + bucket + "/" + object + "?uploadId=" + uploadID,
	}
	return NewRequest(op, a.config, nil)
}

// AbortMultipartUpload aborts a multipart upload for the given uploadID, all parts are deleted
func (a *api) AbortMultipartUpload(bucket, object, uploadID string) error {
	req, err := a.abortMultipartUploadRequest(bucket, object, uploadID)
	if err != nil {
		return err
	}
	resp, err := req.Do()
	if err != nil {
		return err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusNoContent {
			// Abort has no response body, handle it
			return fmt.Errorf("%s", resp.Status)
		}
	}
	return resp.Body.Close()
}

// listPartsRequest wrapper creates a new ListParts request
func (a *api) listPartsRequest(bucket, object, uploadID string) (*Request, error) {
	op := &Operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "GET",
		HTTPPath:   "/" + bucket + "/" + object + "?uploadId=" + uploadID,
	}
	return NewRequest(op, a.config, nil)
}

// ListParts lists the parts that have been uploaded for a specific multipart upload.
func (a *api) ListParts(bucket, object, uploadID string) (*ListPartsResult, error) {
	req, err := a.listPartsRequest(bucket, object, uploadID)
	if err != nil {
		return nil, err
	}
	resp, err := req.Do()
	if err != nil {
		return nil, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, ResponseToError(resp)
		}
	}
	listPartsResult := new(ListPartsResult)
	decoder := xml.NewDecoder(resp.Body)
	err = decoder.Decode(listPartsResult)
	if err != nil {
		return nil, err
	}
	return listPartsResult, resp.Body.Close()
}

// uploadPartRequest wrapper creates a new UploadPart request
func (a *api) uploadPartRequest(bucket, object, uploadID string, partNumber int, size int64, body io.ReadSeeker) (*Request, error) {
	op := &Operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "PUT",
		HTTPPath:   "/" + bucket + "/" + object + "?partNumber=" + strconv.Itoa(partNumber) + "&uploadId=" + uploadID,
	}
	md5Sum, err := contentMD5(body, size)
	if err != nil {
		return nil, err
	}
	r, err := NewRequest(op, a.config, ioutil.NopCloser(body))
	if err != nil {
		return nil, err
	}
	r.Set("Content-MD5", md5Sum)
	r.req.ContentLength = size
	return r, nil
}

// UploadPart uploads a part in a multipart upload.
func (a *api) UploadPart(bucket, object, uploadID string, partNumber int, size int64, body io.ReadSeeker) error {
	req, err := a.uploadPartRequest(bucket, object, uploadID, partNumber, size, body)
	if err != nil {
		return err
	}
	resp, err := req.Do()
	if err != nil {
		return err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return ResponseToError(resp)
		}
	}
	return resp.Body.Close()
}
