/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2015-2023 MinIO, Inc.
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
	"strings"
	"testing"
	"time"

	"github.com/minio/minio-go/v7/pkg/encrypt"
)

func TestPostPolicySetExpires(t *testing.T) {
	tests := []struct {
		name       string
		input      time.Time
		wantErr    bool
		wantResult string
	}{
		{
			name:       "valid time",
			input:      time.Date(2023, time.March, 2, 15, 4, 5, 0, time.UTC),
			wantErr:    false,
			wantResult: "2023-03-02T15:04:05",
		},
		{
			name:    "time before 1970",
			input:   time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pp := NewPostPolicy()

			err := pp.SetExpires(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: want error: %v, got: %v", tt.name, tt.wantErr, err)
			}

			if tt.wantResult != "" {
				result := pp.String()
				if !strings.Contains(result, tt.wantResult) {
					t.Errorf("%s: want result to contain: '%s', got: '%s'", tt.name, tt.wantResult, result)
				}
			}
		})
	}
}

func TestPostPolicySetKey(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantErr    bool
		wantResult string
	}{
		{
			name:       "valid key",
			input:      "my-object",
			wantResult: `"eq","$key","my-object"`,
		},
		{
			name:    "empty key",
			input:   "",
			wantErr: true,
		},
		{
			name:    "key with spaces",
			input:   "  ",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pp := NewPostPolicy()

			err := pp.SetKey(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: want error: %v, got: %v", tt.name, tt.wantErr, err)
			}

			if tt.wantResult != "" {
				result := pp.String()
				if !strings.Contains(result, tt.wantResult) {
					t.Errorf("%s: want result to contain: '%s', got: '%s'", tt.name, tt.wantResult, result)
				}
			}
		})
	}
}

