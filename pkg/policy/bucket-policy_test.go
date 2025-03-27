/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2015-2017 MinIO, Inc.
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

package policy

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/minio/minio-go/v7/pkg/set"
)

// TestUnmarshalBucketPolicy tests unmarsheling various examples
// of bucket policies, to verify the correctness of BucketAccessPolicy
// struct defined in this package.
func TestUnmarshalBucketPolicy(t *testing.T) {
	testCases := []struct {
		policyData    string
		shouldSucceed bool
	}{
		// Test 1
		{policyData: `{
  "Version":"2012-10-17",
  "Statement":[
    {
      "Sid":"AddCannedAcl",
      "Effect":"Allow",
      "Principal": {"AWS": ["arn:aws:iam::111122223333:root","arn:aws:iam::444455556666:root"]},
      "Action":["s3:PutObject","s3:PutObjectAcl"],
      "Resource":["arn:aws:s3:::examplebucket/*"],
      "Condition":{"StringEquals":{"s3:x-amz-acl":["public-read"]}}
    }
  ]
}`, shouldSucceed: true},
		// Test 2
		{policyData: `{
  "Version":"2012-10-17",
  "Statement":[
    {
      "Sid":"AddPerm",
      "Effect":"Allow",
      "Principal": "*",
      "Action":["s3:GetObject"],
      "Resource":["arn:aws:s3:::examplebucket/*"]
    }
  ]
}`, shouldSucceed: true},
		// Test 3
		{policyData: `{
  "Version": "2012-10-17",
  "Id": "S3PolicyId1",
  "Statement": [
    {
      "Sid": "IPAllow",
      "Effect": "Allow",
      "Principal": "*",
      "Action": "s3:*",
      "Resource": "arn:aws:s3:::examplebucket/*",
      "Condition": {
         "IpAddress": {"aws:SourceIp": "54.240.143.0/24"},
         "NotIpAddress": {"aws:SourceIp": "54.240.143.188/32"}
      }
    }
  ]
}`, shouldSucceed: true},
		// Test 4
		{policyData: `{
  "Id":"PolicyId2",
  "Version":"2012-10-17",
  "Statement":[
    {
      "Sid":"AllowIPmix",
      "Effect":"Allow",
      "Principal":"*",
      "Action":"s3:*",
      "Resource":"arn:aws:s3:::examplebucket/*",
      "Condition": {
        "IpAddress": {
          "aws:SourceIp": [
            "54.240.143.0/24",
            "2001:DB8:1234:5678::/64"
          ]
        },
        "NotIpAddress": {
          "aws:SourceIp": [
             "54.240.143.128/30",
             "2001:DB8:1234:5678:ABCD::/80"
          ]
        }
      }
    }
  ]
}`, shouldSucceed: true},
		// Test 5
		{policyData: `{
  "Version":"2012-10-17",
  "Id":"http referer policy example",
  "Statement":[
    {
      "Sid":"Allow get requests originating from www.example.com and example.com.",
      "Effect":"Allow",
      "Principal":"*",
      "Action":"s3:GetObject",
      "Resource":"arn:aws:s3:::examplebucket/*",
      "Condition":{
        "StringLike":{"aws:Referer":["http://www.example.com/*","http://example.com/*"]}
      }
    }
  ]
}`, shouldSucceed: true},
		// Test 6
		{policyData: `{
   "Version": "2012-10-17",
   "Id": "http referer policy example",
   "Statement": [
     {
       "Sid": "Allow get requests referred by www.example.com and example.com.",
       "Effect": "Allow",
       "Principal": "*",
       "Action": "s3:GetObject",
       "Resource": "arn:aws:s3:::examplebucket/*",
       "Condition": {
         "StringLike": {"aws:Referer": ["http://www.example.com/*","http://example.com/*"]}
       }
     },
      {
        "Sid": "Explicit deny to ensure requests are allowed only from specific referer.",
        "Effect": "Deny",
        "Principal": "*",
        "Action": "s3:*",
        "Resource": "arn:aws:s3:::examplebucket/*",
        "Condition": {
          "StringNotLike": {"aws:Referer": ["http://www.example.com/*","http://example.com/*"]}
        }
      }
   ]
}`, shouldSucceed: true},

		// Test 7
		{policyData: `{
   "Version":"2012-10-17",
   "Id":"PolicyForCloudFrontPrivateContent",
   "Statement":[
     {
       "Sid":" Grant a CloudFront Origin Identity access to support private content",
       "Effect":"Allow",
       "Principal":{"CanonicalUser":"79a59df900b949e55d96a1e698fbacedfd6e09d98eacf8f8d5218e7cd47ef2be"},
       "Action":"s3:GetObject",
       "Resource":"arn:aws:s3:::example-bucket/*"
     }
   ]
}`, shouldSucceed: true},
		// Test 8
		{policyData: `{
   "Version":"2012-10-17",
   "Statement":[
     {
       "Sid":"111",
       "Effect":"Allow",
       "Principal":{"AWS":"1111111111"},
       "Action":"s3:PutObject",
       "Resource":"arn:aws:s3:::examplebucket/*"
     },
     {
       "Sid":"112",
       "Effect":"Deny",
       "Principal":{"AWS":"1111111111" },
       "Action":"s3:PutObject",
       "Resource":"arn:aws:s3:::examplebucket/*",
       "Condition": {
         "StringNotEquals": {"s3:x-amz-grant-full-control":["emailAddress=xyz@amazon.com"]}
       }
     }
   ]
}`, shouldSucceed: true},
		// Test 9
		{policyData: `{
  "Version":"2012-10-17",
  "Statement":[
    {
      "Sid":"InventoryAndAnalyticsExamplePolicy",
      "Effect":"Allow",
      "Principal": {"Service": "s3.amazonaws.com"},
      "Action":["s3:PutObject"],
      "Resource":["arn:aws:s3:::destination-bucket/*"],
      "Condition": {
          "ArnLike": {
              "aws:SourceArn": "arn:aws:s3:::source-bucket"
           },
         "StringEquals": {
             "aws:SourceAccount": "1234567890",
             "s3:x-amz-acl": "bucket-owner-full-control"
          }
       }
    }
  ]
}`, shouldSucceed: true},
		// Test 10
		{policyData: `{
	"Version": "2012-10-17",
	"Statement": [{
		"Effect": "Deny",
		"Principal": {
			"AWS": [
				"*"
			]
		},
		"Action": [
			"s3:PutObject"
		],
		"Resource": [
			"arn:aws:s3:::mytest/*"
		],
		"Condition": {
			"Null": {
				"s3:x-amz-server-side-encryption": [
					true
				]
			}
		}
	}]
}`, shouldSucceed: true},
		// Test 11
		{policyData: `{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Deny",
            "Principal": "*",
            "Action": "s3:PutObject",
            "Resource": [
                "arn:aws:s3:::DOC-EXAMPLE-BUCKET1",
                "arn:aws:s3:::DOC-EXAMPLE-BUCKET1/*"
            ],
            "Condition": {
                "NumericLessThan": {
                    "s3:TlsVersion": 1.2
                }
            }
        }
    ]
}`, shouldSucceed: true},
	}

	for i, testCase := range testCases {
		var policy BucketAccessPolicy
		err := json.Unmarshal([]byte(testCase.policyData), &policy)
		if testCase.shouldSucceed && err != nil {
			t.Fatalf("Test %d: expected to succeed but it has an error: %v", i+1, err)
		}
		if !testCase.shouldSucceed && err == nil {
			t.Fatalf("Test %d: expected to fail but succeeded", i+1)
		}
	}
}

