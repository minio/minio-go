/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2021 MinIO, Inc.
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

package lifecycle

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMarshalJSON(t *testing.T) {
	testNow := time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)
	inp := Configuration{
		Rules: []Rule{
			{
				RuleFilter: Filter{
					Prefix: "prefix",
				},
				Expiration: Expiration{
					Days: ExpirationDays(3),
				},
				AbortIncompleteMultipartUpload: AbortIncompleteMultipartUpload{
					DaysAfterInitiation: ExpirationDays(1),
				},
				ID:     "rule-1",
				Status: "Enabled",
			},
			{
				RuleFilter: Filter{
					And: And{
						Prefix: "prefix",
						Tags: []Tag{
							{Key: "key-1",
								Value: "val-1",
							},
						},
					},
				},
				Expiration: Expiration{
					Date: ExpirationDate{
						testNow,
					},
				},
				NoncurrentVersionExpiration: NoncurrentVersionExpiration{
					NoncurrentDays: ExpirationDays(1),
				},
				ID:     "rule-2",
				Status: "Enabled",
			},
			{
				Transition: Transition{
					Days: ExpirationDays(3),
				},
				Expiration: Expiration{
					DeleteMarker: ExpireDeleteMarker(true),
				},
				NoncurrentVersionTransition: NoncurrentVersionTransition{
					NoncurrentDays: ExpirationDays(3),
				},
				ID:     "rule-3",
				Status: "Enabled",
			},
		},
	}

	expected := `{"Rules":[{"AbortIncompleteMultipartUpload":{"DaysAfterInitiation":1},"Expiration":{"Days":3,"DeleteMarker":false},"ID":"rule-1","Filter":{"Prefix":"prefix"},"Status":"Enabled"},{"Expiration":{"Date":"2021-01-01T00:00:00Z","DeleteMarker":false},"ID":"rule-2","Filter":{"And":{"Prefix":"prefix","Tags":[{"Key":"key-1","Value":"val-1"}]}},"NoncurrentVersionExpiration":{"NoncurrentDays":1},"Status":"Enabled"},{"Expiration":{"DeleteMarker":true},"ID":"rule-3","NoncurrentVersionTransition":{"NoncurrentDays":3},"Status":"Enabled","Transition":{"Days":3}}]}`

	got, err := json.Marshal(inp)
	if err != nil {
		t.Fatal("failed to marshal json", err)
	}

	if expected != string(got) {
		t.Fatalf("Expected %s but got %s", expected, got)
	}
}
