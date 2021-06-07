module github.com/minio/minio-go/examples/s3

go 1.14

require (
	github.com/cheggaaa/pb v1.0.29
	github.com/minio/minio-go/v7 v7.0.10
	github.com/minio/sio v0.3.0
	golang.org/x/crypto v0.0.0-20210513164829-c07d793c2f9a
)

replace github.com/minio/minio-go/v7 v7.0.10 => ../..
