/*
 * Minio Go Library for Amazon S3 Compatible Cloud Storage (C) 2016 Minio, Inc.
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
	"net/http"
	"time"
)

// CopyConditions - copy conditions.
type CopyConditions struct {
	conditions map[string]string
	// start and end offset (inclusive) of source object to be
	// copied.
	byteRangeStart int64
	byteRangeEnd   int64
}

// NewCopyConditions - Instantiate new list of conditions. Prefer to
// use this function as it initializes byte-range.
//
func NewCopyConditions() CopyConditions {
	return CopyConditions{
		conditions: make(map[string]string),
		// default values for byte-range indicating that they
		// are not provided by the user
		byteRangeStart: -1,
		byteRangeEnd:   -1,
	}
}

// SetMatchETag - set match etag.
func (c *CopyConditions) SetMatchETag(etag string) error {
	if etag == "" {
		return ErrInvalidArgument("ETag cannot be empty.")
	}
	c.conditions["x-amz-copy-source-if-match"] = etag
	return nil
}

// SetMatchETagExcept - set match etag except.
func (c *CopyConditions) SetMatchETagExcept(etag string) error {
	if etag == "" {
		return ErrInvalidArgument("ETag cannot be empty.")
	}
	c.conditions["x-amz-copy-source-if-none-match"] = etag
	return nil
}

// SetUnmodified - set unmodified time since.
func (c *CopyConditions) SetUnmodified(modTime time.Time) error {
	if modTime.IsZero() {
		return ErrInvalidArgument("Modified since cannot be empty.")
	}
	c.conditions["x-amz-copy-source-if-unmodified-since"] = modTime.Format(http.TimeFormat)
	return nil
}

// SetModified - set modified time since.
func (c *CopyConditions) SetModified(modTime time.Time) error {
	if modTime.IsZero() {
		return ErrInvalidArgument("Modified since cannot be empty.")
	}
	c.conditions["x-amz-copy-source-if-modified-since"] = modTime.Format(http.TimeFormat)
	return nil
}

// SetByteRange - set the start and end of the source object to be
// copied.
func (c *CopyConditions) SetByteRange(start, end int64) error {
	if start < 0 || end < start {
		return ErrInvalidArgument("Range start less than 0 or range end less than range start.")
	}
	if end-start+1 < 1 {
		return ErrInvalidArgument("Offset must refer to a non-zero range length.")
	}
	c.byteRangeEnd = end
	c.byteRangeStart = start
	return nil
}

func (c *CopyConditions) getRangeSize() int64 {
	if c.byteRangeStart < 0 {
		// only happens if byte-range was not set by user
		return 0
	}
	return c.byteRangeEnd - c.byteRangeStart + 1
}

func (c *CopyConditions) duplicate() *CopyConditions {
	r := NewCopyConditions()
	for k, v := range c.conditions {
		r.conditions[k] = v
	}
	r.byteRangeEnd, r.byteRangeStart = c.byteRangeEnd, c.byteRangeStart
	return &r
}
