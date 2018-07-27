/*
 * Minio Go Library for Amazon S3 Compatible Cloud Storage
 * (C) 2018 Minio, Inc.
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
	"context"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"hash"
	"hash/crc32"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// CSVFileHeaderInfo - is the parameter for whether to utilize headers.
type CSVFileHeaderInfo string

// Constants for file header info.
const (
	CSVFileHeaderInfoNone   CSVFileHeaderInfo = "NONE"
	CSVFileHeaderInfoIgnore                   = "IGNORE"
	CSVFileHeaderInfoUse                      = "USE"
)

// SelectCompressionType - is the parameter for what type of compression is
// present
type SelectCompressionType string

// Constants for compression types under select API.
const (
	SelectCompressionNONE SelectCompressionType = "NONE"
	SelectCompressionGZIP                       = "GZIP"
	SelectCompressionBZIP                       = "BZIP2"
)

// CSVQuoteFields - is the parameter for how CSV fields are quoted.
type CSVQuoteFields string

// Constants for csv quote styles.
const (
	CSVQuoteFieldsAlways   CSVQuoteFields = "Always"
	CSVQuoteFieldsAsNeeded                = "AsNeeded"
)

// QueryExpressionType - is of what syntax the expression is, this should only
// be SQL
type QueryExpressionType string

// Constants for expression type.
const (
	QueryExpressionTypeSQL QueryExpressionType = "SQL"
)

// JSONType determines json input serialization type.
type JSONType string

// Constants for JSONTypes.
const (
	JSONDocumentType JSONType = "Document"
	JSONStreamType            = "Stream"
	JSONLinesType             = "Lines"
)

// ObjectSelectRequest - represents the input select body
type ObjectSelectRequest struct {
	XMLName            xml.Name `xml:"SelectObjectContentRequest" json:"-"`
	Expression         string
	ExpressionType     QueryExpressionType
	InputSerialization struct {
		CompressionType SelectCompressionType
		CSV             struct {
			FileHeaderInfo       CSVFileHeaderInfo
			RecordDelimiter      string
			FieldDelimiter       string
			QuoteCharacter       string
			QuoteEscapeCharacter string
			Comments             string
		}
	}
	OutputSerialization struct {
		CSV struct {
			QuoteFields          CSVQuoteFields
			RecordDelimiter      string
			FieldDelimiter       string
			QuoteCharacter       string
			QuoteEscapeCharacter string
		}
	}
}

// SelectObjectType - is the parameter which defines what type of object the
// operation is being performed on.
type SelectObjectType string

// Constants for input data types.
const (
	SelectObjectTypeCSV  SelectObjectType = "CSV"
	SelectObjectTypeJSON                  = "JSON"
)

// SelectObjectInput is a struct for specifying input options.
type SelectObjectInput struct {
	RecordDelimiter string
	FieldDelimiter  string
	Comments        string
	FileHeaderInfo  CSVFileHeaderInfo
}

// SelectObjectOutput is a struct for specifying output options.
type SelectObjectOutput struct {
	RecordDelimiter string
	FieldDelimiter  string
}

// SelectObjectOptions - Encapsulates all of the user options.
type SelectObjectOptions struct {
	Type   SelectObjectType
	Input  SelectObjectInput
	Output SelectObjectOutput
}

// preludeInfo is used for keeping track of necessary information from the
// prelude.
type preludeInfo struct {
	totalLen  uint32
	headerLen uint32
}

// headerInfo is used for keeping track of the headers from the server.
type headerInfo struct {
	messageType  string
	errorType    string
	errorMessage string
}

// SelectResults is used for the streaming responses from the server.
type SelectResults struct {
	Response chan interface{}
	writer   *bytes.Buffer
	lock     chan bool
	stat     *StatEvent
	progress *ProgressEvent
	err      error
}

// RecordEvent is a struct that wraps the byte array of a single record.
type RecordEvent struct {
	Payload []byte
}

// ProgressEvent is a struct that wraps the byte array of a progress xml message.
type ProgressEvent struct {
	Payload []byte
}

// StatEvent is a struct that wraps the byte array of a stat xml message.
type StatEvent struct {
	Payload []byte
}

// EndEvent is a struct that indicates that the response stream has ended.
type EndEvent struct {
}

// ToObjectSelectRequest - generate an object select request statement.
func (opts SelectObjectOptions) ToObjectSelectRequest(expression string, compressionType SelectCompressionType) *ObjectSelectRequest {
	osreq := &ObjectSelectRequest{
		Expression:     expression,
		ExpressionType: QueryExpressionTypeSQL,
	}
	osreq.InputSerialization.CompressionType = compressionType
	osreq.InputSerialization.CSV.FieldDelimiter = opts.Input.FieldDelimiter
	osreq.InputSerialization.CSV.RecordDelimiter = opts.Input.RecordDelimiter
	osreq.InputSerialization.CSV.Comments = opts.Input.Comments
	osreq.InputSerialization.CSV.FileHeaderInfo = opts.Input.FileHeaderInfo

	// Only supports filed and record delimiter for now.
	osreq.OutputSerialization.CSV.FieldDelimiter = opts.Output.FieldDelimiter
	osreq.OutputSerialization.CSV.RecordDelimiter = opts.Output.RecordDelimiter

	return osreq
}

// SelectObjectContent is a implementation of http://docs.aws.amazon.com/AmazonS3/latest/API/RESTObjectSELECTContent.html AWS S3 API.
func (c Client) SelectObjectContent(ctx context.Context, bucketName, objectName, expression string, opts SelectObjectOptions) (*SelectResults, error) {
	objInfo, err := c.statObject(ctx, bucketName, objectName, StatObjectOptions{})
	if err != nil {
		return nil, err
	}

	var compressionType = SelectCompressionNONE
	if strings.Contains(objInfo.ContentType, "gzip") {
		compressionType = SelectCompressionGZIP
	} else if strings.Contains(objInfo.ContentType, "text/csv") && opts.Type == "" {
		opts.Type = SelectObjectTypeCSV
	} else if strings.Contains(objInfo.ContentType, "json") && opts.Type == "" {
		opts.Type = SelectObjectTypeJSON
	}

	selectReqBytes, err := xml.Marshal(opts.ToObjectSelectRequest(expression, compressionType))

	if err != nil {
		return nil, err
	}

	urlValues := make(url.Values)
	urlValues.Set("select", "")
	urlValues.Set("select-type", "2")
	// Execute POST on bucket/object.
	resp, err := c.executeMethod(ctx, "POST", requestMetadata{
		bucketName:       bucketName,
		objectName:       objectName,
		queryValues:      urlValues,
		contentMD5Base64: sumMD5Base64(selectReqBytes),
		contentSHA256Hex: sum256Hex(selectReqBytes),
		contentBody:      bytes.NewReader(selectReqBytes),
		contentLength:    int64(len(selectReqBytes)),
	})
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, httpRespToErrorResponse(resp, bucketName, "")
	}
	// Process needs to return a eventStream to the client
	// Wrap this in a Go routine so as to allow the streaming
	myWriter := bytes.Buffer{}
	streamer := &SelectResults{
		Response: make(chan interface{}, 2),
		writer:   &myWriter,
		lock:     make(chan bool, 2),
		stat:     nil,
		progress: nil,
	}
	go func() {
		processBinary(resp.Body, streamer)
	}()
	return streamer, nil
}

// processBinary is the main function that decodes the large byte array into
// several events that are sent through the eventstream.
func processBinary(message io.ReadCloser, streamer *SelectResults) {
	defer message.Close()
	var partialRecordVal []byte
	lenOfTrim := 0
	for {
		// select block for catching a user request for closing.
		select {
		// Logic is basically that the .Close() method will close the streamer.lock
		// channel, Therefore if Ok is false(the channel is closed) then we can
		// close our response channel and return. The .Close() method blocks on the
		// streamer.Response channel being closed, so that it is guaranteed that this
		// method has exited since it is a defered close.
		case _, ok := <-streamer.lock:
			if !ok {
				defer close(streamer.Response)
				return
			}
		default:
			var prelude preludeInfo
			var headers *headerInfo
			// Create CRC code
			crc := crc32.New(crc32.IEEETable)
			crcReader := io.TeeReader(message, crc)
			var err error
			var payload []byte
			// Extract the prelude(12 bytes) into a struct to extract relevant
			// information.
			prelude, err = processPrelude(crcReader, crc)
			if err != nil {
				if streamer.err == nil {
					streamer.err = err
				}
				close(streamer.Response)
				return
			}
			// Extract the headers(variable bytes) into a struct to extract relevant
			// information
			if prelude.headerLen > 0 {
				headers, err = extractHeader(io.LimitReader(crcReader, int64(prelude.headerLen)))
				if err != nil {
					if streamer.err == nil {
						streamer.err = err
					}
					close(streamer.Response)
					return
				}
			}
			// Calculate the length of the payload so that the appropriate amount of
			// bytes can be read
			payloadLen := calcPayloadLen(prelude)
			// Checks that there is a payload and if there is, then extracts it.
			if payloadLen > 0 {
				payload, err = extractPayload(io.LimitReader(crcReader, int64(payloadLen)), payloadLen)
				if err != nil {
					if streamer.err == nil {
						streamer.err = err
					}
					close(streamer.Response)
					return
				}
				var state bool
				for isPartial(payload) {
					// state indicates whether it is a partial record or not, so if it
					// is false then only processPartial should be called
					payload, partialRecordVal, lenOfTrim = processPartial(payload)
					// sends complete records to client, while partial records are
					// stitched.
					processEvent(payload, headers, streamer)
					// return the complete part of the message so far to the client
					// for loop in case there are multiple sequential partial rows
					// Extracts the partial payload.
					myPayload, err := extractPayload(io.LimitReader(crcReader, int64(lenOfTrim)), lenOfTrim)
					if err != nil {
						if streamer.err == nil {
							streamer.err = err
						}
						close(streamer.Response)
						return
					}
					// knits together the payload with the previous partial one.
					payload = append(partialRecordVal, myPayload...)
					// update state to keep track of whether partial records were
					// processed
					state = !isPartial(payload)
				}
				// state is used to make sure that this processEvent is only called in
				// the case of a partialRecord scenario and never when there is a
				// complete record.
				if state {
					processEvent(payload, headers, streamer)
					payload = nil
				}
			}
			// Ensures that the full message's CRC is correct and that the message is
			// not corrupted
			if err := checkCRC(message, crc.Sum32()); err != nil {
				if streamer.err == nil {
					streamer.err = err
				}
				close(streamer.Response)
				return
			}

			// Processes the Event and wraps it in the appropriate class, if it is the
			// end then returns.
			if payload != nil || headers.messageType == "End" || headers.messageType == "Cont" {
				processEvent(payload, headers, streamer)
				if headers.messageType == "End" {
					// set EOF error if the End header is processed.
					if streamer.err == nil {
						streamer.err = io.EOF
					}
					// close response channel
					close(streamer.Response)
					return
				}
			}
		}
	}
}

// isPartial is a function which returns whether a message's payload is
// partial
func isPartial(payload []byte) bool {
	if len(payload) < 3 {
		return false
	}
	return !(payload[len(payload)-1] == byte(10) && payload[len(payload)-2] == byte(13)) && payload[len(payload)-3] == byte(0)
}

// processPartial is the function which helps to stitch together rows when
// messages return partial rows
func processPartial(payload []byte) ([]byte, []byte, int) {
	myPosition := -1
	for i := len(payload) - 1; i >= 1; i-- {
		if payload[i] == byte(10) && payload[i-1] == byte(13) {
			temp := make([]byte, (myPosition - i))
			copy(temp, payload[(i+1):(myPosition+1)])
			return payload[:i+1], temp, len(payload) - myPosition - 1
		}
		if payload[i] != byte(0) && myPosition == -1 {
			myPosition = i
		}
	}
	return payload, nil, 0
}

// processEvent is a function that takes the extracted information from the
// headers and payload to return a particular event.
func processEvent(payload []byte, headers *headerInfo, streamer *SelectResults) {
	switch headers.messageType {
	case "Records":
		_, err := streamer.writer.Write(payload)
		if err != nil {
			if streamer.err == nil {
				streamer.err = err
			}
		}
	case "Progress":
		myProgress := &ProgressEvent{
			Payload: payload,
		}
		streamer.progress = myProgress
	case "Stats":
		myStat := &StatEvent{
			Payload: payload,
		}
		streamer.stat = myStat
	case "Error":
		streamer.err = errors.New("Error Type of " + headers.errorType + " " + headers.errorMessage)
	}
	return
}

// extractPayload is a function that reads the payload into a byte array.
func extractPayload(payload io.Reader, size int) ([]byte, error) {
	myVal := make([]byte, size)
	_, err := payload.Read(myVal)
	if err == io.EOF {
		return myVal, nil
	}
	return myVal, err
}

// calcPayloadLen is a function that calculates the length of the payload.
func calcPayloadLen(preludeInfo preludeInfo) int {
	return int(preludeInfo.totalLen - preludeInfo.headerLen - 16)
}

// processPrelude is the function that reads the 12 bytes of the prelude and
// ensures the CRC is correct while also extracting relevant information into
// the struct,
func processPrelude(prelude io.Reader, crc hash.Hash32) (preludeInfo, error) {
	var preludeStore preludeInfo
	var err error
	// reads total length of the message (first 4 bytes)
	preludeStore.totalLen, err = extractUint32(prelude)
	if err != nil {
		return preludeStore, err
	}
	// reads total header length of the message (2nd 4 bytes)
	preludeStore.headerLen, err = extractUint32(prelude)
	if err != nil {
		return preludeStore, err
	}
	// checks that the CRC is correct (3rd 4 bytes)
	preCRC := crc.Sum32()
	if err := checkCRC(prelude, preCRC); err != nil {
		return preludeStore, err
	}
	return preludeStore, nil
}

// extracts the relevant information from the Headers.
func extractHeader(headers io.Reader) (*headerInfo, error) {
	var myHeader *headerInfo
	for {
		// extracts the first part of the header,
		headerTypeName, err := extractHeaderType(headers)
		if err != nil {
			//since end of file, we have read all of our headers
			if err == io.EOF {
				break
			}
			return nil, err
		}
		// reads the 7 present in the header.
		extractUint8(headers)
		headerValueName, err := extractHeaderValue(headers)
		if err != nil {
			return nil, err
		}
		// following set of if clauses controls what headers are extracted.
		if headerTypeName == ":event-type" {
			myHeader = &headerInfo{
				messageType: headerValueName,
			}
		} else if headerTypeName == ":error-code" {
			myHeader = &headerInfo{
				messageType: "Error",
				errorType:   headerValueName,
			}
		} else if headerTypeName == ":error-message" {
			myHeader.errorMessage = headerValueName
		}
	}
	return myHeader, nil
}

// extractHeaderType extracts the first half of the header message, the header
// type.
func extractHeaderType(headerType io.Reader) (string, error) {
	// extracts 2 bit integer
	headerNameLen, err := extractUint8(headerType)
	if err != nil {
		return "", err
	}
	// extracts the string with the appropriate number of bytes
	headerTypeName, err := extractString(headerType, int(headerNameLen))
	if err != nil {
		return "", err
	}
	return headerTypeName, nil
}

// extractsHeaderValue extracts the second half of the header message, the
// header value
func extractHeaderValue(headerValue io.Reader) (string, error) {
	headerValueLen, err := extractUint16(headerValue)
	if err != nil {
		return "", err
	}
	headerValueName, err := extractString(headerValue, int(headerValueLen))
	if err != nil {
		return "", err
	}
	return headerValueName, nil
}

// extracts a string from byte array of a particular number of bytes.
func extractString(source io.Reader, lenBytes int) (string, error) {
	myVal := make([]byte, lenBytes)
	_, err := source.Read(myVal)
	if err != nil {
		return "", err
	}
	return string(myVal), nil
}

// extractUint32 extracts a 4 byte integer from the byte array.
func extractUint32(r io.Reader) (uint32, error) {
	buf := make([]byte, 4)
	_, err := io.ReadFull(r, buf)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(buf), nil
}

// extractUint16 extracts a 2 byte integer from the byte array.
func extractUint16(r io.Reader) (uint16, error) {
	buf := make([]byte, 2)
	_, err := io.ReadFull(r, buf)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(buf), nil
}

// extractUint8 extracts a 1 byte integer from the byte array.
func extractUint8(r io.Reader) (uint8, error) {
	buf := make([]byte, 1)
	_, err := io.ReadFull(r, buf)
	if err != nil {
		return 0, err
	}
	return buf[0], nil
}

// checkCRC ensures that the CRC matches with the one from the reader.
func checkCRC(r io.Reader, expect uint32) error {
	msgCRC, err := extractUint32(r)
	if err != nil {
		return err
	}

	if msgCRC != expect {
		return fmt.Errorf("Checksum Mismatch, MessageCRC of 0x%X does not equal expected CRC of 0x%X", msgCRC, expect)

	}
	return nil
}

// Close allows the user to close all the connection and all processes before
// waiting for the End message.``
func (myEvent *SelectResults) Close() {
	if myEvent.err == nil {
		myEvent.err = io.EOF
	}
	// closes the channel which keeps track of State
	// read explanation on Line 264
	close(myEvent.lock)
	// blocks on awaiting for the myEvent.Response channel to be closed so that it
	// is guaranteed the other Go routine has stopped and returned.
	for {
		_, ok := <-myEvent.Response
		// myEvent.Response channel has closed meaning that the other routine has
		// stopped and this thread can now proceed to terminating as well.
		if !ok {
			return
		}
	}
}

func (myEvent *SelectResults) Read(buffer []byte) (int, error) {
	// counter is used to make sure that if the End message is processed before
	// this function, that there will be a single iteration through the for loop
	counter := 0
	if myEvent.err == io.EOF {
		counter++
	}
	for myEvent.err == nil || counter == 1 {
		if myEvent.err == io.EOF {
			counter++
		}
		// Read the message from the buffer of records
		myPayload, err := myEvent.writer.Read(buffer)
		if err != nil {
			// if there is nothing to read then block until a message is ready.
			if err == io.EOF && myPayload == 0 {
				continue
			} else if err == io.EOF {
				// if enough has been read, but there was more than 0 bytes, should
				// return
				return myPayload, nil
			}
			// return out of an error
			return myPayload, err
		}
		// return since by this time, the message has been succesfully procesesed
		return myPayload, err
	}
	// in case there was already a present error, just return it
	return 0, myEvent.err
}

// Stats is a method for returning a Stats structure
func (myEvent *SelectResults) Stats() *StatEvent {
	return myEvent.stat
}

// Progress is a method for returning a Stats structure
func (myEvent *SelectResults) Progress() *ProgressEvent {
	return myEvent.progress
}

// err returns an error if one was found during the streaming.
func (myEvent *SelectResults) notEOF() error {
	if myEvent.err != io.EOF {
		return myEvent.err
	}
	return nil
}
