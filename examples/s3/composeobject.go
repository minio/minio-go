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

	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/encrypt"
)

func main() {
	// Note: YOUR-ACCESSKEYID, YOUR-SECRETACCESSKEY, my-testfile, my-bucketname and
	// my-objectname are dummy values, please replace them with original values.

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

	// Enable trace.
	// s3Client.TraceOn(os.Stderr)

	// Prepare source decryption key (here we assume same key to
	// decrypt all source objects.)
	decKey, _ := encrypt.NewSSEC([]byte{1, 2, 3})

	// Source objects to concatenate. We also specify decryption
	// key for each
	src1 := minio.CopySrcOptions{
		Bucket:     "bucket1",
		Object:     "object1",
		Encryption: decKey,
		MatchETag:  "31624deb84149d2f8ef9c385918b653a",
	}

	src2 := minio.CopySrcOptions{
		Bucket:     "bucket2",
		Object:     "object2",
		Encryption: decKey,
		MatchETag:  "f8ef9c385918b653a31624deb84149d2",
	}

	src3 := minio.CopySrcOptions{
		Bucket:     "bucket3",
		Object:     "object3",
		Encryption: decKey,
		MatchETag:  "5918b653a31624deb84149d2f8ef9c38",
	}

	// Create slice of sources.
	srcs := []minio.SourceInfo{src1, src2, src3}

	// Prepare destination encryption key
	encKey, _ := encrypt.NewSSEC([]byte{8, 9, 0})

	// Create destination info
	dst := minio.CopyDestOptions{
		Bucket:     "bucket",
		Object:     "object",
		Encryption: encKey,
	}

	uploadInfo, err := s3Client.ComposeObject(context.Background(), dst, srcs...)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Composed object successfully:", uploadInfo)
}
