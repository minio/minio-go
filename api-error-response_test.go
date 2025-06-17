/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2015-2017 MinIO, Inc.
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
	"io"
	"net/http"
	"reflect"
	"strconv"
	"testing"
)

// Tests validate the Error generator function for http response with error.
func TestHttpRespToErrorResponse(t *testing.T) {
	// 'genAPIErrorResponse' generates ErrorResponse for given APIError.
	// provides a encodable populated response values.
	genAPIErrorResponse := func(err APIError, bucketName string) ErrorResponse {
		return ErrorResponse{
			Code:       err.Code,
			Message:    err.Description,
			BucketName: bucketName,
		}
	}

	// Encodes the response headers into XML format.
	encodeErr := func(response ErrorResponse) []byte {
		buf := &bytes.Buffer{}
		buf.WriteString(xml.Header)
		encoder := xml.NewEncoder(buf)
		err := encoder.Encode(response)
		if err != nil {
			t.Fatalf("error encoding response: %v", err)
		}
		return buf.Bytes()
	}

	// `createErrorResponse` Mocks a generic error response from the server.
	createErrorResponse := func(statusCode int, body []byte) *http.Response {
		resp := &http.Response{}
		resp.StatusCode = statusCode
		resp.Status = http.StatusText(statusCode)
		resp.Body = io.NopCloser(bytes.NewBuffer(body))
		return resp
	}

	// `createAPIErrorResponse` Mocks XML error response from the server.
	createAPIErrorResponse := func(APIErr APIError, bucketName string) *http.Response {
		// generate error response.
		// response body contains the XML error message.
		errorResponse := genAPIErrorResponse(APIErr, bucketName)
		encodedErrorResponse := encodeErr(errorResponse)
		return createErrorResponse(APIErr.HTTPStatusCode, encodedErrorResponse)
	}

	// 'genErrResponse' contructs error response based http Status Code
	genErrResponse := func(resp *http.Response, code, message, bucketName, objectName string) ErrorResponse {
		errResp := ErrorResponse{
			StatusCode: resp.StatusCode,
			Code:       code,
			Message:    message,
			BucketName: bucketName,
			Key:        objectName,
			RequestID:  resp.Header.Get("x-amz-request-id"),
			HostID:     resp.Header.Get("x-amz-id-2"),
			Region:     resp.Header.Get("x-amz-bucket-region"),
		}
		return errResp
	}

	// Generate invalid argument error.
	genInvalidError := func(message string) error {
		errResp := ErrorResponse{
			StatusCode: http.StatusBadRequest,
			Code:       InvalidArgument,
			Message:    message,
			RequestID:  "minio",
		}
		return errResp
	}

	// Set common http response headers.
	setCommonHeaders := func(resp *http.Response) *http.Response {
		// set headers.
		resp.Header = make(http.Header)
		resp.Header.Set("x-amz-request-id", "xyz")
		resp.Header.Set("x-amz-id-2", "abc")
		resp.Header.Set("x-amz-bucket-region", "us-east-1")
		return resp
	}

	// Generate http response with empty body.
	// Set the StatusCode to the argument supplied.
	// Sets common headers.
	genEmptyBodyResponse := func(statusCode int) *http.Response {
		resp := &http.Response{
			StatusCode: statusCode,
			Status:     http.StatusText(statusCode),
			Body:       io.NopCloser(bytes.NewReader(nil)),
		}
		setCommonHeaders(resp)
		return resp
	}

	// Decode XML error message from the http response body.
	decodeXMLError := func(resp *http.Response) error {
		errResp := ErrorResponse{
			StatusCode: resp.StatusCode,
		}
		err := xmlDecoder(resp.Body, &errResp)
		if err != nil {
			t.Fatalf("XML decoding of response body failed: %v", err)
		}
		return errResp
	}

	// List of APIErrors used to generate/mock server side XML error response.
	APIErrors := []APIError{
		{
			Code:           NoSuchBucketPolicy,
			Description:    "The specified bucket does not have a bucket policy.",
			HTTPStatusCode: http.StatusNotFound,
		},
	}

	// List of expected response.
	// Used for asserting the actual response.
	expectedErrResponse := []error{
		genInvalidError("Empty http response. " + "Please report this issue at https://github.com/minio/minio-go/issues."),
		decodeXMLError(createAPIErrorResponse(APIErrors[0], "minio-bucket")),
		genErrResponse(setCommonHeaders(&http.Response{StatusCode: http.StatusNotFound}), NoSuchBucket, s3ErrorResponseMap[NoSuchBucket], "minio-bucket", ""),
		genErrResponse(setCommonHeaders(&http.Response{StatusCode: http.StatusNotFound}), NoSuchKey, s3ErrorResponseMap[NoSuchKey], "minio-bucket", "Asia/"),
		genErrResponse(setCommonHeaders(&http.Response{StatusCode: http.StatusForbidden}), AccessDenied, s3ErrorResponseMap[AccessDenied], "minio-bucket", ""),
		genErrResponse(setCommonHeaders(&http.Response{StatusCode: http.StatusConflict}), Conflict, s3ErrorResponseMap[Conflict], "minio-bucket", ""),
		genErrResponse(setCommonHeaders(&http.Response{StatusCode: http.StatusBadRequest}), "Bad Request", "Bad Request", "minio-bucket", ""),
		genErrResponse(setCommonHeaders(&http.Response{StatusCode: http.StatusInternalServerError}), "Internal Server Error", "my custom object store error", "minio-bucket", ""),
		genErrResponse(setCommonHeaders(&http.Response{StatusCode: http.StatusInternalServerError}), "Internal Server Error", "my custom object store error, with way too long body", "minio-bucket", ""),
	}

	// List of http response to be used as input.
	inputResponses := []*http.Response{
		nil,
		createAPIErrorResponse(APIErrors[0], "minio-bucket"),
		genEmptyBodyResponse(http.StatusNotFound),
		genEmptyBodyResponse(http.StatusNotFound),
		genEmptyBodyResponse(http.StatusForbidden),
		genEmptyBodyResponse(http.StatusConflict),
		genEmptyBodyResponse(http.StatusBadRequest),
		setCommonHeaders(createErrorResponse(http.StatusInternalServerError, []byte("my custom object store error\n"))),
		setCommonHeaders(createErrorResponse(http.StatusInternalServerError, append([]byte("my custom object store error, with way too long body\n"), bytes.Repeat([]byte("\n"), 2*1024*1024)...))),
	}

	testCases := []struct {
		bucketName    string
		objectName    string
		inputHTTPResp *http.Response
		// expected results.
		expectedResult error
		// flag indicating whether tests should pass.
	}{
		{"minio-bucket", "", inputResponses[0], expectedErrResponse[0]},
		{"minio-bucket", "", inputResponses[1], expectedErrResponse[1]},
		{"minio-bucket", "", inputResponses[2], expectedErrResponse[2]},
		{"minio-bucket", "Asia/", inputResponses[3], expectedErrResponse[3]},
		{"minio-bucket", "", inputResponses[4], expectedErrResponse[4]},
		{"minio-bucket", "", inputResponses[5], expectedErrResponse[5]},
		{"minio-bucket", "", inputResponses[6], expectedErrResponse[6]},
		{"minio-bucket", "", inputResponses[7], expectedErrResponse[7]},
		{"minio-bucket", "", inputResponses[8], expectedErrResponse[8]},
	}

	for i, testCase := range testCases {
		actualResult := httpRespToErrorResponse(testCase.inputHTTPResp, testCase.bucketName, testCase.objectName)
		if !reflect.DeepEqual(testCase.expectedResult, actualResult) {
			t.Errorf("Test %d: Expected result to be '%#v', but instead got '%#v'", i+1, testCase.expectedResult, actualResult)
		}
	}
}

