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
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// ObjectBioOptions configures a GetObjectBio request.
type ObjectBioOptions struct {
	// Limit returns at most this many versions. Zero returns them all.
	Limit int
	// After resumes from the cursor a previous page's NextAfter returned.
	After string
	// Annotations lists the names and sizes of the current version's
	// annotations.
	Annotations bool
}

// ObjectBioAnnotation is one annotation of an object version.
type ObjectBioAnnotation struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	// ChecksumAlgo is CRC32, CRC32C, CRC64NVME, SHA1, SHA256, SHA512,
	// XXHASH64, XXHASH3 or XXHASH128. Empty if the annotation has none.
	ChecksumAlgo string `json:"checksumAlgo,omitempty"`
	// ChecksumValue is the Base64 encoded checksum.
	ChecksumValue string `json:"checksum,omitempty"`
}

// bioChecksumTypes are the algorithms an annotation checksum can name.
var bioChecksumTypes = []ChecksumType{
	ChecksumCRC32, ChecksumCRC32C, ChecksumCRC64NVME, ChecksumSHA1, ChecksumSHA256,
	ChecksumSHA512, ChecksumXXHash64, ChecksumXXHash3, ChecksumXXHash128,
}

// Checksum returns the annotation's checksum. It is not set, with no error,
// when the annotation has none, and an error when the algorithm is unknown or
// the value does not decode to a checksum of that algorithm.
func (o ObjectBioAnnotation) Checksum() (Checksum, error) {
	if o.ChecksumAlgo == "" && o.ChecksumValue == "" {
		return Checksum{}, nil
	}
	for _, t := range bioChecksumTypes {
		if t.String() != o.ChecksumAlgo {
			continue
		}
		// NewChecksumString drops the decode error, and a value with trailing
		// garbage still decodes to bytes of the right length.
		b, err := base64.StdEncoding.DecodeString(o.ChecksumValue)
		c := NewChecksum(t, b)
		if err != nil || !c.IsSet() {
			return Checksum{}, fmt.Errorf("annotation %q: %q is not a valid %s checksum", o.Name, o.ChecksumValue, o.ChecksumAlgo)
		}
		return c, nil
	}
	return Checksum{}, fmt.Errorf("annotation %q: unknown checksum algorithm %q", o.Name, o.ChecksumAlgo)
}

// ObjectBioEntry is one version of an object's history.
type ObjectBioEntry struct {
	VersionID string
	WrittenAt time.Time
	// Principal and Agent are the server-attested writer of the version.
	Principal string
	Agent     string
	// Session is the agent-asserted session the version was written in.
	Session string
	// Stamps are the server-written provenance values, Tags the version's
	// object tags and Meta its user metadata, keyed as the server sends them.
	Stamps      map[string]string
	Tags        map[string]string
	Meta        map[string]string
	Current     bool
	Deleted     bool
	Annotations []ObjectBioAnnotation
}

// ObjectBio is one page of an object's history, newest version first.
type ObjectBio struct {
	Key     string
	Entries []ObjectBioEntry
	// TotalVersions counts the whole history, not only this page.
	TotalVersions int
	// Truncated is set when versions follow this page; NextAfter resumes
	// after it.
	Truncated bool
	NextAfter string
}

type objectBioFrame struct {
	Type    string `json:"type"`
	Key     string `json:"key"`
	Version *struct {
		Version     string                `json:"version"`
		WrittenAt   int64                 `json:"writtenAt"`
		Principal   string                `json:"principal"`
		Agent       string                `json:"agent"`
		Session     string                `json:"session"`
		Stamps      map[string]string     `json:"stamps"`
		Tags        map[string]string     `json:"tags"`
		Meta        map[string]string     `json:"meta"`
		Current     bool                  `json:"current"`
		Deleted     bool                  `json:"deleted"`
		Annotations []ObjectBioAnnotation `json:"annotations"`
	} `json:"version"`
	TotalVersions int    `json:"totalVersions"`
	Truncated     bool   `json:"truncated"`
	NextAfter     string `json:"nextAfter"`
	Error         string `json:"error"`
}

// GetObjectBio returns the history of the object key in cortex: each version
// with its server-attested writer. The answer streams; one that ends before
// its end frame, or whose end frame reports a failure, is refused, so a
// dropped connection never reads as a complete history.
func (c *Client) GetObjectBio(ctx context.Context, cortex, key string, opts ObjectBioOptions) (ObjectBio, error) {
	if err := checkCortexName(cortex); err != nil {
		return ObjectBio{}, err
	}
	if key == "" {
		return ObjectBio{}, errInvalidArgument("object key cannot be empty")
	}
	if opts.Limit < 0 {
		return ObjectBio{}, errInvalidArgument("limit cannot be negative")
	}
	query := url.Values{"key": {key}}
	if opts.Limit > 0 {
		query.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.After != "" {
		query.Set("after", opts.After)
	}
	if opts.Annotations {
		query.Set("annotations", "true")
	}
	resp, err := c.executeMemoryMethod(ctx, memoryRequest{
		method: http.MethodGet,
		path:   []string{cortex, "bio"},
		query:  query,
		retry:  true,
	})
	if err != nil {
		return ObjectBio{}, err
	}
	defer closeMemoryResponse(resp)
	return readObjectBio(resp.Body)
}

func readObjectBio(r io.Reader) (ObjectBio, error) {
	var bio ObjectBio
	ended := false
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64<<10), maxMemoryFrameBytes)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var f objectBioFrame
		if err := json.Unmarshal(line, &f); err != nil {
			return ObjectBio{}, fmt.Errorf("bio frame: %w", err)
		}
		switch f.Type {
		case "begin":
			bio.Key = f.Key
		case "version":
			// Skipping it would leave a history short of what the end
			// frame reports, with no error.
			if f.Version == nil {
				return ObjectBio{}, errors.New("bio: version frame has no version")
			}
			v := f.Version
			bio.Entries = append(bio.Entries, ObjectBioEntry{
				VersionID:   v.Version,
				WrittenAt:   time.UnixMilli(v.WrittenAt).UTC(),
				Principal:   v.Principal,
				Agent:       v.Agent,
				Session:     v.Session,
				Stamps:      v.Stamps,
				Tags:        v.Tags,
				Meta:        v.Meta,
				Current:     v.Current,
				Deleted:     v.Deleted,
				Annotations: v.Annotations,
			})
		case "end":
			if f.Error != "" {
				return ObjectBio{}, fmt.Errorf("bio: %s", f.Error)
			}
			// An empty cursor passed back as After would fetch the first
			// page again.
			if f.Truncated && f.NextAfter == "" {
				return ObjectBio{}, errors.New("bio stream is truncated without a nextAfter cursor; the history is incomplete")
			}
			bio.TotalVersions = f.TotalVersions
			bio.Truncated = f.Truncated
			bio.NextAfter = f.NextAfter
			ended = true
		}
	}
	if err := sc.Err(); err != nil {
		return ObjectBio{}, fmt.Errorf("bio stream: %w", err)
	}
	if !ended {
		return ObjectBio{}, errors.New("bio stream ended without its end frame; the history is incomplete")
	}
	return bio, nil
}
