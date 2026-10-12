/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2017 MinIO, Inc.
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

package signer

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	md5simd "github.com/minio/md5-simd"
)

// hashWrapper implements the md5simd.Hasher interface.
type hashWrapper struct {
	hash.Hash
}

func newSHA256Hasher() md5simd.Hasher {
	return &hashWrapper{Hash: sha256.New()}
}

func (m *hashWrapper) Close() {
	m.Hash = nil
}

func sum256hex(data []byte) string {
	hash := sha256.New()
	hash.Write(data)
	return hex.EncodeToString(hash.Sum(nil))
}

func TestGetSeedSignature(t *testing.T) {
	accessKeyID := "AKIAIOSFODNN7EXAMPLE"
	secretAccessKeyID := "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	dataLen := 66560
	data := bytes.Repeat([]byte("a"), dataLen)
	body := io.NopCloser(bytes.NewReader(data))

	req := NewRequest(http.MethodPut, "/examplebucket/chunkObject.txt", body)
	req.Header.Set("x-amz-storage-class", "REDUCED_REDUNDANCY")
	req.Host = "s3.amazonaws.com"

	reqTime, err := time.Parse("20060102T150405Z", "20130524T000000Z")
	if err != nil {
		t.Fatalf("Failed to parse time - %v", err)
	}

	req = StreamingSignV4(req, accessKeyID, secretAccessKeyID, "", "us-east-1", int64(dataLen), reqTime, newSHA256Hasher())
	actualSeedSignature := req.Body.(*StreamingReader).seedSignature

	expectedSeedSignature := "007480502de61457e955731b0f5d191f7e6f54a8a0f6cc7974a5ebd887965686"
	if actualSeedSignature != expectedSeedSignature {
		t.Errorf("Expected %s but received %s", expectedSeedSignature, actualSeedSignature)
	}
}

func TestChunkSignature(t *testing.T) {
	chunkData := bytes.Repeat([]byte("a"), 65536)
	reqTime, _ := time.Parse(iso8601DateFormat, "20130524T000000Z")
	previousSignature := "4f232c4386841ef735655705268965c44a0e4690baa4adea153f7db9fa80a0a9"
	location := "us-east-1"
	secretAccessKeyID := "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	expectedSignature := "ad80c730a21e5b8d04586a2213dd63b9a0e99e0e2307b0ade35a65485a288648"
	chunkCheckSum := sum256hex(chunkData)
	actualSignature := buildChunkSignature(chunkCheckSum, reqTime, location, previousSignature, secretAccessKeyID, ServiceTypeS3)
	if actualSignature != expectedSignature {
		t.Errorf("Expected %s but received %s", expectedSignature, actualSignature)
	}
}

// Example on https://docs.aws.amazon.com/AmazonS3/latest/API/sigv4-streaming-trailers.html
func TestTrailerChunkSignature(t *testing.T) {
	chunkData := []byte("x-amz-checksum-crc32c:wdBDMA==\n")
	reqTime, _ := time.Parse(iso8601DateFormat, "20130524T000000Z")
	previousSignature := "e05ab64fe1dfdbf0b5870abbaabdb063c371d4e96f2767e6934d90529c5ae850"
	location := "us-east-1"
	secretAccessKeyID := "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	expectedSignature := "41e14ac611e27a8bb3d66c3bad6856f209297767d5dd4fc87d8fa9e422e03faf"
	chunkCheckSum := sum256hex(chunkData)
	actualSignature := buildTrailerChunkSignature(chunkCheckSum, reqTime, location, previousSignature, secretAccessKeyID, ServiceTypeS3)
	if actualSignature != expectedSignature {
		t.Errorf("Expected %s but received %s", expectedSignature, actualSignature)
	}
}

