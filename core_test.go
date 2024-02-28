/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2017-2020 MinIO, Inc.
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
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/encrypt"
)

const (
	serverEndpoint = "SERVER_ENDPOINT"
	accessKey      = "ACCESS_KEY"
	secretKey      = "SECRET_KEY"
	enableSecurity = "ENABLE_HTTPS"
)

// Tests for Core GetObject() function.
func TestGetObjectCore(t *testing.T) {
	if os.Getenv(serverEndpoint) == "" {
		t.Skip("SERVER_ENDPOINT not set")
	}
	if testing.Short() {
		t.Skip("skipping functional tests for the short runs")
	}

	// Instantiate new minio core client object.
	c, err := NewCore(
		os.Getenv(serverEndpoint),
		&Options{
			Creds:  credentials.NewStaticV4(os.Getenv(accessKey), os.Getenv(secretKey), ""),
			Secure: mustParseBool(os.Getenv(enableSecurity)),
		})
	if err != nil {
		t.Fatal("Error:", err)
	}

	// Enable tracing, write to stderr.
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("MinIO-go-FunctionalTest", "0.1.0")

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket.
	err = c.MakeBucket(context.Background(), bucketName, MakeBucketOptions{Region: "us-east-1"})
	if err != nil {
		t.Fatal("Error:", err, bucketName)
	}

	// Generate data more than 32K
	buf := bytes.Repeat([]byte("3"), rand.Intn(1<<20)+32*1024)

	// Save the data
	objectName := randString(60, rand.NewSource(time.Now().UnixNano()), "")
	_, err = c.Client.PutObject(context.Background(), bucketName, objectName, bytes.NewReader(buf), int64(len(buf)), PutObjectOptions{
		ContentType: "binary/octet-stream",
	})
	if err != nil {
		t.Fatal("Error:", err, bucketName, objectName)
	}

	st, err := c.Client.StatObject(context.Background(), bucketName, objectName, StatObjectOptions{})
	if err != nil {
		t.Fatal("Stat error:", err, bucketName, objectName)
	}
	if st.Size != int64(len(buf)) {
		t.Fatalf("Error: number of bytes does not match, want %v, got %v\n", len(buf), st.Size)
	}

	offset := int64(2048)

	// read directly
	buf1 := make([]byte, 512)
	buf2 := make([]byte, 512)
	buf3 := make([]byte, st.Size)
	buf4 := make([]byte, 1)

	opts := GetObjectOptions{}
	opts.SetRange(offset, offset+int64(len(buf1))-1)
	reader, objectInfo, _, err := c.GetObject(context.Background(), bucketName, objectName, opts)
	if err != nil {
		t.Fatal(err)
	}
	m, err := readFull(reader, buf1)
	reader.Close()
	if err != nil {
		t.Fatal(err)
	}

	if objectInfo.Size != int64(m) {
		t.Fatalf("Error: GetObject read shorter bytes before reaching EOF, want %v, got %v\n", objectInfo.Size, m)
	}
	if !bytes.Equal(buf1, buf[offset:offset+512]) {
		t.Fatal("Error: Incorrect read between two GetObject from same offset.")
	}
	offset += 512

	opts.SetRange(offset, offset+int64(len(buf2))-1)
	reader, objectInfo, _, err = c.GetObject(context.Background(), bucketName, objectName, opts)
	if err != nil {
		t.Fatal(err)
	}

	m, err = readFull(reader, buf2)
	reader.Close()
	if err != nil {
		t.Fatal(err)
	}

	if objectInfo.Size != int64(m) {
		t.Fatalf("Error: GetObject read shorter bytes before reaching EOF, want %v, got %v\n", objectInfo.Size, m)
	}
	if !bytes.Equal(buf2, buf[offset:offset+512]) {
		t.Fatal("Error: Incorrect read between two GetObject from same offset.")
	}

	opts.SetRange(0, int64(len(buf3)))
	reader, objectInfo, _, err = c.GetObject(context.Background(), bucketName, objectName, opts)
	if err != nil {
		t.Fatal(err)
	}

	m, err = readFull(reader, buf3)
	if err != nil {
		reader.Close()
		t.Fatal(err)
	}
	reader.Close()

	if objectInfo.Size != int64(m) {
		t.Fatalf("Error: GetObject read shorter bytes before reaching EOF, want %v, got %v\n", objectInfo.Size, m)
	}
	if !bytes.Equal(buf3, buf) {
		t.Fatal("Error: Incorrect data read in GetObject, than what was previously upoaded.")
	}

	opts = GetObjectOptions{}
	opts.SetMatchETag("etag")
	_, _, _, err = c.GetObject(context.Background(), bucketName, objectName, opts)
	if err == nil {
		t.Fatal("Unexpected GetObject should fail with mismatching etags")
	}
	if errResp := ToErrorResponse(err); errResp.Code != "PreconditionFailed" {
		t.Fatalf("Expected \"PreconditionFailed\" as code, got %s instead", errResp.Code)
	}

	opts = GetObjectOptions{}
	opts.SetMatchETagExcept("etag")
	reader, objectInfo, _, err = c.GetObject(context.Background(), bucketName, objectName, opts)
	if err != nil {
		t.Fatal(err)
	}

	m, err = readFull(reader, buf3)
	reader.Close()
	if err != nil {
		t.Fatal(err)
	}

	if objectInfo.Size != int64(m) {
		t.Fatalf("Error: GetObject read shorter bytes before reaching EOF, want %v, got %v\n", objectInfo.Size, m)
	}
	if !bytes.Equal(buf3, buf) {
		t.Fatal("Error: Incorrect data read in GetObject, than what was previously upoaded.")
	}

	opts = GetObjectOptions{}
	opts.SetRange(0, 0)
	reader, objectInfo, _, err = c.GetObject(context.Background(), bucketName, objectName, opts)
	if err != nil {
		t.Fatal(err)
	}

	m, err = readFull(reader, buf4)
	reader.Close()
	if err != nil {
		t.Fatal(err)
	}

	if objectInfo.Size != int64(m) {
		t.Fatalf("Error: GetObject read shorter bytes before reaching EOF, want %v, got %v\n", objectInfo.Size, m)
	}

	opts = GetObjectOptions{}
	opts.SetRange(offset, offset+int64(len(buf2))-1)
	contentLength := len(buf2)
	var header http.Header
	_, _, header, err = c.GetObject(context.Background(), bucketName, objectName, opts)
	if err != nil {
		t.Fatal(err)
	}

	contentLengthValue, err := strconv.Atoi(header.Get("Content-Length"))
	if err != nil {
		t.Fatal("Error: ", err)
	}
	if contentLength != contentLengthValue {
		t.Fatalf("Error: Content Length in response header %v, not equal to set content length %v\n", contentLengthValue, contentLength)
	}

	err = c.RemoveObject(context.Background(), bucketName, objectName, RemoveObjectOptions{})
	if err != nil {
		t.Fatal("Error: ", err)
	}
	err = c.RemoveBucket(context.Background(), bucketName)
	if err != nil {
		t.Fatal("Error:", err)
	}
}

