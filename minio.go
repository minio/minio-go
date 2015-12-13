package minio

import "runtime"

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
