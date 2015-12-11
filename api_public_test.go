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

package minio_test

import (
	"bytes"
	"io"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/minio/minio-go"
)

func TestUserAgent(t *testing.T) {
	userAgent := userAgentHandler(userAgentHandler{})
	server := httptest.NewServer(userAgent)
	defer server.Close()

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal("Error")
	}
	a, err := minio.New(u.Host, "", "", true)
	if err != nil {
		t.Fatal("Error")
	}
	// Set app info again.
	a.SetAppInfo("hello-app", "1.0")

	// Initiate a request.
	a.MakeBucket("bucket", "private", "")

	// Set app info again, this should not have set.
	a.SetAppInfo("new-hello-app", "2.0")

	// This call should succeed, server shouldn't see
	// a new userAgent on the same connection.
	err = a.BucketExists("bucket")
	if err != nil {
		t.Fatal("Error")
	}
}

func TestBucketOperations(t *testing.T) {
	bucket := bucketHandler(bucketHandler{
		resource: "/bucket",
	})
	server := httptest.NewServer(bucket)
	defer server.Close()

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal("Error")
	}
	a, err := minio.New(u.Host, "", "", true)
	if err != nil {
		t.Fatal("Error")
	}
	err = a.MakeBucket("bucket", "private", "")
	if err != nil {
		t.Fatal("Error")
	}

	err = a.BucketExists("bucket")
	if err != nil {
		t.Fatal("Error")
	}

	err = a.BucketExists("bucket1")
	if err == nil {
		t.Fatal("Error")
	}
	if err.Error() != "Access Denied." {
		t.Fatal("Error")
	}

	err = a.SetBucketACL("bucket", "public-read-write")
	if err != nil {
		t.Fatal("Error")
	}

	acl, err := a.GetBucketACL("bucket")
	if err != nil {
		t.Fatal("Error")
	}
	if acl != minio.BucketACL("private") {
		t.Fatal("Error")
	}

	for b := range a.ListBuckets() {
		if b.Err != nil {
			t.Fatal(b.Err.Error())
		}
		if b.Name != "bucket" {
			t.Fatal("Error")
		}
	}

	for o := range a.ListObjects("bucket", "", true) {
		if o.Err != nil {
			t.Fatal(o.Err.Error())
		}
		if o.Key != "object" {
			t.Fatal("Error")
		}
	}

	err = a.RemoveBucket("bucket")
	if err != nil {
		t.Fatal("Error")
	}

	err = a.RemoveBucket("bucket1")
	if err == nil {
		t.Fatal("Error")
	}
	if err.Error() != "The specified bucket does not exist." {
		t.Fatal("Error")
	}
}

func TestBucketOperationsFail(t *testing.T) {
	bucket := bucketHandler(bucketHandler{
		resource: "/bucket",
	})
	server := httptest.NewServer(bucket)
	defer server.Close()

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal("Error")
	}
	a, err := minio.New(u.Host, "", "", true)
	if err != nil {
		t.Fatal("Error")
	}
	err = a.MakeBucket("bucket$$$", "private", "")
	if err == nil {
		t.Fatal("Error")
	}

	err = a.BucketExists("bucket.")
	if err == nil {
		t.Fatal("Error")
	}

	err = a.SetBucketACL("bucket-.", "public-read-write")
	if err == nil {
		t.Fatal("Error")
	}

	_, err = a.GetBucketACL("bucket??")
	if err == nil {
		t.Fatal("Error")
	}

	for o := range a.ListObjects("bucket??", "", true) {
		if o.Err == nil {
			t.Fatal(o.Err.Error())
		}
	}

	err = a.RemoveBucket("bucket??")
	if err == nil {
		t.Fatal("Error")
	}

	if err.Error() != "Bucket name contains invalid characters." {
		t.Fatal("Error")
	}
}

