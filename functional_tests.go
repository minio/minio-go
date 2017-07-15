// +build ignore

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

package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	minio "github.com/minio/minio-go"
	log "github.com/sirupsen/logrus"

	"github.com/minio/minio-go/pkg/encrypt"
	"github.com/minio/minio-go/pkg/policy"
)

// MinPartSize ... Minimum part size
const MinPartSize = 1024 * 1024 * 64
const letterBytes = "abcdefghijklmnopqrstuvwxyz01234569"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)
const (
	serverEndpoint = "SERVER_ENDPOINT"
	accessKey      = "ACCESS_KEY"
	secretKey      = "SECRET_KEY"
	enableHTTPS    = "ENABLE_HTTPS"
)

func init() {
	// If server endpoint is not set, all tests default to
	// using https://play.minio.io:9000
	if os.Getenv(serverEndpoint) == "" {
		os.Setenv(serverEndpoint, "play.minio.io:9000")
		os.Setenv(accessKey, "Q3AM3UQ867SPQQA43P2F")
		os.Setenv(secretKey, "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG")
		os.Setenv(enableHTTPS, "1")
	}
}

func getDataDir() (dir string) {
	dir = os.Getenv("MINT_DATA_DIR")
	if dir == "" {
		dir = "/mint/data"
	}
	return
}

func getFilePath(filename string) (filepath string) {
	if getDataDir() != "" {
		filepath = getDataDir() + "/" + filename
	}
	return
}

// read data from file if it exists or optionally create a buffer of particular size
func getDataBuffer(fileName string, size int) (buf []byte) {
	if _, err := os.Stat(getFilePath(fileName)); os.IsNotExist(err) {
		buf = bytes.Repeat([]byte(string('a')), size)
		return
	}
	buf, _ = ioutil.ReadFile(getFilePath(fileName))
	return
}

// randString generates random names and prepends them with a known prefix.
func randString(n int, src rand.Source, prefix string) string {
	b := make([]byte, n)
	// A rand.Int63() generates 63 random bits, enough for letterIdxMax letters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}
	return prefix + string(b[0:30-len(prefix)])
}

func isQuickMode() bool {
	return os.Getenv("MODE") == "quick"
}

// Tests bucket re-create errors.
func testMakeBucketError() {
	logTrace()

	if os.Getenv(serverEndpoint) != "s3.amazonaws.com" {
		log.Info("skipping region functional tests for non s3 runs")
		return
	}

	// Seed random based on current time.
	rand.Seed(time.Now().Unix())

	// Instantiate new minio client object.
	c, err := minio.New(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatalf("Error: %s", err)
	}

	// Enable tracing, write to stderr.
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("Minio-go-FunctionalTest", "0.1.0")

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket in 'eu-central-1'.
	if err = c.MakeBucket(bucketName, "eu-central-1"); err != nil {
		log.Fatal("Error:", err, bucketName)
	}
	if err = c.MakeBucket(bucketName, "eu-central-1"); err == nil {
		log.Fatal("Error: make bucket should should fail for", bucketName)
	}
	// Verify valid error response from server.
	if minio.ToErrorResponse(err).Code != "BucketAlreadyExists" &&
		minio.ToErrorResponse(err).Code != "BucketAlreadyOwnedByYou" {
		log.Fatal("Error: Invalid error returned by server", err)
	}
	if err = c.RemoveBucket(bucketName); err != nil {
		log.Fatal("Error:", err, bucketName)
	}
}

// Tests various bucket supported formats.
func testMakeBucketRegions() {
	logTrace()

	if os.Getenv(serverEndpoint) != "s3.amazonaws.com" {
		log.Info("skipping region functional tests for non s3 runs")
		return
	}

	// Seed random based on current time.
	rand.Seed(time.Now().Unix())

	// Instantiate new minio client object.
	c, err := minio.New(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Enable tracing, write to stderr.
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("Minio-go-FunctionalTest", "0.1.0")

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket in 'eu-central-1'.
	if err = c.MakeBucket(bucketName, "eu-central-1"); err != nil {
		log.Fatal("Error:", err, bucketName)
	}

	if err = c.RemoveBucket(bucketName); err != nil {
		log.Fatal("Error:", err, bucketName)
	}

	// Make a new bucket with '.' in its name, in 'us-west-2'. This
	// request is internally staged into a path style instead of
	// virtual host style.
	if err = c.MakeBucket(bucketName+".withperiod", "us-west-2"); err != nil {
		log.Fatal("Error:", err, bucketName+".withperiod")
	}

	// Remove the newly created bucket.
	if err = c.RemoveBucket(bucketName + ".withperiod"); err != nil {
		log.Fatal("Error:", err, bucketName+".withperiod")
	}
}

// Test PutObject using a large data to trigger multipart readat
func testPutObjectReadAt() {
	logTrace()

	// Seed random based on current time.
	rand.Seed(time.Now().Unix())

	// Instantiate new minio client object.
	c, err := minio.New(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Enable tracing, write to stderr.
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("Minio-go-FunctionalTest", "0.1.0")

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket.
	err = c.MakeBucket(bucketName, "us-east-1")
	if err != nil {
		log.Fatal("Error:", err, bucketName)
	}

	// Generate data using 4 parts so that all 3 'workers' are utilized and a part is leftover.
	// Use different data for each part for multipart tests to ensure part order at the end.
	var buf = getDataBuffer("datafile-65-MB", MinPartSize)

	// Save the data
	objectName := randString(60, rand.NewSource(time.Now().UnixNano()), "")
	// Object content type
	objectContentType := "binary/octet-stream"

	n, err := c.PutObject(bucketName, objectName, bytes.NewReader(buf), objectContentType)
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}

	if n != int64(len(buf)) {
		log.Fatalf("Error: number of bytes does not match, want %v, got %v\n", len(buf), n)
	}

	// Read the data back
	r, err := c.GetObject(bucketName, objectName)
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}

	st, err := r.Stat()
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}
	if st.Size != int64(len(buf)) {
		log.Fatalf("Error: number of bytes in stat does not match, want %v, got %v\n",
			len(buf), st.Size)
	}
	if st.ContentType != objectContentType {
		log.Fatalf("Error: Content types don't match, expected: %+v, found: %+v\n", objectContentType, st.ContentType)
	}
	if err := r.Close(); err != nil {
		log.Fatal("Error:", err)
	}
	if err := r.Close(); err == nil {
		log.Fatal("Error: object is already closed, should return error")
	}

	err = c.RemoveObject(bucketName, objectName)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	err = c.RemoveBucket(bucketName)
	if err != nil {
		log.Fatal("Error:", err)
	}
}

// Test PutObject using a large data to trigger multipart readat
func testPutObjectWithMetadata() {
	logTrace()
	if isQuickMode() {
		log.Info("skipping functional tests for short runs")
		return
	}

	// Seed random based on current time.
	rand.Seed(time.Now().Unix())

	// Instantiate new minio client object.
	c, err := minio.New(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Enable tracing, write to stderr.
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("Minio-go-FunctionalTest", "0.1.0")

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket.
	err = c.MakeBucket(bucketName, "us-east-1")
	if err != nil {
		log.Fatal("Error:", err, bucketName)
	}

	// Generate data using 2 parts
	// Use different data in each part for multipart tests to ensure part order at the end.
	var buf = getDataBuffer("datafile-65-MB", MinPartSize)

	// Save the data
	objectName := randString(60, rand.NewSource(time.Now().UnixNano()), "")

	// Object custom metadata
	customContentType := "custom/contenttype"

	n, err := c.PutObjectWithMetadata(bucketName, objectName, bytes.NewReader(buf), map[string][]string{
		"Content-Type": {customContentType},
	}, nil)
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}

	if n != int64(len(buf)) {
		log.Fatalf("Error: number of bytes does not match, want %v, got %v\n", len(buf), n)
	}

	// Read the data back
	r, err := c.GetObject(bucketName, objectName)
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}

	st, err := r.Stat()
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}
	if st.Size != int64(len(buf)) {
		log.Fatalf("Error: number of bytes in stat does not match, want %v, got %v\n",
			len(buf), st.Size)
	}
	if st.ContentType != customContentType {
		log.Fatalf("Error: Expected and found content types do not match, want %v, got %v\n",
			customContentType, st.ContentType)
	}
	if err := r.Close(); err != nil {
		log.Fatal("Error:", err)
	}
	if err := r.Close(); err == nil {
		log.Fatal("Error: object is already closed, should return error")
	}

	err = c.RemoveObject(bucketName, objectName)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	err = c.RemoveBucket(bucketName)
	if err != nil {
		log.Fatal("Error:", err)
	}
}

// Test put object with streaming signature.
func testPutObjectStreaming() {
	logTrace()

	// Seed random based on current time.
	rand.Seed(time.Now().Unix())

	// Instantiate new minio client object.
	c, err := minio.NewV4(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Enable tracing, write to stderr.
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("Minio-go-FunctionalTest", "0.1.0")

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()),
		"minio-go-test")

	// Make a new bucket.
	err = c.MakeBucket(bucketName, "us-east-1")
	if err != nil {
		log.Fatal("Error:", err, bucketName)
	}

	// Upload an object.
	sizes := []int64{0, 64*1024 - 1, 64 * 1024}
	objectName := "test-object"
	for i, size := range sizes {
		data := bytes.Repeat([]byte("a"), int(size))
		n, err := c.PutObjectStreaming(bucketName, objectName, bytes.NewReader(data))
		if err != nil {
			log.Fatalf("Test %d Error: %v %s %s", i+1, err, bucketName, objectName)
		}

		if n != size {
			log.Error(fmt.Errorf("Test %d Expected upload object size %d but got %d", i+1, size, n))
		}
	}

	// Remove the object.
	err = c.RemoveObject(bucketName, objectName)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Remove the bucket.
	err = c.RemoveBucket(bucketName)
	if err != nil {
		log.Fatal("Error:", err)
	}
}

// Test listing partially uploaded objects.
func testListPartiallyUploaded() {
	logTrace()

	// Seed random based on current time.
	rand.Seed(time.Now().Unix())

	// Instantiate new minio client object.
	c, err := minio.New(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Set user agent.
	c.SetAppInfo("Minio-go-FunctionalTest", "0.1.0")

	// Enable tracing, write to stdout.
	// c.TraceOn(os.Stderr)

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket.
	err = c.MakeBucket(bucketName, "us-east-1")
	if err != nil {
		log.Fatal("Error:", err, bucketName)
	}

	r := bytes.NewReader(bytes.Repeat([]byte("0"), MinPartSize*2))

	reader, writer := io.Pipe()
	go func() {
		i := 0
		for i < 25 {
			_, cerr := io.CopyN(writer, r, (MinPartSize*2)/25)
			if cerr != nil {
				log.Fatal("Error:", cerr, bucketName)
			}
			i++
			r.Seek(0, 0)
		}
		writer.CloseWithError(errors.New("proactively closed to be verified later"))
	}()

	objectName := bucketName + "-resumable"
	_, err = c.PutObject(bucketName, objectName, reader, "application/octet-stream")
	if err == nil {
		log.Fatal("Error: PutObject should fail.")
	}
	if !strings.Contains(err.Error(), "proactively closed to be verified later") {
		log.Fatal("Error:", err)
	}

	doneCh := make(chan struct{})
	defer close(doneCh)
	isRecursive := true
	multiPartObjectCh := c.ListIncompleteUploads(bucketName, objectName, isRecursive, doneCh)
	for multiPartObject := range multiPartObjectCh {
		if multiPartObject.Err != nil {
			log.Fatalf("Error: Error when listing incomplete upload")
		}
	}

	err = c.RemoveBucket(bucketName)
	if err != nil {
		log.Fatal("Error:", err)
	}
}

// Test get object seeker from the end, using whence set to '2'.
func testGetObjectSeekEnd() {
	logTrace()

	// Seed random based on current time.
	rand.Seed(time.Now().Unix())

	// Instantiate new minio client object.
	c, err := minio.New(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Enable tracing, write to stderr.
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("Minio-go-FunctionalTest", "0.1.0")

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket.
	err = c.MakeBucket(bucketName, "us-east-1")
	if err != nil {
		log.Fatal("Error:", err, bucketName)
	}
	var buf = getDataBuffer("datafile-33-kB", rand.Intn(1<<20)+32*1024)

	// Save the data
	objectName := randString(60, rand.NewSource(time.Now().UnixNano()), "")
	n, err := c.PutObject(bucketName, objectName, bytes.NewReader(buf), "binary/octet-stream")
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}

	if n != int64(len(buf)) {
		log.Fatalf("Error: number of bytes does not match, want %v, got %v\n", len(buf), n)
	}

	// Read the data back
	r, err := c.GetObject(bucketName, objectName)
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}

	st, err := r.Stat()
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}
	if st.Size != int64(len(buf)) {
		log.Fatalf("Error: number of bytes in stat does not match, want %v, got %v\n",
			len(buf), st.Size)
	}

	pos, err := r.Seek(-100, 2)
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}
	if pos != st.Size-100 {
		log.Fatalf("Expected %d, got %d instead", pos, st.Size-100)
	}
	buf2 := make([]byte, 100)
	m, err := io.ReadFull(r, buf2)
	if err != nil {
		log.Fatal("Error: reading through io.ReadFull", err, bucketName, objectName)
	}
	if m != len(buf2) {
		log.Fatalf("Expected %d bytes, got %d", len(buf2), m)
	}
	hexBuf1 := fmt.Sprintf("%02x", buf[len(buf)-100:])
	hexBuf2 := fmt.Sprintf("%02x", buf2[:m])
	if hexBuf1 != hexBuf2 {
		log.Fatalf("Expected %s, got %s instead", hexBuf1, hexBuf2)
	}
	pos, err = r.Seek(-100, 2)
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}
	if pos != st.Size-100 {
		log.Fatalf("Expected %d, got %d instead", pos, st.Size-100)
	}
	if err = r.Close(); err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}
}

