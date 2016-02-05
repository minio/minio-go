package minio

import (
	"io"
)

// ObjectOperator provides operations to interact with objects in the object store
type ObjectOperator interface {
	// ListObjects gets the object list and streams each object back in the returned channel. Objects may not be in order
	ListObjects(bucket, prefix string, recursive bool, doneCh <-chan struct{}) <-chan ObjectInfo
	// GetObject gets the specific object with the given name and returns it
	GetObject(bucketName, objectName string) (reader *Object, err error)
	// PutObject reads data from reader and puts it into bucketName/objectName
	PutObject(bucketName, objectName string, reader io.Reader, contentType string) (n int64, err error)
	// StatObject returns a valid ObjectInfo if the given bucketName/objectName exists, and an error otherwise
	StatObject(bucketName, objectName string) (ObjectInfo, error)
	// RemoveObject removes the object at bucketName/objectName. Any non-nil error means the object was not successfully removed
	RemoveObject(bucketName, objectName string) error
}
