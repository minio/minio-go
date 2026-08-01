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
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// retrieveWithin bounds a synchronous credential retrieval so a context
// wiring regression fails in seconds instead of hanging to the package
// timeout.
func retrieveWithin(t *testing.T, what string, fn func() error) error {
	t.Helper()
	errCh := make(chan error, 1)
	go func() { errCh <- fn() }()
	select {
	case err := <-errCh:
		return err
	case <-time.After(30 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
		return nil
	}
}

// TestSTSAssumeRoleCallerContextCancel verifies that the caller context
// carried by CredContext cancels an in-flight AssumeRole request.
func TestSTSAssumeRoleCallerContextCancel(t *testing.T) {
	requestArrived := make(chan struct{})
	// The unread POST body disables the server's client-disconnect
	// detection, so the handler needs an explicit release for Close.
	handlerDone := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(requestArrived)
		select {
		case <-r.Context().Done():
		case <-handlerDone:
		}
	}))
	defer server.Close()
	defer close(handlerDone)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-requestArrived:
		case <-time.After(10 * time.Second):
		}
		cancel()
	}()

	m := &STSAssumeRole{
		STSEndpoint: server.URL,
		Options: STSAssumeRoleOptions{
			AccessKey: "access",
			SecretKey: "secret",
		},
	}
	err := retrieveWithin(t, "the AssumeRole retrieval", func() error {
		_, err := m.RetrieveWithCredContext(&CredContext{Client: server.Client(), Context: ctx})
		return err
	})
	select {
	case <-requestArrived:
	default:
		t.Fatal("timed out waiting for the AssumeRole request to arrive")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Expected context.Canceled, got %v", err)
	}
}
