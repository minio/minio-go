//go:build example
// +build example

/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2015-2023 MinIO, Inc.
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
	"encoding/base64"
	"hash/crc32"
	"io"
	"log"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
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

	object, err := os.Open("putobject-checksum.go")
	if err != nil {
		log.Fatalln(err)
	}
	defer object.Close()
	objectStat, err := object.Stat()
	if err != nil {
		log.Fatalln(err)
	}
	// Create CRC32C:
	crc := crc32.New(crc32.MakeTable(crc32.Castagnoli))
	_, err = io.Copy(crc, object)
	if err != nil {
		log.Fatalln(err)
	}
	meta := map[string]string{"x-amz-checksum-crc32c": base64.StdEncoding.EncodeToString(crc.Sum(nil))}

	// Reset object.
	_, err = object.Seek(0, io.SeekStart)
	if err != nil {
		log.Fatalln(err)
	}

	// Upload object.
	// Checksums are different with multipart, so we disable that.
	info, err := s3Client.PutObject(context.Background(), "my-bucket", "my-objectname", object, objectStat.Size(), minio.PutObjectOptions{ContentType: "application/octet-stream", UserMetadata: meta, DisableMultipart: true})
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Uploaded", "my-objectname", "of size:", info.Size, "with CRC32C:", info.ChecksumCRC32C, "successfully.")
}
