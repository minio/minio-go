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

package minio

import (
	"io"

	"context"
)

// PutObjectWithContext - creates an object in a bucket.Allows request cancellation.
func (c Client) PutObjectWithContext(ctx context.Context, bucketName, objectName string, reader io.Reader, contentType string) (n int64, err error) {
	// Size of the object.
	var size int64
	// Get reader size.
	size, err = getReaderSize(reader)
	if err != nil {
		return 0, err
	}
	metadata := make(map[string][]string)
	metadata["Content-Type"] = []string{contentType}

	return c.putObjectCommon(ctx, bucketName, objectName, reader, size, metadata, nil)
}