func TestObjectOperations(t *testing.T) {
	object := objectHandler(objectHandler{
		resource: "/bucket/object",
		data:     []byte("Hello, World"),
	})
	server := httptest.NewServer(object)
	defer server.Close()

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal("Error")
	}
	a, err := minio.New(u.Host, "", "", true)
	if err != nil {
		t.Fatal("Error")
	}

	data := []byte("Hello, World")
	n, err := a.PutObject("bucket", "object", bytes.NewReader(data), int64(len(data)), "")
	if err != nil {
		t.Fatal(err)
	}
	if n != int64(len(data)) {
		t.Fatal("Error")
	}

	metadata, err := a.StatObject("bucket", "object")
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Key != "object" {
		t.Fatal("Error")
	}
	if metadata.ETag != "9af2f8218b150c351ad802c6f3d66abe" {
		t.Fatal("Error")
	}

	reader, err := a.GetObject("bucket", "object")
	if err != nil {
		t.Fatal(err)
	}

	var buffer bytes.Buffer
	_, err = io.Copy(&buffer, reader)
	if !bytes.Equal(buffer.Bytes(), data) {
		t.Fatal("Error")
	}

	err = a.RemoveObject("bucket", "object")
	if err != nil {
		t.Fatal("Error")
	}
}

func TestPresignedURL(t *testing.T) {
	object := objectHandler(objectHandler{
		resource: "/bucket/object",
		data:     []byte("Hello, World"),
	})
	server := httptest.NewServer(object)
	defer server.Close()

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal("Error")
	}
	a, err := minio.New(u.Host, "", "", true)
	if err != nil {
		t.Fatal("Error")
	}
	// should error out for invalid access keys.
	_, err = a.PresignedGetObject("bucket", "object", time.Duration(1000)*time.Second)
	if err == nil {
		t.Fatal("Error")
	}

	a, err = minio.New(u.Host, "accessKey", "secretKey", true)
	if err != nil {
		t.Fatal("Error")
	}
	url, err := a.PresignedGetObject("bucket", "object", time.Duration(1000)*time.Second)
	if err != nil {
		t.Fatal("Error")
	}
	if url == "" {
		t.Fatal("Error")
	}
	_, err = a.PresignedGetObject("bucket", "object", time.Duration(0)*time.Second)
	if err == nil {
		t.Fatal("Error")
	}
	_, err = a.PresignedGetObject("bucket", "object", time.Duration(604801)*time.Second)
	if err == nil {
		t.Fatal("Error")
	}
}

func TestErrorResponse(t *testing.T) {
	errorResponse := []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?><Error><Code>AccessDenied</Code><Message>Access Denied</Message><BucketName>mybucket</BucketName><Key>myphoto.jpg</Key><RequestId>F19772218238A85A</RequestId><HostId>GuWkjyviSiGHizehqpmsD1ndz5NClSP19DOT+s2mv7gXGQ8/X1lhbDGiIJEXpGFD</HostId></Error>")
	errorReader := bytes.NewReader(errorResponse)
	err := minio.BodyToErrorResponse(errorReader)
	if err == nil {
		t.Fatal("Error")
	}
	if err.Error() != "Access Denied" {
		t.Fatal("Error")
	}
	resp := minio.ToErrorResponse(err)
	if resp.Code != "AccessDenied" {
		t.Fatal("Error")
	}
	if resp.RequestID != "F19772218238A85A" {
		t.Fatal("Error")
	}
	if resp.Message != "Access Denied" {
		t.Fatal("Error")
	}
	if resp.BucketName != "mybucket" {
		t.Fatal("Error")
	}
	if resp.Key != "myphoto.jpg" {
		t.Fatal("Error")
	}
	if resp.HostID != "GuWkjyviSiGHizehqpmsD1ndz5NClSP19DOT+s2mv7gXGQ8/X1lhbDGiIJEXpGFD" {
		t.Fatal("Error")
	}
	if resp.ToXML() == "" {
		t.Fatal("Error")
	}
	if resp.ToJSON() == "" {
		t.Fatal("Error")
	}
}
