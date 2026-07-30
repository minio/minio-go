/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2021 MinIO, Inc.
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
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestGetObjectReturnSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
		w.Header().Set("Content-Length", "5")

		// Write less bytes than the content length.
		w.Write([]byte("12345"))
	}))
	defer srv.Close()

	// New - instantiate minio client with options
	clnt, err := New(srv.Listener.Addr().String(), &Options{
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	obj, err := clnt.GetObject(context.Background(), "bucketName", "objectName", GetObjectOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// We expect an error when reading back.
	buf, err := io.ReadAll(obj)
	if err != nil {
		t.Fatalf("Expected 'nil', got %v", err)
	}

	if len(buf) != 5 {
		t.Fatalf("Expected read bytes '5', got %v", len(buf))
	}
}

func TestGetObjectReturnErrorIfServerTruncatesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
		w.Header().Set("Content-Length", "100")

		// Write less bytes than the content length.
		w.Write([]byte("12345"))
	}))
	defer srv.Close()

	// New - instantiate minio client with options
	clnt, err := New(srv.Listener.Addr().String(), &Options{
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	obj, err := clnt.GetObject(context.Background(), "bucketName", "objectName", GetObjectOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// We expect an error when reading back.
	if _, err = io.ReadAll(obj); err != io.ErrUnexpectedEOF {
		t.Fatalf("Expected %v, got %v", io.ErrUnexpectedEOF, err)
	}
}

func TestGetObjectReturnErrorIfServerTruncatesResponseDouble(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
		w.Header().Set("Content-Length", "1024")

		// Write less bytes than the content length.
		io.Copy(w, io.LimitReader(rand.Reader, 1023))
	}))
	defer srv.Close()

	// New - instantiate minio client with options
	clnt, err := New(srv.Listener.Addr().String(), &Options{
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	obj, err := clnt.GetObject(context.Background(), "bucketName", "objectName", GetObjectOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// We expect an error when reading back.
	if _, err = io.ReadAll(obj); err != io.ErrUnexpectedEOF {
		t.Fatalf("Expected %v, got %v", io.ErrUnexpectedEOF, err)
	}
}

func TestGetObjectReturnErrorIfServerSendsMore(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
		w.Header().Set("Content-Length", "1")

		// Write less bytes than the content length.
		w.Write([]byte("12345"))
	}))
	defer srv.Close()

	// New - instantiate minio client with options
	clnt, err := New(srv.Listener.Addr().String(), &Options{
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	obj, err := clnt.GetObject(context.Background(), "bucketName", "objectName", GetObjectOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// We expect an error when reading back.
	if _, err = io.ReadAll(obj); err != io.ErrUnexpectedEOF {
		t.Fatalf("Expected %v, got %v", io.ErrUnexpectedEOF, err)
	}
}

// eofRangeTestServer mimics a server answering HEAD with headSize (which may
// deliberately overstate the payload to simulate a stale cached size), plain
// GET with the full payload, and range GET with 206 for satisfiable ranges or
// 416 InvalidRange when the range starts at or beyond the payload size, as
// captured in issue #2166. It counts GET requests.
func eofRangeTestServer(t *testing.T, payload []byte, headSize int, getCount *int32) *httptest.Server {
	t.Helper()
	size := len(payload)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"0123456789abcdef0123456789abcdef"`)
		w.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
		w.Header().Set("Accept-Ranges", "bytes")

		if r.Method == http.MethodHead {
			w.Header().Set("Content-Length", strconv.Itoa(headSize))
			return
		}
		atomic.AddInt32(getCount, 1)
		rng := r.Header.Get("Range")
		if rng == "" {
			w.Header().Set("Content-Length", strconv.Itoa(size))
			w.Write(payload)
			return
		}
		var start, end int
		spec := strings.TrimPrefix(rng, "bytes=")
		if i := strings.Index(spec, "-"); i >= 0 {
			start, _ = strconv.Atoi(spec[:i])
			end = size - 1
			if i+1 < len(spec) {
				end, _ = strconv.Atoi(spec[i+1:])
			}
		}
		if start >= size {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?><Error><Code>InvalidRange</Code><Message>The requested range 'bytes=%d--1' is not satisfiable</Message><Key>objectName</Key><BucketName>bucketName</BucketName><ActualObjectSize>%d</ActualObjectSize><RangeRequested>%s</RangeRequested></Error>`, start, size, spec)
			return
		}
		if end >= size {
			end = size - 1
		}
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, size))
		w.Header().Set("Content-Length", strconv.Itoa(end-start+1))
		w.WriteHeader(http.StatusPartialContent)
		w.Write(payload[start : end+1])
	}))
}

