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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// An agent record in a cortex declares an agent: what it mounts and the
// grants it holds. Creating one makes the server mint the agent an identity with
// a policy derived from the record. The record is JSON whose schema the server
// owns, so these calls take and return it as raw bytes.

// CortexAgentCredentials is the key pair the server mints for an agent. The
// secret key is returned only by the create that minted it. An agent record
// that declares no mounts gets no key pair, and both fields are empty.
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

// agentInfoFromHeader reads the version of an agent record from the response
// headers that answered for it.
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

// checkAgentID refuses an agent ID the server would not route to an agent
// record: it reads a name with a '/' as a memory stored under an agent, so a
// create would write that memory and a remove would delete it.
func checkAgentID(agentID string) error {
	if err := checkMemoryResourceName("agent", agentID); err != nil {
		return err
	}
	if strings.Contains(agentID, "/") {
		return errInvalidArgument("agent ID cannot hold '/'")
	}
	return nil
}

// readAgentRecord reads the agent record a successful GET or update answers
// with, and refuses a body that is not a JSON object.
func readAgentRecord(r io.Reader) ([]byte, error) {
	record, err := io.ReadAll(io.LimitReader(r, maxMemoryDocumentBytes+1))
	if err != nil {
		return nil, err
	}
	if len(record) > maxMemoryDocumentBytes {
		return nil, fmt.Errorf("agent record is larger than %d bytes", maxMemoryDocumentBytes)
	}
	if trimmed := bytes.TrimSpace(record); len(trimmed) == 0 || trimmed[0] != '{' || !json.Valid(trimmed) {
		return nil, errors.New("agent record is not a JSON object")
	}
	return record, nil
}

// CreateCortexAgent creates the agent record agentID in cortex from record,
// a JSON agent record whose id is agentID, and returns the credentials the server
// minted for it, which are empty when the record declares no mounts. It fails
// with AgentAlreadyExists when the record exists and AgentPurgePending while
// the server still reclaims a deleted agent of that id.
//
// The request is sent once: the secret key is returned by this response
// alone, and a retry after a lost response would answer AgentAlreadyExists
// with no way to recover it.
func (c *Client) CreateCortexAgent(ctx context.Context, cortex, agentID string, record []byte) (CreateCortexAgentResult, error) {
	if err := checkCortexName(cortex); err != nil {
		return CreateCortexAgentResult{}, err
	}
	if err := checkAgentID(agentID); err != nil {
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

	// An agent without mounts is stored with no credential, so empty
	// credentials are an answer, not a failure: the record exists, and a retry
	// would fail with AgentAlreadyExists.
	var result CreateCortexAgentResult
	if err := decodeMemoryJSON(resp.Body, &result); err != nil {
		return CreateCortexAgentResult{}, err
	}
	// Only both keys or neither is an answer: a lone key cannot be used, and
	// the secret cannot be fetched again.
	if (result.Credentials.AccessKey == "") != (result.Credentials.SecretKey == "") {
		return CreateCortexAgentResult{}, fmt.Errorf("creating agent %s returned incomplete credentials", agentID)
	}
	return result, nil
}

// GetCortexAgent returns the agent record agentID in cortex, or NoSuchAgent.
func (c *Client) GetCortexAgent(ctx context.Context, cortex, agentID string) ([]byte, CortexAgentInfo, error) {
	if err := checkCortexName(cortex); err != nil {
		return nil, CortexAgentInfo{}, err
	}
	if err := checkAgentID(agentID); err != nil {
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
	if err := checkAgentID(agentID); err != nil {
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
	if err := checkAgentID(agentID); err != nil {
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
