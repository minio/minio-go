/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2015-2017 MinIO, Inc.
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
	"net/url"
	"testing"

	"github.com/minio/minio-go/v6/pkg/credentials"
	"github.com/minio/minio-go/v6/pkg/policy"
)

// Tests valid hosts for location.
func TestValidBucketLocation(t *testing.T) {
	s3Hosts := []struct {
		bucketLocation string
		endpoint       string
	}{
		{"us-east-1", "s3.dualstack.us-east-1.amazonaws.com"},
		{"unknown", "s3.dualstack.us-east-1.amazonaws.com"},
		{"ap-southeast-1", "s3.dualstack.ap-southeast-1.amazonaws.com"},
	}
	for _, s3Host := range s3Hosts {
		endpoint := getS3Endpoint(s3Host.bucketLocation)
		if endpoint != s3Host.endpoint {
			t.Fatal("Error: invalid bucket location", endpoint)
		}
	}
}

// Tests error response structure.
func TestErrorResponse(t *testing.T) {
	var err error
	err = ErrorResponse{
		Code: "Testing",
	}
	errResp := ToErrorResponse(err)
	if errResp.Code != "Testing" {
		t.Fatal("Type conversion failed, we have an empty struct.")
	}

	// Should fail with invalid argument.
	err = httpRespToErrorResponse(nil, "", "")
	errResp = ToErrorResponse(err)
	if errResp.Code != "InvalidArgument" {
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
	_, _, _, err := optimalPartInfo(5000000000000000000, minPartSize)
	if err == nil {
		t.Fatal("Error: should fail")
	}
	totalPartsCount, partSize, lastPartSize, err := optimalPartInfo(5243928576, 5*1024*1024)
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
	totalPartsCount, partSize, lastPartSize, err = optimalPartInfo(5243928576, 0)
	if err != nil {
		t.Fatal("Error: ", err)
	}
	if totalPartsCount != 40 {
		t.Fatalf("Error: expecting total parts count of 40: got %v instead", totalPartsCount)
	}
	if partSize != 134217728 {
		t.Fatalf("Error: expecting part size of 134217728: got %v instead", partSize)
	}
	if lastPartSize != 9437184 {
		t.Fatalf("Error: expecting last part size of 9437184: got %v instead", lastPartSize)
	}
	_, partSize, _, err = optimalPartInfo(5000000000, minPartSize)
	if err != nil {
		t.Fatal("Error:", err)
	}
	if partSize != minPartSize {
		t.Fatalf("Error: expecting part size of %v: got %v instead", minPartSize, partSize)
	}
	// if stream and using default optimal part size determined by sdk
	totalPartsCount, partSize, lastPartSize, err = optimalPartInfo(-1, 0)
	if err != nil {
		t.Fatal("Error:", err)
	}
	if totalPartsCount != 8192 {
		t.Fatalf("Error: expecting total parts count of 8192: got %v instead", totalPartsCount)
	}
	if partSize != 671088640 {
		t.Fatalf("Error: expecting part size of 671088640: got %v instead", partSize)
	}
	if lastPartSize != 671088640 {
		t.Fatalf("Error: expecting last part size of 671088640: got %v instead", lastPartSize)
	}

	totalPartsCount, partSize, lastPartSize, err = optimalPartInfo(-1, 64*1024*1024)
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
	}

	for i, testCase := range testCases {
		// Initialize a MinIO client
		c, _ := New(testCase.addr, "foo", "bar", testCase.secure)
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
