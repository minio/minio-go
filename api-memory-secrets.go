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
	"fmt"
	"io"
	"iter"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// A cortex's secret store keeps opaque values, encrypted with the server's
// KMS, that S3 can neither list nor read in plain text.

const (
	secretVersionHeader = "X-Mem-Secret-Version"
	secretCreatedHeader = "X-Mem-Secret-Created"
	secretUpdatedHeader = "X-Mem-Secret-Updated"
)

// CortexSecretInfo describes one stored secret value.
type CortexSecretInfo struct {
	// Version counts the writes of the secret.
	Version int
	Created time.Time
	Updated time.Time
}

// ListCortexSecretsOptions configures a ListCortexSecrets request.
type ListCortexSecretsOptions struct {
	// Prefix lists only the secrets whose names start with it.
	Prefix string
	// MaxKeys bounds each page the server returns. Zero leaves the server's
	// default.
	MaxKeys int
}

func secretInfoFromHeader(h http.Header) (CortexSecretInfo, error) {
	var info CortexSecretInfo
	var err error
	if v := h.Get(secretVersionHeader); v != "" {
		if info.Version, err = strconv.Atoi(v); err != nil {
			return CortexSecretInfo{}, fmt.Errorf("invalid %s header %q", secretVersionHeader, v)
		}
	}
	if v := h.Get(secretCreatedHeader); v != "" {
		if info.Created, err = time.Parse(time.RFC3339Nano, v); err != nil {
			return CortexSecretInfo{}, fmt.Errorf("invalid %s header %q", secretCreatedHeader, v)
		}
	}
	if v := h.Get(secretUpdatedHeader); v != "" {
		if info.Updated, err = time.Parse(time.RFC3339Nano, v); err != nil {
			return CortexSecretInfo{}, fmt.Errorf("invalid %s header %q", secretUpdatedHeader, v)
		}
	}
	return info, nil
}

// PutCortexSecret stores value, at most 1 MiB, under name in cortex,
// replacing any value stored there. Each put adds a version, so the request
// is sent once.
func (c *Client) PutCortexSecret(ctx context.Context, cortex, name string, value []byte) (CortexSecretInfo, error) {
	if err := checkCortexName(cortex); err != nil {
		return CortexSecretInfo{}, err
	}
	if err := checkMemoryResourceName("secret", name); err != nil {
		return CortexSecretInfo{}, err
	}
	if len(value) > maxSecretValueBytes {
		return CortexSecretInfo{}, errInvalidArgument(fmt.Sprintf("secret value is larger than %d bytes", maxSecretValueBytes))
	}
	if value == nil {
		value = []byte{}
	}
	resp, err := c.executeMemoryMethod(ctx, memoryRequest{
		method:      http.MethodPut,
		path:        []string{cortex, "secrets", name},
		body:        value,
		contentType: "application/octet-stream",
		// The hash of a weak secret, such as a password, is enough to
		// recover it, and the server traces request headers.
		hidePayloadHash: true,
	})
	if err != nil {
		return CortexSecretInfo{}, err
	}
	defer closeMemoryResponse(resp)
	return secretInfoFromHeader(resp.Header)
}

// GetCortexSecret returns the value stored under name in cortex, or
// NoSuchSecret.
func (c *Client) GetCortexSecret(ctx context.Context, cortex, name string) ([]byte, CortexSecretInfo, error) {
	if err := checkCortexName(cortex); err != nil {
		return nil, CortexSecretInfo{}, err
	}
	if err := checkMemoryResourceName("secret", name); err != nil {
		return nil, CortexSecretInfo{}, err
	}
	resp, err := c.executeMemoryMethod(ctx, memoryRequest{
		method: http.MethodGet,
		path:   []string{cortex, "secrets", name},
		retry:  true,
	})
	if err != nil {
		return nil, CortexSecretInfo{}, err
	}
	defer closeMemoryResponse(resp)

	info, err := secretInfoFromHeader(resp.Header)
	if err != nil {
		return nil, CortexSecretInfo{}, err
	}
	value, err := io.ReadAll(io.LimitReader(resp.Body, maxSecretValueBytes+1))
	if err != nil {
		return nil, CortexSecretInfo{}, err
	}
	if len(value) > maxSecretValueBytes {
		return nil, CortexSecretInfo{}, fmt.Errorf("secret %s is larger than %d bytes", name, maxSecretValueBytes)
	}
	return value, info, nil
}

