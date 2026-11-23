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
	"net/http"
	"testing"
)

func TestCredContextLibraryOwnedClient(t *testing.T) {
	own, err := New("localhost:9000", &Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !own.CredContext().LibraryOwnedClient {
		t.Error("Expected a client without Options.Transport to report a library-owned HTTP client")
	}

	custom, err := New("localhost:9000", &Options{Transport: http.DefaultTransport.(*http.Transport).Clone()})
	if err != nil {
		t.Fatal(err)
	}
	if custom.CredContext().LibraryOwnedClient {
		t.Error("Expected a client with a caller-supplied transport not to report a library-owned HTTP client")
	}

	swapped, err := New("localhost:9000", &Options{})
	if err != nil {
		t.Fatal(err)
	}
	swapped.CredContext().Client.Transport = http.DefaultTransport.(*http.Transport).Clone()
	if swapped.CredContext().LibraryOwnedClient {
		t.Error("Expected a client whose transport was replaced after New not to report a library-owned HTTP client")
	}
}

func TestCredContextReplacedDefaultTransport(t *testing.T) {
	orig := DefaultTransport
	t.Cleanup(func() { DefaultTransport = orig })
	DefaultTransport = func(secure bool) (*http.Transport, error) {
		return orig(secure)
	}

	clnt, err := New("localhost:9000", &Options{})
	if err != nil {
		t.Fatal(err)
	}
	if clnt.CredContext().LibraryOwnedClient {
		t.Error("Expected a client built with a replaced DefaultTransport not to report a library-owned HTTP client")
	}
}
