package minio

// FileOperator provides functionality to do operations between local files and the remote object store
type FileOperator interface {
	// FPutObject transfers the data at filePath and the object at bucketName/objectName. Returns the number of bytes transferred, or an error if the file doesn't exist, the network I/O to the object storage system failed, or any other failure occurred
	FPutObject(bucketName, objectName, filePath, contentType string) (n int64, err error)
	// FGetObject gets the object at bucketName/objectName and writes it to filePath
	FGetObject(bucketName, objectName, filePath string) error
}