// StatCortexSecret returns what is known of the secret name in cortex
// without reading its value, or NoSuchSecret.
func (c *Client) StatCortexSecret(ctx context.Context, cortex, name string) (CortexSecretInfo, error) {
	if err := checkCortexName(cortex); err != nil {
		return CortexSecretInfo{}, err
	}
	if err := checkMemoryResourceName("secret", name); err != nil {
		return CortexSecretInfo{}, err
	}
	resp, err := c.executeMemoryMethod(ctx, memoryRequest{
		method: http.MethodHead,
		path:   []string{cortex, "secrets", name},
		retry:  true,
	})
	if err != nil {
		// A HEAD response carries no body, so its error has no code, and a
		// 404 may mean a missing cortex as well as a missing secret. Ask for
		// the cortex before naming the secret as missing.
		if errResp := ToErrorResponse(err); errResp.Code == "" && errResp.StatusCode == http.StatusNotFound {
			if _, cerr := c.GetCortex(ctx, cortex); cerr != nil {
				if ToErrorResponse(cerr).Code == NoSuchCortex {
					return CortexSecretInfo{}, cerr
				}
				return CortexSecretInfo{}, err
			}
			errResp.Code = NoSuchSecret
			errResp.Message = "The specified secret does not exist."
			return CortexSecretInfo{}, errResp
		}
		return CortexSecretInfo{}, err
	}
	defer closeMemoryResponse(resp)
	return secretInfoFromHeader(resp.Header)
}

// RemoveCortexSecret removes the secret name from cortex. It fails with
// NoSuchSecret when there is none. The request is sent once.
func (c *Client) RemoveCortexSecret(ctx context.Context, cortex, name string) error {
	if err := checkCortexName(cortex); err != nil {
		return err
	}
	if err := checkMemoryResourceName("secret", name); err != nil {
		return err
	}
	resp, err := c.executeMemoryMethod(ctx, memoryRequest{
		method: http.MethodDelete,
		path:   []string{cortex, "secrets", name},
	})
	if err != nil {
		return err
	}
	closeMemoryResponse(resp)
	return nil
}

// ListCortexSecrets lists the names of the secrets in cortex, in order,
// following the listing's pages to the end. A failure ends the iteration with
// its error.
func (c *Client) ListCortexSecrets(ctx context.Context, cortex string, opts ListCortexSecretsOptions) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		if err := checkCortexName(cortex); err != nil {
			yield("", err)
			return
		}
		marker := ""
		seen := map[string]bool{}
		for {
			query := url.Values{}
			if opts.Prefix != "" {
				query.Set("prefix", opts.Prefix)
			}
			if opts.MaxKeys > 0 {
				query.Set("max-keys", strconv.Itoa(opts.MaxKeys))
			}
			if marker != "" {
				query.Set("marker", marker)
			}
			page, err := c.listCortexSecretsPage(ctx, cortex, query)
			if err != nil {
				yield("", err)
				return
			}
			for _, name := range page.Secrets {
				if !yield(name, nil) {
					return
				}
			}
			if !page.IsTruncated {
				return
			}
			// A truncated page must lead to a marker not seen before, or the
			// listing would repeat pages forever.
			if page.NextMarker == "" || seen[page.NextMarker] {
				yield("", fmt.Errorf("secret listing of %s did not advance past marker %q", cortex, marker))
				return
			}
			seen[page.NextMarker] = true
			marker = page.NextMarker
		}
	}
}

type listCortexSecretsPage struct {
	Secrets     []string `json:"secrets"`
	NextMarker  string   `json:"nextMarker"`
	IsTruncated bool     `json:"isTruncated"`
}

func (c *Client) listCortexSecretsPage(ctx context.Context, cortex string, query url.Values) (listCortexSecretsPage, error) {
	resp, err := c.executeMemoryMethod(ctx, memoryRequest{
		method: http.MethodGet,
		path:   []string{cortex, "secrets"},
		query:  query,
		retry:  true,
	})
	if err != nil {
		return listCortexSecretsPage{}, err
	}
	defer closeMemoryResponse(resp)

	var page listCortexSecretsPage
	if err := decodeMemoryJSON(resp.Body, &page); err != nil {
		return listCortexSecretsPage{}, err
	}
	return page, nil
}
