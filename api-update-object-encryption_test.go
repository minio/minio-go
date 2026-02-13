/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2025-2026 MinIO, Inc.
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
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/minio/minio-go/v7/pkg/credentials"
)

func TestUpdateObjectEncryptionXMLMarshal(t *testing.T) {
	req := updateObjectEncryptionRequest{
		XMLNS: "http://s3.amazonaws.com/doc/2006-03-01/",
		SSEKMS: &updateObjectEncryptionSSEKMS{
			BucketKeyEnabled: true,
			KMSKeyArn:        "my-minio-key",
		},
	}

	data, err := xml.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal XML: %v", err)
	}

	// Verify we can unmarshal back.
	var decoded updateObjectEncryptionRequest
	if err := xml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal XML: %v", err)
	}
	if decoded.SSEKMS == nil {
		t.Fatal("Expected SSE-KMS element to be present")
	}
	if decoded.SSEKMS.KMSKeyArn != "my-minio-key" {
		t.Fatalf("Expected KMSKeyArn 'my-minio-key', got %q", decoded.SSEKMS.KMSKeyArn)
	}
	if !decoded.SSEKMS.BucketKeyEnabled {
		t.Fatal("Expected BucketKeyEnabled to be true")
	}
}

func TestUpdateObjectEncryptionXMLMarshalNoBucketKey(t *testing.T) {
	req := updateObjectEncryptionRequest{
		XMLNS: "http://s3.amazonaws.com/doc/2006-03-01/",
		SSEKMS: &updateObjectEncryptionSSEKMS{
			KMSKeyArn: "my-minio-key",
		},
	}

	data, err := xml.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal XML: %v", err)
	}

	var decoded updateObjectEncryptionRequest
	if err := xml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal XML: %v", err)
	}
	if decoded.SSEKMS.BucketKeyEnabled {
		t.Fatal("Expected BucketKeyEnabled to be false (omitted)")
	}
}

func TestUpdateObjectEncryptionSuccess(t *testing.T) {
	var capturedMethod string
	var capturedPath string
	var capturedQuery url.Values
	var capturedBody []byte

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		capturedQuery = r.URL.Query()
		capturedBody, _ = io.ReadAll(r.Body)
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
		t.Fatalf("Failed to create client: %v", err)
	}

	opts := UpdateObjectEncryptionOptions{
		KMSKeyArn:        "my-minio-key",
		BucketKeyEnabled: true,
		VersionID:        "test-version-id",
	}

	err = client.UpdateObjectEncryption(context.Background(), "mybucket", "myobject", opts)
	if err != nil {
		t.Fatalf("UpdateObjectEncryption failed: %v", err)
	}

	// Verify request method.
	if capturedMethod != http.MethodPut {
		t.Fatalf("Expected PUT, got %s", capturedMethod)
	}

	// Verify request path.
	if capturedPath != "/mybucket/myobject" {
		t.Fatalf("Expected path '/mybucket/myobject', got %q", capturedPath)
	}

	// Verify query parameters.
	if _, ok := capturedQuery["encryption"]; !ok {
		t.Fatal("Expected 'encryption' query parameter")
	}
	if capturedQuery.Get("versionId") != "test-version-id" {
		t.Fatalf("Expected versionId 'test-version-id', got %q", capturedQuery.Get("versionId"))
	}

	// Verify XML body.
	var body updateObjectEncryptionRequest
	if err := xml.Unmarshal(capturedBody, &body); err != nil {
		t.Fatalf("Failed to unmarshal request body: %v", err)
	}
	if body.SSEKMS == nil {
		t.Fatal("Expected SSE-KMS element in request body")
	}
	if body.SSEKMS.KMSKeyArn != "my-minio-key" {
		t.Fatalf("Expected KMSKeyArn 'my-minio-key', got %q", body.SSEKMS.KMSKeyArn)
	}
	if !body.SSEKMS.BucketKeyEnabled {
		t.Fatal("Expected BucketKeyEnabled to be true in request body")
	}
}

func TestUpdateObjectEncryptionNoVersionID(t *testing.T) {
	var capturedQuery url.Values

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.Query()
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
		t.Fatalf("Failed to create client: %v", err)
	}

	err = client.UpdateObjectEncryption(context.Background(), "mybucket", "myobject", UpdateObjectEncryptionOptions{
		KMSKeyArn: "my-key",
	})
	if err != nil {
		t.Fatalf("UpdateObjectEncryption failed: %v", err)
	}

	// versionId should not be present when not specified.
	if capturedQuery.Get("versionId") != "" {
		t.Fatalf("Expected no versionId query parameter, got %q", capturedQuery.Get("versionId"))
	}
}

func TestUpdateObjectEncryptionServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><Error><Code>InvalidRequest</Code><Message>The encryption type change is not supported</Message><Resource>/mybucket/myobject</Resource><RequestId>test-req-id</RequestId></Error>`))
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
		t.Fatalf("Failed to create client: %v", err)
	}

	err = client.UpdateObjectEncryption(context.Background(), "mybucket", "myobject", UpdateObjectEncryptionOptions{
		KMSKeyArn: "my-key",
	})
	if err == nil {
		t.Fatal("Expected error for 400 response, got nil")
	}

	errResp := ToErrorResponse(err)
	if errResp.Code != "InvalidRequest" {
		t.Fatalf("Expected error code 'InvalidRequest', got %q", errResp.Code)
	}
}

func TestUpdateObjectEncryptionInvalidBucket(t *testing.T) {
	client, err := New("localhost:9000", &Options{
		Creds:  credentials.NewStaticV4("accesskey", "secretkey", ""),
		Secure: false,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	err = client.UpdateObjectEncryption(context.Background(), "", "myobject", UpdateObjectEncryptionOptions{
		KMSKeyArn: "my-key",
	})
	if err == nil {
		t.Fatal("Expected error for empty bucket name, got nil")
	}
}

func TestUpdateObjectEncryptionInvalidObject(t *testing.T) {
	client, err := New("localhost:9000", &Options{
		Creds:  credentials.NewStaticV4("accesskey", "secretkey", ""),
		Secure: false,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	err = client.UpdateObjectEncryption(context.Background(), "mybucket", "", UpdateObjectEncryptionOptions{
		KMSKeyArn: "my-key",
	})
	if err == nil {
		t.Fatal("Expected error for empty object name, got nil")
	}
}

func TestUpdateObjectEncryptionEmptyKMSKeyArn(t *testing.T) {
	client, err := New("localhost:9000", &Options{
		Creds:  credentials.NewStaticV4("accesskey", "secretkey", ""),
		Secure: false,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	err = client.UpdateObjectEncryption(context.Background(), "mybucket", "myobject", UpdateObjectEncryptionOptions{
		KMSKeyArn: "",
	})
	if err == nil {
		t.Fatal("Expected error for empty KMSKeyArn, got nil")
	}

	errResp := ToErrorResponse(err)
	if errResp.Code != InvalidArgument {
		t.Fatalf("Expected error code %q, got %q", InvalidArgument, errResp.Code)
	}
}
