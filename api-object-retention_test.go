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
	"encoding/xml"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestRetainUntilDateIsUTC(t *testing.T) {
	date := time.Date(2030, 1, 2, 3, 4, 5, 0, time.FixedZone("UTC+2", 2*3600))
	want := "2030-01-02T01:04:05Z"

	if got := (PutObjectOptions{Mode: Governance, RetainUntilDate: date}).Header().Get(amzLockRetainUntil); got != want {
		t.Errorf("PutObjectOptions header: expected %q, got %q", want, got)
	}

	h := make(http.Header)
	CopyDestOptions{Mode: Governance, RetainUntilDate: date}.Marshal(h)
	if got := h.Get(amzLockRetainUntil); got != want {
		t.Errorf("CopyDestOptions header: expected %q, got %q", want, got)
	}

	mode := Governance
	retention, err := newObjectRetention(&mode, &date)
	if err != nil {
		t.Fatal(err)
	}
	body, err := xml.Marshal(retention)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "<RetainUntilDate>"+want+"</RetainUntilDate>") {
		t.Errorf("retention XML: expected RetainUntilDate %q, got %s", want, body)
	}
}