// isValidStatement() is called and the result is validated.
func TestIsValidStatement(t *testing.T) {
	testCases := []struct {
		statement      Statement
		bucketName     string
		expectedResult bool
	}{
		// Empty statement and bucket name.
		{Statement{}, "", false},
		// Empty statement.
		{Statement{}, "mybucket", false},
		// Empty bucket name.
		{Statement{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, "", false},
		// Statement with unknown actions.
		{Statement{
			Actions:   set.CreateStringSet("s3:ListBucketVersions"),
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, "mybucket", false},
		// Statement with unknown effect.
		{Statement{
			Actions:   readOnlyBucketActions,
			Effect:    "Deny",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, "mybucket", false},
		// Statement with nil Principal.AWS.
		{Statement{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, "mybucket", false},
		// Statement with unknown Principal.AWS.
		{Statement{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("arn:aws:iam::AccountNumberWithoutHyphens:root")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, "mybucket", false},
		// Statement with different bucket name.
		{Statement{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::testbucket"),
		}, "mybucket", false},
		// Statement with bucket name with suffixed string.
		{Statement{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybuckettest/myobject"),
		}, "mybucket", false},
		// Statement with bucket name and object name.
		{Statement{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/myobject"),
		}, "mybucket", true},
		// Statement with condition, bucket name and object name.
		{Statement{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket/myobject"),
		}, "mybucket", true},
	}

	for _, testCase := range testCases {
		if result := isValidStatement(testCase.statement, testCase.bucketName); result != testCase.expectedResult {
			t.Fatalf("%+v: expected: %t, got: %t", testCase, testCase.expectedResult, result)
		}
	}
}

// newStatements() is called and the result is validated.
func TestNewStatements(t *testing.T) {
	testCases := []struct {
		policy         BucketPolicy
		bucketName     string
		prefix         string
		expectedResult string
	}{
		// BucketPolicyNone: with empty bucket name and prefix.
		{BucketPolicyNone, "", "", `[]`},
		// BucketPolicyNone: with bucket name and empty prefix.
		{BucketPolicyNone, "mybucket", "", `[]`},
		// BucketPolicyNone: with empty bucket name empty prefix.
		{BucketPolicyNone, "", "hello", `[]`},
		// BucketPolicyNone: with bucket name prefix.
		{BucketPolicyNone, "mybucket", "hello", `[]`},
		// BucketPolicyReadOnly: with empty bucket name and prefix.
		{BucketPolicyReadOnly, "", "", `[]`},
		// BucketPolicyReadOnly: with bucket name and empty prefix.
		{BucketPolicyReadOnly, "mybucket", "", `[{"Action":["s3:GetBucketLocation"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:ListBucket"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:GetObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/*"],"Sid":""}]`},
		// BucketPolicyReadOnly: with empty bucket name empty prefix.
		{BucketPolicyReadOnly, "", "hello", `[]`},
		// BucketPolicyReadOnly: with bucket name prefix.
		{BucketPolicyReadOnly, "mybucket", "hello", `[{"Action":["s3:GetBucketLocation"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:ListBucket"],"Condition":{"StringEquals":{"s3:prefix":["hello"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:GetObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/hello*"],"Sid":""}]`},
		// BucketPolicyReadWrite: with empty bucket name and prefix.
		{BucketPolicyReadWrite, "", "", `[]`},
		// BucketPolicyReadWrite: with bucket name and empty prefix.
		{BucketPolicyReadWrite, "mybucket", "", `[{"Action":["s3:GetBucketLocation"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:ListBucket"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:ListBucketMultipartUploads"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:AbortMultipartUpload","s3:DeleteObject","s3:GetObject","s3:ListMultipartUploadParts","s3:PutObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/*"],"Sid":""}]`},
		// BucketPolicyReadWrite: with empty bucket name empty prefix.
		{BucketPolicyReadWrite, "", "hello", `[]`},
		// BucketPolicyReadWrite: with bucket name prefix.
		{BucketPolicyReadWrite, "mybucket", "hello", `[{"Action":["s3:GetBucketLocation"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:ListBucket"],"Condition":{"StringEquals":{"s3:prefix":["hello"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:ListBucketMultipartUploads"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:AbortMultipartUpload","s3:DeleteObject","s3:GetObject","s3:ListMultipartUploadParts","s3:PutObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/hello*"],"Sid":""}]`},
		// BucketPolicyWriteOnly: with empty bucket name and prefix.
		{BucketPolicyWriteOnly, "", "", `[]`},
		// BucketPolicyWriteOnly: with bucket name and empty prefix.
		{BucketPolicyWriteOnly, "mybucket", "", `[{"Action":["s3:GetBucketLocation"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:ListBucketMultipartUploads"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:AbortMultipartUpload","s3:DeleteObject","s3:ListMultipartUploadParts","s3:PutObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/*"],"Sid":""}]`},
		// BucketPolicyWriteOnly: with empty bucket name empty prefix.
		{BucketPolicyWriteOnly, "", "hello", `[]`},
		// BucketPolicyWriteOnly: with bucket name prefix.
		{BucketPolicyWriteOnly, "mybucket", "hello", `[{"Action":["s3:GetBucketLocation"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:ListBucketMultipartUploads"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:AbortMultipartUpload","s3:DeleteObject","s3:ListMultipartUploadParts","s3:PutObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/hello*"],"Sid":""}]`},
	}

	for _, testCase := range testCases {
		statements := newStatements(testCase.policy, testCase.bucketName, testCase.prefix)
		if data, err := json.Marshal(statements); err == nil {
			if string(data) != testCase.expectedResult {
				t.Fatalf("%+v: expected: %s, got: %s", testCase, testCase.expectedResult, string(data))
			}
		}
	}
}

// getInUsePolicy() is called and the result is validated.
func TestGetInUsePolicy(t *testing.T) {
	testCases := []struct {
		statements      []Statement
		bucketName      string
		prefix          string
		expectedResult1 bool
		expectedResult2 bool
	}{
		// All empty statements, bucket name and prefix.
		{[]Statement{}, "", "", false, false},
		// Non-empty statements, empty bucket name and empty prefix.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "", "", false, false},
		// Non-empty statements, non-empty bucket name and empty prefix.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "mybucket", "", false, false},
		// Non-empty statements, empty bucket name and non-empty prefix.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "", "hello", false, false},
		// Empty statements, non-empty bucket name and empty prefix.
		{[]Statement{}, "mybucket", "", false, false},
		// Empty statements, non-empty bucket name non-empty prefix.
		{[]Statement{}, "mybucket", "hello", false, false},
		// Empty statements, empty bucket name and non-empty prefix.
		{[]Statement{}, "", "hello", false, false},
		// Non-empty statements, non-empty bucket name, non-empty prefix.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "mybucket", "hello", false, false},
		// different bucket statements and empty prefix.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::testbucket"),
		}}, "mybucket", "", false, false},
		// different bucket statements.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::testbucket"),
		}}, "mybucket", "hello", false, false},
		// different bucket multi-statements and empty prefix.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::testbucket"),
		}, {
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::testbucket/world"),
		}}, "mybucket", "", false, false},
		// different bucket multi-statements.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::testbucket"),
		}, {
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::testbucket/world"),
		}}, "mybucket", "hello", false, false},
		// read-only in use.
		{[]Statement{{
			Actions:    readOnlyObjectActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket/world"),
		}}, "mybucket", "hello", true, false},
		// write-only in use.
		{[]Statement{{
			Actions:    writeOnlyObjectActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket/world"),
		}}, "mybucket", "hello", false, true},
		// read-write in use.
		{[]Statement{{
			Actions:    readWriteObjectActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket/world"),
		}}, "mybucket", "hello", true, true},
		// read-write multi-statements.
		{[]Statement{{
			Actions:    readOnlyObjectActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket/world"),
		}, {
			Actions:    writeOnlyObjectActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket/ground"),
		}}, "mybucket", "hello", true, true},
	}

	for _, testCase := range testCases {
		result1, result2 := getInUsePolicy(testCase.statements, testCase.bucketName, testCase.prefix)
		if result1 != testCase.expectedResult1 || result2 != testCase.expectedResult2 {
			t.Fatalf("%+v: expected: [%t,%t], got: [%t,%t]", testCase,
				testCase.expectedResult1, testCase.expectedResult2,
				result1, result2)
		}
	}
}

