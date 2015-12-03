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
	"encoding/json"
	"encoding/xml"
	"io"
)

/* **** SAMPLE ERROR RESPONSE ****
<?xml version="1.0" encoding="UTF-8"?>
<Error>
   <Code>AccessDenied</Code>
   <Message>Access Denied</Message>
   <Resource>/mybucket/myphoto.jpg</Resource>
   <RequestId>F19772218238A85A</RequestId>
   <HostId>GuWkjyviSiGHizehqpmsD1ndz5NClSP19DOT+s2mv7gXGQ8/X1lhbDGiIJEXpGFD</HostId>
</Error>
*/

// ErrorResponse is the type error returned by some API operations.
type ErrorResponse struct {
	XMLName   xml.Name `xml:"Error" json:"-"`
	Code      string
	Message   string
	Resource  string
	RequestID string `xml:"RequestId"`
	HostID    string `xml:"HostId"`

	// This is a new undocumented field, set only if available.
	AmzBucketRegion string
}

// ToErrorResponse returns parsed ErrorResponse struct, if input is nil or not ErrorResponse return value is nil
// this fuction is useful when some one wants to dig deeper into the error structures over the network.
//
// For example:
//
//   import s3 "github.com/minio/minio-go"
//   ...
//   ...
//   reader, stat, err := s3.GetObject(...)
//   if err != nil {
//      resp := s3.ToErrorResponse(err)
//      fmt.Println(resp.ToXML())
//   }
//   ...
func ToErrorResponse(err error) *ErrorResponse {
	switch err := err.(type) {
	case ErrorResponse:
		return &err
	default:
		return nil
	}
}

// ToXML send raw xml marshalled as string
func (e ErrorResponse) ToXML() string {
	b, err := xml.Marshal(&e)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// ToJSON send raw json marshalled as string
func (e ErrorResponse) ToJSON() string {
	b, err := json.Marshal(&e)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// Error formats HTTP error string
func (e ErrorResponse) Error() string {
	return e.Message
}

// BodyToErrorResponse returns a new encoded ErrorResponse structure
func BodyToErrorResponse(errBody io.Reader) error {
	var errorResponse ErrorResponse
	err := xmlDecoder(errBody, &errorResponse)
	if err != nil {
		return err
	}
	return errorResponse
}

// ErrInvalidBucketName - invalid bucket name response.
var ErrInvalidBucketName = func() error {
	return ErrorResponse{
		Code:      "InvalidBucketName",
		Message:   "The specified bucket is not valid.",
		RequestID: "minio",
	}
}

// ErrInvalidObjectName - invalid object name response.
var ErrInvalidObjectName = func() error {
	return ErrorResponse{
		Code:      "NoSuchKey",
		Message:   "The specified key does not exist.",
		RequestID: "minio",
	}
}

// ErrInvalidArgument - invalid argument response.
var ErrInvalidArgument = func(message string) error {
	return ErrorResponse{
		Code:      "InvalidArgument",
		Message:   message,
		RequestID: "minio",
	}
}
