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
	"io"

	"github.com/minio/minio-go/pkg/encrypt"
)

// GetEncryptedObject tries to get an server-side-encrypted object.
// It returns an error if the key - derived from the provided password - does not
// match the encryption key of the object. GetEncryptedObject requires a TLS connection.
func (c Client) GetEncryptedObject(bucketName, objectName, password string) (io.ReadCloser, error) {
	salt := []byte(kdfMagicConstant + bucketName + objectName)
	sse, err := encrypt.NewServerSide(defaultPBKDF([]byte(password), salt, 32))
	if err != nil {
		return nil, err
	}
	return c.GetObject(bucketName, objectName, GetObjectOptions{ServerSideEncryption: sse})
}

// FGetEncryptedObject tries to get an server-side-encrypted object and stores
// at the filePath.
// It returns an error if the key - derived from the provided password - does not
// match the encryption key of the object. GetEncryptedObject requires a TLS connection.
func (c Client) FGetEncryptedObject(bucketName, objectName, filePath, password string) error {
	salt := []byte(kdfMagicConstant + bucketName + objectName)
	sse, err := encrypt.NewServerSide(defaultPBKDF([]byte(password), salt, 32))
	if err != nil {
		return err
	}
	return c.FGetObject(bucketName, objectName, filePath, GetObjectOptions{ServerSideEncryption: sse})
}
