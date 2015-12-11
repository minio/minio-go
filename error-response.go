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
	"fmt"
	"io"
	"strconv"
)

/* **** SAMPLE ERROR RESPONSE ****
<?xml version="1.0" encoding="UTF-8"?>
<Error>
   <Code>AccessDenied</Code>
   <Message>Access Denied</Message>
   <BucketName>bucketName</BucketName>
   <Key>objectName</Key>
   <RequestId>F19772218238A85A</RequestId>
   <HostId>GuWkjyviSiGHizehqpmsD1ndz5NClSP19DOT+s2mv7gXGQ8/X1lhbDGiIJEXpGFD</HostId>
</Error>
*/

// ErrorResponse is the type error returned by some API operations.
type ErrorResponse struct {
	XMLName    xml.Name `xml:"Error" json:"-"`
	Code       string
	Message    string
	BucketName string
	Key        string
	RequestID  string `xml:"RequestId"`
	HostID     string `xml:"HostId"`

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
func ToErrorResponse(err error) ErrorResponse {
	switch err := err.(type) {
	case ErrorResponse:
		return err
	default:
		return ErrorResponse{}
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

// ErrEntityTooLarge input size is larger than supported maximum.
func ErrEntityTooLarge(totalSize int64, bucketName, objectName string) error {
	msg := fmt.Sprintf("Your proposed upload size ‘%d’ exceeds the maximum allowed object size '5GB' for single PUT operation.", totalSize)
	return ErrorResponse{
		Code:       "EntityTooLarge",
		Message:    msg,
		BucketName: bucketName,
		Key:        objectName,
	}
}

// ErrUnexpectedShortRead unexpected shorter read of input buffer from target.
func ErrUnexpectedShortRead(totalRead, totalSize int64, bucketName, objectName string) error {
	msg := fmt.Sprintf("Data read ‘%s’ is shorter than the size ‘%s’ of input buffer.",
		strconv.FormatInt(totalRead, 10), strconv.FormatInt(totalSize, 10))
	return ErrorResponse{
		Code:       "UnexpectedShortRead",
		Message:    msg,
		BucketName: bucketName,
		Key:        objectName,
	}
}

// ErrUnexpectedEOF unexpected end of file reached.
func ErrUnexpectedEOF(totalRead, totalSize int64, bucketName, objectName string) error {
	msg := fmt.Sprintf("Data read ‘%s’ is not equal to the size ‘%s’ of the input Reader.",
		strconv.FormatInt(totalRead, 10), strconv.FormatInt(totalSize, 10))
	return ErrorResponse{
		Code:       "UnexpectedEOF",
		Message:    msg,
		BucketName: bucketName,
		Key:        objectName,
	}
}

// ErrInvalidBucketName - invalid bucket name response.
func ErrInvalidBucketName(message string) error {
	return ErrorResponse{
		Code:      "InvalidBucketName",
		Message:   message,
		RequestID: "minio",
	}
}

// ErrInvalidObjectName - invalid object name response.
func ErrInvalidObjectName(message string) error {
	return ErrorResponse{
		Code:      "NoSuchKey",
		Message:   message,
		RequestID: "minio",
	}
}

// ErrInvalidObjectPrefix - invalid object prefix response is
// similar to object name response.
var ErrInvalidObjectPrefix = ErrInvalidObjectName

// ErrInvalidArgument - invalid argument response.
func ErrInvalidArgument(message string) error {
	return ErrorResponse{
		Code:      "InvalidArgument",
		Message:   message,
		RequestID: "minio",
	}
}