// Tests GetObject to return Content-Encoding properly set
// and overrides any auto decoding.
func TestGetObjectContentEncoding(t *testing.T) {
	if os.Getenv(serverEndpoint) == "" {
		t.Skip("SERVER_ENDPOINT not set")
	}
	if testing.Short() {
		t.Skip("skipping functional tests for the short runs")
	}

	// Instantiate new minio core client object.
	c, err := NewCore(
		os.Getenv(serverEndpoint),
		&Options{
			Creds:  credentials.NewStaticV4(os.Getenv(accessKey), os.Getenv(secretKey), ""),
			Secure: mustParseBool(os.Getenv(enableSecurity)),
		})
	if err != nil {
		t.Fatal("Error:", err)
	}

	// Enable tracing, write to stderr.
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("MinIO-go-FunctionalTest", "0.1.0")

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket.
	err = c.MakeBucket(context.Background(), bucketName, MakeBucketOptions{Region: "us-east-1"})
	if err != nil {
		t.Fatal("Error:", err, bucketName)
	}

	// Generate data more than 32K
	buf := bytes.Repeat([]byte("3"), rand.Intn(1<<20)+32*1024)

	// Save the data
	objectName := randString(60, rand.NewSource(time.Now().UnixNano()), "")
	_, err = c.Client.PutObject(context.Background(), bucketName, objectName, bytes.NewReader(buf), int64(len(buf)), PutObjectOptions{
		ContentEncoding: "gzip",
	})
	if err != nil {
		t.Fatal("Error:", err, bucketName, objectName)
	}

	rwc, objInfo, _, err := c.GetObject(context.Background(), bucketName, objectName, GetObjectOptions{})
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	rwc.Close()
	if objInfo.Size != int64(len(buf)) {
		t.Fatalf("Unexpected size of the object %v, expected %v", objInfo.Size, len(buf))
	}
	value, ok := objInfo.Metadata["Content-Encoding"]
	if !ok {
		t.Fatalf("Expected Content-Encoding metadata to be set.")
	}
	if value[0] != "gzip" {
		t.Fatalf("Unexpected content-encoding found, want gzip, got %v", value)
	}

	err = c.RemoveObject(context.Background(), bucketName, objectName, RemoveObjectOptions{})
	if err != nil {
		t.Fatal("Error: ", err)
	}
	err = c.RemoveBucket(context.Background(), bucketName)
	if err != nil {
		t.Fatal("Error:", err)
	}
}

