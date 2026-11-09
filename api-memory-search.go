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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
)

// CortexSearchRequest is a content search over the objects under a prefix of
// a cortex. Every pattern must match.
type CortexSearchRequest struct {
	// Prefix scopes the search and is required. It is sent in the query as
	// well as the body, so a policy can scope the grant by memory:prefix.
	Prefix   string   `json:"prefix"`
	Patterns []string `json:"patterns"`
	// Extensions searches only objects with one of these name extensions.
	Extensions []string `json:"extensions,omitempty"`
	IgnoreCase bool     `json:"ignoreCase,omitempty"`
	// Recursive searches below the prefix's direct children. Nil leaves the
	// server's default, which is recursive.
	Recursive *bool `json:"recursive,omitempty"`
	// MaxFileSize skips objects larger than this; MaxTotalBytes bounds the
	// match text returned. Zero leaves the server's defaults.
	MaxFileSize   int64 `json:"maxFileSize,omitempty"`
	MaxTotalBytes int64 `json:"maxTotalBytes,omitempty"`
	// Multiline matches a pattern across lines.
	Multiline bool `json:"multiline,omitempty"`
	// Before, After and Context ask for that many lines of context around
	// each match.
	Before  int `json:"before,omitempty"`
	After   int `json:"after,omitempty"`
	Context int `json:"context,omitempty"`
}

// CortexSearchOptions configures a SearchCortex request.
type CortexSearchOptions struct {
	// MaxResponseBytes bounds the response read. Zero is 16 MiB, or, when
	// the request sets MaxTotalBytes, twice that plus 16 MiB, which leaves
	// room for the JSON around each match.
	MaxResponseBytes int64
}

// CortexSearchMatch is one match in an object.
type CortexSearchMatch struct {
	// Line and Column are 1-based; Column counts bytes.
	Line   int
	Column int
	// Text is the matching line, or the matched span of a multiline search.
	Text   string
	Before []string
	After  []string
}

// CortexSearchObject is one object with matches.
type CortexSearchObject struct {
	Name    string
	Matches []CortexSearchMatch
}

// CortexSearchStats summarizes a search.
type CortexSearchStats struct {
	FilesScanned    int64 `json:"filesScanned"`
	FilesMatched    int64 `json:"filesMatched"`
	FilesSkipped    int64 `json:"filesSkipped"`
	Errors          int64 `json:"errors"`
	MatchesReturned int64 `json:"matchesReturned"`
	BlocksSkipped   int64 `json:"blocksSkipped"`
}

// CortexSearchResult is the answer to a search.
type CortexSearchResult struct {
	Objects []CortexSearchObject
	Stats   CortexSearchStats
	// Truncated is set when the server stopped at MaxTotalBytes before
	// covering the prefix.
	Truncated bool
}

// Partial reports whether the result may not cover the whole prefix: the
// server truncated it, skipped objects, or failed to read some.
func (r CortexSearchResult) Partial() bool {
	return r.Truncated || r.Stats.FilesSkipped > 0 || r.Stats.Errors > 0
}

type cortexSearchFrame struct {
	Type      string             `json:"type"`
	Name      string             `json:"name"`
	Line      int                `json:"line"`
	Column    int                `json:"column"`
	Text      string             `json:"text"`
	Before    []string           `json:"before"`
	After     []string           `json:"after"`
	Stats     *CortexSearchStats `json:"stats"`
	Truncated bool               `json:"truncated"`
}

// SearchCortex searches the content of the objects under req.Prefix in
// cortex. The answer streams; one that ends before its summary is refused,
// so a dropped connection never reads as a complete search.
func (c *Client) SearchCortex(ctx context.Context, cortex string, req CortexSearchRequest, opts CortexSearchOptions) (CortexSearchResult, error) {
	if err := checkCortexName(cortex); err != nil {
		return CortexSearchResult{}, err
	}
	if strings.Trim(req.Prefix, "/") == "" {
		return CortexSearchResult{}, errInvalidArgument("a search needs a prefix")
	}
	if len(req.Patterns) == 0 {
		return CortexSearchResult{}, errInvalidArgument("a search needs a pattern")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return CortexSearchResult{}, err
	}
	resp, err := c.executeMemoryMethod(ctx, memoryRequest{
		method:      http.MethodPost,
		path:        []string{cortex, "search"},
		query:       url.Values{"prefix": {req.Prefix}},
		body:        body,
		contentType: "application/json",
		retry:       true,
	})
	if err != nil {
		return CortexSearchResult{}, err
	}
	defer closeMemoryResponse(resp)

	limit := opts.MaxResponseBytes
	if limit <= 0 {
		limit = defaultSearchResponseLimit(req.MaxTotalBytes)
	}
	lr := &io.LimitedReader{R: resp.Body, N: limit}
	res, err := readCortexSearch(lr)
	if err != nil && lr.N <= 0 {
		return CortexSearchResult{}, fmt.Errorf("search response is larger than the client's limit of %d bytes, set by CortexSearchOptions.MaxResponseBytes: %w", limit, err)
	}
	return res, err
}

// defaultSearchResponseLimit bounds the response to a search whose match text
// the server bounds by maxTotalBytes. The JSON around each match and the
// context lines add to that text, and the server checks its bound only before
// each object, so a response may exceed it: the bound is doubled, with the
// default bound added for headroom.
func defaultSearchResponseLimit(maxTotalBytes int64) int64 {
	if maxTotalBytes <= 0 {
		return defaultMaxSearchResponseBytes
	}
	if maxTotalBytes > (math.MaxInt64-defaultMaxSearchResponseBytes)/2 {
		return math.MaxInt64
	}
	return 2*maxTotalBytes + defaultMaxSearchResponseBytes
}

func readCortexSearch(r io.Reader) (CortexSearchResult, error) {
	var res CortexSearchResult
	var cur *CortexSearchObject
	summary := false
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64<<10), maxMemoryFrameBytes)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var f cortexSearchFrame
		if err := json.Unmarshal(line, &f); err != nil {
			return CortexSearchResult{}, fmt.Errorf("search frame: %w", err)
		}
		switch f.Type {
		case "begin":
			if cur != nil {
				return CortexSearchResult{}, errors.New("search: begin frame inside an object")
			}
			cur = &CortexSearchObject{Name: f.Name}
		case "match":
			if cur == nil {
				return CortexSearchResult{}, errors.New("search: match frame outside an object")
			}
			cur.Matches = append(cur.Matches, CortexSearchMatch{
				Line: f.Line, Column: f.Column, Text: f.Text, Before: f.Before, After: f.After,
			})
		case "end":
			if cur == nil {
				return CortexSearchResult{}, errors.New("search: end frame outside an object")
			}
			res.Objects = append(res.Objects, *cur)
			cur = nil
		case "summary":
			if f.Stats != nil {
				res.Stats = *f.Stats
			}
			res.Truncated = f.Truncated
			summary = true
		}
	}
	if err := sc.Err(); err != nil {
		return CortexSearchResult{}, fmt.Errorf("search stream: %w", err)
	}
	if cur != nil || !summary {
		return CortexSearchResult{}, errors.New("search stream ended without its summary; the result is incomplete")
	}
	return res, nil
}
