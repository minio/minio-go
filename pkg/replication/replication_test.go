/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2020 MinIO, Inc.
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

package replication

import (
	"testing"
)

// Tests replication rule addition.
func TestAddReplicationRule(t *testing.T) {
	testCases := []struct {
		cfg         Config
		opts        Options
		expectedErr string
	}{
		{ // test case :1
			cfg: Config{},
			opts: Options{
				ID:           "xyz.id",
				Prefix:       "abc/",
				RuleStatus:   "enable",
				Priority:     "3",
				TagString:    "k1=v1&k2=v2",
				StorageClass: "STANDARD",
				DestBucket:   "arn:minio:replication:eu-west-1:c5acb6ac-9918-4dc6-8534-6244ed1a611a:destbucket",
			},
			expectedErr: "",
		},
		{ // test case :2
			cfg: Config{},
			opts: Options{
				ID:           "",
				Prefix:       "abc/",
				RuleStatus:   "",
				Priority:     "3",
				TagString:    "k1=v1&k2=v2",
				StorageClass: "STANDARD",
				DestBucket:   "arn:minio:replication:eu-west-1:c5acb6ac-9918-4dc6-8534-6244ed1a611a:destbucket",
			},
			expectedErr: "rule state should be either [enable|disable]",
		},
		{ // test case :3
			cfg: Config{Rules: []Rule{{Priority: 1}}},
			opts: Options{
				ID:           "xyz.id",
				Prefix:       "abc/",
				RuleStatus:   "enable",
				Priority:     "1",
				TagString:    "k1=v1&k2=v2",
				StorageClass: "STANDARD",
				DestBucket:   "arn:minio:replication:eu-west-1:c5acb6ac-9918-4dc6-8534-6244ed1a611a:destbucket",
			},
			expectedErr: "priority must be unique. Replication configuration already has a rule with this priority",
		},
		{ // test case :4
			cfg: Config{},
			opts: Options{
				ID:           "xyz.id",
				Prefix:       "abc/",
				RuleStatus:   "enable",
				Priority:     "3",
				TagString:    "k1=v1&k2=v2",
				StorageClass: "STANDARD",
				DestBucket:   "arn:minio:eu-west-1:c5acb6ac-9918-4dc6-8534-6244ed1a611a:destbucket",
			},
			expectedErr: "destination bucket needs to be in Arn format",
		},
		{ // test case :5
			cfg: Config{},
			opts: Options{
				ID:           "xyz.id",
				Prefix:       "abc/",
				RuleStatus:   "enable",
				Priority:     "3",
				TagString:    "k1=v1&k2=v2",
				StorageClass: "STANDARD",
				DestBucket:   "arn:destbucket",
			},
			expectedErr: "destination bucket needs to be in Arn format",
		},
		{ // test case :6
			cfg: Config{Role: "arn:minio:replication:eu-west-1:c5acb6ac-9918-4dc6-8534-6244ed1a611a:targetbucket"},
			opts: Options{
				ID:           "xyz.id",
				Prefix:       "abc/",
				RuleStatus:   "enable",
				Priority:     "3",
				TagString:    "k1=v1&k2=v2",
				StorageClass: "STANDARD",
				DestBucket:   "arn:minio:replication:eu-west-1:c5acb6ac-9918-4dc6-8534-6244ed1a611a:destbucket",
			},
			expectedErr: "",
		},
		{ // test case :7
			cfg: Config{},
			opts: Options{
				ID:           "xyz.id",
				Prefix:       "abc/",
				RuleStatus:   "enable",
				Priority:     "3",
				TagString:    "k1=v1&k2=v2",
				StorageClass: "STANDARD",
				DestBucket:   "arn:aws:s3:::destbucket",
			},
			expectedErr: "",
		},
		{ // test case :8
			cfg: Config{
				Rules: []Rule{
					{
						ID: "xyz.id",
						Destination: Destination{
							Bucket: "arn:aws:s3:::destbucket",
						},
					},
				},
			},
			opts: Options{
				ID:           "xyz.id",
				Prefix:       "abc/",
				RuleStatus:   "enable",
				Priority:     "1",
				TagString:    "k1=v1&k2=v2",
				StorageClass: "STANDARD",
				DestBucket:   "arn:minio:replication:eu-west-1:c5acb6ac-9918-4dc6-8534-6244ed1a611a:destbucket",
			},
			expectedErr: "a rule exists with this ID",
		},
	}
	for i, testCase := range testCases {
		cfg := testCase.cfg
		err := cfg.AddRule(testCase.opts)
		if err != nil && testCase.expectedErr != err.Error() {
			t.Errorf("Test %d: Expected %s, got %s", i+1, testCase.expectedErr, err)
		}
		if err == nil && testCase.expectedErr != "" {
			t.Errorf("Test %d: Expected %s, got %s", i+1, testCase.expectedErr, err)
		}
	}
}

