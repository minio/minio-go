/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2015-2024 MinIO, Inc.
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
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/policy"
)

func TestSuccessStatusIncludesAccepted(t *testing.T) {
	if !successStatus.Contains(http.StatusAccepted) {
		t.Fatal("expected 202 Accepted to be treated as a successful response")
	}
}

// TestRestoreObjectAccepts202 verifies that RestoreObject treats the
// documented AWS success response for POST ?restore (202 Accepted, empty
// body) as success, while a genuine error response still surfaces.
// Regression test for https://github.com/minio/minio-go/issues/2223.
func TestRestoreObjectAccepts202(t *testing.T) {
	var restoreCalls int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Query().Has("restore") {
			if atomic.AddInt64(&restoreCalls, 1) == 1 {
				w.WriteHeader(http.StatusAccepted)
				return
			}
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<Error><Code>RestoreAlreadyInProgress</Code><Message>Object restore is already in progress</Message><Resource>/bkt/obj</Resource><RequestId>REQ</RequestId></Error>`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	c, err := New(u.Host, &Options{
		Creds:  credentials.NewStaticV4("ak", "sk", ""),
		Secure: false,
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	req := RestoreRequest{}
	req.SetDays(1)

	if err := c.RestoreObject(context.Background(), "bkt", "obj", "", req); err != nil {
		t.Fatalf("first restore request: expected success on 202 Accepted, got: %v", err)
	}
	err = c.RestoreObject(context.Background(), "bkt", "obj", "", req)
	if err == nil {
		t.Fatal("second restore request: expected RestoreAlreadyInProgress error, got success")
	}
	errResp := ToErrorResponse(err)
	if errResp.Code != "RestoreAlreadyInProgress" || errResp.StatusCode != http.StatusConflict {
		t.Fatalf("second restore request: expected RestoreAlreadyInProgress error, got: %v", err)
	}
	if n := atomic.LoadInt64(&restoreCalls); n != 2 {
		t.Fatalf("expected exactly 2 restore requests (no retries), server saw %d", n)
	}
}

// TestTraceErrorsOnlySkipsSuccessStatuses verifies that errors-only tracing
// suppresses every successStatus member (200, 202, 204, 206) uniformly and
// still dumps a genuine error response.
func TestTraceErrorsOnlySkipsSuccessStatuses(t *testing.T) {
	var wantStatus int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(int(atomic.LoadInt32(&wantStatus)))
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	c, err := New(u.Host, &Options{
		Creds:  credentials.NewStaticV4("ak", "sk", ""),
		Secure: false,
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	c.TraceErrorsOnlyOn(&buf)

	doRequest := func(status int32) {
		t.Helper()
		atomic.StoreInt32(&wantStatus, status)
		req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := c.do(req)
		if err != nil {
			t.Fatalf("status %d: unexpected transport error: %v", status, err)
		}
		if err := resp.Body.Close(); err != nil {
			t.Fatalf("status %d: closing response body: %v", status, err)
		}
	}

	for _, code := range []int32{http.StatusOK, http.StatusAccepted, http.StatusNoContent, http.StatusPartialContent} {
		doRequest(code)
		if buf.Len() != 0 {
			t.Fatalf("status %d: expected no trace output in errors-only mode, got %d bytes", code, buf.Len())
		}
	}

	doRequest(http.StatusConflict)
	if buf.Len() == 0 {
		t.Fatal("status 409: expected error response to be traced in errors-only mode")
	}
}

// Tests valid hosts for location.
func TestValidBucketLocation(t *testing.T) {
	s3Hosts := []struct {
		bucketLocation string
		useDualstack   bool
		endpoint       string
	}{
		{"us-east-1", true, "s3.dualstack.us-east-1.amazonaws.com"},
		{"us-east-1", false, "s3.us-east-1.amazonaws.com"},
		{"unknown", true, "s3.dualstack.us-east-1.amazonaws.com"},
		{"unknown", false, "s3.us-east-1.amazonaws.com"},
		{"ap-southeast-1", true, "s3.dualstack.ap-southeast-1.amazonaws.com"},
		{"ap-southeast-1", false, "s3.ap-southeast-1.amazonaws.com"},
		// ISO regions without dualstack support
		{"us-iso-east-1", true, "s3.us-iso-east-1.c2s.ic.gov"},
		{"us-iso-east-1", false, "s3.us-iso-east-1.c2s.ic.gov"},
		{"us-isob-east-1", true, "s3.us-isob-east-1.sc2s.sgov.gov"},
		{"us-isob-east-1", false, "s3.us-isob-east-1.sc2s.sgov.gov"},
		{"us-iso-west-1", true, "s3.us-iso-west-1.c2s.ic.gov"},
		{"us-iso-west-1", false, "s3.us-iso-west-1.c2s.ic.gov"},
	}
	for _, s3Host := range s3Hosts {
		endpoint := getS3Endpoint(s3Host.bucketLocation, s3Host.useDualstack)
		if endpoint != s3Host.endpoint {
			t.Fatal("Error: invalid bucket location", endpoint)
		}
	}
}

// Tests error response structure.
func TestErrorResponse(t *testing.T) {
	var err error
	err = ErrorResponse{
		Code: Testing,
	}
	errResp := ToErrorResponse(err)
	if errResp.Code != Testing {
		t.Fatal("Type conversion failed, we have an empty struct.")
	}

	// Should fail with invalid argument.
	err = httpRespToErrorResponse(nil, "", "")
	errResp = ToErrorResponse(err)
	if errResp.Code != InvalidArgument {
		t.Fatal("Empty response input should return invalid argument.")
	}
}

// Tests signature type.
func TestSignatureType(t *testing.T) {
	clnt := Client{}
	if !clnt.overrideSignerType.IsV4() {
		t.Fatal("Error")
	}
	clnt.overrideSignerType = credentials.SignatureV2
	if !clnt.overrideSignerType.IsV2() {
		t.Fatal("Error")
	}
	if clnt.overrideSignerType.IsV4() {
		t.Fatal("Error")
	}
	clnt.overrideSignerType = credentials.SignatureV4
	if !clnt.overrideSignerType.IsV4() {
		t.Fatal("Error")
	}
}

// Tests bucket policy types.
func TestBucketPolicyTypes(t *testing.T) {
	want := map[string]bool{
		"none":      true,
		"readonly":  true,
		"writeonly": true,
		"readwrite": true,
		"invalid":   false,
	}
	for bucketPolicy, ok := range want {
		if policy.BucketPolicy(bucketPolicy).IsValidBucketPolicy() != ok {
			t.Fatal("Error")
		}
	}
}

// Tests optimal part size.
func TestPartSize(t *testing.T) {
	_, _, _, err := OptimalPartInfo(5000000000000000000, minPartSize)
	if err == nil {
		t.Fatal("Error: should fail")
	}
	totalPartsCount, partSize, lastPartSize, err := OptimalPartInfo(5243928576, 5*1024*1024)
	if err != nil {
		t.Fatal("Error: ", err)
	}
	if totalPartsCount != 1001 {
		t.Fatalf("Error: expecting total parts count of 1001: got %v instead", totalPartsCount)
	}
	if partSize != 5242880 {
		t.Fatalf("Error: expecting part size of 5242880: got %v instead", partSize)
	}
	if lastPartSize != 1048576 {
		t.Fatalf("Error: expecting last part size of 1048576: got %v instead", lastPartSize)
	}
	totalPartsCount, partSize, lastPartSize, err = OptimalPartInfo(5243928576, 0)
	if err != nil {
		t.Fatal("Error: ", err)
	}
	if totalPartsCount != 313 {
		t.Fatalf("Error: expecting total parts count of 313: got %v instead", totalPartsCount)
	}
	if partSize != 16777216 {
		t.Fatalf("Error: expecting part size of 16777216: got %v instead", partSize)
	}
	if lastPartSize != 9437184 {
		t.Fatalf("Error: expecting last part size of 9437184: got %v instead", lastPartSize)
	}
	_, partSize, _, err = OptimalPartInfo(5000000000, minPartSize)
	if err != nil {
		t.Fatal("Error:", err)
	}
	if partSize != minPartSize {
		t.Fatalf("Error: expecting part size of %v: got %v instead", minPartSize, partSize)
	}
	// if stream and using default optimal part size determined by sdk
	totalPartsCount, partSize, lastPartSize, err = OptimalPartInfo(-1, 0)
	if err != nil {
		t.Fatal("Error:", err)
	}
	if totalPartsCount != 9930 {
		t.Fatalf("Error: expecting total parts count of 9930: got %v instead", totalPartsCount)
	}
	if partSize != 553648128 {
		t.Fatalf("Error: expecting part size of 553648128: got %v instead", partSize)
	}
	if lastPartSize != 385875968 {
		t.Fatalf("Error: expecting last part size of 385875968: got %v instead", lastPartSize)
	}

	totalPartsCount, partSize, lastPartSize, err = OptimalPartInfo(-1, 64*1024*1024)
	if err != nil {
		t.Fatal("Error:", err)
	}
	if totalPartsCount != 10000 {
		t.Fatalf("Error: expecting total parts count of 10000: got %v instead", totalPartsCount)
	}
	if partSize != 67108864 {
		t.Fatalf("Error: expecting part size of 67108864: got %v instead", partSize)
	}
	if lastPartSize != 67108864 {
		t.Fatalf("Error: expecting part size of 67108864: got %v instead", lastPartSize)
	}
}

// TestMakeTargetURL - testing makeTargetURL()
func TestMakeTargetURL(t *testing.T) {
	testCases := []struct {
		addr           string
		secure         bool
		bucketName     string
		objectName     string
		bucketLocation string
		queryValues    map[string][]string
		expectedURL    url.URL
		expectedErr    error
	}{
		// Test 1
		{"localhost:9000", false, "", "", "", nil, url.URL{Host: "localhost:9000", Scheme: "http", Path: "/"}, nil},
		// Test 2
		{"localhost", true, "", "", "", nil, url.URL{Host: "localhost", Scheme: "https", Path: "/"}, nil},
		// Test 3
		{"localhost:9000", true, "mybucket", "", "", nil, url.URL{Host: "localhost:9000", Scheme: "https", Path: "/mybucket/"}, nil},
		// Test 4, testing against google storage API
		{"storage.googleapis.com", true, "mybucket", "", "", nil, url.URL{Host: "mybucket.storage.googleapis.com", Scheme: "https", Path: "/"}, nil},
		// Test 5, testing against AWS S3 API
		{"s3.amazonaws.com", true, "mybucket", "myobject", "", nil, url.URL{Host: "mybucket.s3.dualstack.us-east-1.amazonaws.com", Scheme: "https", Path: "/myobject"}, nil},
		// Test 6
		{"localhost:9000", false, "mybucket", "myobject", "", nil, url.URL{Host: "localhost:9000", Scheme: "http", Path: "/mybucket/myobject"}, nil},
		// Test 7, testing with query
		{"localhost:9000", false, "mybucket", "myobject", "", map[string][]string{"param": {"val"}}, url.URL{Host: "localhost:9000", Scheme: "http", Path: "/mybucket/myobject", RawQuery: "param=val"}, nil},
		// Test 8, testing with port 80
		{"localhost:80", false, "mybucket", "myobject", "", nil, url.URL{Host: "localhost", Scheme: "http", Path: "/mybucket/myobject"}, nil},
		// Test 9, testing with port 443
		{"localhost:443", true, "mybucket", "myobject", "", nil, url.URL{Host: "localhost", Scheme: "https", Path: "/mybucket/myobject"}, nil},
		{"[240b:c0e0:102:54C0:1c05:c2c1:19:5001]:443", true, "mybucket", "myobject", "", nil, url.URL{Host: "[240b:c0e0:102:54C0:1c05:c2c1:19:5001]", Scheme: "https", Path: "/mybucket/myobject"}, nil},
		{"[240b:c0e0:102:54C0:1c05:c2c1:19:5001]:9000", true, "mybucket", "myobject", "", nil, url.URL{Host: "[240b:c0e0:102:54C0:1c05:c2c1:19:5001]:9000", Scheme: "https", Path: "/mybucket/myobject"}, nil},
	}

	for i, testCase := range testCases {
		// Initialize a MinIO client
		c, _ := New(testCase.addr, &Options{
			Creds:  credentials.NewStaticV4("foo", "bar", ""),
			Secure: testCase.secure,
		})
		isVirtualHost := c.isVirtualHostStyleRequest(*c.endpointURL, testCase.bucketName)
		u, err := c.makeTargetURL(testCase.bucketName, testCase.objectName, testCase.bucketLocation, isVirtualHost, testCase.queryValues)
		// Check the returned error
		if testCase.expectedErr == nil && err != nil {
			t.Fatalf("Test %d: Should succeed but failed with err = %v", i+1, err)
		}
		if testCase.expectedErr != nil && err == nil {
			t.Fatalf("Test %d: Should fail but succeeded", i+1)
		}
		if err == nil {
			// Check if the returned url is equal to what we expect
			if u.String() != testCase.expectedURL.String() {
				t.Fatalf("Test %d: Mismatched target url: expected = `%v`, found = `%v`",
					i+1, testCase.expectedURL.String(), u.String())
			}
		}
	}
}