// Test get object reader to not throw error on being closed twice.
func testGetObjectClosedTwice() {
	logTrace()

	// Seed random based on current time.
	rand.Seed(time.Now().Unix())

	// Instantiate new minio client object.
	c, err := minio.New(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Enable tracing, write to stderr.
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("Minio-go-FunctionalTest", "0.1.0")

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket.
	err = c.MakeBucket(bucketName, "us-east-1")
	if err != nil {
		log.Fatal("Error:", err, bucketName)
	}

	// Generate data more than 32K
	var buf = getDataBuffer("datafile-33-kB", rand.Intn(1<<20)+32*1024)
	// Save the data
	objectName := randString(60, rand.NewSource(time.Now().UnixNano()), "")
	n, err := c.PutObject(bucketName, objectName, bytes.NewReader(buf), "binary/octet-stream")
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}

	if n != int64(len(buf)) {
		log.Fatalf("Error: number of bytes does not match, want %v, got %v\n", len(buf), n)
	}

	// Read the data back
	r, err := c.GetObject(bucketName, objectName)
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}

	st, err := r.Stat()
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}
	if st.Size != int64(len(buf)) {
		log.Fatalf("Error: number of bytes in stat does not match, want %v, got %v\n",
			len(buf), st.Size)
	}
	if err := r.Close(); err != nil {
		log.Fatal("Error:", err)
	}
	if err := r.Close(); err == nil {
		log.Fatal("Error: object is already closed, should return error")
	}

	err = c.RemoveObject(bucketName, objectName)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	err = c.RemoveBucket(bucketName)
	if err != nil {
		log.Fatal("Error:", err)
	}
}

// Test removing multiple objects with Remove API
func testRemoveMultipleObjects() {
	logTrace()
	// Seed random based on current time.
	rand.Seed(time.Now().Unix())

	// Instantiate new minio client object.
	c, err := minio.New(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)

	if err != nil {
		log.Fatal("Error:", err)
	}

	// Set user agent.
	c.SetAppInfo("Minio-go-FunctionalTest", "0.1.0")

	// Enable tracing, write to stdout.
	// c.TraceOn(os.Stderr)

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket.
	err = c.MakeBucket(bucketName, "us-east-1")
	if err != nil {
		log.Fatal("Error:", err, bucketName)
	}

	r := bytes.NewReader(bytes.Repeat([]byte("a"), 8))

	// Multi remove of 1100 objects
	nrObjects := 1100

	objectsCh := make(chan string)

	go func() {
		defer close(objectsCh)
		// Upload objects and send them to objectsCh
		for i := 0; i < nrObjects; i++ {
			objectName := "sample" + strconv.Itoa(i) + ".txt"
			_, err = c.PutObject(bucketName, objectName, r, "application/octet-stream")
			if err != nil {
				log.Error("Error: PutObject shouldn't fail.", err)
				continue
			}
			objectsCh <- objectName
		}
	}()

	// Call RemoveObjects API
	errorCh := c.RemoveObjects(bucketName, objectsCh)

	// Check if errorCh doesn't receive any error
	select {
	case r, more := <-errorCh:
		if more {
			log.Fatalf("Unexpected error, objName(%v) err(%v)", r.ObjectName, r.Err)
		}
	}

	// Clean the bucket created by the test
	err = c.RemoveBucket(bucketName)
	if err != nil {
		log.Fatal("Error:", err)
	}
}

// Tests removing partially uploaded objects.
func testRemovePartiallyUploaded() {
	logTrace()

	// Seed random based on current time.
	rand.Seed(time.Now().Unix())

	// Instantiate new minio client object.
	c, err := minio.New(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Set user agent.
	c.SetAppInfo("Minio-go-FunctionalTest", "0.1.0")

	// Enable tracing, write to stdout.
	// c.TraceOn(os.Stderr)

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket.
	err = c.MakeBucket(bucketName, "us-east-1")
	if err != nil {
		log.Fatal("Error:", err, bucketName)
	}

	r := bytes.NewReader(bytes.Repeat([]byte("a"), 128*1024))

	reader, writer := io.Pipe()
	go func() {
		i := 0
		for i < 25 {
			_, cerr := io.CopyN(writer, r, 128*1024)
			if cerr != nil {
				log.Fatal("Error:", cerr, bucketName)
			}
			i++
			r.Seek(0, 0)
		}
		writer.CloseWithError(errors.New("proactively closed to be verified later"))
	}()

	objectName := bucketName + "-resumable"
	_, err = c.PutObject(bucketName, objectName, reader, "application/octet-stream")
	if err == nil {
		log.Fatal("Error: PutObject should fail.")
	}
	if !strings.Contains(err.Error(), "proactively closed to be verified later") {
		log.Fatal("Error:", err)
	}
	err = c.RemoveIncompleteUpload(bucketName, objectName)
	if err != nil {
		log.Fatal("Error:", err)
	}
	err = c.RemoveBucket(bucketName)
	if err != nil {
		log.Fatal("Error:", err)
	}
}

// Tests FPutObject of a big file to trigger multipart
func testFPutObjectMultipart() {
	logTrace()

	// Seed random based on current time.
	rand.Seed(time.Now().Unix())

	// Instantiate new minio client object.
	c, err := minio.New(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Enable tracing, write to stderr.
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("Minio-go-FunctionalTest", "0.1.0")

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket.
	err = c.MakeBucket(bucketName, "us-east-1")
	if err != nil {
		log.Fatal("Error:", err, bucketName)
	}

	// Upload 4 parts to utilize all 3 'workers' in multipart and still have a part to upload.
	var fileName = getFilePath("datafile-65-MB")
	if os.Getenv("MINT_DATA_DIR") == "" {
		// Make a temp file with minPartSize bytes of data.
		file, err := ioutil.TempFile(os.TempDir(), "FPutObjectTest")
		if err != nil {
			log.Fatal("Error:", err)
		}

		// Upload 4 parts to utilize all 3 'workers' in multipart and still have a part to upload.
		var buffer = bytes.Repeat([]byte(string('a')), MinPartSize)
		if _, err := file.Write(buffer); err != nil {
			log.Fatal("Error:", err)
		}
		// Close the file pro-actively for windows.
		err = file.Close()
		if err != nil {
			log.Fatal("Error:", err)
		}
		fileName = file.Name()
	}
	totalSize := MinPartSize * 1
	// Set base object name
	objectName := bucketName + "FPutObject"
	objectContentType := "testapplication/octet-stream"

	// Perform standard FPutObject with contentType provided (Expecting application/octet-stream)
	n, err := c.FPutObject(bucketName, objectName+"-standard", fileName, objectContentType)
	if err != nil {
		log.Fatal("Error:", err)
	}
	if n != int64(totalSize) {
		log.Fatalf("Error: number of bytes does not match, want %v, got %v\n", totalSize, n)
	}

	r, err := c.GetObject(bucketName, objectName+"-standard")
	if err != nil {
		log.Fatalf("Unexpected error: %v\n", err)
	}
	objInfo, err := r.Stat()
	if err != nil {
		log.Fatalf("Unexpected error: %v\n", err)
	}
	if objInfo.Size != int64(totalSize) {
		log.Fatalf("Error: number of bytes does not match, want %v, got %v\n", totalSize, n)
	}
	if objInfo.ContentType != objectContentType {
		log.Fatalf("Error: Content types don't match, want %v, got %v\n", objectContentType, objInfo.ContentType)
	}

	// Remove all objects and bucket and temp file
	err = c.RemoveObject(bucketName, objectName+"-standard")
	if err != nil {
		log.Fatal("Error: ", err)
	}

	err = c.RemoveBucket(bucketName)
	if err != nil {
		log.Fatal("Error:", err)
	}
}

// Tests FPutObject hidden contentType setting
func testFPutObject() {
	logTrace()

	// Seed random based on current time.
	rand.Seed(time.Now().Unix())

	// Instantiate new minio client object.
	c, err := minio.New(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Enable tracing, write to stderr.
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("Minio-go-FunctionalTest", "0.1.0")

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket.
	err = c.MakeBucket(bucketName, "us-east-1")
	if err != nil {
		log.Fatal("Error:", err, bucketName)
	}

	// Upload 3 parts worth of data to use all 3 of multiparts 'workers' and have an extra part.
	// Use different data in part for multipart tests to check parts are uploaded in correct order.
	var fName = getFilePath("datafile-65-MB")
	if os.Getenv("MINT_DATA_DIR") == "" {
		// Make a temp file with minPartSize bytes of data.
		file, err := ioutil.TempFile(os.TempDir(), "FPutObjectTest")
		if err != nil {
			log.Fatal("Error:", err)
		}

		// Upload 4 parts to utilize all 3 'workers' in multipart and still have a part to upload.
		var buffer = bytes.Repeat([]byte(string('a')), MinPartSize)
		if _, err = file.Write(buffer); err != nil {
			log.Fatal("Error:", err)
		}
		// Close the file pro-actively for windows.
		err = file.Close()
		if err != nil {
			log.Fatal("Error:", err)
		}
		fName = file.Name()
	}
	var totalSize = MinPartSize * 1

	// Set base object name
	objectName := bucketName + "FPutObject"

	// Perform standard FPutObject with contentType provided (Expecting application/octet-stream)
	n, err := c.FPutObject(bucketName, objectName+"-standard", fName, "application/octet-stream")
	if err != nil {
		log.Fatal("Error:", err)
	}
	if n != int64(totalSize) {
		log.Fatalf("Error: number of bytes does not match, want %v, got %v\n", totalSize, n)
	}

	// Perform FPutObject with no contentType provided (Expecting application/octet-stream)
	n, err = c.FPutObject(bucketName, objectName+"-Octet", fName, "")
	if err != nil {
		log.Fatal("Error:", err)
	}
	if n != int64(totalSize) {
		log.Fatalf("Error: number of bytes does not match, want %v, got %v\n", totalSize, n)
	}
	srcFile, err := os.Open(fName)
	if err != nil {
		log.Fatal("Error:", err)
	}
	defer srcFile.Close()
	// Add extension to temp file name
	tmpFile, err := os.Create(fName + ".gtar")
	if err != nil {
		log.Fatal("Error:", err)
	}
	defer tmpFile.Close()
	_, err = io.Copy(tmpFile, srcFile)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Perform FPutObject with no contentType provided (Expecting application/x-gtar)
	n, err = c.FPutObject(bucketName, objectName+"-GTar", fName+".gtar", "")
	if err != nil {
		log.Fatal("Error:", err)
	}
	if n != int64(totalSize) {
		log.Fatalf("Error: number of bytes does not match, want %v, got %v\n", totalSize, n)
	}

	// Check headers
	rStandard, err := c.StatObject(bucketName, objectName+"-standard")
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName+"-standard")
	}
	if rStandard.ContentType != "application/octet-stream" {
		log.Fatalf("Error: Content-Type headers mismatched, want %v, got %v\n",
			"application/octet-stream", rStandard.ContentType)
	}

	rOctet, err := c.StatObject(bucketName, objectName+"-Octet")
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName+"-Octet")
	}
	if rOctet.ContentType != "application/octet-stream" {
		log.Fatalf("Error: Content-Type headers mismatched, want %v, got %v\n",
			"application/octet-stream", rStandard.ContentType)
	}

	rGTar, err := c.StatObject(bucketName, objectName+"-GTar")
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName+"-GTar")
	}
	if rGTar.ContentType != "application/x-gtar" {
		log.Fatalf("Error: Content-Type headers mismatched, want %v, got %v\n",
			"application/x-gtar", rStandard.ContentType)
	}

	// Remove all objects and bucket and temp file
	err = c.RemoveObject(bucketName, objectName+"-standard")
	if err != nil {
		log.Fatal("Error: ", err)
	}

	err = c.RemoveObject(bucketName, objectName+"-Octet")
	if err != nil {
		log.Fatal("Error: ", err)
	}

	err = c.RemoveObject(bucketName, objectName+"-GTar")
	if err != nil {
		log.Fatal("Error: ", err)
	}

	err = c.RemoveBucket(bucketName)
	if err != nil {
		log.Fatal("Error:", err)
	}

	err = os.Remove(fName + ".gtar")
	if err != nil {
		log.Fatal("Error:", err)
	}

}

