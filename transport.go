/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2017-2022 MinIO, Inc.
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
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

// mustGetSystemCertPool - return system CAs or empty pool in case of error (or windows)
func mustGetSystemCertPool() *x509.CertPool {
	pool, err := x509.SystemCertPool()
	if err != nil {
		return x509.NewCertPool()
	}
	return pool
}

// DefaultTransport - this default transport is similar to
// http.DefaultTransport but with additional param  DisableCompression
// is set to true to avoid decompressing content with 'gzip' encoding.
var DefaultTransport = func(secure bool) (*http.Transport, error) {
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          256,
		MaxIdleConnsPerHost:   16,
		ResponseHeaderTimeout: time.Minute,
		IdleConnTimeout:       time.Minute,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 10 * time.Second,
		// Set this value so that the underlying transport round-tripper
		// doesn't try to auto decode the body of objects with
		// content-encoding set to `gzip`.
		//
		// Refer:
		//    https://golang.org/src/net/http/transport.go?h=roundTrip#L1843
		DisableCompression: true,
	}

	if secure {
		tr.TLSClientConfig = &tls.Config{
			// Can't use SSLv3 because of POODLE and BEAST
			// Can't use TLSv1.0 because of POODLE and BEAST using CBC cipher
			// Can't use TLSv1.1 because of RC4 cipher usage
			MinVersion: tls.VersionTLS12,
		}
		if f := os.Getenv("SSL_CERT_FILE"); f != "" {
			rootCAs := mustGetSystemCertPool()
			data, err := ioutil.ReadFile(f)
			if err == nil {
				rootCAs.AppendCertsFromPEM(data)
			}
			tr.TLSClientConfig.RootCAs = rootCAs
		}
	}
	return tr, nil
}

// transportTimeoutWrapper wraps a transport to time out on request and response body transfers.
func transportTimeoutWrapper(tripper http.RoundTripper, timeout time.Duration) http.RoundTripper {
	if timeout <= 0 {
		return tripper
	}
	return &transportTimeout{parent: tripper, timeout: timeout}
}

type transportTimeout struct {
	parent  http.RoundTripper
	timeout time.Duration
}

func (t *transportTimeout) RoundTrip(request *http.Request) (*http.Response, error) {
	request, ctx, cancel := newReqAliveChecker(request, t.timeout)
	resp, err := t.parent.RoundTrip(request)
	if err != nil {
		cancel()
		return resp, err
	}
	return newRespAliveChecker(ctx, cancel, resp, t.timeout), nil
}

// newReqAliveChecker will perform timeout checks between read calls.
// This will measure time between body reads.
// When the timeout is exceeded the request will be canceled.
// A new request and context is returned.
//
func newReqAliveChecker(req *http.Request, timeout time.Duration) (request *http.Request, ctx context.Context, cancel context.CancelFunc) {
	ctx, cancel = context.WithCancel(req.Context())
	req = req.WithContext(ctx)

	if req.Body == nil || timeout <= 0 {
		return req, ctx, cancel
	}
	req.Body = &reqAliveChecker{timeout: timeout, rc: req.Body, ctx: ctx, cancel: cancel}
	return req, ctx, cancel
}

type reqAliveChecker struct {
	timedOut uint32
	timeout  time.Duration
	rc       io.ReadCloser

	reset  chan struct{}
	ctx    context.Context
	cancel context.CancelFunc
}

func (a *reqAliveChecker) Read(p []byte) (n int, err error) {
	if atomic.LoadUint32(&a.timedOut) == 1 {
		return 0, context.DeadlineExceeded
	}
	n, err = a.rc.Read(p)
	// Start on first request
	if err == nil {
		if a.reset == nil {
			update := make(chan struct{}, 1)
			a.reset = update
			go func() {
				t := time.NewTimer(a.timeout)
				defer t.Stop()
				for {
					select {
					case _, ok := <-update:
						if !ok {
							// Close was called...
							return
						}
						// Reset on update...
						t.Reset(a.timeout)
					case <-a.ctx.Done():
						atomic.StoreUint32(&a.timedOut, 1)
						a.cancel()
					case <-t.C:
						atomic.StoreUint32(&a.timedOut, 1)
						a.cancel()
					}
				}
			}()
		} else {
			a.reset <- struct{}{}
		}
	}
	return n, err
}

func (a *reqAliveChecker) Close() error {
	if a.reset != nil {
		close(a.reset)
		a.reset = nil
	}
	return a.rc.Close()
}

// newRespAliveChecker will check reads for individual timeouts.
// Any timeout <= 0 will disable timeouts and just return rc.
// When a timeout is hit context.DeadlineExceeded is returned from the reader.
// The context of the original request (if available) should still be canceled
// to release resources of call if possible.
func newRespAliveChecker(ctx context.Context, cancel context.CancelFunc, resp *http.Response, timeout time.Duration) *http.Response {
	if resp == nil {
		return nil
	}
	if timeout <= 0 || resp.Body == nil {
		return resp
	}
	resp.Body = &respAliveChecker{rc: resp.Body, timeout: timeout,
		comms:  make(chan readResponse, 1),
		ctx:    ctx,
		cancel: cancel,
	}

	return resp
}

type respAliveChecker struct {
	timeout  time.Duration
	rc       io.ReadCloser
	comms    chan readResponse
	timedOut bool

	// When wrapping
	ctx    context.Context
	cancel context.CancelFunc
}

type readResponse struct {
	n   int
	err error
}

func (a *respAliveChecker) Read(p []byte) (n int, err error) {
	if a.timedOut {
		return 0, context.DeadlineExceeded
	}
	go func() {
		var res readResponse
		res.n, res.err = a.rc.Read(p)
		a.comms <- res
	}()
	// Default nil, never fires
	var done <-chan struct{}
	if a.ctx != nil {
		done = a.ctx.Done()
	}
	select {
	case <-done:
		if a.cancel != nil {
			a.cancel()
		}
		a.timedOut = true
		return 0, a.ctx.Err()
	case <-time.After(a.timeout):
		a.timedOut = true
		if a.cancel != nil {
			a.cancel()
		}
		return 0, context.DeadlineExceeded
	case v := <-a.comms:
		return v.n, v.err
	}
}

func (a *respAliveChecker) Close() error {
	if a.timedOut {
		select {
		case <-a.comms:
			// Read returned.
		default:
			// We are still blocked on a read.
			// Spawn a goroutine that waits for the request to finish before we Close.
			for range a.comms {
				a.rc.Close()
			}
			return context.DeadlineExceeded
		}
	} else if a.cancel != nil {
		a.cancel()
	}
	return a.rc.Close()
}
