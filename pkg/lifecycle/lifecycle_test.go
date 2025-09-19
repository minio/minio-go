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
	"bytes"
	"encoding/json"
	"encoding/xml"
	"testing"
	"time"
)

func TestLifecycleUnmarshalJSON(t *testing.T) {
	testCases := []struct {
		input string
		err   error
	}{
		{
			input: `{
				"Rules": [
					{
						"ID": "transition-missing",
						"Status": "Enabled",
						"Transition": {
							"Days": 0
						}
					}
				]
			}`,
			err: errMissingStorageClass,
		},
		{
			input: `{
				"Rules": [
					{
						"ID": "transition-missing-1",
						"Status": "Enabled",
						"Transition": {
							"Days": 1
						}
					}
				]
			}`,
			err: errMissingStorageClass,
		},
		{
			input: `{
				"Rules": [
					{
						"ID": "noncurrent-transition-missing",
						"Status": "Enabled",
						"NoncurrentVersionTransition": {
							"NoncurrentDays": 0
						}
					}
				]
			}`,
			err: errMissingStorageClass,
		},
		{
			input: `{
				"Rules": [
					{
						"ID": "noncurrent-transition-missing-1",
						"Status": "Enabled",
						"NoncurrentVersionTransition": {
							"NoncurrentDays": 1
						}
					}
				]
			}`,
			err: errMissingStorageClass,
		},
		{
			input: `{
				"Rules": [
					{
						"ID": "transition",
						"Status": "Enabled",
						"Transition": {
							"StorageClass": "S3TIER-1",
							"Days": 1
						}
					}
				]
			}`,
			err: nil,
		},
		{
			input: `{
				"Rules": [
					{
						"ID": "noncurrent-transition",
						"Status": "Enabled",
						"NoncurrentVersionTransition": {
							"StorageClass": "S3TIER-1",
							"NoncurrentDays": 1
						}
					}
				]
			}`,
			err: nil,
		},
		{
			input: `{
				"Rules": [
					{
						"ID": "transition-lt",
						"Status": "Enabled",
						"Filter": {
							"ObjectSizeLessThan": 1048576
						},
						"Transition": {
							"StorageClass": "S3TIER-1",
							"Days": 1
						}
					}
				]
			}`,
			err: nil,
		},
		{
			input: `{
				"Rules": [
					{
						"ID": "noncurrent-transition-gt",
						"Status": "Enabled",
						"Filter": {
							"ObjectSizeGreaterThan": 10485760
						},
						"NoncurrentVersionTransition": {
							"StorageClass": "S3TIER-1",
							"NoncurrentDays": 1
						}
					}
				]
			}`,
			err: nil,
		},
		{
			input: `{
				"Rules": [
					{
						"ID": "noncurrent-transition-lt-and-gt",
						"Status": "Enabled",
						"Filter": {
							"And": {
								"ObjectSizeGreaterThan": 10485760,
								"ObjectSizeLessThan": 1048576
							}
						},
						"NoncurrentVersionTransition": {
							"StorageClass": "S3TIER-1",
							"NoncurrentDays": 1
						}
					}
				]
			}`,
			err: nil,
		},
	}

	for i, tc := range testCases {
		var lc Configuration
		if err := json.Unmarshal([]byte(tc.input), &lc); err != tc.err {
			t.Fatalf("%d: expected error %v but got %v", i+1, tc.err, err)
		}
	}
}