func TestPostPolicySetKeyStartsWith(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "valid key prefix",
			input: "my-prefix/",
			want:  `["starts-with","$key","my-prefix/"]`,
		},
		{
			name:  "empty prefix (allow any key)",
			input: "",
			want:  `["starts-with","$key",""]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pp := NewPostPolicy()

			err := pp.SetKeyStartsWith(tt.input)
			if err != nil {
				t.Errorf("%s: want no error, got: %v", tt.name, err)
			}

			if tt.want != "" {
				result := pp.String()
				if !strings.Contains(result, tt.want) {
					t.Errorf("%s: want result to contain: '%s', got: '%s'", tt.name, tt.want, result)
				}
			}
		})
	}
}

func TestPostPolicySetBucket(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantErr    bool
		wantResult string
	}{
		{
			name:       "valid bucket",
			input:      "my-bucket",
			wantResult: `"eq","$bucket","my-bucket"`,
		},
		{
			name:    "empty bucket",
			input:   "",
			wantErr: true,
		},
		{
			name:    "bucket with spaces",
			input:   "   ",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pp := NewPostPolicy()

			err := pp.SetBucket(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: want error: %v, got: %v", tt.name, tt.wantErr, err)
			}

			if tt.wantResult != "" {
				result := pp.String()
				if !strings.Contains(result, tt.wantResult) {
					t.Errorf("%s: want result to contain: '%s', got: '%s'", tt.name, tt.wantResult, result)
				}
			}
		})
	}
}

func TestPostPolicySetCondition(t *testing.T) {
	tests := []struct {
		name       string
		matchType  string
		condition  string
		value      string
		wantErr    bool
		wantResult string
	}{
		{
			name:       "valid eq condition",
			matchType:  "eq",
			condition:  "X-Amz-Date",
			value:      "20210324T000000Z",
			wantResult: `"eq","$X-Amz-Date","20210324T000000Z"`,
		},
		{
			name:      "empty value",
			matchType: "eq",
			condition: "X-Amz-Date",
			value:     "",
			wantErr:   true,
		},
		{
			name:      "invalid condition",
			matchType: "eq",
			condition: "Invalid-Condition",
			value:     "somevalue",
			wantErr:   true,
		},
		{
			name:       "valid starts-with condition",
			matchType:  "starts-with",
			condition:  "X-Amz-Credential",
			value:      "my-access-key",
			wantResult: `"starts-with","$X-Amz-Credential","my-access-key"`,
		},
		{
			name:      "empty condition",
			matchType: "eq",
			condition: "",
			value:     "somevalue",
			wantErr:   true,
		},
		{
			name:      "empty matchType",
			matchType: "",
			condition: "X-Amz-Date",
			value:     "somevalue",
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pp := NewPostPolicy()

			err := pp.SetCondition(tt.matchType, tt.condition, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: want error: %v, got: %v", tt.name, tt.wantErr, err)
			}

			if tt.wantResult != "" {
				result := pp.String()
				if !strings.Contains(result, tt.wantResult) {
					t.Errorf("%s: want result to contain: '%s', got: '%s'", tt.name, tt.wantResult, result)
				}
			}
		})
	}
}

func TestPostPolicySetTagging(t *testing.T) {
	tests := []struct {
		name       string
		tagging    string
		wantErr    bool
		wantResult string
	}{
		{
			name:       "valid tagging",
			tagging:    `<Tagging><TagSet><Tag><Key>key1</Key><Value>value1</Value></Tag></TagSet></Tagging>`,
			wantResult: `"eq","$tagging","<Tagging><TagSet><Tag><Key>key1</Key><Value>value1</Value></Tag></TagSet></Tagging>"`,
		},
		{
			name:    "empty tagging",
			tagging: "",
			wantErr: true,
		},
		{
			name:    "whitespace tagging",
			tagging: "   ",
			wantErr: true,
		},
		{
			name:    "invalid XML",
			tagging: `<Tagging><TagSet><Tag><Key>key1</Key><Value>value1</Value></Tag></TagSet>`,
			wantErr: true,
		},
		{
			name:    "invalid schema",
			tagging: `<InvalidTagging></InvalidTagging>`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pp := NewPostPolicy()

			err := pp.SetTagging(tt.tagging)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: want error: %v, got: %v", tt.name, tt.wantErr, err)
			}

			if tt.wantResult != "" {
				result := pp.String()
				if !strings.Contains(result, tt.wantResult) {
					t.Errorf("%s: want result to contain: '%s', got: '%s'", tt.name, tt.wantResult, result)
				}
			}
		})
	}
}

func TestPostPolicySetUserMetadata(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		value      string
		wantErr    bool
		wantResult string
	}{
		{
			name:       "valid metadata",
			key:        "user-key",
			value:      "user-value",
			wantResult: `"eq","$x-amz-meta-user-key","user-value"`,
		},
		{
			name:    "empty key",
			key:     "",
			value:   "somevalue",
			wantErr: true,
		},
		{
			name:    "empty value",
			key:     "user-key",
			value:   "",
			wantErr: true,
		},
		{
			name:    "key with spaces",
			key:     "   ",
			value:   "somevalue",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pp := NewPostPolicy()

			err := pp.SetUserMetadata(tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: want error: %v, got: %v", tt.name, tt.wantErr, err)
			}

			if tt.wantResult != "" {
				result := pp.String()
				if !strings.Contains(result, tt.wantResult) {
					t.Errorf("%s: want result to contain: '%s', got: '%s'", tt.name, tt.wantResult, result)
				}
			}
		})
	}
}

func TestPostPolicySetChecksum(t *testing.T) {
	tests := []struct {
		name       string
		checksum   Checksum
		wantErr    bool
		wantResult string
	}{
		{
			name:       "valid checksum SHA256",
			checksum:   ChecksumSHA256.ChecksumBytes([]byte("somerandomdata")),
			wantResult: `[["eq","$x-amz-checksum-algorithm","SHA256"],["eq","$x-amz-checksum-sha256","29/7Qm/iMzZ1O3zMbO0luv6mYWyS6JIqPYV9lc8w1PA="]]`,
		},
		{
			name:       "valid checksum CRC32",
			checksum:   ChecksumCRC32.ChecksumBytes([]byte("somerandomdata")),
			wantResult: `[["eq","$x-amz-checksum-algorithm","CRC32"],["eq","$x-amz-checksum-crc32","7sOPnw=="]]`,
		},
		{
			name:       "empty checksum",
			checksum:   Checksum{},
			wantResult: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pp := NewPostPolicy()

			err := pp.SetChecksum(tt.checksum)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: want error: %v, got: %v", tt.name, tt.wantErr, err)
			}

			if tt.wantResult != "" {
				result := pp.String()
				if !strings.Contains(result, tt.wantResult) {
					t.Errorf("%s: want result to contain: '%s', got: '%s'", tt.name, tt.wantResult, result)
				}
			}
		})
	}
}

func TestPostPolicySetEncryption(t *testing.T) {
	tests := []struct {
		name    string
		sseType string
		keyID   string
		want    map[string]string
	}{
		{
			name:    "SSE-S3 encryption",
			sseType: "SSE-S3",
			keyID:   "my-key-id",
			want: map[string]string{
				"X-Amz-Server-Side-Encryption":                "aws:kms",
				"X-Amz-Server-Side-Encryption-Aws-Kms-Key-Id": "my-key-id",
			},
		},
		{
			name:    "SSE-C encryption with Key ID",
			sseType: "SSE-C",
			keyID:   "my-key-id",
			want: map[string]string{
				"X-Amz-Server-Side-Encryption-Customer-Key":       "bXktc2VjcmV0LWtleTEyMzQ1Njc4OTBhYmNkZWZnaGk=",
				"X-Amz-Server-Side-Encryption-Customer-Key-Md5":   "T1mefJwyXBH43sRtfEgRZQ==",
				"X-Amz-Server-Side-Encryption-Customer-Algorithm": "AES256",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pp := NewPostPolicy()

			var sse encrypt.ServerSide
			var err error
			if tt.sseType == "SSE-S3" {
				sse, err = encrypt.NewSSEKMS(tt.keyID, nil)
				if err != nil {
					t.Fatalf("Failed to create SSE-KMS: %v", err)
				}
			} else if tt.sseType == "SSE-C" {
				sse, err = encrypt.NewSSEC([]byte("my-secret-key1234567890abcdefghi"))
				if err != nil {
					t.Fatalf("Failed to create SSE-C: %v", err)
				}
			} else {
				t.Fatalf("Unknown SSE type: %s", tt.sseType)
			}

			pp.SetEncryption(sse)

			for k, v := range tt.want {
				if pp.formData[k] != v {
					t.Errorf("%s: want %s: %s, got: %s", tt.name, k, v, pp.formData[k])
				}
			}
		})
	}
}
