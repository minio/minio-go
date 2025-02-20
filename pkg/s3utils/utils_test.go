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

package s3utils

import (
	"errors"
	"github.com/minio/minio-go/v7/pkg/tags"
	"net/url"
	"reflect"
	"testing"
)

// Tests get region from host URL.
func TestGetRegionFromURL(t *testing.T) {
	testCases := []struct {
		u              string
		expectedRegion string
	}{
		{u: "storage.googleapis.com", expectedRegion: ""},
		{u: "s3.cn-north-1.amazonaws.com.cn", expectedRegion: "cn-north-1"},
		{u: "s3.dualstack.cn-north-1.amazonaws.com.cn", expectedRegion: "cn-north-1"},
		{u: "s3.cn-northwest-1.amazonaws.com.cn", expectedRegion: "cn-northwest-1"},
		{u: "s3.dualstack.cn-northwest-1.amazonaws.com.cn", expectedRegion: "cn-northwest-1"},
		{u: "s3-fips-us-gov-west-1.amazonaws.com", expectedRegion: "us-gov-west-1"},
		{u: "s3-fips.us-gov-west-1.amazonaws.com", expectedRegion: "us-gov-west-1"},
		{u: "s3-fips.us-gov-east-1.amazonaws.com", expectedRegion: "us-gov-east-1"},
		{u: "s3-us-gov-west-1.amazonaws.com", expectedRegion: "us-gov-west-1"},
		{u: "192.168.1.1", expectedRegion: ""},
		{u: "s3-eu-west-1.amazonaws.com", expectedRegion: "eu-west-1"},
		{u: "s3.eu-west-1.amazonaws.com", expectedRegion: "eu-west-1"},
		{u: "s3.dualstack.eu-west-1.amazonaws.com", expectedRegion: "eu-west-1"},
		{u: "s3.amazonaws.com", expectedRegion: ""},
		{u: "s3-external-1.amazonaws.com", expectedRegion: ""},
		{u: "s3.kubernetesfrontendlb-caf78da2b1f7516c.elb.us-west-2.amazonaws.com", expectedRegion: ""},
		{u: "s3.kubernetesfrontendlb-caf78da2b1f7516c.elb.amazonaws.com", expectedRegion: ""},
		{u: "s3.kubernetesfrontendlb-caf78da2b1f7516c.elb.amazonaws.com.cn", expectedRegion: ""},
		{u: "bucket.vpce-1a2b3c4d-5e6f.s3.us-east-1.vpce.amazonaws.com", expectedRegion: "us-east-1"},
		{u: "accesspoint.vpce-1a2b3c4d-5e6f.s3.us-east-1.vpce.amazonaws.com", expectedRegion: "us-east-1"},
		{u: "s3-fips.us-east-1.amazonaws.com", expectedRegion: "us-east-1"},
		{u: "s3-fips.dualstack.us-west-1.amazonaws.com", expectedRegion: "us-west-1"},

		// Test cases with port numbers.
		{u: "storage.googleapis.com:80", expectedRegion: ""},
		{u: "s3.cn-north-1.amazonaws.com.cn:80", expectedRegion: "cn-north-1"},
		{u: "s3.dualstack.cn-north-1.amazonaws.com.cn:80", expectedRegion: "cn-north-1"},
		{u: "s3.cn-northwest-1.amazonaws.com.cn:80", expectedRegion: "cn-northwest-1"},
		{u: "s3.dualstack.cn-northwest-1.amazonaws.com.cn:80", expectedRegion: "cn-northwest-1"},
		{u: "s3-fips-us-gov-west-1.amazonaws.com:80", expectedRegion: "us-gov-west-1"},
		{u: "s3-fips.us-gov-west-1.amazonaws.com:80", expectedRegion: "us-gov-west-1"},
		{u: "s3-fips.us-gov-east-1.amazonaws.com:80", expectedRegion: "us-gov-east-1"},
		{u: "s3-us-gov-west-1.amazonaws.com:80", expectedRegion: "us-gov-west-1"},
		{u: "192.168.1.1:80", expectedRegion: ""},
		{u: "s3-eu-west-1.amazonaws.com:80", expectedRegion: "eu-west-1"},
		{u: "s3.eu-west-1.amazonaws.com:80", expectedRegion: "eu-west-1"},
		{u: "s3.dualstack.eu-west-1.amazonaws.com:80", expectedRegion: "eu-west-1"},
		{u: "s3.amazonaws.com:80", expectedRegion: ""},
		{u: "s3-external-1.amazonaws.com:80", expectedRegion: ""},
		{u: "s3.kubernetesfrontendlb-caf78da2b1f7516c.elb.us-west-2.amazonaws.com:80", expectedRegion: ""},
		{u: "s3.kubernetesfrontendlb-caf78da2b1f7516c.elb.amazonaws.com:80", expectedRegion: ""},
		{u: "s3.kubernetesfrontendlb-caf78da2b1f7516c.elb.amazonaws.com.cn:80", expectedRegion: ""},
		{u: "bucket.vpce-1a2b3c4d-5e6f.s3.us-east-1.vpce.amazonaws.com:80", expectedRegion: "us-east-1"},
		{u: "accesspoint.vpce-1a2b3c4d-5e6f.s3.us-east-1.vpce.amazonaws.com:80", expectedRegion: "us-east-1"},
		{u: "s3-fips.us-east-1.amazonaws.com:80", expectedRegion: "us-east-1"},
		{u: "s3-fips.dualstack.us-west-1.amazonaws.com:80", expectedRegion: "us-west-1"},
	}

	for i, testCase := range testCases {
		region := GetRegionFromURL(url.URL{Host: testCase.u})
		if testCase.expectedRegion != region {
			t.Errorf("Test %d: Expected region %s, got %s", i+1, testCase.expectedRegion, region)
		}
	}
}

