// +build ignore

/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2020 MinIO, Inc.
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
	"context"
	"log"
	"time"

	"github.com/minio/minio-go/v7"
)

func main() {
	// Note: YOUR-ACCESSKEYID, YOUR-SECRETACCESSKEY, my-testfile, my-bucketname and
	// my-objectname are dummy values, please replace them with original values.

	// Requests are always secure (HTTPS) by default. Set secure=false to enable insecure (HTTP) access.
	// This boolean value is the last argument for New().

	// New returns an Amazon S3 compatible client object. API compatibility (v2 or v4) is automatically
	// determined based on the Endpoint value.
	s3Client, err := minio.New("s3.amazonaws.com", "YOUR-ACCESSKEYID", "YOUR-SECRETACCESSKEY", true)
	if err != nil {
		log.Fatalln(err)
	}

	// Enable trace.
	// s3Client.TraceOn(os.Stderr)

	// Source object
	src := minio.CopySrcOptions{
		Bucket: "my-sourcebucketname",
		Object: "my-sourceobjectname",
		// All following conditions are allowed and can be combined together.
		// Set modified condition, copy object modified since 2014 April.
		MatchModifiedSince: time.Date(2014, time.April, 0, 0, 0, 0, 0, time.UTC),
	}

	// Destination object
	dst := minio.CopyDestOptions{
		Bucket:      "my-bucketname",
		Object:      "my-objectname",
		ReplaceTags: true,
		UserTags: map[string]string{
			"Tag1": "Value1",
			"Tag2": "Value2",
		},
	}

	// Initiate copy object.
	ui, err := s3Client.CopyObject(context.Background(), dst, src)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("Copied %s, successfully to %s - UploadInfo %v\n", dst, src, ui)
}