// Tests replication rule edits.
func TestEditReplicationRule(t *testing.T) {
	testCases := []struct {
		cfg         Config
		opts        Options
		expectedErr string
	}{
		{ // test case :1 edit a rule in older config with remote ARN in destination bucket
			cfg: Config{
				Role: "arn:minio:replication:eu-west-1:c5acb6ac-9918-4dc6-8534-6244ed1a611a:destbucket",
				Rules: []Rule{{
					ID:          "xyz.id",
					Priority:    1,
					Filter:      Filter{Prefix: "xyz/"},
					Destination: Destination{Bucket: "arn:aws:s3:::destbucket"},
				}},
			},
			opts: Options{
				ID:           "xyz.id",
				Prefix:       "abc/",
				RuleStatus:   "enable",
				Priority:     "3",
				TagString:    "k1=v1&k2=v2",
				StorageClass: "STANDARD",
				DestBucket:   "arn:minio:replication:eu-west-1:c5acb6ac-9918-4dc6-8534-6244ed1a611a:destbucket",
			},
			expectedErr: "",
		},
		{ // test case :2 mismatched rule id
			cfg: Config{
				Role: "arn:minio:replication:eu-west-1:c5acb6ac-9918-4dc6-8534-6244ed1a611a:destbucket",
				Rules: []Rule{{
					ID:          "xyz.id2",
					Priority:    1,
					Filter:      Filter{Prefix: "xyz/"},
					Destination: Destination{Bucket: "arn:aws:s3:::destbucket"},
				}},
			},
			opts: Options{
				ID:           "xyz.id",
				Prefix:       "abc/",
				RuleStatus:   "enable",
				Priority:     "3",
				TagString:    "k1=v1&k2=v2",
				StorageClass: "STANDARD",
				DestBucket:   "arn:minio:replication:eu-west-1:c5acb6ac-9918-4dc6-8534-6244ed1a611a:destbucket",
			},
			expectedErr: "rule with ID xyz.id not found in replication configuration",
		},
		{ // test case :3 missing rule id
			cfg: Config{
				Role: "arn:minio:replication:eu-west-1:c5acb6ac-9918-4dc6-8534-6244ed1a611a:destbucket",
				Rules: []Rule{{
					ID:          "xyz.id2",
					Priority:    1,
					Filter:      Filter{Prefix: "xyz/"},
					Destination: Destination{Bucket: "arn:minio:replication:eu-west-1:c5acb6ac-9918-4dc6-8534-6244ed1a611a:destbucket"},
				}},
			},
			opts: Options{
				Prefix:       "abc/",
				RuleStatus:   "enable",
				Priority:     "3",
				TagString:    "k1=v1&k2=v2",
				StorageClass: "STANDARD",
				DestBucket:   "arn:minio:replication:eu-west-1:c5acb6ac-9918-4dc6-8534-6244ed1a611a:destbucket",
			},
			expectedErr: "rule ID missing",
		},
		{ // test case :4 different destination bucket
			cfg: Config{
				Role: "",
				Rules: []Rule{{
					ID:          "xyz.id",
					Priority:    1,
					Filter:      Filter{Prefix: "xyz/"},
					Destination: Destination{Bucket: "arn:minio:replication:eu-west-1:c5acb6ac-9918-4dc6-8534-6244ed1a611a:destbucket"},
				}},
			},
			opts: Options{
				ID:           "xyz.id",
				Prefix:       "abc/",
				RuleStatus:   "enable",
				Priority:     "3",
				TagString:    "k1=v1&k2=v2",
				StorageClass: "STANDARD",
				DestBucket:   "arn:aws:s3:::differentbucket",
			},
			expectedErr: "invalid destination bucket for this rule",
		},
		{ // test case :5 invalid destination bucket arn format
			cfg: Config{
				Role: "arn:minio:replication:eu-west-1:c5acb6ac-9918-4dc6-8534-6244ed1a611a:destbucket",
				Rules: []Rule{{
					ID:          "xyz.id",
					Priority:    1,
					Filter:      Filter{Prefix: "xyz/"},
					Destination: Destination{Bucket: "arn:aws:s3:::destbucket"},
				}},
			},
			opts: Options{
				ID:           "xyz.id",
				Prefix:       "abc/",
				RuleStatus:   "enable",
				Priority:     "3",
				TagString:    "k1=v1&k2=v2",
				StorageClass: "STANDARD",
				DestBucket:   "arn:destbucket",
			},
			expectedErr: "destination bucket needs to be in Arn format",
		},

		{ // test case :6 invalid rule status
			cfg: Config{
				Rules: []Rule{{
					ID:          "xyz.id",
					Priority:    1,
					Filter:      Filter{Prefix: "xyz/"},
					Destination: Destination{Bucket: "arn:aws:s3:::destbucket"},
				}},
			},
			opts: Options{
				ID:           "xyz.id",
				Prefix:       "abc/",
				RuleStatus:   "xx",
				Priority:     "3",
				TagString:    "k1=v1&k2=v2",
				StorageClass: "STANDARD",
				DestBucket:   "arn:aws:s3:::destbucket",
			},
			expectedErr: "rule state should be either [enable|disable]",
		},
		{ // test case :7 another rule has same priority
			cfg: Config{
				Rules: []Rule{
					{
						ID:          "xyz.id",
						Priority:    0,
						Filter:      Filter{Prefix: "xyz/"},
						Destination: Destination{Bucket: "arn:aws:s3:::destbucket"},
					},
					{
						ID:          "xyz.id2",
						Priority:    1,
						Filter:      Filter{Prefix: "xyz/"},
						Destination: Destination{Bucket: "arn:aws:s3:::destbucket"},
					},
				},
			},
			opts: Options{
				ID:           "xyz.id",
				Prefix:       "abc/",
				RuleStatus:   "disable",
				Priority:     "1",
				TagString:    "k1=v1&k2=v2",
				StorageClass: "STANDARD",
				DestBucket:   "arn:aws:s3:::destbucket",
			},
			expectedErr: "priority must be unique. Replication configuration already has a rule with this priority",
		},
		{ // test case :8 ; edit a rule in older config
			cfg: Config{
				Role: "arn:minio:replication:eu-west-1:c5acb6ac-9918-4dc6-8534-6244ed1a611a:destbucket",
				Rules: []Rule{{
					ID:          "xyz.id",
					Priority:    1,
					Filter:      Filter{Prefix: "xyz/"},
					Destination: Destination{Bucket: "arn:aws:s3:::destbucket"},
				}},
			},
			opts: Options{
				ID:           "xyz.id",
				Prefix:       "abc/",
				RuleStatus:   "enable",
				Priority:     "3",
				TagString:    "k1=v1&k2=v2",
				StorageClass: "STANDARD",
				DestBucket:   "arn:aws:s3:::destbucket",
			},
			expectedErr: "",
		},
	}

	for i, testCase := range testCases {
		cfg := testCase.cfg
		err := cfg.EditRule(testCase.opts)
		if err != nil && testCase.expectedErr != err.Error() {
			t.Errorf("Test %d: Expected %s, got %s", i+1, testCase.expectedErr, err)
		}
		if err == nil && testCase.expectedErr != "" {
			t.Errorf("Test %d: Expected %s, got %s", i+1, testCase.expectedErr, err)
		}
	}
}