// Tests get bucket policy core API.
func TestGetBucketPolicy(t *testing.T) {
	if os.Getenv(serverEndpoint) == "" {
		t.Skip("SERVER_ENDPOINT not set")
	}
	if testing.Short() {
		t.Skip("skipping functional tests for short runs")
	}

	// Instantiate new minio client object.
	c, err := NewCore(
		os.Getenv(serverEndpoint),
		&Options{
			Creds:  credentials.NewStaticV4(os.Getenv(accessKey), os.Getenv(secretKey), ""),
			Secure: mustParseBool(os.Getenv(enableSecurity)),
		})
	if err != nil {
		t.Fatal("Error:", err)
	}

	// Enable to debug
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("MinIO-go-FunctionalTest", "0.1.0")

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket.
	err = c.MakeBucket(context.Background(), bucketName, MakeBucketOptions{Region: "us-east-1"})
	if err != nil {
		t.Fatal("Error:", err, bucketName)
	}

	// Verify if bucket exits and you have access.
	var exists bool
	exists, err = c.BucketExists(context.Background(), bucketName)
	if err != nil {
		t.Fatal("Error:", err, bucketName)
	}
	if !exists {
		t.Fatal("Error: could not find ", bucketName)
	}

	// Asserting the default bucket policy.
	bucketPolicy, err := c.GetBucketPolicy(context.Background(), bucketName)
	if err != nil {
		errResp := ToErrorResponse(err)
		if errResp.Code != "NoSuchBucketPolicy" {
			t.Error("Error:", err, bucketName)
		}
	}
	if bucketPolicy != "" {
		t.Errorf("Bucket policy expected %#v, got %#v", "", bucketPolicy)
	}

	err = c.RemoveBucket(context.Background(), bucketName)
	if err != nil {
		t.Fatal("Error:", err)
	}
}

