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
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7/pkg/s3utils"
)

// Object annotations are named payloads (1 byte to 1 MiB of UTF-8 text)
// attached to a specific object version, independent of the object's data.
// Up to 1,000 annotations may be attached per object version.

const (
	amzObjectIfMatchHeader = "x-amz-object-if-match"

	// maxAnnotationPayloadBytes is the maximum size of a single annotation payload (1 MiB).
	maxAnnotationPayloadBytes = 1 << 20

	// maxAnnotationNameBytes is the maximum length of an annotation name.
	maxAnnotationNameBytes = 512
)

// validateAnnotationName performs a minimal client-side check on the annotation
// name (non-empty, within the length limit). The server enforces the full
// naming rules (allowed characters, reserved prefixes).
func validateAnnotationName(name string) error {
	if name == "" {
		return errInvalidArgument("annotation name must not be empty")
	}
	if len(name) > maxAnnotationNameBytes {
		return errInvalidArgument("annotation name exceeds 512 bytes")
	}
	return nil
}

// PutObjectAnnotationOptions configures a PutObjectAnnotation request.
type PutObjectAnnotationOptions struct {
	// VersionID targets a specific object version (versioned buckets).
	VersionID string
	// IfMatch, when set, only applies the annotation if the parent object's
	// ETag matches this value (sent as x-amz-object-if-match).
	IfMatch string
	// Checksum is the algorithm of the checksum of the payload the client
	// sends, which the server verifies and stores. It defaults to XXH3.
	Checksum ChecksumType
}

// GetObjectAnnotationOptions configures a GetObjectAnnotation request.
type GetObjectAnnotationOptions struct {
	VersionID string
}

// ListObjectAnnotationsOptions configures a ListObjectAnnotations request.
type ListObjectAnnotationsOptions struct {
	VersionID string
	// Prefix limits the listing to annotations whose names start with it.
	Prefix string
}

// RemoveObjectAnnotationOptions configures a DeleteObjectAnnotation request.
type RemoveObjectAnnotationOptions struct {
	VersionID string
	IfMatch   string
}

// ObjectAnnotation describes a single annotation as returned by ListObjectAnnotations.
type ObjectAnnotation struct {
	Name         string
	Size         int64
	ETag         string
	LastModified time.Time
	// ChecksumAlgorithm is the algorithm of the annotation's stored checksum.
	ChecksumAlgorithm string
}

type annotationEntryXML struct {
	AnnotationName    string `xml:"AnnotationName"`
	Size              int64  `xml:"Size"`
	ETag              string `xml:"ETag"`
	LastModified      string `xml:"LastModified"`
	ChecksumAlgorithm string `xml:"ChecksumAlgorithm"`
}

// listObjectAnnotationsOutput maps the ListObjectAnnotations XML response. S3
// wraps the entries in Annotations; earlier AIStor servers list them as
// Annotation elements.
type listObjectAnnotationsOutput struct {
	XMLName               xml.Name             `xml:"ListObjectAnnotationsOutput"`
	Entries               []annotationEntryXML `xml:"Annotations>AnnotationEntry"`
	Flat                  []annotationEntryXML `xml:"Annotation"`
	NextContinuationToken string               `xml:"NextContinuationToken"`
}

func annotationQueryValues(name, versionID string) url.Values {
	urlValues := make(url.Values)
	urlValues.Set("annotation", "")
	if name != "" {
		urlValues.Set("annotationName", name)
	}
	if versionID != "" {
		urlValues.Set("versionId", versionID)
	}
	return urlValues
}

