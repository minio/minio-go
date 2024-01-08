# MinIO Go Client API Reference [![Slack](https://slack.min.io/slack?type=svg)](https://slack.min.io)

## Initialize MinIO Client object.

##  MinIO

```go
package main

import (
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func main() {
	endpoint := "play.min.io"
	accessKeyID := "Q3AM3UQ867SPQQA43P2F"
	secretAccessKey := "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG"
	useSSL := true

	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("%#v\n", minioClient) // minioClient is now setup
}
```

## AWS S3

```go
package main

import (
    "fmt"

    "github.com/minio/minio-go/v7"
    "github.com/minio/minio-go/v7/pkg/credentials"
)

func main() {
        // Initialize minio client object.
        s3Client, err := minio.New("s3.amazonaws.com", &minio.Options{
   	        Creds:  credentials.NewStaticV4("YOUR-ACCESSKEYID", "YOUR-SECRETACCESSKEY", ""),
	        Secure: true,
        })
        if err != nil {
                fmt.Println(err)
                return
        }
}
```

| Bucket operations                                     | Object operations                                   | Presigned operations                          | Bucket Policy/Notification Operations                         | Client custom settings                                |
| :---                                                  | :---                                                | :---                                          | :---                                                          | :---                                                  |
| [`MakeBucket`](#MakeBucket)                           | [`GetObject`](#GetObject)                           | [`PresignedGetObject`](#PresignedGetObject)   | [`SetBucketPolicy`](#SetBucketPolicy)                         | [`SetAppInfo`](#SetAppInfo)                           |
|                                                       | [`PutObject`](#PutObject)                           | [`PresignedPutObject`](#PresignedPutObject)   | [`GetBucketPolicy`](#GetBucketPolicy)                         |                                                       |
| [`ListBuckets`](#ListBuckets)                         | [`CopyObject`](#CopyObject)                         | [`PresignedHeadObject`](#PresignedHeadObject) | [`SetBucketNotification`](#SetBucketNotification)             | [`TraceOn`](#TraceOn)                                 |
| [`BucketExists`](#BucketExists)                       | [`StatObject`](#StatObject)                         | [`PresignedPostPolicy`](#PresignedPostPolicy) | [`GetBucketNotification`](#GetBucketNotification)             | [`TraceOff`](#TraceOff)                               |
| [`RemoveBucket`](#RemoveBucket)                       | [`RemoveObject`](#RemoveObject)                     |                                               | [`RemoveAllBucketNotification`](#RemoveAllBucketNotification) | [`SetS3TransferAccelerate`](#SetS3TransferAccelerate) |
| [`ListObjects`](#ListObjects)                         | [`RemoveObjects`](#RemoveObjects)                   |                                               | [`ListenBucketNotification`](#ListenBucketNotification)       |                                                       |
|                                                       | [`RemoveIncompleteUpload`](#RemoveIncompleteUpload) |                                               | [`SetBucketLifecycle`](#SetBucketLifecycle)                   |                                                       |
| [`ListIncompleteUploads`](#ListIncompleteUploads)     | [`FPutObject`](#FPutObject)                         |                                               | [`GetBucketLifecycle`](#GetBucketLifecycle)                   |                                                       |
| [`SetBucketTagging`](#SetBucketTagging)               | [`FGetObject`](#FGetObject)                         |                                               | [`SetObjectLockConfig`](#SetObjectLockConfig)                 |                                                       |
| [`GetBucketTagging`](#GetBucketTagging)               | [`ComposeObject`](#ComposeObject)                   |                                               | [`GetObjectLockConfig`](#GetObjectLockConfig)                 |                                                       |
| [`RemoveBucketTagging`](#RemoveBucketTagging)         |                                                     |                                               | [`EnableVersioning`](#EnableVersioning)                       |                                                       |
| [`SetBucketReplication`](#SetBucketReplication)       |                                                     |                                               | [`DisableVersioning`](#DisableVersioning)                     |                                                       |
| [`GetBucketReplication`](#GetBucketReplication)       | [`PutObjectRetention`](#PutObjectRetention)         |                                               | [`GetBucketEncryption`](#GetBucketEncryption)                 |                                                       |
| [`RemoveBucketReplication`](#RemoveBucketReplication) | [`GetObjectRetention`](#GetObjectRetention)         |                                               | [`RemoveBucketEncryption`](#RemoveBucketEncryption)           |                                                       |
|                                                       | [`PutObjectLegalHold`](#PutObjectLegalHold)         |                                               |                                                               |                                                       |
|                                                       | [`GetObjectLegalHold`](#GetObjectLegalHold)         |                                               |                                                               |                                                       |
|                                                       | [`SelectObjectContent`](#SelectObjectContent)       |                                               |                                                               |                                                       |
|                                                       | [`PutObjectTagging`](#PutObjectTagging)             |                                               |                                                               |                                                       |
|                                                       | [`GetObjectTagging`](#GetObjectTagging)             |                                               |                                                               |                                                       |
|                                                       | [`RemoveObjectTagging`](#RemoveObjectTagging)       |                                               |                                                               |                                                       |
|                                                       | [`RestoreObject`](#RestoreObject)                   |                                               |                                                               |                                                       |
|                                                       | [`GetObjectAttributes`](#GetObjectAttributes)                   |                                               |                                                               |                                                       |

## 1. Constructor
<a name="MinIO"></a>

### New(endpoint string, opts \*Options) (\*Client, error)
Initializes a new client object.

__Parameters__

| Param      | Type            | Description                           |
|:-----------|:----------------|:--------------------------------------|
| `endpoint` | _string_        | S3 compatible object storage endpoint |
| `opts`     | _minio.Options_ | Options for constructing a new client |

__minio.Options__

| Field               | Type                       | Description                                                                  |
|:--------------------|:---------------------------|:-----------------------------------------------------------------------------|
| `opts.Creds`        | _*credentials.Credentials_ | S3 compatible object storage access credentials                              |
| `opts.Secure`       | _bool_                     | If 'true' API requests will be secure (HTTPS), and insecure (HTTP) otherwise |
| `opts.Transport`    | _http.RoundTripper_        | Custom transport for executing HTTP transactions                             |
| `opts.Region`       | _string_                   | S3 compatible object storage region                                          |
| `opts.BucketLookup` | _BucketLookupType_         | Bucket lookup type can be one of the following values                        |
|                     |                            | _minio.BucketLookupDNS_                                                      |
|                     |                            | _minio.BucketLookupPath_                                                     |
|                     |                            | _minio.BucketLookupAuto_                                                     |

## 2. Bucket operations
<a name="MakeBucket"></a>

### MakeBucket(ctx context.Context, bucketName string, opts MakeBucketOptions)
Creates a new bucket.

__Parameters__

| Param        | Type                      | Description                                                                                                                                                                                                        |
|--------------|---------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `ctx`        | _context.Context_         | Custom context for timeout/cancellation of the call                                                                                                                                                                |
| `bucketName` | _string_                  | Name of the bucket                                                                                                                                                                                                 |
| `opts`       | _minio.MakeBucketOptions_ | Bucket options such as `Region` where the bucket is to be created. Default value is us-east-1. Other valid values are listed below. Note: When used with minio server, use the region specified in its config file (defaults to us-east-1). |
|              |                           | us-east-1                                                                                                                                                                                                          |
|              |                           | us-east-2                                                                                                                                                                                                          |
|              |                           | us-west-1                                                                                                                                                                                                          |
|              |                           | us-west-2                                                                                                                                                                                                          |
|              |                           | ca-central-1                                                                                                                                                                                                       |
|              |                           | eu-west-1                                                                                                                                                                                                          |
|              |                           | eu-west-2                                                                                                                                                                                                          |
|              |                           | eu-west-3                                                                                                                                                                                                          |
|              |                           | eu-central-1                                                                                                                                                                                                       |
|              |                           | eu-north-1                                                                                                                                                                                                         |
|              |                           | ap-east-1                                                                                                                                                                                                          |
|              |                           | ap-south-1                                                                                                                                                                                                         |
|              |                           | ap-southeast-1                                                                                                                                                                                                     |
|              |                           | ap-southeast-2                                                                                                                                                                                                     |
|              |                           | ap-northeast-1                                                                                                                                                                                                     |
|              |                           | ap-northeast-2                                                                                                                                                                                                     |
|              |                           | ap-northeast-3                                                                                                                                                                                                     |
|              |                           | me-south-1                                                                                                                                                                                                         |
|              |                           | sa-east-1                                                                                                                                                                                                          |
|              |                           | us-gov-west-1                                                                                                                                                                                                      |
|              |                           | us-gov-east-1                                                                                                                                                                                                      |
|              |                           | cn-north-1                                                                                                                                                                                                         |
|              |                           | cn-northwest-1                                                                                                                                                                                                     |


__Example__


```go
// Create a bucket at region 'us-east-1' with object locking enabled.
err = minioClient.MakeBucket(context.Background(), "mybucket", minio.MakeBucketOptions{Region: "us-east-1", ObjectLocking: true})
if err != nil {
    fmt.Println(err)
    return
}
fmt.Println("Successfully created mybucket.")
```

<a name="ListBuckets"></a>
### ListBuckets(ctx context.Context) ([]BucketInfo, error)
Lists all buckets.

| Param  | Type  | Description  |
|---|---|---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketList`  | _[]minio.BucketInfo_  | Lists of all buckets |


__minio.BucketInfo__

| Field  | Type  | Description  |
|---|---|---|
|`bucket.Name`  | _string_  | Name of the bucket |
|`bucket.CreationDate`  | _time.Time_  | Date of bucket creation |


__Example__


```go
buckets, err := minioClient.ListBuckets(context.Background())
if err != nil {
    fmt.Println(err)
    return
}
for _, bucket := range buckets {
    fmt.Println(bucket)
}
```

<a name="BucketExists"></a>
### BucketExists(ctx context.Context, bucketName string) (found bool, err error)
Checks if a bucket exists.

__Parameters__


|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket |


__Return Values__

|Param   |Type   |Description   |
|:---|:---| :---|
|`found`  | _bool_ | Indicates whether bucket exists or not  |
|`err` | _error_  | Standard Error  |


__Example__


```go
found, err := minioClient.BucketExists(context.Background(), "mybucket")
if err != nil {
    fmt.Println(err)
    return
}
if found {
    fmt.Println("Bucket found")
}
```

<a name="RemoveBucket"></a>
### RemoveBucket(ctx context.Context, bucketName string) error
Removes a bucket, bucket should be empty to be successfully removed.

__Parameters__


|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket   |

__Example__


```go
err = minioClient.RemoveBucket(context.Background(), "mybucket")
if err != nil {
    fmt.Println(err)
    return
}
```

<a name="ListObjects"></a>
### ListObjects(ctx context.Context, bucketName string, opts ListObjectsOptions) <-chan ObjectInfo
Lists objects in a bucket.

__Parameters__


| Param        | Type                       | Description                                         |
|:-------------|:---------------------------|:----------------------------------------------------|
| `ctx`        | _context.Context_          | Custom context for timeout/cancellation of the call |
| `bucketName` | _string_                   | Name of the bucket                                  |
| `opts`       | _minio.ListObjectsOptions_ | Options per to list objects                    |


__Return Value__

|Param   |Type   |Description   |
|:---|:---| :---|
|`objectInfo`  | _chan minio.ObjectInfo_ |Read channel for all objects in the bucket, the object is of the format listed below: |

__minio.ObjectInfo__

|Field   |Type   |Description   |
|:---|:---| :---|
|`objectInfo.Key`  | _string_ |Name of the object |
|`objectInfo.Size`  | _int64_ |Size of the object |
|`objectInfo.ETag`  | _string_ |MD5 checksum of the object |
|`objectInfo.LastModified`  | _time.Time_ |Time when object was last modified |


```go
ctx, cancel := context.WithCancel(context.Background())

defer cancel()

objectCh := minioClient.ListObjects(ctx, "mybucket", minio.ListObjectsOptions{
       Prefix: "myprefix",
       Recursive: true,
})
for object := range objectCh {
    if object.Err != nil {
        fmt.Println(object.Err)
        return
    }
    fmt.Println(object)
}
```


<a name="ListIncompleteUploads"></a>
### ListIncompleteUploads(ctx context.Context, bucketName, prefix string, recursive bool) <- chan ObjectMultipartInfo
Lists partially uploaded objects in a bucket.


__Parameters__


| Param        | Type              | Description                                                                                              |
|:-------------|:------------------|:---------------------------------------------------------------------------------------------------------|
| `ctx`        | _context.Context_ | Custom context for timeout/cancellation of the call                                                      |
| `bucketName` | _string_          | Name of the bucket                                                                                       |
| `prefix`     | _string_          | Prefix of objects that are partially uploaded                                                            |
| `recursive`  | _bool_            | `true` indicates recursive style listing and `false` indicates directory style listing delimited by '/'. |


__Return Value__

|Param   |Type   |Description   |
|:---|:---| :---|
|`multiPartInfo`  | _chan minio.ObjectMultipartInfo_  |Emits multipart objects of the format listed below: |

__minio.ObjectMultipartInfo__

|Field   |Type   |Description   |
|:---|:---| :---|
|`multiPartObjInfo.Key`  | _string_  |Name of incompletely uploaded object |
|`multiPartObjInfo.UploadID` | _string_ |Upload ID of incompletely uploaded object |
|`multiPartObjInfo.Size` | _int64_ |Size of incompletely uploaded object |

__Example__


```go
isRecursive := true // Recursively list everything at 'myprefix'
multiPartObjectCh := minioClient.ListIncompleteUploads(context.Background(), "mybucket", "myprefix", isRecursive)
for multiPartObject := range multiPartObjectCh {
    if multiPartObject.Err != nil {
        fmt.Println(multiPartObject.Err)
        return
    }
    fmt.Println(multiPartObject)
}
```

<a name="SetBucketTagging"></a>
### SetBucketTagging(ctx context.Context, bucketName string, tags *tags.Tags) error
Sets tags to a bucket.


__Parameters__

| Param        | Type              | Description                                         |
|:-------------|:------------------|:----------------------------------------------------|
| `ctx`        | _context.Context_ | Custom context for timeout/cancellation of the call |
| `bucketName` | _string_          | Name of the bucket                                  |
| `tags`       | _*tags.Tags_      | Bucket tags                                         |

__Example__
```go
// Create tags from a map.
tags, err := tags.NewTags(map[string]string{
	"Tag1": "Value1",
	"Tag2": "Value2",
}, false)
if err != nil {
	log.Fatalln(err)
}

err = minioClient.SetBucketTagging(context.Background(), "my-bucketname", tags)
if err != nil {
	log.Fatalln(err)
}
```

<a name="GetBucketTagging"></a>
### GetBucketTagging(ctx context.Context, bucketName string) (*tags.Tags, error)
Gets tags of a bucket.


__Parameters__

| Param        | Type         | Description        |
|:-------------|:-------------|:-------------------|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
| `bucketName` | _string_     | Name of the bucket |

__Return Value__

| Param  | Type         | Description |
|:-------|:-------------|:------------|
| `tags` | _*tags.Tags_ | Bucket tags |

__Example__
```go
tags, err := minioClient.GetBucketTagging(context.Background(), "my-bucketname")
if err != nil {
	log.Fatalln(err)
}

fmt.Printf("Fetched Object Tags: %v\n", tags)
```

<a name="RemoveBucketTagging"></a>
### RemoveBucketTagging(ctx context.Context, bucketName string) error
Removes all tags on a bucket.


__Parameters__

| Param        | Type         | Description        |
|:-------------|:-------------|:-------------------|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
| `bucketName` | _string_     | Name of the bucket |

__Example__
```go
err := minioClient.RemoveBucketTagging(context.Background(), "my-bucketname")
if err != nil {
	log.Fatalln(err)
}
```

## 3. Object operations

<a name="GetObject"></a>
### GetObject(ctx context.Context, bucketName, objectName string, opts GetObjectOptions) (*Object, error)
Returns a stream of the object data. Most of the common errors occur when reading the stream.


__Parameters__


|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket  |
|`objectName` | _string_  |Name of the object  |
|`opts` | _minio.GetObjectOptions_ | Options for GET requests specifying additional options like encryption, If-Match |


__minio.GetObjectOptions__

|Field | Type | Description |
|:---|:---|:---|
| `opts.ServerSideEncryption` | _encrypt.ServerSide_ | Interface provided by `encrypt` package to specify server-side-encryption. (For more information see https://godoc.org/github.com/minio/minio-go/v7) |
| `opts.Internal`                | _minio.AdvancedGetOptions_               | This option is intended for internal use by MinIO server. This option should not be set unless the application is aware of intended use.

__Return Value__

|Param   |Type   |Description   |
|:---|:---| :---|
|`object`  | _*minio.Object_ |_minio.Object_ represents object reader. It implements io.Reader, io.Seeker, io.ReaderAt and io.Closer interfaces. |


__Example__


```go
object, err := minioClient.GetObject(context.Background(), "mybucket", "myobject", minio.GetObjectOptions{})
if err != nil {
    fmt.Println(err)
    return
}
defer object.Close()

localFile, err := os.Create("/tmp/local-file.jpg")
if err != nil {
    fmt.Println(err)
    return
}
defer localFile.Close()

if _, err = io.Copy(localFile, object); err != nil {
    fmt.Println(err)
    return
}
```

<a name="FGetObject"></a>
### FGetObject(ctx context.Context, bucketName, objectName, filePath string, opts GetObjectOptions) error
Downloads and saves the object as a file in the local filesystem.

__Parameters__


|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket |
|`objectName` | _string_  |Name of the object  |
|`filePath` | _string_  |Path to download object to |
|`opts` | _minio.GetObjectOptions_ | Options for GET requests specifying additional options like encryption, If-Match |


__Example__


```go
err = minioClient.FGetObject(context.Background(), "mybucket", "myobject", "/tmp/myobject", minio.GetObjectOptions{})
if err != nil {
    fmt.Println(err)
    return
}
```

<a name="PutObjectFanOut"></a>
### PutObjectFanOut(ctx context.Context, bucket string, body io.Reader, fanOutReq ...PutObjectFanOutRequest) ([]PutObjectFanOutResponse, error)
A variant of PutObject instead of writing a single object from a single stream multiple objects are written, defined via a list of 
*PutObjectFanOutRequest*. Each entry in *PutObjectFanOutRequest* carries an object keyname and its relevant metadata if any. 
`Key` is mandatory, rest of the other options in *PutObjectFanOutRequest( are optional.

__Parameters__

| Param        | Type                           | Description                                                           |
|:-------------|:-------------------------------|:----------------------------------------------------------------------|
| `ctx`        | _context.Context_              | Custom context for timeout/cancellation of the call                   |
| `bucketName` | _string_                       | Name of the bucket                                                    |
| `fanOutData` | _io.Reader_                    | Any Go type that implements io.Reader                                 |
| `fanOutReq`  | _minio.PutObjectFanOutRequest_ | User input list of all the objects that will be created on the server |
|              |                                |                                                                       |

__minio.PutObjectFanOutRequest__

| Field       | Type                            | Description                                |
|:------------|:--------------------------------|:-------------------------------------------|
| `Entries`   | _[]minio.PutObjectFanOutEntyry_ | List of object fan out entries             |
| `Checksums` | _map[string]string_             | Checksums for the input data               |
| `SSE`       | _encrypt.ServerSide             | Encryption settings for the entire fan-out |

__minio.PutObjectFanOutEntry__

| Field                | Type                  | Description                                                                                        |
|:---------------------|:----------------------|:---------------------------------------------------------------------------------------------------|
| `Key`                | _string_              | Name of the object                                                                                 |
| `UserMetadata`       | _map[string]string_   | Map of user metadata                                                                               |
| `UserTags`           | _map[string]string_   | Map of user object tags                                                                            |
| `ContentType`        | _string_              | Content type of object, e.g "application/text"                                                     |
| `ContentEncoding`    | _string_              | Content encoding of object, e.g "gzip"                                                             |
| `ContentDisposition` | _string_              | Content disposition of object, "inline"                                                            |
| `ContentLanguage`    | _string_              | Content language of object, e.g "French"                                                           |
| `CacheControl`       | _string_              | Used to specify directives for caching mechanisms in both requests and responses e.g "max-age=600" |
| `Retention`          | _minio.RetentionMode_ | Retention mode to be set, e.g "COMPLIANCE"                                                         |
| `RetainUntilDate`    | _time.Time_           | Time until which the retention applied is valid                                                    |

__minio.PutObjectFanOutResponse__

| Field          | Type       | Description                                                     |
|:---------------|:-----------|:----------------------------------------------------------------|
| `Key`          | _string_   | Name of the object                                              |
| `ETag`         | _string_   | ETag opaque unique value of the object                          |
| `VersionID`    | _string_   | VersionID of the uploaded object                                |
| `LastModified` | _time.Time | Last modified time of the latest object                         |
| `Error`        | _error_    | Is non `nil` only when the fan-out for a specific object failed |

<a name="PutObject"></a>
### PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64,opts PutObjectOptions) (info UploadInfo, err error)
Uploads objects that are less than 128MiB in a single PUT operation. For objects that are greater than 128MiB in size, PutObject seamlessly uploads the object as parts of 128MiB or more depending on the actual file size. The max upload size for an object is 5TB.

__Parameters__

|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket  |
|`objectName` | _string_  |Name of the object   |
|`reader` | _io.Reader_  |Any Go type that implements io.Reader |
|`objectSize`| _int64_ |Size of the object being uploaded. Pass -1 if stream size is unknown (Warning: passing -1 will allocate a large amount of memory) |
|`opts` | _minio.PutObjectOptions_  | Allows user to set optional custom metadata, content headers, encryption keys and number of threads for multipart upload operation. |

__minio.PutObjectOptions__

| Field                          | Type                   | Description                                                                                                                                                                        |
|:-------------------------------|:-----------------------|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `opts.UserMetadata`            | _map[string]string_    | Map of user metadata                                                                                                                                                               |
| `opts.UserTags`                | _map[string]string_    | Map of user object tags                                                                                                                                                            |
| `opts.Progress`                | _io.Reader_            | Reader to fetch progress of an upload                                                                                                                                              |
| `opts.ContentType`             | _string_               | Content type of object, e.g "application/text"                                                                                                                                     |
| `opts.ContentEncoding`         | _string_               | Content encoding of object, e.g "gzip"                                                                                                                                             |
| `opts.ContentDisposition`      | _string_               | Content disposition of object, "inline"                                                                                                                                            |
| `opts.ContentLanguage`         | _string_               | Content language of object, e.g "French"                                                                                                                                           |
| `opts.CacheControl`            | _string_               | Used to specify directives for caching mechanisms in both requests and responses e.g "max-age=600"                                                                                 |
| `opts.Mode`                    | _*minio.RetentionMode_ | Retention mode to be set, e.g "COMPLIANCE"                                                                                                                                         |
| `opts.RetainUntilDate`         | _*time.Time_           | Time until which the retention applied is valid                                                                                                                                    |
| `opts.ServerSideEncryption`    | _encrypt.ServerSide_   | Interface provided by `encrypt` package to specify server-side-encryption. (For more information see https://godoc.org/github.com/minio/minio-go/v7)                               |
| `opts.StorageClass`            | _string_               | Specify storage class for the object. Supported values for MinIO server are `REDUCED_REDUNDANCY` and `STANDARD`                                                                    |
| `opts.WebsiteRedirectLocation` | _string_               | Specify a redirect for the object, to another object in the same bucket or to a external URL.                                                                                      |
| `opts.SendContentMd5`          | _bool_                 | Specify if you'd like to send `content-md5` header with PutObject operation. Note that setting this flag will cause higher memory usage because of in-memory `md5sum` calculation. |
| `opts.PartSize`                | _uint64_               | Specify a custom part size used for uploading the object                                                                                                                           |
| `opts.Internal`                | _minio.AdvancedPutOptions_ | This option is intended for internal use by MinIO server and should not be set unless the application is aware of intended use.
|
__minio.UploadInfo__

| Field               | Type     | Description                                                                                                                                                                        |
|:--------------------|:---------|:-------------------------------------------|
| `info.ETag`         | _string_ | The ETag of the new object                 |
| `info.VersionID`    | _string_ | The version identifyer of the new object   |


__Example__


```go
file, err := os.Open("my-testfile")
if err != nil {
    fmt.Println(err)
    return
}
defer file.Close()

fileStat, err := file.Stat()
if err != nil {
    fmt.Println(err)
    return
}

uploadInfo, err := minioClient.PutObject(context.Background(), "mybucket", "myobject", file, fileStat.Size(), minio.PutObjectOptions{ContentType:"application/octet-stream"})
if err != nil {
    fmt.Println(err)
    return
}
fmt.Println("Successfully uploaded bytes: ", uploadInfo)
```

API methods PutObjectWithSize, PutObjectWithMetadata, PutObjectStreaming, and PutObjectWithProgress available in minio-go SDK release v3.0.3 are replaced by the new PutObject call variant that accepts a pointer to PutObjectOptions struct.


<a name="CopyObject"></a>
### CopyObject(ctx context.Context, dst CopyDestOptions, src CopySrcOptions) (UploadInfo, error)
Create or replace an object through server-side copying of an existing object. It supports conditional copying, copying a part of an object and server-side encryption of destination and decryption of source. See the `CopySrcOptions` and `DestinationInfo` types for further details.

To copy multiple source objects into a single destination object see the `ComposeObject` API.

__Parameters__

| Param | Type                    | Description                                         |
|:------|:------------------------|:----------------------------------------------------|
| `ctx` | _context.Context_       | Custom context for timeout/cancellation of the call |
| `dst` | _minio.CopyDestOptions_ | Argument describing the destination object          |
| `src` | _minio.CopySrcOptions_  | Argument describing the source object               |


__minio.UploadInfo__

| Field            | Type     | Description                              |
|:-----------------|:---------|:-----------------------------------------|
| `info.ETag`      | _string_ | The ETag of the new object               |
| `info.VersionID` | _string_ | The version identifyer of the new object |


__Example__


```go
// Use-case 1: Simple copy object with no conditions.
// Source object
srcOpts := minio.CopySrcOptions{
    Bucket: "my-sourcebucketname",
    Object: "my-sourceobjectname",
}

// Destination object
dstOpts := minio.CopyDestOptions{
    Bucket: "my-bucketname",
    Object: "my-objectname",
}

// Copy object call
uploadInfo, err := minioClient.CopyObject(context.Background(), dst, src)
if err != nil {
    fmt.Println(err)
    return
}

fmt.Println("Successfully copied object:", uploadInfo)
```

```go
// Use-case 2:
// Copy object with copy-conditions, and copying only part of the source object.
// 1. that matches a given ETag
// 2. and modified after 1st April 2014
// 3. but unmodified since 23rd April 2014
// 4. copy only first 1MiB of object.

// Source object
srcOpts := minio.CopySrcOptions{
    Bucket: "my-sourcebucketname",
    Object: "my-sourceobjectname",
    MatchETag: "31624deb84149d2f8ef9c385918b653a",
    MatchModifiedSince: time.Date(2014, time.April, 1, 0, 0, 0, 0, time.UTC),
    MatchUnmodifiedSince: time.Date(2014, time.April, 23, 0, 0, 0, 0, time.UTC),
    Start: 0,
    End: 1024*1024-1,
}


// Destination object
dstOpts := minio.CopyDestOptions{
    Bucket: "my-bucketname",
    Object: "my-objectname",
}

// Copy object call
_, err = minioClient.CopyObject(context.Background(), dst, src)
if err != nil {
    fmt.Println(err)
    return
}

fmt.Println("Successfully copied object:", uploadInfo)

```

<a name="ComposeObject"></a>
### ComposeObject(ctx context.Context, dst minio.CopyDestOptions, srcs ...minio.CopySrcOptions) (UploadInfo, error)
Create an object by concatenating a list of source objects using server-side copying.

__Parameters__


| Param  | Type                      | Description                                                                 |
|:-------|:--------------------------|:----------------------------------------------------------------------------|
| `ctx`  | _context.Context_         | Custom context for timeout/cancellation of the call                         |
| `dst`  | _minio.CopyDestOptions_   | Struct with info about the object to be created.                            |
| `srcs` | _...minio.CopySrcOptions_ | Slice of struct with info about source objects to be concatenated in order. |


__minio.UploadInfo__

| Field               | Type     | Description                                                                                                                                                                        |
|:--------------------|:---------|:-------------------------------------------|
| `info.ETag`         | _string_ | The ETag of the new object                 |
| `info.VersionID`    | _string_ | The version identifyer of the new object   |


__Example__

```go
// Prepare source decryption key (here we assume same key to
// decrypt all source objects.)
sseSrc := encrypt.DefaultPBKDF([]byte("password"), []byte("salt"))

// Source objects to concatenate. We also specify decryption
// key for each
src1Opts := minio.CopySrcOptions{
    Bucket: "bucket1",
    Object: "object1",
    Encryption: sseSrc,
    MatchETag: "31624deb84149d2f8ef9c385918b653a",
}

src2Opts := minio.CopySrcOptions{
    Bucket: "bucket2",
    Object: "object2",
    Encryption: sseSrc,
    MatchETag: "f8ef9c385918b653a31624deb84149d2",
}

src3Opts := minio.CopySrcOptions{
    Bucket: "bucket3",
    Object: "object3",
    Encryption: sseSrc,
    MatchETag: "5918b653a31624deb84149d2f8ef9c38",
}

// Prepare destination encryption key
sseDst := encrypt.DefaultPBKDF([]byte("new-password"), []byte("new-salt"))

// Create destination info
dstOpts := CopyDestOptions{
    Bucket: "bucket",
    Object: "object",
    Encryption: sseDst,
}

// Compose object call by concatenating multiple source files.
uploadInfo, err := minioClient.ComposeObject(context.Background(), dst, srcs...)
if err != nil {
    fmt.Println(err)
    return
}

fmt.Println("Composed object successfully:", uploadInfo)
```

<a name="FPutObject"></a>
### FPutObject(ctx context.Context, bucketName, objectName, filePath, opts PutObjectOptions) (info UploadInfo, err error)
Uploads contents from a file to objectName.

FPutObject uploads objects that are less than 128MiB in a single PUT operation. For objects that are greater than the 128MiB in size, FPutObject seamlessly uploads the object in chunks of 128MiB or more depending on the actual file size. The max upload size for an object is 5TB.

__Parameters__


|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket  |
|`objectName` | _string_  |Name of the object |
|`filePath` | _string_  |Path to file to be uploaded |
|`opts` | _minio.PutObjectOptions_  |Pointer to struct that allows user to set optional custom metadata, content-type, content-encoding, content-disposition, content-language and cache-control headers, pass encryption module for encrypting objects, and optionally configure number of threads for multipart put operation.  |


__minio.UploadInfo__

| Field               | Type     | Description                                                                                                                                                                        |
|:--------------------|:---------|:-------------------------------------------|
| `info.ETag`         | _string_ | The ETag of the new object                 |
| `info.VersionID`    | _string_ | The version identifyer of the new object   |


__Example__


```go
uploadInfo, err := minioClient.FPutObject(context.Background(), "my-bucketname", "my-objectname", "my-filename.csv", minio.PutObjectOptions{
	ContentType: "application/csv",
});
if err != nil {
    fmt.Println(err)
    return
}
fmt.Println("Successfully uploaded object: ", uploadInfo)
```

<a name="StatObject"></a>
### StatObject(ctx context.Context, bucketName, objectName string, opts StatObjectOptions) (ObjectInfo, error)
Fetch metadata of an object.

__Parameters__


|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket  |
|`objectName` | _string_  |Name of the object   |
|`opts` | _minio.StatObjectOptions_ | Options for GET info/stat requests specifying additional options like encryption, If-Match |


__Return Value__

|Param   |Type   |Description   |
|:---|:---| :---|
|`objInfo`  | _minio.ObjectInfo_  |Object stat information |


__minio.ObjectInfo__

|Field   |Type   |Description   |
|:---|:---| :---|
|`objInfo.LastModified`  | _time.Time_  |Time when object was last modified |
|`objInfo.ETag` | _string_ |MD5 checksum of the object|
|`objInfo.ContentType` | _string_ |Content type of the object|
|`objInfo.Size` | _int64_ |Size of the object|


__Example__


```go
objInfo, err := minioClient.StatObject(context.Background(), "mybucket", "myobject", minio.StatObjectOptions{})
if err != nil {
    fmt.Println(err)
    return
}
fmt.Println(objInfo)
```

<a name="RemoveObject"></a>
### RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error
Removes an object with some specified options

__Parameters__


|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket  |
|`objectName` | _string_  |Name of the object |
|`opts`	|_minio.RemoveObjectOptions_ |Allows user to set options |

__minio.RemoveObjectOptions__

|Field | Type | Description |
|:--- |:--- | :--- |
| `opts.GovernanceBypass` | _bool_ |Set the bypass governance header to delete an object locked with GOVERNANCE mode|
| `opts.VersionID` | _string_ |Version ID of the object to delete|
| `opts.Internal`                | _minio.AdvancedRemoveOptions_               | This option is intended for internal use by MinIO server and should not be set unless the application is aware of intended use.

```go
opts := minio.RemoveObjectOptions {
		GovernanceBypass: true,
		VersionID: "myversionid",
		}
err = minioClient.RemoveObject(context.Background(), "mybucket", "myobject", opts)
if err != nil {
    fmt.Println(err)
    return
}
```
<a name="PutObjectRetention"></a>
### PutObjectRetention(ctx context.Context, bucketName, objectName string, opts minio.PutObjectRetentionOptions) error
Applies object retention lock onto an object.

__Parameters__


|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket  |
|`objectName` | _string_  |Name of the object |
|`opts`	|_minio.PutObjectRetentionOptions_ |Allows user to set options like retention mode, expiry date and version id |

<a name="RemoveObjects"></a>
### RemoveObjects(ctx context.Context, bucketName string, objectsCh <-chan ObjectInfo, opts RemoveObjectsOptions) <-chan RemoveObjectError
Removes a list of objects obtained from an input channel. The call sends a delete request to the server up to 1000 objects at a time. The errors observed are sent over the error channel.

Parameters

|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket  |
|`objectsCh` |  _chan minio.ObjectInfo_  | Channel of objects to be removed  |
|`opts` |_minio.RemoveObjectsOptions_ | Allows user to set options |

__minio.RemoveObjectsOptions__

|Field | Type | Description |
|:--- |:--- | :--- |
| `opts.GovernanceBypass` | _bool_ |Set the bypass governance header to delete an object locked with GOVERNANCE mode|

__Return Values__

|Param   |Type   |Description   |
|:---|:---| :---|
|`errorCh` | _<-chan minio.RemoveObjectError_  | Receive-only channel of errors observed during deletion.  |

```go
objectsCh := make(chan minio.ObjectInfo)

// Send object names that are needed to be removed to objectsCh
go func() {
	defer close(objectsCh)
	// List all objects from a bucket-name with a matching prefix.
	for object := range minioClient.ListObjects(context.Background(), "my-bucketname", "my-prefixname", true, nil) {
		if object.Err != nil {
			log.Fatalln(object.Err)
		}
		objectsCh <- object
	}
}()

opts := minio.RemoveObjectsOptions{
	GovernanceBypass: true,
}

for rErr := range minioClient.RemoveObjects(context.Background(), "my-bucketname", objectsCh, opts) {
    fmt.Println("Error detected during deletion: ", rErr)
}
```

<a name="GetObjectRetention"></a>
### GetObjectRetention(ctx context.Context, bucketName, objectName, versionID string) (mode *RetentionMode, retainUntilDate *time.Time, err error)
Returns retention set on a given object.

__Parameters__


|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket  |
|`objectName` | _string_  |Name of the object |
|`versionID`	|_string_ |Version ID of the object |

```go
err = minioClient.PutObjectRetention(context.Background(), "mybucket", "myobject", "")
if err != nil {
    fmt.Println(err)
    return
}
```
<a name="PutObjectLegalHold"></a>
### PutObjectLegalHold(ctx context.Context, bucketName, objectName string, opts minio.PutObjectLegalHoldOptions) error
Applies legal-hold onto an object.

__Parameters__


|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket  |
|`objectName` | _string_  |Name of the object |
|`opts`	|_minio.PutObjectLegalHoldOptions_ |Allows user to set options like status and version id |

_minio.PutObjectLegalHoldOptions_

|Field | Type | Description |
|:--- |:--- | :--- |
| `opts.Status` | _*minio.LegalHoldStatus_ |Legal-Hold status to be set|
| `opts.VersionID` | _string_ |Version ID of the object to apply retention on|

```go
s := minio.LegalHoldEnabled
opts := minio.PutObjectLegalHoldOptions {
    Status: &s,
}
err = minioClient.PutObjectLegalHold(context.Background(), "mybucket", "myobject", opts)
if err != nil {
    fmt.Println(err)
    return
}
```
<a name="GetObjectLegalHold"></a>
### GetObjectLegalHold(ctx context.Context, bucketName, objectName, versionID string) (status *LegalHoldStatus, err error)
Returns legal-hold status on a given object.

__Parameters__

|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket  |
|`objectName` | _string_  |Name of the object |
|`opts`	|_minio.GetObjectLegalHoldOptions_ |Allows user to set options like version id |

```go
opts := minio.GetObjectLegalHoldOptions{}
err = minioClient.GetObjectLegalHold(context.Background(), "mybucket", "myobject", opts)
if err != nil {
    fmt.Println(err)
    return
}
```
<a name="SelectObjectContent"></a>
### SelectObjectContent(ctx context.Context, bucketName string, objectsName string, expression string, options SelectObjectOptions) *SelectResults
Parameters

|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`ctx`  | _context.Context_  |Request context  |
|`bucketName`  | _string_  |Name of the bucket  |
|`objectName`  | _string_  |Name of the object |
|`options` |  _SelectObjectOptions_  |  Query Options |

__Return Values__

|Param   |Type   |Description   |
|:---|:---| :---|
|`SelectResults` | _SelectResults_  | Is an io.ReadCloser object which can be directly passed to csv.NewReader for processing output.  |

```go
	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})

	opts := minio.SelectObjectOptions{
		Expression:     "select count(*) from s3object",
		ExpressionType: minio.QueryExpressionTypeSQL,
		InputSerialization: minio.SelectObjectInputSerialization{
			CompressionType: minio.SelectCompressionNONE,
			CSV: &minio.CSVInputOptions{
				FileHeaderInfo:  minio.CSVFileHeaderInfoNone,
				RecordDelimiter: "\n",
				FieldDelimiter:  ",",
			},
		},
		OutputSerialization: minio.SelectObjectOutputSerialization{
			CSV: &minio.CSVOutputOptions{
				RecordDelimiter: "\n",
				FieldDelimiter:  ",",
			},
		},
	}

	reader, err := s3Client.SelectObjectContent(context.Background(), "mycsvbucket", "mycsv.csv", opts)
	if err != nil {
		log.Fatalln(err)
	}
	defer reader.Close()

	if _, err := io.Copy(os.Stdout, reader); err != nil {
		log.Fatalln(err)
	}
```

<a name="PutObjectTagging"></a>
### PutObjectTagging(ctx context.Context, bucketName, objectName string, otags *tags.Tags) error
set new object Tags to the given object, replaces/overwrites any existing tags.

__Parameters__


|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket   |
|`objectName` | _string_  |Name of the object   |
|`objectTags` | _*tags.Tags_ | Map with Object Tag's Key and Value |

__Example__


```go
err = minioClient.PutObjectTagging(context.Background(), bucketName, objectName, objectTags)
if err != nil {
    fmt.Println(err)
    return
}
```

<a name="GetObjectTagging"></a>
### GetObjectTagging(ctx context.Context, bucketName, objectName string) (*tags.Tags, error)
Fetch Object Tags from the given object

__Parameters__


|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket   |
|`objectName` | _string_  |Name of the object   |

__Example__


```go
tags, err = minioClient.GetObjectTagging(context.Background(), bucketName, objectName)
if err != nil {
    fmt.Println(err)
    return
}
fmt.Printf("Fetched Tags: %s", tags)
```

<a name="RemoveObjectTagging"></a>
### RemoveObjectTagging(ctx context.Context, bucketName, objectName string) error
Remove Object Tags from the given object

__Parameters__


|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket   |
|`objectName` | _string_  |Name of the object   |

__Example__


```go
err = minioClient.RemoveObjectTagging(context.Background(), bucketName, objectName)
if err != nil {
    fmt.Println(err)
    return
}
```

<a name="RestoreObject"></a>
### RestoreObject(ctx context.Context, bucketName, objectName, versionID string, opts minio.RestoreRequest) error
Restore or perform SQL operations on an archived object

__Parameters__

|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket   |
|`objectName` | _string_  |Name of the object   |
|`versionID` | _string_  |Version ID of the object   |
|`opts` | _minio.RestoreRequest | Restore request options |

__Example__

```go
opts := minio.RestoreRequest{}
opts.SetDays(1)
opts.SetGlacierJobParameters(minio.GlacierJobParameters{Tier: minio.TierStandard})

err = s3Client.RestoreObject(context.Background(), "your-bucket", "your-object", "", opts)
if err != nil {
    log.Fatalln(err)
}
```

<a name="GetObjectAttributes"></a>
### GetObjectAttributes(ctx context.Context, bucketName, objectName string, opts ObjectAttributesOptions) (*ObjectAttributes, error)
Returns a stream of the object data. Most of the common errors occur when reading the stream.


__Parameters__


|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket  |
|`objectName` | _string_  |Name of the object  |
|`opts` | _minio.ObjectAttributesOptions_ | Configuration for pagination and selection of object attributes |


__minio.ObjectAttributesOptions__

|Field | Type | Description |
|:---|:---|:---|
| `opts.ServerSideEncryption` | _encrypt.ServerSide_ | Interface provided by `encrypt` package to specify server-side-encryption. (For more information see https://godoc.org/github.com/minio/minio-go/v7) |
| `opts.MaxParts`                | _int               | This option defines how many parts should be returned by the API
| `opts.VersionID`                | _string               | VersionID defines which version of the object will be used
| `opts.PartNumberMarker`                | _int               | This options defines which part number pagination will start after, the part which number is equal to PartNumberMarker will not be included in the response

__Return Value__

|Param   |Type   |Description   |
|:---|:---| :---|
|`objectAttributes`  | _*minio.ObjectAttributes_ |_minio.ObjectAttributes_ contains the information about the object and it's parts. |

__Example__


```go
objectAttributes, err := c.GetObjectAttributes(
    context.Background(), 
    "your-bucket", 
    "your-object", 
    minio.ObjectAttributesOptions{
        VersionID:"object-version-id",
        NextPartMarker:0,
        MaxParts:100,
    })

if err != nil {
    fmt.Println(err)
	return
}

fmt.Println(objectAttributes)
```


<a name="RemoveIncompleteUpload"></a>
### RemoveIncompleteUpload(ctx context.Context, bucketName, objectName string) error
Removes a partially uploaded object.

__Parameters__


|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket   |
|`objectName` | _string_  |Name of the object   |

__Example__


```go
err = minioClient.RemoveIncompleteUpload(context.Background(), "mybucket", "myobject")
if err != nil {
    fmt.Println(err)
    return
}
```

## 4. Presigned operations

<a name="PresignedGetObject"></a>
### PresignedGetObject(ctx context.Context, bucketName, objectName string, expiry time.Duration, reqParams url.Values) (*url.URL, error)
Generates a presigned URL for HTTP GET operations. Browsers/Mobile clients may point to this URL to directly download objects even if the bucket is private. This presigned URL can have an associated expiration time in seconds after which it is no longer operational. The maximum expiry is 604800 seconds (i.e. 7 days) and minimum is 1 second. 

__Parameters__


|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket   |
|`objectName` | _string_  |Name of the object   |
|`expiry` | _time.Duration_  |Expiry of presigned URL in seconds   |
|`reqParams` | _url.Values_  |Additional response header overrides supports _response-expires_, _response-content-type_, _response-cache-control_, _response-content-disposition_.  |


__Example__


```go
// Set request parameters for content-disposition.
reqParams := make(url.Values)
reqParams.Set("response-content-disposition", "attachment; filename=\"your-filename.txt\"")

// Generates a presigned url which expires in a day.
presignedURL, err := minioClient.PresignedGetObject(context.Background(), "mybucket", "myobject", time.Second * 24 * 60 * 60, reqParams)
if err != nil {
    fmt.Println(err)
    return
}
fmt.Println("Successfully generated presigned URL", presignedURL)
```

<a name="PresignedPutObject"></a>
### PresignedPutObject(ctx context.Context, bucketName, objectName string, expiry time.Duration) (*url.URL, error)
Generates a presigned URL for HTTP PUT operations. Browsers/Mobile clients may point to this URL to upload objects directly to a bucket even if it is private. This presigned URL can have an associated expiration time in seconds after which it is no longer operational. The default expiry is set to 7 days.

NOTE: you can upload to S3 only with specified object name.

__Parameters__


|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket   |
|`objectName` | _string_  |Name of the object   |
|`expiry` | _time.Duration_  |Expiry of presigned URL in seconds |


__Example__


```go
// Generates a url which expires in a day.
expiry := time.Second * 24 * 60 * 60 // 1 day.
presignedURL, err := minioClient.PresignedPutObject(context.Background(), "mybucket", "myobject", expiry)
if err != nil {
    fmt.Println(err)
    return
}
fmt.Println("Successfully generated presigned URL", presignedURL)
```

<a name="PresignedHeadObject"></a>
### PresignedHeadObject(ctx context.Context, bucketName, objectName string, expiry time.Duration, reqParams url.Values) (*url.URL, error)
Generates a presigned URL for HTTP HEAD operations. Browsers/Mobile clients may point to this URL to directly get metadata from objects even if the bucket is private. This presigned URL can have an associated expiration time in seconds after which it is no longer operational. The default expiry is set to 7 days.

__Parameters__

|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket   |
|`objectName` | _string_  |Name of the object   |
|`expiry` | _time.Duration_  |Expiry of presigned URL in seconds   |
|`reqParams` | _url.Values_  |Additional response header overrides supports _response-expires_, _response-content-type_, _response-cache-control_, _response-content-disposition_.  |


__Example__


```go
// Set request parameters for content-disposition.
reqParams := make(url.Values)
reqParams.Set("response-content-disposition", "attachment; filename=\"your-filename.txt\"")

// Generates a presigned url which expires in a day.
presignedURL, err := minioClient.PresignedHeadObject(context.Background(), "mybucket", "myobject", time.Second * 24 * 60 * 60, reqParams)
if err != nil {
    fmt.Println(err)
    return
}
fmt.Println("Successfully generated presigned URL", presignedURL)
```

<a name="PresignedPostPolicy"></a>
### PresignedPostPolicy(ctx context.Context, post PostPolicy) (*url.URL, map[string]string, error)
Allows setting policy conditions to a presigned URL for POST operations. Policies such as bucket name to receive object uploads, key name prefixes, expiry policy may be set.

```go
// Initialize policy condition config.
policy := minio.NewPostPolicy()

// Apply upload policy restrictions:
policy.SetBucket("mybucket")
policy.SetKey("myobject")
policy.SetExpires(time.Now().UTC().AddDate(0, 0, 10)) // expires in 10 days

// Only allow 'png' images.
policy.SetContentType("image/png")

// Only allow content size in range 1KB to 1MB.
policy.SetContentLengthRange(1024, 1024*1024)

// Add a user metadata using the key "custom" and value "user"
policy.SetUserMetadata("custom", "user")

// Get the POST form key/value object:
url, formData, err := minioClient.PresignedPostPolicy(context.Background(), policy)
if err != nil {
    fmt.Println(err)
    return
}

// POST your content from the command line using `curl`
fmt.Printf("curl ")
for k, v := range formData {
    fmt.Printf("-F %s=%s ", k, v)
}
fmt.Printf("-F file=@/etc/bash.bashrc ")
fmt.Printf("%s\n", url)
```

## 5. Bucket policy/notification operations

<a name="SetBucketPolicy"></a>
### SetBucketPolicy(ctx context.Context, bucketname, policy string) error
Set access permissions on bucket or an object prefix.

__Parameters__

|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName` | _string_  |Name of the bucket|
|`policy` | _string_  |Policy to be set |

__Return Values__

|Param   |Type   |Description   |
|:---|:---| :---|
|`err` | _error_  |Standard Error   |

__Example__

```go
policy := `{"Version": "2012-10-17","Statement": [{"Action": ["s3:GetObject"],"Effect": "Allow","Principal": {"AWS": ["*"]},"Resource": ["arn:aws:s3:::my-bucketname/*"],"Sid": ""}]}`

err = minioClient.SetBucketPolicy(context.Background(), "my-bucketname", policy)
if err != nil {
    fmt.Println(err)
    return
}
```

<a name="GetBucketPolicy"></a>
### GetBucketPolicy(ctx context.Context, bucketName string) (policy string, error)
Get access permissions on a bucket or a prefix.

__Parameters__


|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket   |

__Return Values__


|Param   |Type   |Description   |
|:---|:---| :---|
|`policy`  | _string_ |Policy returned from the server |
|`err` | _error_  |Standard Error  |

__Example__

```go
policy, err := minioClient.GetBucketPolicy(context.Background(), "my-bucketname")
if err != nil {
    log.Fatalln(err)
}
```

<a name="GetBucketNotification"></a>
### GetBucketNotification(ctx context.Context, bucketName string) (notification.Configuration, error)
Get notification configuration on a bucket.

__Parameters__


|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket |

__Return Values__


|Param   |Type   |Description   |
|:---|:---| :---|
|`config`  | _notification.Configuration_ |structure which holds all notification configurations|
|`err` | _error_  |Standard Error  |

__Example__


```go
bucketNotification, err := minioClient.GetBucketNotification(context.Background(), "mybucket")
if err != nil {
    fmt.Println("Failed to get bucket notification configurations for mybucket", err)
    return
}

for _, queueConfig := range bucketNotification.QueueConfigs {
    for _, e := range queueConfig.Events {
        fmt.Println(e + " event is enabled")
    }
}
```

<a name="SetBucketNotification"></a>
### SetBucketNotification(ctx context.Context, bucketName string, config notification.Configuration) error
Set a new bucket notification on a bucket.

__Parameters__


|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket   |
|`config`  | _notification.Configuration_  |Represents the XML to be sent to the configured web service  |

__Return Values__


|Param   |Type   |Description   |
|:---|:---| :---|
|`err` | _error_  |Standard Error  |

__Example__


```go
queueArn := notification.NewArn("aws", "sqs", "us-east-1", "804605494417", "PhotoUpdate")

queueConfig := notification.NewConfig(queueArn)
queueConfig.AddEvents(minio.ObjectCreatedAll, minio.ObjectRemovedAll)
queueConfig.AddFilterPrefix("photos/")
queueConfig.AddFilterSuffix(".jpg")

config := notification.Configuration{}
config.AddQueue(queueConfig)

err = minioClient.SetBucketNotification(context.Background(), "mybucket", config)
if err != nil {
    fmt.Println("Unable to set the bucket notification: ", err)
    return
}
```

<a name="RemoveAllBucketNotification"></a>
### RemoveAllBucketNotification(ctx context.Context, bucketName string) error
Remove all configured bucket notifications on a bucket.

__Parameters__


|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket   |

__Return Values__


|Param   |Type   |Description   |
|:---|:---| :---|
|`err` | _error_  |Standard Error  |

__Example__


```go
err = minioClient.RemoveAllBucketNotification(context.Background(), "mybucket")
if err != nil {
    fmt.Println("Unable to remove bucket notifications.", err)
    return
}
```

<a name="ListenBucketNotification"></a>
### ListenBucketNotification(context context.Context, bucketName, prefix, suffix string, events []string) <-chan notification.Info
ListenBucketNotification API receives bucket notification events through the notification channel. The returned notification channel has two fields 'Records' and 'Err'.

- 'Records' holds the notifications received from the server.
- 'Err' indicates any error while processing the received notifications.

NOTE: Notification channel is closed at the first occurrence of an error.

__Parameters__


|Param   |Type   |Description   |
|:---|:---| :---|
|`bucketName`  | _string_  | Bucket to listen notifications on   |
|`prefix`  | _string_ | Object key prefix to filter notifications for  |
|`suffix`  | _string_ | Object key suffix to filter notifications for  |
|`events`  | _[]string_ | Enables notifications for specific event types |

__Return Values__

|Param   |Type   |Description   |
|:---|:---| :---|
|`notificationInfo` | _chan notification.Info_ | Channel of bucket notifications |

__minio.NotificationInfo__

|Field   |Type   |Description   |
|`notificationInfo.Records` | _[]notification.Event_ | Collection of notification events |
|`notificationInfo.Err` | _error_ | Carries any error occurred during the operation (Standard Error) |


__Example__


```go
// Listen for bucket notifications on "mybucket" filtered by prefix, suffix and events.
for notificationInfo := range minioClient.ListenBucketNotification(context.Background(), "mybucket", "myprefix/", ".mysuffix", []string{
    "s3:ObjectCreated:*",
    "s3:ObjectAccessed:*",
    "s3:ObjectRemoved:*",
    }) {
    if notificationInfo.Err != nil {
        fmt.Println(notificationInfo.Err)
    }
    fmt.Println(notificationInfo)
}
```

<a name="ListenNotification"></a>
### ListenNotification(context context.Context, prefix, suffix string, events []string) <-chan notification.Info
ListenNotification API receives bucket and object notification events through the notification channel. The returned notification channel has two fields 'Records' and 'Err'.

- 'Records' holds the notifications received from the server.
- 'Err' indicates any error while processing the received notifications.

NOTE: Notification channel is closed at the first occurrence of an error.

__Parameters__


|Param   |Type   |Description   |
|:---|:---| :---|
|`bucketName`  | _string_  | Bucket to listen notifications on   |
|`prefix`  | _string_ | Object key prefix to filter notifications for  |
|`suffix`  | _string_ | Object key suffix to filter notifications for  |
|`events`  | _[]string_ | Enables notifications for specific event types |

__Return Values__

|Param   |Type   |Description   |
|:---|:---| :---|
|`notificationInfo` | _chan notification.Info_ | Read channel for all notifications |

__minio.NotificationInfo__

|Field   |Type   |Description   |
|`notificationInfo.Records` | _[]notification.Event_ | Collection of notification events |
|`notificationInfo.Err` | _error_ | Carries any error occurred during the operation (Standard Error) |

__Example__


```go
// Listen for bucket notifications on "mybucket" filtered by prefix, suffix and events.
for notificationInfo := range minioClient.ListenNotification(context.Background(), "myprefix/", ".mysuffix", []string{
    "s3:BucketCreated:*",
    "s3:BucketRemoved:*",
    "s3:ObjectCreated:*",
    "s3:ObjectAccessed:*",
    "s3:ObjectRemoved:*",
    }) {
    if notificationInfo.Err != nil {
        fmt.Println(notificationInfo.Err)
    }
    fmt.Println(notificationInfo)
}
```

<a name="SetBucketLifecycle"></a>
### SetBucketLifecycle(ctx context.Context, bucketname, config *lifecycle.Configuration) error
Set lifecycle on bucket or an object prefix.

__Parameters__

|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName` | _string_  |Name of the bucket|
|`config` | _lifecycle.Configuration_  |Lifecycle to be set |

__Return Values__

|Param   |Type   |Description   |
|:---|:---| :---|
|`err` | _error_  |Standard Error   |

__Example__

```go
config := lifecycle.NewConfiguration()
config.Rules = []lifecycle.Rule{
  {
    ID:     "expire-bucket",
    Status: "Enabled",
    Expiration: lifecycle.Expiration{
       Days: 365,
    },
  },
}

err = minioClient.SetBucketLifecycle(context.Background(), "my-bucketname", config)
if err != nil {
    fmt.Println(err)
    return
}
```

<a name="GetBucketLifecycle"></a>
### GetBucketLifecycle(ctx context.Context, bucketName string) (*lifecycle.Configuration error)
Get lifecycle on a bucket or a prefix.

__Parameters__


|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket   |

__Return Values__


|Param   |Type   |Description   |
|:---|:---| :---|
|`config`  | _lifecycle.Configuration_ |Lifecycle returned from the server |
|`err` | _error_  |Standard Error  |

__Example__

```go
lifecycle, err := minioClient.GetBucketLifecycle(context.Background(), "my-bucketname")
if err != nil {
    log.Fatalln(err)
}
```

<a name="SetBucketEncryption"></a>
### SetBucketEncryption(ctx context.Context, bucketname string, config sse.Configuration) error
Set default encryption configuration on a bucket.

__Parameters__

|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName` | _string_  |Name of the bucket|
|`config` | _sse.Configuration_  | Structure that holds default encryption configuration to be set |

__Return Values__

|Param   |Type   |Description   |
|:---|:---| :---|
|`err` | _error_  |Standard Error   |

__Example__

```go
s3Client, err := minio.New("s3.amazonaws.com", &minio.Options{
	Creds:  credentials.NewStaticV4("YOUR-ACCESSKEYID", "YOUR-SECRETACCESSKEY", ""),
	Secure: true,
})
if err != nil {
    log.Fatalln(err)
}

// Set default encryption configuration on an S3 bucket
err = s3Client.SetBucketEncryption(context.Background(), "my-bucketname", sse.NewConfigurationSSES3())
if err != nil {
    log.Fatalln(err)
}
```

<a name="GetBucketEncryption"></a>
### GetBucketEncryption(ctx context.Context, bucketName string) (*sse.Configuration, error)
Get default encryption configuration set on a bucket.

__Parameters__


|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket   |

__Return Values__


|Param   |Type   |Description   |
|:---|:---| :---|
|`config` | _sse.Configuration_ | Structure that holds default encryption configuration |
|`err` | _error_ |Standard Error  |

__Example__

```go
s3Client, err := minio.New("s3.amazonaws.com", &minio.Options{
	Creds:  credentials.NewStaticV4("YOUR-ACCESSKEYID", "YOUR-SECRETACCESSKEY", ""),
	Secure: true,
})
if err != nil {
    log.Fatalln(err)
}

// Get default encryption configuration set on an S3 bucket and print it out
encryptionConfig, err := s3Client.GetBucketEncryption(context.Background(), "my-bucketname")
if err != nil {
    log.Fatalln(err)
}
fmt.Printf("%+v\n", encryptionConfig)
```

<a name="RemoveBucketEncryption"></a>
### RemoveBucketEncryption(ctx context.Context, bucketName string) (error)
Remove default encryption configuration set on a bucket.

__Parameters__


|Param   |Type   |Description   |
|:---|:---|:---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket   |

__Return Values__


|Param   |Type   |Description   |
|:---|:---| :---|
|`err` | _error_  |Standard Error  |

__Example__

```go
err := s3Client.RemoveBucketEncryption(context.Background(), "my-bucketname")
if err != nil {
    log.Fatalln(err)
}
// "my-bucket" is successfully deleted/removed.
```

<a name="SetObjectLockConfig"></a>
### SetObjectLockConfig(ctx context.Context, bucketname, mode *RetentionMode, validity *uint, unit *ValidityUnit) error
Set object lock configuration in given bucket. mode, validity and unit are either all set or all nil.

__Parameters__

|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName` | _string_  |Name of the bucket|
|`mode` | _RetentionMode_  |Retention mode to be set |
|`validity` | _uint_  |Validity period to be set |
|`unit` | _ValidityUnit_  |Unit of validity period |

__Return Values__

|Param   |Type   |Description   |
|:---|:---| :---|
|`err` | _error_  |Standard Error   |

__Example__

```go
mode := Governance
validity := uint(30)
unit := Days

err = minioClient.SetObjectLockConfig(context.Background(), "my-bucketname", &mode, &validity, &unit)
if err != nil {
    fmt.Println(err)
    return
}
```

<a name="GetObjectLockConfig"></a>
### GetObjectLockConfig(ctx context.Context, bucketName string) (objectLock,*RetentionMode, *uint, *ValidityUnit, error)
Get object lock configuration of given bucket.

__Parameters__


|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket   |

__Return Values__


|Param   |Type   |Description   |
|:---|:---| :---|
|`objectLock` | _objectLock_  |lock enabled status |
|`mode` | _RetentionMode_  |Current retention mode |
|`validity` | _uint_  |Current validity period |
|`unit` | _ValidityUnit_  |Unit of validity period |
|`err` | _error_  |Standard Error  |

__Example__

```go
enabled, mode, validity, unit, err := minioClient.GetObjectLockConfig(context.Background(), "my-bucketname")
if err != nil {
    log.Fatalln(err)
}
fmt.Println("object lock is %s for this bucket",enabled)
if mode != nil {
	fmt.Printf("%v mode is enabled for %v %v for bucket 'my-bucketname'\n", *mode, *validity, *unit)
} else {
	fmt.Println("No mode is enabled for bucket 'my-bucketname'")
}
```

<a name="EnableVersioning"></a>
### EnableVersioning(ctx context.Context, bucketName string) error
Enable bucket versioning support.

__Parameters__


|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket   |

__Return Values__


|Param   |Type   |Description   |
|:---|:---| :---|
|`err` | _error_  |Standard Error  |

__Example__

```go
err := minioClient.EnableVersioning(context.Background(), "my-bucketname")
if err != nil {
    log.Fatalln(err)
}

fmt.Println("versioning enabled for bucket 'my-bucketname'")
```

<a name="DisableVersioning"></a>
### DisableVersioning(ctx context.Context, bucketName) error
Disable bucket versioning support.

__Parameters__


|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket   |

__Return Values__


|Param   |Type   |Description   |
|:---|:---| :---|
|`err` | _error_  |Standard Error  |

__Example__

```go
err := minioClient.DisableVersioning(context.Background(), "my-bucketname")
if err != nil {
    log.Fatalln(err)
}

fmt.Println("versioning disabled for bucket 'my-bucketname'")
```

<a name="GetBucketVersioning"></a>
### GetBucketVersioning(ctx context.Context, bucketName string) (BucketVersioningConfiguration, error)
Get versioning configuration set on a bucket.

__Parameters__


|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket   |

__Return Values__


|Param   |Type   |Description   |
|:---|:---| :---|
|`configuration` | _minio.BucketVersioningConfiguration_ | Structure that holds versioning configuration |
|`err` | _error_ |Standard Error  |

__Example__

```go
s3Client, err := minio.New("s3.amazonaws.com", &minio.Options{
	Creds:  credentials.NewStaticV4("YOUR-ACCESSKEYID", "YOUR-SECRETACCESSKEY", ""),
	Secure: true,
})
if err != nil {
    log.Fatalln(err)
}

// Get versioning configuration set on an S3 bucket and print it out
versioningConfig, err := s3Client.GetBucketVersioning(context.Background(), "my-bucketname")
if err != nil {
    log.Fatalln(err)
}
fmt.Printf("%+v\n", versioningConfig)
```

<a name="SetBucketReplication"></a>

### SetBucketReplication(ctx context.Context, bucketname, cfg replication.Config) error
Set replication configuration on a bucket. Role can be obtained by first defining the replication target on MinIO using `mc admin bucket remote set` to associate the source and destination buckets for replication with the replication endpoint.

__Parameters__

|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName` | _string_  |Name of the bucket|
|`cfg` | _replication.Config_  |Replication configuration to be set |

__Return Values__

|Param   |Type   |Description   |
|:---|:---| :---|
|`err` | _error_  |Standard Error   |

__Example__

```go
replicationStr := `<ReplicationConfiguration>
   <Role></Role>
   <Rule>
      <DeleteMarkerReplication>
         <Status>Disabled</Status>
      </DeleteMarkerReplication>
      <Destination>
         <Bucket>string</Bucket>
         <StorageClass>string</StorageClass>
      </Destination>
      <Filter>
         <And>
            <Prefix>string</Prefix>
            <Tag>
               <Key>string</Key>
               <Value>string</Value>
            </Tag>
            ...
         </And>
         <Prefix>string</Prefix>
         <Tag>
            <Key>string</Key>
            <Value>string</Value>
         </Tag>
      </Filter>
      <ID>string</ID>
      <Prefix>string</Prefix>
      <Priority>integer</Priority>
      <Status>string</Status>
   </Rule>
</ReplicationConfiguration>`
replicationConfig := replication.Config{}
if err := xml.Unmarshal([]byte(replicationStr), &replicationConfig); err != nil {
    log.Fatalln(err)
}
cfg.Role := "arn:minio:s3::598361bf-3cec-49a7-b529-ce870a34d759:*"
err = minioClient.SetBucketReplication(context.Background(), "my-bucketname", replicationConfig)
if err != nil {
    fmt.Println(err)
    return
}
```

<a name="GetBucketReplication"></a>
### GetBucketReplication(ctx context.Context, bucketName string) (replication.Config, error)
Get current replication config on a bucket.

__Parameters__


|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket   |

__Return Values__


|Param   |Type   |Description   |
|:---|:---| :---|
|`replication`  | _replication.Config_ |Replication config returned from the server |
|`err` | _error_  |Standard Error  |

__Example__

```go
replication, err := minioClient.GetBucketReplication(context.Background(), "my-bucketname")
if err != nil {
    log.Fatalln(err)
}
```

<a name="RemoveBucketReplication"></a>
### RemoveBucketReplication(ctx context.Context, bucketname string) error
Removes replication configuration on a bucket.

__Parameters__

|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName` | _string_  |Name of the bucket|

__Return Values__

|Param   |Type   |Description   |
|:---|:---| :---|
|`err` | _error_  |Standard Error   |

__Example__

```go
err = minioClient.RemoveBucketReplication(context.Background(), "my-bucketname")
if err != nil {
    fmt.Println(err)
    return
}
```

<a name="GetBucketReplicationMetrics"></a>
### GetBucketReplicationMetrics(ctx context.Context, bucketName string) (replication.Metrics, error)
Get latest replication metrics on a bucket. This is a MinIO specific extension.

__Parameters__


|Param   |Type   |Description   |
|:---|:---| :---|
|`ctx`  | _context.Context_  | Custom context for timeout/cancellation of the call|
|`bucketName`  | _string_  |Name of the bucket   |

__Return Values__


|Param   |Type   |Description   |
|:---|:---| :---|
|`metrics`  | _replication.Metrics_ |Replication metrics returned from the server |
|`err` | _error_  |Standard Error  |

__Example__

```go
replMetrics, err := minioClient.GetBucketReplicationMetrics(context.Background(), "my-bucketname")
if err != nil {
    log.Fatalln(err)
}
```

## 6. Client custom settings

<a name="SetAppInfo"></a>
### SetAppInfo(appName, appVersion string)
Add custom application details to User-Agent.

__Parameters__

| Param  | Type  | Description  |
|---|---|---|
|`appName`  | _string_  | Name of the application performing the API requests. |
| `appVersion`| _string_ | Version of the application performing the API requests. |


__Example__


```go
// Set Application name and version to be used in subsequent API requests.
minioClient.SetAppInfo("myCloudApp", "1.0.0")
```

<a name="TraceOn"></a>
### TraceOn(outputStream io.Writer)
Enables HTTP tracing. The trace is written to the io.Writer provided. If outputStream is nil, trace is written to os.Stdout.

__Parameters__

| Param  | Type  | Description  |
|---|---|---|
|`outputStream`  | _io.Writer_  | HTTP trace is written into outputStream.|


<a name="TraceOff"></a>
### TraceOff()
Disables HTTP tracing.

<a name="SetS3TransferAccelerate"></a>
### SetS3TransferAccelerate(acceleratedEndpoint string)
Set AWS S3 transfer acceleration endpoint for all API requests hereafter.
NOTE: This API applies only to AWS S3 and is a no operation for S3 compatible object storage services.

__Parameters__

| Param  | Type  | Description  |
|---|---|---|
|`acceleratedEndpoint`  | _string_  | Set to new S3 transfer acceleration endpoint.|