// Tests for 'isValidDomain(host string) bool'.
func TestIsValidDomain(t *testing.T) {
	testCases := []struct {
		// Input.
		host string
		// Expected result.
		result bool
	}{
		{"s3.amazonaws.com", true},
		{"s3.cn-north-1.amazonaws.com.cn", true},
		{"s3.cn-northwest-1.amazonaws.com.cn", true},
		{"s3.amazonaws.com_", false},
		{"%$$$", false},
		{"s3.amz.test.com", true},
		{"s3.%%", false},
		{"localhost", true},
		{"localhost.", true}, // http://www.dns-sd.org/trailingdotsindomainnames.html
		{"-localhost", false},
		{"", false},
		{"\n \t", false},
		{"   ", false},
	}

	for i, testCase := range testCases {
		result := IsValidDomain(testCase.host)
		if testCase.result != result {
			t.Errorf("Test %d: Expected isValidDomain test to be '%v', but found '%v' instead", i+1, testCase.result, result)
		}
	}
}

// Tests validate IP address validator.
func TestIsValidIP(t *testing.T) {
	testCases := []struct {
		// Input.
		ip string
		// Expected result.
		result bool
	}{
		{"192.168.1.1", true},
		{"192.168.1", false},
		{"192.168.1.1.1", false},
		{"-192.168.1.1", false},
		{"260.192.1.1", false},
	}

	for i, testCase := range testCases {
		result := IsValidIP(testCase.ip)
		if testCase.result != result {
			t.Errorf("Test %d: Expected isValidIP to be '%v' for input \"%s\", but found it to be '%v' instead", i+1, testCase.result, testCase.ip, result)
		}
	}
}

// Tests validate virtual host validator.
func TestIsVirtualHostSupported(t *testing.T) {
	testCases := []struct {
		url    string
		bucket string
		// Expeceted result.
		result bool
	}{
		{"https://s3.amazonaws.com", "my-bucket", true},
		{"https://s3.cn-north-1.amazonaws.com.cn", "my-bucket", true},
		{"https://s3.amazonaws.com", "my-bucket.", false},
		{"https://amazons3.amazonaws.com", "my-bucket.", false},
		{"https://storage.googleapis.com/", "my-bucket", true},
		{"https://mystorage.googleapis.com/", "my-bucket", false},
	}

	for i, testCase := range testCases {
		u, err := url.Parse(testCase.url)
		if err != nil {
			t.Errorf("Test %d: Expected to pass, but failed with: <ERROR> %s", i+1, err)
		}
		result := IsVirtualHostSupported(*u, testCase.bucket)
		if testCase.result != result {
			t.Errorf("Test %d: Expected isVirtualHostSupported to be '%v' for input url \"%s\" and bucket \"%s\", but found it to be '%v' instead", i+1, testCase.result, testCase.url, testCase.bucket, result)
		}
	}
}

