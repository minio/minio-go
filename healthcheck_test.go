/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2021 MinIO, Inc.
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
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthCheck(t *testing.T) {
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// New - instantiate minio client with options
	clnt, err := New(srv.Listener.Addr().String(), &Options{
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	hcancel, err := clnt.HealthCheck(1 * time.Second)
	if err != nil {
		t.Fatal(err)
	}

	probeBucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "probe-health-")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	clnt.BucketExists(ctx, probeBucketName)

	if !clnt.IsOffline() {
		t.Fatal("Expected offline but found online")
	}

	srv.Start()
	time.Sleep(2 * time.Second)

	if clnt.IsOffline() {
		t.Fatal("Expected online but found offline")
	}

	hcancel() // healthcheck is canceled.

	if !clnt.IsOnline() {
		t.Fatal("Expected online but found offline")
	}
}
