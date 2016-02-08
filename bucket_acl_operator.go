package minio

type BucketACLOperator interface {
	SetBucketACL(bucketName string, cannedACL BucketACL) error
	GetBucketACL(bucketName string) (BucketACL, error)
}
