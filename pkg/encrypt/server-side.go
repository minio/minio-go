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

package encrypt

import (
	"crypto/md5"
	"encoding/base64"
	"errors"
	"net/http"
)

const (
	// AWS S3 SSE-C HTTP headers: https://docs.aws.amazon.com/en_en/AmazonS3/latest/dev/ServerSideEncryptionCustomerKeys.html
	sseAlgorithm = "X-Amz-Server-Side-Encryption-Customer-Algorithm"
	sseKey       = "X-Amz-Server-Side-Encryption-Customer-Key"
	sseKeyMD5    = "X-Amz-Server-Side-Encryption-Customer-Key-MD5"

	// AWS S3 SSE-C copy HTTP headers: https://docs.aws.amazon.com/en_en/AmazonS3/latest/dev/ServerSideEncryptionCustomerKeys.html
	sseCopyAlgorithm = "X-Amz-Copy-Source-Server-Side-Encryption-Customer-Algorithm"
	sseCopyKey       = "X-Amz-Copy-Source-Server-Side-Encryption-Customer-Key"
	sseCopyKeyMD5    = "X-Amz-Copy-Source-Server-Side-Encryption-Customer-Key-MD5"

	// Only valid value for SSE-C algorithm
	sseAlgorithmAES256 = "AES256"
)

// PBKDF specifies a password-based key derivation function.
// It takes a password, a salt and a key length in bytes
// and produces a high entropy key of the given length.
// A PBKDF should be used to derive a cryptographic key
// e.g. a AES-256 key from a password.
type PBKDF func(password, salt []byte, keyLen int) []byte

// ServerSide represents the encryption key for SSE-C requests.
type ServerSide [32]byte

// NewServerSide returns a new ServerSide from the given key.
// It returns an error if the key is not 32 bytes long.
func NewServerSide(key []byte) (*ServerSide, error) {
	if len(key) != 32 {
		return nil, errors.New("the server side encryption key must be 32 bytes long")
	}
	s := new(ServerSide)
	copy(s[:], key)
	return s, nil
}

// Headers returns the HTTP header representation of the server-side encryption.
func (s ServerSide) Headers() http.Header {
	md5Sum := md5.Sum(s[:])
	h := make(http.Header)
	h.Add(sseAlgorithm, sseAlgorithmAES256)
	h.Add(sseKey, base64.StdEncoding.EncodeToString(s[:]))
	h.Add(sseKeyMD5, base64.StdEncoding.EncodeToString(md5Sum[:]))
	return h
}

// CopySourceHeaders returns the HTTP header representation of the server-side
// encryption used during copy requests. It is only required if:
// - The copy source is encrypted and the destination should also be encrypted.
// - The copy source is encrypted and the copy source == copy destination.
//   This can be used to change the object metadata or the SSE-C encryption
//   key (key rotation).
func (s ServerSide) CopySourceHeaders() http.Header {
	md5Sum := md5.Sum(s[:])
	h := make(http.Header)
	h.Add(sseCopyAlgorithm, sseAlgorithmAES256)
	h.Add(sseCopyKey, base64.StdEncoding.EncodeToString(s[:]))
	h.Add(sseCopyKeyMD5, base64.StdEncoding.EncodeToString(md5Sum[:]))
	return h
}
