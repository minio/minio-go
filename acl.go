package client

// BucketACL - bucket level access control
type BucketACL string

// different types of ACL's currently supported for buckets
const (
	BucketPrivate       = BucketACL("private")
	BucketReadOnly      = BucketACL("public-read")
	BucketPublic        = BucketACL("public-read-write")
	BucketAuthenticated = BucketACL("authenticated-read")
)
