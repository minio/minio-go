package minio

import (
	"context"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// RoundTrip(*Request) (*Response, error)

type mockRoundTrip struct {
	callback func(*http.Request) (*http.Response, error)
}

func (m mockRoundTrip) RoundTrip(request *http.Request) (*http.Response, error) {
	return m.callback(request)
}

func Test_PresignedPostPolicy(t *testing.T) {
	t.Run("non-AWS vendor", func(t *testing.T) {
		client, err := New("localhost:9000", "minioadmin", "minioadmin", false)
		assert.NoError(t, err)

		policy := NewPostPolicy()
		_ = policy.SetBucket("myBucket")
		_ = policy.SetKey("myObject")
		_ = policy.SetExpires(time.Now().Add(5 * time.Minute))

		url, formData, err := client.PresignedPostPolicy(context.Background(), policy)

		if nil != err {
			t.Errorf("failed executing client.PresignedPostPolicy: %s", err)
		}

		if url.String() != "http://localhost:9000/myBucket/" {
			t.Errorf("unexpected URL: %s", url.String())
		}

		if formData["bucket"] != "myBucket" {
			t.Errorf("unexpected bucket: %s", formData["bucket"])
		}

		if formData["key"] != "myObject" {
			t.Errorf("unexpected key: %s", formData["key"])
		}

		if _, ok := formData["x-amz-signature"]; !ok {
			t.Errorf("missing signagure")
		}
	})

	t.Run("AWS vendor", func(t *testing.T) {
		client, err := New("s3.amazonaws.com", "accessKey", "secretKey", true)
		client.httpClient.Transport = mockRoundTrip{
			callback: func(request *http.Request) (*http.Response, error) {
				response := &http.Response{
					Status:     "OK",
					StatusCode: http.StatusOK,
				}

				content := `<?xml version="1.0" encoding="UTF-8"?>`
				content += `<LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/">Europe</LocationConstraint>`
				body := strings.NewReader(content)
				response.Body = ioutil.NopCloser(body)

				return response, nil
			},
		}
		assert.NoError(t, err)

		policy := NewPostPolicy()
		_ = policy.SetBucket("myBucket")
		_ = policy.SetKey("myObject")
		_ = policy.SetExpires(time.Now().Add(5 * time.Minute))

		url, formData, err := client.PresignedPostPolicy(context.Background(), policy)

		if nil != err {
			t.Errorf("failed executing client.PresignedPostPolicy: %s", err)
		}

		if url.String() != "https://myBucket.s3.dualstack.us-east-1.amazonaws.com/" {
			t.Errorf("unexpected URL: %s", url.String())
		}

		if formData["bucket"] != "myBucket" {
			t.Errorf("unexpected bucket: %s", formData["bucket"])
		}

		if formData["key"] != "myObject" {
			t.Errorf("unexpected key: %s", formData["key"])
		}

		if _, ok := formData["x-amz-signature"]; !ok {
			t.Errorf("missing signagure")
		}
	})
}
