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
	"reflect"
	"strings"
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

// TestLifecycleMarshalXML for cases where XML output needs to be well-formed
func TestLifecycleMarshalXML(t *testing.T) {
	testCases := []struct {
		testDescription string
		input           Configuration
		expectedXMLOut  string
	}{
		{
			testDescription: "Ensure Filter is missing if Prefix is present - in order to support server implementations that ignore Prefix if Filter is present",
			input: Configuration{
				Rules: []Rule{
					{
						ID:                             "expire-incomplete-uploads-1",
						Status:                         "Enabled",
						Prefix:                         "my_dir",
						AbortIncompleteMultipartUpload: AbortIncompleteMultipartUpload{DaysAfterInitiation: 1},
					},
				},
			},
			expectedXMLOut: "<LifecycleConfiguration><Rule><AbortIncompleteMultipartUpload><DaysAfterInitiation>1</DaysAfterInitiation></AbortIncompleteMultipartUpload><ID>expire-incomplete-uploads-1</ID><Prefix>my_dir</Prefix><Status>Enabled</Status></Rule></LifecycleConfiguration>",
		},
		{
			testDescription: "Ensure an empty Filter is emitted when both RuleFilter and Prefix are unset. Specification explicitly mentions: 'Filter is required if the LifecycleRule does not contain a Prefix element.' (https://docs.aws.amazon.com/AmazonS3/latest/API/API_LifecycleRule.html)",
			input: Configuration{
				Rules: []Rule{
					{
						ID:                             "expire-incomplete-uploads-2",
						Status:                         "Enabled",
						AbortIncompleteMultipartUpload: AbortIncompleteMultipartUpload{DaysAfterInitiation: 1},
					},
				},
			},
			expectedXMLOut: "<LifecycleConfiguration><Rule><AbortIncompleteMultipartUpload><DaysAfterInitiation>1</DaysAfterInitiation></AbortIncompleteMultipartUpload><ID>expire-incomplete-uploads-2</ID><Status>Enabled</Status><Filter><Prefix></Prefix></Filter></Rule></LifecycleConfiguration>",
		},
		{
			testDescription: "Ensure a non-empty Filter marshals through the default path unchanged",
			input: Configuration{
				Rules: []Rule{
					{
						ID:     "expire-incomplete-uploads-3",
						Status: "Enabled",
						RuleFilter: Filter{
							Prefix: "logs/",
						},
						AbortIncompleteMultipartUpload: AbortIncompleteMultipartUpload{DaysAfterInitiation: 1},
					},
				},
			},
			expectedXMLOut: "<LifecycleConfiguration><Rule><AbortIncompleteMultipartUpload><DaysAfterInitiation>1</DaysAfterInitiation></AbortIncompleteMultipartUpload><ID>expire-incomplete-uploads-3</ID><Filter><Prefix>logs/</Prefix></Filter><Status>Enabled</Status></Rule></LifecycleConfiguration>",
		},
		{
			testDescription: "Ensure every Rule field survives the empty-Filter marshal path",
			input: Configuration{
				Rules: []Rule{
					{
						ID:                             "expire-full",
						Status:                         "Enabled",
						AbortIncompleteMultipartUpload: AbortIncompleteMultipartUpload{DaysAfterInitiation: 1},
						Expiration:                     Expiration{Days: 30},
						DelMarkerExpiration:            DelMarkerExpiration{Days: 7},
						AllVersionsExpiration:          AllVersionsExpiration{Days: 10},
						NoncurrentVersionExpiration:    NoncurrentVersionExpiration{NoncurrentDays: 5},
						NoncurrentVersionTransition:    NoncurrentVersionTransition{NoncurrentDays: 3, StorageClass: "GLACIER"},
						Transition:                     Transition{Days: 60, StorageClass: "GLACIER"},
					},
				},
			},
			expectedXMLOut: "<LifecycleConfiguration><Rule><AbortIncompleteMultipartUpload><DaysAfterInitiation>1</DaysAfterInitiation></AbortIncompleteMultipartUpload><Expiration><Days>30</Days></Expiration><DelMarkerExpiration><Days>7</Days></DelMarkerExpiration><AllVersionsExpiration><Days>10</Days></AllVersionsExpiration><ID>expire-full</ID><NoncurrentVersionExpiration><NoncurrentDays>5</NoncurrentDays></NoncurrentVersionExpiration><NoncurrentVersionTransition><StorageClass>GLACIER</StorageClass><NoncurrentDays>3</NoncurrentDays></NoncurrentVersionTransition><Status>Enabled</Status><Transition><StorageClass>GLACIER</StorageClass><Days>60</Days></Transition><Filter><Prefix></Prefix></Filter></Rule></LifecycleConfiguration>",
		},
		{
			testDescription: "Ensure a size-only Filter marshals its size condition",
			input: Configuration{
				Rules: []Rule{
					{
						ID:         "expire-large",
						Status:     "Enabled",
						RuleFilter: Filter{ObjectSizeGreaterThan: 1048576},
						Expiration: Expiration{Days: 30},
					},
				},
			},
			expectedXMLOut: "<LifecycleConfiguration><Rule><Expiration><Days>30</Days></Expiration><ID>expire-large</ID><Filter><ObjectSizeGreaterThan>1048576</ObjectSizeGreaterThan></Filter><Status>Enabled</Status></Rule></LifecycleConfiguration>",
		},
		{
			testDescription: "Ensure an ObjectSizeLessThan-only Filter marshals its size condition",
			input: Configuration{
				Rules: []Rule{
					{
						ID:         "expire-small",
						Status:     "Enabled",
						RuleFilter: Filter{ObjectSizeLessThan: 1024},
						Expiration: Expiration{Days: 30},
					},
				},
			},
			expectedXMLOut: "<LifecycleConfiguration><Rule><Expiration><Days>30</Days></Expiration><ID>expire-small</ID><Filter><ObjectSizeLessThan>1024</ObjectSizeLessThan></Filter><Status>Enabled</Status></Rule></LifecycleConfiguration>",
		},
		{
			testDescription: "Ensure a tag-only Filter marshals its tag condition",
			input: Configuration{
				Rules: []Rule{
					{
						ID:         "expire-tagged",
						Status:     "Enabled",
						RuleFilter: Filter{Tag: Tag{Key: "env", Value: "prod"}},
						Expiration: Expiration{Days: 30},
					},
				},
			},
			expectedXMLOut: "<LifecycleConfiguration><Rule><Expiration><Days>30</Days></Expiration><ID>expire-tagged</ID><Filter><Tag><Key>env</Key><Value>prod</Value></Tag></Filter><Status>Enabled</Status></Rule></LifecycleConfiguration>",
		},
		{
			testDescription: "Ensure a rule carrying both a top-level Prefix and a non-empty Filter emits both elements through the default path",
			input: Configuration{
				Rules: []Rule{
					{
						ID:         "both-set",
						Status:     "Enabled",
						Prefix:     "abc/",
						RuleFilter: Filter{Prefix: "data/"},
						Expiration: Expiration{Days: 30},
					},
				},
			},
			expectedXMLOut: "<LifecycleConfiguration><Rule><Expiration><Days>30</Days></Expiration><ID>both-set</ID><Filter><Prefix>data/</Prefix></Filter><Prefix>abc/</Prefix><Status>Enabled</Status></Rule></LifecycleConfiguration>",
		},
		{
			testDescription: "Ensure an And Filter marshals its combined conditions",
			input: Configuration{
				Rules: []Rule{
					{
						ID:         "expire-and",
						Status:     "Enabled",
						RuleFilter: Filter{And: And{Prefix: "docs/", Tags: []Tag{{Key: "env", Value: "prod"}}}},
						Expiration: Expiration{Days: 30},
					},
				},
			},
			expectedXMLOut: "<LifecycleConfiguration><Rule><Expiration><Days>30</Days></Expiration><ID>expire-and</ID><Filter><And><Prefix>docs/</Prefix><Tag><Key>env</Key><Value>prod</Value></Tag></And></Filter><Status>Enabled</Status></Rule></LifecycleConfiguration>",
		},
	}

	for i, tc := range testCases {
		xmlBytes, err := xml.Marshal(tc.input)
		if err != nil {
			t.Fatalf("%d (%s): could not marshal the Configuration: %#v, %v", i+1, tc.testDescription, tc.input, err)
		}
		if string(xmlBytes) != tc.expectedXMLOut {
			t.Fatalf("%d (%s): failed\nexpected: %s\ngot:      %s", i+1, tc.testDescription, tc.expectedXMLOut, string(xmlBytes))
		}
	}
}

