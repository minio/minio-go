/*
 * Minio Cloud Storage, (C) 2016 Minio, Inc.
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
 *
 */

package minio

import (
	"errors"
)

// Used for adding entry to the object cache.
// Implements io.WriteCloser
type cappedWriter struct {
	offset int64
	cap    int64
	buffer []byte
}

// Write implements a limited writer, returns error.
// if the writes go beyond allocated size.
func (c *cappedWriter) Write(b []byte) (n int, err error) {
	if c.offset+int64(len(b)) > c.cap {
		return 0, errors.New("excess data")
	}
	n = copy(c.buffer[int(c.offset):int(c.offset)+len(b)], b)
	c.offset = c.offset + int64(n)
	return n, nil
}

func (c *cappedWriter) Len() int {
	return len(c.buffer)
}

func (c *cappedWriter) GetBytes(offset int64) []byte {
	return c.buffer[:offset]
}
