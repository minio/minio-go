/*
 * Minio Go Library for Amazon S3 Compatible Cloud Storage (C) 2015 Minio, Inc.
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
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/minio/minio-go/pkg/set"
)

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
		actualResult := testCase.inputPolicy.isValidBucketPolicy()
		if testCase.expectedResult != actualResult {
			t.Errorf("Test %d: Expected IsValidBucket policy to be '%v' for policy \"%s\", but instead found it to be '%v'", i+1, testCase.expectedResult, testCase.inputPolicy, actualResult)
		}
	}
}

// Tests validate Bucket Policy type identifier.
func TestIdentifyPolicyType(t *testing.T) {
	testCases := []struct {
		inputPolicy BucketAccessPolicy
		bucketName  string
		objName     string

		expectedPolicy BucketPolicy
	}{
		{BucketAccessPolicy{Version: "2012-10-17"}, "my-bucket", "", BucketPolicyNone},
	}
	for i, testCase := range testCases {
		actualBucketPolicy := identifyPolicyType(testCase.inputPolicy, testCase.bucketName, testCase.objName)
		if testCase.expectedPolicy != actualBucketPolicy {
			t.Errorf("Test %d: Expected bucket policy to be '%v', but instead got '%v'", i+1, testCase.expectedPolicy, actualBucketPolicy)
		}
	}
}

// Test validate Resource Statement Generator.
func TestGeneratePolicyStatement(t *testing.T) {

	testCases := []struct {
		bucketPolicy       BucketPolicy
		bucketName         string
		objectPrefix       string
		expectedStatements []Statement

		shouldPass bool
		err        error
	}{
		{BucketPolicy("my-policy"), "my-bucket", "", []Statement{}, false, ErrInvalidArgument(fmt.Sprintf("Invalid bucket policy provided. %s", BucketPolicy("my-policy")))},
		{BucketPolicyNone, "my-bucket", "", []Statement{}, true, nil},
		{BucketPolicyReadOnly, "read-only-bucket", "", setReadOnlyStatement("read-only-bucket", ""), true, nil},
		{BucketPolicyWriteOnly, "write-only-bucket", "", setWriteOnlyStatement("write-only-bucket", ""), true, nil},
		{BucketPolicyReadWrite, "read-write-bucket", "", setReadWriteStatement("read-write-bucket", ""), true, nil},
	}
	for i, testCase := range testCases {
		actualStatements, err := generatePolicyStatement(testCase.bucketPolicy, testCase.bucketName, testCase.objectPrefix)

		if err != nil && testCase.shouldPass {
			t.Errorf("Test %d: Expected to pass, but failed with: <ERROR> %s", i+1, err.Error())
		}

		if err == nil && !testCase.shouldPass {
			t.Errorf("Test %d: Expected to fail with <ERROR> \"%s\", but passed instead", i+1, testCase.err.Error())
		}
		// Failed as expected, but does it fail for the expected reason.
		if err != nil && !testCase.shouldPass {
			if err.Error() != testCase.err.Error() {
				t.Errorf("Test %d: Expected to fail with error \"%s\", but instead failed with error \"%s\" instead", i+1, testCase.err.Error(), err.Error())
			}
		}
		// Test passes as expected, but the output values are verified for correctness here.
		if err == nil && testCase.shouldPass {
			if !reflect.DeepEqual(testCase.expectedStatements, actualStatements) {
				t.Errorf("Test %d: The expected statements from resource statement generator doesn't match the actual statements", i+1)
			}
		}
	}
}

// Tests validating read only statement generator.
func TestsetReadOnlyStatement(t *testing.T) {

	expectedReadOnlyStatement := func(bucketName, objectPrefix string) []Statement {
		bucketResourceStatement := &Statement{}
		bucketListResourceStatement := &Statement{}
		objectResourceStatement := &Statement{}
		statements := []Statement{}

		bucketResourceStatement.Effect = "Allow"
		bucketResourceStatement.Principal.AWS = set.CreateStringSet("*")
		bucketResourceStatement.Resources = set.CreateStringSet(awsResourcePrefix + bucketName)
		bucketResourceStatement.Actions = set.CreateStringSet(readOnlyBucketActions...)
		bucketListResourceStatement.Effect = "Allow"
		bucketListResourceStatement.Principal.AWS = set.CreateStringSet("*")
		bucketListResourceStatement.Resources = set.CreateStringSet(awsResourcePrefix + bucketName)
		bucketListResourceStatement.Actions = set.CreateStringSet("s3:ListBucket")
		if objectPrefix != "" {
			bucketListResourceStatement.Conditions = map[string]map[string]string{
				"StringEquals": {
					"s3:prefix": objectPrefix,
				},
			}
		}
		objectResourceStatement.Effect = "Allow"
		objectResourceStatement.Principal.AWS = set.CreateStringSet("*")
		objectResourceStatement.Resources = set.CreateStringSet(awsResourcePrefix + bucketName + "/" + objectPrefix + "*")
		objectResourceStatement.Actions = set.CreateStringSet(readOnlyObjectActions...)
		// Save the read only policy.
		statements = append(statements, *bucketResourceStatement, *bucketListResourceStatement, *objectResourceStatement)
		return statements
	}

	testCases := []struct {
		// inputs.
		bucketName   string
		objectPrefix string
		// expected result.
		expectedStatements []Statement
	}{
		{"my-bucket", "", expectedReadOnlyStatement("my-bucket", "")},
		{"my-bucket", "Asia/", expectedReadOnlyStatement("my-bucket", "Asia/")},
		{"my-bucket", "Asia/India", expectedReadOnlyStatement("my-bucket", "Asia/India")},
	}

	for i, testCase := range testCases {
		actualStaments := setReadOnlyStatement(testCase.bucketName, testCase.objectPrefix)
		if !reflect.DeepEqual(testCase.expectedStatements, actualStaments) {
			t.Errorf("Test %d: The expected statements from resource statement generator doesn't match the actual statements", i+1)
		}
	}
}

// Tests validating write only statement generator.
func TestsetWriteOnlyStatement(t *testing.T) {

	expectedWriteOnlyStatement := func(bucketName, objectPrefix string) []Statement {
		bucketResourceStatement := &Statement{}
		objectResourceStatement := &Statement{}
		statements := []Statement{}
		// Write only policy.
		bucketResourceStatement.Effect = "Allow"
		bucketResourceStatement.Principal.AWS = set.CreateStringSet("*")
		bucketResourceStatement.Resources = set.CreateStringSet(awsResourcePrefix + bucketName)
		bucketResourceStatement.Actions = set.CreateStringSet(writeOnlyBucketActions...)
		objectResourceStatement.Effect = "Allow"
		objectResourceStatement.Principal.AWS = set.CreateStringSet("*")
		objectResourceStatement.Resources = set.CreateStringSet(awsResourcePrefix + bucketName + "/" + objectPrefix + "*")
		objectResourceStatement.Actions = set.CreateStringSet(writeOnlyObjectActions...)
		// Save the write only policy.
		statements = append(statements, *bucketResourceStatement, *objectResourceStatement)
		return statements
	}
	testCases := []struct {
		// inputs.
		bucketName   string
		objectPrefix string
		// expected result.
		expectedStatements []Statement
	}{
		{"my-bucket", "", expectedWriteOnlyStatement("my-bucket", "")},
		{"my-bucket", "Asia/", expectedWriteOnlyStatement("my-bucket", "Asia/")},
		{"my-bucket", "Asia/India", expectedWriteOnlyStatement("my-bucket", "Asia/India")},
	}

	for i, testCase := range testCases {
		actualStaments := setWriteOnlyStatement(testCase.bucketName, testCase.objectPrefix)
		if !reflect.DeepEqual(testCase.expectedStatements, actualStaments) {
			t.Errorf("Test %d: The expected statements from resource statement generator doesn't match the actual statements", i+1)
		}
	}
}

// Tests validating read-write statement generator.
func TestsetReadWriteStatement(t *testing.T) {
	// Obtain statements for read-write BucketPolicy.
	expectedReadWriteStatement := func(bucketName, objectPrefix string) []Statement {
		bucketResourceStatement := &Statement{}
		bucketListResourceStatement := &Statement{}
		objectResourceStatement := &Statement{}
		statements := []Statement{}

		bucketResourceStatement.Effect = "Allow"
		bucketResourceStatement.Principal.AWS = set.CreateStringSet("*")
		bucketResourceStatement.Resources = set.CreateStringSet(awsResourcePrefix + bucketName)
		bucketResourceStatement.Actions = set.CreateStringSet(readWriteBucketActions...)
		bucketListResourceStatement.Effect = "Allow"
		bucketListResourceStatement.Principal.AWS = set.CreateStringSet("*")
		bucketListResourceStatement.Resources = set.CreateStringSet(awsResourcePrefix + bucketName)
		bucketListResourceStatement.Actions = set.CreateStringSet("s3:ListBucket")
		if objectPrefix != "" {
			bucketListResourceStatement.Conditions = map[string]map[string]string{
				"StringEquals": {
					"s3:prefix": objectPrefix,
				},
			}
		}
		objectResourceStatement.Effect = "Allow"
		objectResourceStatement.Principal.AWS = set.CreateStringSet("*")
		objectResourceStatement.Resources = set.CreateStringSet(awsResourcePrefix + bucketName + "/" + objectPrefix + "*")
		objectResourceStatement.Actions = set.CreateStringSet(readWriteObjectActions...)
		// Save the read write policy.
		statements = append(statements, *bucketResourceStatement, *bucketListResourceStatement, *objectResourceStatement)
		return statements
	}

	testCases := []struct {
		// inputs.
		bucketName   string
		objectPrefix string
		// expected result.
		expectedStatements []Statement
	}{
		{"my-bucket", "", expectedReadWriteStatement("my-bucket", "")},
		{"my-bucket", "Asia/", expectedReadWriteStatement("my-bucket", "Asia/")},
		{"my-bucket", "Asia/India", expectedReadWriteStatement("my-bucket", "Asia/India")},
	}

	for i, testCase := range testCases {
		actualStaments := setReadWriteStatement(testCase.bucketName, testCase.objectPrefix)
		if !reflect.DeepEqual(testCase.expectedStatements, actualStaments) {
			t.Errorf("Test %d: The expected statements from resource statement generator doesn't match the actual statements", i+1)
		}
	}
}

// Tests validate Unmarshalling of BucketAccessPolicy.
func TestUnMarshalBucketPolicy(t *testing.T) {

	bucketAccesPolicies := []BucketAccessPolicy{
		{Version: "1.0"},
		{Version: "1.0", Statements: setReadOnlyStatement("minio-bucket", "")},
		{Version: "1.0", Statements: setReadWriteStatement("minio-bucket", "Asia/")},
		{Version: "1.0", Statements: setWriteOnlyStatement("minio-bucket", "Asia/India/")},
	}

	testCases := []struct {
		inputPolicy BucketAccessPolicy
		// expected results.
		expectedPolicy BucketAccessPolicy
		err            error
		// Flag indicating whether the test should pass.
		shouldPass bool
	}{
		{bucketAccesPolicies[0], bucketAccesPolicies[0], nil, true},
		{bucketAccesPolicies[1], bucketAccesPolicies[1], nil, true},
		{bucketAccesPolicies[2], bucketAccesPolicies[2], nil, true},
		{bucketAccesPolicies[3], bucketAccesPolicies[3], nil, true},
	}
	for i, testCase := range testCases {
		inputPolicyBytes, e := json.Marshal(testCase.inputPolicy)
		if e != nil {
			t.Fatalf("Test %d: Couldn't Marshal bucket policy", i+1)
		}
		actualAccessPolicy := BucketAccessPolicy{}
		err := json.Unmarshal(inputPolicyBytes, &actualAccessPolicy)
		if err != nil && testCase.shouldPass {
			t.Errorf("Test %d: Expected to pass, but failed with: <ERROR> %s", i+1, err.Error())
		}

		if err == nil && !testCase.shouldPass {
			t.Errorf("Test %d: Expected to fail with <ERROR> \"%s\", but passed instead", i+1, testCase.err.Error())
		}
		// Failed as expected, but does it fail for the expected reason.
		if err != nil && !testCase.shouldPass {
			if err.Error() != testCase.err.Error() {
				t.Errorf("Test %d: Expected to fail with error \"%s\", but instead failed with error \"%s\" instead", i+1, testCase.err.Error(), err.Error())
			}
		}
		// Test passes as expected, but the output values are verified for correctness here.
		if err == nil && testCase.shouldPass {
			if !reflect.DeepEqual(testCase.expectedPolicy, actualAccessPolicy) {
				t.Errorf("Test %d: The expected statements from resource statement generator doesn't match the actual statements", i+1)
			}
		}
	}
}

// Tests validate whether access policy is defined for the given object prefix
func TestIsPolicyDefinedForObjectPrefix(t *testing.T) {
	testCases := []struct {
		bucketName      string
		objectPrefix    string
		inputStatements []Statement
		expectedResult  bool
	}{
		{"my-bucket", "abc/", setReadOnlyStatement("my-bucket", "abc/"), true},
		{"my-bucket", "abc/", setReadOnlyStatement("my-bucket", "ab/"), false},
		{"my-bucket", "abc/", setReadOnlyStatement("my-bucket", "abcde"), false},
		{"my-bucket", "abc/", setReadOnlyStatement("my-bucket", "abc/de"), false},
		{"my-bucket", "abc", setReadOnlyStatement("my-bucket", "abc"), true},
		{"bucket", "", setReadOnlyStatement("bucket", "abc/"), false},
	}
	for i, testCase := range testCases {
		actualResult := isPolicyDefinedForObjectPrefix(testCase.inputStatements, testCase.bucketName, testCase.objectPrefix)
		if actualResult != testCase.expectedResult {
			t.Errorf("Test %d: Expected isPolicyDefinedForObjectPrefix to '%v', but instead found '%v'", i+1, testCase.expectedResult, actualResult)
		}
	}
}

// Tests validate removal of policy statement from the list of statements.
func TestRemoveBucketPolicyStatement(t *testing.T) {
	var emptyStatement []Statement
	testCases := []struct {
		bucketName         string
		objectPrefix       string
		inputStatements    []Statement
		expectedStatements []Statement
	}{
		{"my-bucket", "", nil, emptyStatement},
		{"read-only-bucket", "", setReadOnlyStatement("read-only-bucket", ""), emptyStatement},
		{"write-only-bucket", "", setWriteOnlyStatement("write-only-bucket", ""), emptyStatement},
		{"read-write-bucket", "", setReadWriteStatement("read-write-bucket", ""), emptyStatement},
		{"my-bucket", "abcd", setReadOnlyStatement("my-bucket", "abc"), setReadOnlyStatement("my-bucket", "abc")},
		{"my-bucket", "abc/de", setReadOnlyStatement("my-bucket", "abc/"), setReadOnlyStatement("my-bucket", "abc/")},
		{"my-bucket", "abcd", setWriteOnlyStatement("my-bucket", "abc"), setWriteOnlyStatement("my-bucket", "abc")},
		{"my-bucket", "abc/de", setWriteOnlyStatement("my-bucket", "abc/"), setWriteOnlyStatement("my-bucket", "abc/")},
		{"my-bucket", "abcd", setReadWriteStatement("my-bucket", "abc"), setReadWriteStatement("my-bucket", "abc")},
		{"my-bucket", "abc/de", setReadWriteStatement("my-bucket", "abc/"), setReadWriteStatement("my-bucket", "abc/")},
	}
	for i, testCase := range testCases {
		actualStatements := removeBucketPolicyStatement(testCase.inputStatements, testCase.bucketName, testCase.objectPrefix)
		if !reflect.DeepEqual(testCase.expectedStatements, actualStatements) {
			t.Errorf("Test %d: The expected statements from resource statement generator doesn't match the actual statements", i+1)
		}
	}
}

// Tests validate removing of read only bucket statement.
func TestRemoveBucketPolicyStatementReadOnly(t *testing.T) {
	var emptyStatement []Statement
	testCases := []struct {
		bucketName         string
		objectPrefix       string
		inputStatements    []Statement
		expectedStatements []Statement
	}{
		{"my-bucket", "", []Statement{}, emptyStatement},
		{"read-only-bucket", "", setReadOnlyStatement("read-only-bucket", ""), emptyStatement},
		{"read-only-bucket", "abc/", setReadOnlyStatement("read-only-bucket", "abc/"), emptyStatement},
		{"my-bucket", "abc/", append(setReadOnlyStatement("my-bucket", "abc/"), setReadOnlyStatement("my-bucket", "def/")...), setReadOnlyStatement("my-bucket", "def/")},
	}
	for i, testCase := range testCases {
		actualStatements := removeBucketPolicyStatementReadOnly(testCase.inputStatements, testCase.bucketName, testCase.objectPrefix)
		if !reflect.DeepEqual(testCase.expectedStatements, actualStatements) {
			t.Errorf("Test %d: Expected policy statements doesn't match the actual one", i+1)
		}
	}
}

// Tests validate removing of write only bucket statement.
func TestRemoveBucketPolicyStatementWriteOnly(t *testing.T) {
	var emptyStatement []Statement
	testCases := []struct {
		bucketName         string
		objectPrefix       string
		inputStatements    []Statement
		expectedStatements []Statement
	}{
		{"my-bucket", "", []Statement{}, emptyStatement},
		{"write-only-bucket", "", setWriteOnlyStatement("write-only-bucket", ""), emptyStatement},
		{"write-only-bucket", "abc/", setWriteOnlyStatement("write-only-bucket", "abc/"), emptyStatement},
		{"my-bucket", "abc/", append(setWriteOnlyStatement("my-bucket", "abc/"), setWriteOnlyStatement("my-bucket", "def/")...), setWriteOnlyStatement("my-bucket", "def/")},
	}
	for i, testCase := range testCases {
		actualStatements := removeBucketPolicyStatementWriteOnly(testCase.inputStatements, testCase.bucketName, testCase.objectPrefix)
		if !reflect.DeepEqual(testCase.expectedStatements, actualStatements) {
			t.Errorf("Test %d: Expected policy statements doesn't match the actual one", i+1)
		}
	}
}

// Tests validate removing of read-write bucket statement.
func TestRemoveBucketPolicyStatementReadWrite(t *testing.T) {
	var emptyStatement []Statement
	testCases := []struct {
		bucketName         string
		objectPrefix       string
		inputStatements    []Statement
		expectedStatements []Statement
	}{
		{"my-bucket", "", []Statement{}, emptyStatement},
		{"read-write-bucket", "", setReadWriteStatement("read-write-bucket", ""), emptyStatement},
		{"read-write-bucket", "abc/", setReadWriteStatement("read-write-bucket", "abc/"), emptyStatement},
		{"my-bucket", "abc/", append(setReadWriteStatement("my-bucket", "abc/"), setReadWriteStatement("my-bucket", "def/")...), setReadWriteStatement("my-bucket", "def/")},
	}
	for i, testCase := range testCases {
		actualStatements := removeBucketPolicyStatementReadWrite(testCase.inputStatements, testCase.bucketName, testCase.objectPrefix)
		if !reflect.DeepEqual(testCase.expectedStatements, actualStatements) {
			t.Errorf("Test %d: Expected policy statements doesn't match the actual one", i+1)
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

// Tests validate whether the bucket policy is read only.
func TestIsBucketPolicyReadOnly(t *testing.T) {
	testCases := []struct {
		BbucketName      string      `json:"BucketName"`
		OobjectPrefix    string      `json:"ObjectPrefix"`
		IinputStatements []Statement `json:"InputStatements"`
		// expected result.
		EexpectedResult bool `json:"ExpectedResult"`
	}{
		{"my-bucket", "", []Statement{}, false},
		{"read-only-bucket", "", setReadOnlyStatement("read-only-bucket", ""), true},
		{"write-only-bucket", "", setWriteOnlyStatement("write-only-bucket", ""), false},
		{"read-write-bucket", "", setReadWriteStatement("read-write-bucket", ""), true},
		{"my-bucket", "abc", setReadOnlyStatement("my-bucket", ""), true},
		{"my-bucket", "abc", setReadOnlyStatement("my-bucket", "abc"), true},
		{"my-bucket", "abcde", setReadOnlyStatement("my-bucket", "abc"), true},
		{"my-bucket", "abc/d", setReadOnlyStatement("my-bucket", "abc/"), true},
		{"my-bucket", "abc", setWriteOnlyStatement("my-bucket", ""), false},
	}
	for i, testCase := range testCases {
		actualResult := isBucketPolicyReadOnly(testCase.IinputStatements, testCase.BbucketName, testCase.OobjectPrefix)
		if testCase.EexpectedResult != actualResult {
			s, _ := json.Marshal(testCase)
			t.Errorf(string(s))
			t.Errorf("Test %d: Expected isBucketPolicyReadonly to '%v', but instead found '%v'", i+1, testCase.EexpectedResult, actualResult)
		}
	}
}

// Tests validate whether the bucket policy is read-write.
func TestIsBucketPolicyReadWrite(t *testing.T) {
	testCases := []struct {
		bucketName      string
		objectPrefix    string
		inputStatements []Statement
		// expected result.
		expectedResult bool
	}{
		{"my-bucket", "", []Statement{}, false},
		{"read-only-bucket", "", setReadOnlyStatement("read-only-bucket", ""), false},
		{"write-only-bucket", "", setWriteOnlyStatement("write-only-bucket", ""), false},
		{"read-write-bucket", "", setReadWriteStatement("read-write-bucket", ""), true},
		{"my-bucket", "abc", setReadWriteStatement("my-bucket", ""), true},
		{"my-bucket", "abc", setReadWriteStatement("my-bucket", "abc"), true},
		{"my-bucket", "abcde", setReadWriteStatement("my-bucket", "abc"), true},
		{"my-bucket", "abc/d", setReadWriteStatement("my-bucket", "abc/"), true},
	}
	for i, testCase := range testCases {
		actualResult := isBucketPolicyReadWrite(testCase.inputStatements, testCase.bucketName, testCase.objectPrefix)
		if testCase.expectedResult != actualResult {
			t.Errorf("Test %d: Expected isBucketPolicyReadonly to '%v', but instead found '%v'", i+1, testCase.expectedResult, actualResult)
		}
	}
}

// Tests validate whether the bucket policy is read only.
func TestIsBucketPolicyWriteOnly(t *testing.T) {
	testCases := []struct {
		bucketName      string
		objectPrefix    string
		inputStatements []Statement
		// expected result.
		expectedResult bool
	}{
		{"my-bucket", "", []Statement{}, false},
		{"read-only-bucket", "", setReadOnlyStatement("read-only-bucket", ""), false},
		{"write-only-bucket", "", setWriteOnlyStatement("write-only-bucket", ""), true},
		{"read-write-bucket", "", setReadWriteStatement("read-write-bucket", ""), true},
		{"my-bucket", "abc", setWriteOnlyStatement("my-bucket", ""), true},
		{"my-bucket", "abc", setWriteOnlyStatement("my-bucket", "abc"), true},
		{"my-bucket", "abcde", setWriteOnlyStatement("my-bucket", "abc"), true},
		{"my-bucket", "abc/d", setWriteOnlyStatement("my-bucket", "abc/"), true},
		{"my-bucket", "abc", setReadOnlyStatement("my-bucket", ""), false},
	}
	for i, testCase := range testCases {
		actualResult := isBucketPolicyWriteOnly(testCase.inputStatements, testCase.bucketName, testCase.objectPrefix)
		if testCase.expectedResult != actualResult {
			t.Errorf("Test %d: Expected isBucketPolicyReadonly to '%v', but instead found '%v'", i+1, testCase.expectedResult, actualResult)
		}
	}
}
