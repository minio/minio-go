//go:build rdma

/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2024-2026 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * SPDX-License-Identifier: Apache-2.0
 */

package minio

// A file carrying //export may hold declarations in its cgo preamble but no
// definitions, so the trampoline that reaches this lives in rdma_stream.go.
// The blank line below matters: only the comment block touching `import "C"`
// is the preamble, and this prose is not C.

// #include <sys/types.h>
import "C"

import (
	"io"
	"runtime/cgo"
	"unsafe"
)

// rdmaStreamSource is what the C read callback pulls from.
//
// The read error is kept here rather than returned through C: the callback can
// only say "failed", and losing the reason turns an ordinary error on the
// caller's stream into an opaque upload failure.
type rdmaStreamSource struct {
	reader io.Reader
	err    error
}

//export minioRDMAReadGo
func minioRDMAReadGo(userdata unsafe.Pointer, buf *C.char, size C.size_t) C.ssize_t {
	if userdata == nil {
		return -1
	}
	src, ok := (*(*cgo.Handle)(userdata)).Value().(*rdmaStreamSource)
	if !ok {
		return -1
	}
	if size == 0 {
		return 0
	}

	// Alias the C buffer rather than copying: Read then writes straight into
	// the part buffer libminiocpp registered for RDMA.
	dst := unsafe.Slice((*byte)(unsafe.Pointer(buf)), int(size))

	// An io.Reader may return fewer bytes than asked for without being at EOF,
	// and libminiocpp reads a short return as end-of-part. Filling only
	// partially mid-stream would upload a short part and truncate the object
	// with no error anywhere, so keep pulling until the buffer is full or the
	// stream genuinely ends.
	total := 0
	for total < len(dst) {
		n, err := src.reader.Read(dst[total:])
		total += n
		if err == io.EOF {
			break
		}
		if err != nil {
			src.err = err
			return -1
		}
		if n == 0 {
			// Reader made no progress and reported no error; treat as EOF
			// rather than spinning.
			break
		}
	}
	return C.ssize_t(total)
}
