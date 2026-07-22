// Copyright 2024-2026 - MinIO, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// rdma-test exercises the unified minio-go RDMA path against a running
// MinIO server. Requires -tags=rdma at build time and libminiocpp.so on
// the host's library search path.
//
//   go build -tags=rdma -o rdma-test ./cmd/rdma-test
//   MINIO_ENDPOINT=coe01:9000 MINIO_ACCESS_KEY=... MINIO_SECRET_KEY=... ./rdma-test

package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"unsafe"

	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	testBucket = "rdma-test"
	testObject = "test-object-cpu"
	testSize   = 1 << 20 // 1 MiB
)

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	endpoint := envOr("MINIO_ENDPOINT", "coe01:9000")
	accessKey := envOr("MINIO_ACCESS_KEY", "minioadmin")
	secretKey := envOr("MINIO_SECRET_KEY", "minioadmin")

	fmt.Printf("endpoint=%s rdma_available=%v\n", endpoint, minio.IsRDMAAvailable())

	client, err := minio.New(endpoint, &minio.Options{
		Creds:      credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure:     false,
		EnableRDMA: true,
	})
	if err != nil {
		return fmt.Errorf("New: %w", err)
	}

	ctx := context.Background()

	exists, err := client.BucketExists(ctx, testBucket)
	if err != nil {
		return fmt.Errorf("BucketExists: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, testBucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("MakeBucket: %w", err)
		}
	}

	src := minio.AlignedBuffer(testSize)
	if src == nil {
		return fmt.Errorf("AlignedBuffer(%d) returned nil", testSize)
	}
	defer minio.FreeAlignedBuffer(src)
	srcSlice := unsafe.Slice((*byte)(src), testSize)
	for i := range srcSlice {
		srcSlice[i] = byte(i)
	}

	fmt.Print("PutObject (RDMA)... ")
	info, err := client.PutObject(ctx, testBucket, testObject, nil, 0, minio.PutObjectOptions{
		RDMABuffer:     src,
		RDMABufferSize: testSize,
	})
	if err != nil {
		return fmt.Errorf("PutObject: %w", err)
	}
	fmt.Printf("ok etag=%s size=%d checksum=%s\n", info.ETag, info.Size, info.ChecksumCRC64NVME)

	dst := minio.AlignedBuffer(testSize)
	if dst == nil {
		return fmt.Errorf("AlignedBuffer(%d) returned nil", testSize)
	}
	defer minio.FreeAlignedBuffer(dst)
	dstSlice := unsafe.Slice((*byte)(dst), testSize)

	fmt.Print("GetObject (RDMA)... ")
	obj, err := client.GetObject(ctx, testBucket, testObject, minio.GetObjectOptions{
		RDMABuffer:     dst,
		RDMABufferSize: testSize,
	})
	if err != nil {
		return fmt.Errorf("GetObject: %w", err)
	}
	stat, err := obj.Stat()
	if err != nil {
		return fmt.Errorf("Stat: %w", err)
	}
	fmt.Printf("ok size=%d\n", stat.Size)

	if !bytes.Equal(srcSlice, dstSlice) {
		return fmt.Errorf("FAIL: roundtrip data mismatch")
	}
	fmt.Println("PASS: roundtrip verified")
	return nil
}
