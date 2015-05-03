// +build ignore

package main

import (
	"fmt"

	"github.com/minio-io/objectstorage-go"
)

func main() {
	config := new(objectstorage.Config)
	config.Endpoint = "https://s3.amazonaws.com"
	config.AccessKeyID = "AKIAIA3SEGOYCMTCTF4A"
	config.SecretAccessKey = "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"
	m := objectstorage.New(config)

	err := m.PutBucket("testbucket")
	fmt.Println(err)

	err = m.PutBucketACL("testbucket", "public-read")
	fmt.Println(err)

	err = m.PutBucketACL("testbucket", "invalid")
	fmt.Println(err)

	err = m.HeadBucket("testbucket")
	fmt.Println(err)

	_, err = m.ListBuckets()
	fmt.Println(err)
}