// Tests get object ReaderSeeker interface methods.
func testGetObjectReadSeekFunctional() {
	logTrace()

	// Seed random based on current time.
	rand.Seed(time.Now().Unix())

	// Instantiate new minio client object.
	c, err := minio.New(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Enable tracing, write to stderr.
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("Minio-go-FunctionalTest", "0.1.0")

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket.
	err = c.MakeBucket(bucketName, "us-east-1")
	if err != nil {
		log.Fatal("Error:", err, bucketName)
	}

	// Generate data more than 32K
	var buf = getDataBuffer("datafile-33-kB", rand.Intn(1<<20)+32*1024)

	bufSize := len(buf)

	// Save the data
	objectName := randString(60, rand.NewSource(time.Now().UnixNano()), "")
	n, err := c.PutObject(bucketName, objectName, bytes.NewReader(buf), "binary/octet-stream")
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}

	if n != int64(bufSize) {
		log.Fatalf("Error: number of bytes does not match, want %v, got %v\n", len(buf), n)
	}

	defer func() {
		err = c.RemoveObject(bucketName, objectName)
		if err != nil {
			log.Fatal("Error: ", err)
		}
		err = c.RemoveBucket(bucketName)
		if err != nil {
			log.Fatal("Error:", err)
		}
	}()

	// Read the data back
	r, err := c.GetObject(bucketName, objectName)
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}

	st, err := r.Stat()
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}
	if st.Size != int64(bufSize) {
		log.Fatalf("Error: number of bytes in stat does not match, want %v, got %v\n",
			len(buf), st.Size)
	}

	// This following function helps us to compare data from the reader after seek
	// with the data from the original buffer
	cmpData := func(r io.Reader, start, end int) {
		if end-start == 0 {
			return
		}
		buffer := bytes.NewBuffer([]byte{})
		if _, err := io.CopyN(buffer, r, int64(bufSize)); err != nil {
			if err != io.EOF {
				log.Fatal("Error:", err)
			}
		}
		if !bytes.Equal(buf[start:end], buffer.Bytes()) {
			log.Fatal("Error: Incorrect read bytes v/s original buffer.")
		}
	}

	// Generic seek error for errors other than io.EOF
	seekErr := errors.New("seek error")

	testCases := []struct {
		offset    int64
		whence    int
		pos       int64
		err       error
		shouldCmp bool
		start     int
		end       int
	}{
		// Start from offset 0, fetch data and compare
		{0, 0, 0, nil, true, 0, 0},
		// Start from offset 2048, fetch data and compare
		{2048, 0, 2048, nil, true, 2048, bufSize},
		// Start from offset larger than possible
		{int64(bufSize) + 1024, 0, 0, seekErr, false, 0, 0},
		// Move to offset 0 without comparing
		{0, 0, 0, nil, false, 0, 0},
		// Move one step forward and compare
		{1, 1, 1, nil, true, 1, bufSize},
		// Move larger than possible
		{int64(bufSize), 1, 0, seekErr, false, 0, 0},
		// Provide negative offset with CUR_SEEK
		{int64(-1), 1, 0, seekErr, false, 0, 0},
		// Test with whence SEEK_END and with positive offset
		{1024, 2, int64(bufSize) - 1024, io.EOF, true, 0, 0},
		// Test with whence SEEK_END and with negative offset
		{-1024, 2, int64(bufSize) - 1024, nil, true, bufSize - 1024, bufSize},
		// Test with whence SEEK_END and with large negative offset
		{-int64(bufSize) * 2, 2, 0, seekErr, true, 0, 0},
	}

	for i, testCase := range testCases {
		// Perform seek operation
		n, err := r.Seek(testCase.offset, testCase.whence)
		// We expect an error
		if testCase.err == seekErr && err == nil {
			log.Fatalf("Test %d, unexpected err value: expected: %v, found: %v", i+1, testCase.err, err)
		}
		// We expect a specific error
		if testCase.err != seekErr && testCase.err != err {
			log.Fatalf("Test %d, unexpected err value: expected: %v, found: %v", i+1, testCase.err, err)
		}
		// If we expect an error go to the next loop
		if testCase.err != nil {
			continue
		}
		// Check the returned seek pos
		if n != testCase.pos {
			log.Fatalf("Test %d, error: number of bytes seeked does not match, want %v, got %v\n", i+1,
				testCase.pos, n)
		}
		// Compare only if shouldCmp is activated
		if testCase.shouldCmp {
			cmpData(r, testCase.start, testCase.end)
		}
	}
}

// Tests get object ReaderAt interface methods.
func testGetObjectReadAtFunctional() {
	logTrace()

	// Seed random based on current time.
	rand.Seed(time.Now().Unix())

	// Instantiate new minio client object.
	c, err := minio.New(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Enable tracing, write to stderr.
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("Minio-go-FunctionalTest", "0.1.0")

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket.
	err = c.MakeBucket(bucketName, "us-east-1")
	if err != nil {
		log.Fatal("Error:", err, bucketName)
	}

	// Generate data more than 32K
	var buf = getDataBuffer("datafile-33-kB", rand.Intn(1<<20)+32*1024)

	// Save the data
	objectName := randString(60, rand.NewSource(time.Now().UnixNano()), "")
	n, err := c.PutObject(bucketName, objectName, bytes.NewReader(buf), "binary/octet-stream")
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}

	if n != int64(len(buf)) {
		log.Fatalf("Error: number of bytes does not match, want %v, got %v\n", len(buf), n)
	}

	// read the data back
	r, err := c.GetObject(bucketName, objectName)
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}
	offset := int64(2048)

	// read directly
	buf1 := make([]byte, 512)
	buf2 := make([]byte, 512)
	buf3 := make([]byte, 512)
	buf4 := make([]byte, 512)

	// Test readAt before stat is called.
	m, err := r.ReadAt(buf1, offset)
	if err != nil {
		log.Fatal("Error:", err, len(buf1), offset)
	}
	if m != len(buf1) {
		log.Fatalf("Error: ReadAt read shorter bytes before reaching EOF, want %v, got %v\n", m, len(buf1))
	}
	if !bytes.Equal(buf1, buf[offset:offset+512]) {
		log.Fatal("Error: Incorrect read between two ReadAt from same offset.")
	}
	offset += 512

	st, err := r.Stat()
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}
	if st.Size != int64(len(buf)) {
		log.Fatalf("Error: number of bytes in stat does not match, want %v, got %v\n",
			len(buf), st.Size)
	}

	m, err = r.ReadAt(buf2, offset)
	if err != nil {
		log.Fatal("Error:", err, st.Size, len(buf2), offset)
	}
	if m != len(buf2) {
		log.Fatalf("Error: ReadAt read shorter bytes before reaching EOF, want %v, got %v\n", m, len(buf2))
	}
	if !bytes.Equal(buf2, buf[offset:offset+512]) {
		log.Fatal("Error: Incorrect read between two ReadAt from same offset.")
	}
	offset += 512
	m, err = r.ReadAt(buf3, offset)
	if err != nil {
		log.Fatal("Error:", err, st.Size, len(buf3), offset)
	}
	if m != len(buf3) {
		log.Fatalf("Error: ReadAt read shorter bytes before reaching EOF, want %v, got %v\n", m, len(buf3))
	}
	if !bytes.Equal(buf3, buf[offset:offset+512]) {
		log.Fatal("Error: Incorrect read between two ReadAt from same offset.")
	}
	offset += 512
	m, err = r.ReadAt(buf4, offset)
	if err != nil {
		log.Fatal("Error:", err, st.Size, len(buf4), offset)
	}
	if m != len(buf4) {
		log.Fatalf("Error: ReadAt read shorter bytes before reaching EOF, want %v, got %v\n", m, len(buf4))
	}
	if !bytes.Equal(buf4, buf[offset:offset+512]) {
		log.Fatal("Error: Incorrect read between two ReadAt from same offset.")
	}

	buf5 := make([]byte, n)
	// Read the whole object.
	m, err = r.ReadAt(buf5, 0)
	if err != nil {
		if err != io.EOF {
			log.Fatal("Error:", err, len(buf5))
		}
	}
	if m != len(buf5) {
		log.Fatalf("Error: ReadAt read shorter bytes before reaching EOF, want %v, got %v\n", m, len(buf5))
	}
	if !bytes.Equal(buf, buf5) {
		log.Fatal("Error: Incorrect data read in GetObject, than what was previously upoaded.")
	}

	buf6 := make([]byte, n+1)
	// Read the whole object and beyond.
	_, err = r.ReadAt(buf6, 0)
	if err != nil {
		if err != io.EOF {
			log.Fatal("Error:", err, len(buf6))
		}
	}
	err = c.RemoveObject(bucketName, objectName)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	err = c.RemoveBucket(bucketName)
	if err != nil {
		log.Fatal("Error:", err)
	}
}

// Test Presigned Post Policy
func testPresignedPostPolicy() {
	logTrace()

	// Seed random based on current time.
	rand.Seed(time.Now().Unix())

	// Instantiate new minio client object
	c, err := minio.NewV4(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Enable tracing, write to stderr.
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("Minio-go-FunctionalTest", "0.1.0")

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket in 'us-east-1' (source bucket).
	err = c.MakeBucket(bucketName, "us-east-1")
	if err != nil {
		log.Fatal("Error:", err, bucketName)
	}

	// Generate data more than 32K
	var buf = getDataBuffer("datafile-33-kB", rand.Intn(1<<20)+32*1024)

	// Save the data
	objectName := randString(60, rand.NewSource(time.Now().UnixNano()), "")
	n, err := c.PutObject(bucketName, objectName, bytes.NewReader(buf), "binary/octet-stream")
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}

	if n != int64(len(buf)) {
		log.Fatalf("Error: number of bytes does not match want %v, got %v",
			len(buf), n)
	}

	policy := minio.NewPostPolicy()

	if err := policy.SetBucket(""); err == nil {
		log.Fatalf("Error: %s", err)
	}
	if err := policy.SetKey(""); err == nil {
		log.Fatalf("Error: %s", err)
	}
	if err := policy.SetKeyStartsWith(""); err == nil {
		log.Fatalf("Error: %s", err)
	}
	if err := policy.SetExpires(time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)); err == nil {
		log.Fatalf("Error: %s", err)
	}
	if err := policy.SetContentType(""); err == nil {
		log.Fatalf("Error: %s", err)
	}
	if err := policy.SetContentLengthRange(1024*1024, 1024); err == nil {
		log.Fatalf("Error: %s", err)
	}

	policy.SetBucket(bucketName)
	policy.SetKey(objectName)
	policy.SetExpires(time.Now().UTC().AddDate(0, 0, 10)) // expires in 10 days
	policy.SetContentType("image/png")
	policy.SetContentLengthRange(1024, 1024*1024)

	_, _, err = c.PresignedPostPolicy(policy)
	if err != nil {
		log.Fatal("Error:", err)
	}

	policy = minio.NewPostPolicy()

	// Remove all objects and buckets
	err = c.RemoveObject(bucketName, objectName)
	if err != nil {
		log.Fatal("Error:", err)
	}

	err = c.RemoveBucket(bucketName)
	if err != nil {
		log.Fatal("Error:", err)
	}
}

// Tests copy object
func testCopyObject() {

	// Seed random based on current time.
	rand.Seed(time.Now().Unix())

	// Instantiate new minio client object
	c, err := minio.NewV4(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Enable tracing, write to stderr.
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("Minio-go-FunctionalTest", "0.1.0")

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket in 'us-east-1' (source bucket).
	err = c.MakeBucket(bucketName, "us-east-1")
	if err != nil {
		log.Fatal("Error:", err, bucketName)
	}

	// Make a new bucket in 'us-east-1' (destination bucket).
	err = c.MakeBucket(bucketName+"-copy", "us-east-1")
	if err != nil {
		log.Fatal("Error:", err, bucketName+"-copy")
	}

	// Generate data more than 32K
	buf := bytes.Repeat([]byte("5"), rand.Intn(1<<20)+32*1024)

	// Save the data
	objectName := randString(60, rand.NewSource(time.Now().UnixNano()), "")
	n, err := c.PutObject(bucketName, objectName, bytes.NewReader(buf), "binary/octet-stream")
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}

	if n != int64(len(buf)) {
		log.Fatalf("Error: number of bytes does not match want %v, got %v",
			len(buf), n)
	}

	r, err := c.GetObject(bucketName, objectName)
	if err != nil {
		log.Fatal("Error:", err)
	}
	// Check the various fields of source object against destination object.
	objInfo, err := r.Stat()
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Copy Source
	src := minio.NewSourceInfo(bucketName, objectName, nil)

	// Set copy conditions.

	// All invalid conditions first.
	err = src.SetModifiedSinceCond(time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC))
	if err == nil {
		log.Fatal("Error:", err)
	}
	err = src.SetUnmodifiedSinceCond(time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC))
	if err == nil {
		log.Fatal("Error:", err)
	}
	err = src.SetMatchETagCond("")
	if err == nil {
		log.Fatal("Error:", err)
	}
	err = src.SetMatchETagExceptCond("")
	if err == nil {
		log.Fatal("Error:", err)
	}

	err = src.SetModifiedSinceCond(time.Date(2014, time.April, 0, 0, 0, 0, 0, time.UTC))
	if err != nil {
		log.Fatal("Error:", err)
	}
	err = src.SetMatchETagCond(objInfo.ETag)
	if err != nil {
		log.Fatal("Error:", err)
	}

	dst, err := minio.NewDestinationInfo(bucketName+"-copy", objectName+"-copy", nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Perform the Copy
	err = c.CopyObject(dst, src)
	if err != nil {
		log.Fatal("Error:", err, bucketName+"-copy", objectName+"-copy")
	}

	// Source object
	reader, err := c.GetObject(bucketName, objectName)
	if err != nil {
		log.Fatal("Error:", err)
	}
	// Destination object
	readerCopy, err := c.GetObject(bucketName+"-copy", objectName+"-copy")
	if err != nil {
		log.Fatal("Error:", err)
	}
	// Check the various fields of source object against destination object.
	objInfo, err = reader.Stat()
	if err != nil {
		log.Fatal("Error:", err)
	}
	objInfoCopy, err := readerCopy.Stat()
	if err != nil {
		log.Fatal("Error:", err)
	}
	if objInfo.Size != objInfoCopy.Size {
		log.Fatalf("Error: number of bytes does not match, want %v, got %v\n",
			objInfo.Size, objInfoCopy.Size)
	}

	// CopyObject again but with wrong conditions
	src = minio.NewSourceInfo(bucketName, objectName, nil)
	err = src.SetUnmodifiedSinceCond(time.Date(2014, time.April, 0, 0, 0, 0, 0, time.UTC))
	if err != nil {
		log.Fatal("Error:", err)
	}
	err = src.SetMatchETagExceptCond(objInfo.ETag)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Perform the Copy which should fail
	err = c.CopyObject(dst, src)
	if err == nil {
		log.Fatal("Error:", err, bucketName+"-copy", objectName+"-copy should fail")
	}

	// Remove all objects and buckets
	err = c.RemoveObject(bucketName, objectName)
	if err != nil {
		log.Fatal("Error:", err)
	}

	err = c.RemoveObject(bucketName+"-copy", objectName+"-copy")
	if err != nil {
		log.Fatal("Error:", err)
	}

	err = c.RemoveBucket(bucketName)
	if err != nil {
		log.Fatal("Error:", err)
	}

	err = c.RemoveBucket(bucketName + "-copy")
	if err != nil {
		log.Fatal("Error:", err)
	}
}

