package minio

// BucketOperator provides operations on a bucket in the object storage system
type BucketOperator interface {
	// ListBuckets returns a slice of all the buckets in the object stora
	ListBuckets() ([]BucketInfo, error)
	// MakeBucket creates a new bucket with the given name and ACL
	MakeBucket(bucketName string, cannedACL BucketACL, location string) error
	// BucketExists returns nil if the bucket exists, error otherwise
	BucketExists(bucketName string) error
	// RemoveBucket removes the bucket with the given name. Any non-nil error means the bucket was not removed
	RemoveBucket(bucketName string) error
}
