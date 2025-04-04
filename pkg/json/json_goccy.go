//go:build !stdlibjson

package json

import "github.com/goccy/go-json"

var (
	Unmarshal  = json.Unmarshal
	Marshal    = json.Marshal
	NewEncoder = json.NewEncoder
	NewDecoder = json.NewDecoder
)

type (
	Encoder = json.Encoder
	Decoder = json.Decoder
)
