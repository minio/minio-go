/*
 * Minio Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2017 Minio, Inc.
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

package minio

import (
	"context"
	"io"

	"github.com/minio/minio-go/pkg/encrypt"
	"golang.org/x/crypto/argon2"
)

// A magic value used to prevent that client side encryption and server side encryption
// derive the same key if Argon2i(password, bucket+object) is used.
const kdfMagicConstant = "SSE-C"

func defaultPBKDF(password, salt []byte, keyLen int) []byte {
	return argon2.Key(password, salt, 5, 32*1024, 4, uint32(keyLen)) // medium secure Argon2i security parameters
}

// PutEncryptedObject creates a server-side encrypted object at the given bucketName/objectName.
// The object is encrypted with a key derived from the given password using server-side encryption.
func (c Client) PutEncryptedObject(bucketName, objectName string, reader io.Reader, size int64, password string) (n int64, err error) {
	salt := []byte(kdfMagicConstant + bucketName + objectName)
	sse, err := encrypt.NewServerSide(defaultPBKDF([]byte(password), salt, 32))
	if err != nil {
		return 0, err
	}
	return c.PutObjectWithContext(context.Background(), bucketName, objectName, reader, size, PutObjectOptions{ServerSideEncryption: sse})
}

// FPutEncryptedObject creates a server-side encrypted object from the given filePath at the given
// bucketName/objectName. The object is encrypted with a key derived from the given password using
// server-side encryption.
func (c Client) FPutEncryptedObject(bucketName, objectName, filePath, password string) (n int64, err error) {
	salt := []byte(kdfMagicConstant + bucketName + objectName)
	sse, err := encrypt.NewServerSide(defaultPBKDF([]byte(password), salt, 32))
	if err != nil {
		return 0, err
	}
	return c.FPutObjectWithContext(context.Background(), bucketName, objectName, filePath, PutObjectOptions{ServerSideEncryption: sse})
}
