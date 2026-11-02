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
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

// Cortex describes a memory bucket: a versioned bucket hidden from
// bucket listings whose objects are compressed and encrypted, with memory
// metadata stamped on every write.
type Cortex struct {
	Name        string    `json:"name"`
	Bucket      string    `json:"bucket"`
	UUID        string    `json:"uuid"`
	CreatedAt   time.Time `json:"createdAt"`
	Encryption  bool      `json:"encryption"`
	Compression bool      `json:"compression"`
	KMSKeyID    string    `json:"kmsKeyId,omitempty"`
	// Format is the memory format version stamped on the cortex's objects.
	Format string `json:"format"`
	// Upgraded is set for a cortex made from an existing bucket, which may
	// hold objects without the memory stamp.
	Upgraded bool `json:"upgraded,omitempty"`
	// Stale is set when the registry records the cortex but its bucket is
	// missing or a plain bucket.
	Stale    bool              `json:"stale,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// CortexRetentionRule expires the version history of the objects under a
// prefix, through bucket lifecycle rules.
type CortexRetentionRule struct {
	Prefix                  string `json:"prefix"`
	NoncurrentDays          int    `json:"noncurrentDays,omitempty"`
	NewerNoncurrentVersions int    `json:"newerNoncurrentVersions,omitempty"`
	ExpireDeleteMarkers     bool   `json:"expireDeleteMarkers,omitempty"`
}

// CreateCortexOptions configures a CreateCortex request.
type CreateCortexOptions struct {
	// KMSKeyID pins the KMS key the cortex's objects are encrypted with.
	// Empty uses the server's default key.
	KMSKeyID string
	// UpgradeExisting turns an existing plain bucket of the same name into
	// a cortex. Without it such a bucket is refused.
	UpgradeExisting bool
	// Metadata records operator-supplied information on the cortex.
	Metadata map[string]string
	// Retention expires version history under the given prefixes.
	Retention []CortexRetentionRule
}

// RemoveCortexOptions configures a RemoveCortex request.
type RemoveCortexOptions struct {
	// Force removes a cortex that still holds objects.
	Force bool
}

// CreateCortex creates the memory bucket name. It fails with
// CortexAlreadyExists when a cortex or a plain bucket of that name exists
// (unless UpgradeExisting is set), CortexRegistryStale when the registry
// records the cortex over a plain bucket, and KMSNotConfigured when the
// server has no KMS. The request is sent once.
func (c *Client) CreateCortex(ctx context.Context, name string, opts CreateCortexOptions) (Cortex, error) {
	if err := checkCortexName(name); err != nil {
		return Cortex{}, err
	}
	body, err := json.Marshal(struct {
		Name            string                `json:"name"`
		KMSKeyID        string                `json:"kmsKeyId,omitempty"`
		UpgradeExisting bool                  `json:"upgradeExisting,omitempty"`
		Metadata        map[string]string     `json:"metadata,omitempty"`
		Retention       []CortexRetentionRule `json:"retention,omitempty"`
	}{name, opts.KMSKeyID, opts.UpgradeExisting, opts.Metadata, opts.Retention})
	if err != nil {
		return Cortex{}, err
	}

	resp, err := c.executeMemoryMethod(ctx, memoryRequest{
		method:      http.MethodPost,
		body:        body,
		contentType: "application/json",
	})
	if err != nil {
		return Cortex{}, err
	}
	defer closeResponse(resp)

	var cortex Cortex
	if err := decodeMemoryJSON(resp.Body, &cortex); err != nil {
		return Cortex{}, err
	}
	return cortex, nil
}

// GetCortex returns the memory bucket name, or NoSuchCortex.
func (c *Client) GetCortex(ctx context.Context, name string) (Cortex, error) {
	if err := checkCortexName(name); err != nil {
		return Cortex{}, err
	}
	resp, err := c.executeMemoryMethod(ctx, memoryRequest{
		method: http.MethodGet,
		path:   []string{name},
		retry:  true,
	})
	if err != nil {
		return Cortex{}, err
	}
	defer closeResponse(resp)

	var cortex Cortex
	if err := decodeMemoryJSON(resp.Body, &cortex); err != nil {
		return Cortex{}, err
	}
	return cortex, nil
}

// RemoveCortex removes the memory bucket name. It fails with NoSuchCortex
// when there is none. The request is sent once.
func (c *Client) RemoveCortex(ctx context.Context, name string, opts RemoveCortexOptions) error {
	if err := checkCortexName(name); err != nil {
		return err
	}
	var query url.Values
	if opts.Force {
		query = url.Values{"force": {"true"}}
	}
	resp, err := c.executeMemoryMethod(ctx, memoryRequest{
		method: http.MethodDelete,
		path:   []string{name},
		query:  query,
	})
	if err != nil {
		return err
	}
	closeResponse(resp)
	return nil
}
