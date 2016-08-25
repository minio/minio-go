/*
 * Minio Go Library for Amazon S3 Compatible Cloud Storage (C) 2015, 2016 Minio, Inc.
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
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// GetObject - returns an seekable, readable object.
func (c Client) GetObject(bucketName, objectName string) (*Object, error) {
	// Input validation.
	if err := isValidBucketName(bucketName); err != nil {
		return nil, err
	}
	if err := isValidObjectName(objectName); err != nil {
		return nil, err
	}

	var httpReader io.ReadCloser
	var objectInfo ObjectInfo
	var err error

	// Create start channel.
	startCh := make(chan firstRequest)
	// Create first response channel.
	firstResCh := make(chan firstReqRes)
	// Create request channel.
	reqCh := make(chan readRequest)
	// Create response channel.
	resCh := make(chan readResponse)
	// Create done channel.
	doneCh := make(chan struct{})

	// This routine feeds partial object data as and when the caller reads.
	go func() {
		defer close(reqCh)
		defer close(resCh)
		defer close(startCh)
		defer close(firstResCh)

		// Loop through the incoming control messages and read data.
		for {
			select {
			// This channel signals that we have received the first request for this object.
			// Could be either: Read/ReadAt/Stat/Seek.
			case firstReq := <-startCh:
				// First request is a Read/ReadAt.
				if firstReq.isReadOp {
					// Differentiate between wanting the whole object and just a range.
					if firstReq.isReadAt {
						// If this is a ReadAt request only get the specified range.
						// Range is set with respect to the offset and length of the buffer requested.
						httpReader, objectInfo, err = c.getObject(bucketName, objectName, firstReq.Offset, int64(len(firstReq.Buffer)))
					} else {
						// First request is a Read request.
						httpReader, objectInfo, err = c.getObject(bucketName, objectName, firstReq.Offset, 0)
					}
					if err != nil {
						firstResCh <- firstReqRes{
							Error: err,
						}
						return
					}
					// Read at least firstReq.Buffer bytes, if not we have
					// reached our EOF.
					size, err := io.ReadFull(httpReader, firstReq.Buffer)
					if err == io.ErrUnexpectedEOF {
						// If an EOF happens after reading some but not
						// all the bytes ReadFull returns ErrUnexpectedEOF
						err = io.EOF
					}
					firstResCh <- firstReqRes{
						objectInfo: objectInfo,
						Size:       int(size),
						Error:      err,
						didRead:    true,
					}
				} else {
					// First request is a Stat or Seek call.
					// Only need to run a StatObject until an actual Read or ReadAt request comes through.
					objectInfo, err = c.StatObject(bucketName, objectName)
					if err != nil {
						firstResCh <- firstReqRes{
							Error: err,
						}
						return
					}
					firstResCh <- firstReqRes{
						objectInfo: objectInfo,
					}
				}
			// When the done channel is closed exit our routine.
			case <-doneCh:
				// Close the http response body before returning.
				// This ends the connection with the server.
				if httpReader != nil {
					httpReader.Close()
				}
				return
			// Subsequent requests.
			case req := <-reqCh:
				// Offset changes fetch the new object at an Offset.
				// Because the httpReader may not be set by the first
				// request if it was a stat or seek it must be checked
				// if the object has been read or not to only initialize
				// new ones when they haven't been already.
				// All readAt requests are new requests.
				if req.DidOffsetChange || !req.beenRead || req.isReadAt {
					if httpReader != nil {
						// Close previously opened http reader.
						httpReader.Close()
					}
					// If this request is a readAt only get the specified range.
					if req.isReadAt {
						// Range is set with respect to the offset and length of the buffer requested.
						httpReader, _, err = c.getObject(bucketName, objectName, req.Offset, int64(len(req.Buffer)))
					} else {
						httpReader, _, err = c.getObject(bucketName, objectName, req.Offset, 0)
					}
					if err != nil {
						resCh <- readResponse{
							Error: err,
						}
						return
					}
				}

				// Read at least req.Buffer bytes, if not we have
				// reached our EOF.
				size, err := io.ReadFull(httpReader, req.Buffer)
				if err == io.ErrUnexpectedEOF {
					// If an EOF happens after reading some but not
					// all the bytes ReadFull returns ErrUnexpectedEOF
					err = io.EOF
				}
				// Reply back how much was read.
				resCh <- readResponse{
					Size:  int(size),
					Error: err,
				}
			}
		}
	}()
	// Create a newObject through the information sent back by firstReq.
	return newObject(startCh, reqCh, resCh, doneCh, firstResCh), nil
}

// Read response message container to reply back for the request.
type readResponse struct {
	Size  int
	Error error
}

// firstRequest message container to communicate with internal
// go-routine on the first Stat/Read/ReadAt/Seek request.
type firstRequest struct {
	isReadOp bool   // Determines if this request is a ReadAt/Read request of Seek/Stat.
	Offset   int64  // Used for ReadAt offset.
	Buffer   []byte // Used for ReadAt/Read requests.
	isReadAt bool   // Determines if this request is a ReadAt request to allow "getting" only a specific range.
}

// The first request asked must send back the objectInfo or an error to create a new Object.
type firstReqRes struct {
	Error      error
	objectInfo ObjectInfo
	Size       int
	didRead    bool // Lets know subsequent calls whether or not httpReader has been initiated.
}

// Read request message container to communicate with internal
// go-routine.
type readRequest struct {
	Buffer          []byte
	Offset          int64 // readAt offset.
	DidOffsetChange bool
	beenRead        bool
	isReadAt        bool // Determines if this request is a ReadAt request to allow "getting" only a specific range
}

// Object represents an open object. It implements Read, ReadAt,
// Seeker, Close for a HTTP stream.
type Object struct {
	// Mutex.
	mutex *sync.Mutex

	// User allocated and defined.
	startCh    chan<- firstRequest
	firstResCh <-chan firstReqRes
	reqCh      chan<- readRequest
	resCh      <-chan readResponse
	doneCh     chan<- struct{}
	prevOffset int64
	currOffset int64
	objectInfo ObjectInfo

	// Keeps track of closed call.
	isClosed bool

	// Keeps track of if this is the first call.
	isStarted bool

	// Previous error saved for future calls.
	prevErr error

	// Keeps track of if this object has been read yet.
	beenRead bool
}

// setObjectInfo - blocks until the first request is completed and then either errors out
// or sets the received objectInfo.
func (o *Object) setObjectInfo() (int, error) {
	firstResponse := <-o.firstResCh
	// The first response has succesfully returned.
	o.isStarted = true
	// Error out here if the firstRequest caused an error.
	if firstResponse.Error != nil {
		o.prevErr = firstResponse.Error
		return 0, firstResponse.Error
	}
	// Set the objectInfo.
	o.objectInfo = firstResponse.objectInfo
	// Set if the object was read as part of the first request.
	o.beenRead = firstResponse.didRead
	return firstResponse.Size, nil
}

// Read reads up to len(p) bytes into p. It returns the number of
// bytes read (0 <= n <= len(p)) and any error encountered. Returns
// io.EOF upon end of file.
func (o *Object) Read(b []byte) (n int, err error) {
	if o == nil {
		return 0, ErrInvalidArgument("Object is nil")
	}

	// Locking.
	o.mutex.Lock()
	defer o.mutex.Unlock()

	// This is the first request.
	if !o.isStarted {
		// Create the first request.
		firstReq := firstRequest{
			isReadOp: true,
			Offset:   0,
			Buffer:   b,
		}
		// Send the first request.
		o.startCh <- firstReq
		// Set the objectInfo from what was returned by the first request.
		size, err := o.setObjectInfo()
		if err != nil {
			return 0, err
		}

		// Bytes read.
		bytesRead := int64(size)

		// Update current offset.
		o.currOffset += bytesRead

		// Save the current offset as previous offset.
		o.prevOffset = o.currOffset

		// If currOffset read is equal to objectSize
		// We have reached end of file, we return io.EOF.
		if o.currOffset >= o.objectInfo.Size {
			return size, io.EOF
		}
		return size, nil
	}
	// prevErr is previous error saved from previous operation.
	if o.prevErr != nil || o.isClosed {
		return 0, o.prevErr
	}

	// If current offset has reached Size limit, return EOF.
	if o.currOffset >= o.objectInfo.Size {
		return 0, io.EOF
	}

	// Send current information over control channel to indicate we are ready.
	reqMsg := readRequest{}
	// Send the pointer to the buffer over the channel.
	reqMsg.Buffer = b

	// Set if the object has been read yet.
	reqMsg.beenRead = o.beenRead

	// Verify if offset has changed and currOffset is greater than
	// previous offset. Perhaps due to Seek().
	offsetChange := o.prevOffset - o.currOffset
	if offsetChange < 0 {
		offsetChange = -offsetChange
	}
	if offsetChange > 0 {
		// Fetch the new reader at the current offset again.
		reqMsg.Offset = o.currOffset
		reqMsg.DidOffsetChange = true
	} else {
		// No offset changes no need to fetch new reader, continue
		// reading.
		reqMsg.DidOffsetChange = false
		reqMsg.Offset = 0
	}

	// Send read request over the control channel.
	o.reqCh <- reqMsg

	// Get data over the response channel.
	dataMsg := <-o.resCh

	// Object has now been read.
	o.beenRead = true

	// Bytes read.
	bytesRead := int64(dataMsg.Size)

	// Update current offset.
	o.currOffset += bytesRead

	// Save the current offset as previous offset.
	o.prevOffset = o.currOffset

	if dataMsg.Error == nil {
		// If currOffset read is equal to objectSize
		// We have reached end of file, we return io.EOF.
		if o.currOffset >= o.objectInfo.Size {
			return dataMsg.Size, io.EOF
		}
		return dataMsg.Size, nil
	}

	// Save any error.
	o.prevErr = dataMsg.Error
	return dataMsg.Size, dataMsg.Error
}

// Stat returns the ObjectInfo structure describing object.
func (o *Object) Stat() (ObjectInfo, error) {
	if o == nil {
		return ObjectInfo{}, ErrInvalidArgument("Object is nil")
	}
	// Locking.
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if !o.isStarted {
		// Create the first request.
		firstReq := firstRequest{
			isReadOp: false, // This is a Stat not a Read/ReadAt.
			Offset:   0,
		}
		// Send the first request.
		o.startCh <- firstReq
		if _, err := o.setObjectInfo(); err != nil {
			return ObjectInfo{}, err
		}

		return o.objectInfo, nil
	}
	if o.prevErr != nil || o.isClosed {
		return ObjectInfo{}, o.prevErr
	}

	return o.objectInfo, nil
}

// ReadAt reads len(b) bytes from the File starting at byte offset
// off. It returns the number of bytes read and the error, if any.
// ReadAt always returns a non-nil error when n < len(b). At end of
// file, that error is io.EOF.
func (o *Object) ReadAt(b []byte, offset int64) (n int, err error) {
	if o == nil {
		return 0, ErrInvalidArgument("Object is nil")
	}

	// Locking.
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if !o.isStarted {
		// Create the first request.
		firstReq := firstRequest{
			isReadOp: true,
			Offset:   offset,
			Buffer:   b,
			isReadAt: true, // This is a readAt request.
		}
		// Send the first request.
		o.startCh <- firstReq

		// Get the amount read and set the objectInfo.
		size, err := o.setObjectInfo()
		if err != nil {
			return 0, err
		}

		// Bytes read.
		bytesRead := int64(size)

		// Update current offset.
		o.currOffset += bytesRead

		// Save current offset as previous offset before returning.
		o.prevOffset = o.currOffset

		// If currOffset read is equal to objectSize
		// We have reached end of file, we return io.EOF.
		if o.currOffset >= o.objectInfo.Size {
			return size, io.EOF
		}
		return size, nil
	}
	// prevErr is error which was saved in previous operation.
	if o.prevErr != nil || o.isClosed {
		return 0, o.prevErr
	}

	// if offset is greater than or equal to object size we return io.EOF.
	// If offset is negative then we return io.EOF.
	if offset < 0 || offset >= o.objectInfo.Size {
		return 0, io.EOF
	}

	// Send current information over control channel to indicate we
	// are ready.
	reqMsg := readRequest{}
	// Set if this is the first read request or not.
	reqMsg.beenRead = o.beenRead
	// Send the offset and pointer to the buffer over the channel.
	reqMsg.Buffer = b
	// Notify that we are only getting a range of the object.
	reqMsg.isReadAt = true

	reqMsg.DidOffsetChange = offset != o.currOffset
	// Set the offset.
	reqMsg.Offset = offset

	// Send read request over the control channel.
	o.reqCh <- reqMsg

	// Get data over the response channel.
	dataMsg := <-o.resCh

	// Object has now been read.
	o.beenRead = true

	// Bytes read.
	bytesRead := int64(dataMsg.Size)

	// Update current offset.
	o.currOffset += bytesRead

	// Save current offset as previous offset before returning.
	o.prevOffset = o.currOffset

	if dataMsg.Error == nil {
		// If currentOffset is equal to objectSize
		// we have reached end of file, we return io.EOF.
		if o.currOffset >= o.objectInfo.Size {
			return dataMsg.Size, io.EOF
		}
		return dataMsg.Size, nil
	}

	// Save any error.
	o.prevErr = dataMsg.Error
	return dataMsg.Size, dataMsg.Error
}

// Seek sets the offset for the next Read or Write to offset,
// interpreted according to whence: 0 means relative to the
// origin of the file, 1 means relative to the current offset,
// and 2 means relative to the end.
// Seek returns the new offset and an error, if any.
//
// Seeking to a negative offset is an error. Seeking to any positive
// offset is legal, subsequent io operations succeed until the
// underlying object is not closed.
func (o *Object) Seek(offset int64, whence int) (n int64, err error) {
	if o == nil {
		return 0, ErrInvalidArgument("Object is nil")
	}

	// Locking.
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if !o.isStarted {
		// This is the first request.
		firstReq := firstRequest{
			isReadOp: false,
			Offset:   0,
		}
		// Send the first request.
		o.startCh <- firstReq
		// Set the objectInfo.
		_, err := o.setObjectInfo()
		if err != nil {
			return 0, err
		}
	}

	if o.prevErr != nil {
		// At EOF seeking is legal allow only io.EOF, for any other errors we return.
		if o.prevErr != io.EOF {
			return 0, o.prevErr
		}
	}

	// Negative offset is valid for whence of '2'.
	if offset < 0 && whence != 2 {
		return 0, ErrInvalidArgument(fmt.Sprintf("Negative position not allowed for %d.", whence))
	}

	// Save current offset as previous offset.
	o.prevOffset = o.currOffset

	// Switch through whence.
	switch whence {
	default:
		return 0, ErrInvalidArgument(fmt.Sprintf("Invalid whence %d", whence))
	case 0:
		if offset > o.objectInfo.Size {
			return 0, io.EOF
		}
		o.currOffset = offset
	case 1:
		if o.currOffset+offset > o.objectInfo.Size {
			return 0, io.EOF
		}
		o.currOffset += offset
	case 2:
		// Seeking to positive offset is valid for whence '2', but
		// since we are backing a Reader we have reached 'EOF' if
		// offset is positive.
		if offset > 0 {
			return 0, io.EOF
		}
		// Seeking to negative position not allowed for whence.
		if o.objectInfo.Size+offset < 0 {
			return 0, ErrInvalidArgument(fmt.Sprintf("Seeking at negative offset not allowed for %d", whence))
		}
		o.currOffset = o.objectInfo.Size + offset
	}
	// Reset the saved error since we successfully seeked, let the Read
	// and ReadAt decide.
	if o.prevErr == io.EOF {
		o.prevErr = nil
	}
	// Return the effective offset.
	return o.currOffset, nil
}

// Close - The behavior of Close after the first call returns error
// for subsequent Close() calls.
func (o *Object) Close() (err error) {
	if o == nil {
		return ErrInvalidArgument("Object is nil")
	}
	// Locking.
	o.mutex.Lock()
	defer o.mutex.Unlock()

	// if already closed return an error.
	if o.isClosed {
		return o.prevErr
	}

	// Close successfully.
	close(o.doneCh)

	// Save for future operations.
	errMsg := "Object is already closed. Bad file descriptor."
	o.prevErr = errors.New(errMsg)
	// Save here that we closed done channel successfully.
	o.isClosed = true
	return nil
}

// newObject instantiates a new *minio.Object*
// ObjectInfo will be set by setObjectInfo
func newObject(startCh chan<- firstRequest, reqCh chan<- readRequest, resCh <-chan readResponse, doneCh chan<- struct{}, firstResCh <-chan firstReqRes) *Object {
	return &Object{
		mutex:      &sync.Mutex{},
		reqCh:      reqCh,
		resCh:      resCh,
		doneCh:     doneCh,
		startCh:    startCh,
		firstResCh: firstResCh,
	}
}

// getObject - retrieve object from Object Storage.
//
// Additionally this function also takes range arguments to download the specified
// range bytes of an object. Setting offset and length = 0 will download the full object.
//
// For more information about the HTTP Range header.
// go to http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.35.
func (c Client) getObject(bucketName, objectName string, offset, length int64) (io.ReadCloser, ObjectInfo, error) {
	// Validate input arguments.
	if err := isValidBucketName(bucketName); err != nil {
		return nil, ObjectInfo{}, err
	}
	if err := isValidObjectName(objectName); err != nil {
		return nil, ObjectInfo{}, err
	}

	customHeader := make(http.Header)
	// Set ranges if length and offset are valid.
	// See  https://tools.ietf.org/html/rfc7233#section-3.1 for reference.
	if length > 0 && offset >= 0 {
		customHeader.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, offset+length-1))
	} else if offset > 0 && length == 0 {
		customHeader.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	} else if length < 0 && offset == 0 {
		customHeader.Set("Range", fmt.Sprintf("bytes=%d", length))
	}

	// Execute GET on objectName.
	resp, err := c.executeMethod("GET", requestMetadata{
		bucketName:   bucketName,
		objectName:   objectName,
		customHeader: customHeader,
	})
	if err != nil {
		return nil, ObjectInfo{}, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
			return nil, ObjectInfo{}, httpRespToErrorResponse(resp, bucketName, objectName)
		}
	}

	// Trim off the odd double quotes from ETag in the beginning and end.
	md5sum := strings.TrimPrefix(resp.Header.Get("ETag"), "\"")
	md5sum = strings.TrimSuffix(md5sum, "\"")

	// Parse the date.
	date, err := time.Parse(http.TimeFormat, resp.Header.Get("Last-Modified"))
	if err != nil {
		msg := "Last-Modified time format not recognized. " + reportIssue
		return nil, ObjectInfo{}, ErrorResponse{
			Code:      "InternalError",
			Message:   msg,
			RequestID: resp.Header.Get("x-amz-request-id"),
			HostID:    resp.Header.Get("x-amz-id-2"),
			Region:    resp.Header.Get("x-amz-bucket-region"),
		}
	}
	// Get content-type.
	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	var objectStat ObjectInfo
	objectStat.ETag = md5sum
	objectStat.Key = objectName
	objectStat.Size = resp.ContentLength
	objectStat.LastModified = date
	objectStat.ContentType = contentType

	// do not close body here, caller will close
	return resp.Body, objectStat, nil
}