func TestLifecycleJSONRoundtrip(t *testing.T) {
	testNow := time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)
	lc := Configuration{
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
							{
								Key:   "key-1",
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
					Days:         ExpirationDays(3),
					StorageClass: "MINIOTIER-1",
				},
				Expiration: Expiration{
					DeleteMarker: ExpireDeleteMarker(true),
				},
				NoncurrentVersionTransition: NoncurrentVersionTransition{
					NoncurrentDays: ExpirationDays(3),
					StorageClass:   "MINIOTIER-2",
				},
				ID:     "rule-3",
				Status: "Enabled",
			},
			{
				Transition: Transition{
					Date:         ExpirationDate{testNow},
					StorageClass: "MINIOTIER-1",
				},
				ID:     "rule-4",
				Status: "Enabled",
			},
			{
				NoncurrentVersionExpiration: NoncurrentVersionExpiration{
					NoncurrentDays:          ExpirationDays(3),
					NewerNoncurrentVersions: 1,
				},
				NoncurrentVersionTransition: NoncurrentVersionTransition{
					NoncurrentDays:          ExpirationDays(3),
					NewerNoncurrentVersions: 1,
					StorageClass:            "MINIOTIER-2",
				},
				ID:     "rule-5",
				Status: "Enabled",
			},
			{
				Expiration: Expiration{
					DeleteMarker: true,
				},
				ID:     "rule-6",
				Status: "Enabled",
			},
			{
				DelMarkerExpiration: DelMarkerExpiration{
					Days: 10,
				},
				ID:     "rule-7",
				Status: "Enabled",
			},
			{
				AllVersionsExpiration: AllVersionsExpiration{
					Days: 10,
				},
				ID:     "rule-8",
				Status: "Enabled",
			},
			{
				AllVersionsExpiration: AllVersionsExpiration{
					Days: 0,
				},
				ID:     "rule-9",
				Status: "Enabled",
			},
			{
				AllVersionsExpiration: AllVersionsExpiration{
					Days:         7,
					DeleteMarker: ExpireDeleteMarker(true),
				},
				ID:     "rule-10",
				Status: "Enabled",
			},
		},
	}

	buf, err := json.Marshal(lc)
	if err != nil {
		t.Fatal("failed to marshal json", err)
	}

	var got Configuration
	if err = json.Unmarshal(buf, &got); err != nil {
		t.Fatal("failed to unmarshal json", err)
	}

	for i := range lc.Rules {
		if !lc.Rules[i].NoncurrentVersionTransition.equals(got.Rules[i].NoncurrentVersionTransition) {
			t.Fatalf("expected %#v got %#v", lc.Rules[i].NoncurrentVersionTransition, got.Rules[i].NoncurrentVersionTransition)
		}

		if !lc.Rules[i].NoncurrentVersionExpiration.equals(got.Rules[i].NoncurrentVersionExpiration) {
			t.Fatalf("expected %#v got %#v", lc.Rules[i].NoncurrentVersionExpiration, got.Rules[i].NoncurrentVersionExpiration)
		}

		if !lc.Rules[i].Transition.equals(got.Rules[i].Transition) {
			t.Fatalf("expected %#v got %#v", lc.Rules[i].Transition, got.Rules[i].Transition)
		}
		if lc.Rules[i].Expiration != got.Rules[i].Expiration {
			t.Fatalf("expected %#v got %#v", lc.Rules[i].Expiration, got.Rules[i].Expiration)
		}
		if !lc.Rules[i].DelMarkerExpiration.equals(got.Rules[i].DelMarkerExpiration) {
			t.Fatalf("expected %#v got %#v", lc.Rules[i].DelMarkerExpiration, got.Rules[i].DelMarkerExpiration)
		}
		if !lc.Rules[i].AllVersionsExpiration.equals(got.Rules[i].AllVersionsExpiration) {
			t.Fatalf("expected %#v got %#v", lc.Rules[i].AllVersionsExpiration, got.Rules[i].AllVersionsExpiration)
		}
	}
}

