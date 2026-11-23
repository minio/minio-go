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
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/minio/minio-go/v7/pkg/credentials"
)

func annotationTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	c, err := New(u.Host, &Options{Creds: credentials.NewStaticV4("access", "secret", ""), Region: "us-east-1"})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// The payload the annotation probe sent to AWS S3, and S3's CRC32 and XXH3 of
// it.
const (
	testAnnotationPayload = `{"label":"probe"}`
	testAnnotationCRC32   = "tvaadQ=="
	testAnnotationXXHash3 = "0ytq3XX6rZ0="
)

func TestPutObjectAnnotationSendsChecksum(t *testing.T) {
	var got http.Header
	c := annotationTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		body, _ := io.ReadAll(r.Body)
		if string(body) != testAnnotationPayload {
			t.Errorf("payload %q", body)
		}
		w.Header().Set("ETag", `"abc"`)
	})
	etag, err := c.PutObjectAnnotation(context.Background(), "bucket", "object", "a", []byte(testAnnotationPayload),
		PutObjectAnnotationOptions{Checksum: ChecksumCRC32})
	if err != nil || etag != "abc" {
		t.Fatalf("got %q, %v", etag, err)
	}
	if v := got.Get("X-Amz-Checksum-Crc32"); v != testAnnotationCRC32 {
		t.Fatalf("sent checksum %q, want %q", v, testAnnotationCRC32)
	}

	// Without a checksum type the client sends XXH3.
	if _, err := c.PutObjectAnnotation(context.Background(), "bucket", "object", "a", []byte(testAnnotationPayload),
		PutObjectAnnotationOptions{}); err != nil {
		t.Fatal(err)
	}
	if v := got.Get("X-Amz-Checksum-Xxhash3"); v != testAnnotationXXHash3 {
		t.Fatalf("sent default checksum %q, want XXH3 %q", v, testAnnotationXXHash3)
	}

	// Two algorithms at once name no checksum to send.
	got = nil
	if _, err := c.PutObjectAnnotation(context.Background(), "bucket", "object", "a", []byte(testAnnotationPayload),
		PutObjectAnnotationOptions{Checksum: ChecksumCRC32 | ChecksumSHA256}); err == nil || got != nil {
		t.Fatalf("a put with two checksum algorithms: %v, sent %v", err, got != nil)
	}
}

func TestGetObjectAnnotationVerifiesChecksum(t *testing.T) {
	c := annotationTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Amz-Checksum-Mode") != "ENABLED" {
			t.Errorf("checksum mode not asked for")
		}
		switch r.URL.Query().Get("annotationName") {
		case "good":
			w.Header().Set("X-Amz-Checksum-Crc32", testAnnotationCRC32)
		case "bad":
			w.Header().Set("X-Amz-Checksum-Crc32", "AAAAAA==")
		}
		io.WriteString(w, testAnnotationPayload)
	})
	// A server that returns no checksum, as earlier ones, leaves nothing to verify.
	for _, tc := range []struct {
		name string
		ok   bool
	}{{"good", true}, {"bad", false}, {"none", true}} {
		body, err := c.GetObjectAnnotation(context.Background(), "bucket", "object", tc.name, GetObjectAnnotationOptions{})
		if (err == nil) != tc.ok || tc.ok && string(body) != testAnnotationPayload {
			t.Fatalf("%s: read %q, %v", tc.name, body, err)
		}
	}
}

