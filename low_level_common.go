/*
 * Minimal object storage library (C) 2015 Minio, Inc.
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

package objectstorage

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"io"
)

func Sum256Reader(reader io.ReadSeeker) ([]byte, error) {
	h := sha256.New()
	var err error

	start, _ := reader.Seek(0, 1)
	defer reader.Seek(start, 0)

	for err == nil {
		length := 0
		byteBuffer := make([]byte, 1024*1024)
		length, err = reader.Read(byteBuffer)
		byteBuffer = byteBuffer[0:length]
		h.Write(byteBuffer)
	}

	if err != io.EOF {
		return nil, err
	}

	return h.Sum(nil), nil
}

func Sum256(data []byte) []byte {
	hash := sha256.New()
	hash.Write(data)
	return hash.Sum(nil)
}

func SumHMAC(key []byte, data []byte) []byte {
	hash := hmac.New(sha256.New, key)
	hash.Write(data)
	return hash.Sum(nil)
}

// calculate md5
func contentMD5(body io.ReadSeeker, size int64) (string, error) {
	hasher := md5.New()
	_, err := io.CopyN(hasher, body, size)
	if err != nil {
		return "", err
	}
	// seek back
	_, err = body.Seek(0, 0)
	if err != nil {
		return "", err
	}
	// encode the md5 checksum in base64 and set the request header.
	return base64.StdEncoding.EncodeToString(hasher.Sum(nil)), nil
}
