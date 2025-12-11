/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2015-2025 MinIO, Inc.
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

package set

import (
	"github.com/tinylib/msgp/msgp"
	"github.com/tinylib/msgp/msgp/setof"
)

// EncodeMsg encodes the message to the writer.
// Values are stored as a slice of strings or nil.
func (s StringSet) EncodeMsg(writer *msgp.Writer) error {
	return setof.StringSorted(s).EncodeMsg(writer)
}

// MarshalMsg encodes the message to the bytes.
// Values are stored as a slice of strings or nil.
func (s StringSet) MarshalMsg(bytes []byte) ([]byte, error) {
	return setof.StringSorted(s).MarshalMsg(bytes)
}

// DecodeMsg decodes the message from the reader.
func (s *StringSet) DecodeMsg(reader *msgp.Reader) error {
	var ss setof.String
	if err := ss.DecodeMsg(reader); err != nil {
		return err
	}
	*s = StringSet(ss)
	return nil
}

// UnmarshalMsg decodes the message from the bytes.
func (s *StringSet) UnmarshalMsg(bytes []byte) ([]byte, error) {
	var ss setof.String
	bytes, err := ss.UnmarshalMsg(bytes)
	if err != nil {
		return nil, err
	}
	*s = StringSet(ss)
	return bytes, nil
}

// Msgsize returns the maximum size of the message.
func (s StringSet) Msgsize() int {
	return setof.String(s).Msgsize()
}

// MarshalBinary encodes the receiver into a binary form and returns the result.
func (s StringSet) MarshalBinary() ([]byte, error) {
	return s.MarshalMsg(nil)
}

// AppendBinary appends the binary representation of itself to the end of b
func (s StringSet) AppendBinary(b []byte) ([]byte, error) {
	return s.MarshalMsg(b)
}

// UnmarshalBinary decodes the binary representation of itself from b
func (s *StringSet) UnmarshalBinary(b []byte) error {
	_, err := s.UnmarshalMsg(b)
	return err
}
