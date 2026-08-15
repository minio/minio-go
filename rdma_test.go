//go:build rdma

/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2024-2026 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * SPDX-License-Identifier: Apache-2.0
 */

package minio

import (
	"context"
	"strings"
	"testing"
)

// The accepted range is asserted against the predicate rather than by calling
// the entry points: on a host that does have a device, a size that clears the
// guard goes on to the native call, and RDMABuffer is nil here. Handing
// libminiocpp a nil pointer with a 4 GiB length is not something a unit test
// should do to find out that the guard let it by.
func TestRDMABufferSizeInRange(t *testing.T) {
	for _, tc := range []struct {
		size int
		want bool
	}{
		{-1, false},
		{0, true},
		{1, true},
		{1<<32 - 2, true},
		{int(maxRDMABufferSize), true}, // 1<<32 - 1
		{1 << 32, false},
		{1<<32 + 1, false},
	} {
		if got := rdmaBufferSizeInRange(tc.size); got != tc.want {
			t.Errorf("rdmaBufferSizeInRange(%d) = %v, want %v", tc.size, got, tc.want)
		}
	}
}

func TestMaxRDMABufferSizeIsThe32BitWindow(t *testing.T) {
	if maxRDMABufferSize != 1<<32-1 {
		t.Fatalf("maxRDMABufferSize = %d, want %d", maxRDMABufferSize, int64(1<<32-1))
	}
}

// A rejected size must be reported by the guard, which runs before c.rdma()
// and before the conversion to C.size_t -- the conversion is what would wrap a
// negative size into a huge length. Rejection is therefore safe to drive
// through the real entry points on any host, device or not.
func TestRDMAEntryPointsRejectOutOfRangeSizes(t *testing.T) {
	c, err := New("play.min.io", &Options{Creds: nil})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	for _, size := range []int{-1, 1 << 32, 1<<32 + 1} {
		_, putErr := c.putObjectRDMA(context.Background(), "bucket", "object",
			PutObjectOptions{RDMABufferSize: size})
		if putErr == nil {
			t.Fatalf("put size %d: expected an error, got nil", size)
		}
		if !strings.Contains(putErr.Error(), "RDMA put: buffer size") {
			t.Fatalf("put size %d: want the guard's error, got %v", size, putErr)
		}

		_, getErr := c.getObjectRDMA(context.Background(), "bucket", "object",
			GetObjectOptions{RDMABufferSize: size})
		if getErr == nil {
			t.Fatalf("get size %d: expected an error, got nil", size)
		}
		if !strings.Contains(getErr.Error(), "RDMA get: buffer size") {
			t.Fatalf("get size %d: want the guard's error, got %v", size, getErr)
		}
	}
}
