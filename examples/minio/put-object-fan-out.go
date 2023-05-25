//go:build example
// +build example

/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2023 MinIO, Inc.
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
	"io"
	"log"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/encrypt"
)

func main() {
	const (
		// Note: These constants are dummy values,
		// please replace them with values for your setup.
		YOURACCESSKEYID     = "Q3AM3UQ867SPQQA43P2F"
		YOURSECRETACCESSKEY = "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG"
		YOURENDPOINT        = "play.min.io"
		YOURBUCKET          = "mybucket" // 'mc mb play/mybucket' if it does not exist.
	)

	// Requests are always secure (HTTPS) by default. Set secure=false to enable insecure (HTTP) access.
	// This boolean value is the last argument for New().

	// New returns an Amazon S3 compatible client object. API compatibility (v2 or v4) is automatically
	// determined based on the Endpoint value.
	minioClient, err := minio.New(YOURENDPOINT, &minio.Options{
		Creds:  credentials.NewStaticV4(YOURACCESSKEYID, YOURSECRETACCESSKEY, ""),
		Secure: true,
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Enable tracing.
	minioClient.TraceOn(os.Stdout)

	filePath := "my-testfile" // Specify a local file that we will upload

	// Open a local file that we will upload
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	cs, err := minio.ChecksumCRC32C.ChecksumReader(file)
	if err != nil {
		log.Fatalln(err)
	}

	// Seek to beginning before upload.
	file.Seek(0, io.SeekStart)

	fanOutReq := minio.PutObjectFanOutRequest{
		Entries: []minio.PutObjectFanOutEntry{
			{Key: "my1-prefix/1.txt"},
			{Key: "my1-prefix/2.txt"},
			{Key: "my1-prefix/3.txt"},
			{Key: "my1-prefix/4.txt"},
			{Key: "my1-prefix/5.txt"},
			{Key: "my1-prefix/6.txt"},
		},
		SSE:      encrypt.NewSSE(),
		Checksum: cs,
	}

	fanOutResp, err := minioClient.PutObjectFanOut(context.Background(), YOURBUCKET, file, fanOutReq)
	if err != nil {
		log.Fatalln(err)
	}

	for _, resp := range fanOutResp {
		fmt.Println(resp)
	}
}
