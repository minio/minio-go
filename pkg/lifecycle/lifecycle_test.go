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
	"reflect"
	"testing"
	"time"
)

func TestExpiredObjectDeleteMarker(t *testing.T) {
	input := `{
  "Rules": [
    {
      "ID": "expire-obj-delmarker",
      "Status": "Enabled",
      "Expiration": {
        "ExpiredObjectDeleteMarker": true
      }
    }
   ]
   }`
	var cfg Configuration
	err := json.Unmarshal([]byte(input), &cfg)
	if err != nil {
		t.Fatalf("Failed to unmarshal lifecycle config %v", err)
	}

	if !cfg.Rules[0].Expiration.DeleteMarker {
		t.Fatal("Expected Expiration.DeleteMarker to be true but got false")
	}

	// Round-trip test
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal lifecycle config %v", err)
	}
	var newcfg Configuration
	err = json.Unmarshal(b, &newcfg)
	if err != nil {
		t.Fatalf("Failed to unmarshal lifecycle config json %v", err)
	}
	if !reflect.DeepEqual(newcfg, cfg) {
		t.Fatalf("Expected configs to be equal but they aren't: newcfg %v cfg %v", newcfg, cfg)
	}
}

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

	expected := `{"Rules":[{"AbortIncompleteMultipartUpload":{"DaysAfterInitiation":1},"Expiration":{"Days":3},"ID":"rule-1","Filter":{"Prefix":"prefix"},"Status":"Enabled"},{"Expiration":{"Date":"2021-01-01T00:00:00Z"},"ID":"rule-2","Filter":{"And":{"Prefix":"prefix","Tags":[{"Key":"key-1","Value":"val-1"}]}},"NoncurrentVersionExpiration":{"NoncurrentDays":1},"Status":"Enabled"},{"Expiration":{"ExpiredObjectDeleteMarker":true},"ID":"rule-3","NoncurrentVersionTransition":{"NoncurrentDays":3},"Status":"Enabled","Transition":{"Days":3}}]}`

	got, err := json.Marshal(inp)
	if err != nil {
		t.Fatal("failed to marshal json", err)
	}

	if expected != string(got) {
		t.Fatalf("Expected %s but got %s", expected, got)
	}
}
