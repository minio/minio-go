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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/minio/minio-go/v7/pkg/s3utils"
)

// The Memory API (/_mem/v1) manages memory buckets, which the API calls
// cortexes, and what lives beside their objects: secrets, agent records,
// content search and object biographies. Its errors are JSON, and they are
// returned as ErrorResponse with
// the Memory API's code, such as NoSuchCortex or NoSuchSecret. A response
// that is not a Memory API error, such as the S3 XML error of a server that
// does not serve the API, is returned as an ErrorResponse with an empty Code
// and the response body as its Message.

const memoryAPIPrefix = "/_mem/v1/cortexes"

// Memory API error codes a caller commonly acts on.
const (
	NoSuchCortex        = "NoSuchCortex"
	CortexAlreadyExists = "CortexAlreadyExists"
	CortexRegistryStale = "CortexRegistryStale"
	KMSNotConfigured    = "KMSNotConfigured"
	NoSuchSecret        = "NoSuchSecret"
	NoSuchAgent         = "NoSuchAgent"
	AgentAlreadyExists  = "AgentAlreadyExists"
	AgentPurgePending   = "AgentPurgePending"
)

const (
	// maxMemoryErrorBytes bounds the error body read from a Memory API
	// response.
	maxMemoryErrorBytes = 64 << 10

	// maxSecretValueBytes is the largest secret value the server stores.
	maxSecretValueBytes = 1 << 20

	// maxMemoryDocumentBytes bounds a JSON document read from a Memory API
	// response: a cortex, a secret listing page or an agent record.
	maxMemoryDocumentBytes = 4 << 20

	// maxMemoryFrameBytes bounds one line of a streamed Memory API response.
	maxMemoryFrameBytes = 4 << 20

	// defaultMaxSearchResponseBytes bounds a search response when the caller
	// sets no bound of its own.
	defaultMaxSearchResponseBytes = 16 << 20
)

// memoryRequest is one Memory API request.
type memoryRequest struct {
	method      string
	path        []string // segments under /_mem/v1/cortexes
	query       url.Values
	header      http.Header
	body        []byte
	contentType string

	// retry is set only for a request whose repetition cannot change its
	// outcome: a read. A write is sent once, because a retry after a lost
	// response would answer AlreadyExists, a failed precondition or
	// NoSuch... for a request that succeeded, a secret put would add a
	// version, and an agent create would lose the key it minted.
	retry bool

	// hidePayloadHash keeps the hash of the body out of the request headers
	// over TLS, for a body the server keeps out of its traces: a secret.
	hidePayloadHash bool
}

// memoryPath joins segments under the Memory API prefix.
func memoryPath(segments ...string) string {
	if len(segments) == 0 {
		return memoryAPIPrefix
	}
	return memoryAPIPrefix + "/" + strings.Join(segments, "/")
}

// executeMemoryMethod sends a Memory API request, signed like any other
// request of the client, and returns the response of a successful one. The
// caller closes its body.
func (c *Client) executeMemoryMethod(ctx context.Context, mr memoryRequest) (*http.Response, error) {
	if ctx == nil {
		return nil, errInvalidArgument("context cannot be nil")
	}
	if c.IsOffline() {
		return nil, errors.New(c.endpointURL.String() + " is offline.")
	}

	header := mr.header.Clone()
	if header == nil {
		header = make(http.Header)
	}
	if mr.contentType != "" {
		header.Set("Content-Type", mr.contentType)
	}
	// The server traces request headers, so the hash of a body it keeps out
	// of its traces is not sent over TLS, which protects the body on its own:
	// the request is signed with UNSIGNED-PAYLOAD.
	var contentSHA256Hex string
	if !mr.hidePayloadHash || !c.secure {
		sum := sha256.Sum256(mr.body)
		contentSHA256Hex = hex.EncodeToString(sum[:])
	}
	metadata := requestMetadata{
		rawPath:          memoryPath(mr.path...),
		queryValues:      mr.query,
		customHeader:     header,
		contentLength:    int64(len(mr.body)),
		contentSHA256Hex: contentSHA256Hex,
	}

	reqRetry := 1
	if mr.retry {
		reqRetry = c.maxRetries
	}
	// A non-positive MaxRetry would yield no attempt and a nil response
	// with a nil error.
	if reqRetry < 1 {
		reqRetry = 1
	}

	// A retryable failure on the last attempt ends the loop at once, so the
	// call does not wait out a backoff it will not use and then report the
	// context's expiry in place of that failure.
	var err error
	regionRetried := false
	for attempt := range c.newRetryTimer(ctx, reqRetry, DefaultRetryUnit, DefaultRetryCap, MaxJitter) {
		var res *http.Response
		var retryable bool
		res, retryable, err = c.sendMemoryRequest(ctx, mr, metadata)
		// A server in another region refuses the signature before it acts,
		// so the request, even a write, is sent once more signed for the
		// region the server names.
		if region := c.memoryRedirectRegion(err, metadata.bucketLocation); region != "" && !regionRetried {
			regionRetried = true
			metadata.bucketLocation = region
			res, retryable, err = c.sendMemoryRequest(ctx, mr, metadata)
		}
		if err == nil {
			return res, nil
		}
		if !retryable {
			return nil, err
		}
		if attempt == reqRetry-1 {
			break
		}
	}

	if e := ctx.Err(); e != nil {
		return nil, e
	}
	return nil, err
}

