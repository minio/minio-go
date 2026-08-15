/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2024-2026 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * SPDX-License-Identifier: Apache-2.0
 */

package minio

import (
	"strings"
	"testing"
	"time"
)

func TestUnsupportedRDMAOptions(t *testing.T) {
	// An empty-but-allocated map is not a request for anything, and callers
	// that copy options around produce them routinely -- rejecting those would
	// refuse uploads that ask for nothing unsupported.
	t.Run("empty maps are not set", func(t *testing.T) {
		opts := PutObjectOptions{
			UserMetadata: map[string]string{},
			UserTags:     map[string]string{},
		}
		if got := unsupportedRDMAOptions(opts); len(got) != 0 {
			t.Fatalf("expected none, got %v", got)
		}
	})

	// Transfer shaping does not change what is stored, so it stays permitted.
	t.Run("transfer shaping is permitted", func(t *testing.T) {
		opts := PutObjectOptions{
			PartSize:              64 << 20,
			NumThreads:            8,
			ConcurrentStreamParts: true,
			ContentType:           "application/octet-stream",
		}
		if got := unsupportedRDMAOptions(opts); len(got) != 0 {
			t.Fatalf("expected none, got %v", got)
		}
	})

	for _, tc := range []struct {
		name string
		opts PutObjectOptions
		want string
	}{
		{"metadata", PutObjectOptions{UserMetadata: map[string]string{"k": "v"}}, "UserMetadata"},
		{"tags", PutObjectOptions{UserTags: map[string]string{"k": "v"}}, "UserTags"},
		{"legal hold", PutObjectOptions{LegalHold: LegalHoldEnabled}, "LegalHold"},
		{"retention mode", PutObjectOptions{Mode: Governance}, "Mode"},
		{"retain until", PutObjectOptions{RetainUntilDate: time.Now()}, "RetainUntilDate"},
		{"storage class", PutObjectOptions{StorageClass: "REDUCED_REDUNDANCY"}, "StorageClass"},
		{"cache control", PutObjectOptions{CacheControl: "no-cache"}, "CacheControl"},
		{"content encoding", PutObjectOptions{ContentEncoding: "gzip"}, "ContentEncoding"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := unsupportedRDMAOptions(tc.opts)
			if len(got) != 1 || got[0] != tc.want {
				t.Fatalf("got %v, want [%s]", got, tc.want)
			}
			err := checkRDMAOptions(tc.opts)
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error does not name the option: %v", err)
			}
		})
	}

	// Several at once must all be reported, so a caller fixes them in one go.
	t.Run("reports every offender", func(t *testing.T) {
		opts := PutObjectOptions{
			UserMetadata: map[string]string{"k": "v"},
			LegalHold:    LegalHoldEnabled,
			CacheControl: "no-cache",
		}
		if got := unsupportedRDMAOptions(opts); len(got) != 3 {
			t.Fatalf("expected 3, got %v", got)
		}
	})
}
