/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2026 MinIO, Inc.
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
	"reflect"
	"strings"
	"testing"
)

func TestGetAmzGrantACL(t *testing.T) {
	const body = `<AccessControlPolicy><Owner><ID>owner</ID></Owner><AccessControlList>` +
		`<Grant><Grantee><ID>owner</ID></Grantee><Permission>FULL_CONTROL</Permission></Grant>` +
		`<Grant><Grantee><URI>http://acs.amazonaws.com/groups/global/AllUsers</URI></Grantee><Permission>READ</Permission></Grant>` +
		`<Grant><Grantee><URI>http://acs.amazonaws.com/groups/global/AuthenticatedUsers</URI></Grantee><Permission>READ</Permission></Grant>` +
		`<Grant><Grantee><ID>user1</ID></Grantee><Permission>WRITE</Permission></Grant>` +
		`<Grant><Grantee><URI>http://acs.amazonaws.com/groups/global/LogDelivery</URI></Grantee><Permission>WRITE_ACP</Permission></Grant>` +
		`<Grant><Grantee><ID>user2</ID></Grantee><Permission>READ_ACP</Permission></Grant>` +
		`</AccessControlList></AccessControlPolicy>`

	res := &accessControlPolicy{}
	if err := xmlDecoder(strings.NewReader(body), res); err != nil {
		t.Fatal(err)
	}

	want := map[string][]string{
		"X-Amz-Grant-Full-Control": {"id=owner"},
		"X-Amz-Grant-Read": {
			"uri=http://acs.amazonaws.com/groups/global/AllUsers",
			"uri=http://acs.amazonaws.com/groups/global/AuthenticatedUsers",
		},
		"X-Amz-Grant-Write":     {"id=user1"},
		"X-Amz-Grant-Write-Acp": {"uri=http://acs.amazonaws.com/groups/global/LogDelivery"},
		"X-Amz-Grant-Read-Acp":  {"id=user2"},
	}
	if got := getAmzGrantACL(res); !reflect.DeepEqual(got, want) {
		t.Fatalf("getAmzGrantACL() = %v, want %v", got, want)
	}
}
