/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2024-2026 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * SPDX-License-Identifier: Apache-2.0
 */

package minio

import (
	"fmt"
	"strings"
	"time"
)

// unsupportedRDMAOptions names the options set on opts that the RDMA path
// cannot carry.
//
// The C entry point takes a bucket, an object, a buffer or reader, and nothing
// else, so anything that changes what the stored object *is* would be dropped
// on the floor. Silently storing an object without the metadata, tags,
// encryption or retention the caller asked for is worse than refusing the
// upload, because nothing downstream can tell the difference.
//
// Transfer-shaping options are deliberately absent from this list: part size
// and concurrency change how the bytes move, not what is stored, and
// libminiocpp chooses its own. ContentType is absent too -- it is not applied
// either, which is a real gap, but rejecting it would refuse the common case
// of a caller that sets it for the HTTP path and falls through to RDMA.
func unsupportedRDMAOptions(opts PutObjectOptions) []string {
	var set []string
	add := func(cond bool, name string) {
		if cond {
			set = append(set, name)
		}
	}
	add(len(opts.UserMetadata) > 0, "UserMetadata")
	add(len(opts.UserTags) > 0, "UserTags")
	add(opts.ServerSideEncryption != nil, "ServerSideEncryption")
	add(opts.Mode != "", "Mode")
	add(!opts.RetainUntilDate.IsZero(), "RetainUntilDate")
	add(opts.LegalHold != "", "LegalHold")
	add(opts.StorageClass != "", "StorageClass")
	add(opts.WebsiteRedirectLocation != "", "WebsiteRedirectLocation")
	add(opts.ContentEncoding != "", "ContentEncoding")
	add(opts.ContentDisposition != "", "ContentDisposition")
	add(opts.ContentLanguage != "", "ContentLanguage")
	add(opts.CacheControl != "", "CacheControl")
	add(!opts.Expires.Equal(time.Time{}), "Expires")
	add(opts.Checksum.IsSet(), "Checksum")
	return set
}

// checkRDMAOptions rejects an RDMA upload whose options cannot be honored.
func checkRDMAOptions(opts PutObjectOptions) error {
	if unsupported := unsupportedRDMAOptions(opts); len(unsupported) > 0 {
		return fmt.Errorf(
			"RDMA put: %s cannot be applied on the RDMA path; unset %s or omit RDMABuffer to use the HTTP path",
			strings.Join(unsupported, ", "),
			map[bool]string{true: "it", false: "them"}[len(unsupported) == 1])
	}
	return nil
}