// Test validates 'ErrEntityTooLarge' error response.
func TestErrEntityTooLarge(t *testing.T) {
	msg := fmt.Sprintf("Your proposed upload size ‘%d’ exceeds the maximum allowed object size ‘%d’ for single PUT operation.", 1000000, 99999)
	expectedResult := ErrorResponse{
		StatusCode: http.StatusBadRequest,
		Code:       EntityTooLarge,
		Message:    msg,
		BucketName: "minio-bucket",
		Key:        "Asia/",
	}
	actualResult := errEntityTooLarge(1000000, 99999, "minio-bucket", "Asia/")
	if !reflect.DeepEqual(expectedResult, actualResult) {
		t.Errorf("Expected result to be '%#v', but instead got '%#v'", expectedResult, actualResult)
	}
}

// Test validates 'ErrEntityTooSmall' error response.
func TestErrEntityTooSmall(t *testing.T) {
	msg := fmt.Sprintf("Your proposed upload size ‘%d’ is below the minimum allowed object size ‘0B’ for single PUT operation.", -1)
	expectedResult := ErrorResponse{
		StatusCode: http.StatusBadRequest,
		Code:       EntityTooSmall,
		Message:    msg,
		BucketName: "minio-bucket",
		Key:        "Asia/",
	}
	actualResult := errEntityTooSmall(-1, "minio-bucket", "Asia/")
	if !reflect.DeepEqual(expectedResult, actualResult) {
		t.Errorf("Expected result to be '%#v', but instead got '%#v'", expectedResult, actualResult)
	}
}

