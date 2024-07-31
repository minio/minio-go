//go:build example
// +build example

/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2015-2024 MinIO, Inc.
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
	"fmt"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/cors"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func main() {
	// Note: YOUR-ACCESSKEYID, YOUR-SECRETACCESSKEY, my-bucketname and my-prefixname
	// are dummy values, please replace them with original values.

	// Requests are always secure (HTTPS) by default. Set secure=false to enable insecure (HTTP) access.
	// This boolean value is the last argument for New().

	// New returns an Amazon S3 compatible client object. API compatibility (v2 or v4) is automatically
	// determined based on the Endpoint value.
	s3Client, err := minio.New("s3.amazonaws.com", &minio.Options{
		Creds:  credentials.NewStaticV4("YOUR-ACCESSKEYID", "YOUR-SECRETACCESSKEY", ""),
		Secure: true,
	})
	if err != nil {
		log.Fatalln(err)
	}
	bucket := "my-bucket-name"

	corsRules := []cors.Rule{
		{
			AllowedHeader: []string{"*"},
			AllowedMethod: []string{"GET", "PUT"},
			AllowedOrigin: []string{"https://example.com"},
		},
	}
	corsConfig := cors.NewConfig(corsRules)

	err = s3Client.SetBucketCors(context.Background(), bucket, corsConfig)
	if err != nil {
		log.Fatalln(fmt.Errorf("Error setting bucket cors: %v", err))
	}

	retCors, err := s3Client.GetBucketCors(context.Background(), bucket)
	if err != nil {
		log.Fatalln(fmt.Errorf("Error getting bucket cors: %v", err))
	}

	fmt.Printf("Returned Bucket CORS configuration: %+v\n", retCors)

	err = s3Client.SetBucketCors(context.Background(), bucket, nil)
	if err != nil {
		log.Fatalln(fmt.Errorf("Error removing bucket cors: %v", err))
	}
}
