package minio

import (
	"context"
	"encoding/xml"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/minio/minio-go/v7/pkg/encrypt"
	"github.com/minio/minio-go/v7/pkg/s3utils"
)

// ObjectAttributesOptions ...
// https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetObjectAttributes.html
type ObjectAttributesOptions struct {
	ObjectAttributes     string
	VersionID            string
	MaxParts             int
	PartNumberMarker     int
	ServerSideEncryption encrypt.ServerSide

	// Bucket onwer is an S3 specific parameter
	BucketOwner string
}

// ObjectAttributes ...
// https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetObjectAttributes.html
type ObjectAttributes struct {
	ObjectAttributesResponse
	LastModified   time.Time
	VersionID      string
	RequestCharged string
}

// ParseResponse ...
// https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetObjectAttributes.html
func (o *ObjectAttributes) ParseResponse(resp *http.Response) (err error) {
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
// https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetObjectAttributes.html
type ObjectAttributesResponse struct {
	ETag     string `xml:",omitempty"`
	Checksum struct {
		ChecksumCRC32  string `xml:",omitempty"`
		ChecksumCRC32C string `xml:",omitempty"`
		ChecksumSHA1   string `xml:",omitempty"`
		ChecksumSHA256 string `xml:",omitempty"`
	}
	ObjectParts struct {
		PartsCount           int
		PartNumberMarker     int
		NextPartNumberMarker int
		MaxParts             int
		IsTruncated          bool
		Parts                []*struct {
			ChecksumCRC32  string `xml:",omitempty"`
			ChecksumCRC32C string `xml:",omitempty"`
			ChecksumSHA1   string `xml:",omitempty"`
			ChecksumSHA256 string `xml:",omitempty"`
			PartNumber     int
			Size           int
		} `xml:"Part"`
	}
	StorageClass string
	ObjectSize   int
}

// GetObjectAttributes verifies if object exists, you have permission to access it
// and returns information about the object.
// https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetObjectAttributes.html
func (c *Client) GetObjectAttributes(ctx context.Context, bucketName, objectName string, opts ObjectAttributesOptions) (ObjectAttributes, error) {
	// Input validation.

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
	headers.Set(amzObjectAttributes, opts.ObjectAttributes)

	if len(opts.ObjectAttributes) < 1 {
		return ObjectAttributes{}, errors.New("object attribute tags are required")
	}

	headers.Set(amzPartNumberMarker, strconv.Itoa(opts.PartNumberMarker))

	if opts.BucketOwner != "" {
		headers.Set(amzExpectedBucketOnwer, opts.BucketOwner)
	}

	if opts.MaxParts != 0 {
		headers.Set(amzMaxParts, strconv.Itoa(opts.MaxParts))
	}

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

	if resp.StatusCode != http.StatusOK {
		ER := new(ErrorResponse)
		if err := xml.NewDecoder(resp.Body).Decode(ER); err != nil {
			return ObjectAttributes{}, err
		}

		return ObjectAttributes{}, *ER
	}

	defer closeResponse(resp)

	OA := new(ObjectAttributes)
	err = OA.ParseResponse(resp)
	if err != nil {
		return ObjectAttributes{}, err
	}

	return *OA, nil
}