// Tests Core CopyObject API implementation.
func TestCoreCopyObject(t *testing.T) {
	if os.Getenv(serverEndpoint) == "" {
		t.Skip("SERVER_ENDPOINT not set")
	}
	if testing.Short() {
		t.Skip("skipping functional tests for short runs")
	}

	// Instantiate new minio client object.
	c, err := NewCore(
		os.Getenv(serverEndpoint),
		&Options{
			Creds:  credentials.NewStaticV4(os.Getenv(accessKey), os.Getenv(secretKey), ""),
			Secure: mustParseBool(os.Getenv(enableSecurity)),
		})
	if err != nil {
		t.Fatal("Error:", err)
	}

	// Enable tracing, write to stderr.
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("MinIO-go-FunctionalTest", "0.1.0")

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket.
	err = c.MakeBucket(context.Background(), bucketName, MakeBucketOptions{Region: "us-east-1"})
	if err != nil {
		t.Fatal("Error:", err, bucketName)
	}

	buf := bytes.Repeat([]byte("a"), 32*1024)

	// Save the data
	objectName := randString(60, rand.NewSource(time.Now().UnixNano()), "")

	putopts := PutObjectOptions{
		UserMetadata: map[string]string{
			"Content-Type": "binary/octet-stream",
		},
	}
	uploadInfo, err := c.PutObject(context.Background(), bucketName, objectName, bytes.NewReader(buf), int64(len(buf)), "", "", putopts)
	if err != nil {
		t.Fatal("Error:", err, bucketName, objectName)
	}

	st, err := c.StatObject(context.Background(), bucketName, objectName, StatObjectOptions{})
	if err != nil {
		t.Fatal("Error:", err, bucketName, objectName)
	}

	if st.Size != int64(len(buf)) {
		t.Fatalf("Error: number of bytes does not match, want %v, got %v\n", len(buf), st.Size)
	}

	destBucketName := bucketName
	destObjectName := objectName + "-dest"

	cuploadInfo, err := c.CopyObject(context.Background(), bucketName, objectName, destBucketName, destObjectName, map[string]string{
		"X-Amz-Metadata-Directive": "REPLACE",
		"Content-Type":             "application/javascript",
	}, CopySrcOptions{}, PutObjectOptions{})
	if err != nil {
		t.Fatal("Error:", err, bucketName, objectName, destBucketName, destObjectName)
	}
	if cuploadInfo.ETag != uploadInfo.ETag {
		t.Fatalf("Error: expected etag to be same as source object %s, but found different etag %s", uploadInfo.ETag, cuploadInfo.ETag)
	}

	// Attempt to read from destBucketName and object name.
	r, err := c.Client.GetObject(context.Background(), destBucketName, destObjectName, GetObjectOptions{})
	if err != nil {
		t.Fatal("Error:", err, bucketName, objectName)
	}

	st, err = r.Stat()
	if err != nil {
		t.Fatal("Error:", err, bucketName, objectName)
	}

	if st.Size != int64(len(buf)) {
		t.Fatalf("Error: number of bytes in stat does not match, want %v, got %v\n",
			len(buf), st.Size)
	}

	if st.ContentType != "application/javascript" {
		t.Fatalf("Error: Content types don't match, expected: application/javascript, found: %+v\n", st.ContentType)
	}

	if st.ETag != uploadInfo.ETag {
		t.Fatalf("Error: expected etag to be same as source object %s, but found different etag :%s", uploadInfo.ETag, st.ETag)
	}

	if err := r.Close(); err != nil {
		t.Fatal("Error:", err)
	}

	if err := r.Close(); err == nil {
		t.Fatal("Error: object is already closed, should return error")
	}

	err = c.RemoveObject(context.Background(), bucketName, objectName, RemoveObjectOptions{})
	if err != nil {
		t.Fatal("Error: ", err)
	}

	err = c.RemoveObject(context.Background(), destBucketName, destObjectName, RemoveObjectOptions{})
	if err != nil {
		t.Fatal("Error: ", err)
	}

	err = c.RemoveBucket(context.Background(), bucketName)
	if err != nil {
		t.Fatal("Error:", err)
	}

	// Do not need to remove destBucketName its same as bucketName.
}

