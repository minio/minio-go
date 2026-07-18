/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2015-2026 MinIO, Inc.
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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/minio/minio-go/v7/pkg/credentials"
)

// TestNewHookSeekability verifies that the reader returned by newHook
// implements io.Seeker if and only if the source does, and never
// implements io.Closer. Regression test for
// https://github.com/minio/minio-go/issues/2078: a fake Seek on a
// non-seekable source silently defeated executeMethod's no-retry path
// and caused empty-body retries.
func TestNewHookSeekability(t *testing.T) {
	payload := []byte("hello world")
	progress := bytes.NewBuffer(nil)

	tests := []struct {
		name       string
		source     io.Reader
		hook       io.Reader
		wantSeeker bool
	}{
		{"non-seekable source, nil hook", bytes.NewBuffer(payload), nil, false},
		{"non-seekable source, with hook", bytes.NewBuffer(payload), progress, false},
		{"seekable source, nil hook", bytes.NewReader(payload), nil, true},
		{"seekable source, with hook", bytes.NewReader(payload), progress, true},
		{"non-seekable ReadCloser source", io.NopCloser(bytes.NewReader(payload)), nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newHook(tt.source, tt.hook)
			if _, ok := r.(io.Seeker); ok != tt.wantSeeker {
				t.Errorf("newHook(%T, %T) seekable = %v, want %v", tt.source, tt.hook, ok, tt.wantSeeker)
			}
			// The wrapper must never expose Close: executeMethod
			// closes io.Closer bodies, and a caller-supplied reader
			// must not be closed by the SDK.
			if _, ok := r.(io.Closer); ok {
				t.Errorf("newHook(%T, %T) implements io.Closer, must not", tt.source, tt.hook)
			}
		})
	}
}

// TestHookReadSeekerSeek verifies seeking still works on a seekable
// source and keeps a seekable hook in sync.
func TestHookReadSeekerSeek(t *testing.T) {
	payload := []byte("0123456789")
	source := bytes.NewReader(payload)
	hook := bytes.NewReader(payload)

	rs, ok := newHook(source, hook).(io.ReadSeeker)
	if !ok {
		t.Fatal("newHook over a seekable source must return an io.ReadSeeker")
	}
	buf := make([]byte, 4)
	if _, err := io.ReadFull(rs, buf); err != nil {
		t.Fatal(err)
	}
	n, err := rs.Seek(2, io.SeekStart)
	if err != nil {
		t.Fatalf("Seek: %v", err)
	}
	if n != 2 {
		t.Fatalf("Seek returned %d, want 2", n)
	}
	if _, err = io.ReadFull(rs, buf); err != nil {
		t.Fatal(err)
	}
	if got := string(buf); got != "2345" {
		t.Fatalf("read after seek = %q, want %q", got, "2345")
	}
	if m, _ := hook.Seek(0, io.SeekCurrent); m != 6 {
		t.Fatalf("hook position = %d, want 6", m)
	}
}

// TestHookReadSeekerSeekNonSeekableHook verifies Seek succeeds when the
// source is seekable but the hook is not: the hook sync is skipped
// rather than failing the seek. Progress readers are plain io.Readers,
// so this is the shape a retried upload with a progress bar takes.
func TestHookReadSeekerSeekNonSeekableHook(t *testing.T) {
	payload := []byte("0123456789")
	source := bytes.NewReader(payload)
	hook := bytes.NewBuffer(nil)

	rs, ok := newHook(source, hook).(io.ReadSeeker)
	if !ok {
		t.Fatal("newHook over a seekable source must return an io.ReadSeeker")
	}
	buf := make([]byte, 4)
	if _, err := io.ReadFull(rs, buf); err != nil {
		t.Fatal(err)
	}
	n, err := rs.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatalf("Seek with a non-seekable hook: %v", err)
	}
	if n != 0 {
		t.Fatalf("Seek returned %d, want 0", n)
	}
	if _, err := io.ReadFull(rs, buf); err != nil {
		t.Fatal(err)
	}
	if got := string(buf); got != "0123" {
		t.Fatalf("read after seek = %q, want %q", got, "0123")
	}
}

// stubSeeker is a ReadSeeker whose Seek reports a fixed offset and
// error, for driving the error paths of hookReadSeeker.Seek.
type stubSeeker struct {
	off int64
	err error
}

func (s *stubSeeker) Read([]byte) (int, error) { return 0, io.EOF }

func (s *stubSeeker) Seek(int64, int) (int64, error) { return s.off, s.err }

