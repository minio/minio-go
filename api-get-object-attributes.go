package minio

import (
	"context"
	"encoding/xml"
	"net/http"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7/pkg/encrypt"
	"github.com/minio/minio-go/v7/pkg/s3utils"
)

// ObjectAttributesOptions is an API call that combines
// HeadObject and ListParts.
//
// VersionID - The object version you want to attributes for
// ServerSideEncryption - The server-side encryption algorithm used when storing this object in Minio
type ObjectAttributesOptions struct {
	VersionID            string
	ServerSideEncryption encrypt.ServerSide
}

// ObjectAttributes ...
type ObjectAttributes struct {
	ObjectAttributesResponse
	LastModified time.Time
	VersionID    string
}

func (o *ObjectAttributes) parseResponse(resp *http.Response) (err error) {
	mod, err := parseRFC7231Time(resp.Header.Get("Last-Modified"))
	if err != nil {
		return err
	}
	o.LastModified = mod
	o.VersionID = resp.Header.Get(amzVersionID)

	response := new(ObjectAttributesResponse)
	if err := xml.NewDecoder(resp.Body).Decode(response); err != nil {
		return err
	}
	o.ObjectAttributesResponse = *response

	return
}

// ObjectAttributesResponse ...
type ObjectAttributesResponse struct {
	ETag         string `xml:",omitempty"`
	StorageClass string
	ObjectSize   int
	Checksum     struct {
		ChecksumCRC32  string `xml:",omitempty"`
		ChecksumCRC32C string `xml:",omitempty"`
		ChecksumSHA1   string `xml:",omitempty"`
		ChecksumSHA256 string `xml:",omitempty"`
	}
	ObjectParts struct {
		PartsCount int
		Parts      []*ObjectAttributePart `xml:"Part"`
	}
}

// ObjectAttributesResponse ...
type ObjectAttributePart struct {
	ChecksumCRC32  string `xml:",omitempty"`
	ChecksumCRC32C string `xml:",omitempty"`
	ChecksumSHA1   string `xml:",omitempty"`
	ChecksumSHA256 string `xml:",omitempty"`
	PartNumber     int
	Size           int
}

// GetObjectAttributes ...
// This API combines HeadObject and ListParts.
func (c *Client) GetObjectAttributes(ctx context.Context, bucketName, objectName string, opts ObjectAttributesOptions) (ObjectAttributes, error) {
	if err := s3utils.CheckValidBucketName(bucketName); err != nil {
		return ObjectAttributes{}, err
	}

	if err := s3utils.CheckValidObjectName(objectName); err != nil {
		return ObjectAttributes{}, err
	}

	urlValues := make(url.Values)
	urlValues.Add("attributes", "")
	if opts.VersionID != "" {
		urlValues.Add("versionId", opts.VersionID)
	}

	headers := make(http.Header)
	headers.Set(amzObjectAttributes, GetObjectAttributesTags)

	headers.Set(amzPartNumberMarker, "0")
	headers.Set(amzMaxParts, "0")

	if opts.ServerSideEncryption != nil {
		opts.ServerSideEncryption.Marshal(headers)
	}

	resp, err := c.executeMethod(ctx, http.MethodGet, requestMetadata{
		bucketName:       bucketName,
		objectName:       objectName,
		queryValues:      urlValues,
		contentSHA256Hex: emptySHA256Hex,
		customHeader:     headers,
	})
	if err != nil {
		return ObjectAttributes{}, err
	}
	defer closeResponse(resp)

	if resp.StatusCode != http.StatusOK {
		ER := new(ErrorResponse)
		if err := xml.NewDecoder(resp.Body).Decode(ER); err != nil {
			return ObjectAttributes{}, err
		}

		return ObjectAttributes{}, *ER
	}

	OA := new(ObjectAttributes)
	err = OA.parseResponse(resp)
	if err != nil {
		return ObjectAttributes{}, err
	}

	return *OA, nil
}
