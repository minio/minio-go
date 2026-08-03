/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2026 MinIO, Inc.
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

package credentials

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// probeWaitBound caps every cross-goroutine wait in the caller-context
// tests so a wiring regression fails in seconds instead of hanging to the
// package timeout.
const probeWaitBound = 10 * time.Second

// retrieveWithin bounds a synchronous credential retrieval so a context
// wiring regression fails in seconds instead of hanging to the package
// timeout. On timeout the spawned goroutine is abandoned — it stays
// blocked in the provider for the rest of the process, which is
// acceptable for a test that is already failing.
func retrieveWithin(t *testing.T, what string, fn func() error) error {
	t.Helper()
	errCh := make(chan error, 1)
	go func() { errCh <- fn() }()
	select {
	case err := <-errCh:
		return err
	case <-time.After(3 * probeWaitBound):
		t.Fatalf("timed out waiting for %s", what)
		return nil
	}
}

// newCancelProbeServer starts an httptest server whose handler signals the
// first request's arrival on the returned channel and then blocks until
// the request context ends or the test finishes; cancel runs once the
// request arrives (or after probeWaitBound when it never does, so the
// retrieval under test still returns and the caller's arrival guard
// reports the miss). The arrival signal is a sync.Once close: a
// transport-level retry must not panic on a second request. The
// end-of-test release keeps the server Close from waiting on the handler
// when a regression leaves the request context un-canceled.
func newCancelProbeServer(t *testing.T, cancel context.CancelFunc) (*httptest.Server, <-chan struct{}) {
	t.Helper()
	requestArrived := make(chan struct{})
	var arrivedOnce sync.Once
	handlerDone := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		arrivedOnce.Do(func() { close(requestArrived) })
		select {
		case <-r.Context().Done():
		case <-handlerDone:
		}
	}))
	t.Cleanup(server.Close)
	t.Cleanup(func() { close(handlerDone) })
	go func() {
		select {
		case <-requestArrived:
		case <-time.After(probeWaitBound):
		}
		cancel()
	}()
	return server, requestArrived
}

// requireRequestArrived fails the test when the probe server never saw
// the request the retrieval was expected to send.
func requireRequestArrived(t *testing.T, requestArrived <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-requestArrived:
	default:
		t.Fatalf("timed out waiting for %s to arrive", what)
	}
}
