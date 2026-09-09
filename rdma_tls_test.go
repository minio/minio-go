/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2024-2026 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * SPDX-License-Identifier: Apache-2.0
 */

package minio

import (
	"crypto/tls"
	"net/http"
	"testing"
)

type wrappedRoundTripper struct{ inner http.RoundTripper }

func (w wrappedRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return w.inner.RoundTrip(r)
}

func TestRDMASkipCertVerify(t *testing.T) {
	insecureTransport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // the case under test
	}

	tests := []struct {
		name   string
		client *http.Client
		secure bool
		want   bool
	}{
		// The case this exists for: a caller that opted out of verification
		// must have that honored on the RDMA path too, or RDMA is the only
		// thing that fails against a self-signed endpoint.
		{
			name:   "insecure transport over https",
			client: &http.Client{Transport: insecureTransport},
			secure: true,
			want:   true,
		},
		{
			name:   "verifying transport over https",
			client: &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}}},
			secure: true,
			want:   false,
		},
		// No TLS at all: there is nothing to verify, so never claim a skip.
		{
			name:   "insecure transport without https",
			client: &http.Client{Transport: insecureTransport},
			secure: false,
			want:   false,
		},
		{
			name:   "transport with no TLS config",
			client: &http.Client{Transport: &http.Transport{}},
			secure: true,
			want:   false,
		},
		{
			name:   "default transport",
			client: &http.Client{},
			secure: true,
			want:   false,
		},
		{
			name:   "nil client",
			client: nil,
			secure: true,
			want:   false,
		},
		// A wrapped transport cannot be inspected, so it keeps verification
		// rather than guessing its way into a weaker setting.
		{
			name:   "wrapped insecure transport is not inspectable",
			client: &http.Client{Transport: wrappedRoundTripper{inner: insecureTransport}},
			secure: true,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rdmaSkipCertVerify(tt.client, tt.secure); got != tt.want {
				t.Errorf("rdmaSkipCertVerify() = %v, want %v", got, tt.want)
			}
		})
	}
}
