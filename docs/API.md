MinIO Go Client API Reference [![Slack](https://slack.min.io/slack?type=svg)](https://slack.min.io)
===================================================================================================

Initialize MinIO Client object.
-------------------------------

MinIO
-----

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

AWS S3
------

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

| Bucket operations                                             | Object operations                                   | Presigned operations                          | Bucket Policy/Notification Operations                         | Client custom settings                                |
|:--------------------------------------------------------------|:----------------------------------------------------|:----------------------------------------------|:--------------------------------------------------------------|:------------------------------------------------------|
| [`MakeBucket`](#MakeBucket)                                   | [`AppendObject`](#AppendObject)                     | [`PresignedGetObject`](#PresignedGetObject)   | [`SetBucketPolicy`](#SetBucketPolicy)                         | [`SetAppInfo`](#SetAppInfo)                           |
| [`ListBuckets`](#ListBuckets)                                 | [`GetObject`](#GetObject)                           | [`PresignedPutObject`](#PresignedPutObject)   | [`GetBucketPolicy`](#GetBucketPolicy)                         | [`TraceOn`](#TraceOn)                                 |
| [`BucketExists`](#BucketExists)                               | [`PutObject`](#PutObject)                           | [`PresignedHeadObject`](#PresignedHeadObject) | [`SetBucketNotification`](#SetBucketNotification)             | [`TraceOff`](#TraceOff)                               |
| [`RemoveBucket`](#RemoveBucket)                               | [`PutObjectFanOut`](#PutObjectFanOut)               | [`PresignedPostPolicy`](#PresignedPostPolicy) | [`GetBucketNotification`](#GetBucketNotification)             | [`SetS3TransferAccelerate`](#SetS3TransferAccelerate) |
| [`ListObjects`](#ListObjects)                                 | [`CopyObject`](#CopyObject)                         |                                               | [`RemoveAllBucketNotification`](#RemoveAllBucketNotification) |                                                       |
| [`ListIncompleteUploads`](#ListIncompleteUploads)             | [`ComposeObject`](#ComposeObject)                   |                                               | [`ListenBucketNotification`](#ListenBucketNotification)       |                                                       |
| [`SetBucketTagging`](#SetBucketTagging)                       | [`StatObject`](#StatObject)                         |                                               | [`ListenNotification`](#ListenNotification)                   |                                                       |
| [`GetBucketTagging`](#GetBucketTagging)                       | [`RemoveObject`](#RemoveObject)                     |                                               | [`SetBucketLifecycle`](#SetBucketLifecycle)                   |                                                       |
| [`RemoveBucketTagging`](#RemoveBucketTagging)                 | [`RemoveObjects`](#RemoveObjects)                   |                                               | [`GetBucketLifecycle`](#GetBucketLifecycle)                   |                                                       |
| [`SetBucketCors`](#SetBucketCors)                             | [`RemoveIncompleteUpload`](#RemoveIncompleteUpload) |                                               | [`SetBucketEncryption`](#SetBucketEncryption)                 |                                                       |
| [`GetBucketCors`](#GetBucketCors)                             | [`FPutObject`](#FPutObject)                         |                                               | [`GetBucketEncryption`](#GetBucketEncryption)                 |                                                       |
| [`SetBucketReplication`](#SetBucketReplication)               | [`FGetObject`](#FGetObject)                         |                                               | [`RemoveBucketEncryption`](#RemoveBucketEncryption)           |                                                       |
| [`GetBucketReplication`](#GetBucketReplication)               | [`PutObjectRetention`](#PutObjectRetention)         |                                               | [`SetObjectLockConfig`](#SetObjectLockConfig)                 |                                                       |
| [`RemoveBucketReplication`](#RemoveBucketReplication)         | [`GetObjectRetention`](#GetObjectRetention)         |                                               | [`GetObjectLockConfig`](#GetObjectLockConfig)                 |                                                       |
| [`GetBucketReplicationMetrics`](#GetBucketReplicationMetrics) | [`PutObjectLegalHold`](#PutObjectLegalHold)         |                                               | [`EnableVersioning`](#EnableVersioning)                       |                                                       |
| [`GetBucketLocation`](#GetBucketLocation)                     | [`GetObjectLegalHold`](#GetObjectLegalHold)         |                                               | [`SuspendVersioning`](#SuspendVersioning)                     |                                                       |
|                                                               | [`SelectObjectContent`](#SelectObjectContent)       |                                               | [`GetBucketVersioning`](#GetBucketVersioning)                 |                                                       |
|                                                               | [`PutObjectTagging`](#PutObjectTagging)             |                                               |                                                               |                                                       |
|                                                               | [`GetObjectTagging`](#GetObjectTagging)             |                                               |                                                               |                                                       |
|                                                               | [`RemoveObjectTagging`](#RemoveObjectTagging)       |                                               |                                                               |                                                       |
|                                                               | [`RestoreObject`](#RestoreObject)                   |                                               |                                                               |                                                       |
|                                                               | [`GetObjectAttributes`](#GetObjectAttributes)       |                                               |                                                               |                                                       |
|                                                               | [`PromptObject`](#PromptObject)                     |                                               |                                                               |                                                       |

1.	Constructor --------------

<a name="MinIO"></a>

### New(endpoint string, opts \*Options) (\*Client, error)

Initializes a new client object.

**Parameters**

| Param      | Type            | Description                           |
|:-----------|:----------------|:--------------------------------------|
| `endpoint` | *string*        | S3 compatible object storage endpoint |
| `opts`     | *minio.Options* | Options for constructing a new client |

**minio.Options**

| Field               | Type                        | Description                                                                  |
|:--------------------|:----------------------------|:-----------------------------------------------------------------------------|
| `opts.Creds`        | \**credentials.Credentials* | S3 compatible object storage access credentials                              |
| `opts.Secure`       | *bool*                      | If 'true' API requests will be secure (HTTPS), and insecure (HTTP) otherwise |
| `opts.Transport`    | *http.RoundTripper*         | Custom transport for executing HTTP transactions                             |
| `opts.Region`       | *string*                    | S3 compatible object storage region                                          |
| `opts.BucketLookup` | *BucketLookupType*          | Bucket lookup type can be one of the following values                        |
|                     |                             | *minio.BucketLookupDNS*                                                      |
|                     |                             | *minio.BucketLookupPath*                                                     |
|                     |                             | *minio.BucketLookupAuto*                                                     |

1.	Bucket operations --------------------

<a name="MakeBucket"></a>

### MakeBucket(ctx context.Context, bucketName string, opts MakeBucketOptions) error

Creates a new bucket.

**Parameters**

| Param        | Type                      | Description                                                                                                                                                                                                                                 |
|--------------|---------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `ctx`        | *context.Context*         | Custom context for timeout/cancellation of the call                                                                                                                                                                                         |
| `bucketName` | *string*                  | Name of the bucket                                                                                                                                                                                                                          |
| `opts`       | *minio.MakeBucketOptions* | Bucket options such as `Region` where the bucket is to be created. Default value is us-east-1. Other valid values are listed below. Note: When used with minio server, use the region specified in its config file (defaults to us-east-1). |
|              |                           | us-east-1                                                                                                                                                                                                                                   |
|              |                           | us-east-2                                                                                                                                                                                                                                   |
|              |                           | us-west-1                                                                                                                                                                                                                                   |
|              |                           | us-west-2                                                                                                                                                                                                                                   |
|              |                           | ca-central-1                                                                                                                                                                                                                                |
|              |                           | eu-west-1                                                                                                                                                                                                                                   |
|              |                           | eu-west-2                                                                                                                                                                                                                                   |
|              |                           | eu-west-3                                                                                                                                                                                                                                   |
|              |                           | eu-central-1                                                                                                                                                                                                                                |
|              |                           | eu-north-1                                                                                                                                                                                                                                  |
|              |                           | ap-east-1                                                                                                                                                                                                                                   |
|              |                           | ap-south-1                                                                                                                                                                                                                                  |
|              |                           | ap-southeast-1                                                                                                                                                                                                                              |
|              |                           | ap-southeast-2                                                                                                                                                                                                                              |
|              |                           | ap-northeast-1                                                                                                                                                                                                                              |
|              |                           | ap-northeast-2                                                                                                                                                                                                                              |
|              |                           | ap-northeast-3                                                                                                                                                                                                                              |
|              |                           | me-south-1                                                                                                                                                                                                                                  |
|              |                           | sa-east-1                                                                                                                                                                                                                                   |
|              |                           | us-gov-west-1                                                                                                                                                                                                                               |
|              |                           | us-gov-east-1                                                                                                                                                                                                                               |
|              |                           | cn-north-1                                                                                                                                                                                                                                  |
|              |                           | cn-northwest-1                                                                                                                                                                                                                              |

**Example**

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

| Param        | Type                 | Description                                         |
|--------------|----------------------|-----------------------------------------------------|
| `ctx`        | *context.Context*    | Custom context for timeout/cancellation of the call |
| `bucketList` | *[]minio.BucketInfo* | Lists of all buckets                                |

**minio.BucketInfo**

| Field                 | Type        | Description             |
|-----------------------|-------------|-------------------------|
| `bucket.Name`         | *string*    | Name of the bucket      |
| `bucket.CreationDate` | *time.Time* | Date of bucket creation |

**Example**

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

**Parameters**

| Param        | Type              | Description                                         |
|:-------------|:------------------|:----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |

**Return Values**

| Param   | Type    | Description                            |
|:--------|:--------|:---------------------------------------|
| `found` | *bool*  | Indicates whether bucket exists or not |
| `err`   | *error* | Standard Error                         |

**Example**

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

**Parameters**

| Param        | Type              | Description                                         |
|:-------------|:------------------|:----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |

**Example**

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

**Parameters**

| Param        | Type                       | Description                                         |
|:-------------|:---------------------------|:----------------------------------------------------|
| `ctx`        | *context.Context*          | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*                   | Name of the bucket                                  |
| `opts`       | *minio.ListObjectsOptions* | Options per to list objects                         |

**Return Value**

| Param        | Type                    | Description                                                                           |
|:-------------|:------------------------|:--------------------------------------------------------------------------------------|
| `objectInfo` | *chan minio.ObjectInfo* | Read channel for all objects in the bucket, the object is of the format listed below: |

**minio.ObjectInfo**

| Field                     | Type        | Description                        |
|:--------------------------|:------------|:-----------------------------------|
| `objectInfo.Key`          | *string*    | Name of the object                 |
| `objectInfo.Size`         | *int64*     | Size of the object                 |
| `objectInfo.ETag`         | *string*    | MD5 checksum of the object         |
| `objectInfo.LastModified` | *time.Time* | Time when object was last modified |

```go
ctx, cancel := context.WithCancel(context.Background())

defer cancel()

objectCh := minioClient.ListObjects(ctx, "mybucket", minio.ListObjectsOptions{
	Prefix:    "myprefix",
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

<a name="ListObjectsIter"></a>

### ListObjectsIter(ctx context.Context, bucketName string, opts ListObjectsOptions) iter.Seq[ObjectInfo]

Lists objects in a bucket using an iterator. This is a modern Go 1.23+ alternative to the channel-based ListObjects API.

**Parameters**

| Param        | Type                       | Description                                         |
|:-------------|:---------------------------|:----------------------------------------------------|
| `ctx`        | *context.Context*          | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*                   | Name of the bucket                                  |
| `opts`       | *minio.ListObjectsOptions* | Options to list objects                             |

**Return Value**

| Param      | Type                         | Description                         |
|:-----------|:-----------------------------|:------------------------------------|
| `iterator` | *iter.Seq[minio.ObjectInfo]* | Iterator yielding ObjectInfo values |

**Example**

```go
opts := minio.ListObjectsOptions{
	Prefix:    "myprefix",
	Recursive: true,
}

for object := range minioClient.ListObjectsIter(context.Background(), "mybucket", opts) {
	if object.Err != nil {
		fmt.Println(object.Err)
		return
	}
	fmt.Println(object.Key, object.Size)
}
```

<a name="ListIncompleteUploads"></a>

### ListIncompleteUploads(ctx context.Context, bucketName, prefix string, recursive bool) <- chan ObjectMultipartInfo

Lists partially uploaded objects in a bucket.

**Parameters**

| Param        | Type              | Description                                                                                              |
|:-------------|:------------------|:---------------------------------------------------------------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call                                                      |
| `bucketName` | *string*          | Name of the bucket                                                                                       |
| `prefix`     | *string*          | Prefix of objects that are partially uploaded                                                            |
| `recursive`  | *bool*            | `true` indicates recursive style listing and `false` indicates directory style listing delimited by '/'. |

**Return Value**

| Param           | Type                             | Description                                         |
|:----------------|:---------------------------------|:----------------------------------------------------|
| `multiPartInfo` | *chan minio.ObjectMultipartInfo* | Emits multipart objects of the format listed below: |

**minio.ObjectMultipartInfo**

| Field                       | Type     | Description                               |
|:----------------------------|:---------|:------------------------------------------|
| `multiPartObjInfo.Key`      | *string* | Name of incompletely uploaded object      |
| `multiPartObjInfo.UploadID` | *string* | Upload ID of incompletely uploaded object |
| `multiPartObjInfo.Size`     | *int64*  | Size of incompletely uploaded object      |

**Example**

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

**Parameters**

| Param        | Type              | Description                                         |
|:-------------|:------------------|:----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |
| `tags`       | \**tags.Tags*     | Bucket tags                                         |

**Example**

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

**Parameters**

| Param        | Type              | Description                                         |
|:-------------|:------------------|:----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |

**Return Value**

| Param  | Type          | Description |
|:-------|:--------------|:------------|
| `tags` | \**tags.Tags* | Bucket tags |

**Example**

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

**Parameters**

| Param        | Type              | Description                                         |
|:-------------|:------------------|:----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |

**Example**

```go
err := minioClient.RemoveBucketTagging(context.Background(), "my-bucketname")
if err != nil {
	log.Fatalln(err)
}
```

1.	Object operations --------------------

<a name="AppendObject"></a>

### AppendObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts AppendObjectOptions) (UploadInfo, error)

**Parameters** |Param | Type | Description | |:--- | :--- | :--- | |`ctx` | *context.Context* | Custom Context for timeout/cancellation of the call| |`bucketName`| *string* | Name of bucket | |`objectName`| *string* | Name of Object | |`reader` | *io.Reader* | standard Reader Interface | |`objectSize` | *int64* | Size of the object | |`opts` | *minio.AppendObjectOptions* | Additional Options for Append Operation|

**Return Value** |Param | Type | Description | |:--- | :--- | :--- | |`info`| *minio.UploadInfo* | Information about the newly uploaded or copied object | |`err`| *error* | Standard error |

**minio.AppendObjectOptions** | Field | Type | Description | |:--- | :--- | :--- | |`opts.Progress`| *io.Reader* | A progress reader to indicate progress| |`opts.ChunkSize`| *uint64* | Maximum Append Size | |`opts.DisableContentSha256`| *bool* | Aggressively disable sha256 payload. |

**minio.UploadInfo** | Field | Type | Description | | :--- | :--- | :--- | | `info.Bucket` | *string* | Name of bucket | | `info.Key` | *string* | Name of object | | `info.ETag` | *string* | MD5 checksum of the object | | `info.Size` | *string* | Size of object |

**Example**

```go
opt := minio.AppendObjectOptions{}
info, err := minio.AppendObject(context.Background(), "my-bucket-name", "my-object-name", my_progress_reader, size, opt)
if err != nil {
	log.Fatalln(err)
}
```

<a name="GetObject"></a>

### GetObject(ctx context.Context, bucketName, objectName string, opts GetObjectOptions) (*Object, error)

Returns a stream of the object data. Most of the common errors occur when reading the stream.

**Parameters**

| Param        | Type                     | Description                                                                      |
|:-------------|:-------------------------|:---------------------------------------------------------------------------------|
| `ctx`        | *context.Context*        | Custom context for timeout/cancellation of the call                              |
| `bucketName` | *string*                 | Name of the bucket                                                               |
| `objectName` | *string*                 | Name of the object                                                               |
| `opts`       | *minio.GetObjectOptions* | Options for GET requests specifying additional options like encryption, If-Match |

**minio.GetObjectOptions**

| Field                       | Type                       | Description                                                                                                                                           |
|:----------------------------|:---------------------------|:------------------------------------------------------------------------------------------------------------------------------------------------------|
| `opts.ServerSideEncryption` | *encrypt.ServerSide*       | Interface provided by `encrypt` package to specify server-side-encryption. (For more information see https://godoc.org/github.com/minio/minio-go/v7\) |
| `opts.Internal`             | *minio.AdvancedGetOptions* | This option is intended for internal use by MinIO server. This option should not be set unless the application is aware of intended use.              |

**Return Value**

| Param    | Type             | Description                                                                                                        |
|:---------|:-----------------|:-------------------------------------------------------------------------------------------------------------------|
| `object` | \**minio.Object* | *minio.Object* represents object reader. It implements io.Reader, io.Seeker, io.ReaderAt and io.Closer interfaces. |

**Example**

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

**Parameters**

| Param        | Type                     | Description                                                                      |
|:-------------|:-------------------------|:---------------------------------------------------------------------------------|
| `ctx`        | *context.Context*        | Custom context for timeout/cancellation of the call                              |
| `bucketName` | *string*                 | Name of the bucket                                                               |
| `objectName` | *string*                 | Name of the object                                                               |
| `filePath`   | *string*                 | Path to download object to                                                       |
| `opts`       | *minio.GetObjectOptions* | Options for GET requests specifying additional options like encryption, If-Match |

**Example**

```go
err = minioClient.FGetObject(context.Background(), "mybucket", "myobject", "/tmp/myobject", minio.GetObjectOptions{})
if err != nil {
	fmt.Println(err)
	return
}
```

<a name="PutObjectFanOut"></a>

### PutObjectFanOut(ctx context.Context, bucket string, body io.Reader, fanOutReq ...PutObjectFanOutRequest) ([]PutObjectFanOutResponse, error)

A variant of PutObject instead of writing a single object from a single stream multiple objects are written, defined via a list of *PutObjectFanOutRequest*. Each entry in *PutObjectFanOutRequest* carries an object keyname and its relevant metadata if any. `Key` is mandatory, rest of the other options in *PutObjectFanOutRequest( are optional.

**Parameters**

| Param        | Type                           | Description                                                           |
|:-------------|:-------------------------------|:----------------------------------------------------------------------|
| `ctx`        | *context.Context*              | Custom context for timeout/cancellation of the call                   |
| `bucketName` | *string*                       | Name of the bucket                                                    |
| `fanOutData` | *io.Reader*                    | Any Go type that implements io.Reader                                 |
| `fanOutReq`  | *minio.PutObjectFanOutRequest* | User input list of all the objects that will be created on the server |
|              |                                |                                                                       |

**minio.PutObjectFanOutRequest**

| Field       | Type                            | Description                                |
|:------------|:--------------------------------|:-------------------------------------------|
| `Entries`   | *[]minio.PutObjectFanOutEntyry* | List of object fan out entries             |
| `Checksums` | *map[string]string*             | Checksums for the input data               |
| `SSE`       | _encrypt.ServerSide             | Encryption settings for the entire fan-out |

**minio.PutObjectFanOutEntry**

| Field                | Type                  | Description                                                                                        |
|:---------------------|:----------------------|:---------------------------------------------------------------------------------------------------|
| `Key`                | *string*              | Name of the object                                                                                 |
| `UserMetadata`       | *map[string]string*   | Map of user metadata                                                                               |
| `UserTags`           | *map[string]string*   | Map of user object tags                                                                            |
| `ContentType`        | *string*              | Content type of object, e.g "application/text"                                                     |
| `ContentEncoding`    | *string*              | Content encoding of object, e.g "gzip"                                                             |
| `ContentDisposition` | *string*              | Content disposition of object, "inline"                                                            |
| `ContentLanguage`    | *string*              | Content language of object, e.g "French"                                                           |
| `CacheControl`       | *string*              | Used to specify directives for caching mechanisms in both requests and responses e.g "max-age=600" |
| `Retention`          | *minio.RetentionMode* | Retention mode to be set, e.g "COMPLIANCE"                                                         |
| `RetainUntilDate`    | *time.Time*           | Time until which the retention applied is valid                                                    |

**minio.PutObjectFanOutResponse**

| Field          | Type       | Description                                                     |
|:---------------|:-----------|:----------------------------------------------------------------|
| `Key`          | *string*   | Name of the object                                              |
| `ETag`         | *string*   | ETag opaque unique value of the object                          |
| `VersionID`    | *string*   | VersionID of the uploaded object                                |
| `LastModified` | _time.Time | Last modified time of the latest object                         |
| `Error`        | *error*    | Is non `nil` only when the fan-out for a specific object failed |

<a name="PutObject"></a>

### PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64,opts PutObjectOptions) (info UploadInfo, err error)

Uploads objects that are less than 128MiB in a single PUT operation. For objects that are greater than 128MiB in size, PutObject seamlessly uploads the object as parts of 128MiB or more depending on the actual file size. The max upload size for an object is ~48.83TiB (5GiB * 10000 parts). When using unknown size (-1), the default limit is 5TiB; set `PartSize` in options to upload larger objects.

**Parameters**

| Param        | Type                     | Description                                                                                                                         |
|:-------------|:-------------------------|:------------------------------------------------------------------------------------------------------------------------------------|
| `ctx`        | *context.Context*        | Custom context for timeout/cancellation of the call                                                                                 |
| `bucketName` | *string*                 | Name of the bucket                                                                                                                  |
| `objectName` | *string*                 | Name of the object                                                                                                                  |
| `reader`     | *io.Reader*              | Any Go type that implements io.Reader                                                                                               |
| `objectSize` | *int64*                  | Size of the object being uploaded. Pass -1 if stream size is unknown (Warning: passing -1 will allocate a large amount of memory)   |
| `opts`       | *minio.PutObjectOptions* | Allows user to set optional custom metadata, content headers, encryption keys and number of threads for multipart upload operation. |

**minio.PutObjectOptions**

| Field                          | Type                       | Description                                                                                                                                                                        |
|:-------------------------------|:---------------------------|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `opts.UserMetadata`            | *map[string]string*        | Map of user metadata                                                                                                                                                               |
| `opts.UserTags`                | *map[string]string*        | Map of user object tags                                                                                                                                                            |
| `opts.Progress`                | *io.Reader*                | Reader to fetch progress of an upload                                                                                                                                              |
| `opts.ContentType`             | *string*                   | Content type of object, e.g "application/text"                                                                                                                                     |
| `opts.ContentEncoding`         | *string*                   | Content encoding of object, e.g "gzip"                                                                                                                                             |
| `opts.ContentDisposition`      | *string*                   | Content disposition of object, "inline"                                                                                                                                            |
| `opts.ContentLanguage`         | *string*                   | Content language of object, e.g "French"                                                                                                                                           |
| `opts.CacheControl`            | *string*                   | Used to specify directives for caching mechanisms in both requests and responses e.g "max-age=600"                                                                                 |
| `opts.Mode`                    | \**minio.RetentionMode*    | Retention mode to be set, e.g "COMPLIANCE"                                                                                                                                         |
| `opts.RetainUntilDate`         | \**time.Time*              | Time until which the retention applied is valid                                                                                                                                    |
| `opts.ServerSideEncryption`    | *encrypt.ServerSide*       | Interface provided by `encrypt` package to specify server-side-encryption. (For more information see https://godoc.org/github.com/minio/minio-go/v7\)                              |
| `opts.StorageClass`            | *string*                   | Specify storage class for the object. Supported values for MinIO server are `REDUCED_REDUNDANCY` and `STANDARD`                                                                    |
| `opts.WebsiteRedirectLocation` | *string*                   | Specify a redirect for the object, to another object in the same bucket or to a external URL.                                                                                      |
| `opts.SendContentMd5`          | *bool*                     | Specify if you'd like to send `content-md5` header with PutObject operation. Note that setting this flag will cause higher memory usage because of in-memory `md5sum` calculation. |
| `opts.PartSize`                | *uint64*                   | Specify a custom part size used for uploading the object                                                                                                                           |
| `opts.Internal`                | *minio.AdvancedPutOptions* | This option is intended for internal use by MinIO server and should not be set unless the application is aware of intended use.                                                    |
|                                |                            |                                                                                                                                                                                    |

**minio.UploadInfo**

| Field            | Type     | Description                              |
|:-----------------|:---------|:-----------------------------------------|
| `info.ETag`      | *string* | The ETag of the new object               |
| `info.VersionID` | *string* | The version identifier of the new object |

**Example**

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

uploadInfo, err := minioClient.PutObject(context.Background(), "mybucket", "myobject", file, fileStat.Size(), minio.PutObjectOptions{ContentType: "application/octet-stream"})
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

**Parameters**

| Param | Type                    | Description                                         |
|:------|:------------------------|:----------------------------------------------------|
| `ctx` | *context.Context*       | Custom context for timeout/cancellation of the call |
| `dst` | *minio.CopyDestOptions* | Argument describing the destination object          |
| `src` | *minio.CopySrcOptions*  | Argument describing the source object               |

**minio.UploadInfo**

| Field            | Type     | Description                              |
|:-----------------|:---------|:-----------------------------------------|
| `info.ETag`      | *string* | The ETag of the new object               |
| `info.VersionID` | *string* | The version identifier of the new object |

**Example**

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
uploadInfo, err := minioClient.CopyObject(context.Background(), dstOpts, srcOpts)
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
	Bucket:               "my-sourcebucketname",
	Object:               "my-sourceobjectname",
	MatchETag:            "31624deb84149d2f8ef9c385918b653a",
	MatchModifiedSince:   time.Date(2014, time.April, 1, 0, 0, 0, 0, time.UTC),
	MatchUnmodifiedSince: time.Date(2014, time.April, 23, 0, 0, 0, 0, time.UTC),
	Start:                0,
	End:                  1024*1024 - 1,
}

// Destination object
dstOpts := minio.CopyDestOptions{
	Bucket: "my-bucketname",
	Object: "my-objectname",
}

// Copy object call
_, err = minioClient.CopyObject(context.Background(), dstOpts, srcOpts)
if err != nil {
	fmt.Println(err)
	return
}

fmt.Println("Successfully copied object:", uploadInfo)

```

<a name="ComposeObject"></a>

### ComposeObject(ctx context.Context, dst minio.CopyDestOptions, srcs ...minio.CopySrcOptions) (UploadInfo, error)

Create an object by concatenating a list of source objects using server-side copying.

**Parameters**

| Param  | Type                      | Description                                                                 |
|:-------|:--------------------------|:----------------------------------------------------------------------------|
| `ctx`  | *context.Context*         | Custom context for timeout/cancellation of the call                         |
| `dst`  | *minio.CopyDestOptions*   | Struct with info about the object to be created.                            |
| `srcs` | *...minio.CopySrcOptions* | Slice of struct with info about source objects to be concatenated in order. |

**minio.UploadInfo**

| Field            | Type     | Description                              |
|:-----------------|:---------|:-----------------------------------------|
| `info.ETag`      | *string* | The ETag of the new object               |
| `info.VersionID` | *string* | The version identifier of the new object |

**Example**

```go
// Prepare source decryption key (here we assume same key to
// decrypt all source objects.)
sseSrc := encrypt.DefaultPBKDF([]byte("password"), []byte("salt"))

// Source objects to concatenate. We also specify decryption
// key for each
src1Opts := minio.CopySrcOptions{
	Bucket:     "bucket1",
	Object:     "object1",
	Encryption: sseSrc,
	MatchETag:  "31624deb84149d2f8ef9c385918b653a",
}

src2Opts := minio.CopySrcOptions{
	Bucket:     "bucket2",
	Object:     "object2",
	Encryption: sseSrc,
	MatchETag:  "f8ef9c385918b653a31624deb84149d2",
}

src3Opts := minio.CopySrcOptions{
	Bucket:     "bucket3",
	Object:     "object3",
	Encryption: sseSrc,
	MatchETag:  "5918b653a31624deb84149d2f8ef9c38",
}

// Prepare destination encryption key
sseDst := encrypt.DefaultPBKDF([]byte("new-password"), []byte("new-salt"))

// Create destination info
dstOpts := CopyDestOptions{
	Bucket:     "bucket",
	Object:     "object",
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

### FPutObject(ctx context.Context, bucketName, objectName, filePath string, opts PutObjectOptions) (info UploadInfo, err error)

Uploads contents from a file to objectName.

FPutObject uploads objects that are less than 128MiB in a single PUT operation. For objects that are greater than the 128MiB in size, FPutObject seamlessly uploads the object in chunks of 128MiB or more depending on the actual file size. The max upload size for an object is ~48.83TiB (5GiB * 10000 parts).

**Parameters**

| Param        | Type                     | Description                                                                                                                                                                                                                                                                                 |
|:-------------|:-------------------------|:--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `ctx`        | *context.Context*        | Custom context for timeout/cancellation of the call                                                                                                                                                                                                                                         |
| `bucketName` | *string*                 | Name of the bucket                                                                                                                                                                                                                                                                          |
| `objectName` | *string*                 | Name of the object                                                                                                                                                                                                                                                                          |
| `filePath`   | *string*                 | Path to file to be uploaded                                                                                                                                                                                                                                                                 |
| `opts`       | *minio.PutObjectOptions* | Pointer to struct that allows user to set optional custom metadata, content-type, content-encoding, content-disposition, content-language and cache-control headers, pass encryption module for encrypting objects, and optionally configure number of threads for multipart put operation. |

**minio.UploadInfo**

| Field            | Type     | Description                              |
|:-----------------|:---------|:-----------------------------------------|
| `info.ETag`      | *string* | The ETag of the new object               |
| `info.VersionID` | *string* | The version identifier of the new object |

**Example**

```go
uploadInfo, err := minioClient.FPutObject(context.Background(), "my-bucketname", "my-objectname", "my-filename.csv", minio.PutObjectOptions{
	ContentType: "application/csv",
})
if err != nil {
	fmt.Println(err)
	return
}
fmt.Println("Successfully uploaded object: ", uploadInfo)
```

<a name="StatObject"></a>

### StatObject(ctx context.Context, bucketName, objectName string, opts StatObjectOptions) (ObjectInfo, error)

Fetch metadata of an object.

**Parameters**

| Param        | Type                      | Description                                                                                |
|:-------------|:--------------------------|:-------------------------------------------------------------------------------------------|
| `ctx`        | *context.Context*         | Custom context for timeout/cancellation of the call                                        |
| `bucketName` | *string*                  | Name of the bucket                                                                         |
| `objectName` | *string*                  | Name of the object                                                                         |
| `opts`       | *minio.StatObjectOptions* | Options for GET info/stat requests specifying additional options like encryption, If-Match |

**Return Value**

| Param     | Type               | Description             |
|:----------|:-------------------|:------------------------|
| `objInfo` | *minio.ObjectInfo* | Object stat information |

**minio.ObjectInfo**

| Field                  | Type        | Description                        |
|:-----------------------|:------------|:-----------------------------------|
| `objInfo.LastModified` | *time.Time* | Time when object was last modified |
| `objInfo.ETag`         | *string*    | MD5 checksum of the object         |
| `objInfo.ContentType`  | *string*    | Content type of the object         |
| `objInfo.Size`         | *int64*     | Size of the object                 |

**Example**

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

**Parameters**

| Param        | Type                        | Description                                         |
|:-------------|:----------------------------|:----------------------------------------------------|
| `ctx`        | *context.Context*           | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*                    | Name of the bucket                                  |
| `objectName` | *string*                    | Name of the object                                  |
| `opts`       | *minio.RemoveObjectOptions* | Allows user to set options                          |

**minio.RemoveObjectOptions**

| Field                   | Type                          | Description                                                                                                                     |
|:------------------------|:------------------------------|:--------------------------------------------------------------------------------------------------------------------------------|
| `opts.GovernanceBypass` | *bool*                        | Set the bypass governance header to delete an object locked with GOVERNANCE mode                                                |
| `opts.VersionID`        | *string*                      | Version ID of the object to delete                                                                                              |
| `opts.Internal`         | *minio.AdvancedRemoveOptions* | This option is intended for internal use by MinIO server and should not be set unless the application is aware of intended use. |

```go
opts := minio.RemoveObjectOptions{
	GovernanceBypass: true,
	VersionID:        "myversionid",
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

**Parameters**

| Param        | Type                              | Description                                                                |
|:-------------|:----------------------------------|:---------------------------------------------------------------------------|
| `ctx`        | *context.Context*                 | Custom context for timeout/cancellation of the call                        |
| `bucketName` | *string*                          | Name of the bucket                                                         |
| `objectName` | *string*                          | Name of the object                                                         |
| `opts`       | *minio.PutObjectRetentionOptions* | Allows user to set options like retention mode, expiry date and version id |

<a name="RemoveObjects"></a>

### RemoveObjects(ctx context.Context, bucketName string, objectsCh <-chan ObjectInfo, opts RemoveObjectsOptions) <-chan RemoveObjectError

Removes a list of objects obtained from an input channel. The call sends a delete request to the server up to 1000 objects at a time. The errors observed are sent over the error channel.

Parameters

| Param        | Type                         | Description                                         |
|:-------------|:-----------------------------|:----------------------------------------------------|
| `ctx`        | *context.Context*            | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*                     | Name of the bucket                                  |
| `objectsCh`  | *chan minio.ObjectInfo*      | Channel of objects to be removed                    |
| `opts`       | *minio.RemoveObjectsOptions* | Allows user to set options                          |

**minio.RemoveObjectsOptions**

| Field                   | Type   | Description                                                                      |
|:------------------------|:-------|:---------------------------------------------------------------------------------|
| `opts.GovernanceBypass` | *bool* | Set the bypass governance header to delete an object locked with GOVERNANCE mode |

**Return Values**

| Param     | Type                             | Description                                              |
|:----------|:---------------------------------|:---------------------------------------------------------|
| `errorCh` | *<-chan minio.RemoveObjectError* | Receive-only channel of errors observed during deletion. |

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

<a name="RemoveObjectsWithResult"></a>

### RemoveObjectsWithResult(ctx context.Context, bucketName string, objectsCh <-chan ObjectInfo, opts RemoveObjectsOptions) <-chan RemoveObjectResult

Removes a list of objects and returns both successful deletions and errors. This is an enhanced version of RemoveObjects that provides complete deletion results.

**Parameters**

| Param        | Type                         | Description                                         |
|:-------------|:-----------------------------|:----------------------------------------------------|
| `ctx`        | *context.Context*            | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*                     | Name of the bucket                                  |
| `objectsCh`  | *chan minio.ObjectInfo*      | Channel of objects to be removed                    |
| `opts`       | *minio.RemoveObjectsOptions* | Allows user to set options                          |

**Return Values**

| Param      | Type                              | Description                                            |
|:-----------|:----------------------------------|:-------------------------------------------------------|
| `resultCh` | *<-chan minio.RemoveObjectResult* | Channel of results including both successes and errors |

**minio.RemoveObjectResult**

| Field        | Type     | Description                               |
|:-------------|:---------|:------------------------------------------|
| `ObjectName` | *string* | Name of the object                        |
| `VersionID`  | *string* | Version ID of the object (if versioned)   |
| `Err`        | *error*  | Error during deletion (nil if successful) |

**Example**

```go
objectsCh := make(chan minio.ObjectInfo)

go func() {
	defer close(objectsCh)
	for object := range minioClient.ListObjects(context.Background(), "my-bucketname", minio.ListObjectsOptions{Prefix: "my-prefixname", Recursive: true}) {
		if object.Err != nil {
			log.Fatalln(object.Err)
		}
		objectsCh <- object
	}
}()

opts := minio.RemoveObjectsOptions{
	GovernanceBypass: true,
}

successCount := 0
errorCount := 0
for result := range minioClient.RemoveObjectsWithResult(context.Background(), "my-bucketname", objectsCh, opts) {
	if result.Err != nil {
		fmt.Printf("Error deleting %s: %v\n", result.ObjectName, result.Err)
		errorCount++
	} else {
		fmt.Printf("Successfully deleted %s\n", result.ObjectName)
		successCount++
	}
}
fmt.Printf("Deleted: %d, Errors: %d\n", successCount, errorCount)
```

<a name="RemoveObjectsWithIter"></a>

### RemoveObjectsWithIter(ctx context.Context, bucketName string, objectsIter iter.Seq[ObjectInfo], opts RemoveObjectsOptions) (iter.Seq[RemoveObjectResult], error)

Iterator-based version of RemoveObjects for Go 1.23+. Removes objects using an iterator input and returns results via an iterator.

**Parameters**

| Param         | Type                         | Description                                         |
|:--------------|:-----------------------------|:----------------------------------------------------|
| `ctx`         | *context.Context*            | Custom context for timeout/cancellation of the call |
| `bucketName`  | *string*                     | Name of the bucket                                  |
| `objectsIter` | *iter.Seq[minio.ObjectInfo]* | Iterator of objects to be removed                   |
| `opts`        | *minio.RemoveObjectsOptions* | Allows user to set options                          |

**Return Values**

| Param        | Type                                 | Description                          |
|:-------------|:-------------------------------------|:-------------------------------------|
| `resultIter` | *iter.Seq[minio.RemoveObjectResult]* | Iterator yielding removal results    |
| `err`        | *error*                              | Error initializing removal operation |

**Example**

```go
opts := minio.ListObjectsOptions{
	Prefix:    "my-prefixname",
	Recursive: true,
}

removeOpts := minio.RemoveObjectsOptions{
	GovernanceBypass: true,
}

for err := range minioClient.RemoveObjectsWithIter(context.Background(), "my-bucketname", minioClient.ListObjectsIter(context.Background(), "my-bucketname", opts), removeOpts) {
	fmt.Printf("Error detected during deletion: %v\n", err)
}
```

<a name="GetObjectRetention"></a>

### GetObjectRetention(ctx context.Context, bucketName, objectName, versionID string) (mode *RetentionMode, retainUntilDate *time.Time, err error)

Returns retention set on a given object.

**Parameters**

| Param        | Type              | Description                                         |
|:-------------|:------------------|:----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |
| `objectName` | *string*          | Name of the object                                  |
| `versionID`  | *string*          | Version ID of the object                            |

```go
mode, retainUntilDate, err := minioClient.GetObjectRetention(context.Background(), "mybucket", "myobject", "")
if err != nil {
	fmt.Println(err)
	return
}
fmt.Printf("Retention mode: %v, Retain until: %v\n", mode, retainUntilDate)
```

<a name="PutObjectLegalHold"></a>

### PutObjectLegalHold(ctx context.Context, bucketName, objectName string, opts minio.PutObjectLegalHoldOptions) error

Applies legal-hold onto an object.

**Parameters**

| Param        | Type                              | Description                                           |
|:-------------|:----------------------------------|:------------------------------------------------------|
| `ctx`        | *context.Context*                 | Custom context for timeout/cancellation of the call   |
| `bucketName` | *string*                          | Name of the bucket                                    |
| `objectName` | *string*                          | Name of the object                                    |
| `opts`       | *minio.PutObjectLegalHoldOptions* | Allows user to set options like status and version id |

*minio.PutObjectLegalHoldOptions*

| Field            | Type                      | Description                                    |
|:-----------------|:--------------------------|:-----------------------------------------------|
| `opts.Status`    | \**minio.LegalHoldStatus* | Legal-Hold status to be set                    |
| `opts.VersionID` | *string*                  | Version ID of the object to apply retention on |

```go
s := minio.LegalHoldEnabled
opts := minio.PutObjectLegalHoldOptions{
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

**Parameters**

| Param        | Type                              | Description                                         |
|:-------------|:----------------------------------|:----------------------------------------------------|
| `ctx`        | *context.Context*                 | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*                          | Name of the bucket                                  |
| `objectName` | *string*                          | Name of the object                                  |
| `opts`       | *minio.GetObjectLegalHoldOptions* | Allows user to set options like version id          |

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

| Param        | Type                  | Description                                         |
|:-------------|:----------------------|:----------------------------------------------------|
| `ctx`        | *context.Context*     | Custom context for timeout/cancellation of the call |
| `ctx`        | *context.Context*     | Request context                                     |
| `bucketName` | *string*              | Name of the bucket                                  |
| `objectName` | *string*              | Name of the object                                  |
| `options`    | *SelectObjectOptions* | Query Options                                       |

**Return Values**

| Param           | Type            | Description                                                                                     |
|:----------------|:----------------|:------------------------------------------------------------------------------------------------|
| `SelectResults` | *SelectResults* | Is an io.ReadCloser object which can be directly passed to csv.NewReader for processing output. |

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

### PutObjectTagging(ctx context.Context, bucketName, objectName string, otags *tags.Tags, opts PutObjectTaggingOptions) error

set new object Tags to the given object, replaces/overwrites any existing tags.

**Parameters**

| Param        | Type              | Description                                         |
|:-------------|:------------------|:----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |
| `objectName` | *string*          | Name of the object                                  |
| `objectTags` | \**tags.Tags*     | Map with Object Tag's Key and Value                 |

**Example**

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

**Parameters**

| Param        | Type              | Description                                         |
|:-------------|:------------------|:----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |
| `objectName` | *string*          | Name of the object                                  |

**Example**

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

**Parameters**

| Param        | Type              | Description                                         |
|:-------------|:------------------|:----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |
| `objectName` | *string*          | Name of the object                                  |

**Example**

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

**Parameters**

| Param        | Type                  | Description                                         |
|:-------------|:----------------------|:----------------------------------------------------|
| `ctx`        | *context.Context*     | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*              | Name of the bucket                                  |
| `objectName` | *string*              | Name of the object                                  |
| `versionID`  | *string*              | Version ID of the object                            |
| `opts`       | _minio.RestoreRequest | Restore request options                             |

**Example**

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

**Parameters**

| Param        | Type                            | Description                                                     |
|:-------------|:--------------------------------|:----------------------------------------------------------------|
| `ctx`        | *context.Context*               | Custom context for timeout/cancellation of the call             |
| `bucketName` | *string*                        | Name of the bucket                                              |
| `objectName` | *string*                        | Name of the object                                              |
| `opts`       | *minio.ObjectAttributesOptions* | Configuration for pagination and selection of object attributes |

**minio.ObjectAttributesOptions**

| Field                       | Type                 | Description                                                                                                                                                 |
|:----------------------------|:---------------------|:------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `opts.ServerSideEncryption` | *encrypt.ServerSide* | Interface provided by `encrypt` package to specify server-side-encryption. (For more information see https://godoc.org/github.com/minio/minio-go/v7\)       |
| `opts.MaxParts`             | _int                 | This option defines how many parts should be returned by the API                                                                                            |
| `opts.VersionID`            | _string              | VersionID defines which version of the object will be used                                                                                                  |
| `opts.PartNumberMarker`     | _int                 | This options defines which part number pagination will start after, the part which number is equal to PartNumberMarker will not be included in the response |

**Return Value**

| Param              | Type                       | Description                                                                        |
|:-------------------|:---------------------------|:-----------------------------------------------------------------------------------|
| `objectAttributes` | \**minio.ObjectAttributes* | *minio.ObjectAttributes* contains the information about the object and it's parts. |

**Example**

```go
objectAttributes, err := c.GetObjectAttributes(
	context.Background(),
	"your-bucket",
	"your-object",
	minio.ObjectAttributesOptions{
		VersionID:      "object-version-id",
		NextPartMarker: 0,
		MaxParts:       100,
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

**Parameters**

| Param        | Type              | Description                                         |
|:-------------|:------------------|:----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |
| `objectName` | *string*          | Name of the object                                  |

**Example**

```go
err = minioClient.RemoveIncompleteUpload(context.Background(), "mybucket", "myobject")
if err != nil {
	fmt.Println(err)
	return
}
```

1.	Presigned operations -----------------------

<a name="PresignedGetObject"></a>

### PresignedGetObject(ctx context.Context, bucketName, objectName string, expiry time.Duration, reqParams url.Values) (*url.URL, error)

Generates a presigned URL for HTTP GET operations. Browsers/Mobile clients may point to this URL to directly download objects even if the bucket is private. This presigned URL can have an associated expiration time in seconds after which it is no longer operational. The maximum expiry is 604800 seconds (i.e. 7 days) and minimum is 1 second.

**Parameters**

| Param        | Type              | Description                                                                                                                                          |
|:-------------|:------------------|:-----------------------------------------------------------------------------------------------------------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call                                                                                                  |
| `bucketName` | *string*          | Name of the bucket                                                                                                                                   |
| `objectName` | *string*          | Name of the object                                                                                                                                   |
| `expiry`     | *time.Duration*   | Expiry of presigned URL in seconds                                                                                                                   |
| `reqParams`  | *url.Values*      | Additional response header overrides supports *response-expires*, *response-content-type*, *response-cache-control*, *response-content-disposition*. |

**Example**

```go
// Set request parameters for content-disposition.
reqParams := make(url.Values)
reqParams.Set("response-content-disposition", "attachment; filename=\"your-filename.txt\"")

// Generates a presigned url which expires in a day.
presignedURL, err := minioClient.PresignedGetObject(context.Background(), "mybucket", "myobject", time.Second*24*60*60, reqParams)
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

**Parameters**

| Param        | Type              | Description                                         |
|:-------------|:------------------|:----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |
| `objectName` | *string*          | Name of the object                                  |
| `expiry`     | *time.Duration*   | Expiry of presigned URL in seconds                  |

**Example**

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

**Parameters**

| Param        | Type              | Description                                                                                                                                          |
|:-------------|:------------------|:-----------------------------------------------------------------------------------------------------------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call                                                                                                  |
| `bucketName` | *string*          | Name of the bucket                                                                                                                                   |
| `objectName` | *string*          | Name of the object                                                                                                                                   |
| `expiry`     | *time.Duration*   | Expiry of presigned URL in seconds                                                                                                                   |
| `reqParams`  | *url.Values*      | Additional response header overrides supports *response-expires*, *response-content-type*, *response-cache-control*, *response-content-disposition*. |

**Example**

```go
// Set request parameters for content-disposition.
reqParams := make(url.Values)
reqParams.Set("response-content-disposition", "attachment; filename=\"your-filename.txt\"")

// Generates a presigned url which expires in a day.
presignedURL, err := minioClient.PresignedHeadObject(context.Background(), "mybucket", "myobject", time.Second*24*60*60, reqParams)
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

1.	Bucket policy/notification operations ----------------------------------------

<a name="SetBucketPolicy"></a>

### SetBucketPolicy(ctx context.Context, bucketname, policy string) error

Set access permissions on bucket or an object prefix.

**Parameters**

| Param        | Type              | Description                                         |
|:-------------|:------------------|:----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |
| `policy`     | *string*          | Policy to be set                                    |

**Return Values**

| Param | Type    | Description    |
|:------|:--------|:---------------|
| `err` | *error* | Standard Error |

**Example**

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

**Parameters**

| Param        | Type              | Description                                         |
|:-------------|:------------------|:----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |

**Return Values**

| Param    | Type     | Description                     |
|:---------|:---------|:--------------------------------|
| `policy` | *string* | Policy returned from the server |
| `err`    | *error*  | Standard Error                  |

**Example**

```go
policy, err := minioClient.GetBucketPolicy(context.Background(), "my-bucketname")
if err != nil {
	log.Fatalln(err)
}
```

<a name="GetBucketNotification"></a>

### GetBucketNotification(ctx context.Context, bucketName string) (notification.Configuration, error)

Get notification configuration on a bucket.

**Parameters**

| Param        | Type              | Description                                         |
|:-------------|:------------------|:----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |

**Return Values**

| Param    | Type                         | Description                                           |
|:---------|:-----------------------------|:------------------------------------------------------|
| `config` | *notification.Configuration* | structure which holds all notification configurations |
| `err`    | *error*                      | Standard Error                                        |

**Example**

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

**Parameters**

| Param        | Type                         | Description                                                 |
|:-------------|:-----------------------------|:------------------------------------------------------------|
| `ctx`        | *context.Context*            | Custom context for timeout/cancellation of the call         |
| `bucketName` | *string*                     | Name of the bucket                                          |
| `config`     | *notification.Configuration* | Represents the XML to be sent to the configured web service |

**Return Values**

| Param | Type    | Description    |
|:------|:--------|:---------------|
| `err` | *error* | Standard Error |

**Example**

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

**Parameters**

| Param        | Type              | Description                                         |
|:-------------|:------------------|:----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |

**Return Values**

| Param | Type    | Description    |
|:------|:--------|:---------------|
| `err` | *error* | Standard Error |

**Example**

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

-	'Records' holds the notifications received from the server.
-	'Err' indicates any error while processing the received notifications.

NOTE: Notification channel is closed at the first occurrence of an error.

**Parameters**

| Param        | Type       | Description                                    |
|:-------------|:-----------|:-----------------------------------------------|
| `bucketName` | *string*   | Bucket to listen notifications on              |
| `prefix`     | *string*   | Object key prefix to filter notifications for  |
| `suffix`     | *string*   | Object key suffix to filter notifications for  |
| `events`     | *[]string* | Enables notifications for specific event types |

**Return Values**

| Param              | Type                     | Description                     |
|:-------------------|:-------------------------|:--------------------------------|
| `notificationInfo` | *chan notification.Info* | Channel of bucket notifications |

**minio.NotificationInfo**

|Field |Type |Description | |`notificationInfo.Records` | *[]notification.Event* | Collection of notification events | |`notificationInfo.Err` | *error* | Carries any error occurred during the operation (Standard Error) |

**Example**

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

-	'Records' holds the notifications received from the server.
-	'Err' indicates any error while processing the received notifications.

NOTE: Notification channel is closed at the first occurrence of an error.

**Parameters**

| Param        | Type       | Description                                    |
|:-------------|:-----------|:-----------------------------------------------|
| `bucketName` | *string*   | Bucket to listen notifications on              |
| `prefix`     | *string*   | Object key prefix to filter notifications for  |
| `suffix`     | *string*   | Object key suffix to filter notifications for  |
| `events`     | *[]string* | Enables notifications for specific event types |

**Return Values**

| Param              | Type                     | Description                        |
|:-------------------|:-------------------------|:-----------------------------------|
| `notificationInfo` | *chan notification.Info* | Read channel for all notifications |

**minio.NotificationInfo**

|Field |Type |Description | |`notificationInfo.Records` | *[]notification.Event* | Collection of notification events | |`notificationInfo.Err` | *error* | Carries any error occurred during the operation (Standard Error) |

**Example**

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

### SetBucketLifecycle(ctx context.Context, bucketName string, config *lifecycle.Configuration) error

Set lifecycle on bucket or an object prefix.

**Parameters**

| Param        | Type                      | Description                                         |
|:-------------|:--------------------------|:----------------------------------------------------|
| `ctx`        | *context.Context*         | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*                  | Name of the bucket                                  |
| `config`     | *lifecycle.Configuration* | Lifecycle to be set                                 |

**Return Values**

| Param | Type    | Description    |
|:------|:--------|:---------------|
| `err` | *error* | Standard Error |

**Example**

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

### GetBucketLifecycle(ctx context.Context, bucketName string) (*lifecycle.Configuration, error)

Get lifecycle on a bucket or a prefix.

**Parameters**

| Param        | Type              | Description                                         |
|:-------------|:------------------|:----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |

**Return Values**

| Param    | Type                      | Description                        |
|:---------|:--------------------------|:-----------------------------------|
| `config` | *lifecycle.Configuration* | Lifecycle returned from the server |
| `err`    | *error*                   | Standard Error                     |

**Example**

```go
lifecycle, err := minioClient.GetBucketLifecycle(context.Background(), "my-bucketname")
if err != nil {
	log.Fatalln(err)
}
```

<a name="SetBucketEncryption"></a>

### SetBucketEncryption(ctx context.Context, bucketName string, config *sse.Configuration) error

Set default encryption configuration on a bucket.

**Parameters**

| Param        | Type                | Description                                                     |
|:-------------|:--------------------|:----------------------------------------------------------------|
| `ctx`        | *context.Context*   | Custom context for timeout/cancellation of the call             |
| `bucketName` | *string*            | Name of the bucket                                              |
| `config`     | *sse.Configuration* | Structure that holds default encryption configuration to be set |

**Return Values**

| Param | Type    | Description    |
|:------|:--------|:---------------|
| `err` | *error* | Standard Error |

**Example**

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

**Parameters**

| Param        | Type              | Description                                         |
|:-------------|:------------------|:----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |

**Return Values**

| Param    | Type                | Description                                           |
|:---------|:--------------------|:------------------------------------------------------|
| `config` | *sse.Configuration* | Structure that holds default encryption configuration |
| `err`    | *error*             | Standard Error                                        |

**Example**

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

### RemoveBucketEncryption(ctx context.Context, bucketName string) error

Remove default encryption configuration set on a bucket.

**Parameters**

| Param        | Type              | Description                                         |
|:-------------|:------------------|:----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |

**Return Values**

| Param | Type    | Description    |
|:------|:--------|:---------------|
| `err` | *error* | Standard Error |

**Example**

```go
err := s3Client.RemoveBucketEncryption(context.Background(), "my-bucketname")
if err != nil {
	log.Fatalln(err)
}
// "my-bucket" is successfully deleted/removed.
```

<a name="SetObjectLockConfig"></a>

### SetObjectLockConfig(ctx context.Context, bucketName string, mode *RetentionMode, validity *uint, unit *ValidityUnit) error

Set object lock configuration in given bucket. mode, validity and unit are either all set or all nil.

**Parameters**

| Param        | Type              | Description                                         |
|:-------------|:------------------|:----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |
| `mode`       | *RetentionMode*   | Retention mode to be set                            |
| `validity`   | *uint*            | Validity period to be set                           |
| `unit`       | *ValidityUnit*    | Unit of validity period                             |

**Return Values**

| Param | Type    | Description    |
|:------|:--------|:---------------|
| `err` | *error* | Standard Error |

**Example**

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

### GetObjectLockConfig(ctx context.Context, bucketName string) (objectLock string, mode *RetentionMode, validity *uint, unit *ValidityUnit, err error)

Get object lock configuration of given bucket.

**Parameters**

| Param        | Type              | Description                                         |
|:-------------|:------------------|:----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |

**Return Values**

| Param        | Type            | Description             |
|:-------------|:----------------|:------------------------|
| `objectLock` | *objectLock*    | lock enabled status     |
| `mode`       | *RetentionMode* | Current retention mode  |
| `validity`   | *uint*          | Current validity period |
| `unit`       | *ValidityUnit*  | Unit of validity period |
| `err`        | *error*         | Standard Error          |

**Example**

```go
enabled, mode, validity, unit, err := minioClient.GetObjectLockConfig(context.Background(), "my-bucketname")
if err != nil {
	log.Fatalln(err)
}
fmt.Println("object lock is %s for this bucket", enabled)
if mode != nil {
	fmt.Printf("%v mode is enabled for %v %v for bucket 'my-bucketname'\n", *mode, *validity, *unit)
} else {
	fmt.Println("No mode is enabled for bucket 'my-bucketname'")
}
```

<a name="EnableVersioning"></a>

### EnableVersioning(ctx context.Context, bucketName string) error

Enable bucket versioning support.

**Parameters**

| Param        | Type              | Description                                         |
|:-------------|:------------------|:----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |

**Return Values**

| Param | Type    | Description    |
|:------|:--------|:---------------|
| `err` | *error* | Standard Error |

**Example**

```go
err := minioClient.EnableVersioning(context.Background(), "my-bucketname")
if err != nil {
	log.Fatalln(err)
}

fmt.Println("versioning enabled for bucket 'my-bucketname'")
```

<a name="SuspendVersioning"></a>

### SuspendVersioning(ctx context.Context, bucketName string) error

Suspend bucket versioning support.

**Parameters**

| Param        | Type              | Description                                         |
|:-------------|:------------------|:----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |

**Return Values**

| Param | Type    | Description    |
|:------|:--------|:---------------|
| `err` | *error* | Standard Error |

**Example**

```go
err := minioClient.SuspendVersioning(context.Background(), "my-bucketname")
if err != nil {
	log.Fatalln(err)
}

fmt.Println("versioning suspended for bucket 'my-bucketname'")
```

<a name="GetBucketVersioning"></a>

### GetBucketVersioning(ctx context.Context, bucketName string) (BucketVersioningConfiguration, error)

Get versioning configuration set on a bucket.

**Parameters**

| Param        | Type              | Description                                         |
|:-------------|:------------------|:----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |

**Return Values**

| Param           | Type                                  | Description                                   |
|:----------------|:--------------------------------------|:----------------------------------------------|
| `configuration` | *minio.BucketVersioningConfiguration* | Structure that holds versioning configuration |
| `err`           | *error*                               | Standard Error                                |

**Example**

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

### SetBucketReplication(ctx context.Context, bucketName string, cfg replication.Config) error

Set replication configuration on a bucket. Role can be obtained by first defining the replication target on MinIO using `mc admin bucket remote set` to associate the source and destination buckets for replication with the replication endpoint.

**Parameters**

| Param        | Type                 | Description                                         |
|:-------------|:---------------------|:----------------------------------------------------|
| `ctx`        | *context.Context*    | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*             | Name of the bucket                                  |
| `cfg`        | *replication.Config* | Replication configuration to be set                 |

**Return Values**

| Param | Type    | Description    |
|:------|:--------|:---------------|
| `err` | *error* | Standard Error |

**Example**

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
replicationConfig.Role = "arn:minio:s3::598361bf-3cec-49a7-b529-ce870a34d759:*"
err = minioClient.SetBucketReplication(context.Background(), "my-bucketname", replicationConfig)
if err != nil {
	fmt.Println(err)
	return
}
```

<a name="GetBucketReplication"></a>

### GetBucketReplication(ctx context.Context, bucketName string) (replication.Config, error)

Get current replication config on a bucket.

**Parameters**

| Param        | Type              | Description                                         |
|:-------------|:------------------|:----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |

**Return Values**

| Param         | Type                 | Description                                 |
|:--------------|:---------------------|:--------------------------------------------|
| `replication` | *replication.Config* | Replication config returned from the server |
| `err`         | *error*              | Standard Error                              |

**Example**

```go
replication, err := minioClient.GetBucketReplication(context.Background(), "my-bucketname")
if err != nil {
	log.Fatalln(err)
}
```

<a name="RemoveBucketReplication"></a>

### RemoveBucketReplication(ctx context.Context, bucketName string) error

Removes replication configuration on a bucket.

**Parameters**

| Param        | Type              | Description                                         |
|:-------------|:------------------|:----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |

**Return Values**

| Param | Type    | Description    |
|:------|:--------|:---------------|
| `err` | *error* | Standard Error |

**Example**

```go
err = minioClient.RemoveBucketReplication(context.Background(), "my-bucketname")
if err != nil {
	fmt.Println(err)
	return
}
```

<a name="CancelBucketReplicationResync"></a>

### CancelBucketReplicationResync(ctx context.Context, bucketName string, tgtArn string) (id string, err error)

Cancels in progress replication resync (MinIO AiStor Only API)

**Parameters**

| Param        | Type              | Description                                        |
|:-------------|:------------------|:---------------------------------------------------|
| `ctx`        | *context.Context* | Custom context of timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                 |
| `tgtArn`     | *string*          | Target Amazon Resource Name                        |

**Return Values** |Param |Type |Description | |:---|:--|:---| |`id`|*string*| Recieved upon successful cancellation of replication resync| |`err`| *error*| Standard Error|

**Example**

```go
id, err := minioClient.CancelBucketReplicationResync(context.Background(), "my-bucket-name", "my-target-arn")
if err != nil {
	fmt.Println(err)
	return
}
```

<a name="GetBucketReplicationMetrics"></a>

### GetBucketReplicationMetrics(ctx context.Context, bucketName string) (replication.Metrics, error)

Get latest replication metrics on a bucket. This is a MinIO specific extension.

**Parameters**

| Param        | Type              | Description                                         |
|:-------------|:------------------|:----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |

**Return Values**

| Param     | Type                  | Description                                  |
|:----------|:----------------------|:---------------------------------------------|
| `metrics` | *replication.Metrics* | Replication metrics returned from the server |
| `err`     | *error*               | Standard Error                               |

**Example**

```go
replMetrics, err := minioClient.GetBucketReplicationMetrics(context.Background(), "my-bucketname")
if err != nil {
	log.Fatalln(err)
}
```

<a name="GetBucketReplicationMetricsV2"></a>

### GetBucketReplicationMetricsV2(ctx context.Context, bucketName string) (replication.MetricsV2, error)

Get latest replication metrics using the V2 API on a bucket. This is a MinIO specific extension with enhanced metrics.

**Parameters**

| Param        | Type              | Description                                         |
|:-------------|:------------------|:----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |

**Return Values**

| Param     | Type                    | Description                                           |
|:----------|:------------------------|:------------------------------------------------------|
| `metrics` | *replication.MetricsV2* | Enhanced replication metrics returned from the server |
| `err`     | *error*                 | Standard Error                                        |

**Example**

```go
replMetrics, err := minioClient.GetBucketReplicationMetricsV2(context.Background(), "my-bucketname")
if err != nil {
	log.Fatalln(err)
}
fmt.Printf("Replication metrics: %+v\n", replMetrics)
```

<a name="CheckBucketReplication"></a>

### CheckBucketReplication(ctx context.Context, bucketName string) error

Validate whether replication is properly configured for a bucket. This is a MinIO specific extension.

**Parameters**

| Param        | Type              | Description                                         |
|:-------------|:------------------|:----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |

**Return Values**

| Param | Type    | Description                                                  |
|:------|:--------|:-------------------------------------------------------------|
| `err` | *error* | Returns nil if valid, or error describing validation failure |

**Example**

```go
err := minioClient.CheckBucketReplication(context.Background(), "my-bucketname")
if err != nil {
	log.Printf("Replication configuration is invalid: %v\n", err)
} else {
	log.Println("Replication configuration is valid")
}
```

<a name="ResetBucketReplication"></a>

### ResetBucketReplication(ctx context.Context, bucketName string, olderThan time.Duration) (string, error)

Initiate replication of previously replicated objects. Requires ExistingObjectReplication to be enabled in the replication configuration. This is a MinIO specific extension.

**Parameters**

| Param        | Type              | Description                                                         |
|:-------------|:------------------|:--------------------------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call                 |
| `bucketName` | *string*          | Name of the bucket                                                  |
| `olderThan`  | *time.Duration*   | Only replicate objects older than this duration (0 for all objects) |

**Return Values**

| Param     | Type     | Description                         |
|:----------|:---------|:------------------------------------|
| `resetID` | *string* | Reset ID for tracking the operation |
| `err`     | *error*  | Standard Error                      |

**Example**

```go
// Reset replication for all objects
resetID, err := minioClient.ResetBucketReplication(context.Background(), "my-bucketname", 0)
if err != nil {
	log.Fatalln(err)
}
fmt.Printf("Replication reset initiated with ID: %s\n", resetID)
```

<a name="ResetBucketReplicationOnTarget"></a>

### ResetBucketReplicationOnTarget(ctx context.Context, bucketName string, olderThan time.Duration, tgtArn string) (replication.ResyncTargetsInfo, error)

Initiate replication of previously replicated objects to a specific target. Requires ExistingObjectReplication to be enabled in the replication configuration. This is a MinIO specific extension.

**Parameters**

| Param        | Type              | Description                                                         |
|:-------------|:------------------|:--------------------------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call                 |
| `bucketName` | *string*          | Name of the bucket                                                  |
| `olderThan`  | *time.Duration*   | Only replicate objects older than this duration (0 for all objects) |
| `tgtArn`     | *string*          | ARN of the target to reset replication for                          |

**Return Values**

| Param        | Type                            | Description               |
|:-------------|:--------------------------------|:--------------------------|
| `resyncInfo` | *replication.ResyncTargetsInfo* | Resync target information |
| `err`        | *error*                         | Standard Error            |

**Example**

```go
resyncInfo, err := minioClient.ResetBucketReplicationOnTarget(context.Background(), "my-bucketname", 0, "arn:aws:s3:::target-bucket")
if err != nil {
	log.Fatalln(err)
}
fmt.Printf("Resync info: %+v\n", resyncInfo)
```

<a name="GetBucketReplicationResyncStatus"></a>

### GetBucketReplicationResyncStatus(ctx context.Context, bucketName, arn string) (replication.ResyncTargetsInfo, error)

Retrieve the status of a replication resync operation. This is a MinIO specific extension.

**Parameters**

| Param        | Type              | Description                                                  |
|:-------------|:------------------|:-------------------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call          |
| `bucketName` | *string*          | Name of the bucket                                           |
| `arn`        | *string*          | ARN of the replication target (empty string for all targets) |

**Return Values**

| Param        | Type                            | Description               |
|:-------------|:--------------------------------|:--------------------------|
| `resyncInfo` | *replication.ResyncTargetsInfo* | Resync status information |
| `err`        | *error*                         | Standard Error            |

**Example**

```go
// Get resync status for all targets
resyncInfo, err := minioClient.GetBucketReplicationResyncStatus(context.Background(), "my-bucketname", "")
if err != nil {
	log.Fatalln(err)
}
fmt.Printf("Resync status: %+v\n", resyncInfo)
```

1.	Client custom settings -------------------------

<a name="SetAppInfo"></a>

### SetAppInfo(appName, appVersion string)

Add custom application details to User-Agent.

**Parameters**

| Param        | Type     | Description                                             |
|--------------|----------|---------------------------------------------------------|
| `appName`    | *string* | Name of the application performing the API requests.    |
| `appVersion` | *string* | Version of the application performing the API requests. |

**Example**

```go
// Set Application name and version to be used in subsequent API requests.
minioClient.SetAppInfo("myCloudApp", "1.0.0")
```

<a name="TraceOn"></a>

### TraceOn(outputStream io.Writer)

Enables HTTP tracing. The trace is written to the io.Writer provided. If outputStream is nil, trace is written to os.Stdout.

**Parameters**

| Param          | Type        | Description                              |
|----------------|-------------|------------------------------------------|
| `outputStream` | *io.Writer* | HTTP trace is written into outputStream. |

<a name="TraceOff"></a>

### TraceOff()

Disables HTTP tracing.

<a name="SetS3TransferAccelerate"></a>

### SetS3TransferAccelerate(acceleratedEndpoint string)

Set AWS S3 transfer acceleration endpoint for all API requests hereafter. NOTE: This API applies only to AWS S3 and is a no operation for S3 compatible object storage services.

**Parameters**

| Param                 | Type     | Description                                   |
|-----------------------|----------|-----------------------------------------------|
| `acceleratedEndpoint` | *string* | Set to new S3 transfer acceleration endpoint. |

<a name="EndpointURL"></a>

### EndpointURL() *url.URL

Returns the URL of the S3-compatible endpoint that this client connects to. Returns a copy to prevent modification of internal state.

**Return Values**

| Param      | Type        | Description  |
|------------|-------------|--------------|
| `endpoint` | \**url.URL* | Endpoint URL |

**Example**

```go
endpointURL := minioClient.EndpointURL()
fmt.Printf("Connected to: %s\n", endpointURL.String())
```

<a name="GetCreds"></a>

### GetCreds() (credentials.Value, error)

Returns the current credentials being used by the client. Useful for debugging or when credentials need to be passed to other components.

**Return Values**

| Param   | Type                | Description                  |
|---------|---------------------|------------------------------|
| `creds` | *credentials.Value* | Current credentials          |
| `err`   | *error*             | Error retrieving credentials |

**Example**

```go
creds, err := minioClient.GetCreds()
if err != nil {
	log.Fatalln(err)
}
fmt.Printf("Access Key: %s\n", creds.AccessKeyID)
```

1.	Additional Operations ------------------------

<a name="SetBucketCors"></a>

### SetBucketCors(ctx context.Context, bucketName string, corsConfig *cors.Config) error

Set CORS (Cross-Origin Resource Sharing) configuration on a bucket.

**Parameters**

| Param        | Type              | Description                                         |
|--------------|-------------------|-----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |
| `corsConfig` | \**cors.Config*   | CORS configuration to be set                        |

**Example**

{% raw %}
```go
// Create CORS configuration
corsConfig := &cors.Config{
	CORSRules: []cors.Rule{{
		AllowedHeaders: []string{"*"},
		AllowedMethods: []string{"PUT", "GET", "DELETE"},
		AllowedOrigins: []string{"*"},
		MaxAgeSeconds:  3000,
	}},
}

err := minioClient.SetBucketCors(context.Background(), "mybucket", corsConfig)
if err != nil {
	log.Fatalln(err)
}
```
{% endraw %}

<a name="GetBucketCors"></a>

### GetBucketCors(ctx context.Context, bucketName string) (*cors.Config, error)

Get CORS configuration of a bucket.

**Parameters**

| Param        | Type              | Description                                         |
|--------------|-------------------|-----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |

**Return Values**

| Param        | Type            | Description        |
|--------------|-----------------|--------------------|
| `corsConfig` | \**cors.Config* | CORS configuration |
| `err`        | *error*         | Standard Error     |

**Example**

```go
corsConfig, err := minioClient.GetBucketCors(context.Background(), "mybucket")
if err != nil {
	log.Fatalln(err)
}
fmt.Printf("CORS configuration: %+v\n", corsConfig)
```

<a name="GetBucketQOS"></a>

### GetBucketQOS(ctx context.Context, bucket string) (*QOSConfig, error)

Get Quality of Service (QoS) configuration for a bucket. This is a MinIO-specific API.

**Parameters**

| Param    | Type              | Description                                         |
|----------|-------------------|-----------------------------------------------------|
| `ctx`    | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucket` | *string*          | Name of the bucket                                  |

**Return Values**

| Param       | Type          | Description                  |
|-------------|---------------|------------------------------|
| `qosConfig` | \**QOSConfig* | QoS configuration with rules |
| `err`       | *error*       | Standard Error               |

**Example**

```go
qosConfig, err := minioClient.GetBucketQOS(context.Background(), "mybucket")
if err != nil {
	log.Fatalln(err)
}
fmt.Printf("QoS configuration: %+v\n", qosConfig)
```

<a name="SetBucketQOS"></a>

### SetBucketQOS(ctx context.Context, bucket string, qosCfg *QOSConfig) error

Set Quality of Service (QoS) configuration for a bucket. This is a MinIO-specific API.

**Parameters**

| Param    | Type              | Description                                         |
|----------|-------------------|-----------------------------------------------------|
| `ctx`    | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucket` | *string*          | Name of the bucket                                  |
| `qosCfg` | \**QOSConfig*     | QoS configuration to apply                          |

**Example**

```go
qosConfig := &minio.QOSConfig{
	Version: "v1",
	Rules: []minio.QOSRule{{
		ID:           "rule1",
		Priority:     1,
		ObjectPrefix: "logs/",
		API:          "s3:GetObject",
		Rate:         1000,
		Burst:        100,
		Limit:        "rps",
	}},
}

err := minioClient.SetBucketQOS(context.Background(), "mybucket", qosConfig)
if err != nil {
	log.Fatalln(err)
}
```

<a name="GetBucketQOSMetrics"></a>

### GetBucketQOSMetrics(ctx context.Context, bucketName, nodeName string) ([]QOSNodeStats, error)

Get Quality of Service (QoS) metrics for a bucket. This is a MinIO-specific API.

**Parameters**

| Param        | Type              | Description                                         |
|--------------|-------------------|-----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |
| `nodeName`   | *string*          | Name of the node (empty string for all nodes)       |

**Return Values**

| Param     | Type             | Description          |
|-----------|------------------|----------------------|
| `metrics` | *[]QOSNodeStats* | QoS metrics per node |
| `err`     | *error*          | Standard Error       |

**Example**

```go
// Get QoS metrics for all nodes
metrics, err := minioClient.GetBucketQOSMetrics(context.Background(), "mybucket", "")
if err != nil {
	log.Fatalln(err)
}
for _, nodeStats := range metrics {
	fmt.Printf("Node: %s, Stats: %+v\n", nodeStats.NodeName, nodeStats.Stats)
}
```

<a name="GenerateInventoryConfigYAML"></a>

### GenerateInventoryConfigYAML(ctx context.Context, bucket, id string) (string, error)

Generate a YAML template for an inventory configuration. This is a MinIO-specific API.

**Parameters**

| Param    | Type              | Description                                         |
|----------|-------------------|-----------------------------------------------------|
| `ctx`    | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucket` | *string*          | Name of the bucket                                  |
| `id`     | *string*          | Unique identifier for the inventory configuration   |

**Return Values**

| Param          | Type     | Description          |
|----------------|----------|----------------------|
| `yamlTemplate` | *string* | YAML template string |
| `err`          | *error*  | Standard Error       |

**Example**

```go
yamlTemplate, err := minioClient.GenerateInventoryConfigYAML(context.Background(), "mybucket", "inventory1")
if err != nil {
	log.Fatalln(err)
}
fmt.Println(yamlTemplate)
```

<a name="PutBucketInventoryConfiguration"></a>

### PutBucketInventoryConfiguration(ctx context.Context, bucket, id, yamlDef string, opts ...InventoryPutConfigOption) error

Create or update an inventory configuration for a bucket. This is a MinIO-specific API.

**Parameters**

| Param     | Type                          | Description                                         |
|-----------|-------------------------------|-----------------------------------------------------|
| `ctx`     | *context.Context*             | Custom context for timeout/cancellation of the call |
| `bucket`  | *string*                      | Name of the bucket                                  |
| `id`      | *string*                      | Unique identifier for the inventory configuration   |
| `yamlDef` | *string*                      | YAML definition of the inventory configuration      |
| `opts`    | *...InventoryPutConfigOption* | Optional configuration options                      |

**Example**

```go
yamlDef := `---
version: v1
destination:
  bucket: destination-bucket
  prefix: inventory-reports
schedule:
  frequency: Daily
...`

err := minioClient.PutBucketInventoryConfiguration(context.Background(), "mybucket", "inventory1", yamlDef)
if err != nil {
	log.Fatalln(err)
}
```

<a name="GetBucketInventoryConfiguration"></a>

### GetBucketInventoryConfiguration(ctx context.Context, bucket, id string) (*InventoryConfiguration, error)

Retrieve the inventory configuration for a bucket. This is a MinIO-specific API.

**Parameters**

| Param    | Type              | Description                                         |
|----------|-------------------|-----------------------------------------------------|
| `ctx`    | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucket` | *string*          | Name of the bucket                                  |
| `id`     | *string*          | Unique identifier for the inventory configuration   |

**Return Values**

| Param    | Type                       | Description             |
|----------|----------------------------|-------------------------|
| `config` | \**InventoryConfiguration* | Inventory configuration |
| `err`    | *error*                    | Standard Error          |

**Example**

```go
config, err := minioClient.GetBucketInventoryConfiguration(context.Background(), "mybucket", "inventory1")
if err != nil {
	log.Fatalln(err)
}
fmt.Printf("Inventory: %+v\n", config)
```

<a name="DeleteBucketInventoryConfiguration"></a>

### DeleteBucketInventoryConfiguration(ctx context.Context, bucket, id string) error

Delete an inventory configuration from a bucket. This is a MinIO-specific API.

**Parameters**

| Param    | Type              | Description                                         |
|----------|-------------------|-----------------------------------------------------|
| `ctx`    | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucket` | *string*          | Name of the bucket                                  |
| `id`     | *string*          | Unique identifier for the inventory configuration   |

**Example**

```go
err := minioClient.DeleteBucketInventoryConfiguration(context.Background(), "mybucket", "inventory1")
if err != nil {
	log.Fatalln(err)
}
```

<a name="ListBucketInventoryConfigurations"></a>

### ListBucketInventoryConfigurations(ctx context.Context, bucket, continuationToken string) (*InventoryListResult, error)

List up to 100 inventory configurations for a bucket with pagination support. This is a MinIO-specific API.

**Parameters**

| Param               | Type              | Description                                           |
|---------------------|-------------------|-------------------------------------------------------|
| `ctx`               | *context.Context* | Custom context for timeout/cancellation of the call   |
| `bucket`            | *string*          | Name of the bucket                                    |
| `continuationToken` | *string*          | Token for pagination (empty string for first request) |

**Return Values**

| Param    | Type                    | Description                                            |
|----------|-------------------------|--------------------------------------------------------|
| `result` | \**InventoryListResult* | List result with configurations and continuation token |
| `err`    | *error*                 | Standard Error                                         |

**Example**

```go
result, err := minioClient.ListBucketInventoryConfigurations(context.Background(), "mybucket", "")
if err != nil {
	log.Fatalln(err)
}
for _, item := range result.Items {
	fmt.Printf("Inventory ID: %s\n", item.ID)
}
if result.NextContinuationToken != "" {
	// Fetch next page
	nextResult, _ := minioClient.ListBucketInventoryConfigurations(context.Background(), "mybucket", result.NextContinuationToken)
}
```

<a name="ListBucketInventoryConfigurationsIterator"></a>

### ListBucketInventoryConfigurationsIterator(ctx context.Context, bucket string) iter.Seq2[InventoryConfiguration, error]

Return an iterator that lists all inventory configurations for a bucket. This is a MinIO-specific API. Requires Go 1.23+.

**Parameters**

| Param    | Type              | Description                                         |
|----------|-------------------|-----------------------------------------------------|
| `ctx`    | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucket` | *string*          | Name of the bucket                                  |

**Return Values**

| Param      | Type                                       | Description                                 |
|------------|--------------------------------------------|---------------------------------------------|
| `iterator` | *iter.Seq2[InventoryConfiguration, error]* | Iterator yielding configurations and errors |

**Example**

```go
for config, err := range minioClient.ListBucketInventoryConfigurationsIterator(context.Background(), "mybucket") {
	if err != nil {
		log.Fatalln(err)
		break
	}
	fmt.Printf("Inventory ID: %s, Bucket: %s\n", config.ID, config.Bucket)
}
```

<a name="GetBucketInventoryJobStatus"></a>

### GetBucketInventoryJobStatus(ctx context.Context, bucket, id string) (*InventoryJobStatus, error)

Retrieve the status of an inventory job for a bucket. This is a MinIO-specific API.

**Parameters**

| Param    | Type              | Description                                         |
|----------|-------------------|-----------------------------------------------------|
| `ctx`    | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucket` | *string*          | Name of the bucket                                  |
| `id`     | *string*          | Unique identifier for the inventory job             |

**Return Values**

| Param    | Type                   | Description                                      |
|----------|------------------------|--------------------------------------------------|
| `status` | \**InventoryJobStatus* | Job status including state, progress, and errors |
| `err`    | *error*                | Standard Error                                   |

**InventoryJobStatus Fields**

The `InventoryJobStatus` struct contains comprehensive job information:

| Field                | Type            | Description                                                          |
|----------------------|-----------------|----------------------------------------------------------------------|
| `Bucket`             | string          | Source bucket name                                                   |
| `ID`                 | string          | Inventory configuration ID                                           |
| `User`               | string          | User who created the job                                             |
| `AccessKey`          | string          | Access key used for job execution                                    |
| `Schedule`           | string          | Job schedule (once, hourly, daily, weekly, monthly, yearly)          |
| `State`              | string          | Current job state (sleeping, pending, running, completed, etc.)      |
| `NextScheduledTime`  | time.Time       | When next execution will start (periodic jobs only)                  |
| `StartTime`          | time.Time       | When current/last execution started                                  |
| `EndTime`            | time.Time       | When execution completed                                             |
| `LastUpdate`         | time.Time       | Last time job metadata was updated                                   |
| `Scanned`            | string          | Last scanned object path                                             |
| `Matched`            | string          | Last matched object path                                             |
| `ScannedCount`       | uint64          | Total objects scanned                                                |
| `MatchedCount`       | uint64          | Total objects matched by filters                                     |
| `RecordsWritten`     | uint64          | Number of records written to output files                            |
| `OutputFilesCount`   | uint64          | Number of output files created                                       |
| `ExecutionTime`      | time.Duration   | Total execution time                                                 |
| `NumStarts`          | uint64          | Number of times job has started                                      |
| `NumErrors`          | uint64          | Total errors encountered                                             |
| `NumLockLosses`      | uint64          | Number of distributed lock losses                                    |
| `ManifestPath`       | string          | Full path to manifest.json file                                      |
| `RetryAttempts`      | uint64          | Number of retry attempts                                             |
| `LastFailTime`       | time.Time       | When last failure occurred (only present on errors)                  |
| `LastFailErrors`     | []string        | Up to 5 most recent error messages (only present on errors)          |

**Example**

```go
status, err := minioClient.GetBucketInventoryJobStatus(context.Background(), "mybucket", "inventory1")
if err != nil {
	log.Fatalln(err)
}

fmt.Printf("Job ID: %s, State: %s\n", status.ID, status.State)
fmt.Printf("Progress: %d/%d objects scanned/matched\n", status.ScannedCount, status.MatchedCount)
fmt.Printf("Output: %d records in %d files\n", status.RecordsWritten, status.OutputFilesCount)

if !status.StartTime.IsZero() {
	fmt.Printf("Started: %s\n", status.StartTime)
	if status.ExecutionTime > 0 {
		fmt.Printf("Execution time: %s\n", status.ExecutionTime)
	}
}

if status.ManifestPath != "" {
	fmt.Printf("Manifest: %s\n", status.ManifestPath)
}

if len(status.LastFailErrors) > 0 {
	fmt.Printf("Recent errors: %v\n", status.LastFailErrors)
}
```

<a name="PutObjectsSnowball"></a>

### PutObjectsSnowball(ctx context.Context, bucketName string, opts SnowballOptions, objs <-chan SnowballObject) error

Bulk upload multiple objects using snowball archive method for efficient batch operations.

**Parameters**

| Param        | Type                          | Description                                         |
|--------------|-------------------------------|-----------------------------------------------------|
| `ctx`        | *context.Context*             | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*                      | Name of the bucket                                  |
| `opts`       | *minio.SnowballOptions*       | Snowball upload options                             |
| `objs`       | *<-chan minio.SnowballObject* | Channel of objects to upload                        |

<a name="PromptObject"></a>

### PromptObject(ctx context.Context, bucketName, objectName, prompt string, opts PromptObjectOptions) (io.ReadCloser, error)

Perform language model inference with prompt and object context for AI/ML integration.

**Parameters**

| Param        | Type                        | Description                                         |
|--------------|-----------------------------|-----------------------------------------------------|
| `ctx`        | *context.Context*           | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*                    | Name of the bucket                                  |
| `objectName` | *string*                    | Name of the object                                  |
| `prompt`     | *string*                    | AI prompt text                                      |
| `opts`       | *minio.PromptObjectOptions* | Prompt operation options                            |

**Return Values**

| Param    | Type            | Description        |
|----------|-----------------|--------------------|
| `result` | *io.ReadCloser* | AI response stream |
| `err`    | *error*         | Standard Error     |

<a name="GetBucketLocation"></a>

### GetBucketLocation(ctx context.Context, bucketName string) (string, error)

Get the region/location constraint of a bucket.

**Parameters**

| Param        | Type              | Description                                         |
|--------------|-------------------|-----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |

**Return Values**

| Param      | Type     | Description            |
|------------|----------|------------------------|
| `location` | *string* | Bucket region/location |
| `err`      | *error*  | Standard Error         |

**Example**

```go
location, err := minioClient.GetBucketLocation(context.Background(), "mybucket")
if err != nil {
	log.Fatalln(err)
}
fmt.Printf("Bucket location: %s\n", location)
```

<a name="GetBucketReplicationMetrics"></a>

### GetBucketReplicationMetrics(ctx context.Context, bucketName string) (replication.Metrics, error)

Get replication metrics for a bucket. This is a MinIO specific extension.

**Parameters**

| Param        | Type              | Description                                         |
|--------------|-------------------|-----------------------------------------------------|
| `ctx`        | *context.Context* | Custom context for timeout/cancellation of the call |
| `bucketName` | *string*          | Name of the bucket                                  |

**Return Values**

| Param     | Type                  | Description         |
|-----------|-----------------------|---------------------|
| `metrics` | *replication.Metrics* | Replication metrics |
| `err`     | *error*               | Standard Error      |

**Example**

```go
metrics, err := minioClient.GetBucketReplicationMetrics(context.Background(), "mybucket")
if err != nil {
	log.Fatalln(err)
}
fmt.Printf("Replication metrics: %+v\n", metrics)
```

<a name="TraceErrorsOnlyOn"></a>

### TraceErrorsOnlyOn(outputStream io.Writer)

Enable HTTP tracing for errors only. The trace is written to the io.Writer provided.

**Parameters**

| Param          | Type        | Description                                    |
|----------------|-------------|------------------------------------------------|
| `outputStream` | *io.Writer* | HTTP error trace is written into outputStream. |

<a name="TraceErrorsOnlyOff"></a>

### TraceErrorsOnlyOff()

Disable HTTP error-only tracing.

<a name="SetS3EnableDualstack"></a>

### SetS3EnableDualstack(enabled bool)

Enable or disable S3 dual-stack endpoints which support both IPv4 and IPv6.

**Parameters**

| Param     | Type   | Description                                 |
|-----------|--------|---------------------------------------------|
| `enabled` | *bool* | Enable dual-stack if true, disable if false |

<a name="IsOnline"></a>

### IsOnline() bool

Check if the MinIO client is online and can reach the server.

**Return Value**

| Param    | Type   | Description              |
|----------|--------|--------------------------|
| `online` | *bool* | true if client is online |

<a name="IsOffline"></a>

### IsOffline() bool

Check if the MinIO client is offline and cannot reach the server.

**Return Value**

| Param     | Type   | Description               |
|-----------|--------|---------------------------|
| `offline` | *bool* | true if client is offline |

<a name="HealthCheck"></a>

### HealthCheck(hcDuration time.Duration) (context.CancelFunc, error)

Start continuous health check monitoring of the MinIO server.

**Parameters**

| Param        | Type            | Description           |
|--------------|-----------------|-----------------------|
| `hcDuration` | *time.Duration* | Health check interval |

**Return Values**

| Param    | Type                 | Description                     |
|----------|----------------------|---------------------------------|
| `cancel` | *context.CancelFunc* | Function to cancel health check |
| `err`    | *error*              | Standard Error                  |
