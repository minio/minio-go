/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2015-2026 MinIO, Inc.
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

import "testing"

func TestGetS3ExpressEndpoint(t *testing.T) {
	tests := []struct {
		region     string
		bucketName string
		dualstack  bool
		want       string
	}{
		// No bucket: regional (control) endpoint.
		{"us-east-1", "", false, "s3express-control.us-east-1.amazonaws.com"},
		// Non S3 Express bucket: regional endpoint.
		{"us-east-1", "regular-bucket", false, "s3express-control.us-east-1.amazonaws.com"},
		// S3 Express buckets: the AZ encoded in the bucket name selects
		// the zonal endpoint.
		{"us-east-1", "mybucket--use1-az4--x-s3", false, "s3express-use1-az4.us-east-1.amazonaws.com"},
		{"us-east-1", "mybucket--use1-az5--x-s3", false, "s3express-use1-az5.us-east-1.amazonaws.com"},
		{"eu-west-1", "mybucket--euw1-az3--x-s3", false, "s3express-euw1-az3.eu-west-1.amazonaws.com"},
		{"eu-north-1", "mybucket--eun1-az2--x-s3", false, "s3express-eun1-az2.eu-north-1.amazonaws.com"},
		// Frankfurt (eu-central-1): regional and zonal endpoints.
		{"eu-central-1", "", false, "s3express-control.eu-central-1.amazonaws.com"},
		{"eu-central-1", "mybucket--euc1-az2--x-s3", false, "s3express-euc1-az2.eu-central-1.amazonaws.com"},
		// Local Zone: the full zone ID (usw2-lax1-az1) selects the zonal
		// endpoint of the parent region.
		{"us-west-2", "bucket--usw2-lax1-az1--x-s3", false, "s3express-usw2-lax1-az1.us-west-2.amazonaws.com"},
		// AZ numbers are not capped at 6 (AWS has since added az7+).
		{"us-west-2", "mybucket--usw2-az7--x-s3", false, "s3express-usw2-az7.us-west-2.amazonaws.com"},
		// "--" runs in the base name must not confuse zone ID extraction:
		// the last "-azN" segment wins.
		{"us-east-1", "mybucket--foo-az1--use1-az4--x-s3", false, "s3express-use1-az4.us-east-1.amazonaws.com"},
		// AZ not in the region's map: the endpoint is constructed from the
		// regular "s3express-<az-id>.<region>.amazonaws.com" pattern.
		{"us-east-1", "mybucket--use1-az6--x-s3", false, "s3express-use1-az6.us-east-1.amazonaws.com"},
		{"us-east-1", "mybucket--use2-az1--x-s3", false, "s3express-use2-az1.us-east-1.amazonaws.com"},
		// Dualstack variants, per
		// https://docs.aws.amazon.com/AmazonS3/latest/userguide/endpoint-directory-buckets-AZ.html
		{"us-east-1", "", true, "s3express-control-dualstack.us-east-1.amazonaws.com"},
		{"us-east-1", "mybucket--use1-az4--x-s3", true, "s3express-use1-az4.dualstack.us-east-1.amazonaws.com"},
		{"us-west-2", "bucket--usw2-lax1-az1--x-s3", true, "s3express-usw2-lax1-az1.dualstack.us-west-2.amazonaws.com"},
		// Region not in the map: unknown, so no endpoint.
		{"unknown-region", "mybucket--use1-az4--x-s3", false, ""},
		{"unknown-region", "regular-bucket", false, ""},
		{"unknown-region", "mybucket--use1-az4--x-s3", true, ""},
	}
	for _, tt := range tests {
		if got := getS3ExpressEndpoint(tt.region, tt.bucketName, tt.dualstack); got != tt.want {
			t.Errorf("getS3ExpressEndpoint(%q, %q, %v) = %q, want %q", tt.region, tt.bucketName, tt.dualstack, got, tt.want)
		}
	}
}
