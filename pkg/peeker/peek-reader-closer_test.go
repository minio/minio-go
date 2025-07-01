/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2015-2025 MinIO, Inc.
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

package peeker

import (
	"bytes"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPeekReadCloser(t *testing.T) {
	body := make([]byte, 1024*1024)
	rand.Read(body)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
	defer ts.Close()

	for _, pLen := range []int{1, 4, 1024, 32 * 1024} {
		res, err := http.Get(ts.URL)
		if err != nil {
			t.Fatal(err)
		}
		prc := NewPeekReadCloser(res.Body, 1024)

		p := make([]byte, pLen)
		n, err := prc.Read(p)
		if err != nil {
			t.Fatalf("Read() returned an error while len(p) = %d, error: %v", pLen, err)
		}
		if n != pLen {
			t.Fatalf("unexpected read bytes length, expected: `%d`, found: `%d`", pLen, n)
		}

		prc.ReplayFromStart()
		all, err := ioutil.ReadAll(prc)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(all, body) {
			t.Fatal("unexpected content after replay")
		}
	}
}
