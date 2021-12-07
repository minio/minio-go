//go:build example
// +build example

/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2015-2021 MinIO, Inc.
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
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
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
	minioClient.TraceOn(os.Stdout)

	input := make(chan minio.SnowballObject, 1)
	opts := minio.SnowballOptions{
		Opts: minio.PutObjectOptions{},
		// Keep in memory. We use this since we have small total payload.
		InMemory: true,
		// Compress data when uploading to a MinIO host.
		Compress: true,
	}

	// Generate a shared prefix.
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	prefix := []byte("aaaaaaaaaaaaaaa")
	for i := range prefix {
		prefix[i] += byte(rng.Intn(25))
	}

	// Generate
	go func() {
		defer close(input)

		// Create 100 objects
		for i := 0; i < 100; i++ {
			// With random size 0 -> 10000
			size := rng.Intn(10000)
			key := fmt.Sprintf("%s/%d-%d.bin", string(prefix), i, size)
			input <- minio.SnowballObject{
				// Create path to store objects within the bucket.
				Key:     key,
				Size:    int64(size),
				ModTime: time.Now(),
				Content: bytes.NewBuffer(make([]byte, size)),
				Close: func() {
					fmt.Println(key, "Close function called")
				},
			}
		}
	}()

	// Collect and upload all entries.
	err = minioClient.PutObjectsSnowball(context.TODO(), YOURBUCKET, opts, input)
	if err != nil {
		log.Fatalln(err)
	}
	// Objects successfully uploaded.

	// List the content of the prefix:
	lopts := minio.ListObjectsOptions{
		Recursive: true,
		Prefix:    string(prefix) + "/",
	}

	// List all objects from a bucket-name with a matching prefix.
	for object := range minioClient.ListObjects(context.Background(), YOURBUCKET, lopts) {
		if object.Err != nil {
			log.Fatalln(object.Err)
		}
		fmt.Println(object.Key, "Size:", object.Size)
	}
}