// TestHookReadSeekerSeekErrors pins the error paths of Seek: a
// directly-constructed wrapper over a non-seekable source errors
// instead of panicking, a failing source seek propagates, and a hook
// offset diverging from the source is reported.
func TestHookReadSeekerSeekErrors(t *testing.T) {
	t.Run("non-seekable source errors", func(t *testing.T) {
		hr := &hookReadSeeker{hookReader{source: bytes.NewBuffer(nil)}}
		if _, err := hr.Seek(0, io.SeekStart); err == nil {
			t.Fatal("Seek over a non-seekable source must error, not panic")
		}
	})
	t.Run("source seek failure propagates", func(t *testing.T) {
		wantErr := errors.New("seek failed")
		hr := &hookReadSeeker{hookReader{source: &stubSeeker{err: wantErr}}}
		n, err := hr.Seek(0, io.SeekStart)
		if !errors.Is(err, wantErr) || n != 0 {
			t.Fatalf("Seek = (%d, %v), want (0, %v)", n, err, wantErr)
		}
	})
	t.Run("hook offset mismatch is reported", func(t *testing.T) {
		source := bytes.NewReader([]byte("0123456789"))
		hr := &hookReadSeeker{hookReader{source: source, hook: &stubSeeker{off: 5}}}
		_, err := hr.Seek(2, io.SeekStart)
		if err == nil || !strings.Contains(err.Error(), "hook seeker sought to offset 5, expected source offset 2") {
			t.Fatalf("want offset-mismatch error, got %v", err)
		}
	})
}

// TestPutObjectRetryNonSeekable verifies retry behavior around
// retryable server errors: a non-seekable body must be sent exactly
// once and surface the server's error, while a seekable body is
// retried with the full body. Regression test for
// https://github.com/minio/minio-go/issues/2078.
func TestPutObjectRetryNonSeekable(t *testing.T) {
	var mu sync.Mutex
	attempts := make(map[string][]int) // object -> body bytes per attempt

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.WriteHeader(http.StatusOK)
			return
		}
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		attempts[r.URL.Path] = append(attempts[r.URL.Path], len(body))
		first := len(attempts[r.URL.Path]) == 1
		mu.Unlock()
		if first {
			// Retryable error on the first attempt for each object.
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?><Error><Code>SlowDown</Code><Message>Please reduce your request rate.</Message></Error>`)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	srv, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	client, err := New(srv.Host, &Options{
		Creds:  credentials.NewStaticV4("accesskey", "secretkey", ""),
		Secure: false,
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	payload := bytes.Repeat([]byte("x"), 1024)
	size := int64(len(payload))

	t.Run("non-seekable body fails fast with the server error", func(t *testing.T) {
		_, err := client.PutObject(context.Background(), "bucket", "non-seekable",
			bytes.NewBuffer(payload), size, PutObjectOptions{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if errResp := ToErrorResponse(err); errResp.Code != "SlowDown" {
			t.Fatalf("expected the server's SlowDown error, got %v", err)
		}
		if strings.Contains(err.Error(), "ContentLength") {
			t.Fatalf("transport error leaked instead of server error: %v", err)
		}
		mu.Lock()
		got := attempts["/bucket/non-seekable"]
		mu.Unlock()
		if len(got) != 1 {
			t.Fatalf("non-seekable body sent %d times %v, want exactly 1 attempt", len(got), got)
		}
	})

	t.Run("seekable body retries with the full body", func(t *testing.T) {
		_, err := client.PutObject(context.Background(), "bucket", "seekable",
			bytes.NewReader(payload), size, PutObjectOptions{})
		if err != nil {
			t.Fatalf("expected retry to succeed, got %v", err)
		}
		mu.Lock()
		got := attempts["/bucket/seekable"]
		mu.Unlock()
		if len(got) != 2 {
			t.Fatalf("seekable body sent %d times %v, want 2 attempts", len(got), got)
		}
		// The wire body carries aws-chunked signature framing on top
		// of the payload, so assert a lower bound plus an identical
		// resend rather than exact payload length.
		for i, n := range got {
			if n < len(payload) {
				t.Fatalf("attempt %d sent %d body bytes, want at least %d", i+1, n, len(payload))
			}
		}
		if got[0] != got[1] {
			t.Fatalf("retry sent %d body bytes, first attempt sent %d; retry must resend the identical body", got[1], got[0])
		}
	})

	t.Run("non-seekable body with progress hook fails fast", func(t *testing.T) {
		_, err := client.PutObject(context.Background(), "bucket", "non-seekable-progress",
			bytes.NewBuffer(payload), size, PutObjectOptions{Progress: bytes.NewBuffer(nil)})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if errResp := ToErrorResponse(err); errResp.Code != "SlowDown" {
			t.Fatalf("expected the server's SlowDown error, got %v", err)
		}
		mu.Lock()
		got := attempts["/bucket/non-seekable-progress"]
		mu.Unlock()
		if len(got) != 1 {
			t.Fatalf("non-seekable body sent %d times %v, want exactly 1 attempt", len(got), got)
		}
	})
}
