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

// Package lifecycle contains all the lifecycle related data types and marshallers.
package throttle

import (
	"encoding/json"
	"encoding/xml"
	//"errors"
	//"time"
)

// MarshalJSON customizes json encoding by omitting empty values
func (r Rule) MarshalJSON() ([]byte, error) {
	type rule struct {
		ConcurrentRequestsCount uint64 `json:"ConcurrentRequestsCount,omitempty"`
		APIs                    string `json:"APIs",omitempty`
		ID                      string `json:"ID"`
	}
	newr := rule{
		ConcurrentRequestsCount: r.ConcurrentRequestsCount,
		APIs:                    r.APIs,
		ID:                      r.ID,
	}

	return json.Marshal(newr)
}

// Rule represents a single rule in throttle configuration
type Rule struct {
	XMLName                 xml.Name `xml:"Rule,omitempty" json:"-"`
	ConcurrentRequestsCount uint64  `xml:"ConcurrentRequestsCount" json:"ConcurrentRequestsCount"`
	APIs                    string   `xml:"APIs" json:"APIs"`
	ID                      string   `xml:"ID" json:"ID"`
}

// Configuration is a collection of Rule objects.
type Configuration struct {
	XMLName xml.Name `xml:"ThrottleConfiguration,omitempty" json:"-"`
	Rules   []Rule   `xml:"Rule"`
}

// Empty check if lifecycle configuration is empty
func (c *Configuration) Empty() bool {
	if c == nil {
		return true
	}
	return len(c.Rules) == 0
}

// NewConfiguration initializes a fresh lifecycle configuration
// for manipulation, such as setting and removing lifecycle rules
// and filters.
func NewConfiguration() *Configuration {
	return &Configuration{}
}