func TestSetStreamingAuthorization(t *testing.T) {
	location := "us-east-1"
	secretAccessKeyID := "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	accessKeyID := "AKIAIOSFODNN7EXAMPLE"

	req := NewRequest(http.MethodPut, "/examplebucket/chunkObject.txt", nil)
	req.Header.Set("x-amz-storage-class", "REDUCED_REDUNDANCY")
	req.Host = ""
	req.URL.Host = "s3.amazonaws.com"

	dataLen := int64(65 * 1024)
	reqTime, _ := time.Parse(iso8601DateFormat, "20130524T000000Z")
	req = StreamingSignV4(req, accessKeyID, secretAccessKeyID, "", location, dataLen, reqTime, newSHA256Hasher())

	expectedAuthorization := "AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20130524/us-east-1/s3/aws4_request,SignedHeaders=content-encoding;host;x-amz-content-sha256;x-amz-date;x-amz-decoded-content-length;x-amz-storage-class,Signature=007480502de61457e955731b0f5d191f7e6f54a8a0f6cc7974a5ebd887965686"

	actualAuthorization := req.Header.Get("Authorization")
	if actualAuthorization != expectedAuthorization {
		t.Errorf("Expected %s but received %s", expectedAuthorization, actualAuthorization)
	}
}

// Test against https://docs.aws.amazon.com/AmazonS3/latest/API/sigv4-streaming-trailers.html
func TestSetStreamingAuthorizationTrailer(t *testing.T) {
	location := "us-east-1"
	secretAccessKeyID := "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	accessKeyID := "AKIAIOSFODNN7EXAMPLE"

	req := NewRequest(http.MethodPut, "/examplebucket/chunkObject.txt", nil)
	req.Header.Set("Content-Encoding", "aws-chunked")
	req.Header.Set("x-amz-decoded-content-length", "66560")
	req.Header.Set("x-amz-storage-class", "REDUCED_REDUNDANCY")
	req.Host = ""
	req.URL.Host = "s3.amazonaws.com"
	req.Trailer = http.Header{}
	req.Trailer.Set("x-amz-checksum-crc32c", "wdBDMA==")

	dataLen := int64(65 * 1024)
	reqTime, _ := time.Parse(iso8601DateFormat, "20130524T000000Z")
	req = StreamingSignV4(req, accessKeyID, secretAccessKeyID, "", location, dataLen, reqTime, newSHA256Hasher())

	// (order of signed headers is different)
	expectedAuthorization := "AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20130524/us-east-1/s3/aws4_request,SignedHeaders=content-encoding;host;x-amz-content-sha256;x-amz-date;x-amz-decoded-content-length;x-amz-storage-class;x-amz-trailer,Signature=106e2a8a18243abcf37539882f36619c00e2dfc72633413f02d3b74544bfeb8e"

	actualAuthorization := req.Header.Get("Authorization")
	if actualAuthorization != expectedAuthorization {
		t.Errorf("Expected \n%s but received \n%s", expectedAuthorization, actualAuthorization)
	}
	chunkData := []byte("x-amz-checksum-crc32c:wdBDMA==\n")
	t.Log(hex.EncodeToString(sum256(chunkData)))
}

func TestStreamingReader(t *testing.T) {
	reqTime, _ := time.Parse("20060102T150405Z", "20130524T000000Z")
	location := "us-east-1"
	secretAccessKeyID := "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	accessKeyID := "AKIAIOSFODNN7EXAMPLE"
	dataLen := int64(65 * 1024)

	req := NewRequest(http.MethodPut, "/examplebucket/chunkObject.txt", nil)
	req.Header.Set("x-amz-storage-class", "REDUCED_REDUNDANCY")
	req.ContentLength = 65 * 1024
	req.Host = ""
	req.URL.Host = "s3.amazonaws.com"

	baseReader := io.NopCloser(bytes.NewReader(bytes.Repeat([]byte("a"), 65*1024)))
	req.Body = baseReader
	req = StreamingSignV4(req, accessKeyID, secretAccessKeyID, "", location, dataLen, reqTime, newSHA256Hasher())

	b, err := io.ReadAll(req.Body)
	if err != nil {
		t.Errorf("Expected no error but received %v  %d", err, len(b))
	}
	req.Body.Close()
}

