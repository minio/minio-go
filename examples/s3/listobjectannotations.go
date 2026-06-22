//go:build example
// +build example

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

package main

import (
	"context"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func main() {
	// Note: my-bucketname and my-objectname are dummy values, please replace
	// them with original values.

	s3Client, err := minio.New("play.min.io", &minio.Options{
		Creds:  credentials.NewStaticV4("Q3AM3UQ867SPQQA43P2F", "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG", ""),
		Secure: true,
	})
	if err != nil {
		log.Fatalln(err)
	}

	annotations, err := s3Client.ListObjectAnnotations(context.Background(), "my-bucketname", "my-objectname", minio.ListObjectAnnotationsOptions{})
	if err != nil {
		log.Fatalln(err)
	}

	for _, a := range annotations {
		log.Printf("name=%s size=%d etag=%s lastModified=%s\n", a.Name, a.Size, a.ETag, a.LastModified)
	}
}
