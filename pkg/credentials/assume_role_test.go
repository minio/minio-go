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
	"strconv"
	"testing"
)

func TestSTSAssumeRoleDurationSeconds(t *testing.T) {
	testCases := []struct {
		requested int
		want      string
		wantSent  bool
	}{
		{requested: 0, wantSent: false},
		{requested: -1, want: "-1", wantSent: true},
		{requested: 900, want: "900", wantSent: true},
		{requested: 1200, want: "1200", wantSent: true},
		{requested: 3600, want: "3600", wantSent: true},
		{requested: 7200, want: "7200", wantSent: true},
	}
	for _, tc := range testCases {
		t.Run(strconv.Itoa(tc.requested), func(t *testing.T) {
			var got string
			var sent bool
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := r.ParseForm(); err != nil {
					t.Errorf("parse form: %v", err)
				}
				sent = r.PostForm.Has("DurationSeconds")
				got = r.PostForm.Get("DurationSeconds")
				w.WriteHeader(http.StatusBadRequest)
			}))
			defer server.Close()

			_, _ = getAssumeRoleCredentials(context.Background(), server.Client(), server.URL, STSAssumeRoleOptions{
				AccessKey:       "access",
				SecretKey:       "secret",
				DurationSeconds: tc.requested,
			})
			if sent != tc.wantSent {
				t.Fatalf("DurationSeconds sent = %v, want %v", sent, tc.wantSent)
			}
			if got != tc.want {
				t.Fatalf("DurationSeconds = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestSTSAssumeRoleCallerContextCancel verifies that the caller context
// carried by CredContext cancels an in-flight AssumeRole request.
func TestSTSAssumeRoleCallerContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server, requestArrived := newCancelProbeServer(t, cancel)

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
	requireRequestArrived(t, requestArrived, "the AssumeRole request")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Expected context.Canceled, got %v", err)
	}
}
