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
	"net/url"
	"testing"

	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/tags"
)

func TestObjectTaggingConditionalHeaders(t *testing.T) {
	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	c, err := New(u.Host, &Options{
		Creds:  credentials.NewStaticV4("ak", "sk", ""),
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	tagSet, err := tags.NewTags(map[string]string{"k": "v"}, true)
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name            string
		matchETag       string
		matchETagExcept string
		wantIfMatch     string
		wantIfNoneMatch string
	}{
		{name: "no condition"},
		{name: "etag", matchETag: "abc", wantIfMatch: `"abc"`},
		{name: "any etag", matchETag: "*", wantIfMatch: "*"},
		{name: "no tags", matchETagExcept: "*", wantIfNoneMatch: "*"},
		{name: "etag except", matchETagExcept: "abc", wantIfNoneMatch: `"abc"`},
		{name: "both", matchETag: "abc", matchETagExcept: "*", wantIfMatch: `"abc"`, wantIfNoneMatch: "*"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			check := func(op string) {
				t.Helper()
				if v := got.Get("If-Match"); v != tc.wantIfMatch {
					t.Errorf("%s: If-Match = %q, want %q", op, v, tc.wantIfMatch)
				}
				if v := got.Get("If-None-Match"); v != tc.wantIfNoneMatch {
					t.Errorf("%s: If-None-Match = %q, want %q", op, v, tc.wantIfNoneMatch)
				}
			}
			var putOpts PutObjectTaggingOptions
			var removeOpts RemoveObjectTaggingOptions
			if tc.matchETag != "" {
				putOpts.SetMatchETag(tc.matchETag)
				removeOpts.SetMatchETag(tc.matchETag)
			}
			if tc.matchETagExcept != "" {
				putOpts.SetMatchETagExcept(tc.matchETagExcept)
				removeOpts.SetMatchETagExcept(tc.matchETagExcept)
			}
			err := c.PutObjectTagging(context.Background(), "bucket", "object", tagSet, putOpts)
			if err != nil {
				t.Fatal(err)
			}
			check("PutObjectTagging")
			err = c.RemoveObjectTagging(context.Background(), "bucket", "object", removeOpts)
			if err != nil {
				t.Fatal(err)
			}
			check("RemoveObjectTagging")
		})
	}
}