// TestEncryptionPutGet tests client side encryption
func testEncryptionPutGet() {
	logTrace()

	// Seed random based on current time.
	rand.Seed(time.Now().Unix())

	// Instantiate new minio client object.
	c, err := minio.New(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Enable tracing, write to stderr.
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("Minio-go-FunctionalTest", "0.1.0")

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket.
	err = c.MakeBucket(bucketName, "us-east-1")
	if err != nil {
		log.Fatal("Error:", err, bucketName)
	}

	// Generate a symmetric key
	symKey := encrypt.NewSymmetricKey([]byte("my-secret-key-00"))

	// Generate an assymmetric key from predefine public and private certificates
	privateKey, err := hex.DecodeString(
		"30820277020100300d06092a864886f70d0101010500048202613082025d" +
			"0201000281810087b42ea73243a3576dc4c0b6fa245d339582dfdbddc20c" +
			"bb8ab666385034d997210c54ba79275c51162a1221c3fb1a4c7c61131ca6" +
			"5563b319d83474ef5e803fbfa7e52b889e1893b02586b724250de7ac6351" +
			"cc0b7c638c980acec0a07020a78eed7eaa471eca4b92071394e061346c06" +
			"15ccce2f465dee2080a89e43f29b5702030100010281801dd5770c3af8b3" +
			"c85cd18cacad81a11bde1acfac3eac92b00866e142301fee565365aa9af4" +
			"57baebf8bb7711054d071319a51dd6869aef3848ce477a0dc5f0dbc0c336" +
			"5814b24c820491ae2bb3c707229a654427e03307fec683e6b27856688f08" +
			"bdaa88054c5eeeb773793ff7543ee0fb0e2ad716856f2777f809ef7e6fa4" +
			"41024100ca6b1edf89e8a8f93cce4b98c76c6990a09eb0d32ad9d3d04fbf" +
			"0b026fa935c44f0a1c05dd96df192143b7bda8b110ec8ace28927181fd8c" +
			"d2f17330b9b63535024100aba0260afb41489451baaeba423bee39bcbd1e" +
			"f63dd44ee2d466d2453e683bf46d019a8baead3a2c7fca987988eb4d565e" +
			"27d6be34605953f5034e4faeec9bdb0241009db2cb00b8be8c36710aff96" +
			"6d77a6dec86419baca9d9e09a2b761ea69f7d82db2ae5b9aae4246599bb2" +
			"d849684d5ab40e8802cfe4a2b358ad56f2b939561d2902404e0ead9ecafd" +
			"bb33f22414fa13cbcc22a86bdf9c212ce1a01af894e3f76952f36d6c904c" +
			"bd6a7e0de52550c9ddf31f1e8bfe5495f79e66a25fca5c20b3af5b870241" +
			"0083456232aa58a8c45e5b110494599bda8dbe6a094683a0539ddd24e19d" +
			"47684263bbe285ad953d725942d670b8f290d50c0bca3d1dc9688569f1d5" +
			"9945cb5c7d")

	if err != nil {
		log.Fatal(err)
	}

	publicKey, err := hex.DecodeString("30819f300d06092a864886f70d010101050003818d003081890281810087" +
		"b42ea73243a3576dc4c0b6fa245d339582dfdbddc20cbb8ab666385034d9" +
		"97210c54ba79275c51162a1221c3fb1a4c7c61131ca65563b319d83474ef" +
		"5e803fbfa7e52b889e1893b02586b724250de7ac6351cc0b7c638c980ace" +
		"c0a07020a78eed7eaa471eca4b92071394e061346c0615ccce2f465dee20" +
		"80a89e43f29b570203010001")
	if err != nil {
		log.Fatal(err)
	}

	// Generate an asymmetric key
	asymKey, err := encrypt.NewAsymmetricKey(privateKey, publicKey)
	if err != nil {
		log.Fatal(err)
	}

	// Object custom metadata
	customContentType := "custom/contenttype"

	testCases := []struct {
		buf    []byte
		encKey encrypt.Key
	}{
		{encKey: symKey, buf: bytes.Repeat([]byte("F"), 0)},
		{encKey: symKey, buf: bytes.Repeat([]byte("F"), 1)},
		{encKey: symKey, buf: bytes.Repeat([]byte("F"), 15)},
		{encKey: symKey, buf: bytes.Repeat([]byte("F"), 16)},
		{encKey: symKey, buf: bytes.Repeat([]byte("F"), 17)},
		{encKey: symKey, buf: bytes.Repeat([]byte("F"), 31)},
		{encKey: symKey, buf: bytes.Repeat([]byte("F"), 32)},
		{encKey: symKey, buf: bytes.Repeat([]byte("F"), 33)},
		{encKey: symKey, buf: bytes.Repeat([]byte("F"), 1024)},
		{encKey: symKey, buf: bytes.Repeat([]byte("F"), 1024*2)},
		{encKey: symKey, buf: bytes.Repeat([]byte("F"), 1024*1024)},

		{encKey: asymKey, buf: bytes.Repeat([]byte("F"), 0)},
		{encKey: asymKey, buf: bytes.Repeat([]byte("F"), 1)},
		{encKey: asymKey, buf: bytes.Repeat([]byte("F"), 16)},
		{encKey: asymKey, buf: bytes.Repeat([]byte("F"), 32)},
		{encKey: asymKey, buf: bytes.Repeat([]byte("F"), 1024)},
		{encKey: asymKey, buf: bytes.Repeat([]byte("F"), 1024*1024)},
	}

	for i, testCase := range testCases {
		// Generate a random object name
		objectName := randString(60, rand.NewSource(time.Now().UnixNano()), "")

		// Secured object
		cbcMaterials, err := encrypt.NewCBCSecureMaterials(testCase.encKey)
		if err != nil {
			log.Fatal(err)
		}

		// Put encrypted data
		_, err = c.PutEncryptedObject(bucketName, objectName, bytes.NewReader(testCase.buf), cbcMaterials, map[string][]string{"Content-Type": {customContentType}}, nil)
		if err != nil {
			log.Fatalf("Test %d, error: %v %v %v", i+1, err, bucketName, objectName)
		}

		// Read the data back
		r, err := c.GetEncryptedObject(bucketName, objectName, cbcMaterials)
		if err != nil {
			log.Fatalf("Test %d, error: %v %v %v", i+1, err, bucketName, objectName)
		}
		defer r.Close()

		// Compare the sent object with the received one
		recvBuffer := bytes.NewBuffer([]byte{})
		if _, err = io.Copy(recvBuffer, r); err != nil {
			log.Fatalf("Test %d, error: %v", i+1, err)
		}
		if recvBuffer.Len() != len(testCase.buf) {
			log.Fatalf("Test %d, error: number of bytes of received object does not match, want %v, got %v\n",
				i+1, len(testCase.buf), recvBuffer.Len())
		}
		if !bytes.Equal(testCase.buf, recvBuffer.Bytes()) {
			log.Fatalf("Test %d, error: Encrypted sent is not equal to decrypted, want `%x`, go `%x`", i+1, testCase.buf, recvBuffer.Bytes())
		}

		// Remove test object
		err = c.RemoveObject(bucketName, objectName)
		if err != nil {
			log.Fatalf("Test %d, error: %v", i+1, err)
		}

	}

	// Remove test bucket
	err = c.RemoveBucket(bucketName)
	if err != nil {
		log.Fatal("Error:", err)
	}

}

func testBucketNotification() {
	logTrace()

	if os.Getenv("NOTIFY_BUCKET") == "" ||
		os.Getenv("NOTIFY_SERVICE") == "" ||
		os.Getenv("NOTIFY_REGION") == "" ||
		os.Getenv("NOTIFY_ACCOUNTID") == "" ||
		os.Getenv("NOTIFY_RESOURCE") == "" {
		log.Info("skipping notification test if not configured")
		return
	}

	// Seed random based on current time.
	rand.Seed(time.Now().Unix())

	c, err := minio.New(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Enable to debug
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("Minio-go-FunctionalTest", "0.1.0")

	bucketName := os.Getenv("NOTIFY_BUCKET")

	topicArn := minio.NewArn("aws", os.Getenv("NOTIFY_SERVICE"), os.Getenv("NOTIFY_REGION"), os.Getenv("NOTIFY_ACCOUNTID"), os.Getenv("NOTIFY_RESOURCE"))
	queueArn := minio.NewArn("aws", "dummy-service", "dummy-region", "dummy-accountid", "dummy-resource")

	topicConfig := minio.NewNotificationConfig(topicArn)
	topicConfig.AddEvents(minio.ObjectCreatedAll, minio.ObjectRemovedAll)
	topicConfig.AddFilterSuffix("jpg")

	queueConfig := minio.NewNotificationConfig(queueArn)
	queueConfig.AddEvents(minio.ObjectCreatedAll)
	queueConfig.AddFilterPrefix("photos/")

	bNotification := minio.BucketNotification{}
	bNotification.AddTopic(topicConfig)

	// Add the same topicConfig again, should have no effect
	// because it is duplicated
	bNotification.AddTopic(topicConfig)
	if len(bNotification.TopicConfigs) != 1 {
		log.Fatal("Error: duplicated entry added")
	}

	// Add and remove a queue config
	bNotification.AddQueue(queueConfig)
	bNotification.RemoveQueueByArn(queueArn)

	err = c.SetBucketNotification(bucketName, bNotification)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	bNotification, err = c.GetBucketNotification(bucketName)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	if len(bNotification.TopicConfigs) != 1 {
		log.Fatal("Error: Topic config is empty")
	}

	if bNotification.TopicConfigs[0].Filter.S3Key.FilterRules[0].Value != "jpg" {
		log.Fatal("Error: cannot get the suffix")
	}

	err = c.RemoveAllBucketNotification(bucketName)
	if err != nil {
		log.Fatal("Error: cannot delete bucket notification")
	}
}

// Tests comprehensive list of all methods.
func testFunctional() {
	logTrace()

	// Seed random based on current time.
	rand.Seed(time.Now().Unix())

	c, err := minio.New(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Enable to debug
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("Minio-go-FunctionalTest", "0.1.0")

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket.
	err = c.MakeBucket(bucketName, "us-east-1")
	if err != nil {
		log.Fatal("Error:", err, bucketName)
	}

	// Generate a random file name.
	fileName := randString(60, rand.NewSource(time.Now().UnixNano()), "")
	file, err := os.Create(fileName)
	if err != nil {
		log.Fatal("Error:", err)
	}
	for i := 0; i < 3; i++ {
		buf := make([]byte, rand.Intn(1<<19))
		_, err = file.Write(buf)
		if err != nil {
			log.Fatal("Error:", err)
		}
	}
	file.Close()

	// Verify if bucket exits and you have access.
	var exists bool
	exists, err = c.BucketExists(bucketName)
	if err != nil {
		log.Fatal("Error:", err, bucketName)
	}
	if !exists {
		log.Fatal("Error: could not find ", bucketName)
	}

	// Asserting the default bucket policy.
	policyAccess, err := c.GetBucketPolicy(bucketName, "")
	if err != nil {
		log.Fatal("Error:", err)
	}
	if policyAccess != "none" {
		log.Fatalf("Default bucket policy incorrect")
	}
	// Set the bucket policy to 'public readonly'.
	err = c.SetBucketPolicy(bucketName, "", policy.BucketPolicyReadOnly)
	if err != nil {
		log.Fatal("Error:", err)
	}
	// should return policy `readonly`.
	policyAccess, err = c.GetBucketPolicy(bucketName, "")
	if err != nil {
		log.Fatal("Error:", err)
	}
	if policyAccess != "readonly" {
		log.Fatalf("Expected bucket policy to be readonly")
	}

	// Make the bucket 'public writeonly'.
	err = c.SetBucketPolicy(bucketName, "", policy.BucketPolicyWriteOnly)
	if err != nil {
		log.Fatal("Error:", err)
	}
	// should return policy `writeonly`.
	policyAccess, err = c.GetBucketPolicy(bucketName, "")
	if err != nil {
		log.Fatal("Error:", err)
	}
	if policyAccess != "writeonly" {
		log.Fatalf("Expected bucket policy to be writeonly")
	}
	// Make the bucket 'public read/write'.
	err = c.SetBucketPolicy(bucketName, "", policy.BucketPolicyReadWrite)
	if err != nil {
		log.Fatal("Error:", err)
	}
	// should return policy `readwrite`.
	policyAccess, err = c.GetBucketPolicy(bucketName, "")
	if err != nil {
		log.Fatal("Error:", err)
	}
	if policyAccess != "readwrite" {
		log.Fatalf("Expected bucket policy to be readwrite")
	}
	// List all buckets.
	buckets, err := c.ListBuckets()
	if len(buckets) == 0 {
		log.Fatal("Error: list buckets cannot be empty", buckets)
	}
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Verify if previously created bucket is listed in list buckets.
	bucketFound := false
	for _, bucket := range buckets {
		if bucket.Name == bucketName {
			bucketFound = true
		}
	}

	// If bucket not found error out.
	if !bucketFound {
		log.Fatal("Error: bucket ", bucketName, "not found")
	}

	objectName := bucketName + "unique"

	// Generate data
	buf := bytes.Repeat([]byte("f"), 1<<19)

	n, err := c.PutObject(bucketName, objectName, bytes.NewReader(buf), "")
	if err != nil {
		log.Fatal("Error: ", err)
	}
	if n != int64(len(buf)) {
		log.Fatal("Error: bad length ", n, len(buf))
	}

	n, err = c.PutObject(bucketName, objectName+"-nolength", bytes.NewReader(buf), "binary/octet-stream")
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName+"-nolength")
	}

	if n != int64(len(buf)) {
		log.Fatalf("Error: number of bytes does not match, want %v, got %v\n", len(buf), n)
	}

	// Instantiate a done channel to close all listing.
	doneCh := make(chan struct{})
	defer close(doneCh)

	objFound := false
	isRecursive := true // Recursive is true.
	for obj := range c.ListObjects(bucketName, objectName, isRecursive, doneCh) {
		if obj.Key == objectName {
			objFound = true
			break
		}
	}
	if !objFound {
		log.Fatal("Error: object " + objectName + " not found.")
	}

	objFound = false
	isRecursive = true // Recursive is true.
	for obj := range c.ListObjectsV2(bucketName, objectName, isRecursive, doneCh) {
		if obj.Key == objectName {
			objFound = true
			break
		}
	}
	if !objFound {
		log.Fatal("Error: object " + objectName + " not found.")
	}

	incompObjNotFound := true
	for objIncompl := range c.ListIncompleteUploads(bucketName, objectName, isRecursive, doneCh) {
		if objIncompl.Key != "" {
			incompObjNotFound = false
			break
		}
	}
	if !incompObjNotFound {
		log.Fatal("Error: unexpected dangling incomplete upload found.")
	}

	newReader, err := c.GetObject(bucketName, objectName)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	newReadBytes, err := ioutil.ReadAll(newReader)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	if !bytes.Equal(newReadBytes, buf) {
		log.Fatal("Error: bytes mismatch.")
	}

	err = c.FGetObject(bucketName, objectName, fileName+"-f")
	if err != nil {
		log.Fatal("Error: ", err)
	}

	// Generate presigned GET object url.
	presignedGetURL, err := c.PresignedGetObject(bucketName, objectName, 3600*time.Second, nil)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	// Verify if presigned url works.
	resp, err := http.Get(presignedGetURL.String())
	if err != nil {
		log.Fatal("Error: ", err)
	}
	if resp.StatusCode != http.StatusOK {
		log.Fatal("Error: ", resp.Status)
	}
	newPresignedBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	if !bytes.Equal(newPresignedBytes, buf) {
		log.Fatal("Error: bytes mismatch.")
	}

	// Set request parameters.
	reqParams := make(url.Values)
	reqParams.Set("response-content-disposition", "attachment; filename=\"test.txt\"")
	presignedGetURL, err = c.PresignedGetObject(bucketName, objectName, 3600*time.Second, reqParams)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	// Verify if presigned url works.
	resp, err = http.Get(presignedGetURL.String())
	if err != nil {
		log.Fatal("Error: ", err)
	}
	if resp.StatusCode != http.StatusOK {
		log.Fatal("Error: ", resp.Status)
	}
	newPresignedBytes, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	if !bytes.Equal(newPresignedBytes, buf) {
		log.Fatal("Error: bytes mismatch for presigned GET URL.")
	}
	if resp.Header.Get("Content-Disposition") != "attachment; filename=\"test.txt\"" {
		log.Fatalf("Error: wrong Content-Disposition received %s", resp.Header.Get("Content-Disposition"))
	}

	presignedPutURL, err := c.PresignedPutObject(bucketName, objectName+"-presigned", 3600*time.Second)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	buf = bytes.Repeat([]byte("g"), 1<<19)

	req, err := http.NewRequest("PUT", presignedPutURL.String(), bytes.NewReader(buf))
	if err != nil {
		log.Fatal("Error: ", err)
	}
	httpClient := &http.Client{
		// Setting a sensible time out of 30secs to wait for response
		// headers. Request is pro-actively cancelled after 30secs
		// with no response.
		Timeout:   30 * time.Second,
		Transport: http.DefaultTransport,
	}
	resp, err = httpClient.Do(req)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	newReader, err = c.GetObject(bucketName, objectName+"-presigned")
	if err != nil {
		log.Fatal("Error: ", err)
	}

	newReadBytes, err = ioutil.ReadAll(newReader)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	if !bytes.Equal(newReadBytes, buf) {
		log.Fatal("Error: bytes mismatch.")
	}

	err = c.RemoveObject(bucketName, objectName)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	err = c.RemoveObject(bucketName, objectName+"-f")
	if err != nil {
		log.Fatal("Error: ", err)
	}
	err = c.RemoveObject(bucketName, objectName+"-nolength")
	if err != nil {
		log.Fatal("Error: ", err)
	}
	err = c.RemoveObject(bucketName, objectName+"-presigned")
	if err != nil {
		log.Fatal("Error: ", err)
	}
	err = c.RemoveBucket(bucketName)
	if err != nil {
		log.Fatal("Error:", err)
	}
	err = c.RemoveBucket(bucketName)
	if err == nil {
		log.Fatal("Error:")
	}
	if err.Error() != "The specified bucket does not exist" {
		log.Fatal("Error: ", err)
	}
	if err = os.Remove(fileName); err != nil {
		log.Fatal("Error: ", err)
	}
	if err = os.Remove(fileName + "-f"); err != nil {
		log.Fatal("Error: ", err)
	}
}

// Test for validating GetObject Reader* methods functioning when the
// object is modified in the object store.
func testGetObjectObjectModified() {
	logTrace()

	// Instantiate new minio client object.
	c, err := minio.NewV4(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Enable tracing, write to stderr.
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("Minio-go-FunctionalTest", "0.1.0")

	// Make a new bucket.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")
	err = c.MakeBucket(bucketName, "us-east-1")
	if err != nil {
		log.Fatal("Error:", err, bucketName)
	}
	defer c.RemoveBucket(bucketName)

	// Upload an object.
	objectName := "myobject"
	content := "helloworld"
	_, err = c.PutObject(bucketName, objectName, strings.NewReader(content), "application/text")
	if err != nil {
		log.Fatalf("Failed to upload %s/%s: %v", bucketName, objectName, err)
	}

	defer c.RemoveObject(bucketName, objectName)

	reader, err := c.GetObject(bucketName, objectName)
	if err != nil {
		log.Fatalf("Failed to get object %s/%s: %v", bucketName, objectName, err)
	}
	defer reader.Close()

	// Read a few bytes of the object.
	b := make([]byte, 5)
	n, err := reader.ReadAt(b, 0)
	if err != nil {
		log.Fatalf("Failed to read object %s/%s at an offset: %v", bucketName, objectName, err)
	}

	// Upload different contents to the same object while object is being read.
	newContent := "goodbyeworld"
	_, err = c.PutObject(bucketName, objectName, strings.NewReader(newContent), "application/text")
	if err != nil {
		log.Fatalf("Failed to upload %s/%s: %v", bucketName, objectName, err)
	}

	// Confirm that a Stat() call in between doesn't change the Object's cached etag.
	_, err = reader.Stat()
	if err.Error() != "At least one of the pre-conditions you specified did not hold" {
		log.Error(fmt.Errorf("Expected Stat to fail with error %s but received %s", "At least one of the pre-conditions you specified did not hold", err.Error()))
	}

	// Read again only to find object contents have been modified since last read.
	_, err = reader.ReadAt(b, int64(n))
	if err.Error() != "At least one of the pre-conditions you specified did not hold" {
		log.Error(fmt.Errorf("Expected ReadAt to fail with error %s but received %s", "At least one of the pre-conditions you specified did not hold", err.Error()))
	}
}

// Test validates putObject to upload a file seeked at a given offset.
func testPutObjectUploadSeekedObject() {
	logTrace()

	// Instantiate new minio client object.
	c, err := minio.NewV4(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Enable tracing, write to stderr.
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("Minio-go-FunctionalTest", "0.1.0")

	// Make a new bucket.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")
	err = c.MakeBucket(bucketName, "us-east-1")
	if err != nil {
		log.Fatal("Error:", err, bucketName)
	}
	defer c.RemoveBucket(bucketName)

	tempfile, err := ioutil.TempFile("", "minio-go-upload-test-")
	if err != nil {
		log.Fatal("Error:", err)
	}

	var data []byte
	if fileName := getFilePath("datafile-100-kB"); fileName != "" {
		data, _ = ioutil.ReadFile(fileName)
	} else {
		// Generate data more than 32K
		data = bytes.Repeat([]byte("1"), 120000)
	}
	var length = len(data)
	if _, err = tempfile.Write(data); err != nil {
		log.Fatal("Error:", err)
	}

	objectName := fmt.Sprintf("test-file-%v", rand.Uint32())

	offset := length / 2
	if _, err := tempfile.Seek(int64(offset), 0); err != nil {
		log.Fatal("Error:", err)
	}

	n, err := c.PutObject(bucketName, objectName, tempfile, "binary/octet-stream")
	if err != nil {
		log.Fatal("Error:", err)
	}
	if n != int64(length-offset) {
		log.Fatalf("Invalid length returned, want %v, got %v", int64(length-offset), n)
	}
	tempfile.Close()
	if err = os.Remove(tempfile.Name()); err != nil {
		log.Fatal("Error:", err)
	}

	length = int(n)

	obj, err := c.GetObject(bucketName, objectName)
	if err != nil {
		log.Fatal("Error:", err)
	}

	n, err = obj.Seek(int64(offset), 0)
	if err != nil {
		log.Fatal("Error:", err)
	}
	if n != int64(offset) {
		log.Fatalf("Invalid offset returned, want %v, got %v", int64(offset), n)
	}

	n, err = c.PutObject(bucketName, objectName+"getobject", obj, "binary/octet-stream")
	if err != nil {
		log.Fatal("Error:", err)
	}
	if n != int64(length-offset) {
		log.Fatalf("Invalid length returned, want %v, got %v", int64(length-offset), n)
	}

	if err = c.RemoveObject(bucketName, objectName); err != nil {
		log.Fatal("Error:", err)
	}

	if err = c.RemoveObject(bucketName, objectName+"getobject"); err != nil {
		log.Fatal("Error:", err)
	}
}

// Tests bucket re-create errors.
func testMakeBucketErrorV2() {
	logTrace()

	if os.Getenv(serverEndpoint) != "s3.amazonaws.com" {
		log.Info("skipping region functional tests for non s3 runs")
		return
	}

	// Seed random based on current time.
	rand.Seed(time.Now().Unix())

	// Instantiate new minio client object.
	c, err := minio.NewV2(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Enable tracing, write to stderr.
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("Minio-go-FunctionalTest", "0.1.0")

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket in 'eu-west-1'.
	if err = c.MakeBucket(bucketName, "eu-west-1"); err != nil {
		log.Fatal("Error:", err, bucketName)
	}
	if err = c.MakeBucket(bucketName, "eu-west-1"); err == nil {
		log.Fatal("Error: make bucket should should fail for", bucketName)
	}
	// Verify valid error response from server.
	if minio.ToErrorResponse(err).Code != "BucketAlreadyExists" &&
		minio.ToErrorResponse(err).Code != "BucketAlreadyOwnedByYou" {
		log.Fatal("Error: Invalid error returned by server", err)
	}
	if err = c.RemoveBucket(bucketName); err != nil {
		log.Fatal("Error:", err, bucketName)
	}
}

// Test get object reader to not throw error on being closed twice.
func testGetObjectClosedTwiceV2() {
	logTrace()

	// Seed random based on current time.
	rand.Seed(time.Now().Unix())

	// Instantiate new minio client object.
	c, err := minio.NewV2(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Enable tracing, write to stderr.
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("Minio-go-FunctionalTest", "0.1.0")

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket.
	err = c.MakeBucket(bucketName, "us-east-1")
	if err != nil {
		log.Fatal("Error:", err, bucketName)
	}

	// Generate data more than 32K.
	var buf = getDataBuffer("datafile-33-kB", rand.Intn(1<<20)+32*1024)

	// Save the data
	objectName := randString(60, rand.NewSource(time.Now().UnixNano()), "")
	n, err := c.PutObject(bucketName, objectName, bytes.NewReader(buf), "binary/octet-stream")
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}

	if n != int64(len(buf)) {
		log.Fatalf("Error: number of bytes does not match, want %v, got %v\n", len(buf), n)
	}

	// Read the data back
	r, err := c.GetObject(bucketName, objectName)
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}

	st, err := r.Stat()
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}
	if st.Size != int64(len(buf)) {
		log.Fatalf("Error: number of bytes in stat does not match, want %v, got %v\n",
			len(buf), st.Size)
	}
	if err := r.Close(); err != nil {
		log.Fatal("Error:", err)
	}
	if err := r.Close(); err == nil {
		log.Fatal("Error: object is already closed, should return error")
	}

	err = c.RemoveObject(bucketName, objectName)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	err = c.RemoveBucket(bucketName)
	if err != nil {
		log.Fatal("Error:", err)
	}
}

// Tests removing partially uploaded objects.
func testRemovePartiallyUploadedV2() {
	logTrace()

	// Seed random based on current time.
	rand.Seed(time.Now().Unix())

	// Instantiate new minio client object.
	c, err := minio.NewV2(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Set user agent.
	c.SetAppInfo("Minio-go-FunctionalTest", "0.1.0")

	// Enable tracing, write to stdout.
	// c.TraceOn(os.Stderr)

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// make a new bucket.
	err = c.MakeBucket(bucketName, "us-east-1")
	if err != nil {
		log.Fatal("Error:", err, bucketName)
	}

	r := bytes.NewReader(bytes.Repeat([]byte("a"), 128*1024))

	reader, writer := io.Pipe()
	go func() {
		i := 0
		for i < 25 {
			_, cerr := io.CopyN(writer, r, 128*1024)
			if cerr != nil {
				log.Fatal("Error:", cerr, bucketName)
			}
			i++
			r.Seek(0, 0)
		}
		writer.CloseWithError(errors.New("proactively closed to be verified later"))
	}()

	objectName := bucketName + "-resumable"
	_, err = c.PutObject(bucketName, objectName, reader, "application/octet-stream")
	if err == nil {
		log.Fatal("Error: PutObject should fail.")
	}
	if err.Error() != "proactively closed to be verified later" {
		log.Fatal("Error:", err)
	}
	err = c.RemoveIncompleteUpload(bucketName, objectName)
	if err != nil {
		log.Fatal("Error:", err)
	}
	err = c.RemoveBucket(bucketName)
	if err != nil {
		log.Fatal("Error:", err)
	}
}

// Tests FPutObject hidden contentType setting
func testFPutObjectV2() {
	logTrace()

	// Seed random based on current time.
	rand.Seed(time.Now().Unix())

	// Instantiate new minio client object.
	c, err := minio.NewV2(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Enable tracing, write to stderr.
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("Minio-go-FunctionalTest", "0.1.0")

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket.
	err = c.MakeBucket(bucketName, "us-east-1")
	if err != nil {
		log.Fatal("Error:", err, bucketName)
	}

	// Make a temp file with 11*1024*1024 bytes of data.
	file, err := ioutil.TempFile(os.TempDir(), "FPutObjectTest")
	if err != nil {
		log.Fatal("Error:", err)
	}

	r := bytes.NewReader(bytes.Repeat([]byte("b"), 11*1024*1024))
	n, err := io.CopyN(file, r, 11*1024*1024)
	if err != nil {
		log.Fatal("Error:", err)
	}
	if n != int64(11*1024*1024) {
		log.Fatalf("Error: number of bytes does not match, want %v, got %v\n", 11*1024*1024, n)
	}

	// Close the file pro-actively for windows.
	err = file.Close()
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Set base object name
	objectName := bucketName + "FPutObject"

	// Perform standard FPutObject with contentType provided (Expecting application/octet-stream)
	n, err = c.FPutObject(bucketName, objectName+"-standard", file.Name(), "application/octet-stream")
	if err != nil {
		log.Fatal("Error:", err)
	}
	if n != int64(11*1024*1024) {
		log.Fatalf("Error: number of bytes does not match, want %v, got %v\n", 11*1024*1024, n)
	}

	// Perform FPutObject with no contentType provided (Expecting application/octet-stream)
	n, err = c.FPutObject(bucketName, objectName+"-Octet", file.Name(), "")
	if err != nil {
		log.Fatal("Error:", err)
	}
	if n != int64(11*1024*1024) {
		log.Fatalf("Error: number of bytes does not match, want %v, got %v\n", 11*1024*1024, n)
	}

	// Add extension to temp file name
	fileName := file.Name()
	err = os.Rename(file.Name(), fileName+".gtar")
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Perform FPutObject with no contentType provided (Expecting application/x-gtar)
	n, err = c.FPutObject(bucketName, objectName+"-GTar", fileName+".gtar", "")
	if err != nil {
		log.Fatal("Error:", err)
	}
	if n != int64(11*1024*1024) {
		log.Fatalf("Error: number of bytes does not match, want %v, got %v\n", 11*1024*1024, n)
	}

	// Check headers
	rStandard, err := c.StatObject(bucketName, objectName+"-standard")
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName+"-standard")
	}
	if rStandard.ContentType != "application/octet-stream" {
		log.Fatalf("Error: Content-Type headers mismatched, want %v, got %v\n",
			"application/octet-stream", rStandard.ContentType)
	}

	rOctet, err := c.StatObject(bucketName, objectName+"-Octet")
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName+"-Octet")
	}
	if rOctet.ContentType != "application/octet-stream" {
		log.Fatalf("Error: Content-Type headers mismatched, want %v, got %v\n",
			"application/octet-stream", rStandard.ContentType)
	}

	rGTar, err := c.StatObject(bucketName, objectName+"-GTar")
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName+"-GTar")
	}
	if rGTar.ContentType != "application/x-gtar" {
		log.Fatalf("Error: Content-Type headers mismatched, want %v, got %v\n",
			"application/x-gtar", rStandard.ContentType)
	}

	// Remove all objects and bucket and temp file
	err = c.RemoveObject(bucketName, objectName+"-standard")
	if err != nil {
		log.Fatal("Error: ", err)
	}

	err = c.RemoveObject(bucketName, objectName+"-Octet")
	if err != nil {
		log.Fatal("Error: ", err)
	}

	err = c.RemoveObject(bucketName, objectName+"-GTar")
	if err != nil {
		log.Fatal("Error: ", err)
	}

	err = c.RemoveBucket(bucketName)
	if err != nil {
		log.Fatal("Error:", err)
	}

	err = os.Remove(fileName + ".gtar")
	if err != nil {
		log.Fatal("Error:", err)
	}

}

