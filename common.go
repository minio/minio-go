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
	"crypto/md5"
	"encoding/base64"
	"io"
	"strings"
)

// find if prefix is case insensitive
func isPrefixCaseInsensitive(s, pfx string) bool {
	if len(pfx) > len(s) {
		return false
	}
	shead := s[:len(pfx)]
	if shead == pfx {
		return true
	}
	shead = strings.ToLower(shead)
	return shead == pfx || shead == strings.ToLower(pfx)
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
