/*
 * Minio Go Library for Amazon S3 Compatible Cloud Storage (C) 2015 Minio, Inc.
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
)

// PutEncryptedObject - Encrypt and store object.
func (c Client) PutEncryptedObject(bucketName, objectName string, reader io.Reader, password string) (n int64, err error) {
	key, err := encrypt.SCrypt2017.DeriveKey([]byte(password), []byte(bucketName+objectName), 32)
	if err != nil {
		return 0, err
	}
	return c.PutObjectWithContext(context.Background(), bucketName, objectName, reader, -1, PutObjectOptions{
		ServerSideEncryption: &encrypt.ServerSide{
			Key:       key,
			Algorithm: "AES256",
		},
	})
}

// FPutEncryptedObject - Encrypt and store an object with contents from file at filePath.
func (c Client) FPutEncryptedObject(bucketName, objectName, filePath, password string) (n int64, err error) {
	key, err := encrypt.SCrypt2017.DeriveKey([]byte(password), []byte(bucketName+objectName), 32)
	if err != nil {
		return 0, err
	}
	return c.FPutObjectWithContext(context.Background(), bucketName, objectName, filePath, PutObjectOptions{
		ServerSideEncryption: &encrypt.ServerSide{
			Key:       key,
			Algorithm: "AES256",
		},
	})
}