// Test Core CopyObjectPart implementation
func TestCoreCopyObjectPart(t *testing.T) {
	if os.Getenv(serverEndpoint) == "" {
		t.Skip("SERVER_ENDPOINT not set")
	}
	if testing.Short() {
		t.Skip("skipping functional tests for short runs")
	}

	// Instantiate new minio client object.
	c, err := NewCore(
		os.Getenv(serverEndpoint),
		&Options{
			Creds:  credentials.NewStaticV4(os.Getenv(accessKey), os.Getenv(secretKey), ""),
			Secure: mustParseBool(os.Getenv(enableSecurity)),
		})
	if err != nil {
		t.Fatal("Error:", err)
	}

	// Enable tracing, write to stderr.
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("MinIO-go-FunctionalTest", "0.1.0")

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket.
	err = c.MakeBucket(context.Background(), bucketName, MakeBucketOptions{Region: "us-east-1"})
	if err != nil {
		t.Fatal("Error:", err, bucketName)
	}

	// Make a buffer with 5MB of data
	buf := bytes.Repeat([]byte("abcde"), 1024*1024)
	metadata := map[string]string{
		"Content-Type": "binary/octet-stream",
	}
	putopts := PutObjectOptions{
		UserMetadata: metadata,
	}
	// Save the data
	objectName := randString(60, rand.NewSource(time.Now().UnixNano()), "")
	_, err = c.PutObject(context.Background(), bucketName, objectName, bytes.NewReader(buf), int64(len(buf)), "", "", putopts)
	if err != nil {
		t.Fatal("Error:", err, bucketName, objectName)
	}

	st, err := c.StatObject(context.Background(), bucketName, objectName, StatObjectOptions{})
	if err != nil {
		t.Fatal("Error:", err, bucketName, objectName)
	}

	if st.Size != int64(len(buf)) {
		t.Fatalf("Error: number of bytes does not match, want %v, got %v\n", len(buf), st.Size)
	}

	destBucketName := bucketName
	destObjectName := objectName + "-dest"

	uploadID, err := c.NewMultipartUpload(context.Background(), destBucketName, destObjectName, PutObjectOptions{})
	if err != nil {
		t.Fatal("Error:", err, bucketName, objectName)
	}

	// Content of the destination object will be two copies of
	// `objectName` concatenated, followed by first byte of
	// `objectName`.

	// First of three parts
	fstPart, err := c.CopyObjectPart(context.Background(), bucketName, objectName, destBucketName, destObjectName, uploadID, 1, 0, -1, nil)
	if err != nil {
		t.Fatal("Error:", err, destBucketName, destObjectName)
	}

	// Second of three parts
	sndPart, err := c.CopyObjectPart(context.Background(), bucketName, objectName, destBucketName, destObjectName, uploadID, 2, 0, -1, nil)
	if err != nil {
		t.Fatal("Error:", err, destBucketName, destObjectName)
	}

	// Last of three parts
	lstPart, err := c.CopyObjectPart(context.Background(), bucketName, objectName, destBucketName, destObjectName, uploadID, 3, 0, 1, nil)
	if err != nil {
		t.Fatal("Error:", err, destBucketName, destObjectName)
	}

	// Complete the multipart upload
	_, err = c.CompleteMultipartUpload(context.Background(), destBucketName, destObjectName, uploadID, []CompletePart{fstPart, sndPart, lstPart}, PutObjectOptions{})
	if err != nil {
		t.Fatal("Error:", err, destBucketName, destObjectName)
	}

	// Stat the object and check its length matches
	objInfo, err := c.StatObject(context.Background(), destBucketName, destObjectName, StatObjectOptions{})
	if err != nil {
		t.Fatal("Error:", err, destBucketName, destObjectName)
	}

	if objInfo.Size != (5*1024*1024)*2+1 {
		t.Fatal("Destination object has incorrect size!")
	}

	// Now we read the data back
	getOpts := GetObjectOptions{}
	getOpts.SetRange(0, 5*1024*1024-1)
	r, _, _, err := c.GetObject(context.Background(), destBucketName, destObjectName, getOpts)
	if err != nil {
		t.Fatal("Error:", err, destBucketName, destObjectName)
	}
	getBuf := make([]byte, 5*1024*1024)
	_, err = readFull(r, getBuf)
	if err != nil {
		t.Fatal("Error:", err, destBucketName, destObjectName)
	}
	if !bytes.Equal(getBuf, buf) {
		t.Fatal("Got unexpected data in first 5MB")
	}

	getOpts.SetRange(5*1024*1024, 0)
	r, _, _, err = c.GetObject(context.Background(), destBucketName, destObjectName, getOpts)
	if err != nil {
		t.Fatal("Error:", err, destBucketName, destObjectName)
	}
	getBuf = make([]byte, 5*1024*1024+1)
	_, err = readFull(r, getBuf)
	if err != nil {
		t.Fatal("Error:", err, destBucketName, destObjectName)
	}
	if !bytes.Equal(getBuf[:5*1024*1024], buf) {
		t.Fatal("Got unexpected data in second 5MB")
	}
	if getBuf[5*1024*1024] != buf[0] {
		t.Fatal("Got unexpected data in last byte of copied object!")
	}

	if err := c.RemoveObject(context.Background(), destBucketName, destObjectName, RemoveObjectOptions{}); err != nil {
		t.Fatal("Error: ", err)
	}

	if err := c.RemoveObject(context.Background(), bucketName, objectName, RemoveObjectOptions{}); err != nil {
		t.Fatal("Error: ", err)
	}

	if err := c.RemoveBucket(context.Background(), bucketName); err != nil {
		t.Fatal("Error: ", err)
	}

	// Do not need to remove destBucketName its same as bucketName.
}

