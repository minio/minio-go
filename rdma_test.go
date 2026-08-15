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

// The size guards run before c.rdma(), so an out-of-range size must be
// reported as such even on a host with no RDMA device. That ordering is the
// point: a negative size would otherwise wrap when converted to C.size_t, and
// the conversion happens on the far side of the guard.
func rdmaGuardTestClient(t *testing.T) *Client {
	t.Helper()
	c, err := New("play.min.io", &Options{Creds: nil})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestPutObjectRDMARejectsOutOfRangeSizes(t *testing.T) {
	c := rdmaGuardTestClient(t)
	for _, size := range []int{-1, 1 << 32, 1<<32 + 1} {
		_, err := c.putObjectRDMA(context.Background(), "bucket", "object",
			PutObjectOptions{RDMABufferSize: size})
		if err == nil {
			t.Fatalf("size %d: expected an error", size)
		}
		if !strings.Contains(err.Error(), "RDMA put: buffer size") {
			t.Fatalf("size %d: guard did not reject it, got %v", size, err)
		}
	}
}

func TestGetObjectRDMARejectsOutOfRangeSizes(t *testing.T) {
	c := rdmaGuardTestClient(t)
	for _, size := range []int{-1, 1 << 32, 1<<32 + 1} {
		_, err := c.getObjectRDMA(context.Background(), "bucket", "object",
			GetObjectOptions{RDMABufferSize: size})
		if err == nil {
			t.Fatalf("size %d: expected an error", size)
		}
		if !strings.Contains(err.Error(), "RDMA get: buffer size") {
			t.Fatalf("size %d: guard did not reject it, got %v", size, err)
		}
	}
}

// 0 and the largest descriptor-addressable size must clear the guard. They
// fail later for want of a device, which is what proves the guard let them by.
func TestRDMAGuardsAcceptTheAddressableRange(t *testing.T) {
	c := rdmaGuardTestClient(t)
	for _, size := range []int{0, int(maxRDMABufferSize)} {
		_, putErr := c.putObjectRDMA(context.Background(), "bucket", "object",
			PutObjectOptions{RDMABufferSize: size})
		if putErr != nil && strings.Contains(putErr.Error(), "RDMA put: buffer size") {
			t.Fatalf("size %d: guard rejected an addressable size: %v", size, putErr)
		}

		_, getErr := c.getObjectRDMA(context.Background(), "bucket", "object",
			GetObjectOptions{RDMABufferSize: size})
		if getErr != nil && strings.Contains(getErr.Error(), "RDMA get: buffer size") {
			t.Fatalf("size %d: guard rejected an addressable size: %v", size, getErr)
		}
	}
}

func TestMaxRDMABufferSizeIsThe32BitWindow(t *testing.T) {
	if maxRDMABufferSize != 1<<32-1 {
		t.Fatalf("maxRDMABufferSize = %d, want %d", maxRDMABufferSize, int64(1<<32-1))
	}
}