// Tests validate Amazon endpoint validator.
func TestIsAmazonEndpoint(t *testing.T) {
	testCases := []struct {
		url string
		// Expected result.
		result bool
	}{
		{"https://192.168.1.1", false},
		{"192.168.1.1", false},
		{"http://storage.googleapis.com", false},
		{"https://storage.googleapis.com", false},
		{"storage.googleapis.com", false},
		{"s3.amazonaws.com", false},
		{"https://amazons3.amazonaws.com", false},
		{"-192.168.1.1", false},
		{"260.192.1.1", false},
		{"https://s3-.amazonaws.com", false},
		{"https://s3..amazonaws.com", false},
		{"https://s3.dualstack.us-west-1.amazonaws.com.cn", false},
		{"https://s3..us-west-1.amazonaws.com.cn", false},
		// valid inputs.
		{"https://s3.amazonaws.com", true},
		{"https://s3-external-1.amazonaws.com", true},
		{"https://s3.cn-north-1.amazonaws.com.cn", true},
		{"https://s3-us-west-1.amazonaws.com", true},
		{"https://s3.us-west-1.amazonaws.com", true},
		{"https://s3.dualstack.us-west-1.amazonaws.com", true},
		{"https://bucket.vpce-1a2b3c4d-5e6f.s3.us-east-1.vpce.amazonaws.com", true},
		{"https://accesspoint.vpce-1a2b3c4d-5e6f.s3.us-east-1.vpce.amazonaws.com", true},
	}

	for i, testCase := range testCases {
		u, err := url.Parse(testCase.url)
		if err != nil {
			t.Errorf("Test %d: Expected to pass, but failed with: <ERROR> %s", i+1, err)
		}
		result := IsAmazonEndpoint(*u)
		if testCase.result != result {
			t.Errorf("Test %d: Expected isAmazonEndpoint to be '%v' for input \"%s\", but found it to be '%v' instead", i+1, testCase.result, testCase.url, result)
		}
	}
}

// Tests validate Google Cloud end point validator.
func TestIsGoogleEndpoint(t *testing.T) {
	testCases := []struct {
		url string
		// Expected result.
		result bool
	}{
		{"192.168.1.1", false},
		{"https://192.168.1.1", false},
		{"s3.amazonaws.com", false},
		{"http://s3.amazonaws.com", false},
		{"https://s3.amazonaws.com", false},
		{"https://s3.cn-north-1.amazonaws.com.cn", false},
		{"-192.168.1.1", false},
		{"260.192.1.1", false},
		// valid inputs.
		{"http://storage.googleapis.com", true},
		{"https://storage.googleapis.com", true},
		{"http://storage.googleapis.com:80", true},
		{"https://storage.googleapis.com:443", true},
	}

	for i, testCase := range testCases {
		u, err := url.Parse(testCase.url)
		if err != nil {
			t.Errorf("Test %d: Expected to pass, but failed with: <ERROR> %s", i+1, err)
		}
		result := IsGoogleEndpoint(*u)
		if testCase.result != result {
			t.Errorf("Test %d: Expected isGoogleEndpoint to be '%v' for input \"%s\", but found it to be '%v' instead", i+1, testCase.result, testCase.url, result)
		}
	}
}

func TestPercentEncodeSlash(t *testing.T) {
	testCases := []struct {
		input  string
		output string
	}{
		{"test123", "test123"},
		{"abc,+_1", "abc,+_1"},
		{"%40prefix=test%40123", "%40prefix=test%40123"},
		{"key1=val1/val2", "key1=val1%2Fval2"},
		{"%40prefix=test%40123/", "%40prefix=test%40123%2F"},
	}

	for i, testCase := range testCases {
		receivedOutput := percentEncodeSlash(testCase.input)
		if testCase.output != receivedOutput {
			t.Errorf(
				"Test %d: Input: \"%s\" --> Expected percentEncodeSlash to return \"%s\", but it returned \"%s\" instead!",
				i+1, testCase.input, testCase.output,
				receivedOutput,
			)
		}
	}
}