// Tests various bucket supported formats.
func testMakeBucketRegionsV2() {

	if os.Getenv(serverEndpoint) != "s3.amazonaws.com" {
		log.Info("skipping region functional tests for non s3 runs")
		return
	}

	// Seed random based on current time.
	rand.Seed(time.Now().Unix())

	// Instantiate new minio client object.
	c, err := minio.NewV2(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Enable tracing, write to stderr.
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("Minio-go-FunctionalTest", "0.1.0")

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket in 'eu-central-1'.
	if err = c.MakeBucket(bucketName, "eu-west-1"); err != nil {
		log.Fatal("Error:", err, bucketName)
	}

	if err = c.RemoveBucket(bucketName); err != nil {
		log.Fatal("Error:", err, bucketName)
	}

	// Make a new bucket with '.' in its name, in 'us-west-2'. This
	// request is internally staged into a path style instead of
	// virtual host style.
	if err = c.MakeBucket(bucketName+".withperiod", "us-west-2"); err != nil {
		log.Fatal("Error:", err, bucketName+".withperiod")
	}

	// Remove the newly created bucket.
	if err = c.RemoveBucket(bucketName + ".withperiod"); err != nil {
		log.Fatal("Error:", err, bucketName+".withperiod")
	}
}

// Tests get object ReaderSeeker interface methods.
func testGetObjectReadSeekFunctionalV2() {
	logTrace()
	// Seed random based on current time.
	rand.Seed(time.Now().Unix())

	// Instantiate new minio client object.
	c, err := minio.NewV2(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Enable tracing, write to stderr.
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("Minio-go-FunctionalTest", "0.1.0")

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket.
	err = c.MakeBucket(bucketName, "us-east-1")
	if err != nil {
		log.Fatal("Error:", err, bucketName)
	}

	// Generate data more than 32K.
	var buf = getDataBuffer("datafile-33-kB", rand.Intn(1<<20)+32*1024)

	// Save the data.
	objectName := randString(60, rand.NewSource(time.Now().UnixNano()), "")
	n, err := c.PutObject(bucketName, objectName, bytes.NewReader(buf), "binary/octet-stream")
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}

	if n != int64(len(buf)) {
		log.Fatalf("Error: number of bytes does not match, want %v, got %v\n", len(buf), n)
	}

	// Read the data back
	r, err := c.GetObject(bucketName, objectName)
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}

	st, err := r.Stat()
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}
	if st.Size != int64(len(buf)) {
		log.Fatalf("Error: number of bytes in stat does not match, want %v, got %v\n",
			len(buf), st.Size)
	}

	offset := int64(2048)
	n, err = r.Seek(offset, 0)
	if err != nil {
		log.Fatal("Error:", err, offset)
	}
	if n != offset {
		log.Fatalf("Error: number of bytes seeked does not match, want %v, got %v\n",
			offset, n)
	}
	n, err = r.Seek(0, 1)
	if err != nil {
		log.Fatal("Error:", err)
	}
	if n != offset {
		log.Fatalf("Error: number of current seek does not match, want %v, got %v\n",
			offset, n)
	}
	_, err = r.Seek(offset, 2)
	if err == nil {
		log.Fatal("Error: seek on positive offset for whence '2' should error out")
	}
	n, err = r.Seek(-offset, 2)
	if err != nil {
		log.Fatal("Error:", err)
	}
	if n != st.Size-offset {
		log.Fatalf("Error: number of bytes seeked back does not match, want %d, got %v\n", st.Size-offset, n)
	}

	var buffer1 bytes.Buffer
	if _, err = io.CopyN(&buffer1, r, st.Size); err != nil {
		if err != io.EOF {
			log.Fatal("Error:", err)
		}
	}
	if !bytes.Equal(buf[len(buf)-int(offset):], buffer1.Bytes()) {
		log.Fatal("Error: Incorrect read bytes v/s original buffer.")
	}

	// Seek again and read again.
	n, err = r.Seek(offset-1, 0)
	if err != nil {
		log.Fatal("Error:", err)
	}
	if n != (offset - 1) {
		log.Fatalf("Error: number of bytes seeked back does not match, want %v, got %v\n", offset-1, n)
	}

	var buffer2 bytes.Buffer
	if _, err = io.CopyN(&buffer2, r, st.Size); err != nil {
		if err != io.EOF {
			log.Fatal("Error:", err)
		}
	}
	// Verify now lesser bytes.
	if !bytes.Equal(buf[2047:], buffer2.Bytes()) {
		log.Fatal("Error: Incorrect read bytes v/s original buffer.")
	}

	err = c.RemoveObject(bucketName, objectName)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	err = c.RemoveBucket(bucketName)
	if err != nil {
		log.Fatal("Error:", err)
	}
}