func TestStreamingSignV4ContentEncoding(t *testing.T) {
	accessKeyID := "AKIAIOSFODNN7EXAMPLE"
	secretAccessKeyID := "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	reqTime, err := time.Parse(iso8601DateFormat, "20130524T000000Z")
	if err != nil {
		t.Fatalf("Failed to parse time - %v", err)
	}

	testCases := []struct {
		name            string
		contentEncoding string
		trailer         http.Header
		expected        string
	}{
		{"no trailer", "", nil, "aws-chunked"},
		{"with trailer", "", http.Header{"X-Amz-Checksum-Crc32c": []string{""}}, "aws-chunked"},
		{"user encoding preserved", "gzip", nil, "aws-chunked,gzip"},
		{"already aws-chunked", "aws-chunked", nil, "aws-chunked"},
		{"already aws-chunked mixed case", "AWS-CHUNKED", nil, "AWS-CHUNKED"},
		{"token after user encoding", "gzip, aws-chunked", nil, "gzip, aws-chunked"},
		{"whitespace only", " ", nil, "aws-chunked"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			dataLen := int64(payloadChunkSize)
			body := io.NopCloser(bytes.NewReader(bytes.Repeat([]byte("a"), int(dataLen))))
			req := NewRequest(http.MethodPut, "/examplebucket/chunkObject.txt", body)
			if testCase.contentEncoding != "" {
				req.Header.Set("Content-Encoding", testCase.contentEncoding)
			}
			req.Trailer = testCase.trailer

			req = StreamingSignV4(req, accessKeyID, secretAccessKeyID, "", "us-east-1", dataLen, reqTime, newSHA256Hasher())

			if gotEncoding := req.Header.Get("Content-Encoding"); gotEncoding != testCase.expected {
				t.Errorf("Content-Encoding = %q, want %q", gotEncoding, testCase.expected)
			}
			auth := req.Header.Get("Authorization")
			_, after, found := strings.Cut(auth, "SignedHeaders=")
			if !found {
				t.Fatalf("SignedHeaders missing from Authorization: %s", auth)
			}
			signedHeaders, _, _ := strings.Cut(after, ",")
			if !slices.Contains(strings.Split(signedHeaders, ";"), "content-encoding") {
				t.Errorf("content-encoding missing from SignedHeaders %q", signedHeaders)
			}
			if err := req.Body.Close(); err != nil {
				t.Errorf("Failed to close request body - %v", err)
			}
		})
	}
}

func TestStreamingSignV4ContentEncodingOnWire(t *testing.T) {
	accessKeyID := "AKIAIOSFODNN7EXAMPLE"
	secretAccessKeyID := "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	reqTime, err := time.Parse(iso8601DateFormat, "20130524T000000Z")
	if err != nil {
		t.Fatalf("Failed to parse time - %v", err)
	}

	type wireHeaders struct {
		contentEncoding  string
		transferEncoding []string
	}
	received := make(chan wireHeaders, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		received <- wireHeaders{r.Header.Get("Content-Encoding"), r.TransferEncoding}
	}))
	defer srv.Close()

	dataLen := int64(payloadChunkSize)
	req, err := http.NewRequest(http.MethodPut, srv.URL+"/examplebucket/chunkObject.txt", bytes.NewReader(bytes.Repeat([]byte("a"), int(dataLen))))
	if err != nil {
		t.Fatalf("Failed to create request - %v", err)
	}
	req = StreamingSignV4(req, accessKeyID, secretAccessKeyID, "", "us-east-1", dataLen, reqTime, newSHA256Hasher())

	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("Request failed - %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Errorf("Failed to close response body - %v", err)
	}

	got := <-received
	if got.contentEncoding != "aws-chunked" {
		t.Errorf("server received Content-Encoding = %q, want %q", got.contentEncoding, "aws-chunked")
	}
	if len(got.transferEncoding) != 0 {
		t.Errorf("server received Transfer-Encoding = %v, want none", got.transferEncoding)
	}
}

