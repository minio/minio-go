/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2018 MinIO, Inc.
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
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"

	"golang.org/x/crypto/argon2"
)

// AWS S3 server-side encryption headers for SSE-S3, SSE-KMS and SSE-C.
const (
	amzServerSideEncryption         = "X-Amz-Server-Side-Encryption"
	amzServerSideEncryptionKmsKeyID = amzServerSideEncryption + "-Aws-Kms-Key-Id"
	amzServerSideEncryptionContext  = amzServerSideEncryption + "-Context"

	amzServerSideEncryptionCustomerAlgorithm = amzServerSideEncryption + "-Customer-Algorithm"
	amzServerSideEncryptionCustomerKey       = amzServerSideEncryption + "-Customer-Key"
	amzServerSideEncryptionCustomerKeyMD5    = amzServerSideEncryption + "-Customer-Key-Md5"

	amzServerSideEncryptionCopyCustomerAlgorithm = "X-Amz-Copy-Source-Server-Side-Encryption-Customer-Algorithm"
	amzServerSideEncryptionCopyCustomerKey       = "X-Amz-Copy-Source-Server-Side-Encryption-Customer-Key"
	amzServerSideEncryptionCopyCustomerKeyMD5    = "X-Amz-Copy-Source-Server-Side-Encryption-Customer-Key-Md5"
)

// PBKDF creates a SSE-C key from the provided password and salt.
// PBKDF is a password-based key derivation function
// which can be used to derive a high-entropy cryptographic
// key from a low-entropy password and a salt.
type PBKDF func(password, salt []byte) ServerSide

// DefaultPBKDF is the default PBKDF. It uses Argon2id with the
// recommended parameters from the RFC draft (1 pass, 64 MB memory, 4 threads).
var DefaultPBKDF PBKDF = func(password, salt []byte) ServerSide {
	return WithClientKey(argon2.IDKey(password, salt, 1, 64*1024, 4, 32))
}

// BucketConfig contains the server-side configuration for an S3 bucket.
type BucketConfig struct {
	Name  xml.Name     `xml:"ServerSideEncryptionConfiguration"`
	Rules []BucketRule `xml:"Rule"`
}

// BucketWithS3 returns a server-side encryption BucketConfig that
// configures the bucket to be encrypted with SSE-S3.
func BucketWithS3() *BucketConfig {
	return &BucketConfig{
		Rules: []BucketRule{ApplyS3()},
	}
}

// BucketWithS3 returns a server-side encryption BucketConfig that
// configures the bucket to be encrypted with SSE-KMS.
func BucketWithKMS(keyID string) *BucketConfig {
	return &BucketConfig{
		Rules: []BucketRule{ApplyKMS(keyID)},
	}
}

// BucketRule defines the server-side encryption rule for a bucket.
// It specifies what server-side encryption method should be applied.
type BucketRule struct {
	Rule defaultBucketRule `xml:"ApplyServerSideEncryptionByDefault"`
}

// ApplyS3 returns a BucketRule that specifies SSE-S3.
func ApplyS3() BucketRule { return sses3Rule }

// ApplyKMS returns a BucketRule that specifies SSE-KMS.
func ApplyKMS(keyID string) BucketRule {
	return BucketRule{
		Rule: defaultBucketRule{
			Algorithm:   "aws:kms",
			MasterKeyID: keyID,
		},
	}
}

type defaultBucketRule struct {
	Algorithm   string `xml:"SSEAlgorithm"`
	MasterKeyID string `xml:"KMSMasterKeyID,omitempty"`
}

var sses3Rule = BucketRule{
	Rule: defaultBucketRule{
		Algorithm: "AES256",
	},
}

// ServerSide represents S3 server-side encryption.
// It represents one of the following encryption methods:
//  - SSE-C: server-side-encryption with customer provided keys
//  - KMS:   server-side-encryption with managed keys
//  - S3:    server-side-encryption using S3 storage encryption
type ServerSide interface {
	fmt.Stringer

	// MarshalPUT adds S3 server-side encryption headers to the
	// provided HTTP headers used for PUT operations.
	MarshalPUT(http.Header)

	// MarshalGET adds S3 server-side encryption headers to the
	// provided HTTP headers used for GET operations.
	MarshalGET(http.Header)

	// MarshalCOPY adds S3 server-side encryption headers to the
	// provided HTTP headers that apply to either the copy source
	// or copy target - depending upon the CopyOp.
	MarshalCOPY(http.Header, CopyOp)
}

var ( // compiler check
	_ ServerSide = S3
	_ ServerSide = ssekms{}
	_ ServerSide = ssec{}
)