// Tests get object ReaderAt interface methods.
func testGetObjectReadAtFunctionalV2() {
	logTrace()
	// Seed random based on current time.
	rand.Seed(time.Now().Unix())

	// Instantiate new minio client object.
	c, err := minio.NewV2(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Enable tracing, write to stderr.
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("Minio-go-FunctionalTest", "0.1.0")

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket.
	err = c.MakeBucket(bucketName, "us-east-1")
	if err != nil {
		log.Fatal("Error:", err, bucketName)
	}

	// Generate data more than 32K
	var buf = getDataBuffer("datafile-33-kB", rand.Intn(1<<20)+32*1024)

	// Save the data
	objectName := randString(60, rand.NewSource(time.Now().UnixNano()), "")
	n, err := c.PutObject(bucketName, objectName, bytes.NewReader(buf), "binary/octet-stream")
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}

	if n != int64(len(buf)) {
		log.Fatalf("Error: number of bytes does not match, want %v, got %v\n", len(buf), n)
	}

	// Read the data back
	r, err := c.GetObject(bucketName, objectName)
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}

	st, err := r.Stat()
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}
	if st.Size != int64(len(buf)) {
		log.Fatalf("Error: number of bytes in stat does not match, want %v, got %v\n",
			len(buf), st.Size)
	}

	offset := int64(2048)

	// Read directly
	buf2 := make([]byte, 512)
	buf3 := make([]byte, 512)
	buf4 := make([]byte, 512)

	m, err := r.ReadAt(buf2, offset)
	if err != nil {
		log.Fatal("Error:", err, st.Size, len(buf2), offset)
	}
	if m != len(buf2) {
		log.Fatalf("Error: ReadAt read shorter bytes before reaching EOF, want %v, got %v\n", m, len(buf2))
	}
	if !bytes.Equal(buf2, buf[offset:offset+512]) {
		log.Fatal("Error: Incorrect read between two ReadAt from same offset.")
	}
	offset += 512
	m, err = r.ReadAt(buf3, offset)
	if err != nil {
		log.Fatal("Error:", err, st.Size, len(buf3), offset)
	}
	if m != len(buf3) {
		log.Fatalf("Error: ReadAt read shorter bytes before reaching EOF, want %v, got %v\n", m, len(buf3))
	}
	if !bytes.Equal(buf3, buf[offset:offset+512]) {
		log.Fatal("Error: Incorrect read between two ReadAt from same offset.")
	}
	offset += 512
	m, err = r.ReadAt(buf4, offset)
	if err != nil {
		log.Fatal("Error:", err, st.Size, len(buf4), offset)
	}
	if m != len(buf4) {
		log.Fatalf("Error: ReadAt read shorter bytes before reaching EOF, want %v, got %v\n", m, len(buf4))
	}
	if !bytes.Equal(buf4, buf[offset:offset+512]) {
		log.Fatal("Error: Incorrect read between two ReadAt from same offset.")
	}

	buf5 := make([]byte, n)
	// Read the whole object.
	m, err = r.ReadAt(buf5, 0)
	if err != nil {
		if err != io.EOF {
			log.Fatal("Error:", err, len(buf5))
		}
	}
	if m != len(buf5) {
		log.Fatalf("Error: ReadAt read shorter bytes before reaching EOF, want %v, got %v\n", m, len(buf5))
	}
	if !bytes.Equal(buf, buf5) {
		log.Fatal("Error: Incorrect data read in GetObject, than what was previously upoaded.")
	}

	buf6 := make([]byte, n+1)
	// Read the whole object and beyond.
	_, err = r.ReadAt(buf6, 0)
	if err != nil {
		if err != io.EOF {
			log.Fatal("Error:", err, len(buf6))
		}
	}
	err = c.RemoveObject(bucketName, objectName)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	err = c.RemoveBucket(bucketName)
	if err != nil {
		log.Fatal("Error:", err)
	}
}