// sendMemoryRequest sends mr once and returns the response of a successful
// request, or the failure and whether a later attempt may succeed.
func (c *Client) sendMemoryRequest(ctx context.Context, mr memoryRequest, metadata requestMetadata) (*http.Response, bool, error) {
	metadata.contentBody = bytes.NewReader(mr.body)
	req, err := c.newRequest(ctx, mr.method, metadata)
	if err != nil {
		return nil, isS3CodeRetryable(ToErrorResponse(err).Code), err
	}
	res, err := c.do(req)
	if err != nil {
		return nil, isRequestErrorRetryable(ctx, err), err
	}
	if successStatus.Contains(res.StatusCode) || res.StatusCode == http.StatusCreated {
		return res, false, nil
	}
	errResp := memoryErrorResponse(res)
	closeMemoryResponse(res)
	return nil, isS3CodeRetryable(errResp.Code) || isHTTPStatusRetryable(errResp.StatusCode), errResp
}

// memoryRedirectRegion returns the region named by a server that refused a
// request signed for another region, when the client sets no region of its
// own and the request was not already signed for that region.
func (c *Client) memoryRedirectRegion(err error, signedFor string) string {
	if c.region != "" {
		return ""
	}
	var errResp ErrorResponse
	if !errors.As(err, &errResp) {
		return ""
	}
	switch errResp.Code {
	case AuthorizationHeaderMalformed, InvalidRegion, AccessDenied:
		if errResp.Region != "" && errResp.Region != signedFor {
			return errResp.Region
		}
	}
	return ""
}

// memoryErrorResponse decodes a Memory API error. It never quotes the request,
// whose body may be a secret.
func memoryErrorResponse(res *http.Response) ErrorResponse {
	errResp := ErrorResponse{
		StatusCode: res.StatusCode,
		Server:     res.Header.Get("Server"),
	}
	body, _ := io.ReadAll(io.LimitReader(res.Body, maxMemoryErrorBytes))
	body = bytes.TrimSpace(body)

	var decoded ErrorResponse
	if json.Unmarshal(body, &decoded) == nil && decoded.Code != "" {
		decoded.StatusCode = errResp.StatusCode
		decoded.Server = errResp.Server
		return decoded
	}

	if len(body) > 0 {
		errResp.Message = fmt.Sprintf("HTTP %d: %s", res.StatusCode, body)
	} else {
		errResp.Message = fmt.Sprintf("HTTP %d %s", res.StatusCode, http.StatusText(res.StatusCode))
	}
	return errResp
}

// closeMemoryResponse closes resp. Unlike closeResponse it does not drain
// what is left unread: a response abandoned on an early return could keep
// the call waiting on a server that holds the body open. A body already read
// to its end still returns its connection for reuse.
func closeMemoryResponse(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
}

// decodeMemoryJSON decodes a JSON document from a Memory API response. A
// successful response is a JSON object, so a null document is refused rather
// than decoded into a zero value that reads as an empty answer.
func decodeMemoryJSON(r io.Reader, v any) error {
	body, err := io.ReadAll(io.LimitReader(r, maxMemoryDocumentBytes+1))
	if err != nil {
		return err
	}
	if len(body) > maxMemoryDocumentBytes {
		return fmt.Errorf("memory API response is larger than %d bytes", maxMemoryDocumentBytes)
	}
	if bytes.Equal(bytes.TrimSpace(body), []byte("null")) {
		return errors.New("memory API response is a null document")
	}
	return json.Unmarshal(body, v)
}

// checkCortexName refuses a cortex name that is not a valid bucket name.
func checkCortexName(cortex string) error {
	return s3utils.CheckValidBucketName(cortex)
}

// checkMemoryResourceName refuses a name that would leave the route it is
// meant for. The server applies the full naming rules.
func checkMemoryResourceName(kind, name string) error {
	switch {
	case name == "":
		return errInvalidArgument(kind + " name cannot be empty")
	case strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/"):
		return errInvalidArgument(kind + " name cannot start or end with '/'")
	}
	for _, segment := range strings.Split(name, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return errInvalidArgument(kind + " name cannot hold an empty, '.' or '..' segment")
		}
	}
	return nil
}