// CopyOp is the operand of a S3 COPY operation.
// An operand is either the COPY source or target.
type CopyOp bool

const (
	CopySource = CopyOp(false)
	CopyTarget = CopyOp(true)
)

// S3 encrypts the object with a key internally managed by the
// S3 server.
const S3 = sses3(0)

type sses3 int

// String returns "SSE-S3" as the string representation
// of SSE-S3.
func (sses3) String() string { return "SSE-S3" }

// MarshalPUT adds the SSE-S3 server-side encryption HTTP
// header.
func (sses3) MarshalPUT(h http.Header) { h.Add(amzServerSideEncryption, "AES256") }

// MarshalGET does nothing since the SSE-S3 encryption HTTP
// header must not be specified for GET / HEAD operations.
func (sses3) MarshalGET(h http.Header) {}

// MarshalPUT adds the SSE-S3 server-side encryption HTTP
// header.
func (sses3) MarshalCOPY(h http.Header, op CopyOp) {
	if op == CopyTarget {
		h.Add(amzServerSideEncryption, "AES256")
	}
}

type ssekms struct {
	keyID   string
	context []byte
}

// WithKMS encrypts the object with the key referenced by keyID
// that is managed a KMS.
//
// The context, if not nil, is bound to the encrypted object
// and must be provided whenever the object should be accessed
// again.
func WithKMS(keyID string, context map[string]string) (ServerSide, error) {
	var (
		bytes []byte
		err   error
	)
	if context != nil {
		bytes, err = json.Marshal(context)
		if err != nil {
			return nil, err
		}
	}
	return ssekms{
		keyID:   keyID,
		context: bytes,
	}, nil
}

// String returns "SSE-KMS" as the string representation
// of SSE-KMS.
func (s ssekms) String() string { return "SSE-KMS" }

// MarshalPUT adds the SSE-KMS server-side encryption HTTP
// headers.
func (s ssekms) MarshalPUT(h http.Header) {
	h.Add(amzServerSideEncryption, "aws:kms")
	if s.keyID != "" {
		h.Add(amzServerSideEncryptionKmsKeyID, s.keyID)
	}
	if s.context != nil {
		h.Add(amzServerSideEncryptionContext, base64.StdEncoding.EncodeToString(s.context))
	}
}

// MarshalGET adds the SSE-KMS server-side encryption HTTP
// header containing the encryption context, if any.
func (s ssekms) MarshalGET(h http.Header) {
	if s.context != nil {
		h.Add(amzServerSideEncryptionContext, base64.StdEncoding.EncodeToString(s.context))
	}
}

// MarshalCOPY adds the SSE-KMS server-side encryption HTTP
// header depending upon the copy operand.
func (s ssekms) MarshalCOPY(h http.Header, op CopyOp) {
	if op == CopySource {
		s.MarshalGET(h)
	} else {
		s.MarshalPUT(h)
	}
}

type ssec []byte

// WithClientKey encrypts an object with the given key.
//
// The S3 request must be made over a secure connection
// via HTTPS since it contains the key as part of the
// HTTP headers.
func WithClientKey(key []byte) ServerSide {
	k := make(ssec, len(key))
	copy(k, key)
	return k
}

// String returns "SSE-C" as the string representation
// of SSE-C.
func (ssec) String() string { return "SSE-C" }

// MarshalPUT adds the SSE-C server-side encryption HTTP
// headers containing the client secret key.
func (s ssec) MarshalPUT(h http.Header) {
	md5Sum := md5.Sum(s)

	h.Add(amzServerSideEncryptionCustomerAlgorithm, "AES256")
	h.Add(amzServerSideEncryptionCustomerKey, base64.StdEncoding.EncodeToString(s))
	h.Add(amzServerSideEncryptionCustomerKeyMD5, base64.StdEncoding.EncodeToString(md5Sum[:]))
}

// MarshalPUT adds the SSE-C server-side encryption HTTP
// headers containing the client secret key.
func (s ssec) MarshalGET(h http.Header) { s.MarshalPUT(h) }

// MarshalPUT adds the SSE-C server-side encryption copy HTTP
// headers containing the client secret key.
func (s ssec) MarshalCOPY(h http.Header, op CopyOp) {
	if op == CopyTarget {
		s.MarshalPUT(h)
	} else {
		md5Sum := md5.Sum(s)
		h.Add(amzServerSideEncryptionCopyCustomerAlgorithm, "AES256")
		h.Add(amzServerSideEncryptionCopyCustomerKey, base64.StdEncoding.EncodeToString(s))
		h.Add(amzServerSideEncryptionCopyCustomerKeyMD5, base64.StdEncoding.EncodeToString(md5Sum[:]))
	}
}
