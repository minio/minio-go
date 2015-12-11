package minio

import (
	"net"
	"net/http"
	"net/url"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

// SignatureType is type of Authorization requested for a given HTTP request.
type SignatureType int

// Different types of supported signatures - default is Latest i.e SignatureV4.
const (
	Latest SignatureType = iota
	SignatureV4
	SignatureV2
)

// isV2 - is signature SignatureV2?
func (s SignatureType) isV2() bool {
	return s == SignatureV2
}

// isV4 - is signature SignatureV4?
func (s SignatureType) isV4() bool {
	return s == SignatureV4
}

// isLatest - is signature Latest?
func (s SignatureType) isLatest() bool {
	return s == Latest
}

// clientCredentials - main configuration struct used for credentials.
type clientCredentials struct {
	///  Standard options.
	AccessKeyID     string        // AccessKeyID required for authorized requests.
	SecretAccessKey string        // SecretAccessKey required for authorized requests.
	Signature       SignatureType // choose a signature type if necessary.
}

// Global constants.
const (
	libraryName    = "minio-go"
	libraryVersion = "0.2.5"
)

// User Agent should always following the below style.
// Please open an issue to discuss any new changes here.
//
//       Minio (OS; ARCH) LIB/VER APP/VER
const (
	libraryUserAgentPrefix = "Minio (" + runtime.GOOS + "; " + runtime.GOARCH + ") "
	libraryUserAgent       = libraryUserAgentPrefix + libraryName + "/" + libraryVersion
)

// API is a container which delegates methods that comply with CloudStorageAPI interface.
type API struct {
	// Needs allocation.
	mutex     *sync.Mutex
	regionMap map[string]string

	// User supplied.
	userAgent   string
	credentials *clientCredentials
	endpointURL *url.URL

	// This http transport is usually needed for debugging OR to add your own
	// custom TLS certificates on the client transport, for custom CA's and
	// certs which are not part of standard certificate authority.
	httpTransport http.RoundTripper
}

// validEndpointDomain - regex for validating domain names.
var validEndpointDomain = regexp.MustCompile(`^(([a-zA-Z]{1})|([a-zA-Z]{1}[a-zA-Z]{1})|([a-zA-Z]{1}[0-9]{1})|([0-9]{1}[a-zA-Z]{1})|([a-zA-Z0-9][a-zA-Z0-9-_]{1,61}[a-zA-Z0-9]))\.([a-zA-Z]{2,6}|[a-zA-Z0-9-]{2,30}\.[a-zA-Z]{2,3})$`)

// validIPAddress - regex for validating ip address.
var validIPAddress = regexp.MustCompile(`^(\d+\.){3}\d+$`)

// getEndpointURL - construct a new endpoint.
func getEndpointURL(endpoint string, inSecure bool) (*url.URL, error) {
	if strings.Contains(endpoint, ":") {
		host, _, err := net.SplitHostPort(endpoint)
		if err != nil {
			return nil, err
		}
		if !validIPAddress.MatchString(host) && !validEndpointDomain.MatchString(host) {
			msg := "Endpoint: " + endpoint + " does not follow ip address or domain name standards."
			return nil, ErrInvalidArgument(msg)
		}
	} else {
		if !validIPAddress.MatchString(endpoint) && !validEndpointDomain.MatchString(endpoint) {
			msg := "Endpoint: " + endpoint + " does not follow ip address or domain name standards."
			return nil, ErrInvalidArgument(msg)
		}
	}
	// if inSecure is true, use 'http' scheme.
	scheme := "https"
	if inSecure {
		scheme = "http"
	}

	// Construct a secured endpoint URL.
	endpointURL := new(url.URL)
	endpointURL.Host = endpoint
	endpointURL.Scheme = scheme

	// Validate incoming endpoint URL.
	if err := isValidEndpointURL(endpointURL); err != nil {
		return nil, err
	}
	return endpointURL, nil
}

// NewV2 - instantiate minio client API with signature version '2'.
func NewV2(endpoint string, accessKeyID, secretAccessKey string, inSecure bool) (API, error) {
	// construct endpoint.
	endpointURL, err := getEndpointURL(endpoint, inSecure)
	if err != nil {
		return API{}, err
	}

	// create a new client Config.
	credentials := &clientCredentials{}
	credentials.AccessKeyID = accessKeyID
	credentials.SecretAccessKey = secretAccessKey
	credentials.Signature = SignatureV2

	// instantiate new API.
	api := API{
		// Allocate.
		mutex:     &sync.Mutex{},
		regionMap: make(map[string]string),
		// Save for lower level calls.
		userAgent:   libraryUserAgent,
		credentials: credentials,
		endpointURL: endpointURL,
	}
	return api, nil
}

// NewV4 - instantiate minio client API with signature version '4'.
func NewV4(endpoint string, accessKeyID, secretAccessKey string, inSecure bool) (API, error) {
	// construct endpoint.
	endpointURL, err := getEndpointURL(endpoint, inSecure)
	if err != nil {
		return API{}, err
	}

	// create a new client Config.
	credentials := &clientCredentials{}
	credentials.AccessKeyID = accessKeyID
	credentials.SecretAccessKey = secretAccessKey
	credentials.Signature = SignatureV4

	// instantiate new API.
	api := API{
		// Allocate.
		mutex:     &sync.Mutex{},
		regionMap: make(map[string]string),
		// Save for lower level calls.
		userAgent:   libraryUserAgent,
		credentials: credentials,
		endpointURL: endpointURL,
	}
	return api, nil
}

// New - instantiate minio client API.
/// TODO - add automatic verification of signature.
func New(endpoint string, accessKeyID, secretAccessKey string, inSecure bool) (API, error) {
	// construct endpoint.
	endpointURL, err := getEndpointURL(endpoint, inSecure)
	if err != nil {
		return API{}, err
	}

	// create a new client Config.
	credentials := &clientCredentials{}
	credentials.AccessKeyID = accessKeyID
	credentials.SecretAccessKey = secretAccessKey

	// Google cloud storage should be set to signature V2, force it if not.
	if isGoogleEndpoint(endpointURL) {
		credentials.Signature = SignatureV2
	}
	// If Amazon S3 set to signature v2.
	if isAmazonEndpoint(endpointURL) {
		credentials.Signature = SignatureV4
	}

	// instantiate new API.
	api := API{
		// Allocate.
		mutex:     &sync.Mutex{},
		regionMap: make(map[string]string),
		// Save for lower level calls.
		userAgent:   libraryUserAgent,
		credentials: credentials,
		endpointURL: endpointURL,
	}
	return api, nil
}