// Tests copy object
func testCopyObjectV2() {
	logTrace()

	// Seed random based on current time.
	rand.Seed(time.Now().Unix())

	// Instantiate new minio client object
	c, err := minio.NewV4(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Enable tracing, write to stderr.
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("Minio-go-FunctionalTest", "0.1.0")

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket in 'us-east-1' (source bucket).
	err = c.MakeBucket(bucketName, "us-east-1")
	if err != nil {
		log.Fatal("Error:", err, bucketName)
	}

	// Make a new bucket in 'us-east-1' (destination bucket).
	err = c.MakeBucket(bucketName+"-copy", "us-east-1")
	if err != nil {
		log.Fatal("Error:", err, bucketName+"-copy")
	}

	// Generate data more than 32K
	var buf = getDataBuffer("datafile-33-kB", rand.Intn(1<<20)+32*1024)

	// Save the data
	objectName := randString(60, rand.NewSource(time.Now().UnixNano()), "")
	n, err := c.PutObject(bucketName, objectName, bytes.NewReader(buf), "binary/octet-stream")
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName)
	}

	if n != int64(len(buf)) {
		log.Fatalf("Error: number of bytes does not match want %v, got %v",
			len(buf), n)
	}

	r, err := c.GetObject(bucketName, objectName)
	if err != nil {
		log.Fatal("Error:", err)
	}
	// Check the various fields of source object against destination object.
	objInfo, err := r.Stat()
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Copy Source
	src := minio.NewSourceInfo(bucketName, objectName, nil)

	// Set copy conditions.

	// All invalid conditions first.
	err = src.SetModifiedSinceCond(time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC))
	if err == nil {
		log.Fatal("Error:", err)
	}
	err = src.SetUnmodifiedSinceCond(time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC))
	if err == nil {
		log.Fatal("Error:", err)
	}
	err = src.SetMatchETagCond("")
	if err == nil {
		log.Fatal("Error:", err)
	}
	err = src.SetMatchETagExceptCond("")
	if err == nil {
		log.Fatal("Error:", err)
	}

	err = src.SetModifiedSinceCond(time.Date(2014, time.April, 0, 0, 0, 0, 0, time.UTC))
	if err != nil {
		log.Fatal("Error:", err)
	}
	err = src.SetMatchETagCond(objInfo.ETag)
	if err != nil {
		log.Fatal("Error:", err)
	}

	dst, err := minio.NewDestinationInfo(bucketName+"-copy", objectName+"-copy", nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Perform the Copy
	err = c.CopyObject(dst, src)
	if err != nil {
		log.Fatal("Error:", err, bucketName+"-copy", objectName+"-copy")
	}

	// Source object
	reader, err := c.GetObject(bucketName, objectName)
	if err != nil {
		log.Fatal("Error:", err)
	}
	// Destination object
	readerCopy, err := c.GetObject(bucketName+"-copy", objectName+"-copy")
	if err != nil {
		log.Fatal("Error:", err)
	}
	// Check the various fields of source object against destination object.
	objInfo, err = reader.Stat()
	if err != nil {
		log.Fatal("Error:", err)
	}
	objInfoCopy, err := readerCopy.Stat()
	if err != nil {
		log.Fatal("Error:", err)
	}
	if objInfo.Size != objInfoCopy.Size {
		log.Fatalf("Error: number of bytes does not match, want %v, got %v\n",
			objInfo.Size, objInfoCopy.Size)
	}

	// CopyObject again but with wrong conditions
	src = minio.NewSourceInfo(bucketName, objectName, nil)
	err = src.SetUnmodifiedSinceCond(time.Date(2014, time.April, 0, 0, 0, 0, 0, time.UTC))
	if err != nil {
		log.Fatal("Error:", err)
	}
	err = src.SetMatchETagExceptCond(objInfo.ETag)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Perform the Copy which should fail
	err = c.CopyObject(dst, src)
	if err == nil {
		log.Fatal("Error:", err, bucketName+"-copy", objectName+"-copy should fail")
	}

	// Remove all objects and buckets
	err = c.RemoveObject(bucketName, objectName)
	if err != nil {
		log.Fatal("Error:", err)
	}

	err = c.RemoveObject(bucketName+"-copy", objectName+"-copy")
	if err != nil {
		log.Fatal("Error:", err)
	}

	err = c.RemoveBucket(bucketName)
	if err != nil {
		log.Fatal("Error:", err)
	}

	err = c.RemoveBucket(bucketName + "-copy")
	if err != nil {
		log.Fatal("Error:", err)
	}
}