// PutObjectAnnotation creates or overwrites a named annotation on an object
// version. The payload must be 1 byte to 1 MiB. It is sent with its checksum,
// XXH3 unless opts names another, which the server verifies and stores. It
// returns the annotation's ETag. The parent object's ETag is never modified.
func (c *Client) PutObjectAnnotation(ctx context.Context, bucketName, objectName, annotationName string, payload []byte, opts PutObjectAnnotationOptions) (string, error) {
	if err := s3utils.CheckValidBucketName(bucketName); err != nil {
		return "", err
	}
	if err := s3utils.CheckValidObjectName(objectName); err != nil {
		return "", err
	}
	if err := validateAnnotationName(annotationName); err != nil {
		return "", err
	}
	if len(payload) == 0 {
		return "", errInvalidArgument("annotation payload must be at least 1 byte")
	}
	if len(payload) > maxAnnotationPayloadBytes {
		return "", errInvalidArgument("annotation payload exceeds the 1 MiB maximum")
	}
	checksum := opts.Checksum
	if checksum == ChecksumNone {
		checksum = ChecksumXXHash3
	}
	if !checksum.IsSet() {
		return "", errInvalidArgument("annotation checksum must name exactly one algorithm")
	}

	headers := make(http.Header)
	if opts.IfMatch != "" {
		headers.Set(amzObjectIfMatchHeader, opts.IfMatch)
	}
	headers.Set(checksum.Key(), checksum.EncodeToString(payload))

	resp, err := c.executeMethod(ctx, http.MethodPut, requestMetadata{
		bucketName:    bucketName,
		objectName:    objectName,
		queryValues:   annotationQueryValues(annotationName, opts.VersionID),
		contentBody:   bytes.NewReader(payload),
		contentLength: int64(len(payload)),
		customHeader:  headers,
	})
	defer closeResponse(resp)
	if err != nil {
		return "", err
	}
	if resp != nil && resp.StatusCode != http.StatusOK {
		return "", httpRespToErrorResponse(resp, bucketName, objectName)
	}
	return trimEtag(resp.Header.Get("ETag")), nil
}

// GetObjectAnnotation returns the payload of a single named annotation, at most
// 1 MiB. It is verified against the checksum the server stored with it, when
// the server returns one.
func (c *Client) GetObjectAnnotation(ctx context.Context, bucketName, objectName, annotationName string, opts GetObjectAnnotationOptions) ([]byte, error) {
	if err := s3utils.CheckValidBucketName(bucketName); err != nil {
		return nil, err
	}
	if err := s3utils.CheckValidObjectName(objectName); err != nil {
		return nil, err
	}
	if err := validateAnnotationName(annotationName); err != nil {
		return nil, err
	}

	headers := make(http.Header)
	headers.Set("x-amz-checksum-mode", "ENABLED")
	resp, err := c.executeMethod(ctx, http.MethodGet, requestMetadata{
		bucketName:   bucketName,
		objectName:   objectName,
		queryValues:  annotationQueryValues(annotationName, opts.VersionID),
		customHeader: headers,
	})
	defer closeResponse(resp)
	if err != nil {
		return nil, err
	}
	if resp != nil && resp.StatusCode != http.StatusOK {
		return nil, httpRespToErrorResponse(resp, bucketName, objectName)
	}
	payload, err := io.ReadAll(io.LimitReader(resp.Body, maxAnnotationPayloadBytes+1))
	if err != nil {
		return nil, err
	}
	if len(payload) > maxAnnotationPayloadBytes {
		// closeResponse would drain the rest, which can be of any length.
		resp.Body.Close()
		resp.Body = nil
		return nil, fmt.Errorf("annotation %s exceeds the 1 MiB maximum", annotationName)
	}
	for t := ChecksumSHA256; t < checksumLast; t <<= 1 {
		if want := resp.Header.Get(t.Key()); want != "" {
			if got := t.EncodeToString(payload); got != want {
				return nil, fmt.Errorf("annotation %s %s checksum mismatch: got %s, want %s", annotationName, t, got, want)
			}
			break
		}
	}
	return payload, nil
}

