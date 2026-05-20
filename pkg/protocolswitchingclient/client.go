package protocolswitchingclient

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"sync"
	"time"

	"github.com/quic-go/quic-go/http3"
)

const (
	// StateUnknown when we don't currently know if server will support http or http3
	StateUnknown = iota // 0
	// StateHttp server is currently known to support http
	StateHttp // 1
	// StateHttp3 server is currently known to support http3
	StateHttp3 // 2
)

// HttpState one of the 'enum' values above
type HttpState int

// DynamicClient holds the state and client for a http/http3 client connection
type DynamicClient struct {
	mux              sync.RWMutex
	desiredHttpState HttpState
	httpState        HttpState
	timeToLive       time.Duration
	connectionStart  time.Time
	http3Client      *http.Client //client connection used for http3 requests
	fallbackClient   *http.Client //client connection used for http reequest
}

// NewDynamicClient create a new DynamicClient
func NewDynamicClient(transport http.RoundTripper, jar *cookiejar.Jar, CheckRedirect func(_ *http.Request, _ []*http.Request) error, DesiredHttp HttpState, TTL time.Duration) *DynamicClient {
	insecureSkipVerify := false
	//If the RoundTripper is actually a http.Transport pointer, use it's value for InsecureSkipVerify
	if p, ok := transport.(*http.Transport); ok && p.TLSClientConfig != nil {
		insecureSkipVerify = p.TLSClientConfig.InsecureSkipVerify
	}

	http3Transport := &http3.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecureSkipVerify, // For local dev/self-signed certs
			MinVersion:         tls.VersionTLS13, // Lowest version that will support http3/quic
		},
	}

	// Create HTTP client using the HTTP/3 transport
	http3Client := &http.Client{
		Transport:     http3Transport,
		CheckRedirect: CheckRedirect,
	}

	//If jar is nil, this would core dump,
	//there is a difference between a unspecified jar, and Jar explictly set to nil
	if jar != nil {
		http3Client.Jar = jar
	}

	fallbackClient := &http.Client{
		Transport:     transport,
		CheckRedirect: CheckRedirect,
	}
	if jar != nil {
		fallbackClient.Jar = jar
	}

	return &DynamicClient{timeToLive: TTL, desiredHttpState: DesiredHttp, httpState: StateUnknown, fallbackClient: fallbackClient, http3Client: http3Client}
}

// ResponseAndError holds the response from a web request
type ResponseAndError struct {
	resp  *http.Response
	err   error
	http3 bool
}

// close You need to call Close on DynamicConnection once you are done using it
func (dc *DynamicClient) Close() {
	dc.mux.Lock()
	if p, ok := dc.http3Client.Transport.(*http3.Transport); ok {
		p.Close()
	}
	dc.mux.Unlock()
}

// Reset simulates a TTL expire
func (dc *DynamicClient) Reset() {
	dc.mux.Lock()
	if dc.desiredHttpState == StateHttp3 {
		dc.httpState = StateUnknown
	} else {
		dc.httpState = StateHttp
	}
	dc.mux.Unlock()
}

func (dc *DynamicClient) GetHttpClient() *http.Client {
	return dc.fallbackClient
}

// synchronouslyTestBoth when http3 needs to be revalidated,
// on first connect or TTL expire retest
// do call on both http and http3 clients, if http return first
// wait another 150 ms to make sure http3 isn't coming.
// if http3 doesn't come in time, cancel it and create go routine to wait until it's returned
// making sure we don't leak a file descriptor
func (dc *DynamicClient) synchronouslyTestBoth(req *http.Request) (*http.Response, HttpState, error) {
	ch := make(chan ResponseAndError)
	go func() {
		//trying http3 path
		resp, err := dc.http3Client.Do(req)
		ret := ResponseAndError{resp: resp, err: err, http3: true}
		ch <- ret
	}()
	go func() {
		//trying http path
		resp, err := dc.fallbackClient.Do(req)
		ret := ResponseAndError{resp: resp, err: err, http3: false}
		ch <- ret
	}()

	ret := <-ch

	//if first error free response was http3
	if ret.http3 && ret.err == nil {
		go func() {
			//cleaning up http response
			httpRet := <-ch
			if httpRet.err == nil {
				httpRet.resp.Body.Close()
			}
		}()
		return ret.resp, StateHttp3, ret.err
	}

	// http returned first, wait extra 150ms for http3
	var http3Ret ResponseAndError
	select {
	case http3Ret = <-ch:
		// Preferred http3 endpoint responded in time
		if http3Ret.err == nil {
			// Close un-needed http resp
			if ret.resp != nil && ret.resp.Body != nil {
				ret.resp.Body.Close()
			}
			return http3Ret.resp, StateHttp3, http3Ret.err
		}
		//http3 had an error, utilize http
		return ret.resp, StateHttp, ret.err

	case <-time.After(150 * time.Millisecond):
	}

	//http3 endpoints timed out, return http
	go func() {
		//cleaning up http3 resp
		http3Ret = <-ch
			if http3Ret.resp != nil && http3Ret.resp.Body != nil {
				http3Ret.resp.Body.Close()
			}
	}()

	// return http response
	return ret.resp, StateHttp, ret.err
}

// Do based on desired http protocol, try http3 and fallback to http if that fails
// only do both protocals when trying to figure out which you can use, can ove onto 'fast' path
// NewDynamicClient makes sure all pointers are inited
func (dc *DynamicClient) Do(req *http.Request) (resp *http.Response, err error) {
	if req == nil {
		return nil, fmt.Errorf("incoming http.Request pointer was nil")
	}

	dc.mux.RLock()
	desiredHttpState := dc.desiredHttpState
	connectionStart := dc.connectionStart
	timeToLive := dc.timeToLive
	dc.mux.RUnlock()

	// If we only want to use http
	if desiredHttpState == StateHttp {
		return dc.fallbackClient.Do(req)
	}

	// See if we need to retry Http3 connection
	if time.Since(connectionStart) > timeToLive {
		dc.mux.Lock()
		dc.connectionStart = time.Now()
		dc.httpState = StateUnknown
		dc.mux.Unlock()
	}

	dc.mux.RLock()
	originalEnabledHttp3 := dc.httpState
	dc.mux.RUnlock()

	var state HttpState
	if originalEnabledHttp3 == StateUnknown {
		resp, state, err = dc.synchronouslyTestBoth(req)
		dc.mux.Lock()
		dc.httpState = state
		dc.mux.Unlock()
		return
	}

	if originalEnabledHttp3 == StateHttp3 {
		resp, err = dc.http3Client.Do(req)
		if err != nil {
			dc.mux.Lock()
			dc.httpState = StateHttp
			dc.mux.Unlock()
			resp, err := dc.fallbackClient.Do(req)
			// both http and http3 failed
			// try to reconnect using http3 next time,
			// if http3 fails and http works a second time, we'll switch to http
			if err == nil {
				dc.mux.Lock()
				dc.httpState = StateUnknown
				dc.mux.Unlock()
			}
			return resp, err
		}
		return resp, err
	}

	//server is currently only excepting http requests, don't waste time on http3
	resp, err = dc.fallbackClient.Do(req)
	if err != nil {
		dc.mux.Lock()
		dc.httpState = originalEnabledHttp3
		dc.mux.Unlock()
	}
	return
}
