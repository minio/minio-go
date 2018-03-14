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
	"errors"
	"io"
	"log"
	"os"
	"path"

	"github.com/minio/minio-go"
	"github.com/minio/sio"
	"golang.org/x/crypto/argon2"
)

const (
	// SSE DARE package block size.
	sseDAREPackageBlockSize = 64 * 1024 // 64KiB bytes

	// SSE DARE package meta padding bytes.
	sseDAREPackageMetaSize = 32 // 32 bytes
)

func decryptedSize(encryptedSize int64) (int64, error) {
	if encryptedSize == 0 {
		return encryptedSize, nil
	}
	size := (encryptedSize / (sseDAREPackageBlockSize + sseDAREPackageMetaSize)) * sseDAREPackageBlockSize
	if mod := encryptedSize % (sseDAREPackageBlockSize + sseDAREPackageMetaSize); mod > 0 {
		if mod < sseDAREPackageMetaSize+1 {
			return -1, errors.New("object is tampered")
		}
		size += mod - sseDAREPackageMetaSize
	}
	return size, nil
}

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

	obj, err := s3Client.GetObject("my-bucketname", "my-objectname", minio.GetObjectOptions{})
	if err != nil {
		log.Fatalln(err)
	}

	objSt, err := obj.Stat()
	if err != nil {
		log.Fatalln(err)
	}

	size, err := decryptedSize(objSt.Size)
	if err != nil {
		log.Fatalln(err)
	}

	localFile, err := os.Create("my-testfile")
	if err != nil {
		log.Fatalln(err)
	}
	defer localFile.Close()

	password := []byte("myfavoritepassword")                    // Change as per your needs.
	salt := []byte(path.Join("my-bucketname", "my-objectname")) // Change as per your needs.
	decrypted, err := sio.DecryptReader(obj, sio.Config{
		// generate a 256 bit long key.
		Key: argon2.IDKey(password, salt, 1, 64*1024, 4, 32),
	})
	if err != nil {
		log.Fatalln(err)
	}

	if _, err := io.CopyN(localFile, decrypted, size); err != nil {
		log.Fatalln(err)
	}

	log.Println("Successfully decrypted 'my-objectname' to local file 'my-testfile'")
}
