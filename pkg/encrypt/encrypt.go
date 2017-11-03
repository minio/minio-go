/*
 * Minio Go Library for Amazon S3 Compatible Cloud Storage (C) 2017 Minio, Inc.
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

// Package encrypt implements a generic interface to encrypt S3 objects.
// Currently this package implements three types of encryption.
// - Symmetric encryption using the encryption capabilities of the server as defined by
//   the AWS S3 specification.
// - Symmetric encryption using DARE-HMAC-SHA256. This algorithm provides authenticated
//   encryption and should be preferred over any current AWS client-side-encryption
//   algorithm.
// - Symmetric encryption using AES-CBC-PKCS-5. This algorithm is provided for
//   AWS compability but is not recommended because of security issues.
package encrypt

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"golang.org/x/crypto/scrypt"
)

// AWS client-side-encryption headers.
// See: https://docs.aws.amazon.com/AmazonS3/latest/dev/UsingClientSideEncryption.html
const (
	cseIV        = "X-Amz-Meta-X-Amz-Iv"
	cseKey       = "X-Amz-Meta-X-Amz-Key-v2"
	cseAlgorithm = "X-Amz-Meta-X-Amz-Cek-Alg"
)

// AWS client-side-encryption headers.
// See: https://docs.aws.amazon.com/AmazonS3/latest/dev/ServerSideEncryptionCustomerKeys.html
const (
	sseAlgorithm = "X-Amz-Server-Side-Encryption-Customer-Algorithm"
	sseKey       = "X-Amz-Server-Side-Encryption-Customer-Key"
	sseKeyMD5    = "X-Amz-Server-Side-Encryption-Customer-Key-Md5"
)

// Cipher is a generic interface for en/decrypting streams using
// S3 client/server side encryption. Cipher is the functional equivalent
// of EncryptionMaterials of the aws-go-sdk.
type Cipher interface {
	// Seal returns an io.ReadCloser encrypting everything it reads from
	// the provided io.Reader. It adds HTTP headers to the provided header
	// if necessary. Seal returns an error if it is not able to encrypt
	// the io.Reader
	Seal(header map[string]string, src io.Reader) (io.ReadCloser, error)

	// Open returns an io.ReadCloser decrypting everything it reads from
	// the provided io.Reader. It reads HTTP headers from the provided header
	// if necessary. Open returns an error if it is not able to decrypt
	// the io.Reader
	Open(header map[string]string, src io.Reader) (io.ReadCloser, error)

	// Overhead returns the size of an encrypted stream with the provided
	// size. The size of an encrypted stream is usually larger than an
	// unencrypted one.
	Overhead(size int64) int64
}

const (
	// SCrypt2017 specifies the PBKDF scrypt with the recommended security parameters
	// for 2017. (N = 32768, r = 8, p = 1)
	SCrypt2017 PBKDF = 1 + iota
)

// PBKDF specifies a password-based key-derivation-function
// to derive a secret key from a password and salt.
type PBKDF uint

// DeriveKey derives a 'size'-bytes long secret key from the provided
// password and salt.
func (kdf PBKDF) DeriveKey(password, salt []byte, size int) ([]byte, error) {
	switch kdf {
	default:
		return nil, fmt.Errorf("PBKDF '%s' is not supported", kdf)
	case SCrypt2017:
		return scrypt.Key(password, salt, 32768, 8, 1, size)
	}
}

func (kdf PBKDF) String() string {
	switch kdf {
	default:
		return strconv.Itoa(int(kdf))
	case SCrypt2017:
		return "scrypt: N=32768 r=8 p=1"
	}
}

// ServerSide implements Cipher and specifies server-side-encryption as
// defined by the AWS S3 specification.
type ServerSide struct {
	// The secret encryption key. Notice that this key will be sent to the server.
	// The key must be 32 bytes long.
	Key []byte
	// The Algorithm used to encrypt the object at the server. The only valid
	// value is "AES256".
	Algorithm string
}

// Header returns the HTTP header representation of the S3 server-side-encryption
// key and algorithm.
func (s *ServerSide) Header() http.Header {
	keyMD5 := md5.Sum(s.Key)
	return http.Header{
		sseAlgorithm: []string{s.Algorithm},
		sseKey:       []string{base64.StdEncoding.EncodeToString(s.Key)},
		sseKeyMD5:    []string{base64.StdEncoding.EncodeToString(keyMD5[:])},
	}
}
