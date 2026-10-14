/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2026 MinIO, Inc.
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
	"encoding/binary"
	"hash/crc32"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

// encodeSelectMessage builds one event-stream message as sent by SelectObjectContent.
func encodeSelectMessage(headers map[string]string, payload []byte) []byte {
	var hdr bytes.Buffer
	for k, v := range headers {
		hdr.WriteByte(byte(len(k) + 1))
		hdr.WriteString(":" + k)
		hdr.WriteByte(7) // string value type
		binary.Write(&hdr, binary.BigEndian, uint16(len(v)))
		hdr.WriteString(v)
	}

	var msg bytes.Buffer
	binary.Write(&msg, binary.BigEndian, uint32(16+hdr.Len()+len(payload)))
	binary.Write(&msg, binary.BigEndian, uint32(hdr.Len()))
	binary.Write(&msg, binary.BigEndian, crc32.ChecksumIEEE(msg.Bytes()))
	msg.Write(hdr.Bytes())
	msg.Write(payload)
	binary.Write(&msg, binary.BigEndian, crc32.ChecksumIEEE(msg.Bytes()))
	return msg.Bytes()
}

// exclusiveBody fails the test if Read or Close runs while another Read or
// Close is in progress. Decompressing bodies (for example the zstd reader in
// gzhttp) are not safe for concurrent use and panic when this happens.
type exclusiveBody struct {
	r          io.Reader
	inUse      atomic.Int32
	concurrent atomic.Bool
}

func (b *exclusiveBody) enter() {
	if b.inUse.Add(1) > 1 {
		b.concurrent.Store(true)
	}
	time.Sleep(time.Millisecond) // widen the window for overlapping calls
}

func (b *exclusiveBody) exit() { b.inUse.Add(-1) }

func (b *exclusiveBody) Read(p []byte) (int, error) {
	b.enter()
	defer b.exit()
	return b.r.Read(p)
}

func (b *exclusiveBody) Close() error {
	b.enter()
	defer b.exit()
	return nil
}

func TestSelectResultsCloseDoesNotReadConcurrently(t *testing.T) {
	var stream bytes.Buffer
	for i := 0; i < 20; i++ {
		stream.Write(encodeSelectMessage(map[string]string{
			"message-type": "event",
			"event-type":   "Records",
			"content-type": "application/octet-stream",
		}, []byte("row\n")))
	}
	stream.Write(encodeSelectMessage(map[string]string{
		"message-type": "event",
		"event-type":   "End",
	}, nil))
	streamBytes := stream.Bytes()

	for i := 0; i < 20; i++ {
		body := &exclusiveBody{r: bytes.NewReader(streamBytes)}
		res, err := NewSelectResults(&http.Response{StatusCode: http.StatusOK, Body: body}, "bucket")
		if err != nil {
			t.Fatal(err)
		}

		// Read part of the results, then close early, as `mc sql` does
		// once it has printed what it needs.
		buf := make([]byte, 8)
		if _, err := io.ReadFull(res, buf); err != nil {
			t.Fatal(err)
		}
		if err := res.Close(); err != nil {
			t.Fatal(err)
		}

		if body.concurrent.Load() {
			t.Fatalf("run %d: response body was read or closed concurrently", i)
		}
	}
}
