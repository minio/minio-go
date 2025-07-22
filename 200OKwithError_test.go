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
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/minio/minio-go/v7/pkg/credentials"
)

func Test200MultipartUploadWithSpaces(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>`))
		w.(http.Flusher).Flush()
		for i := 0; i < 10; i++ {
			time.Sleep(time.Second)
			w.Write([]byte(" "))
			w.(http.Flusher).Flush()
		}

		w.Write([]byte(`<CompleteMultipartUploadResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Location>http://random/bucket/object</Location><Bucket>bucket</Bucket><Key>object</Key><ETag>&#34;2b3ffa539769372e2df9553358fe26b2-2&#34;</ETag></CompleteMultipartUploadResult>`))
	}))

	srv, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	// Instantiate new minio client object.
	core, err := NewCore(
		srv.Host,
		&Options{
			Creds:  credentials.NewStaticV4("foo", "foo12345", ""),
			Secure: srv.Scheme == "https",
		})
	if err != nil {
		t.Fatal("Error:", err)
	}

	parts := []CompletePart{
		{PartNumber: 1, ETag: "b386a859d8a22ff986c0b1252be34658"},
		{PartNumber: 2, ETag: "78c577a580bbbba92845789cda1fa932"},
	}

	foundUploadInfo, err := core.CompleteMultipartUpload(context.Background(),
		"bucket",
		"object",
		"jY1M2U5NWMtZGY2OC00ZjYyLTljZGYtYmZlOWEzODM3MDMwLjlmZWY5OGNlLWQ1Y2EtNDgwMC04N2Y4LWZkNTNkMDM4ZDdiMXgxNzQ4NjA0NzI0NzE4NjU3MTY3",
		parts,
		PutObjectOptions{},
	)
	if err != nil {
		t.Fatal("Error:", err)
	}

	expectedUploadInfo := UploadInfo{
		Bucket:   "bucket",
		Key:      "object",
		ETag:     "2b3ffa539769372e2df9553358fe26b2-2",
		Location: "http://random/bucket/object",
	}

	if foundUploadInfo != expectedUploadInfo {
		t.Fatalf("Unexpected upload info, expected: `%v`, found: `%v`", expectedUploadInfo, foundUploadInfo)
	}
}

func Test200MultipartUploadWithError(t *testing.T) {
	const maxRetries = 3
	retries := maxRetries

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		retries--
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>`))
		w.(http.Flusher).Flush()
		for i := 0; i < 5; i++ {
			time.Sleep(time.Second)
			w.Write([]byte(" "))
			w.(http.Flusher).Flush()
		}

		w.Write([]byte(`<Error><Code>SlowDownWrite</Code><Message>Resource requested is unwritable, please reduce your request rate</Message><Key>object</Key><BucketName>bucket</BucketName><Resource>/bucket/object</Resource><RequestId>18413E84F6C30613</RequestId><HostId>49371f38c0d7ec74eae2befc695360a3dfece04732914e58a4281759cd2eba4f</HostId></Error>`))
	}))

	srv, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	// Instantiate new minio client object.
	core, err := NewCore(
		srv.Host,
		&Options{
			Creds:      credentials.NewStaticV4("foo", "foo12345", ""),
			Secure:     srv.Scheme == "https",
			Region:     "us-east-1",
			MaxRetries: retries,
		})
	if err != nil {
		t.Fatal("Error:", err)
	}

	parts := []CompletePart{
		{PartNumber: 1, ETag: "b386a859d8a22ff986c0b1252be34658"},
		{PartNumber: 2, ETag: "78c577a580bbbba92845789cda1fa932"},
	}

	_, err = core.CompleteMultipartUpload(context.Background(),
		"bucket",
		"object",
		"jY1M2U5NWMtZGY2OC00ZjYyLTljZGYtYmZlOWEzODM3MDMwLjlmZWY5OGNlLWQ1Y2EtNDgwMC04N2Y4LWZkNTNkMDM4ZDdiMXgxNzQ4NjA0NzI0NzE4NjU3MTY3",
		parts,
		PutObjectOptions{},
	)
	if err == nil {
		t.Fatal("CompleteMultipartUpload() returned <nil>, which is unexpected")
	}

	expectedErrorMsg := "Resource requested is unwritable, please reduce your request rate"
	if err.Error() != expectedErrorMsg {
		t.Fatalf("Unexpected returned error, expected: `%v`, found: `%v`", expectedErrorMsg, err.Error())
	}

	if retries != 0 {
		t.Fatalf("CompleteMultipart request was not retried enough times, expected: %d, found: %d", maxRetries, retries)
	}
}

func Test200DeleteObjectsWithError(t *testing.T) {
	const maxRetries = 3
	retries := maxRetries

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		retries--
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>`))
		w.(http.Flusher).Flush()
		for i := 0; i < 5; i++ {
			time.Sleep(time.Second)
			w.Write([]byte(" "))
			w.(http.Flusher).Flush()
		}

		w.Write([]byte(`<Error><Code>SlowDownWrite</Code><Message>Resource requested is unwritable, please reduce your request rate</Message><Key>object</Key><BucketName>bucket</BucketName><Resource>/bucket/object</Resource><RequestId>18413E84F6C30613</RequestId><HostId>49371f38c0d7ec74eae2befc695360a3dfece04732914e58a4281759cd2eba4f</HostId></Error>`))
	}))

	srv, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	// Instantiate new minio client object.
	core, err := NewCore(
		srv.Host,
		&Options{
			Creds:      credentials.NewStaticV4("foo", "foo12345", ""),
			Secure:     srv.Scheme == "https",
			Region:     "us-east-1",
			MaxRetries: retries,
		})
	if err != nil {
		t.Fatal("Error:", err)
	}

	// core.TraceOn(os.Stderr)

	objs := make(chan ObjectInfo, 1000)
	for i := range 1000 {
		objs <- ObjectInfo{Key: fmt.Sprintf("obj-%d", i)}
	}
	close(objs)

	delErrCh := core.RemoveObjects(context.Background(), "bucket", objs, RemoveObjectsOptions{})
	delErr := <-delErrCh
	err = delErr.Err
	if err == nil {
		t.Fatal("RemoveObjects() returned <nil>, which is unexpected")
	}

	expectedErrorMsg := "Resource requested is unwritable, please reduce your request rate"
	if err.Error() != expectedErrorMsg {
		t.Fatalf("Unexpected returned error, expected: `%v`, found: `%v`", expectedErrorMsg, err.Error())
	}

	if retries != 0 {
		t.Fatalf("RemoveObjects() request was not retried enough times, expected: %d, found: %d", maxRetries, retries)
	}
}