// TestGetObjectAnnotationStopsOversizedBody checks that a read of a payload over
// 1 MiB returns without reading the rest of the body, which can be of any length.
func TestGetObjectAnnotationStopsOversizedBody(t *testing.T) {
	c := annotationTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		chunk := make([]byte, 64<<10)
		for {
			if _, err := w.Write(chunk); err != nil {
				return
			}
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := c.GetObjectAnnotation(ctx, "bucket", "object", "a", GetObjectAnnotationOptions{}); err == nil || ctx.Err() != nil {
		t.Fatalf("read of an endless body: %v, context %v", err, ctx.Err())
	}
}

func TestListObjectAnnotationsPagesAndForms(t *testing.T) {
	var prefixes []string
	c := annotationTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		prefixes = append(prefixes, q.Get("annotation-prefix"))
		switch {
		case q.Get("annotation-prefix") == "loop":
			io.WriteString(w, `<ListObjectAnnotationsOutput><Annotations></Annotations><NextContinuationToken>same</NextContinuationToken></ListObjectAnnotationsOutput>`)
		case q.Get("annotation-prefix") == "none":
			io.WriteString(w, `<ListObjectAnnotationsOutput><Annotations></Annotations><AnnotationCount>0</AnnotationCount></ListObjectAnnotationsOutput>`)
		case q.Get("annotation-prefix") == "flat":
			// An earlier AIStor server: flat entries, prefix ignored.
			io.WriteString(w, `<ListObjectAnnotationsOutput><Annotation><AnnotationName>flat-a</AnnotationName><Size>1</Size></Annotation>`+
				`<Annotation><AnnotationName>other</AnnotationName><Size>1</Size></Annotation></ListObjectAnnotationsOutput>`)
		case q.Get("continuation-token") == "":
			io.WriteString(w, `<ListObjectAnnotationsOutput xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Annotations>`+
				`<AnnotationEntry><AnnotationName>a</AnnotationName><LastModified>2026-11-11T10:55:21.000Z</LastModified><ETag>"e1"</ETag><ChecksumAlgorithm>CRC32</ChecksumAlgorithm><Size>17</Size></AnnotationEntry>`+
				`</Annotations><Bucket>bucket</Bucket><Key>object</Key><AnnotationCount>1</AnnotationCount><NextContinuationToken>t1</NextContinuationToken></ListObjectAnnotationsOutput>`)
		case q.Get("continuation-token") == "t1":
			io.WriteString(w, `<ListObjectAnnotationsOutput xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Annotations>`+
				`<AnnotationEntry><AnnotationName>b</AnnotationName><ETag>"e2"</ETag><ChecksumAlgorithm>CRC64NVME</ChecksumAlgorithm><Size>3</Size></AnnotationEntry>`+
				`</Annotations><AnnotationCount>1</AnnotationCount><ContinuationToken>t1</ContinuationToken></ListObjectAnnotationsOutput>`)
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	})
	anns, err := c.ListObjectAnnotations(context.Background(), "bucket", "object", ListObjectAnnotationsOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(anns) != 2 || anns[0].Name != "a" || anns[0].ChecksumAlgorithm != "CRC32" || anns[0].ETag != "e1" || anns[0].Size != 17 ||
		anns[0].LastModified.IsZero() || anns[1].Name != "b" || anns[1].ChecksumAlgorithm != "CRC64NVME" {
		t.Fatalf("listed %+v", anns)
	}
	anns, err = c.ListObjectAnnotations(context.Background(), "bucket", "object", ListObjectAnnotationsOptions{Prefix: "flat"})
	if err != nil || len(anns) != 1 || anns[0].Name != "flat-a" {
		t.Fatalf("flat listing %+v, %v", anns, err)
	}
	// An empty listing is an empty slice, as it always was, not nil.
	anns, err = c.ListObjectAnnotations(context.Background(), "bucket", "object", ListObjectAnnotationsOptions{Prefix: "none"})
	if err != nil || anns == nil || len(anns) != 0 {
		t.Fatalf("empty listing %#v, %v", anns, err)
	}
	// A token that does not move on is an error, not the end of the listing.
	if _, err = c.ListObjectAnnotations(context.Background(), "bucket", "object", ListObjectAnnotationsOptions{Prefix: "loop"}); err == nil {
		t.Fatal("a repeated continuation token ended the listing without an error")
	}
	if strings.Join(prefixes, ",") != ",,flat,none,loop,loop" {
		t.Fatalf("sent prefixes %q", prefixes)
	}
}
