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
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// An agent record in a cortex declares an agent: what it mounts and the
// grants it holds. Creating one makes the server mint the agent an identity with
// a policy derived from the record. The record is JSON whose schema the server
// owns, so these calls take and return it as raw bytes.

// CortexAgentCredentials is the key pair the server mints for an agent. The
// secret key is returned only by the create that minted it.
type CortexAgentCredentials struct {
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
}

// CortexAgentTelemetry names the warehouse and namespace the agent's
// credential may write its telemetry to.
type CortexAgentTelemetry struct {
	Warehouse string `json:"warehouse"`
	Namespace string `json:"namespace"`
}

// CreateCortexAgentResult is the answer to a CreateCortexAgent request.
type CreateCortexAgentResult struct {
	// Agent is the record as stored.
	Agent       json.RawMessage        `json:"agent"`
	Credentials CortexAgentCredentials `json:"credentials"`
	Telemetry   CortexAgentTelemetry   `json:"telemetry"`
}

// CortexAgentInfo describes a stored agent record.
type CortexAgentInfo struct {
	// ETag is what UpdateCortexAgent's IfMatch names to update this version
	// of the record.
	ETag         string
	LastModified time.Time
	VersionID    string
}

// UpdateCortexAgentOptions configures an UpdateCortexAgent request.
type UpdateCortexAgentOptions struct {
	// IfMatch is the ETag of the record the update was made from. The server
	// requires it, and refuses the update when the record has changed since.
	IfMatch string
}

func agentInfoFromHeader(h http.Header) CortexAgentInfo {
	info := CortexAgentInfo{
		ETag:      trimEtag(h.Get("ETag")),
		VersionID: h.Get(amzVersionID),
	}
	if lm := h.Get("Last-Modified"); lm != "" {
		info.LastModified, _ = time.Parse(http.TimeFormat, lm)
	}
	return info
}

func readAgentRecord(r io.Reader) ([]byte, error) {
	record, err := io.ReadAll(io.LimitReader(r, maxMemoryDocumentBytes+1))
	if err != nil {
		return nil, err
	}
	if len(record) > maxMemoryDocumentBytes {
		return nil, fmt.Errorf("agent record is larger than %d bytes", maxMemoryDocumentBytes)
	}
	return record, nil
}

// CreateCortexAgent creates the agent record agentID in cortex from record,
// a JSON agent record whose id is agentID, and returns the credentials the server
// minted for it. It fails with AgentAlreadyExists when the record exists and
// AgentPurgePending while the server still reclaims a deleted agent of that id.
//
// The request is sent once: the secret key is returned by this response
// alone, and a retry after a lost response would answer AgentAlreadyExists
// with no way to recover it.
func (c *Client) CreateCortexAgent(ctx context.Context, cortex, agentID string, record []byte) (CreateCortexAgentResult, error) {
	if err := checkCortexName(cortex); err != nil {
		return CreateCortexAgentResult{}, err
	}
	if err := checkMemoryResourceName("agent", agentID); err != nil {
		return CreateCortexAgentResult{}, err
	}
	resp, err := c.executeMemoryMethod(ctx, memoryRequest{
		method:      http.MethodPut,
		path:        []string{cortex, "agents", agentID},
		body:        record,
		contentType: "application/json",
	})
	if err != nil {
		return CreateCortexAgentResult{}, err
	}
	defer closeMemoryResponse(resp)

	var result CreateCortexAgentResult
	if err := decodeMemoryJSON(resp.Body, &result); err != nil {
		return CreateCortexAgentResult{}, err
	}
	// The secret key is returned by this call alone; a response without one
	// is a failure the caller must see, not an agent it cannot sign as.
	if result.Credentials.SecretKey == "" {
		return CreateCortexAgentResult{}, fmt.Errorf("creating agent %s returned no secret key", agentID)
	}
	return result, nil
}

// GetCortexAgent returns the agent record agentID in cortex, or NoSuchAgent.
func (c *Client) GetCortexAgent(ctx context.Context, cortex, agentID string) ([]byte, CortexAgentInfo, error) {
	if err := checkCortexName(cortex); err != nil {
		return nil, CortexAgentInfo{}, err
	}
	if err := checkMemoryResourceName("agent", agentID); err != nil {
		return nil, CortexAgentInfo{}, err
	}
	resp, err := c.executeMemoryMethod(ctx, memoryRequest{
		method: http.MethodGet,
		path:   []string{cortex, "agents", agentID},
		retry:  true,
	})
	if err != nil {
		return nil, CortexAgentInfo{}, err
	}
	defer closeMemoryResponse(resp)

	record, err := readAgentRecord(resp.Body)
	if err != nil {
		return nil, CortexAgentInfo{}, err
	}
	return record, agentInfoFromHeader(resp.Header), nil
}

// UpdateCortexAgent applies update, a JSON partial agent record, to the
// agent record agentID in cortex, keeping its identity and key, and returns
// the record as stored. It fails with PreconditionFailed when the record
// changed since opts.IfMatch was read. The request is sent once.
func (c *Client) UpdateCortexAgent(ctx context.Context, cortex, agentID string, update []byte, opts UpdateCortexAgentOptions) ([]byte, CortexAgentInfo, error) {
	if err := checkCortexName(cortex); err != nil {
		return nil, CortexAgentInfo{}, err
	}
	if err := checkMemoryResourceName("agent", agentID); err != nil {
		return nil, CortexAgentInfo{}, err
	}
	if opts.IfMatch == "" {
		return nil, CortexAgentInfo{}, errInvalidArgument("an agent update needs the ETag of the record it was made from")
	}
	resp, err := c.executeMemoryMethod(ctx, memoryRequest{
		method:      http.MethodPut,
		path:        []string{cortex, "agents", agentID},
		query:       url.Values{"update": {""}},
		header:      http.Header{"If-Match": {"\"" + trimEtag(opts.IfMatch) + "\""}},
		body:        update,
		contentType: "application/json",
	})
	if err != nil {
		return nil, CortexAgentInfo{}, err
	}
	defer closeMemoryResponse(resp)

	record, err := readAgentRecord(resp.Body)
	if err != nil {
		return nil, CortexAgentInfo{}, err
	}
	return record, agentInfoFromHeader(resp.Header), nil
}

// RemoveCortexAgent removes the agent record agentID from cortex, with the
// identity the server minted for it; the server reclaims what the agent stored
// afterwards. It fails with NoSuchAgent when there is none. The request is
// sent once.
func (c *Client) RemoveCortexAgent(ctx context.Context, cortex, agentID string) error {
	if err := checkCortexName(cortex); err != nil {
		return err
	}
	if err := checkMemoryResourceName("agent", agentID); err != nil {
		return err
	}
	resp, err := c.executeMemoryMethod(ctx, memoryRequest{
		method: http.MethodDelete,
		path:   []string{cortex, "agents", agentID},
	})
	if err != nil {
		return err
	}
	closeMemoryResponse(resp)
	return nil
}