// removeStatements() is called and the result is validated.
func TestRemoveStatements(t *testing.T) {
	unknownCondMap1 := make(ConditionMap)
	unknownCondKeyMap1 := make(ConditionKeyMap)
	unknownCondKeyMap1.Add("s3:prefix", set.CreateStringSet("hello"))
	unknownCondMap1.Add("StringNotEquals", unknownCondKeyMap1)

	unknownCondMap11 := make(ConditionMap)
	unknownCondKeyMap11 := make(ConditionKeyMap)
	unknownCondKeyMap11.Add("s3:prefix", set.CreateStringSet("hello"))
	unknownCondMap11.Add("StringNotEquals", unknownCondKeyMap11)

	unknownCondMap12 := make(ConditionMap)
	unknownCondKeyMap12 := make(ConditionKeyMap)
	unknownCondKeyMap12.Add("s3:prefix", set.CreateStringSet("hello"))
	unknownCondMap12.Add("StringNotEquals", unknownCondKeyMap12)

	knownCondMap1 := make(ConditionMap)
	knownCondKeyMap1 := make(ConditionKeyMap)
	knownCondKeyMap1.Add("s3:prefix", set.CreateStringSet("hello"))
	knownCondMap1.Add("StringEquals", knownCondKeyMap1)

	knownCondMap11 := make(ConditionMap)
	knownCondKeyMap11 := make(ConditionKeyMap)
	knownCondKeyMap11.Add("s3:prefix", set.CreateStringSet("hello"))
	knownCondMap11.Add("StringEquals", knownCondKeyMap11)

	knownCondMap12 := make(ConditionMap)
	knownCondKeyMap12 := make(ConditionKeyMap)
	knownCondKeyMap12.Add("s3:prefix", set.CreateStringSet("hello"))
	knownCondMap12.Add("StringEquals", knownCondKeyMap12)

	knownCondMap13 := make(ConditionMap)
	knownCondKeyMap13 := make(ConditionKeyMap)
	knownCondKeyMap13.Add("s3:prefix", set.CreateStringSet("hello"))
	knownCondMap13.Add("StringEquals", knownCondKeyMap13)

	knownCondMap14 := make(ConditionMap)
	knownCondKeyMap14 := make(ConditionKeyMap)
	knownCondKeyMap14.Add("s3:prefix", set.CreateStringSet("hello"))
	knownCondMap14.Add("StringEquals", knownCondKeyMap14)

	knownCondMap2 := make(ConditionMap)
	knownCondKeyMap2 := make(ConditionKeyMap)
	knownCondKeyMap2.Add("s3:prefix", set.CreateStringSet("hello", "world"))
	knownCondMap2.Add("StringEquals", knownCondKeyMap2)

	testCases := []struct {
		statements     []Statement
		bucketName     string
		prefix         string
		expectedResult string
	}{
		// All empty statements, bucket name and prefix.
		{[]Statement{}, "", "", `[]`},
		// Non-empty statements, empty bucket name and empty prefix.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "", "", `[{"Action":["s3:ListBucket"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""}]`},
		// Non-empty statements, non-empty bucket name and empty prefix.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "mybucket", "", `[{"Action":["s3:ListBucket"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""}]`},
		// Non-empty statements, empty bucket name and non-empty prefix.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "", "hello", `[{"Action":["s3:ListBucket"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""}]`},
		// Empty statements, non-empty bucket name and empty prefix.
		{[]Statement{}, "mybucket", "", `[]`},
		// Empty statements, non-empty bucket name non-empty prefix.
		{[]Statement{}, "mybucket", "hello", `[]`},
		// Empty statements, empty bucket name and non-empty prefix.
		{[]Statement{}, "", "hello", `[]`},
		// Statement with unknown Actions with empty prefix.
		{[]Statement{{
			Actions:   set.CreateStringSet("s3:ListBucketVersions", "s3:ListAllMyBuckets"),
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "mybucket", "", `[{"Action":["s3:ListAllMyBuckets","s3:ListBucketVersions"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""}]`},
		// Statement with unknown Actions.
		{[]Statement{{
			Actions:   set.CreateStringSet("s3:ListBucketVersions", "s3:ListAllMyBuckets"),
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "mybucket", "hello", `[{"Action":["s3:ListAllMyBuckets","s3:ListBucketVersions"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""}]`},
		// Statement with unknown Effect with empty prefix.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Deny",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "mybucket", "", `[{"Action":["s3:ListBucket"],"Effect":"Deny","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""}]`},
		// Statement with unknown Effect.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Deny",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "mybucket", "hello", `[{"Action":["s3:ListBucket"],"Effect":"Deny","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""}]`},
		// Statement with unknown Principal.User.AWS with empty prefix.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("arn:aws:iam::AccountNumberWithoutHyphens:root")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "mybucket", "", `[{"Action":["s3:ListBucket"],"Effect":"Allow","Principal":{"AWS":["arn:aws:iam::AccountNumberWithoutHyphens:root"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""}]`},
		// Statement with unknown Principal.User.AWS.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("arn:aws:iam::AccountNumberWithoutHyphens:root")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "mybucket", "hello", `[{"Action":["s3:ListBucket"],"Effect":"Allow","Principal":{"AWS":["arn:aws:iam::AccountNumberWithoutHyphens:root"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""}]`},
		// Statement with unknown Principal.User.CanonicalUser with empty prefix.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{CanonicalUser: set.CreateStringSet("649262f44b8145cb")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "mybucket", "", `[{"Action":["s3:ListBucket"],"Effect":"Allow","Principal":{"CanonicalUser":["649262f44b8145cb"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""}]`},
		// Statement with unknown Principal.User.CanonicalUser.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{CanonicalUser: set.CreateStringSet("649262f44b8145cb")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "mybucket", "hello", `[{"Action":["s3:ListBucket"],"Effect":"Allow","Principal":{"CanonicalUser":["649262f44b8145cb"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""}]`},
		// Statement with unknown Conditions with empty prefix.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: unknownCondMap1,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "mybucket", "", `[{"Action":["s3:ListBucket"],"Condition":{"StringNotEquals":{"s3:prefix":["hello"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""}]`},
		// Statement with unknown Conditions.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: unknownCondMap1,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "mybucket", "hello", `[{"Action":["s3:ListBucket"],"Condition":{"StringNotEquals":{"s3:prefix":["hello"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""}]`},
		// Statement with unknown Resource and empty prefix.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::testbucket"),
		}}, "mybucket", "", `[{"Action":["s3:ListBucket"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::testbucket"],"Sid":""}]`},
		// Statement with unknown Resource.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::testbucket"),
		}}, "mybucket", "hello", `[{"Action":["s3:ListBucket"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::testbucket"],"Sid":""}]`},
		// Statement with known Actions with empty prefix.
		{[]Statement{{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "mybucket", "", `[]`},
		// Statement with known Actions.
		{[]Statement{{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "mybucket", "hello", `[]`},
		// Statement with known multiple Actions with empty prefix.
		{[]Statement{{
			Actions:   readOnlyBucketActions.Union(writeOnlyBucketActions).Union(commonBucketActions),
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "mybucket", "", `[]`},
		// Statement with known multiple Actions.
		{[]Statement{{
			Actions:   readOnlyBucketActions.Union(writeOnlyBucketActions).Union(commonBucketActions),
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "mybucket", "hello", `[]`},
		// RemoveBucketActions with readOnlyInUse.
		{[]Statement{{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, {
			Actions:   readOnlyObjectActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/world"),
		}}, "mybucket", "", `[{"Action":["s3:ListBucket"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:GetObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/world"],"Sid":""}]`},
		// RemoveBucketActions with prefix, readOnlyInUse.
		{[]Statement{{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, {
			Actions:   readOnlyObjectActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/world"),
		}}, "mybucket", "hello", `[{"Action":["s3:ListBucket"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:GetObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/world"],"Sid":""}]`},
		// RemoveBucketActions with writeOnlyInUse.
		{[]Statement{{
			Actions:   writeOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, {
			Actions:   writeOnlyObjectActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/world"),
		}}, "mybucket", "", `[{"Action":["s3:ListBucketMultipartUploads"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:AbortMultipartUpload","s3:DeleteObject","s3:ListMultipartUploadParts","s3:PutObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/world"],"Sid":""}]`},
		// RemoveBucketActions with prefix, writeOnlyInUse.
		{[]Statement{{
			Actions:   writeOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, {
			Actions:   writeOnlyObjectActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/world"),
		}}, "mybucket", "hello", `[{"Action":["s3:ListBucketMultipartUploads"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:AbortMultipartUpload","s3:DeleteObject","s3:ListMultipartUploadParts","s3:PutObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/world"],"Sid":""}]`},
		// RemoveBucketActions with readOnlyInUse and writeOnlyInUse.
		{[]Statement{{
			Actions:   readOnlyBucketActions.Union(writeOnlyBucketActions),
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, {
			Actions:   readWriteObjectActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/world"),
		}}, "mybucket", "", `[{"Action":["s3:ListBucket","s3:ListBucketMultipartUploads"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:AbortMultipartUpload","s3:DeleteObject","s3:GetObject","s3:ListMultipartUploadParts","s3:PutObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/world"],"Sid":""}]`},
		// RemoveBucketActions with prefix, readOnlyInUse and writeOnlyInUse.
		{[]Statement{{
			Actions:   readOnlyBucketActions.Union(writeOnlyBucketActions),
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, {
			Actions:   readWriteObjectActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/world"),
		}}, "mybucket", "hello", `[{"Action":["s3:ListBucket","s3:ListBucketMultipartUploads"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:AbortMultipartUpload","s3:DeleteObject","s3:GetObject","s3:ListMultipartUploadParts","s3:PutObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/world"],"Sid":""}]`},
		// RemoveBucketActions with known Conditions, readOnlyInUse.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: knownCondMap1,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, {
			Actions:   readOnlyObjectActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/world"),
		}}, "mybucket", "", `[{"Action":["s3:ListBucket"],"Condition":{"StringEquals":{"s3:prefix":["hello"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:GetObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/world"],"Sid":""}]`},
		// RemoveBucketActions with prefix, known Conditions, readOnlyInUse.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: knownCondMap1,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, {
			Actions:   readOnlyObjectActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/world"),
		}}, "mybucket", "hello", `[{"Action":["s3:GetObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/world"],"Sid":""}]`},
		// RemoveBucketActions with prefix, known Conditions contains other object prefix, readOnlyInUse.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: knownCondMap2,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, {
			Actions:   readOnlyObjectActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/world"),
		}}, "mybucket", "hello", `[{"Action":["s3:ListBucket"],"Condition":{"StringEquals":{"s3:prefix":["world"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:GetObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/world"],"Sid":""}]`},
		// RemoveBucketActions with unknown Conditions, readOnlyInUse.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: unknownCondMap1,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, {
			Actions:   readOnlyObjectActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/world"),
		}}, "mybucket", "", `[{"Action":["s3:ListBucket"],"Condition":{"StringNotEquals":{"s3:prefix":["hello"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:GetObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/world"],"Sid":""}]`},
		// RemoveBucketActions with prefix, unknown Conditions, readOnlyInUse.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: unknownCondMap1,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, {
			Actions:   readOnlyObjectActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/world"),
		}}, "mybucket", "hello", `[{"Action":["s3:ListBucket"],"Condition":{"StringNotEquals":{"s3:prefix":["hello"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:GetObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/world"],"Sid":""}]`},
		// RemoveBucketActions with known Conditions, writeOnlyInUse.
		{[]Statement{{
			Actions:    writeOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: knownCondMap11,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, {
			Actions:   writeOnlyObjectActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/world"),
		}}, "mybucket", "", `[{"Action":["s3:ListBucketMultipartUploads"],"Condition":{"StringEquals":{"s3:prefix":["hello"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:AbortMultipartUpload","s3:DeleteObject","s3:ListMultipartUploadParts","s3:PutObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/world"],"Sid":""}]`},
		// RemoveBucketActions with prefix, known Conditions, writeOnlyInUse.
		{[]Statement{{
			Actions:    writeOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: knownCondMap11,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, {
			Actions:   writeOnlyObjectActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/world"),
		}}, "mybucket", "hello", `[{"Action":["s3:ListBucketMultipartUploads"],"Condition":{"StringEquals":{"s3:prefix":["hello"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:AbortMultipartUpload","s3:DeleteObject","s3:ListMultipartUploadParts","s3:PutObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/world"],"Sid":""}]`},
		// RemoveBucketActions with unknown Conditions, writeOnlyInUse.
		{[]Statement{{
			Actions:    writeOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: unknownCondMap11,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, {
			Actions:   writeOnlyObjectActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/world"),
		}}, "mybucket", "", `[{"Action":["s3:ListBucketMultipartUploads"],"Condition":{"StringNotEquals":{"s3:prefix":["hello"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:AbortMultipartUpload","s3:DeleteObject","s3:ListMultipartUploadParts","s3:PutObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/world"],"Sid":""}]`},
		// RemoveBucketActions with prefix, unknown Conditions, writeOnlyInUse.
		{[]Statement{{
			Actions:    writeOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: unknownCondMap11,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, {
			Actions:   writeOnlyObjectActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/world"),
		}}, "mybucket", "hello", `[{"Action":["s3:ListBucketMultipartUploads"],"Condition":{"StringNotEquals":{"s3:prefix":["hello"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:AbortMultipartUpload","s3:DeleteObject","s3:ListMultipartUploadParts","s3:PutObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/world"],"Sid":""}]`},
		// RemoveBucketActions with known Conditions, readOnlyInUse and writeOnlyInUse.
		{[]Statement{{
			Actions:    readOnlyBucketActions.Union(writeOnlyBucketActions),
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: knownCondMap12,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, {
			Actions:   readOnlyObjectActions.Union(writeOnlyObjectActions),
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/world"),
		}}, "mybucket", "", `[{"Action":["s3:ListBucket","s3:ListBucketMultipartUploads"],"Condition":{"StringEquals":{"s3:prefix":["hello"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:AbortMultipartUpload","s3:DeleteObject","s3:GetObject","s3:ListMultipartUploadParts","s3:PutObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/world"],"Sid":""}]`},
		// RemoveBucketActions with prefix, known Conditions, readOnlyInUse and writeOnlyInUse.
		{[]Statement{{
			Actions:    readOnlyBucketActions.Union(writeOnlyBucketActions),
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: knownCondMap12,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, {
			Actions:   readOnlyObjectActions.Union(writeOnlyObjectActions),
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/world"),
		}}, "mybucket", "hello", `[{"Action":["s3:AbortMultipartUpload","s3:DeleteObject","s3:GetObject","s3:ListMultipartUploadParts","s3:PutObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/world"],"Sid":""}]`},
		// RemoveBucketActions with unknown Conditions, readOnlyInUse and writeOnlyInUse.
		{[]Statement{{
			Actions:    readOnlyBucketActions.Union(writeOnlyBucketActions),
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: unknownCondMap12,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, {
			Actions:   readOnlyObjectActions.Union(writeOnlyObjectActions),
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/world"),
		}}, "mybucket", "", `[{"Action":["s3:ListBucket","s3:ListBucketMultipartUploads"],"Condition":{"StringNotEquals":{"s3:prefix":["hello"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:AbortMultipartUpload","s3:DeleteObject","s3:GetObject","s3:ListMultipartUploadParts","s3:PutObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/world"],"Sid":""}]`},
		// RemoveBucketActions with prefix, unknown Conditions, readOnlyInUse and writeOnlyInUse.
		{[]Statement{{
			Actions:    readOnlyBucketActions.Union(writeOnlyBucketActions),
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: unknownCondMap12,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, {
			Actions:   readOnlyObjectActions.Union(writeOnlyObjectActions),
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/world"),
		}}, "mybucket", "hello", `[{"Action":["s3:ListBucket","s3:ListBucketMultipartUploads"],"Condition":{"StringNotEquals":{"s3:prefix":["hello"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:AbortMultipartUpload","s3:DeleteObject","s3:GetObject","s3:ListMultipartUploadParts","s3:PutObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/world"],"Sid":""}]`},
		// readOnlyObjectActions - RemoveObjectActions with known condition.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: knownCondMap1,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, {
			Actions:   readOnlyObjectActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/hello*"),
		}}, "mybucket", "", `[{"Action":["s3:ListBucket"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:GetObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/hello*"],"Sid":""}]`},
		// readOnlyObjectActions - RemoveObjectActions with prefix, known condition.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: knownCondMap1,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, {
			Actions:   readOnlyObjectActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/hello*"),
		}}, "mybucket", "hello", `[]`},
		// readOnlyObjectActions - RemoveObjectActions with unknown condition.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: unknownCondMap1,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, {
			Actions:   readOnlyObjectActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/hello*"),
		}}, "mybucket", "", `[{"Action":["s3:ListBucket"],"Condition":{"StringNotEquals":{"s3:prefix":["hello"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:GetObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/hello*"],"Sid":""}]`},
		// readOnlyObjectActions - RemoveObjectActions with prefix, unknown condition.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: unknownCondMap1,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, {
			Actions:   readOnlyObjectActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/hello*"),
		}}, "mybucket", "hello", `[{"Action":["s3:ListBucket"],"Condition":{"StringNotEquals":{"s3:prefix":["hello"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""}]`},
		// writeOnlyObjectActions - RemoveObjectActions with known condition.
		{[]Statement{{
			Actions:    writeOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: knownCondMap13,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, {
			Actions:   writeOnlyObjectActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/hello*"),
		}}, "mybucket", "", `[{"Action":["s3:ListBucketMultipartUploads"],"Condition":{"StringEquals":{"s3:prefix":["hello"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:AbortMultipartUpload","s3:DeleteObject","s3:ListMultipartUploadParts","s3:PutObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/hello*"],"Sid":""}]`},
		// writeOnlyObjectActions - RemoveObjectActions with prefix, known condition.
		{[]Statement{{
			Actions:    writeOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: knownCondMap13,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, {
			Actions:   writeOnlyObjectActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/hello*"),
		}}, "mybucket", "hello", `[{"Action":["s3:ListBucketMultipartUploads"],"Condition":{"StringEquals":{"s3:prefix":["hello"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""}]`},
		// writeOnlyObjectActions - RemoveObjectActions with unknown condition.
		{[]Statement{{
			Actions:    writeOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: unknownCondMap1,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, {
			Actions:   writeOnlyObjectActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/hello*"),
		}}, "mybucket", "", `[{"Action":["s3:ListBucketMultipartUploads"],"Condition":{"StringNotEquals":{"s3:prefix":["hello"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:AbortMultipartUpload","s3:DeleteObject","s3:ListMultipartUploadParts","s3:PutObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/hello*"],"Sid":""}]`},
		// writeOnlyObjectActions - RemoveObjectActions with prefix, unknown condition.
		{[]Statement{{
			Actions:    writeOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: unknownCondMap1,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, {
			Actions:   writeOnlyObjectActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/hello*"),
		}}, "mybucket", "hello", `[{"Action":["s3:ListBucketMultipartUploads"],"Condition":{"StringNotEquals":{"s3:prefix":["hello"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""}]`},
		// readWriteObjectActions - RemoveObjectActions with known condition.
		{[]Statement{{
			Actions:    readOnlyBucketActions.Union(writeOnlyBucketActions),
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: knownCondMap14,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, {
			Actions:   readOnlyObjectActions.Union(writeOnlyObjectActions),
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/hello*"),
		}}, "mybucket", "", `[{"Action":["s3:ListBucket","s3:ListBucketMultipartUploads"],"Condition":{"StringEquals":{"s3:prefix":["hello"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:AbortMultipartUpload","s3:DeleteObject","s3:GetObject","s3:ListMultipartUploadParts","s3:PutObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/hello*"],"Sid":""}]`},
		// readWriteObjectActions - RemoveObjectActions with prefix, known condition.
		{[]Statement{{
			Actions:    readOnlyBucketActions.Union(writeOnlyBucketActions),
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: knownCondMap13,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, {
			Actions:   readOnlyObjectActions.Union(writeOnlyObjectActions),
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/hello*"),
		}}, "mybucket", "hello", `[]`},
		// readWriteObjectActions - RemoveObjectActions with unknown condition.
		{[]Statement{{
			Actions:    readOnlyBucketActions.Union(writeOnlyBucketActions),
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: unknownCondMap1,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, {
			Actions:   readOnlyObjectActions.Union(writeOnlyObjectActions),
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/hello*"),
		}}, "mybucket", "", `[{"Action":["s3:ListBucket","s3:ListBucketMultipartUploads"],"Condition":{"StringNotEquals":{"s3:prefix":["hello"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:AbortMultipartUpload","s3:DeleteObject","s3:GetObject","s3:ListMultipartUploadParts","s3:PutObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/hello*"],"Sid":""}]`},
		// readWriteObjectActions - RemoveObjectActions with prefix, unknown condition.
		{[]Statement{{
			Actions:    readOnlyBucketActions.Union(writeOnlyBucketActions),
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: unknownCondMap1,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, {
			Actions:   readOnlyObjectActions.Union(writeOnlyObjectActions),
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/hello*"),
		}}, "mybucket", "hello", `[{"Action":["s3:ListBucket","s3:ListBucketMultipartUploads"],"Condition":{"StringNotEquals":{"s3:prefix":["hello"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""}]`},
	}

	for _, testCase := range testCases {
		statements := removeStatements(testCase.statements, testCase.bucketName, testCase.prefix)
		if data, err := json.Marshal(statements); err != nil {
			t.Fatalf("unable encoding to json, %s", err)
		} else if string(data) != testCase.expectedResult {
			t.Fatalf("%+v: expected: %s, got: %s", testCase, testCase.expectedResult, string(data))
		}
	}
}

// appendStatement() is called and the result is validated.
func TestAppendStatement(t *testing.T) {
	condMap := make(ConditionMap)
	condKeyMap := make(ConditionKeyMap)
	condKeyMap.Add("s3:prefix", set.CreateStringSet("hello"))
	condMap.Add("StringEquals", condKeyMap)

	condMap1 := make(ConditionMap)
	condKeyMap1 := make(ConditionKeyMap)
	condKeyMap1.Add("s3:prefix", set.CreateStringSet("world"))
	condMap1.Add("StringEquals", condKeyMap1)

	unknownCondMap1 := make(ConditionMap)
	unknownCondKeyMap1 := make(ConditionKeyMap)
	unknownCondKeyMap1.Add("s3:prefix", set.CreateStringSet("world"))
	unknownCondMap1.Add("StringNotEquals", unknownCondKeyMap1)

	testCases := []struct {
		statements     []Statement
		statement      Statement
		expectedResult string
	}{
		// Empty statements and empty new statement.
		{[]Statement{}, Statement{}, `[]`},
		// Non-empty statements and empty new statement.
		{[]Statement{{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, Statement{}, `[{"Action":["s3:ListBucket"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""}]`},
		// Empty statements and non-empty new statement.
		{[]Statement{}, Statement{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, `[{"Action":["s3:ListBucket"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""}]`},
		// Append existing statement.
		{[]Statement{{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, Statement{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, `[{"Action":["s3:ListBucket"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""}]`},
		// Append same statement with different resource.
		{[]Statement{{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, Statement{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::testbucket"),
		}, `[{"Action":["s3:ListBucket"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket","arn:aws:s3:::testbucket"],"Sid":""}]`},
		// Append same statement with different actions.
		{[]Statement{{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, Statement{
			Actions:   writeOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, `[{"Action":["s3:ListBucket","s3:ListBucketMultipartUploads"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""}]`},
		// Elements of new statement contains elements in statements.
		{[]Statement{{
			Actions:   readOnlyBucketActions.Union(writeOnlyBucketActions),
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket", "arn:aws:s3:::testbucket"),
		}}, Statement{
			Actions:   writeOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, `[{"Action":["s3:ListBucket","s3:ListBucketMultipartUploads"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket","arn:aws:s3:::testbucket"],"Sid":""}]`},
		// Elements of new statement with conditions contains elements in statements.
		{[]Statement{{
			Actions:    readOnlyBucketActions.Union(writeOnlyBucketActions),
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: condMap,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket", "arn:aws:s3:::testbucket"),
		}}, Statement{
			Actions:    writeOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: condMap,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, `[{"Action":["s3:ListBucket","s3:ListBucketMultipartUploads"],"Condition":{"StringEquals":{"s3:prefix":["hello"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket","arn:aws:s3:::testbucket"],"Sid":""}]`},
		// Statements with condition and new statement with condition.
		{[]Statement{{
			Actions:    readOnlyBucketActions.Union(writeOnlyBucketActions),
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: condMap,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket", "arn:aws:s3:::testbucket"),
		}}, Statement{
			Actions:    writeOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: condMap1,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, `[{"Action":["s3:ListBucket","s3:ListBucketMultipartUploads"],"Condition":{"StringEquals":{"s3:prefix":["hello"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket","arn:aws:s3:::testbucket"],"Sid":""},{"Action":["s3:ListBucketMultipartUploads"],"Condition":{"StringEquals":{"s3:prefix":["world"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""}]`},
		// Statements with condition and same resources, and new statement with condition.
		{[]Statement{{
			Actions:    readOnlyBucketActions.Union(writeOnlyBucketActions),
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: condMap,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, Statement{
			Actions:    writeOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: condMap1,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, `[{"Action":["s3:ListBucket","s3:ListBucketMultipartUploads"],"Condition":{"StringEquals":{"s3:prefix":["hello","world"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""}]`},
		// Statements with unknown condition and same resources, and new statement with known condition.
		{[]Statement{{
			Actions:    readOnlyBucketActions.Union(writeOnlyBucketActions),
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: unknownCondMap1,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, Statement{
			Actions:    writeOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: condMap1,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, `[{"Action":["s3:ListBucket","s3:ListBucketMultipartUploads"],"Condition":{"StringEquals":{"s3:prefix":["world"]},"StringNotEquals":{"s3:prefix":["world"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""}]`},
		// Statements without condition and new statement with condition.
		{[]Statement{{
			Actions:   readOnlyBucketActions.Union(writeOnlyBucketActions),
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket", "arn:aws:s3:::testbucket"),
		}}, Statement{
			Actions:    writeOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: condMap,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, `[{"Action":["s3:ListBucket","s3:ListBucketMultipartUploads"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket","arn:aws:s3:::testbucket"],"Sid":""},{"Action":["s3:ListBucketMultipartUploads"],"Condition":{"StringEquals":{"s3:prefix":["hello"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""}]`},
		// Statements with condition and new statement without condition.
		{[]Statement{{
			Actions:    readOnlyBucketActions.Union(writeOnlyBucketActions),
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: condMap,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket", "arn:aws:s3:::testbucket"),
		}}, Statement{
			Actions:   writeOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, `[{"Action":["s3:ListBucket","s3:ListBucketMultipartUploads"],"Condition":{"StringEquals":{"s3:prefix":["hello"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket","arn:aws:s3:::testbucket"],"Sid":""},{"Action":["s3:ListBucketMultipartUploads"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""}]`},
		// Statements and new statement are different.
		{[]Statement{{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, Statement{
			Actions:   readOnlyObjectActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/hello*"),
		}, `[{"Action":["s3:ListBucket"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:GetObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/hello*"],"Sid":""}]`},
	}

	for _, testCase := range testCases {
		statements := appendStatement(testCase.statements, testCase.statement)
		if data, err := json.Marshal(statements); err != nil {
			t.Fatalf("unable encoding to json, %s", err)
		} else if string(data) != testCase.expectedResult {
			t.Fatalf("%+v: expected: %s, got: %s", testCase, testCase.expectedResult, string(data))
		}
	}
}

// getBucketPolicy() is called and the result is validated.
func TestGetBucketPolicy(t *testing.T) {
	helloCondMap := make(ConditionMap)
	helloCondKeyMap := make(ConditionKeyMap)
	helloCondKeyMap.Add("s3:prefix", set.CreateStringSet("hello"))
	helloCondMap.Add("StringEquals", helloCondKeyMap)

	worldCondMap := make(ConditionMap)
	worldCondKeyMap := make(ConditionKeyMap)
	worldCondKeyMap.Add("s3:prefix", set.CreateStringSet("world"))
	worldCondMap.Add("StringEquals", worldCondKeyMap)

	notHelloCondMap := make(ConditionMap)
	notHelloCondMap.Add("StringNotEquals", worldCondKeyMap)

	testCases := []struct {
		statement       Statement
		prefix          string
		expectedResult1 bool
		expectedResult2 bool
		expectedResult3 bool
	}{
		// Statement with invalid Effect.
		{Statement{
			Actions:   readOnlyBucketActions,
			Effect:    "Deny",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, "", false, false, false},
		// Statement with invalid Effect with prefix.
		{Statement{
			Actions:   readOnlyBucketActions,
			Effect:    "Deny",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, "hello", false, false, false},
		// Statement with invalid Principal.AWS.
		{Statement{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("arn:aws:iam::AccountNumberWithoutHyphens:root")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, "", false, false, false},
		// Statement with invalid Principal.AWS with prefix.
		{Statement{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("arn:aws:iam::AccountNumberWithoutHyphens:root")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, "hello", false, false, false},

		// Statement with commonBucketActions.
		{Statement{
			Actions:   commonBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, "", true, false, false},
		// Statement with commonBucketActions.
		{Statement{
			Actions:   commonBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, "hello", true, false, false},

		// Statement with commonBucketActions and condition.
		{Statement{
			Actions:    commonBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, "", false, false, false},
		// Statement with commonBucketActions and condition.
		{Statement{
			Actions:    commonBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, "hello", false, false, false},
		// Statement with writeOnlyBucketActions.
		{Statement{
			Actions:   writeOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, "", false, false, true},
		// Statement with writeOnlyBucketActions.
		{Statement{
			Actions:   writeOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, "hello", false, false, true},
		// Statement with writeOnlyBucketActions and condition
		{Statement{
			Actions:    writeOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, "", false, false, false},
		// Statement with writeOnlyBucketActions and condition.
		{Statement{
			Actions:    writeOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, "hello", false, false, false},
		// Statement with readOnlyBucketActions.
		{Statement{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, "", false, true, false},
		// Statement with readOnlyBucketActions.
		{Statement{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, "hello", false, true, false},
		// Statement with readOnlyBucketActions with empty condition.
		{Statement{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, "", false, false, false},
		// Statement with readOnlyBucketActions with empty condition.
		{Statement{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, "hello", false, false, false},
		// Statement with readOnlyBucketActions with matching condition.
		{Statement{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: helloCondMap,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, "", false, false, false},
		// Statement with readOnlyBucketActions with matching condition.
		{Statement{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: helloCondMap,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, "hello", false, true, false},

		// Statement with readOnlyBucketActions with different condition.
		{Statement{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: worldCondMap,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, "", false, false, false},
		// Statement with readOnlyBucketActions with different condition.
		{Statement{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: worldCondMap,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, "hello", false, false, false},

		// Statement with readOnlyBucketActions with StringNotEquals condition.
		{Statement{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: notHelloCondMap,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, "", false, false, false},
		// Statement with readOnlyBucketActions with StringNotEquals condition.
		{Statement{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: notHelloCondMap,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}, "hello", false, true, false},
	}

	for _, testCase := range testCases {
		commonFound, readOnly, writeOnly := getBucketPolicy(testCase.statement, testCase.prefix)
		if testCase.expectedResult1 != commonFound || testCase.expectedResult2 != readOnly || testCase.expectedResult3 != writeOnly {
			t.Fatalf("%+v: expected: [%t,%t,%t], got: [%t,%t,%t]", testCase,
				testCase.expectedResult1, testCase.expectedResult2, testCase.expectedResult3,
				commonFound, readOnly, writeOnly)
		}
	}
}

// getObjectPolicy() is called and the result is validated.
func TestGetObjectPolicy(t *testing.T) {
	testCases := []struct {
		statement       Statement
		expectedResult1 bool
		expectedResult2 bool
	}{
		// Statement with invalid Effect.
		{Statement{
			Actions:   readOnlyObjectActions,
			Effect:    "Deny",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/hello*"),
		}, false, false},
		// Statement with invalid Principal.AWS.
		{Statement{
			Actions:   readOnlyObjectActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("arn:aws:iam::AccountNumberWithoutHyphens:root")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/hello*"),
		}, false, false},
		// Statement with condition.
		{Statement{
			Actions:    readOnlyObjectActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: make(ConditionMap),
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket/hello*"),
		}, false, false},
		// Statement with readOnlyObjectActions.
		{Statement{
			Actions:   readOnlyObjectActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/hello*"),
		}, true, false},
		// Statement with writeOnlyObjectActions.
		{Statement{
			Actions:   writeOnlyObjectActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/hello*"),
		}, false, true},
		// Statement with readOnlyObjectActions and writeOnlyObjectActions.
		{Statement{
			Actions:   readOnlyObjectActions.Union(writeOnlyObjectActions),
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket/hello*"),
		}, true, true},
	}

	for _, testCase := range testCases {
		readOnly, writeOnly := getObjectPolicy(testCase.statement)
		if testCase.expectedResult1 != readOnly || testCase.expectedResult2 != writeOnly {
			t.Fatalf("%+v: expected: [%t,%t], got: [%t,%t]", testCase,
				testCase.expectedResult1, testCase.expectedResult2,
				readOnly, writeOnly)
		}
	}
}

// GetPolicyRules is called and the result is validated
func TestListBucketPolicies(t *testing.T) {
	// Condition for read objects
	downloadCondMap := make(ConditionMap)
	downloadCondKeyMap := make(ConditionKeyMap)
	downloadCondKeyMap.Add("s3:prefix", set.CreateStringSet("download"))
	downloadCondMap.Add("StringEquals", downloadCondKeyMap)

	// Condition for readwrite objects
	downloadUploadCondMap := make(ConditionMap)
	downloadUploadCondKeyMap := make(ConditionKeyMap)
	downloadUploadCondKeyMap.Add("s3:prefix", set.CreateStringSet("both"))
	downloadUploadCondMap.Add("StringEquals", downloadUploadCondKeyMap)

	commonSetActions := commonBucketActions.Union(readOnlyBucketActions)
	testCases := []struct {
		statements     []Statement
		bucketName     string
		prefix         string
		expectedResult map[string]BucketPolicy
	}{
		// Empty statements, bucket name and prefix.
		{[]Statement{}, "", "", map[string]BucketPolicy{}},
		// Non-empty statements, empty bucket name and empty prefix.
		{[]Statement{{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "", "", map[string]BucketPolicy{}},
		// Empty statements, non-empty bucket name and empty prefix.
		{[]Statement{}, "mybucket", "", map[string]BucketPolicy{}},
		// Readonly object statement
		{[]Statement{
			{
				Actions:   commonBucketActions,
				Effect:    "Allow",
				Principal: User{AWS: set.CreateStringSet("*")},
				Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
			},
			{
				Actions:    readOnlyBucketActions,
				Effect:     "Allow",
				Principal:  User{AWS: set.CreateStringSet("*")},
				Conditions: downloadCondMap,
				Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
			},
			{
				Actions:   readOnlyObjectActions,
				Effect:    "Allow",
				Principal: User{AWS: set.CreateStringSet("*")},
				Resources: set.CreateStringSet("arn:aws:s3:::mybucket/download*"),
			},
		}, "mybucket", "", map[string]BucketPolicy{"mybucket/download*": BucketPolicyReadOnly}},
		{[]Statement{
			{
				Actions:   commonSetActions.Union(readOnlyObjectActions),
				Effect:    "Allow",
				Principal: User{AWS: set.CreateStringSet("*")},
				Resources: set.CreateStringSet("arn:aws:s3:::mybucket", "arn:aws:s3:::mybucket/*"),
			},
		}, "mybucket", "", map[string]BucketPolicy{"mybucket/*": BucketPolicyReadOnly}},
		// Write Only
		{[]Statement{
			{
				Actions:   commonBucketActions.Union(writeOnlyBucketActions),
				Effect:    "Allow",
				Principal: User{AWS: set.CreateStringSet("*")},
				Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
			},
			{
				Actions:   writeOnlyObjectActions,
				Effect:    "Allow",
				Principal: User{AWS: set.CreateStringSet("*")},
				Resources: set.CreateStringSet("arn:aws:s3:::mybucket/upload*"),
			},
		}, "mybucket", "", map[string]BucketPolicy{"mybucket/upload*": BucketPolicyWriteOnly}},
		// Readwrite
		{[]Statement{
			{
				Actions:   commonBucketActions.Union(writeOnlyBucketActions),
				Effect:    "Allow",
				Principal: User{AWS: set.CreateStringSet("*")},
				Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
			},
			{
				Actions:    readOnlyBucketActions,
				Effect:     "Allow",
				Principal:  User{AWS: set.CreateStringSet("*")},
				Conditions: downloadUploadCondMap,
				Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
			},
			{
				Actions:   writeOnlyObjectActions.Union(readOnlyObjectActions),
				Effect:    "Allow",
				Principal: User{AWS: set.CreateStringSet("*")},
				Resources: set.CreateStringSet("arn:aws:s3:::mybucket/both*"),
			},
		}, "mybucket", "", map[string]BucketPolicy{"mybucket/both*": BucketPolicyReadWrite}},
	}

	for _, testCase := range testCases {
		policyRules := GetPolicies(testCase.statements, testCase.bucketName, "")
		if !reflect.DeepEqual(testCase.expectedResult, policyRules) {
			t.Fatalf("%+v:\n expected: %+v, got: %+v", testCase, testCase.expectedResult, policyRules)
		}
	}
}

// GetPolicy() is called and the result is validated.
func TestGetPolicy(t *testing.T) {
	helloCondMap := make(ConditionMap)
	helloCondKeyMap := make(ConditionKeyMap)
	helloCondKeyMap.Add("s3:prefix", set.CreateStringSet("hello"))
	helloCondMap.Add("StringEquals", helloCondKeyMap)

	testCases := []struct {
		statements     []Statement
		bucketName     string
		prefix         string
		expectedResult BucketPolicy
	}{
		// Empty statements, bucket name and prefix.
		{[]Statement{}, "", "", BucketPolicyNone},
		// Non-empty statements, empty bucket name and empty prefix.
		{[]Statement{{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "", "", BucketPolicyNone},
		// Empty statements, non-empty bucket name and empty prefix.
		{[]Statement{}, "mybucket", "", BucketPolicyNone},
		// not-matching Statements.
		{[]Statement{{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::testbucket"),
		}}, "mybucket", "", BucketPolicyNone},
		// not-matching Statements with prefix.
		{[]Statement{{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::testbucket"),
		}}, "mybucket", "hello", BucketPolicyNone},
		// Statements with only commonBucketActions.
		{[]Statement{{
			Actions:   commonBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "mybucket", "", BucketPolicyNone},
		// Statements with only commonBucketActions with prefix.
		{[]Statement{{
			Actions:   commonBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "mybucket", "hello", BucketPolicyNone},
		// Statements with only readOnlyBucketActions.
		{[]Statement{{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "mybucket", "", BucketPolicyNone},
		// Statements with only readOnlyBucketActions with prefix.
		{[]Statement{{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "mybucket", "hello", BucketPolicyNone},
		// Statements with only readOnlyBucketActions with conditions.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: helloCondMap,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "mybucket", "", BucketPolicyNone},
		// Statements with only readOnlyBucketActions with prefix with conditons.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: helloCondMap,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "mybucket", "hello", BucketPolicyNone},
		// Statements with only writeOnlyBucketActions.
		{[]Statement{{
			Actions:   writeOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "mybucket", "", BucketPolicyNone},
		// Statements with only writeOnlyBucketActions with prefix.
		{[]Statement{{
			Actions:   writeOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "mybucket", "hello", BucketPolicyNone},
		// Statements with only readOnlyBucketActions + writeOnlyBucketActions.
		{[]Statement{{
			Actions:   readOnlyBucketActions.Union(writeOnlyBucketActions),
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "mybucket", "", BucketPolicyNone},
		// Statements with only readOnlyBucketActions + writeOnlyBucketActions with prefix.
		{[]Statement{{
			Actions:   readOnlyBucketActions.Union(writeOnlyBucketActions),
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "mybucket", "hello", BucketPolicyNone},
		// Statements with only readOnlyBucketActions + writeOnlyBucketActions and conditions.
		{[]Statement{{
			Actions:    readOnlyBucketActions.Union(writeOnlyBucketActions),
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: helloCondMap,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "mybucket", "", BucketPolicyNone},
		// Statements with only readOnlyBucketActions + writeOnlyBucketActions and conditions with prefix.
		{[]Statement{{
			Actions:    readOnlyBucketActions.Union(writeOnlyBucketActions),
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: helloCondMap,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, "mybucket", "hello", BucketPolicyNone},
	}

	for _, testCase := range testCases {
		policy := GetPolicy(testCase.statements, testCase.bucketName, testCase.prefix)
		if testCase.expectedResult != policy {
			t.Fatalf("%+v: expected: %s, got: %s", testCase, testCase.expectedResult, policy)
		}
	}
}

// SetPolicy() is called and the result is validated.
func TestSetPolicy(t *testing.T) {
	helloCondMap := make(ConditionMap)
	helloCondKeyMap := make(ConditionKeyMap)
	helloCondKeyMap.Add("s3:prefix", set.CreateStringSet("hello"))
	helloCondMap.Add("StringEquals", helloCondKeyMap)

	testCases := []struct {
		statements     []Statement
		policy         BucketPolicy
		bucketName     string
		prefix         string
		expectedResult string
	}{
		// BucketPolicyNone - empty statements, bucket name and prefix.
		{[]Statement{}, BucketPolicyNone, "", "", `[]`},
		// BucketPolicyNone - non-empty statements, bucket name and prefix.
		{[]Statement{{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, BucketPolicyNone, "", "", `[{"Action":["s3:ListBucket"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""}]`},
		// BucketPolicyNone - empty statements, non-empty bucket name and prefix.
		{[]Statement{}, BucketPolicyNone, "mybucket", "", `[]`},
		// BucketPolicyNone - empty statements, bucket name and non-empty prefix.
		{[]Statement{}, BucketPolicyNone, "", "hello", `[]`},
		// BucketPolicyReadOnly - empty statements, bucket name and prefix.
		{[]Statement{}, BucketPolicyReadOnly, "", "", `[]`},
		// BucketPolicyReadOnly - non-empty statements, bucket name and prefix.
		{[]Statement{{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::testbucket"),
		}}, BucketPolicyReadOnly, "", "", `[{"Action":["s3:ListBucket"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::testbucket"],"Sid":""}]`},
		// BucketPolicyReadOnly - empty statements, non-empty bucket name and prefix.
		{[]Statement{}, BucketPolicyReadOnly, "mybucket", "", `[{"Action":["s3:GetBucketLocation","s3:ListBucket"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:GetObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/*"],"Sid":""}]`},
		// BucketPolicyReadOnly - empty statements, bucket name and non-empty prefix.
		{[]Statement{}, BucketPolicyReadOnly, "", "hello", `[]`},
		// BucketPolicyReadOnly - empty statements, non-empty bucket name and non-empty prefix.
		{[]Statement{}, BucketPolicyReadOnly, "mybucket", "hello", `[{"Action":["s3:GetBucketLocation"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:ListBucket"],"Condition":{"StringEquals":{"s3:prefix":["hello"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:GetObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/hello*"],"Sid":""}]`},
		// BucketPolicyWriteOnly - empty statements, bucket name and prefix.
		{[]Statement{}, BucketPolicyReadOnly, "", "", `[]`},
		// BucketPolicyWriteOnly - non-empty statements, bucket name and prefix.
		{[]Statement{{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::testbucket"),
		}}, BucketPolicyWriteOnly, "", "", `[{"Action":["s3:ListBucket"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::testbucket"],"Sid":""}]`},
		// BucketPolicyWriteOnly - empty statements, non-empty bucket name and prefix.
		{[]Statement{}, BucketPolicyWriteOnly, "mybucket", "", `[{"Action":["s3:GetBucketLocation","s3:ListBucketMultipartUploads"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:AbortMultipartUpload","s3:DeleteObject","s3:ListMultipartUploadParts","s3:PutObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/*"],"Sid":""}]`},
		// BucketPolicyWriteOnly - empty statements, bucket name and non-empty prefix.
		{[]Statement{}, BucketPolicyWriteOnly, "", "hello", `[]`},
		// BucketPolicyWriteOnly - empty statements, non-empty bucket name and non-empty prefix.
		{[]Statement{}, BucketPolicyWriteOnly, "mybucket", "hello", `[{"Action":["s3:GetBucketLocation","s3:ListBucketMultipartUploads"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:AbortMultipartUpload","s3:DeleteObject","s3:ListMultipartUploadParts","s3:PutObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/hello*"],"Sid":""}]`},
		// BucketPolicyReadWrite - empty statements, bucket name and prefix.
		{[]Statement{}, BucketPolicyReadWrite, "", "", `[]`},
		// BucketPolicyReadWrite - non-empty statements, bucket name and prefix.
		{[]Statement{{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::testbucket"),
		}}, BucketPolicyReadWrite, "", "", `[{"Action":["s3:ListBucket"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::testbucket"],"Sid":""}]`},
		// BucketPolicyReadWrite - empty statements, non-empty bucket name and prefix.
		{[]Statement{}, BucketPolicyReadWrite, "mybucket", "", `[{"Action":["s3:GetBucketLocation","s3:ListBucket","s3:ListBucketMultipartUploads"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:AbortMultipartUpload","s3:DeleteObject","s3:GetObject","s3:ListMultipartUploadParts","s3:PutObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/*"],"Sid":""}]`},
		// BucketPolicyReadWrite - empty statements, bucket name and non-empty prefix.
		{[]Statement{}, BucketPolicyReadWrite, "", "hello", `[]`},
		// BucketPolicyReadWrite - empty statements, non-empty bucket name and non-empty prefix.
		{[]Statement{}, BucketPolicyReadWrite, "mybucket", "hello", `[{"Action":["s3:GetBucketLocation","s3:ListBucketMultipartUploads"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:ListBucket"],"Condition":{"StringEquals":{"s3:prefix":["hello"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:AbortMultipartUpload","s3:DeleteObject","s3:GetObject","s3:ListMultipartUploadParts","s3:PutObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/hello*"],"Sid":""}]`},
		// Set readonly.
		{[]Statement{{
			Actions:   writeOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, BucketPolicyReadOnly, "mybucket", "", `[{"Action":["s3:GetBucketLocation","s3:ListBucket"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:GetObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/*"],"Sid":""}]`},
		// Set readonly with prefix.
		{[]Statement{{
			Actions:   writeOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, BucketPolicyReadOnly, "mybucket", "hello", `[{"Action":["s3:GetBucketLocation"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:ListBucket"],"Condition":{"StringEquals":{"s3:prefix":["hello"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:GetObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/hello*"],"Sid":""}]`},
		// Set writeonly.
		{[]Statement{{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, BucketPolicyWriteOnly, "mybucket", "", `[{"Action":["s3:GetBucketLocation","s3:ListBucketMultipartUploads"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:AbortMultipartUpload","s3:DeleteObject","s3:ListMultipartUploadParts","s3:PutObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/*"],"Sid":""}]`},
		// Set writeonly with prefix.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: helloCondMap,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, BucketPolicyWriteOnly, "mybucket", "hello", `[{"Action":["s3:GetBucketLocation","s3:ListBucketMultipartUploads"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:AbortMultipartUpload","s3:DeleteObject","s3:ListMultipartUploadParts","s3:PutObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/hello*"],"Sid":""}]`},

		// Set readwrite.
		{[]Statement{{
			Actions:   readOnlyBucketActions,
			Effect:    "Allow",
			Principal: User{AWS: set.CreateStringSet("*")},
			Resources: set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, BucketPolicyReadWrite, "mybucket", "", `[{"Action":["s3:GetBucketLocation","s3:ListBucket","s3:ListBucketMultipartUploads"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:AbortMultipartUpload","s3:DeleteObject","s3:GetObject","s3:ListMultipartUploadParts","s3:PutObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/*"],"Sid":""}]`},
		// Set readwrite with prefix.
		{[]Statement{{
			Actions:    readOnlyBucketActions,
			Effect:     "Allow",
			Principal:  User{AWS: set.CreateStringSet("*")},
			Conditions: helloCondMap,
			Resources:  set.CreateStringSet("arn:aws:s3:::mybucket"),
		}}, BucketPolicyReadWrite, "mybucket", "hello", `[{"Action":["s3:GetBucketLocation","s3:ListBucketMultipartUploads"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:ListBucket"],"Condition":{"StringEquals":{"s3:prefix":["hello"]}},"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket"],"Sid":""},{"Action":["s3:AbortMultipartUpload","s3:DeleteObject","s3:GetObject","s3:ListMultipartUploadParts","s3:PutObject"],"Effect":"Allow","Principal":{"AWS":["*"]},"Resource":["arn:aws:s3:::mybucket/hello*"],"Sid":""}]`},
	}

	for _, testCase := range testCases {
		statements := SetPolicy(testCase.statements, testCase.policy, testCase.bucketName, testCase.prefix)
		if data, err := json.Marshal(statements); err != nil {
			t.Fatalf("unable encoding to json, %s", err)
		} else if string(data) != testCase.expectedResult {
			t.Fatalf("%+v: expected: %s, got: %s", testCase, testCase.expectedResult, string(data))
		}
	}
}

// Validates bucket policy string.
func TestIsValidBucketPolicy(t *testing.T) {
	testCases := []struct {
		inputPolicy    BucketPolicy
		expectedResult bool
	}{
		// valid inputs.
		{BucketPolicy("none"), true},
		{BucketPolicy("readonly"), true},
		{BucketPolicy("readwrite"), true},
		{BucketPolicy("writeonly"), true},
		// invalid input.
		{BucketPolicy("readwriteonly"), false},
		{BucketPolicy("writeread"), false},
	}

	for i, testCase := range testCases {
		actualResult := testCase.inputPolicy.IsValidBucketPolicy()
		if testCase.expectedResult != actualResult {
			t.Errorf("Test %d: Expected IsValidBucket policy to be '%v' for policy \"%s\", but instead found it to be '%v'", i+1, testCase.expectedResult, testCase.inputPolicy, actualResult)
		}
	}
}

// Tests validate Bucket policy resource matcher.
func TestBucketPolicyResourceMatch(t *testing.T) {
	// generates\ statement with given resource..
	generateStatement := func(resource string) Statement {
		statement := Statement{}
		statement.Resources = set.CreateStringSet(resource)
		return statement
	}

	// generates resource prefix.
	generateResource := func(bucketName, objectName string) string {
		return awsResourcePrefix + bucketName + "/" + objectName
	}

	testCases := []struct {
		resourceToMatch       string
		statement             Statement
		expectedResourceMatch bool
	}{
		// Test case 1-4.
		// Policy with resource ending with bucket/* allows access to all objects inside the given bucket.
		{generateResource("minio-bucket", ""), generateStatement(fmt.Sprintf("%s%s", awsResourcePrefix, "minio-bucket"+"/*")), true},
		{generateResource("minio-bucket", ""), generateStatement(fmt.Sprintf("%s%s", awsResourcePrefix, "minio-bucket"+"/*")), true},
		{generateResource("minio-bucket", ""), generateStatement(fmt.Sprintf("%s%s", awsResourcePrefix, "minio-bucket"+"/*")), true},
		{generateResource("minio-bucket", ""), generateStatement(fmt.Sprintf("%s%s", awsResourcePrefix, "minio-bucket"+"/*")), true},
		// Test case - 5.
		// Policy with resource ending with bucket/oo* should not allow access to bucket/output.txt.
		{generateResource("minio-bucket", "output.txt"), generateStatement(fmt.Sprintf("%s%s", awsResourcePrefix, "minio-bucket"+"/oo*")), false},
		// Test case - 6.
		// Policy with resource ending with bucket/oo* should allow access to bucket/ootput.txt.
		{generateResource("minio-bucket", "ootput.txt"), generateStatement(fmt.Sprintf("%s%s", awsResourcePrefix, "minio-bucket"+"/oo*")), true},
		// Test case - 7.
		// Policy with resource ending with bucket/oo* allows access to all subfolders starting with "oo" inside given bucket.
		{generateResource("minio-bucket", "oop-bucket/my-file"), generateStatement(fmt.Sprintf("%s%s", awsResourcePrefix, "minio-bucket"+"/oo*")), true},
		// Test case - 8.
		{generateResource("minio-bucket", "Asia/India/1.pjg"), generateStatement(fmt.Sprintf("%s%s", awsResourcePrefix, "minio-bucket"+"/Asia/Japan/*")), false},
		// Test case - 9.
		{generateResource("minio-bucket", "Asia/India/1.pjg"), generateStatement(fmt.Sprintf("%s%s", awsResourcePrefix, "minio-bucket"+"/Asia/Japan/*")), false},
		// Test case - 10.
		// Proves that the name space is flat.
		{generateResource("minio-bucket", "Africa/Bihar/India/design_info.doc/Bihar"), generateStatement(fmt.Sprintf("%s%s", awsResourcePrefix,
			"minio-bucket"+"/*/India/*/Bihar")), true},
		// Test case - 11.
		// Proves that the name space is flat.
		{generateResource("minio-bucket", "Asia/China/India/States/Bihar/output.txt"), generateStatement(fmt.Sprintf("%s%s", awsResourcePrefix,
			"minio-bucket"+"/*/India/*/Bihar/*")), true},
	}
	for i, testCase := range testCases {
		resources := testCase.statement.Resources.FuncMatch(resourceMatch, testCase.resourceToMatch)
		actualResourceMatch := resources.Equals(testCase.statement.Resources)
		if testCase.expectedResourceMatch != actualResourceMatch {
			t.Errorf("Test %d: Expected Resource match to be `%v`, but instead found it to be `%v`", i+1, testCase.expectedResourceMatch, actualResourceMatch)
		}
	}
}
