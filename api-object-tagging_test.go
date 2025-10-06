/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 *
 * Copyright (c) 2015-2025 MinIO, Inc.
 * Copyright (c) 2025 iamzoy <https://github.com/iamzoy>
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
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/minio/minio-go/v7/pkg/tags"
)

// mockRoundTripper intercepts requests so we can inspect them
type mockRoundTripper struct {
	lastRequest *http.Request
	body        string
	status      int
}

// reusableBody can be read multiple times.
type reusableBody struct {
	xml string
}

func (r *reusableBody) Read(p []byte) (n int, err error) {
	if len(r.xml) == 0 {
		return 0, io.EOF
	}
	n = copy(p, []byte(r.xml))
	r.xml = r.xml[n:]
	if len(r.xml) == 0 {
		err = io.EOF
	}
	return n, err
}
func (r *reusableBody) Close() error { return nil }

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.lastRequest = req

	switch {
	case req.Method == http.MethodGet && strings.Contains(req.URL.RawQuery, "tagging"):
		// Valid, fully-qualified XML with namespace.
		xmlBody := `<Tagging xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
		                <TagSet>
		                    <Tag>
		                        <Key>env</Key>
		                        <Value>dev</Value>
		                    </Tag>
		                </TagSet>
		            </Tagging>`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       &reusableBody{xml: xmlBody},
			Header:     make(http.Header),
		}, nil

	case req.Method == http.MethodPut && strings.Contains(req.URL.RawQuery, "tagging"):
		data, _ := io.ReadAll(req.Body)
		m.body = string(data)
		xmlBody := `<Tagging xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><TagSet/></Tagging>`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       &reusableBody{xml: xmlBody},
			Header:     make(http.Header),
		}, nil

	default:
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       &reusableBody{xml: `<Tagging xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><TagSet/></Tagging>`},
			Header:     make(http.Header),
		}, nil
	}
}

func TestPutObjectTaggingIfChanged(t *testing.T) {
	mock := &mockRoundTripper{status: http.StatusOK}

	// Create fake client with mock transport
	client, err := New("play.min.io", &Options{
		Creds:     nil,
		Secure:    true,
		Transport: mock,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	bucketName := "test-bucket"
	objectName := "test-object"

	// Same tag case (should NOT perform PUT)
	tagSame, _ := tags.NewTags(map[string]string{"env": "dev"}, false)
	err = client.PutObjectTaggingIfChanged(ctx, bucketName, objectName, tagSame, PutObjectTaggingOptions{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mock.body != "" {
		t.Errorf("expected no PUT request since tags are unchanged, got body: %s", mock.body)
	}

	// Changed tag case (should perform PUT)
	tagChanged, _ := tags.NewTags(map[string]string{"env": "prod"}, false)
	err = client.PutObjectTaggingIfChanged(ctx, bucketName, objectName, tagChanged, PutObjectTaggingOptions{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if mock.body == "" {
		t.Errorf("expected PUT request to be made for changed tags")
	}

	// Validate the XML contains new tag
	if !strings.Contains(mock.body, "<Value>prod</Value>") {
		t.Errorf("expected XML body to contain new tag value, got: %s", mock.body)
	}
}