// ListObjectAnnotations returns all annotations attached to an object version,
// or those whose names start with opts.Prefix, following the listing's pages.
func (c *Client) ListObjectAnnotations(ctx context.Context, bucketName, objectName string, opts ListObjectAnnotationsOptions) ([]ObjectAnnotation, error) {
	if err := s3utils.CheckValidBucketName(bucketName); err != nil {
		return nil, err
	}
	if err := s3utils.CheckValidObjectName(objectName); err != nil {
		return nil, err
	}

	annotations := []ObjectAnnotation{}
	token := ""
	for {
		page, next, err := c.listObjectAnnotationsPage(ctx, bucketName, objectName, opts, token)
		if err != nil {
			return nil, err
		}
		annotations = append(annotations, page...)
		if next == "" {
			return annotations, nil
		}
		if next == token {
			return nil, fmt.Errorf("listObjectAnnotations repeated continuationToken %q, %s S3 server is buggy", next, c.endpointURL)
		}
		token = next
	}
}

func (c *Client) listObjectAnnotationsPage(ctx context.Context, bucketName, objectName string, opts ListObjectAnnotationsOptions, token string) ([]ObjectAnnotation, string, error) {
	query := annotationQueryValues("", opts.VersionID)
	if opts.Prefix != "" {
		query.Set("annotation-prefix", opts.Prefix)
	}
	if token != "" {
		query.Set("continuation-token", token)
	}
	resp, err := c.executeMethod(ctx, http.MethodGet, requestMetadata{
		bucketName:  bucketName,
		objectName:  objectName,
		queryValues: query,
	})
	defer closeResponse(resp)
	if err != nil {
		return nil, "", err
	}
	if resp != nil && resp.StatusCode != http.StatusOK {
		return nil, "", httpRespToErrorResponse(resp, bucketName, objectName)
	}

	var out listObjectAnnotationsOutput
	if err := xml.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, "", err
	}

	annotations := make([]ObjectAnnotation, 0, len(out.Entries)+len(out.Flat))
	for _, a := range append(out.Entries, out.Flat...) {
		// A server that ignores the prefix lists every annotation.
		if !strings.HasPrefix(a.AnnotationName, opts.Prefix) {
			continue
		}
		// LastModified is an RFC3339 timestamp. A malformed value from the
		// server leaves LastModified as the zero time rather than failing the
		// entire listing for one bad entry.
		var lastModified time.Time
		if a.LastModified != "" {
			lastModified, _ = time.Parse(time.RFC3339, a.LastModified)
		}
		annotations = append(annotations, ObjectAnnotation{
			Name:              a.AnnotationName,
			Size:              a.Size,
			ETag:              trimEtag(a.ETag),
			LastModified:      lastModified,
			ChecksumAlgorithm: a.ChecksumAlgorithm,
		})
	}
	return annotations, out.NextContinuationToken, nil
}

// RemoveObjectAnnotation permanently deletes a single named annotation. Deletion
// is irreversible: annotations have no version history.
func (c *Client) RemoveObjectAnnotation(ctx context.Context, bucketName, objectName, annotationName string, opts RemoveObjectAnnotationOptions) error {
	if err := s3utils.CheckValidBucketName(bucketName); err != nil {
		return err
	}
	if err := s3utils.CheckValidObjectName(objectName); err != nil {
		return err
	}
	if err := validateAnnotationName(annotationName); err != nil {
		return err
	}

	headers := make(http.Header)
	if opts.IfMatch != "" {
		headers.Set(amzObjectIfMatchHeader, opts.IfMatch)
	}

	resp, err := c.executeMethod(ctx, http.MethodDelete, requestMetadata{
		bucketName:   bucketName,
		objectName:   objectName,
		queryValues:  annotationQueryValues(annotationName, opts.VersionID),
		customHeader: headers,
	})
	defer closeResponse(resp)
	if err != nil {
		return err
	}
	if resp != nil && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return httpRespToErrorResponse(resp, bucketName, objectName)
	}
	return nil
}
