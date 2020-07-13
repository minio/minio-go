// +build ignore

/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2020 MinIO, Inc.
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

package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/minio/minio-go/v7"
)

func main() {
	// Note: YOUR-ACCESSKEYID, YOUR-SECRETACCESSKEY and my-bucketname are
	// dummy values, please replace them with original values.

	// Requests are always secure (HTTPS) by default. Set secure=false to enable insecure (HTTP) access.
	// This boolean value is the last argument for New().

	// New returns an Amazon S3 compatible client object. API compatibility (v2 or v4) is automatically
	// determined based on the Endpoint value.
	s3Client, err := minio.New("s3.amazonaws.com", "YOUR-ACCESSKEYID", "YOUR-SECRETACCESSKEY", true)
	if err != nil {
		log.Fatalln(err)
	}
	// s3Client.TraceOn(os.Stderr)
	// Get bucket replication configuration from S3
	replicationCfg, err := s3Client.GetBucketReplication(context.Background(), "my-bucketname")
	if err != nil {
		log.Fatalln(err)
	}
	// Create replication config file
	localReplicationCfgFile, err := os.Create("replication.xml")
	if err != nil {
		log.Fatalln(err)
	}
	defer localReplicationCfgFile.Close()

	replBytes, err := json.Marshal(replicationCfg)
	if err != nil {
		log.Fatalln(err)
	}
	localReplicationCfgFile.Write(replBytes)
}
