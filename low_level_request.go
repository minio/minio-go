/*
 * Minimal object storage library (C) 2015 Minio, Inc.
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

package objectstorage

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// operation - rest operation
type operation struct {
	HTTPServer string
	HTTPMethod string
	HTTPPath   string
}

// request - a http request
type request struct {
	req    *http.Request
	config *Config
}

// newRequest - instantiate a new request
func newRequest(op *operation, config *Config, body io.ReadCloser) (*request, error) {
	// if no method default to POST
	method := op.HTTPMethod
	if method == "" {
		method = "POST"
	}
	// parse URL for the combination of HTTPServer + HTTPPath
	u, err := url.Parse(op.HTTPServer + op.HTTPPath)
	if err != nil {
		return nil, err
	}
	// get a new HTTP request, for the requested method
	req, err := http.NewRequest(method, u.String(), nil)
	if err != nil {
		return nil, err
	}

	// add body
	req.Body = body
	// set UserAgent
	req.Header.Set("User-Agent", config.userAgent)

	// set ContentType, if available
	if config.ContentType != "" {
		req.Header.Set("Content-Type", config.ContentType)
	}
	// save for subsequent use
	r := new(request)
	r.config = config
	r.req = req
	return r, nil
}

// Do - start the request
func (r *request) Do() (resp *http.Response, err error) {
	r.SignV2()
	client := &http.Client{}
	return client.Do(r.req)
}

// Set - set additional headers if any
func (r *request) Set(key, value string) {
	r.req.Header.Set(key, value)
}

// Get - get header values
func (r *request) Get(key string) string {
	return r.req.Header.Get(key)
}

// SignV4 the request before Do() (version 4.0)
func (r *request) SignV4() {
	// TODO
}

// SignV2 the request before Do() (version 2.0)
func (r *request) SignV2() {
	// Add date if not present
	if date := r.Get("Date"); date == "" {
		r.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	}

	// calculate HMAC for the secretaccesskey
	hm := hmac.New(sha1.New, []byte(r.config.SecretAccessKey))
	ss := r.mustGetStringToSign()
	hm.Write([]byte(ss))

	// prepare auth header
	authHeader := new(bytes.Buffer)
	fmt.Fprintf(authHeader, "AWS %s:", r.config.AccessKeyID)
	encoder := base64.NewEncoder(base64.StdEncoding, authHeader)
	encoder.Write(hm.Sum(nil))
	encoder.Close()

	// Set Authorization header
	r.Set("Authorization", authHeader.String())
}

// From the Amazon docs:
//
// StringToSign = HTTP-Verb + "\n" +
// 	 Content-MD5 + "\n" +
//	 Content-Type + "\n" +
//	 Date + "\n" +
//	 CanonicalizedAmzHeaders +
//	 CanonicalizedResource;
func (r *request) mustGetStringToSign() string {
	buf := new(bytes.Buffer)
	// write standard headers
	r.mustWriteDefaultHeaders(buf)
	// write canonicalized AMZ headers if any
	r.mustWriteCanonicalizedAmzHeaders(buf)
	// write canonicalized Query resources if any
	r.mustWriteCanonicalizedResource(buf)
	return buf.String()
}

func (r *request) mustWriteDefaultHeaders(buf *bytes.Buffer) {
	buf.WriteString(r.req.Method)
	buf.WriteByte('\n')
	buf.WriteString(r.req.Header.Get("Content-MD5"))
	buf.WriteByte('\n')
	buf.WriteString(r.req.Header.Get("Content-Type"))
	buf.WriteByte('\n')
	buf.WriteString(r.req.Header.Get("Date"))
	buf.WriteByte('\n')
}

func (r *request) mustWriteCanonicalizedAmzHeaders(buf *bytes.Buffer) {
	var amzHeaders []string
	vals := make(map[string][]string)
	for k, vv := range r.req.Header {
		// all the AMZ headers go lower
		if isPrefixCaseInsensitive(k, "x-amz-") {
			lk := strings.ToLower(k)
			amzHeaders = append(amzHeaders, lk)
			vals[lk] = vv
		}
	}
	sort.Strings(amzHeaders)
	for _, k := range amzHeaders {
		buf.WriteString(k)
		buf.WriteByte(':')
		for idx, v := range vals[k] {
			if idx > 0 {
				buf.WriteByte(',')
			}
			if strings.Contains(v, "\n") {
				// TODO: "Unfold" long headers that
				// span multiple lines (as allowed by
				// RFC 2616, section 4.2) by replacing
				// the folding white-space (including
				// new-line) by a single space.
				buf.WriteString(v)
			} else {
				buf.WriteString(v)
			}
		}
		buf.WriteByte('\n')
	}
}

// Must be sorted:
var resourceList = []string{
	"acl",
	"location",
	"logging",
	"notification",
	"partNumber",
	"policy",
	"response-content-type",
	"response-content-language",
	"response-expires",
	"response-cache-control",
	"response-content-disposition",
	"response-content-encoding",
	"requestPayment",
	"torrent",
	"uploadId",
	"uploads",
	"versionId",
	"versioning",
	"versions",
	"website",
}

// From the Amazon docs:
//
// CanonicalizedResource = [ "/" + Bucket ] +
// 	  <HTTP-Request-URI, from the protocol name up to the query string> +
// 	  [ sub-resource, if present. For example "?acl", "?location", "?logging", or "?torrent"];
func (r *request) mustWriteCanonicalizedResource(buf *bytes.Buffer) {
	requestURL := r.req.URL
	buf.WriteString(requestURL.Path)
	sort.Strings(resourceList)
	if requestURL.RawQuery != "" {
		var n int
		vals, _ := url.ParseQuery(requestURL.RawQuery)
		// loop through all the supported resourceList
		for _, resource := range resourceList {
			if vv, ok := vals[resource]; ok && len(vv) > 0 {
				n++
				// first element
				switch n {
				case 1:
					buf.WriteByte('?')
				// the rest
				default:
					buf.WriteByte('&')
				}
				buf.WriteString(resource)
				// request parameters
				if len(vv[0]) > 0 {
					buf.WriteByte('=')
					buf.WriteString(url.QueryEscape(vv[0]))
				}
			}
		}
	}
}
