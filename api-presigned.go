package minio

import (
	"errors"
	"fmt"
	"net/url"
	"time"
)

// PresignedGetObject returns a presigned URL to access an object without credentials.
// Expires maximum is 7days - ie. 604800 and minimum is 1.
func (a API) PresignedGetObject(bucketName, objectName string, expires time.Duration) (string, error) {
	if err := isValidExpiry(expires); err != nil {
		return "", err
	}
	expireSeconds := int64(expires / time.Second)
	return a.presignedGetObject(bucketName, objectName, expireSeconds, 0, 0)
}

// presignedGetObject - generate presigned get object URL.
func (a API) presignedGetObject(bucketName, objectName string, expires, offset, length int64) (string, error) {
	// get targetURL.
	targetURL, err := getTargetURL(a.endpointURL, bucketName, objectName, url.Values{})
	if err != nil {
		return "", err
	}

	// get bucket region.
	region, err := a.getRegion(bucketName)
	if err != nil {
		return "", err
	}

	// Instantiate a new request.
	req, err := newRequest("GET", targetURL, requestMetadata{
		credentials:      a.credentials,
		expires:          expires,
		userAgent:        a.userAgent,
		bucketRegion:     region,
		contentTransport: a.httpTransport,
	})
	if err != nil {
		return "", err
	}

	// Set ranges if length and offset are valid.
	if length > 0 && offset >= 0 {
		req.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, offset+length-1))
	} else if offset > 0 && length == 0 {
		req.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	} else if length > 0 && offset == 0 {
		req.Set("Range", fmt.Sprintf("bytes=-%d", length))
	}
	if req.credentials.Signature.isV2() {
		return req.PreSignV2()
	}
	return req.PreSignV4()
}

// PresignedPutObject returns a presigned URL to upload an object without credentials.
// Expires maximum is 7days - ie. 604800 and minimum is 1.
func (a API) PresignedPutObject(bucketName, objectName string, expires time.Duration) (string, error) {
	if err := isValidExpiry(expires); err != nil {
		return "", err
	}
	expireSeconds := int64(expires / time.Second)
	return a.presignedPutObject(bucketName, objectName, expireSeconds)
}

// presignedPutObject - generate presigned PUT url.
func (a API) presignedPutObject(bucketName, objectName string, expires int64) (string, error) {
	// get targetURL.
	targetURL, err := getTargetURL(a.endpointURL, bucketName, objectName, url.Values{})
	if err != nil {
		return "", err
	}

	// get bucket region.
	region, err := a.getRegion(bucketName)
	if err != nil {
		return "", err
	}

	// Instantiate a new request.
	req, err := newRequest("PUT", targetURL, requestMetadata{
		credentials:      a.credentials,
		expires:          expires,
		userAgent:        a.userAgent,
		bucketRegion:     region,
		contentTransport: a.httpTransport,
	})
	if err != nil {
		return "", err
	}
	if req.credentials.Signature.isV2() {
		return req.PreSignV2()
	}
	return req.PreSignV4()
}

// PresignedPostPolicy returns POST form data to upload an object at a location.
func (a API) PresignedPostPolicy(p *PostPolicy) (map[string]string, error) {
	if p.expiration.IsZero() {
		return nil, errors.New("Expiration time must be specified")
	}
	if _, ok := p.formData["key"]; !ok {
		return nil, errors.New("object key must be specified")
	}
	if _, ok := p.formData["bucket"]; !ok {
		return nil, errors.New("bucket name must be specified")
	}
	return a.presignedPostPolicy(p)
}

// presignedPostPolicy - generate post form data.
func (a API) presignedPostPolicy(p *PostPolicy) (map[string]string, error) {
	// get targetURL.
	targetURL, err := getTargetURL(a.endpointURL, p.formData["bucket"], "", url.Values{})
	if err != nil {
		return nil, err
	}

	// get bucket region.
	region, err := a.getRegion(p.formData["bucket"])
	if err != nil {
		return nil, err
	}

	// Instantiate a new request.
	req, err := newRequest("POST", targetURL, requestMetadata{
		credentials:      a.credentials,
		userAgent:        a.userAgent,
		bucketRegion:     region,
		contentTransport: a.httpTransport,
	})
	if err != nil {
		return nil, err
	}

	// Keep time.
	t := time.Now().UTC()
	if req.credentials.Signature.isV2() {
		policyBase64 := p.base64()
		p.formData["policy"] = policyBase64
		// for all other regions set this value to be 'AWSAccessKeyId'.
		if isGoogleEndpoint(a.endpointURL) {
			p.formData["GoogleAccessId"] = req.credentials.AccessKeyID
		} else {
			p.formData["AWSAccessKeyId"] = req.credentials.AccessKeyID
		}
		p.formData["signature"] = req.PostPresignSignatureV2(policyBase64)
		return p.formData, nil
	}
	credential := getCredential(req.credentials.AccessKeyID, req.bucketRegion, t)
	p.addNewPolicy(policyCondition{
		matchType: "eq",
		condition: "$x-amz-date",
		value:     t.Format(iso8601DateFormat),
	})
	p.addNewPolicy(policyCondition{
		matchType: "eq",
		condition: "$x-amz-algorithm",
		value:     authHeader,
	})
	p.addNewPolicy(policyCondition{
		matchType: "eq",
		condition: "$x-amz-credential",
		value:     credential,
	})
	policyBase64 := p.base64()
	p.formData["policy"] = policyBase64
	p.formData["x-amz-algorithm"] = authHeader
	p.formData["x-amz-credential"] = credential
	p.formData["x-amz-date"] = t.Format(iso8601DateFormat)
	p.formData["x-amz-signature"] = req.PostPresignSignatureV4(policyBase64, t)
	return p.formData, nil
}