// Test Core PutObject.
func TestCorePutObject(t *testing.T) {
	if os.Getenv(serverEndpoint) == "" {
		t.Skip("SERVER_ENDPOINT not set")
	}
	if testing.Short() {
		t.Skip("skipping functional tests for short runs")
	}

	// Instantiate new minio client object.
	c, err := NewCore(
		os.Getenv(serverEndpoint),
		&Options{
			Creds:  credentials.NewStaticV4(os.Getenv(accessKey), os.Getenv(secretKey), ""),
			Secure: mustParseBool(os.Getenv(enableSecurity)),
		})
	if err != nil {
		t.Fatal("Error:", err)
	}

	// Enable tracing, write to stderr.
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("MinIO-go-FunctionalTest", "0.1.0")

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket.
	err = c.MakeBucket(context.Background(), bucketName, MakeBucketOptions{Region: "us-east-1"})
	if err != nil {
		t.Fatal("Error:", err, bucketName)
	}

	buf := bytes.Repeat([]byte("a"), 32*1024)

	// Save the data
	objectName := randString(60, rand.NewSource(time.Now().UnixNano()), "")
	// Object content type
	objectContentType := "binary/octet-stream"
	metadata := make(map[string]string)
	metadata["Content-Type"] = objectContentType
	putopts := PutObjectOptions{
		UserMetadata: metadata,
	}
	_, err = c.PutObject(context.Background(), bucketName, objectName, bytes.NewReader(buf), int64(len(buf)), "1B2M2Y8AsgTpgAmY7PhCfg==", "", putopts)
	if err == nil {
		t.Fatal("Error expected: error, got: nil(success)")
	}

	_, err = c.PutObject(context.Background(), bucketName, objectName, bytes.NewReader(buf), int64(len(buf)), "", "", putopts)
	if err != nil {
		t.Fatal("Error:", err, bucketName, objectName)
	}

	// Read the data back
	r, err := c.Client.GetObject(context.Background(), bucketName, objectName, GetObjectOptions{})
	if err != nil {
		t.Fatal("Error:", err, bucketName, objectName)
	}

	st, err := r.Stat()
	if err != nil {
		t.Fatal("Error:", err, bucketName, objectName)
	}

	if st.Size != int64(len(buf)) {
		t.Fatalf("Error: number of bytes in stat does not match, want %v, got %v\n",
			len(buf), st.Size)
	}

	if st.ContentType != objectContentType {
		t.Fatalf("Error: Content types don't match, expected: %+v, found: %+v\n", objectContentType, st.ContentType)
	}

	if err := r.Close(); err != nil {
		t.Fatal("Error:", err)
	}

	if err := r.Close(); err == nil {
		t.Fatal("Error: object is already closed, should return error")
	}

	err = c.RemoveObject(context.Background(), bucketName, objectName, RemoveObjectOptions{})
	if err != nil {
		t.Fatal("Error: ", err)
	}

	err = c.RemoveBucket(context.Background(), bucketName)
	if err != nil {
		t.Fatal("Error:", err)
	}
}