// TestFilterMarshalXMLEmpty pins the standalone contract that a zero-value
// Filter marshals as an explicit empty element rather than nothing.
func TestFilterMarshalXMLEmpty(t *testing.T) {
	got, err := xml.Marshal(Filter{})
	if err != nil {
		t.Fatalf("could not marshal a zero-value Filter: %v", err)
	}
	const want = "<Filter><Prefix></Prefix></Filter>"
	if string(got) != want {
		t.Fatalf("zero-value Filter marshal\nexpected: %s\ngot:      %s", want, string(got))
	}
}

// fillNonZero recursively sets v to a non-zero value so the field it belongs
// to cannot be suppressed by an omitempty tag or an isNull-style check.
func fillNonZero(t *testing.T, v reflect.Value) {
	t.Helper()
	// Unexported fields cannot be set and are invisible to xml.Marshal, so
	// they need no exercising.
	if !v.CanSet() {
		return
	}
	switch v.Kind() {
	case reflect.String:
		v.SetString("x")
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(1)
	case reflect.Struct:
		switch v.Type() {
		case reflect.TypeOf(time.Time{}):
			v.Set(reflect.ValueOf(time.Date(2026, time.July, 21, 0, 0, 0, 0, time.UTC)))
			return
		case reflect.TypeOf(xml.Name{}):
			// XMLName is element metadata, not data; filling it would
			// rename elements instead of exercising field emission.
			return
		}
		for i := range v.NumField() {
			fillNonZero(t, v.Field(i))
		}
	default:
		t.Fatalf("fillNonZero: extend for unsupported kind %v (%v)", v.Kind(), v.Type())
	}
}

