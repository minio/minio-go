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

package encrypt

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"io/ioutil"

	"github.com/minio/sio"
)

// dareHmacSha256 implements encrypt.Cipher with the DARE format
// and HMAC-SHA256 as encryption key derivation function.
type dareHmacSha256 [32]byte

func (d dareHmacSha256) Seal(header map[string]string, r io.Reader) (io.ReadCloser, error) {
	iv := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	mac := hmac.New(sha256.New, d[:])
	mac.Write(iv)
	reader, err := sio.EncryptReader(r, sio.Config{Key: mac.Sum(nil)})
	if err != nil {
		return nil, err
	}
	header[cseIV] = base64.StdEncoding.EncodeToString(iv)
	header[cseAlgorithm] = DareHmacSha256

	if closer, ok := r.(io.Closer); ok {
		type readCloser struct {
			io.Reader
			io.Closer
		}
		return readCloser{reader, closer}, nil
	}
	return ioutil.NopCloser(reader), nil
}

func (d dareHmacSha256) Open(header map[string]string, r io.Reader) (io.ReadCloser, error) {
	if header[cseAlgorithm] != DareHmacSha256 {
		return nil, errors.New("unexpected encryption algorithm")
	}
	iv, err := base64.StdEncoding.DecodeString(header[cseIV])
	if err != nil {
		return nil, err
	}
	mac := hmac.New(sha256.New, d[:])
	mac.Write(iv)
	reader, err := sio.DecryptReader(r, sio.Config{Key: mac.Sum(nil)})
	if err != nil {
		return nil, err
	}
	if closer, ok := r.(io.Closer); ok {
		type readCloser struct {
			io.Reader
			io.Closer
		}
		return readCloser{reader, closer}, nil
	}
	return ioutil.NopCloser(reader), nil
}

func (d dareHmacSha256) Overhead(size int64) int64 {
	// See https://github.com/minio/sio/blob/master/DARE.md#3-package-format
	encSize := (size / (64 * 1024)) * (64*1024 + 32)
	if mod := size % (64 * 1024); mod > 0 {
		encSize += mod + 32
	}
	return encSize
}
