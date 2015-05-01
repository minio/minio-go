// +build ignore

package main

import (
	"fmt"

	"github.com/minio-io/objectstorage-go"
)

func main() {
	config := new(objectstorage.Config)
	config.Endpoint = "http://localhost:9000"
	m := objectstorage.New(config)
	err := m.PutBucket("testbucket")
	fmt.Println(err)

	err = m.PutBucketACL("testbucket", "public-read")
	fmt.Println(err)

	err = m.PutBucketACL("testbucket", "invalid")
	fmt.Println(err)

	err = m.HeadBucket("testbucket")
	fmt.Println(err)
}
