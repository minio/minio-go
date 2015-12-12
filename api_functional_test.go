package minio_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/minio/minio-go"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyz01234569"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func randString(n int, src rand.Source) string {
	b := make([]byte, n)
	// A rand.Int63() generates 63 random bits, enough for letterIdxMax letters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}
	return string(b[0:30])
}

func TestFunctional(t *testing.T) {
	a, err := minio.New("play.minio.io:9002",
		"Q3AM3UQ867SPQQA43P2F", "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG", false)
	if err != nil {
		t.Fatal("Error:", err)
	}

	bucketName := randString(60, rand.NewSource(time.Now().UnixNano()))
	err = a.MakeBucket(bucketName, "private", "us-east-1")
	if err != nil {
		t.Fatal("Error:", err, bucketName)
	}

	err = a.BucketExists(bucketName)
	if err != nil {
		t.Fatal("Error:", err, bucketName)
	}

	err = a.SetBucketACL(bucketName, "public-read-write")
	if err != nil {
		t.Fatal("Error:", err)
	}

	acl, err := a.GetBucketACL(bucketName)
	if err != nil {
		t.Fatal("Error:", err)
	}
	if acl != minio.BucketACL("public-read-write") {
		t.Fatal("Error:", acl)
	}

	for b := range a.ListBuckets() {
		if b.Err != nil {
			t.Fatal("Error:", b.Err)
		}
	}

	err = a.RemoveBucket(bucketName)
	if err != nil {
		t.Fatal("Error:", err)
	}

	err = a.RemoveBucket("bucket1")
	if err == nil {
		t.Fatal("Error:")
	}

	if err.Error() != "The specified bucket does not exist." {
		t.Fatal("Error:", err)
	}

}
