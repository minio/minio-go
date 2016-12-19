/*
 * Minio Go Library for Amazon S3 Compatible Cloud Storage (C) 2016 Minio, Inc.
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

package s3utils

import (
	"bytes"
	"net"
	"net/url"
	"sort"
	"strings"

	"github.com/minio/minio-go/pkg/s3signer"
)

// Sentinel URL is the default url value which is invalid.
var sentinelURL = url.URL{}

// IsValidDomain validates if input string is a valid domain name.
func IsValidDomain(host string) bool {
	// See RFC 1035, RFC 3696.
	host = strings.TrimSpace(host)
	if len(host) == 0 || len(host) > 255 {
		return false
	}
	// host cannot start or end with "-"
	if host[len(host)-1:] == "-" || host[:1] == "-" {
		return false
	}
	// host cannot start or end with "_"
	if host[len(host)-1:] == "_" || host[:1] == "_" {
		return false
	}
	// host cannot start or end with a "."
	if host[len(host)-1:] == "." || host[:1] == "." {
		return false
	}
	// All non alphanumeric characters are invalid.
	if strings.ContainsAny(host, "`~!@#$%^&*()+={}[]|\\\"';:><?/") {
		return false
	}
	// No need to regexp match, since the list is non-exhaustive.
	// We let it valid and fail later.
	return true
}

// IsValidIP parses input string for ip address validity.
func IsValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// IsVirtualHostSupported - verifies if bucketName can be part of
// virtual host. Currently only Amazon S3 and Google Cloud Storage
// would support this.
func IsVirtualHostSupported(endpointURL url.URL, bucketName string) bool {
	if endpointURL == sentinelURL {
		return false
	}
	// bucketName can be valid but '.' in the hostname will fail SSL
	// certificate validation. So do not use host-style for such buckets.
	if endpointURL.Scheme == "https" && strings.Contains(bucketName, ".") {
		return false
	}
	// Return true for all other cases
	return IsAmazonEndpoint(endpointURL) || IsGoogleEndpoint(endpointURL)
}

// Match if it is exactly Amazon S3 endpoint.
func IsAmazonEndpoint(endpointURL url.URL) bool {
	if IsAmazonChinaEndpoint(endpointURL) {
		return true
	}

	if IsAmazonS3AccelerateEndpoint(endpointURL) {
		return true
	}

	return endpointURL.Host == "s3.amazonaws.com"
}

// Match if it is exactly Amazon S3 China endpoint.
// Customers who wish to use the new Beijing Region are required
// to sign up for a separate set of account credentials unique to
// the China (Beijing) Region. Customers with existing AWS credentials
// will not be able to access resources in the new Region, and vice versa.
// For more info https://aws.amazon.com/about-aws/whats-new/2013/12/18/announcing-the-aws-china-beijing-region/
func IsAmazonChinaEndpoint(endpointURL url.URL) bool {
	if endpointURL == sentinelURL {
		return false
	}
	return endpointURL.Host == "s3.cn-north-1.amazonaws.com.cn"
}

func IsAmazonS3AccelerateEndpoint(endpointURL url.URL) bool {
	return strings.HasSuffix(endpointURL.Host, ".s3-accelerate.amazonaws.com")
}

// Match if it is exactly Google cloud storage endpoint.
func IsGoogleEndpoint(endpointURL url.URL) bool {
	if endpointURL == sentinelURL {
		return false
	}
	return endpointURL.Host == "storage.googleapis.com"
}

// Expects ascii encoded strings - from output of urlEncodePath
func percentEncodeSlash(s string) string {
	return strings.Replace(s, "/", "%2F", -1)
}

// QueryEncode - encodes query values in their URL encoded form. In
// addition to the percent encoding performed by urlEncodePath() used
// here, it also percent encodes '/' (forward slash)
func QueryEncode(v url.Values) string {
	if v == nil {
		return ""
	}
	var buf bytes.Buffer
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		vs := v[k]
		prefix := percentEncodeSlash(s3signer.EncodePath(k)) + "="
		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(prefix)
			buf.WriteString(percentEncodeSlash(s3signer.EncodePath(v)))
		}
	}
	return buf.String()
}
