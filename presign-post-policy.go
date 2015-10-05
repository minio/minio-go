package minio

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

const expirationFormat = "2006-01-02T15:04:05.999Z"

// PolicyConditions currently supported conditions for presign POST
type PolicyConditions struct {
	Expires      int
	Bucket       string
	Object       string
	ObjectPrefix bool
	ContentType  string
}

// ContentLengthRange - min and max size of content
type ContentLengthRange struct {
	Min int
	Max int
}

func (c ContentLengthRange) marshalJSON() string {
	return fmt.Sprintf("[\"content-length-range\",%d,%d]", c.Min, c.Max)
}

// Policy explanation: http://docs.aws.amazon.com/AmazonS3/latest/API/sigv4-HTTPPOSTConstructPolicy.html
type Policy struct {
	MatchType string
	Key       string
	Value     string
}

func (policy Policy) marshalJSON() string {
	return fmt.Sprintf("[\"%s\",\"%s\",\"%s\"]", policy.MatchType, policy.Key, policy.Value)
}

// PostPolicyForm provides strict static type conversion and validation for Amazon S3's POST policy JSON string.
type PostPolicyForm struct {
	Expiration         time.Time // Expiration date and time of the POST policy.
	Policies           []Policy
	ContentLengthRange ContentLengthRange
}

// MarshalJSON provides Marshalled JSON
func (P PostPolicyForm) MarshalJSON() ([]byte, error) {
	expirationstr := ""
	if P.Expiration.IsZero() == false {
		expirationstr = `"expiration":"` + P.Expiration.Format(expirationFormat) + `"`
	}
	policiesstr := ""
	policies := []string{}
	for _, policy := range P.Policies {
		policies = append(policies, policy.marshalJSON())
	}
	if P.ContentLengthRange.Min != 0 || P.ContentLengthRange.Max != 0 {
		policies = append(policies, P.ContentLengthRange.marshalJSON())
	}
	if len(policies) > 0 {
		policiesstr = `"conditions":[` + strings.Join(policies, ",") + "]"
	}
	retstr := "{"
	if len(expirationstr) > 0 {
		retstr = retstr + expirationstr
	}
	if len(policiesstr) > 0 {
		retstr = retstr + "," + policiesstr
	}
	retstr = retstr + "}"
	return []byte(retstr), nil
}

// Base64 Base64() of PostPolicyForm's Marshalled json
func (P PostPolicyForm) Base64() (string, error) {
	b, err := P.MarshalJSON()
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