// TestRuleWrapperCopyCompleteness fills every Rule field except XMLName and
// RuleFilter (kept null so the wrapper path runs), then requires the
// wrapper-path output to match the default struct-tag encoding. This pins
// encoding/xml's depth-based shadowing of the embedded RuleFilter: if the
// shallower pointer ever stopped shadowing it, the Filter would be dropped,
// doubled, or misplaced here.
func TestRuleWrapperCopyCompleteness(t *testing.T) {
	var r Rule
	rv := reflect.ValueOf(&r).Elem()
	for i := range rv.NumField() {
		switch rv.Type().Field(i).Name {
		case "XMLName", "RuleFilter":
			continue
		}
		fillNonZero(t, rv.Field(i))
	}

	type aliasConfiguration struct {
		XMLName xml.Name    `xml:"LifecycleConfiguration"`
		Rules   []ruleAlias `xml:"Rule"`
	}
	const nullFilter = "<Filter><Prefix></Prefix></Filter>"

	marshalBoth := func(r Rule) (got, want string) {
		gotBytes, err := xml.Marshal(Configuration{Rules: []Rule{r}})
		if err != nil {
			t.Fatalf("could not marshal Rule through the wrapper path: %v", err)
		}
		wantBytes, err := xml.Marshal(aliasConfiguration{Rules: []ruleAlias{ruleAlias(r)}})
		if err != nil {
			t.Fatalf("could not marshal Rule through the default encoding: %v", err)
		}
		if !strings.Contains(string(wantBytes), nullFilter) {
			t.Fatalf("default encoding no longer emits a null Filter as %s; update this test\ngot: %s", nullFilter, string(wantBytes))
		}
		return string(gotBytes), string(wantBytes)
	}

	t.Run("prefix set omits Filter", func(t *testing.T) {
		got, want := marshalBoth(r)
		want = strings.Replace(want, nullFilter, "", 1)
		if got != want {
			t.Fatalf("wrapper path dropped or altered a field\nexpected: %s\ngot:      %s", want, got)
		}
	})

	t.Run("prefix empty emits Filter last", func(t *testing.T) {
		r := r
		r.Prefix = ""
		got, want := marshalBoth(r)
		// The default encoding emits the null Filter in its declared field
		// position and Prefix as an empty element is omitted; the wrapper
		// moves the Filter to the last child of Rule.
		want = strings.Replace(want, nullFilter, "", 1)
		want = strings.Replace(want, "</Rule>", nullFilter+"</Rule>", 1)
		if got != want {
			t.Fatalf("wrapper path dropped, doubled, or misplaced a field\nexpected: %s\ngot:      %s", want, got)
		}
	})
}
