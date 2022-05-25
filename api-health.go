/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2015-2022 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package minio

import (
	"context"
	"net/http"
	"strconv"
)

// ClusterHealthResult represents the cluster health result
type ClusterHealthResult struct {
	Healthy         bool
	MaintenanceMode bool
	WriteQuorum     int
	HealingDrives   int
}

// ClusterHealth will hit `/minio/health/cluster` anonymous API to check the cluster health
func (c *Client) ClusterHealth(ctx context.Context) (result ClusterHealthResult, err error) {
	resp, err := c.executeMethod(ctx, http.MethodGet, requestMetadata{
		relPath: "minio/health/cluster",
	})
	defer closeResponse(resp)
	if err != nil {
		return result, err
	}

	if resp != nil {
		writeQuorumStr := resp.Header.Get(minioWriteQuorumHeader)
		if writeQuorumStr != "" {
			result.WriteQuorum, err = strconv.Atoi(writeQuorumStr)
			if err != nil {
				return result, err
			}
		}
		healingDrivesStr := resp.Header.Get(minIOHealingDrives)
		if healingDrivesStr != "" {
			result.HealingDrives, err = strconv.Atoi(healingDrivesStr)
			if err != nil {
				return result, err
			}
		}
		switch resp.StatusCode {
		case http.StatusOK:
			result.Healthy = true
		case http.StatusPreconditionFailed:
			result.MaintenanceMode = true
		default:
			// Not Healthy
		}
	}
	return result, nil
}
