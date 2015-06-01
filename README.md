# Minimal object storage library in Go [![Build Status](https://travis-ci.org/minio/minio-go.svg)](https://travis-ci.org/minio/minio-go)

## Install

```sh
$ go get github.com/minio/minio-go
```

## API

### Bucket

~~~
 MakeBucket(bucket string) error
 ListBuckets(bucket string) (<-chan BucketStatCh{BucketStat, error})
 BucketExists(bucket string) error
 RemoveBucket(bucket string) error
 GetBucketACL(bucket string) (BucketACL, error)
 SetBucketACL(bucket, cannedACL BucketACL) error
 DropAllIncompleteUploads(bucket string) error
~~~

### Object

~~~
 GetObject(bucket, key string) (io.ReadCloser, ObjectStat, error)
 PutObject(bucket, key string) error
 ListObjects(bucket, prefix string, recursive bool) <-chan ObjectStatCh{ObjectStat, error}
 StatObject(bucket, key string) (ObjectStat, error)
 RemoveObject(bucket, key string) error
 DropIncompleteUpload(bucket, key string) error
~~~

### Error

~~~
 type ErrorResponse struct {
      XMLName   xml.Name `xml:"Error" json:"-"`
      Code      string
      Message   string
      Resource  string
      RequestID string `xml:"RequestId"`
      HostID    string `xml:"HostId"`
 }

 func (e *ErrorResponse) Error() string {return str}
 func (e *ErrorResponse) XML() string {return str}
~~~

## Documentation

More detailed documentation for minimal object storage library http://godoc.org/github.com/minio/minio-go

## Join The Community
* Community hangout on Gitter    [![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/minio/minio?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
* Ask questions on Quora  [![Quora](http://upload.wikimedia.org/wikipedia/commons/thumb/5/57/Quora_logo.svg/55px-Quora_logo.svg.png)](http://www.quora.com/Minio)

## Contribute

[Contributors Guide](./CONTRIBUTING.md)
