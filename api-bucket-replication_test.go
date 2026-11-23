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

package minio

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/minio/minio-go/v7/pkg/replication"
)

// TestRemoveBucketReplicationResponse verifies that both replication removal
// entry points accept HTTP 200 and 204 while preserving AccessDenied errors.
func TestRemoveBucketReplicationResponse(t *testing.T) {
	for _, operation := range []struct {
		name string
		call func(*Client) error
	}{
		{"RemoveBucketReplication", func(c *Client) error {
			return c.RemoveBucketReplication(context.Background(), "bucket")
		}},
		{"SetBucketReplicationEmpty", func(c *Client) error {
			return c.SetBucketReplication(context.Background(), "bucket", replication.Config{})
		}},
	} {
		t.Run(operation.name, func(t *testing.T) {
			for _, status := range []int{http.StatusOK, http.StatusNoContent, http.StatusForbidden} {
				t.Run(http.StatusText(status), func(t *testing.T) {
					srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.Method != http.MethodDelete || r.URL.Path != "/bucket/" || !r.URL.Query().Has("replication") {
							t.Errorf("unexpected request: %s %s", r.Method, r.URL)
							w.WriteHeader(http.StatusBadRequest)
							return
						}
						w.WriteHeader(status)
						if status == http.StatusForbidden {
							w.Write([]byte(`<Error><Code>AccessDenied</Code><Message>Access Denied.</Message></Error>`))
						}
					}))
					defer srv.Close()

					clnt, err := New(srv.Listener.Addr().String(), &Options{Region: "us-east-1"})
					if err != nil {
						t.Fatal(err)
					}
					err = operation.call(clnt)
					if status == http.StatusForbidden {
						if response := ToErrorResponse(err); response.Code != "AccessDenied" || response.StatusCode != status {
							t.Fatalf("expected AccessDenied with status %d, got %v", status, err)
						}
					} else if err != nil {
						t.Fatalf("expected success for status %d, got %v", status, err)
					}
				})
			}
		})
	}
}