// Tests validate the query encoder.
func TestQueryEncode(t *testing.T) {
	testCases := []struct {
		queryKey      string
		valueToEncode []string
		// Expected result.
		result string
	}{
		{"prefix", []string{"test@123", "test@456"}, "prefix=test%40123&prefix=test%40456"},
		{"@prefix", []string{"test@123"}, "%40prefix=test%40123"},
		{"@prefix", []string{"a/b/c/"}, "%40prefix=a%2Fb%2Fc%2F"},
		{"prefix", []string{"test#123"}, "prefix=test%23123"},
		{"prefix#", []string{"test#123"}, "prefix%23=test%23123"},
		{"prefix", []string{"test123"}, "prefix=test123"},
		{"prefix", []string{"test本語123", "test123"}, "prefix=test%E6%9C%AC%E8%AA%9E123&prefix=test123"},
	}

	for i, testCase := range testCases {
		urlValues := make(url.Values)
		for _, valueToEncode := range testCase.valueToEncode {
			urlValues.Add(testCase.queryKey, valueToEncode)
		}
		result := QueryEncode(urlValues)
		if testCase.result != result {
			t.Errorf("Test %d: Expected queryEncode result to be \"%s\", but found it to be \"%s\" instead", i+1, testCase.result, result)
		}
	}
}

// Tests tag decode to map
func TestTagDecode(t *testing.T) {
	testCases := []struct {
		// canonical input
		canonicalInput string

		// Expected result.
		resultMap map[string]string
	}{
		{"k=thisisthe%25url", map[string]string{"k": "thisisthe%url"}},
		{"k=%E6%9C%AC%E8%AA%9E", map[string]string{"k": "本語"}},
		{"k=%E6%9C%AC%E8%AA%9E.1", map[string]string{"k": "本語.1"}},
		{"k=%3E123", map[string]string{"k": ">123"}},
		{"k=myurl%23link", map[string]string{"k": "myurl#link"}},
		{"k=space%20in%20url", map[string]string{"k": "space in url"}},
		{"k=url%2Bpath", map[string]string{"k": "url+path"}},
		{"k=url%2Fpath", map[string]string{"k": "url/path"}},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run("", func(t *testing.T) {
			gotResult := TagDecode(testCase.canonicalInput)
			if !reflect.DeepEqual(testCase.resultMap, gotResult) {
				t.Errorf("Expected %s, got %s", testCase.resultMap, gotResult)
			}
		})
	}
}

// Tests tag encode function for user tags.
func TestTagEncode(t *testing.T) {
	testCases := []struct {
		// Input.
		inputMap map[string]string
		// Expected result.
		result string
	}{
		{map[string]string{
			"k": "thisisthe%url",
		}, "k=thisisthe%25url"},
		{map[string]string{
			"k": "本語",
		}, "k=%E6%9C%AC%E8%AA%9E"},
		{map[string]string{
			"k": "本語.1",
		}, "k=%E6%9C%AC%E8%AA%9E.1"},
		{map[string]string{
			"k": ">123",
		}, "k=%3E123"},
		{map[string]string{
			"k": "myurl#link",
		}, "k=myurl%23link"},
		{map[string]string{
			"k": "space in url",
		}, "k=space%20in%20url"},
		{map[string]string{
			"k": "url+path",
		}, "k=url%2Bpath"},
		{map[string]string{
			"k": "url/path",
		}, "k=url%2Fpath"},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run("", func(t *testing.T) {
			gotResult := TagEncode(testCase.inputMap)
			if testCase.result != gotResult {
				t.Errorf("Expected %s, got %s", testCase.result, gotResult)
			}
		})
	}
}

// Tests validate the URL path encoder.
func TestEncodePath(t *testing.T) {
	testCases := []struct {
		// Input.
		inputStr string
		// Expected result.
		result string
	}{
		{"thisisthe%url", "thisisthe%25url"},
		{"本語", "%E6%9C%AC%E8%AA%9E"},
		{"本語.1", "%E6%9C%AC%E8%AA%9E.1"},
		{">123", "%3E123"},
		{"myurl#link", "myurl%23link"},
		{"space in url", "space%20in%20url"},
		{"url+path", "url%2Bpath"},
		{"url/path", "url/path"},
	}

	for i, testCase := range testCases {
		result := EncodePath(testCase.inputStr)
		if testCase.result != result {
			t.Errorf("Test %d: Expected queryEncode result to be \"%s\", but found it to be \"%s\" instead", i+1, testCase.result, result)
		}
	}
}

