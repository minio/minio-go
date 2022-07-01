package minio

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func Test_transportTimeoutWrapper_Response(t *testing.T) {
	t.Parallel()
	// We need a reasonably long timeout to ensure it is caught.
	wait := time.Second
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client")
		w.(http.Flusher).Flush()
		time.Sleep(wait)
	}))
	defer ts.Close()

	started := time.Now()
	client := http.Client{Transport: transportTimeoutWrapper(http.DefaultTransport, wait/2)}
	req, err := http.NewRequest("GET", ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.ReadAll(res.Body)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("expected DeadlineExceeded, got", err)
	}
	err = res.Body.Close()
	if err != context.DeadlineExceeded {
		t.Fatal("expected DeadlineExceeded, got", err)
	}
	if time.Since(started) >= wait {
		t.Fatalf("Took longer (%v) than double timeout (%v)", time.Since(started), wait)
	}
	t.Log("timeout took", time.Since(started))
}

func Test_transportTimeoutWrapper_ResponseTrickle(t *testing.T) {
	t.Parallel()
	// We need a reasonably long timeout to ensure it is caught.
	wait := time.Second
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for i := 0; i < 4; i++ {
			fmt.Fprintln(w, "Hello, client")
			w.(http.Flusher).Flush()
			time.Sleep(wait / 4)
		}
	}))
	defer ts.Close()

	client := http.Client{Transport: transportTimeoutWrapper(http.DefaultTransport, wait/2)}
	req, err := http.NewRequest("GET", ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.ReadAll(res.Body)
	if err != nil {
		t.Fatal("got error", err)
	}
	err = res.Body.Close()
	if err != nil {
		t.Fatal("got error", err)
	}
}

type fakeRoundtripper struct {
	// This many times...
	times int
	// read this many bytes from input...
	readEach int
	// ... then wait this long.
	wait time.Duration
}

func (f fakeRoundtripper) RoundTrip(request *http.Request) (*http.Response, error) {
	ctx := request.Context()
	defer request.Body.Close()
	var buf = make([]byte, f.readEach)
	for i := 0; i < f.times; i++ {
		_, err := io.ReadFull(request.Body, buf)
		if err != nil {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, request.Context().Err()
		case <-time.NewTimer(f.wait).C:
		}
	}
	_, err := io.Copy(io.Discard, request.Body)
	return &http.Response{}, err
}

func Test_transportTimeoutWrapper_Request(t *testing.T) {
	// Seems like `httptest.NewServer` buffers bodies, so we cannot use it.
	// Resort to manual testing
	t.Parallel()
	// We need a reasonably long timeout to ensure it is caught.
	const wait = time.Second
	const size = 10000

	transport := transportTimeoutWrapper(fakeRoundtripper{readEach: size, wait: wait, times: 1}, wait/2)
	client := http.Client{Transport: transport}

	req, err := http.NewRequest("GET", "", bytes.NewBuffer(make([]byte, size*2)))
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	_, err = client.Do(req)
	if time.Since(started) >= wait {
		t.Fatalf("Took longer (%v) than double timeout (%v)", time.Since(started), wait)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("expected timeout, got", err)
	}
}

func Test_transportTimeoutWrapper_RequestTrickle(t *testing.T) {
	// Seems like `httptest.NewServer` buffers bodies, so we cannot use it.
	// Resort to manual testing
	t.Parallel()
	// We need a reasonably long timeout to ensure it is caught.
	const wait = time.Second
	const size = 10000

	// Read every wait / 2, timeout is wait.
	transport := transportTimeoutWrapper(fakeRoundtripper{readEach: size / 2, wait: wait / 2, times: 4}, wait)
	client := http.Client{Transport: transport}

	req, err := http.NewRequest("GET", "", bytes.NewBuffer(make([]byte, size*2)))
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Do(req)
	if err != nil {
		t.Fatal("got error", err)
	}
}