func testComposeObjectErrorCasesWrapper(c *minio.Client) {
	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket in 'us-east-1' (source bucket).
	err := c.MakeBucket(bucketName, "us-east-1")
	if err != nil {
		log.Fatal("Error:", err, bucketName)
	}

	// Test that more than 10K source objects cannot be
	// concatenated.
	srcArr := [10001]minio.SourceInfo{}
	srcSlice := srcArr[:]
	dst, err := minio.NewDestinationInfo(bucketName, "object", nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	if err := c.ComposeObject(dst, srcSlice); err == nil {
		log.Fatal("Error was expected.")
	} else if err.Error() != "There must be as least one and upto 10000 source objects." {
		log.Fatal("Got unexpected error: ", err)
	}

	// Create a source with invalid offset spec and check that
	// error is returned:
	// 1. Create the source object.
	const badSrcSize = 5 * 1024 * 1024
	buf := bytes.Repeat([]byte("1"), badSrcSize)
	_, err = c.PutObject(bucketName, "badObject", bytes.NewReader(buf), "")
	if err != nil {
		log.Fatal("Error:", err)
	}
	// 2. Set invalid range spec on the object (going beyond
	// object size)
	badSrc := minio.NewSourceInfo(bucketName, "badObject", nil)
	err = badSrc.SetRange(1, badSrcSize)
	if err != nil {
		log.Fatal("Error:", err)
	}
	// 3. ComposeObject call should fail.
	if err := c.ComposeObject(dst, []minio.SourceInfo{badSrc}); err == nil {
		log.Fatal("Error was expected.")
	} else if !strings.Contains(err.Error(), "has invalid segment-to-copy") {
		log.Fatal("Got unexpected error: ", err)
	}
}

// Test expected error cases
func testComposeObjectErrorCasesV2() {
	logTrace()

	// Instantiate new minio client object
	c, err := minio.NewV2(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	testComposeObjectErrorCasesWrapper(c)
}

func testComposeMultipleSources(c *minio.Client) {
	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")
	// Make a new bucket in 'us-east-1' (source bucket).
	err := c.MakeBucket(bucketName, "us-east-1")
	if err != nil {
		log.Fatal("Error:", err, bucketName)
	}

	// Upload a small source object
	const srcSize = 1024 * 1024 * 5
	buf := bytes.Repeat([]byte("1"), srcSize)
	_, err = c.PutObject(bucketName, "srcObject", bytes.NewReader(buf), "binary/octet-stream")
	if err != nil {
		log.Fatal("Error:", err)
	}

	// We will append 10 copies of the object.
	srcs := []minio.SourceInfo{}
	for i := 0; i < 10; i++ {
		srcs = append(srcs, minio.NewSourceInfo(bucketName, "srcObject", nil))
	}
	// make the last part very small
	err = srcs[9].SetRange(0, 0)
	if err != nil {
		log.Fatal("unexpected error:", err)
	}

	dst, err := minio.NewDestinationInfo(bucketName, "dstObject", nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	err = c.ComposeObject(dst, srcs)
	if err != nil {
		log.Fatal("Error:", err)
	}

	objProps, err := c.StatObject(bucketName, "dstObject")
	if err != nil {
		log.Fatal("Error:", err)
	}

	if objProps.Size != 9*srcSize+1 {
		log.Fatal("Size mismatched! Expected:", 10000*srcSize, "but got:", objProps.Size)
	}
}

// Test concatenating multiple objects objects
func testCompose10KSourcesV2() {
	logTrace()

	// Instantiate new minio client object
	c, err := minio.NewV2(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	testComposeMultipleSources(c)
}
func testEncryptedCopyObjectWrapper(c *minio.Client) {
	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")
	// Make a new bucket in 'us-east-1' (source bucket).
	err := c.MakeBucket(bucketName, "us-east-1")
	if err != nil {
		log.Fatal("Error:", err, bucketName)
	}

	key1 := minio.NewSSEInfo([]byte("32byteslongsecretkeymustbegiven1"), "AES256")
	key2 := minio.NewSSEInfo([]byte("32byteslongsecretkeymustbegiven2"), "AES256")

	// 1. create an sse-c encrypted object to copy by uploading
	const srcSize = 1024 * 1024
	buf := bytes.Repeat([]byte("abcde"), srcSize) // gives a buffer of 5MiB
	metadata := make(map[string][]string)
	for k, v := range key1.GetSSEHeaders() {
		metadata[k] = append(metadata[k], v)
	}
	_, err = c.PutObjectWithSize(bucketName, "srcObject", bytes.NewReader(buf), int64(len(buf)), metadata, nil)
	if err != nil {
		log.Fatal("PutObjectWithSize Error:", err)
	}

	// 2. copy object and change encryption key
	src := minio.NewSourceInfo(bucketName, "srcObject", &key1)
	dst, err := minio.NewDestinationInfo(bucketName, "dstObject", &key2, nil)
	if err != nil {
		log.Fatal("Error:", err)
	}

	err = c.CopyObject(dst, src)
	if err != nil {
		log.Fatal("CopyObject Error:", err)
	}

	// 3. get copied object and check if content is equal
	reqH := minio.NewGetReqHeaders()
	for k, v := range key2.GetSSEHeaders() {
		reqH.Set(k, v)
	}
	coreClient := minio.Core{c}
	reader, _, err := coreClient.GetObject(bucketName, "dstObject", reqH)
	if err != nil {
		log.Fatal("GetObject Error:", err)
	}
	defer reader.Close()

	decBytes, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Fatalln(err)
	}
	if !bytes.Equal(decBytes, buf) {
		log.Fatal("downloaded object mismatched for encrypted object")
	}
}

// Test encrypted copy object
func testEncryptedCopyObject() {
	logTrace()

	// Instantiate new minio client object
	c, err := minio.NewV4(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// c.TraceOn(os.Stderr)
	testEncryptedCopyObjectWrapper(c)
}

// Test encrypted copy object
func testEncryptedCopyObjectV2() {

	// Instantiate new minio client object
	c, err := minio.NewV2(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	testEncryptedCopyObjectWrapper(c)
}
func testUserMetadataCopying() {
	logTrace()

	// Instantiate new minio client object
	c, err := minio.NewV4(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// c.TraceOn(os.Stderr)
	testUserMetadataCopyingWrapper(c)
}
func testUserMetadataCopyingWrapper(c *minio.Client) {
	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")
	// Make a new bucket in 'us-east-1' (source bucket).
	err := c.MakeBucket(bucketName, "us-east-1")
	if err != nil {
		log.Fatal("Error:", err, bucketName)
	}

	fetchMeta := func(object string) (h http.Header) {
		objInfo, err := c.StatObject(bucketName, object)
		if err != nil {
			log.Fatal("Metadata fetch error:", err)
		}
		h = make(http.Header)
		for k, vs := range objInfo.Metadata {
			if strings.HasPrefix(strings.ToLower(k), "x-amz-meta-") {
				for _, v := range vs {
					h.Add(k, v)
				}
			}
		}
		return h
	}

	// 1. create a client encrypted object to copy by uploading
	const srcSize = 1024 * 1024
	buf := bytes.Repeat([]byte("abcde"), srcSize) // gives a buffer of 5MiB
	metadata := make(http.Header)
	metadata.Set("x-amz-meta-myheader", "myvalue")
	_, err = c.PutObjectWithMetadata(bucketName, "srcObject",
		bytes.NewReader(buf), metadata, nil)
	if err != nil {
		log.Fatal("Put Error:", err)
	}
	if !reflect.DeepEqual(metadata, fetchMeta("srcObject")) {
		log.Fatal("Unequal metadata")
	}

	// 2. create source
	src := minio.NewSourceInfo(bucketName, "srcObject", nil)
	// 2.1 create destination with metadata set
	dst1, err := minio.NewDestinationInfo(bucketName, "dstObject-1", nil, map[string]string{"notmyheader": "notmyvalue"})
	if err != nil {
		log.Fatal("Error:", err)
	}

	// 3. Check that copying to an object with metadata set resets
	// the headers on the copy.
	err = c.CopyObject(dst1, src)
	if err != nil {
		log.Fatal("Error:", err)
	}

	expectedHeaders := make(http.Header)
	expectedHeaders.Set("x-amz-meta-notmyheader", "notmyvalue")
	if !reflect.DeepEqual(expectedHeaders, fetchMeta("dstObject-1")) {
		log.Fatal("Unequal metadata")
	}

	// 4. create destination with no metadata set and same source
	dst2, err := minio.NewDestinationInfo(bucketName, "dstObject-2", nil, nil)
	if err != nil {
		log.Fatal("Error:", err)

	}
	src = minio.NewSourceInfo(bucketName, "srcObject", nil)

	// 5. Check that copying to an object with no metadata set,
	// copies metadata.
	err = c.CopyObject(dst2, src)
	if err != nil {
		log.Fatal("Error:", err)
	}

	expectedHeaders = metadata
	if !reflect.DeepEqual(expectedHeaders, fetchMeta("dstObject-2")) {
		log.Fatal("Unequal metadata")
	}

	// 6. Compose a pair of sources.
	srcs := []minio.SourceInfo{
		minio.NewSourceInfo(bucketName, "srcObject", nil),
		minio.NewSourceInfo(bucketName, "srcObject", nil),
	}
	dst3, err := minio.NewDestinationInfo(bucketName, "dstObject-3", nil, nil)
	if err != nil {
		log.Fatal("Error:", err)

	}

	err = c.ComposeObject(dst3, srcs)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Check that no headers are copied in this case
	if !reflect.DeepEqual(make(http.Header), fetchMeta("dstObject-3")) {
		log.Fatal("Unequal metadata")
	}

	// 7. Compose a pair of sources with dest user metadata set.
	srcs = []minio.SourceInfo{
		minio.NewSourceInfo(bucketName, "srcObject", nil),
		minio.NewSourceInfo(bucketName, "srcObject", nil),
	}
	dst4, err := minio.NewDestinationInfo(bucketName, "dstObject-4", nil, map[string]string{"notmyheader": "notmyvalue"})
	if err != nil {
		log.Fatal("Error:", err)

	}

	err = c.ComposeObject(dst4, srcs)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Check that no headers are copied in this case
	expectedHeaders = make(http.Header)
	expectedHeaders.Set("x-amz-meta-notmyheader", "notmyvalue")
	if !reflect.DeepEqual(expectedHeaders, fetchMeta("dstObject-4")) {
		log.Fatal("Unequal metadata")
	}
}

func testUserMetadataCopyingV2() {
	logTrace()

	// Instantiate new minio client object
	c, err := minio.NewV2(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// c.TraceOn(os.Stderr)
	testUserMetadataCopyingWrapper(c)
}

// Test expected error cases
func testComposeObjectErrorCases() {
	logTrace()

	// Instantiate new minio client object
	c, err := minio.NewV4(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	testComposeObjectErrorCasesWrapper(c)
}

// Test concatenating 10K objects
func testCompose10KSources() {
	logTrace()

	// Instantiate new minio client object
	c, err := minio.NewV4(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	testComposeMultipleSources(c)
}

// Tests comprehensive list of all methods.
func testFunctionalV2() {
	logTrace()
	// Seed random based on current time.
	rand.Seed(time.Now().Unix())

	c, err := minio.NewV2(
		os.Getenv(serverEndpoint),
		os.Getenv(accessKey),
		os.Getenv(secretKey),
		mustParseBool(os.Getenv(enableHTTPS)),
	)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Enable to debug
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("Minio-go-FunctionalTest", "0.1.0")

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket.
	err = c.MakeBucket(bucketName, "us-east-1")
	if err != nil {
		log.Fatal("Error:", err, bucketName)
	}

	// Generate a random file name.
	fileName := randString(60, rand.NewSource(time.Now().UnixNano()), "")
	file, err := os.Create(fileName)
	if err != nil {
		log.Fatal("Error:", err)
	}
	for i := 0; i < 3; i++ {
		buf := make([]byte, rand.Intn(1<<19))
		_, err = file.Write(buf)
		if err != nil {
			log.Fatal("Error:", err)
		}
	}
	file.Close()

	// Verify if bucket exits and you have access.
	var exists bool
	exists, err = c.BucketExists(bucketName)
	if err != nil {
		log.Fatal("Error:", err, bucketName)
	}
	if !exists {
		log.Fatal("Error: could not find ", bucketName)
	}

	// Make the bucket 'public read/write'.
	err = c.SetBucketPolicy(bucketName, "", policy.BucketPolicyReadWrite)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// List all buckets.
	buckets, err := c.ListBuckets()
	if len(buckets) == 0 {
		log.Fatal("Error: list buckets cannot be empty", buckets)
	}
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Verify if previously created bucket is listed in list buckets.
	bucketFound := false
	for _, bucket := range buckets {
		if bucket.Name == bucketName {
			bucketFound = true
		}
	}

	// If bucket not found error out.
	if !bucketFound {
		log.Fatal("Error: bucket ", bucketName, "not found")
	}

	objectName := bucketName + "unique"

	// Generate data
	buf := bytes.Repeat([]byte("n"), rand.Intn(1<<19))

	n, err := c.PutObject(bucketName, objectName, bytes.NewReader(buf), "")
	if err != nil {
		log.Fatal("Error: ", err)
	}
	if n != int64(len(buf)) {
		log.Fatal("Error: bad length ", n, len(buf))
	}

	n, err = c.PutObject(bucketName, objectName+"-nolength", bytes.NewReader(buf), "binary/octet-stream")
	if err != nil {
		log.Fatal("Error:", err, bucketName, objectName+"-nolength")
	}

	if n != int64(len(buf)) {
		log.Fatalf("Error: number of bytes does not match, want %v, got %v\n", len(buf), n)
	}

	// Instantiate a done channel to close all listing.
	doneCh := make(chan struct{})
	defer close(doneCh)

	objFound := false
	isRecursive := true // Recursive is true.
	for obj := range c.ListObjects(bucketName, objectName, isRecursive, doneCh) {
		if obj.Key == objectName {
			objFound = true
			break
		}
	}
	if !objFound {
		log.Fatal("Error: object " + objectName + " not found.")
	}

	objFound = false
	isRecursive = true // Recursive is true.
	for obj := range c.ListObjects(bucketName, objectName, isRecursive, doneCh) {
		if obj.Key == objectName {
			objFound = true
			break
		}
	}
	if !objFound {
		log.Fatal("Error: object " + objectName + " not found.")
	}

	incompObjNotFound := true
	for objIncompl := range c.ListIncompleteUploads(bucketName, objectName, isRecursive, doneCh) {
		if objIncompl.Key != "" {
			incompObjNotFound = false
			break
		}
	}
	if !incompObjNotFound {
		log.Fatal("Error: unexpected dangling incomplete upload found.")
	}

	newReader, err := c.GetObject(bucketName, objectName)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	newReadBytes, err := ioutil.ReadAll(newReader)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	if !bytes.Equal(newReadBytes, buf) {
		log.Fatal("Error: bytes mismatch.")
	}

	err = c.FGetObject(bucketName, objectName, fileName+"-f")
	if err != nil {
		log.Fatal("Error: ", err)
	}

	// Generate presigned GET object url.
	presignedGetURL, err := c.PresignedGetObject(bucketName, objectName, 3600*time.Second, nil)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	// Verify if presigned url works.
	resp, err := http.Get(presignedGetURL.String())
	if err != nil {
		log.Fatal("Error: ", err)
	}
	if resp.StatusCode != http.StatusOK {
		log.Fatal("Error: ", resp.Status)
	}
	newPresignedBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	if !bytes.Equal(newPresignedBytes, buf) {
		log.Fatal("Error: bytes mismatch.")
	}

	// Set request parameters.
	reqParams := make(url.Values)
	reqParams.Set("response-content-disposition", "attachment; filename=\"test.txt\"")
	// Generate presigned GET object url.
	presignedGetURL, err = c.PresignedGetObject(bucketName, objectName, 3600*time.Second, reqParams)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	// Verify if presigned url works.
	resp, err = http.Get(presignedGetURL.String())
	if err != nil {
		log.Fatal("Error: ", err)
	}
	if resp.StatusCode != http.StatusOK {
		log.Fatal("Error: ", resp.Status)
	}
	newPresignedBytes, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	if !bytes.Equal(newPresignedBytes, buf) {
		log.Fatal("Error: bytes mismatch for presigned GET url.")
	}
	// Verify content disposition.
	if resp.Header.Get("Content-Disposition") != "attachment; filename=\"test.txt\"" {
		log.Fatalf("Error: wrong Content-Disposition received %s", resp.Header.Get("Content-Disposition"))
	}

	presignedPutURL, err := c.PresignedPutObject(bucketName, objectName+"-presigned", 3600*time.Second)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	// Generate data more than 32K
	buf = bytes.Repeat([]byte("1"), rand.Intn(1<<20)+32*1024)

	req, err := http.NewRequest("PUT", presignedPutURL.String(), bytes.NewReader(buf))
	if err != nil {
		log.Fatal("Error: ", err)
	}
	httpClient := &http.Client{
		// Setting a sensible time out of 30secs to wait for response
		// headers. Request is pro-actively cancelled after 30secs
		// with no response.
		Timeout:   30 * time.Second,
		Transport: http.DefaultTransport,
	}
	resp, err = httpClient.Do(req)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	newReader, err = c.GetObject(bucketName, objectName+"-presigned")
	if err != nil {
		log.Fatal("Error: ", err)
	}

	newReadBytes, err = ioutil.ReadAll(newReader)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	if !bytes.Equal(newReadBytes, buf) {
		log.Fatal("Error: bytes mismatch.")
	}

	err = c.RemoveObject(bucketName, objectName)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	err = c.RemoveObject(bucketName, objectName+"-f")
	if err != nil {
		log.Fatal("Error: ", err)
	}
	err = c.RemoveObject(bucketName, objectName+"-nolength")
	if err != nil {
		log.Fatal("Error: ", err)
	}
	err = c.RemoveObject(bucketName, objectName+"-presigned")
	if err != nil {
		log.Fatal("Error: ", err)
	}
	err = c.RemoveBucket(bucketName)
	if err != nil {
		log.Fatal("Error:", err)
	}
	err = c.RemoveBucket(bucketName)
	if err == nil {
		log.Fatal("Error:")
	}
	if err.Error() != "The specified bucket does not exist" {
		log.Fatal("Error: ", err)
	}
	if err = os.Remove(fileName); err != nil {
		log.Fatal("Error: ", err)
	}
	if err = os.Remove(fileName + "-f"); err != nil {
		log.Fatal("Error: ", err)
	}
}

// Convert string to bool and always return false if any error
func mustParseBool(str string) bool {
	b, err := strconv.ParseBool(str)
	if err != nil {
		return false
	}
	return b
}
func logTrace() {
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	_, line := f.FileLine(pc[0])
	log.Info(fmt.Sprintf("Running %s at line:%d", f.Name(), line))
}

func main() {
	log.Info("Running functional tests for minio-go sdk....")
	if !isQuickMode() {
		testMakeBucketErrorV2()
		testGetObjectClosedTwiceV2()
		testRemovePartiallyUploadedV2()
		testFPutObjectV2()
		testMakeBucketRegionsV2()
		testGetObjectReadSeekFunctionalV2()
		testGetObjectReadAtFunctionalV2()
		testCopyObjectV2()
		testFunctionalV2()
		testComposeObjectErrorCasesV2()
		testCompose10KSourcesV2()
		testEncryptedCopyObjectV2()
		testUserMetadataCopyingV2()
		testMakeBucketError()
		testMakeBucketRegions()
		testPutObjectWithMetadata()

		testPutObjectReadAt()

		testPutObjectStreaming()
		testListPartiallyUploaded()
		testGetObjectSeekEnd()
		testGetObjectClosedTwice()
		testRemoveMultipleObjects()
		testRemovePartiallyUploaded()
		testFPutObjectMultipart()
		testFPutObject()
		testGetObjectReadSeekFunctional()
		testGetObjectReadAtFunctional()
		testPresignedPostPolicy()
		testCopyObject()
		testEncryptionPutGet()
		testComposeObjectErrorCases()
		testCompose10KSources()
		testUserMetadataCopying()
		testEncryptedCopyObject()
		testBucketNotification()
		testFunctional()
		testGetObjectObjectModified()
		testPutObjectUploadSeekedObject()
	} else {
		log.Info("Running short functional tests")
		testFunctional()
		testFunctionalV2()
	}

	log.Info("Functional tests complete for minio-go sdk")
}