func TestGetObjectReadAtEOFReturnsEOF(t *testing.T) {
	payload := []byte("0123456789")
	var gets int32
	srv := eofRangeTestServer(t, payload, len(payload), &gets)
	defer srv.Close()

	clnt, err := New(srv.Listener.Addr().String(), &Options{
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Seek to the object size, then Read: io.EOF without any GET request.
	obj, err := clnt.GetObject(context.Background(), "bucketName", "objectName", GetObjectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	n64, err := obj.Seek(int64(len(payload)), io.SeekStart)
	if err != nil || n64 != int64(len(payload)) {
		t.Fatalf("Seek(size, SeekStart): expected (%d, nil), got (%d, %v)", len(payload), n64, err)
	}
	buf := make([]byte, 4)
	if _, err = obj.Read(buf); err != io.EOF {
		t.Fatalf("Read at object size: expected io.EOF, got %v", err)
	}
	if got := atomic.LoadInt32(&gets); got != 0 {
		t.Fatalf("Read at object size issued %d GET request(s), expected 0", got)
	}

	// The object must stay usable: seek back and read real data.
	if _, err = obj.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("Seek(0, SeekStart) after EOF read: expected success, got %v", err)
	}
	n, err := obj.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Read after re-seek: expected data, got %v", err)
	}
	if n != len(buf) || !bytes.Equal(buf, payload[:n]) {
		t.Fatalf("Read after re-seek: expected %q, got %q", payload[:len(buf)], buf[:n])
	}
	obj.Close()

	// ReadAt in chunks up to and past the end: the object size is never
	// known client-side on the ReadAt path, so the at-EOF request is issued
	// and the server's 416 InvalidRange must surface as io.EOF.
	obj, err = clnt.GetObject(context.Background(), "bucketName", "objectName", GetObjectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer obj.Close()
	n, err = obj.ReadAt(buf, 0)
	if err != nil && err != io.EOF {
		t.Fatalf("ReadAt(0): expected data, got %v", err)
	}
	if n != len(buf) || !bytes.Equal(buf, payload[:n]) {
		t.Fatalf("ReadAt(0): expected %q, got %q", payload[:len(buf)], buf[:n])
	}
	if _, err = obj.ReadAt(buf, int64(len(payload))); err != io.EOF {
		t.Fatalf("ReadAt(size): expected io.EOF, got %v", err)
	}

	// The at-EOF ReadAt must not end the stream: an in-range ReadAt on the
	// same object still returns data.
	n, err = obj.ReadAt(buf, 0)
	if err != nil && err != io.EOF {
		t.Fatalf("ReadAt(0) after at-EOF ReadAt: expected data, got %v", err)
	}
	if n != len(buf) || !bytes.Equal(buf, payload[:n]) {
		t.Fatalf("ReadAt(0) after at-EOF ReadAt: expected %q, got %q", payload[:len(buf)], buf[:n])
	}

	// A metadata request between a failed fetch and the next read clears the
	// re-establish flag; the read must still not use the dead reader.
	if _, err = obj.ReadAt(buf, int64(len(payload))); err != io.EOF {
		t.Fatalf("ReadAt(size) second time: expected io.EOF, got %v", err)
	}
	if _, err = obj.Stat(); err != nil {
		t.Fatalf("Stat after at-EOF ReadAt: expected success, got %v", err)
	}
	n, err = obj.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Read after Stat following at-EOF ReadAt: expected data, got %v", err)
	}
	if n != len(buf) || !bytes.Equal(buf, payload[:n]) {
		t.Fatalf("Read after Stat following at-EOF ReadAt: expected %q, got %q", payload[:len(buf)], buf[:n])
	}
}

// TestGetObjectReadUserRangeInvalidRangeSurfaces pins that a caller-supplied
// unsatisfiable range - the only range a read at offset zero ever sends - is
// caller misuse, not EOF, and keeps surfacing the server's InvalidRange.
func TestGetObjectReadUserRangeInvalidRangeSurfaces(t *testing.T) {
	payload := []byte("0123456789")
	var gets int32
	srv := eofRangeTestServer(t, payload, len(payload), &gets)
	defer srv.Close()

	clnt, err := New(srv.Listener.Addr().String(), &Options{
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	opts := GetObjectOptions{}
	if err := opts.SetRange(int64(len(payload)+1), int64(len(payload)+10)); err != nil {
		t.Fatal(err)
	}
	obj, err := clnt.GetObject(context.Background(), "bucketName", "objectName", opts)
	if err != nil {
		t.Fatal(err)
	}
	defer obj.Close()
	_, err = obj.Read(make([]byte, 4))
	if err == io.EOF {
		t.Fatal("Read with caller-set unsatisfiable range: expected InvalidRange to surface, got io.EOF")
	}
	if code := ToErrorResponse(err).Code; code != InvalidRange {
		t.Fatalf("Read with caller-set unsatisfiable range: expected code %q, got %v", InvalidRange, err)
	}
}

// TestGetObjectReadStaleSizeReturnsEOF covers the read path where the
// advertised size is stale-high (object shrunk after Stat): the local at-EOF
// guard cannot fire, the offset-generated range draws a 416 from the server,
// and the error must translate to io.EOF.
func TestGetObjectReadStaleSizeReturnsEOF(t *testing.T) {
	payload := []byte("0123456789")
	var gets int32
	srv := eofRangeTestServer(t, payload, 2*len(payload), &gets)
	defer srv.Close()

	clnt, err := New(srv.Listener.Addr().String(), &Options{
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	obj, err := clnt.GetObject(context.Background(), "bucketName", "objectName", GetObjectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer obj.Close()
	// Seek runs Stat, which reports the stale doubled size, so seeking past
	// the real payload end succeeds and the local at-EOF guard cannot fire.
	if _, err = obj.Seek(int64(len(payload)+5), io.SeekStart); err != nil {
		t.Fatalf("Seek past real size with stale Stat size: expected success, got %v", err)
	}
	if _, err = obj.Read(make([]byte, 4)); err != io.EOF {
		t.Fatalf("Read past real object end: expected io.EOF, got %v", err)
	}
	if got := atomic.LoadInt32(&gets); got != 1 {
		t.Fatalf("stale-size read issued %d GET request(s), expected exactly 1", got)
	}

	// The translated EOF must be recoverable: seek back and read real data.
	if _, err = obj.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("Seek(0, SeekStart) after translated EOF: expected success, got %v", err)
	}
	buf := make([]byte, 4)
	n, err := obj.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Read after re-seek: expected data, got %v", err)
	}
	if n != len(buf) || !bytes.Equal(buf, payload[:n]) {
		t.Fatalf("Read after re-seek: expected %q, got %q", payload[:len(buf)], buf[:n])
	}
}

// TestObjectSeekAtObjectSizeAllowsSubsequentReadEOF verifies that seeking to
// exactly the object size succeeds for every whence, and that io.EOF is
// reported by the subsequent Read rather than by Seek. See
// https://github.com/minio/minio-go/issues/2155 and #2166.
func TestObjectSeekAtObjectSizeAllowsSubsequentReadEOF(t *testing.T) {
	o := &Object{
		mutex:         &sync.Mutex{},
		objectInfo:    ObjectInfo{Size: 10},
		objectInfoSet: true,
		isStarted:     true,
		totalSize:     10,
	}

	n, err := o.Seek(10, io.SeekStart)
	if err != nil {
		t.Fatalf("expected seeking to object size to succeed, got %v", err)
	}
	if n != 10 {
		t.Fatalf("expected offset 10, got %d", n)
	}
	if _, err = o.Read(make([]byte, 1)); err != io.EOF {
		t.Fatalf("expected read at object size to return io.EOF, got %v", err)
	}

	o.currOffset = 9
	n, err = o.Seek(1, io.SeekCurrent)
	if err != nil {
		t.Fatalf("expected seeking current to object size to succeed, got %v", err)
	}
	if n != 10 {
		t.Fatalf("expected offset 10, got %d", n)
	}
	if _, err = o.Read(make([]byte, 1)); err != io.EOF {
		t.Fatalf("expected read at object size to return io.EOF, got %v", err)
	}

	n, err = o.Seek(0, io.SeekEnd)
	if err != nil {
		t.Fatalf("expected seeking to object end to succeed, got %v", err)
	}
	if n != 10 {
		t.Fatalf("expected offset 10, got %d", n)
	}
	if _, err = o.Read(make([]byte, 1)); err != io.EOF {
		t.Fatalf("expected read at object end to return io.EOF, got %v", err)
	}
}

// TestObjectSeekPastObjectSizeReturnsEOF verifies that seeking past the object
// size is rejected by Seek with io.EOF for every whence, leaving the offset
// unchanged, and that the object stays usable afterwards.
func TestObjectSeekPastObjectSizeReturnsEOF(t *testing.T) {
	o := &Object{
		mutex:         &sync.Mutex{},
		objectInfo:    ObjectInfo{Size: 10},
		objectInfoSet: true,
		isStarted:     true,
		totalSize:     10,
	}

	n, err := o.Seek(11, io.SeekStart)
	if err != io.EOF {
		t.Fatalf("expected seeking past object size to return io.EOF, got %v", err)
	}
	if n != 0 {
		t.Fatalf("expected returned offset 0, got %d", n)
	}
	if o.currOffset != 0 {
		t.Fatalf("expected offset to stay 0 after failed seek, got %d", o.currOffset)
	}

	o.currOffset = 9
	n, err = o.Seek(2, io.SeekCurrent)
	if err != io.EOF {
		t.Fatalf("expected seeking current past object size to return io.EOF, got %v", err)
	}
	if n != 0 {
		t.Fatalf("expected returned offset 0, got %d", n)
	}
	if o.currOffset != 9 {
		t.Fatalf("expected offset to stay 9 after failed seek, got %d", o.currOffset)
	}

	n, err = o.Seek(1, io.SeekEnd)
	if err != io.EOF {
		t.Fatalf("expected seeking past object end to return io.EOF, got %v", err)
	}
	if n != 0 {
		t.Fatalf("expected returned offset 0, got %d", n)
	}
	if o.currOffset != 9 {
		t.Fatalf("expected offset to stay 9 after failed seek, got %d", o.currOffset)
	}

	n, err = o.Seek(10, io.SeekStart)
	if err != nil {
		t.Fatalf("expected seeking to object size to succeed after failed seek, got %v", err)
	}
	if n != 10 {
		t.Fatalf("expected offset 10, got %d", n)
	}
	if _, err = o.Read(make([]byte, 1)); err != io.EOF {
		t.Fatalf("expected read at object size to return io.EOF, got %v", err)
	}
}

// TestObjectSeekCurrentNegativeOffset verifies that a negative offset with
// io.SeekCurrent is honored when the resulting absolute position is within the
// object, and rejected only when it would move before the start of the object.
// Regression test for https://github.com/minio/minio-go/issues/2155.
func TestObjectSeekCurrentNegativeOffset(t *testing.T) {
	o := &Object{
		mutex:         &sync.Mutex{},
		objectInfo:    ObjectInfo{Size: 10},
		objectInfoSet: true,
		isStarted:     true,
		currOffset:    5,
	}

	n, err := o.Seek(-2, io.SeekCurrent)
	if err != nil {
		t.Fatalf("expected SeekCurrent with negative offset to succeed, got %v", err)
	}
	if n != 3 {
		t.Fatalf("expected offset 3, got %d", n)
	}

	_, err = o.Seek(-10, io.SeekCurrent)
	if err == nil {
		t.Fatal("expected error when SeekCurrent would move before start of object")
	}

	_, err = o.Seek(-1, io.SeekStart)
	if err == nil {
		t.Fatal("expected error when SeekStart is given a negative offset")
	}
}

// TestObjectSeekEndUnknownSize verifies that io.SeekEnd is rejected when the
// object size is unknown.
func TestObjectSeekEndUnknownSize(t *testing.T) {
	o := &Object{
		mutex:         &sync.Mutex{},
		objectInfo:    ObjectInfo{Size: -1},
		objectInfoSet: true,
		isStarted:     true,
		totalSize:     -1,
	}

	if _, err := o.Seek(0, io.SeekEnd); err == nil {
		t.Fatal("expected error when seeking from end with unknown object size")
	}
}

// TestObjectSeekInvalidWhence verifies that an unsupported whence value is
// rejected.
func TestObjectSeekInvalidWhence(t *testing.T) {
	o := &Object{
		mutex:         &sync.Mutex{},
		objectInfo:    ObjectInfo{Size: 10},
		objectInfoSet: true,
		isStarted:     true,
	}

	if _, err := o.Seek(0, 3); err == nil {
		t.Fatal("expected error for invalid whence")
	}
}

// TestObjectSeekWithSetRange documents the current interplay between
// GetObjectOptions.SetRange and Object.Seek: a Seek issued before any Read
// stats the object without the range header, so Seek offsets and the
// end-of-object boundary refer to the full object, and a Read after a Seek
// requests a fresh range starting at the seek offset, discarding the range
// supplied by the caller. See https://github.com/minio/minio-go/pull/2267 for
// a proposed change to the related Stat behavior.
func TestObjectSeekWithSetRange(t *testing.T) {
	content := []byte("0123456789")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
		if r.Method == http.MethodHead {
			w.Header().Set("Content-Length", strconv.Itoa(len(content)))
			return
		}
		body := content
		if rng := r.Header.Get("Range"); rng != "" {
			var start, end int
			if n, _ := fmt.Sscanf(rng, "bytes=%d-%d", &start, &end); n == 2 {
				body = content[start:min(end+1, len(content))]
			} else if n, _ := fmt.Sscanf(rng, "bytes=%d-", &start); n == 1 {
				body = content[start:]
			}
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, start+len(body)-1, len(content)))
			w.Header().Set("Content-Length", strconv.Itoa(len(body)))
			w.WriteHeader(http.StatusPartialContent)
		} else {
			w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		}
		w.Write(body)
	}))
	defer srv.Close()

	clnt, err := New(srv.Listener.Addr().String(), &Options{
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	opts := GetObjectOptions{}
	if err = opts.SetRange(2, 8); err != nil {
		t.Fatal(err)
	}
	obj, err := clnt.GetObject(context.Background(), "bucketName", "objectName", opts)
	if err != nil {
		t.Fatal(err)
	}

	// Seek relative to the end resolves against the full object size, not the
	// requested range.
	n, err := obj.Seek(0, io.SeekEnd)
	if err != nil {
		t.Fatalf("expected seeking to object end to succeed, got %v", err)
	}
	if n != int64(len(content)) {
		t.Fatalf("expected offset %d, got %d", len(content), n)
	}
	if _, err = obj.Read(make([]byte, 1)); err != io.EOF {
		t.Fatalf("expected read at object end to return io.EOF, got %v", err)
	}

	// The past-end boundary also refers to the full object size.
	if _, err = obj.Seek(int64(len(content))+1, io.SeekStart); err != io.EOF {
		t.Fatalf("expected seeking past object end to return io.EOF, got %v", err)
	}

	// A Read after a Seek fetches from the seek offset, not from the range
	// supplied by the caller.
	if _, err = obj.Seek(4, io.SeekStart); err != nil {
		t.Fatalf("expected seeking within the object to succeed, got %v", err)
	}
	buf := make([]byte, 3)
	rn, err := obj.Read(buf)
	if err != nil {
		t.Fatalf("expected read after seek to succeed, got %v", err)
	}
	if rn != 3 || string(buf) != "456" {
		t.Fatalf("expected to read %q, got %q (%d bytes)", "456", string(buf[:rn]), rn)
	}
}