// Test validates 'ErrUnexpectedEOF' error response.
func TestErrUnexpectedEOF(t *testing.T) {
	msg := fmt.Sprintf("Data read ‘%s’ is not equal to the size ‘%s’ of the input Reader.",
		strconv.FormatInt(100, 10), strconv.FormatInt(101, 10))
	expectedResult := ErrorResponse{
		StatusCode: http.StatusBadRequest,
		Code:       UnexpectedEOF,
		Message:    msg,
		BucketName: "minio-bucket",
		Key:        "Asia/",
	}
	actualResult := errUnexpectedEOF(100, 101, "minio-bucket", "Asia/")
	if !reflect.DeepEqual(expectedResult, actualResult) {
		t.Errorf("Expected result to be '%#v', but instead got '%#v'", expectedResult, actualResult)
	}
}

// Test validates 'errInvalidArgument' response.
func TestErrInvalidArgument(t *testing.T) {
	expectedResult := ErrorResponse{
		StatusCode: http.StatusBadRequest,
		Code:       InvalidArgument,
		Message:    "Invalid Argument",
		RequestID:  "minio",
	}
	actualResult := errInvalidArgument("Invalid Argument")
	if !reflect.DeepEqual(expectedResult, actualResult) {
		t.Errorf("Expected result to be '%#v', but instead got '%#v'", expectedResult, actualResult)
	}
}

// Tests if the Message field is missing.
func TestErrWithoutMessage(t *testing.T) {
	errResp := ErrorResponse{
		Code:      AccessDenied,
		RequestID: "minio",
	}

	if errResp.Error() != s3ErrorResponseMap[AccessDenied] {
		t.Errorf("Expected \"%s\", got %s", s3ErrorResponseMap[AccessDenied], errResp)
	}

	errResp = ErrorResponse{
		Code:      InvalidArgument,
		RequestID: "minio",
	}
	if errResp.Error() != fmt.Sprintf("Error response code %s.", errResp.Code) {
		t.Errorf("Expected \"Error response code %s.\", got \"%s\"", InvalidArgument, errResp)
	}
}

// Tests if ErrorResponse is comparable since it is compared
// inside golang http code (https://github.com/golang/go/issues/29768)
func TestErrorResponseComparable(t *testing.T) {
	var e1 interface{} = ErrorResponse{}
	var e2 interface{} = ErrorResponse{}
	if e1 != e2 {
		t.Fatalf("ErrorResponse should be comparable")
	}
}
