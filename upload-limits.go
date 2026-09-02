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

package minio

import "math"

// UploadLimits overrides the upload limits the client enforces before sending
// a request. The defaults are the limits Amazon S3 imposes; only raise them
// when the remote endpoint is known to accept the larger values.
//
// A zero field means "use the default", so the zero UploadLimits behaves
// exactly like Amazon S3.
type UploadLimits struct {
	// MinPartSize is the smallest size allowed for a part that is not the
	// last part of a multipart upload. Defaults to 5 MiB.
	MinPartSize int64

	// MaxPartSize is the largest size allowed for a single part.
	// Defaults to 5 GiB.
	MaxPartSize int64

	// MaxPartsCount is the maximum number of parts in a single multipart
	// upload. Together with MaxPartSize this caps the object size the client
	// is willing to upload. Defaults to 10000.
	MaxPartsCount int64

	// MaxSinglePutObjectSize is the largest object the remote accepts in a
	// single PUT. Defaults to 5 GiB.
	//
	// Client.PutObject enforces this by switching to a multipart upload, or by
	// returning EntityTooLarge when PutObjectOptions.DisableMultipart is set.
	// Core.PutObject sends the PUT as given and does not check it.
	MaxSinglePutObjectSize int64
}

// UploadLimits returns the upload limits this client enforces, with any
// zero field resolved to its Amazon S3 default.
func (c *Client) UploadLimits() UploadLimits {
	return UploadLimits{
		MinPartSize:            c.limits.minPartSize(),
		MaxPartSize:            c.limits.maxPartSize(),
		MaxPartsCount:          c.limits.maxPartsCount(),
		MaxSinglePutObjectSize: c.limits.maxSinglePutObjectSize(),
	}
}

// Accessors resolve zero fields to their defaults, so a Client that was not
// built by New still sees the S3 limits.

func (l UploadLimits) minPartSize() int64 {
	if l.MinPartSize > 0 {
		return l.MinPartSize
	}
	return defaultMinPartSize
}

func (l UploadLimits) maxPartSize() int64 {
	if l.MaxPartSize > 0 {
		return l.MaxPartSize
	}
	return defaultMaxPartSize
}

func (l UploadLimits) maxPartsCount() int64 {
	if l.MaxPartsCount > 0 {
		return l.MaxPartsCount
	}
	return defaultMaxPartsCount
}

func (l UploadLimits) maxSinglePutObjectSize() int64 {
	if l.MaxSinglePutObjectSize > 0 {
		return l.MaxSinglePutObjectSize
	}
	return defaultMaxSinglePutObjectSize
}

// maxObjectSize is the largest object that can be uploaded as a multipart
// upload, ~48.83TiB with the default limits.
func (l UploadLimits) maxObjectSize() int64 {
	return l.maxPartSize() * l.maxPartsCount()
}

func (l UploadLimits) validate() error {
	for _, f := range []struct {
		name  string
		value int64
	}{
		{"MinPartSize", l.MinPartSize},
		{"MaxPartSize", l.MaxPartSize},
		{"MaxPartsCount", l.MaxPartsCount},
		{"MaxSinglePutObjectSize", l.MaxSinglePutObjectSize},
	} {
		if f.value < 0 {
			return errInvalidArgument("UploadLimits." + f.name + " cannot be negative")
		}
	}
	if l.minPartSize() > l.maxPartSize() {
		return errInvalidArgument("UploadLimits.MinPartSize cannot be larger than UploadLimits.MaxPartSize")
	}
	if l.maxPartSize() > math.MaxInt64/l.maxPartsCount() {
		return errInvalidArgument("UploadLimits.MaxPartSize multiplied by UploadLimits.MaxPartsCount overflows int64")
	}
	return nil
}
