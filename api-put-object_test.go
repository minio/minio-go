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
	"context"
	"encoding/base64"
	"net/http"
	"reflect"
	"testing"

	"github.com/minio/minio-go/v7/pkg/encrypt"
)

func TestPutObjectOptionsValidate(t *testing.T) {
	testCases := []struct {
		name, value string
		shouldPass  bool
	}{
		// Invalid cases.
		{"X-Amz-Matdesc", "blah", false},
		{"x-amz-meta-X-Amz-Iv", "blah", false},
		{"x-amz-meta-X-Amz-Key", "blah", false},
		{"x-amz-meta-X-Amz-Matdesc", "blah", false},
		{"It has spaces", "v", false},
		{"It,has@illegal=characters", "v", false},
		{"X-Amz-Iv", "blah", false},
		{"X-Amz-Key", "blah", false},
		{"X-Amz-Key-prefixed-header", "blah", false},
		{"Content-Type", "custom/content-type", false},
		{"content-type", "custom/content-type", false},
		{"Content-Encoding", "gzip", false},
		{"Cache-Control", "blah", false},
		{"Content-Disposition", "something", false},
		{"Content-Language", "somelanguage", false},

		// Valid metadata names.
		{"my-custom-header", "blah", true},
		{"custom-X-Amz-Key-middle", "blah", true},
		{"my-custom-header-X-Amz-Key", "blah", true},
		{"blah-X-Amz-Matdesc", "blah", true},
		{"X-Amz-MatDesc-suffix", "blah", true},
		{"It-Is-Fine", "v", true},
		{"Numbers-098987987-Should-Work", "v", true},
		{"Crazy-!#$%&'*+-.^_`|~-Should-193832-Be-Fine", "v", true},
	}
	for i, testCase := range testCases {
		err := PutObjectOptions{UserMetadata: map[string]string{
			testCase.name: testCase.value,
		}}.validate(nil)
		if testCase.shouldPass && err != nil {
			t.Errorf("Test %d - output did not match with reference results, %s", i+1, err)
		}
	}
}

type InterceptRouteTripper struct {
	request *http.Request
}

func (i *InterceptRouteTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	i.request = request
	return &http.Response{StatusCode: 200}, nil
}

func Test_SSEHeaders(t *testing.T) {
	rt := &InterceptRouteTripper{}
	c, err := New("s3.amazonaws.com", &Options{
		Transport: rt,
	})
	if err != nil {
		t.Error(err)
	}

	testCases := map[string]struct {
		sse                            func() encrypt.ServerSide
		initiateMultipartUploadHeaders http.Header
		headerNotAllowedAfterInit      []string
	}{
		"noEncryption": {
			sse:                            func() encrypt.ServerSide { return nil },
			initiateMultipartUploadHeaders: http.Header{},
		},
		"sse": {
			sse: func() encrypt.ServerSide {
				s, err := encrypt.NewSSEKMS("keyId", nil)
				if err != nil {
					t.Error(err)
				}
				return s
			},
			initiateMultipartUploadHeaders: http.Header{
				encrypt.SseGenericHeader: []string{"aws:kms"},
				encrypt.SseKmsKeyID:      []string{"keyId"},
			},
			headerNotAllowedAfterInit: []string{encrypt.SseGenericHeader, encrypt.SseKmsKeyID, encrypt.SseEncryptionContext},
		},
		"sse with context": {
			sse: func() encrypt.ServerSide {
				s, err := encrypt.NewSSEKMS("keyId", "context")
				if err != nil {
					t.Error(err)
				}
				return s
			},
			initiateMultipartUploadHeaders: http.Header{
				encrypt.SseGenericHeader:     []string{"aws:kms"},
				encrypt.SseKmsKeyID:          []string{"keyId"},
				encrypt.SseEncryptionContext: []string{base64.StdEncoding.EncodeToString([]byte("\"context\""))},
			},
			headerNotAllowedAfterInit: []string{encrypt.SseGenericHeader, encrypt.SseKmsKeyID, encrypt.SseEncryptionContext},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			opts := PutObjectOptions{
				ServerSideEncryption: tc.sse(),
			}
			c.bucketLocCache.Set("test", "region")
			c.initiateMultipartUpload(context.Background(), "test", "test", opts)
			for s, vls := range tc.initiateMultipartUploadHeaders {
				if !reflect.DeepEqual(rt.request.Header[s], vls) {
					t.Errorf("Header %v are not equal, want: %v got %v", s, vls, rt.request.Header[s])
				}
			}

			_, err := c.uploadPart(context.Background(), uploadPartParams{
				bucketName: "test",
				objectName: "test",
				partNumber: 1,
				uploadID:   "upId",
				sse:        opts.ServerSideEncryption,
			})
			if err != nil {
				t.Error(err)
			}

			for _, k := range tc.headerNotAllowedAfterInit {
				if rt.request.Header.Get(k) != "" {
					t.Errorf("header %v should not be set", k)
				}
			}

			c.completeMultipartUpload(context.Background(), "test", "test", "upId", completeMultipartUpload{}, opts)

			for _, k := range tc.headerNotAllowedAfterInit {
				if rt.request.Header.Get(k) != "" {
					t.Errorf("header %v should not be set", k)
				}
			}
		})
	}
}
