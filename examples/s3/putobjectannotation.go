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
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func main() {
	// Note: my-bucketname, my-objectname and my-annotationname are dummy
	// values, please replace them with original values.

	s3Client, err := minio.New("play.min.io", &minio.Options{
		Creds:  credentials.NewStaticV4("Q3AM3UQ867SPQQA43P2F", "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG", ""),
		Secure: true,
	})
	if err != nil {
		log.Fatalln(err)
	}

	// The annotation payload is any byte stream (JSON/XML/YAML/plain), 1 byte to
	// 1 MiB, supplied as an io.ReadSeeker so it can be streamed without buffering.
	payload := strings.NewReader(`{"label":"cat","confidence":0.98}`)

	etag, err := s3Client.PutObjectAnnotation(context.Background(), "my-bucketname", "my-objectname", "model.labels.json", payload, minio.PutObjectAnnotationOptions{
		// VersionID: "target a specific object version",
		// IfMatch:   "only write if the object's ETag matches",
	})
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("annotation written, etag: %s\n", etag)
}
