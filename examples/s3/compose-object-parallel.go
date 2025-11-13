//go:build ignore
// +build ignore

/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2024 MinIO, Inc.
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
	// Note: YOUR-ACCESSKEYID, YOUR-SECRETACCESSKEY, my-bucketname and my-objectname
	// are dummy values, please replace them with original values.

	// New returns an Amazon S3 compatible client object. API compatibility (v2 or v4) is automatically
	// determined based on the Endpoint value.
	s3Client, err := minio.New("s3.amazonaws.com", &minio.Options{
		Creds:  credentials.NewStaticV4("YOUR-ACCESSKEYID", "YOUR-SECRETACCESSKEY", ""),
		Secure: true,
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Prepare source objects to concatenate. We need to specify information
	// about the source objects being concatenated. Since we are using a
	// list of copy sources, none of the source objects can be less than
	// the minimum part size, except the last one.
	srcOpts1 := minio.CopySrcOptions{
		Bucket: "my-bucketname",
		Object: "my-objectname-part-1",
	}
	srcOpts2 := minio.CopySrcOptions{
		Bucket: "my-bucketname",
		Object: "my-objectname-part-2",
	}
	srcOpts3 := minio.CopySrcOptions{
		Bucket: "my-bucketname",
		Object: "my-objectname-part-3",
	}

	// Prepare destination object.
	dstOpts := minio.CopyDestOptions{
		Bucket: "my-bucketname",
		Object: "my-objectname-composite",

		// Configure parallel uploads with 10 concurrent threads
		NumThreads: 10,

		// Configure custom part size (10 MiB)
		// This is useful for controlling memory usage and optimizing for
		// different network conditions. If not specified, uses automatic calculation.
		PartSize: 10 * 1024 * 1024, // 10 MiB
	}

	// Perform the compose operation with parallel uploads
	uploadInfo, err := s3Client.ComposeObject(context.Background(), dstOpts, srcOpts1, srcOpts2, srcOpts3)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Composed object successfully:")
	log.Printf("Bucket: %s\n", uploadInfo.Bucket)
	log.Printf("Object: %s\n", uploadInfo.Key)
	log.Printf("Size: %d bytes\n", uploadInfo.Size)
	log.Printf("ETag: %s\n", uploadInfo.ETag)

	log.Println("\nParallel compose completed with:")
	log.Printf("- %d concurrent threads\n", dstOpts.NumThreads)
	log.Printf("- Part size: %d bytes\n", dstOpts.PartSize)
}