func TestLifecycleXMLRoundtrip(t *testing.T) {
	lc := Configuration{
		Rules: []Rule{
			{
				ID:     "immediate-noncurrent",
				Status: "Enabled",
				NoncurrentVersionTransition: NoncurrentVersionTransition{
					NoncurrentDays: 0,
					StorageClass:   "S3TIER-1",
				},
			},
			{
				ID:     "immediate-current",
				Status: "Enabled",
				Transition: Transition{
					StorageClass: "S3TIER-1",
					Days:         0,
				},
			},
			{
				ID:     "current",
				Status: "Enabled",
				Transition: Transition{
					StorageClass: "S3TIER-1",
					Date:         ExpirationDate{time.Date(2021, time.September, 1, 0, 0, 0, 0, time.UTC)},
				},
			},
			{
				ID:     "noncurrent",
				Status: "Enabled",
				NoncurrentVersionTransition: NoncurrentVersionTransition{
					NoncurrentDays: ExpirationDays(5),
					StorageClass:   "S3TIER-1",
				},
			},
			{
				ID:     "max-noncurrent-versions",
				Status: "Enabled",
				NoncurrentVersionExpiration: NoncurrentVersionExpiration{
					NewerNoncurrentVersions: 5,
				},
			},
			{
				ID:     "delmarker-expiration",
				Status: "Enabled",
				DelMarkerExpiration: DelMarkerExpiration{
					Days: 5,
				},
			},
			{
				ID:     "all-versions-expiration-1",
				Status: "Enabled",
				AllVersionsExpiration: AllVersionsExpiration{
					Days: 5,
				},
			},
			{
				ID:     "all-versions-expiration-2",
				Status: "Enabled",
				AllVersionsExpiration: AllVersionsExpiration{
					Days:         10,
					DeleteMarker: ExpireDeleteMarker(true),
				},
				RuleFilter: Filter{
					Tag: Tag{
						Key:   "key-1",
						Value: "value-1",
					},
				},
			},
		},
	}

	buf, err := xml.Marshal(lc)
	if err != nil {
		t.Fatalf("failed to marshal lifecycle configuration %v", err)
	}

	var got Configuration
	err = xml.Unmarshal(buf, &got)
	if err != nil {
		t.Fatalf("failed to unmarshal lifecycle %v", err)
	}

	for i := range lc.Rules {
		if !lc.Rules[i].NoncurrentVersionTransition.equals(got.Rules[i].NoncurrentVersionTransition) {
			t.Fatalf("%d: expected %#v got %#v", i+1, lc.Rules[i].NoncurrentVersionTransition, got.Rules[i].NoncurrentVersionTransition)
		}

		if !lc.Rules[i].Transition.equals(got.Rules[i].Transition) {
			t.Fatalf("%d: expected %#v got %#v", i+1, lc.Rules[i].Transition, got.Rules[i].Transition)
		}

		if !lc.Rules[i].NoncurrentVersionExpiration.equals(got.Rules[i].NoncurrentVersionExpiration) {
			t.Fatalf("%d: expected %#v got %#v", i+1, lc.Rules[i].NoncurrentVersionExpiration, got.Rules[i].NoncurrentVersionExpiration)
		}

		if !lc.Rules[i].DelMarkerExpiration.equals(got.Rules[i].DelMarkerExpiration) {
			t.Fatalf("%d: expected %#v got %#v", i+1, lc.Rules[i].DelMarkerExpiration, got.Rules[i].DelMarkerExpiration)
		}

		if !lc.Rules[i].AllVersionsExpiration.equals(got.Rules[i].AllVersionsExpiration) {
			t.Fatalf("%d: expected %#v got %#v", i+1, lc.Rules[i].AllVersionsExpiration, got.Rules[i].AllVersionsExpiration)
		}
	}
}

func (n NoncurrentVersionTransition) equals(m NoncurrentVersionTransition) bool {
	return n.NoncurrentDays == m.NoncurrentDays && n.StorageClass == m.StorageClass
}

func (n NoncurrentVersionExpiration) equals(m NoncurrentVersionExpiration) bool {
	return n.NoncurrentDays == m.NoncurrentDays && n.NewerNoncurrentVersions == m.NewerNoncurrentVersions
}

func (t Transition) equals(u Transition) bool {
	return t.Days == u.Days && t.Date.Equal(u.Date.Time) && t.StorageClass == u.StorageClass
}

func (a DelMarkerExpiration) equals(b DelMarkerExpiration) bool {
	return a.Days == b.Days
}

func (a AllVersionsExpiration) equals(b AllVersionsExpiration) bool {
	return a.Days == b.Days && a.DeleteMarker == b.DeleteMarker
}

func TestExpiredObjectDeleteMarker(t *testing.T) {
	expected := []byte(`{"Rules":[{"Expiration":{"ExpiredObjectDeleteMarker":true},"ID":"expired-object-delete-marker","Status":"Enabled"}]}`)
	lc := Configuration{
		Rules: []Rule{
			{
				Expiration: Expiration{
					DeleteMarker: true,
				},
				ID:     "expired-object-delete-marker",
				Status: "Enabled",
			},
		},
	}

	got, err := json.Marshal(lc)
	if err != nil {
		t.Fatalf("Failed to marshal due to %v", err)
	}
	if !bytes.Equal(expected, got) {
		t.Fatalf("Expected %s but got %s", expected, got)
	}
}

func TestAllVersionsExpiration(t *testing.T) {
	expected := []byte(`{"Rules":[{"AllVersionsExpiration":{"Days":2,"DeleteMarker":true},"ID":"all-versions-expiration","Status":"Enabled"}]}`)
	lc := Configuration{
		Rules: []Rule{
			{
				AllVersionsExpiration: AllVersionsExpiration{
					Days:         2,
					DeleteMarker: ExpireDeleteMarker(true),
				},
				ID:     "all-versions-expiration",
				Status: "Enabled",
			},
		},
	}

	got, err := json.Marshal(lc)
	if err != nil {
		t.Fatalf("Failed to marshal due to %v", err)
	}
	if !bytes.Equal(expected, got) {
		t.Fatalf("Expected %s but got %s", expected, got)
	}
}
