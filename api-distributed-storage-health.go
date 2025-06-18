package minio

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type StorageStatus struct {
	Host   string `json:"host"`
	Status string `json:"status"`
	Region string `json:"region"`
}

func (c *Client) GetSDSHealth(ctx context.Context, bucketName, objectName string, opts GetObjectOptions) ([]StorageStatus, error) {
	resp, err := c.executeMethod(ctx, http.MethodGet, requestMetadata{contentSHA256Hex: emptySHA256Hex, health: true})
	defer closeResponse(resp)
	if err != nil {
		return nil, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, httpRespToErrorResponse(resp, "", "")
		}
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return []StorageStatus{}, err
	}

	healthCheck := []StorageStatus{}
	err = json.Unmarshal(body, &healthCheck)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return []StorageStatus{}, err
	}
	return healthCheck, nil
}
