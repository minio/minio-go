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
	"encoding/xml"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/replication"
)

func main() {
	// Note: YOUR-ACCESSKEYID, YOUR-SECRETACCESSKEY and my-bucketname are
	// dummy values, please replace them with original values.

	// Requests are always secure (HTTPS) by default. Set secure=false to enable insecure (HTTP) access.
	// This boolean value is the last argument for New().

	// New returns an Amazon S3 compatible client object. API compatibility (v2 or v4) is automatically
	// determined based on the Endpoint value.
	s3Client, err := minio.New("s3.amazonaws.com", &minio.Options{
		Creds:  credentials.NewStaticV4("YOUR-ACCESSKEYID", "YOUR-SECRETACCESSKEY", ""),
		Secure: true,
	})
	if err != nil {
		log.Fatalln(err)
	}

	// s3Client.TraceOn(os.Stderr)

	replicationStr := `<ReplicationConfiguration><Rule><ID>string</ID><Status>Enabled</Status><Priority>1</Priority><DeleteMarkerReplication><Status>Disabled</Status></DeleteMarkerReplication><Destination><Bucket>arn:aws:s3:::dest</Bucket></Destination><Filter><And><Prefix>Prefix</Prefix><Tag><Key>Tag-Key1</Key><Value>Tag-Value1</Value></Tag><Tag><Key>Tag-Key2</Key><Value>Tag-Value2</Value></Tag></And></Filter></Rule></ReplicationConfiguration>`
	var replCfg replication.Config
	err = xml.Unmarshal([]byte(replicationStr), &replCfg)
	if err != nil {
		log.Fatalln(err)
	}

	// This replication ARN should have been generated for replication endpoint using `mc admin bucket remote` command
	replCfg.Role = "arn:minio:replica::dadddae7-f1d7-440f-b5d6-651aa9a8c8a7:dest"
	// Set replication config on a bucket
	err = s3Client.SetBucketReplication(context.Background(), "my-bucketname", replCfg)
	if err != nil {
		log.Fatalln(err)
	}
}
