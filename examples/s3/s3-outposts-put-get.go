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

// This example uses the minio-go client against AWS S3 on Outposts.
// It performs a PutObject and GetObject to verify the client works with
// Outposts endpoints (s3-outposts signing and path-style requests).
//
// Set environment variables (do not commit real credentials):
//
//	S3_OUTPOSTS_ENDPOINT  - Outposts access point endpoint (e.g. myap-123.op-xxx.s3-outposts.region.amazonaws.com)
//	S3_OUTPOSTS_BUCKET    - Access point alias / bucket name (e.g. mybucket--op-xxx--op-s3)
//	S3_OUTPOSTS_REGION    - AWS region (e.g. eu-central-1). Optional if AWS_REGION is set.
//	S3_OUTPOSTS_PROFILE   - Optional. AWS credentials profile. If unset, uses AWS_ACCESS_KEY_ID + AWS_SECRET_ACCESS_KEY.
//	AWS_ACCESS_KEY_ID     - Required if S3_OUTPOSTS_PROFILE is unset
//	AWS_SECRET_ACCESS_KEY - Required if S3_OUTPOSTS_PROFILE is unset
//
// Run from repo root (uses local minio-go):
//
//	go run -tags example ./examples/s3/s3-outposts-put-get.go
//
// Or from examples/s3 (replace in go.mod points to parent minio-go):
//
//	cd examples/s3 && go run -tags example s3-outposts-put-get.go
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func main() {
	endpoint := os.Getenv("S3_OUTPOSTS_ENDPOINT")
	bucket := os.Getenv("S3_OUTPOSTS_BUCKET")
	region := os.Getenv("S3_OUTPOSTS_REGION")
	if region == "" {
		region = os.Getenv("AWS_REGION")
	}
	profile := os.Getenv("S3_OUTPOSTS_PROFILE")

	if endpoint == "" || bucket == "" || region == "" {
		log.Fatalf("Missing required env: set S3_OUTPOSTS_ENDPOINT, S3_OUTPOSTS_BUCKET, and S3_OUTPOSTS_REGION (or AWS_REGION)")
	}

	var creds *credentials.Credentials
	if profile != "" {
		creds = credentials.NewFileAWSCredentials("", profile)
	} else {
		accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
		secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
		if accessKey == "" || secretKey == "" {
			log.Fatalf("Set S3_OUTPOSTS_PROFILE or both AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY")
		}
		creds = credentials.NewStaticV4(accessKey, secretKey, "")
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  creds,
		Secure: true,
		Region: region,
	})
	if err != nil {
		log.Fatalf("New client: %v", err)
	}

	objectKey := "outposts-test/hello-minio-go.txt"
	objectBody := "Hello from minio-go S3 on Outposts\n"

	ctx := context.Background()

	fmt.Println("PutObject...")
	_, err = client.PutObject(ctx, bucket, objectKey, strings.NewReader(objectBody), int64(len(objectBody)), minio.PutObjectOptions{
		ContentType: "text/plain",
	})
	if err != nil {
		log.Fatalf("PutObject: %v", err)
	}
	fmt.Println("PutObject OK")

	fmt.Println("GetObject...")
	obj, err := client.GetObject(ctx, bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		log.Fatalf("GetObject: %v", err)
	}
	defer obj.Close()

	data, err := io.ReadAll(obj)
	if err != nil {
		log.Fatalf("Read body: %v", err)
	}
	fmt.Println("GetObject OK")
	fmt.Println("Content:", string(data))
	fmt.Println("Done. MinIO client works with S3 on Outposts.")
}
