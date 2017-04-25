/*
 * Minio Go Library for Amazon S3 Compatible Cloud Storage (C) 2017 Minio, Inc.
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
	"math/rand"
	"os"
	"reflect"
	"testing"
	"time"
)

// Tests get bucket policy core API.
func TestGetBucketPolicy(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping functional tests for short runs")
	}

	// Seed random based on current time.
	rand.Seed(time.Now().Unix())

	// Instantiate new minio client object.
	c, err := NewCore(
		os.Getenv("S3_ADDRESS"),
		os.Getenv("ACCESS_KEY"),
		os.Getenv("SECRET_KEY"),
		mustParseBool(os.Getenv("S3_SECURE")),
	)
	if err != nil {
		t.Fatal("Error:", err)
	}

	// Enable to debug
	// c.TraceOn(os.Stderr)

	// Set user agent.
	c.SetAppInfo("Minio-go-FunctionalTest", "0.1.0")

	// Generate a new random bucket name.
	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "minio-go-test")

	// Make a new bucket.
	err = c.MakeBucket(bucketName, "us-east-1")
	if err != nil {
		t.Fatal("Error:", err, bucketName)
	}

	// Verify if bucket exits and you have access.
	var exists bool
	exists, err = c.BucketExists(bucketName)
	if err != nil {
		t.Fatal("Error:", err, bucketName)
	}
	if !exists {
		t.Fatal("Error: could not find ", bucketName)
	}

	// Asserting the default bucket policy.
	bucketPolicy, err := c.GetBucketPolicy(bucketName)
	if err != nil {
		errResp := ToErrorResponse(err)
		if errResp.Code != "NoSuchBucketPolicy" {
			t.Error("Error:", err, bucketName)
		}
	}
	if !reflect.DeepEqual(bucketPolicy, emptyBucketAccessPolicy) {
		t.Errorf("Bucket policy expected %#v, got %#v", emptyBucketAccessPolicy, bucketPolicy)
	}

	err = c.RemoveBucket(bucketName)
	if err != nil {
		t.Fatal("Error:", err)
	}
}