func TestCoreGetObjectMetadata(t *testing.T) {
	if os.Getenv(serverEndpoint) == "" {
		t.Skip("SERVER_ENDPOINT not set")
	}
	if testing.Short() {
		t.Skip("skipping functional tests for the short runs")
	}

	core, err := NewCore(
		os.Getenv(serverEndpoint),
		&Options{
			Creds:  credentials.NewStaticV4(os.Getenv(accessKey), os.Getenv(secretKey), ""),
			Secure: mustParseBool(os.Getenv(enableSecurity)),
		})
	if err != nil {
		t.Fatal(err)
	}

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket.
	err = core.MakeBucket(context.Background(), bucketName, MakeBucketOptions{Region: "us-east-1"})
	if err != nil {
		t.Fatal("Error:", err, bucketName)
	}

	metadata := map[string]string{
		"X-Amz-Meta-Key-1": "Val-1",
	}
	putopts := PutObjectOptions{
		UserMetadata: metadata,
	}

	_, err = core.PutObject(context.Background(), bucketName, "my-objectname",
		bytes.NewReader([]byte("hello")), 5, "", "", putopts)
	if err != nil {
		t.Fatal(err)
	}

	reader, objInfo, _, err := core.GetObject(context.Background(), bucketName, "my-objectname", GetObjectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	reader.Close()

	if objInfo.Metadata.Get("X-Amz-Meta-Key-1") != "Val-1" {
		t.Fatal("Expected metadata to be available but wasn't")
	}

	err = core.RemoveObject(context.Background(), bucketName, "my-objectname", RemoveObjectOptions{})
	if err != nil {
		t.Fatal("Error: ", err)
	}
	err = core.RemoveBucket(context.Background(), bucketName)
	if err != nil {
		t.Fatal("Error:", err)
	}
}

func TestCoreMultipartUpload(t *testing.T) {
	if os.Getenv(serverEndpoint) == "" {
		t.Skip("SERVER_ENDPOINT not set")
	}
	if testing.Short() {
		t.Skip("skipping functional tests for the short runs")
	}

	// Instantiate new minio client object.
	core, err := NewCore(
		os.Getenv(serverEndpoint),
		&Options{
			Creds:  credentials.NewStaticV4(os.Getenv(accessKey), os.Getenv(secretKey), ""),
			Secure: mustParseBool(os.Getenv(enableSecurity)),
		})
	if err != nil {
		t.Fatal("Error:", err)
	}

	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")
	// Make a new bucket.
	err = core.MakeBucket(context.Background(), bucketName, MakeBucketOptions{Region: "us-east-1"})
	if err != nil {
		t.Fatal("Error:", err, bucketName)
	}
	objectName := randString(60, rand.NewSource(time.Now().UnixNano()), "")

	objectContentType := "binary/octet-stream"
	metadata := make(map[string]string)
	metadata["Content-Type"] = objectContentType
	putopts := PutObjectOptions{
		UserMetadata: metadata,
	}
	uploadID, err := core.NewMultipartUpload(context.Background(), bucketName, objectName, putopts)
	if err != nil {
		t.Fatal("Error:", err, bucketName, objectName)
	}
	buf := bytes.Repeat([]byte("a"), 32*1024*1024)
	r := bytes.NewReader(buf)
	partBuf := make([]byte, 100*1024*1024)
	parts := make([]CompletePart, 0, 5)
	partID := 0
	for {
		n, err := r.Read(partBuf)
		if err != nil && err != io.EOF {
			t.Fatal("Error:", err)
		}
		if err == io.EOF {
			break
		}
		if n > 0 {
			partID++
			data := bytes.NewReader(partBuf[:n])
			dataLen := int64(len(partBuf[:n]))
			objectPart, err := core.PutObjectPart(context.Background(), bucketName, objectName, uploadID, partID,
				data, dataLen,
				PutObjectPartOptions{
					Md5Base64:    "",
					Sha256Hex:    "",
					SSE:          encrypt.NewSSE(),
					CustomHeader: nil,
					Trailer:      nil,
				},
			)
			if err != nil {
				t.Fatal("Error:", err, bucketName, objectName)
			}
			parts = append(parts, CompletePart{
				PartNumber: partID,
				ETag:       objectPart.ETag,
			})
		}
	}
	objectParts, err := core.listObjectParts(context.Background(), bucketName, objectName, uploadID)
	if err != nil {
		t.Fatal("Error:", err)
	}
	if len(objectParts) != len(parts) {
		t.Fatal("Error", len(objectParts), len(parts))
	}
	_, err = core.CompleteMultipartUpload(context.Background(), bucketName, objectName, uploadID, parts, putopts)
	if err != nil {
		t.Fatal("Error:", err)
	}

	if err := core.RemoveObject(context.Background(), bucketName, objectName, RemoveObjectOptions{}); err != nil {
		t.Fatal("Error: ", err)
	}

	if err := core.RemoveBucket(context.Background(), bucketName); err != nil {
		t.Fatal("Error: ", err)
	}
}
