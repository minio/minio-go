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
	"regexp"
	"strings"
	"unicode/utf8"
)

// isValidBucketName - verify bucket name in accordance with
//  - http://docs.aws.amazon.com/AmazonS3/latest/dev/UsingBucket.html
func isValidBucketName(bucketName string) bool {
	if strings.TrimSpace(bucketName) == "" {
		return false
	}
	if len(bucketName) < 3 || len(bucketName) > 63 {
		return false
	}
	if bucketName[0] == '.' || bucketName[len(bucketName)-1] == '.' {
		return false
	}
	if match, _ := regexp.MatchString("\\.\\.", bucketName); match == true {
		return false
	}
	// We don't support bucketNames with '.' in them
	match, _ := regexp.MatchString("^[a-zA-Z][a-zA-Z0-9\\-]+[a-zA-Z0-9]$", bucketName)
	return match
}

// isValidObjectName - verify object name in accordance with
//   - http://docs.aws.amazon.com/AmazonS3/latest/dev/UsingMetadata.html
func isValidObjectName(objectName string) bool {
	if strings.TrimSpace(objectName) == "" {
		return false
	}
	if len(objectName) > 1024 || len(objectName) == 0 {
		return false
	}
	if !utf8.ValidString(objectName) {
		return false
	}
	return true
}