// Tests validate the bucket name validator.
func TestIsValidBucketName(t *testing.T) {
	testCases := []struct {
		// Input.
		bucketName string
		// Expected result.
		err error
		// Flag to indicate whether test should Pass.
		shouldPass bool
	}{
		{".mybucket", errors.New("Bucket name contains invalid characters"), false},
		{"$mybucket", errors.New("Bucket name contains invalid characters"), false},
		{"mybucket-", errors.New("Bucket name contains invalid characters"), false},
		{"my", errors.New("Bucket name cannot be shorter than 3 characters"), false},
		{"", errors.New("Bucket name cannot be empty"), false},
		{"my..bucket", errors.New("Bucket name contains invalid characters"), false},
		{"my.-bucket", errors.New("Bucket name contains invalid characters"), false},
		{"my-.bucket", errors.New("Bucket name contains invalid characters"), false},
		{"192.168.1.168", errors.New("Bucket name cannot be an ip address"), false},
		{":bucketname", errors.New("Bucket name contains invalid characters"), false},
		{"_bucketName", errors.New("Bucket name contains invalid characters"), false},
		{"my.bucket.com", nil, true},
		{"my-bucket", nil, true},
		{"123my-bucket", nil, true},
		{"Mybucket", nil, true},
		{"My_bucket", nil, true},
		{"My:bucket", nil, true},
	}

	for i, testCase := range testCases {
		err := CheckValidBucketName(testCase.bucketName)
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

	}
}

// Tests validate the bucket name validator stricter.
func TestIsValidBucketNameStrict(t *testing.T) {
	testCases := []struct {
		// Input.
		bucketName string
		// Expected result.
		err error
		// Flag to indicate whether test should Pass.
		shouldPass bool
	}{
		{".mybucket", errors.New("Bucket name contains invalid characters"), false},
		{"$mybucket", errors.New("Bucket name contains invalid characters"), false},
		{"mybucket-", errors.New("Bucket name contains invalid characters"), false},
		{"my", errors.New("Bucket name cannot be shorter than 3 characters"), false},
		{"", errors.New("Bucket name cannot be empty"), false},
		{"my..bucket", errors.New("Bucket name contains invalid characters"), false},
		{"my.-bucket", errors.New("Bucket name contains invalid characters"), false},
		{"my-.bucket", errors.New("Bucket name contains invalid characters"), false},
		{"192.168.1.168", errors.New("Bucket name cannot be an ip address"), false},
		{"Mybucket", errors.New("Bucket name contains invalid characters"), false},
		{"my.bucket.com", nil, true},
		{"my-bucket", nil, true},
		{"123my-bucket", nil, true},
	}

	for i, testCase := range testCases {
		err := CheckValidBucketNameStrict(testCase.bucketName)
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

	}
}

func TestTagsTagDecode(t *testing.T) {
	tagsOMap := map[string]string{"A": "B", "C": "D", "D:A": "A", "E": "PA"}
	// putObject - mock mc cp file remote/path --tags="A=B&C=D&D:W=A&E=PA"
	tagStr := TagEncode(tagsOMap)
	// getObject - mock client getObject with tags return
	tagsNMap := TagDecode(tagStr)
	// putObject - mock put file to target
	tagNStr := TagEncode(tagsNMap)
	_, err := tags.ParseObjectTags(tagNStr)
	if err != nil {
		t.Log("we got error here", err)
	}
}

func TestTagsTagDecodeAfter(t *testing.T) {
	tagsOMap := map[string]string{"A": "B", "C": "D", "D:A": "A", "E": "PA"}
	// putObject - mock mc cp file remote/path --tags="A=B&C=D&D:W=A&E=PA"
	tagStr := TagEncode(tagsOMap)
	// getObject - mock client getObject with tags return
	tagsNMap := TagDecodeAfter(tagStr)
	// putObject - mock put file to target
	tagNStr := TagEncode(tagsNMap)
	_, err := tags.ParseObjectTags(tagNStr)
	if err != nil {
		t.Log("we got error here", err)
	}
}
