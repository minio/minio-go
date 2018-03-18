// +build ignore

/*
 * Minio Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2018 Minio, Inc.
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
	"log"
	"os"
	"path"

	"github.com/minio/minio-go"
	"github.com/minio/sio"
	"golang.org/x/crypto/argon2"
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

	object, err := os.Open("my-testfile")
	if err != nil {
		log.Fatalln(err)
	}
	defer object.Close()
	objectStat, err := object.Stat()
	if err != nil {
		log.Fatalln(err)
	}

	password := []byte("myfavoritepassword")                    // Change as per your needs.
	salt := []byte(path.Join("my-bucketname", "my-objectname")) // Change as per your needs.
	encrypted, err := sio.EncryptReader(object, sio.Config{
		// generate a 256 bit long key.
		Key: argon2.IDKey(password, salt, 1, 64*1024, 4, 32),
	})
	if err != nil {
		log.Fatalln(err)
	}

	encSize, err := sio.EncryptedSize(uint64(objectStat.Size()))
	if err != nil {
		log.Fatalln(err)
	}
	_, err = s3Client.PutObject("my-bucketname", "my-objectname", encrypted, int64(encSize), minio.PutObjectOptions{})
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Successfully encrypted 'my-objectname'")
}
