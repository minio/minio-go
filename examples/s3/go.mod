module github.com/minio/minio-go/examples/s3

go 1.17

require (
	github.com/cheggaaa/pb v1.0.29
	github.com/minio/minio-go/v7 v7.0.49
	github.com/minio/sio v0.3.0
	golang.org/x/crypto v0.9.0
)

require (
	github.com/klauspost/compress v1.16.5 // indirect
	github.com/klauspost/cpuid/v2 v2.2.4 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/rivo/uniseg v0.4.4 // indirect
)

replace github.com/minio/minio-go/v7 v7.0.49 => ../..
