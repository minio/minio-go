module github.com/minio/minio-go/examples/s3

go 1.14

require (
	github.com/cheggaaa/pb v1.0.29
	github.com/google/uuid v1.3.0 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/minio/minio-go/v7 v7.0.32
	github.com/minio/sha256-simd v1.0.0 // indirect
	github.com/minio/sio v0.3.0
	github.com/rs/xid v1.4.0 // indirect
	github.com/sirupsen/logrus v1.9.0 // indirect
	golang.org/x/crypto v0.6.0
)

replace github.com/minio/minio-go/v7 v7.0.32 => ../..
