// +build ignore

package main

import (
	"fmt"
	"log"

	"github.com/minio-io/objectstorage-go"
)

func main() {
	config := new(objectstorage.Config)
	config.Endpoint = "https://s3.amazonaws.com"
	config.AccessKeyID = ""
	config.SecretAccessKey = ""
	config.UserAgent = "Minio"
	m := objectstorage.New(config)

	err := m.PutBucket("testbucket")
	if err != nil {
		log.Println(err)
	}

	err = m.PutBucketACL("testbucket", "public-read")
	if err != nil {
		log.Println(err)
	}

	err = m.PutBucketACL("testbucket", "invalid")
	if err != nil {
		log.Println(err)
	}

	err = m.HeadBucket("testbucket")
	if err != nil {
		log.Println(err)
	}

	listBuckets, err := m.ListBuckets()
	if err != nil {
		log.Println(err)
	}
	if err == nil {
		buckets := listBuckets.Buckets
		for _, bucket := range buckets.Bucket {
			fmt.Println(bucket)
		}
	}
}