func TestStreamingUnsignedV4ContentEncoding(t *testing.T) {
	reqTime, err := time.Parse(iso8601DateFormat, "20130524T000000Z")
	if err != nil {
		t.Fatalf("Failed to parse time - %v", err)
	}
	dataLen := int64(payloadChunkSize)
	body := io.NopCloser(bytes.NewReader(bytes.Repeat([]byte("a"), int(dataLen))))
	req := NewRequest(http.MethodPut, "/examplebucket/chunkObject.txt", body)
	req.Trailer = http.Header{"X-Amz-Checksum-Crc32c": []string{""}}

	req = StreamingUnsignedV4(req, "", dataLen, reqTime)

	if gotEncoding := req.Header.Get("Content-Encoding"); gotEncoding != "aws-chunked" {
		t.Errorf("Content-Encoding = %q, want %q", gotEncoding, "aws-chunked")
	}
	if err := req.Body.Close(); err != nil {
		t.Errorf("Failed to close request body - %v", err)
	}
}

func TestSignV4TrailerContentEncoding(t *testing.T) {
	accessKeyID := "AKIAIOSFODNN7EXAMPLE"
	secretAccessKeyID := "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	trailer := http.Header{"X-Amz-Checksum-Crc32c": []string{""}}

	dataLen := int64(payloadChunkSize)
	body := io.NopCloser(bytes.NewReader(bytes.Repeat([]byte("a"), int(dataLen))))
	req := NewRequest(http.MethodPut, "/examplebucket/chunkObject.txt", body)
	req.ContentLength = dataLen
	req.Header.Set("Content-Encoding", "gzip")

	req = SignV4Trailer(*req, accessKeyID, secretAccessKeyID, "", "us-east-1", trailer)

	if gotEncoding := req.Header.Get("Content-Encoding"); gotEncoding != "aws-chunked,gzip" {
		t.Errorf("Content-Encoding = %q, want %q", gotEncoding, "aws-chunked,gzip")
	}
	if err := req.Body.Close(); err != nil {
		t.Errorf("Failed to close request body - %v", err)
	}
}

func TestUnsignedTrailerContentEncoding(t *testing.T) {
	trailer := http.Header{"X-Amz-Checksum-Crc32c": []string{""}}

	dataLen := int64(payloadChunkSize)
	body := io.NopCloser(bytes.NewReader(bytes.Repeat([]byte("a"), int(dataLen))))
	req := NewRequest(http.MethodPut, "/examplebucket/chunkObject.txt", body)
	req.ContentLength = dataLen
	req.Header.Set("Content-Encoding", "gzip")

	req = UnsignedTrailer(*req, trailer)

	if gotEncoding := req.Header.Get("Content-Encoding"); gotEncoding != "aws-chunked,gzip" {
		t.Errorf("Content-Encoding = %q, want %q", gotEncoding, "aws-chunked,gzip")
	}
	if err := req.Body.Close(); err != nil {
		t.Errorf("Failed to close request body - %v", err)
	}
}

func TestSetAwsChunkedContentEncodingMultiValue(t *testing.T) {
	req := NewRequest(http.MethodPut, "/examplebucket/chunkObject.txt", nil)
	req.Header.Add("Content-Encoding", "gzip")
	req.Header.Add("Content-Encoding", "br")

	setAwsChunkedContentEncoding(req)

	if gotEncoding := req.Header.Get("Content-Encoding"); gotEncoding != "aws-chunked,gzip,br" {
		t.Errorf("Content-Encoding = %q, want %q", gotEncoding, "aws-chunked,gzip,br")
	}

	req = NewRequest(http.MethodPut, "/examplebucket/chunkObject.txt", nil)
	req.Header.Add("Content-Encoding", "gzip")
	req.Header.Add("Content-Encoding", "aws-chunked")

	setAwsChunkedContentEncoding(req)

	if got := req.Header.Values("Content-Encoding"); !slices.Equal(got, []string{"gzip", "aws-chunked"}) {
		t.Errorf("Content-Encoding values = %v, want unchanged [gzip aws-chunked]", got)
	}
}
