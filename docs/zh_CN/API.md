# MinIO Go Client API文档 [![Slack](https://slack.min.io/slack?type=svg)](https://slack.min.io)

## 初使化MinIO Client对象。

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
	// 使用ssl
	useSSL := true

	// 初使化minio client对象。
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
	// 初始化minio client对象.
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

| 操作存储桶                                            | 操作对象                                            | Presigned操作                                 | 存储桶策略/通知                                              | 客户端自定义设置                                      |
| :---------------------------------------------------- | :-------------------------------------------------- | :-------------------------------------------- | :----------------------------------------------------------- | :---------------------------------------------------- |
| [`MakeBucket`](#MakeBucket)                           | [`GetObject`](#GetObject)                           | [`PresignedGetObject`](#PresignedGetObject)   | [`SetBucketPolicy`](#SetBucketPolicy)                        | [`SetAppInfo`](#SetAppInfo)                           |
|                                                       | [`PutObject`](#PutObject)                           | [`PresignedPutObject`](#PresignedPutObject)   | [`GetBucketPolicy`](#GetBucketPolicy)                        |                                                       |
| [`ListBuckets`](#ListBuckets)                         | [`CopyObject`](#CopyObject)                         | [`PresignedHeadObject`](#PresignedHeadObject) | [`SetBucketNotification`](#SetBucketNotification)            | [`TraceOn`](#TraceOn)                                 |
| [`BucketExists`](#BucketExists)                       | [`StatObject`](#StatObject)                         | [`PresignedPostPolicy`](#PresignedPostPolicy) | [`GetBucketNotification`](#GetBucketNotification)            | [`TraceOff`](#TraceOff)                               |
| [`RemoveBucket`](#RemoveBucket)                       | [`RemoveObject`](#RemoveObject)                     |                                               | [`RemoveAllBucketNotification`](#RemoveAllBucketNotification) | [`SetS3TransferAccelerate`](#SetS3TransferAccelerate) |
| [`ListObjects`](#ListObjects)                         | [`RemoveObjects`](#RemoveObjects)                   |                                               | [`ListenBucketNotification`](#ListenBucketNotification)      |                                                       |
|                                                       | [`RemoveIncompleteUpload`](#RemoveIncompleteUpload) |                                               | [`SetBucketLifecycle`](#SetBucketLifecycle)                  |                                                       |
| [`ListIncompleteUploads`](#ListIncompleteUploads)     | [`FPutObject`](#FPutObject)                         |                                               | [`GetBucketLifecycle`](#GetBucketLifecycle)                  |                                                       |
| [`SetBucketTagging`](#SetBucketTagging)               | [`FGetObject`](#FGetObject)                         |                                               | [`SetObjectLockConfig`](#SetObjectLockConfig)                |                                                       |
| [`GetBucketTagging`](#GetBucketTagging)               | [`ComposeObject`](#ComposeObject)                   |                                               | [`GetObjectLockConfig`](#GetObjectLockConfig)                |                                                       |
| [`RemoveBucketTagging`](#RemoveBucketTagging)         |                                                     |                                               | [`EnableVersioning`](#EnableVersioning)                      |                                                       |
| [`SetBucketReplication`](#SetBucketReplication)       |                                                     |                                               | [`DisableVersioning`](#DisableVersioning)                    |                                                       |
| [`GetBucketReplication`](#GetBucketReplication)       | [`PutObjectRetention`](#PutObjectRetention)         |                                               | [`GetBucketEncryption`](#GetBucketEncryption)                |                                                       |
| [`RemoveBucketReplication`](#RemoveBucketReplication) | [`GetObjectRetention`](#GetObjectRetention)         |                                               | [`RemoveBucketEncryption`](#RemoveBucketEncryption)          |                                                       |
|                                                       | [`PutObjectLegalHold`](#PutObjectLegalHold)         |                                               |                                                              |                                                       |
|                                                       | [`GetObjectLegalHold`](#GetObjectLegalHold)         |                                               |                                                              |                                                       |
|                                                       | [`SelectObjectContent`](#SelectObjectContent)       |                                               |                                                              |                                                       |
|                                                       | [`PutObjectTagging`](#PutObjectTagging)             |                                               |                                                              |                                                       |
|                                                       | [`GetObjectTagging`](#GetObjectTagging)             |                                               |                                                              |                                                       |
|                                                       | [`RemoveObjectTagging`](#RemoveObjectTagging)       |                                               |                                                              |                                                       |
|                                                       | [`RestoreObject`](#RestoreObject)                   |                                               |                                                              |                                                       |
## 1. 构造函数
<a name="MinIO"></a>

### New(endpoint string, opts \*Options) (\*Client, error)
初使化一个新的client对象。

__参数__

|参数   | 类型   |描述   |
|:---|:---| :---|
|`endpoint`   | _string_  |S3兼容对象存储服务endpoint   |
|`opts`  |_minio.Options_   |构造新客户端的配置选项 |

__minio.Options__

| 字段                | 类型                       | 描述                            |
| :------------------ | :------------------------- | :------------------------------ |
| `opts.Creds`        | **credentials.Credentials* | S3兼容的对象存储访问凭据        |
| `opts.Secure`       | *bool*                     | true代表使用HTTPS               |
| `opts.Transport`    | *http.RoundTripper*        | 用于执行 HTTP 事务的自定义传输  |
| `opts.Region`       | *string*                   | S3兼容的对象存储region          |
| `opts.BucketLookup` | *BucketLookupType*         | Bucket 查找类型可以是下列值之一 |
|                     |                            | *minio.BucketLookupDNS*         |
|                     |                            | *minio.BucketLookupPath*        |
|                     |                            | *minio.BucketLookupAuto*        |

## 2. 操作存储桶

<a name="MakeBucket"></a>
### MakeBucket(ctx context.Context, bucketName string, opts MakeBucketOptions)
创建一个存储桶。

__参数__

| 参数         | 类型                      | 描述                                                         |
| ------------ | ------------------------- | ------------------------------------------------------------ |
| `ctx`        | *context.Context*         | 用于超时/取消调用的自定义上下文                              |
| `bucketName` | *string*                  | 存储桶的名字                                                 |
| `opts`       | *minio.MakeBucketOptions* | 桶选项，如“区域”中的桶将被创建。默认值是 us-east-1。下面列出了其他有效值。注意: 当与 minio 服务器一起使用时，使用其配置文件中指定的区域(默认为 us-east-1) |
|              |                           | us-east-1                                                    |
|              |                           | us-east-2                                                    |
|              |                           | us-west-1                                                    |
|              |                           | us-west-2                                                    |
|              |                           | ca-central-1                                                 |
|              |                           | eu-west-1                                                    |
|              |                           | eu-west-2                                                    |
|              |                           | eu-west-3                                                    |
|              |                           | eu-central-1                                                 |
|              |                           | eu-north-1                                                   |
|              |                           | ap-east-1                                                    |
|              |                           | ap-south-1                                                   |
|              |                           | ap-southeast-1                                               |
|              |                           | ap-southeast-2                                               |
|              |                           | ap-northeast-1                                               |
|              |                           | ap-northeast-2                                               |
|              |                           | ap-northeast-3                                               |
|              |                           | me-south-1                                                   |
|              |                           | sa-east-1                                                    |
|              |                           | us-gov-west-1                                                |
|              |                           | us-gov-east-1                                                |
|              |                           | cn-north-1                                                   |
|              |                           | cn-northwest-1                                               |


__示例__


```go
// 在“ us-east-1”区域创建一个启用了对象锁定的 存储桶。
err = minioClient.MakeBucket(context.Background(), "mybucket", minio.MakeBucketOptions{Region: "us-east-1", ObjectLocking: true})
if err != nil {
    fmt.Println(err)
    return
}
fmt.Println("Successfully created mybucket.")
```

<a name="ListBuckets"></a>
### ListBuckets(ctx context.Context) ([]BucketInfo, error)
列出所有的存储桶。

| 参数  | 类型   | 描述  |
|---|---|---|
|`ctx` | _context.Context_ | 用于超时/取消调用的自定义上下文 |
|`bucketList`  | _[]minio.BucketInfo_  | 所有存储桶的list。 |


__minio.BucketInfo__

| 参数  | 类型   | 描述  |
|---|---|---|
|`bucket.Name`  | _string_  | 存储桶名称 |
|`bucket.CreationDate`  | _time.Time_  | 存储桶的创建时间 |


__示例__


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
检查存储桶是否存在。

__参数__


|参数   |类型   |描述   |
|:---|:---| :---|
|`ctx` |_context.Context_ |用于超时/取消调用的自定义上下文 |
|`bucketName`  | _string_  |存储桶名称 |


__返回值__

|参数   |类型   |描述   |
|:---|:---| :---|
|`found`  | _bool_ | 存储桶是否存在  |
|`err` | _error_  | 标准Error  |


__示例__


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
删除一个存储桶，存储桶必须为空才能被成功删除。

__参数__


|参数   |类型   |描述   |
|:---|:---| :---|
|`ctx` |_context.Context_ |用于超时/取消调用的自定义上下文 |
|`bucketName`  | _string_  |存储桶名称   |

__示例__


```go
err = minioClient.RemoveBucket(context.Background(), "mybucket")
if err != nil {
    fmt.Println(err)
    return
}
```

<a name="ListObjects"></a>
### ListObjects(ctx context.Context, bucketName string, opts ListObjectsOptions) <-chan ObjectInfo
列举存储桶里的对象。

__参数__

| Param        | Type                       | Description                     |
| :----------- | :------------------------- | :------------------------------ |
| `ctx`        | *context.Context_          | 用于超时/取消调用的自定义上下文 |
| `bucketName` | *string*                   | 存储桶名称                      |
| `opts`       | *minio.ListObjectsOptions* | 每个列表对象的配置选项          |


__返回值__

|参数   |类型   |描述   |
|:---|:---| :---|
|`objectInfo`  | _chan minio.ObjectInfo_ |存储桶中所有对象的read channel，对象的格式如下： |

__minio.ObjectInfo__

|属性   |类型   |描述   |
|:---|:---| :---|
|`objectInfo.Key`  | _string_ |对象的名称 |
|`objectInfo.Size`  | _int64_ |对象的大小 |
|`objectInfo.ETag`  | _string_ |对象的MD5校验码 |
|`objectInfo.LastModified`  | _time.Time_ |对象的最后修改时间 |


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


<a name="ListObjectsV2"></a>
### ListIncompleteUploads(ctx context.Context, bucketName, prefix string, recursive bool) <- chan ObjectMultipartInfo
列举存储桶中的对象。

__参数__


|参数   |类型   |描述   |
|:---|:---| :---|
|`bucketName`  | _string_  |存储桶名称 |
| `prefix` |_string_   | 要列举的对象前缀 |
| `recursive`  | _bool_  |`true`代表递归查找，`false`代表类似文件夹查找，以'/'分隔，不查子文件夹。  |
|`ctx`  | _context.Context_ | 用于超时/取消调用的自定义上下文 |


__返回值__

|参数   |类型   |描述   |
|:---|:---| :---|
|`multiPartInfo`  | _chan minio.ObjectMultipartInfo_ |发出下列格式的多部分对象: |

__minio.ObjectMultipartInfo__

| 字段                        | 类型     | 描述                     |
| :-------------------------- | :------- | :----------------------- |
| `multiPartObjInfo.Key`      | *string* | 未完全上传对象的名称     |
| `multiPartObjInfo.UploadID` | *string* | 未完全上传对象的 ID      |
| `multiPartObjInfo.Size`     | *int64*  | 未完全上传对象的内存大小 |


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

### SetBucketTagging(ctx context.Context, bucketName string, tags *tags.Tags) error

为存储桶添加一个tag


__参数__

| 参数         | 类型              | 描述                            |
| :----------- | :---------------- | :------------------------------ |
| `ctx`        | _context.Context_ | 用于超时/取消调用的自定义上下文 |
| `bucketName` | _string_          | 存储桶的名称                    |
| `tags`       | _*tags.Tags_      | 存储桶的标签                    |

__例子__

```go
// 从字典中新建标签
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

获取存储桶的标签


__参数__

| 参数         | 类型              | 描述                            |
| :----------- | :---------------- | :------------------------------ |
| `ctx`        | _context.Context_ | 用于超时/取消调用的自定义上下文 |
| `bucketName` | _string_          | 存储桶的名称                    |

__返回值__

| 参数   | 类型         | 描述 |
| :----- | :----------- | :--- |
| `tags` | _*tags.Tags_ | 标签 |

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

移除存储桶的标签


__参数__

| 参数         | 类型              | 描述                            |
| :----------- | :---------------- | :------------------------------ |
| `ctx`        | _context.Context_ | 用于超时/取消调用的自定义上下文 |
| `bucketName` | _string_          | 存储桶的标签                    |

__例子__

```go
err := minioClient.RemoveBucketTagging(context.Background(), "my-bucketname")
if err != nil {
	log.Fatalln(err)
}
```

## 3. 操作对象

<a name="GetObject"></a>
### GetObject(ctx context.Context, bucketName, objectName string, opts GetObjectOptions) (*Object, error)
返回对象数据的流，error是读流时经常抛的那些错。


__参数__


|参数   |类型   |描述   |
|:---|:---| :---|
|`ctx` |_context.Context_ |用于超时/取消调用的自定义上下文 |
|`bucketName`  | _string_  |存储桶名称  |
|`objectName` | _string_  |对象的名称  |
|`opts` | _minio.GetObjectOptions_ | GET请求的一些额外参数，像encryption，If-Match |


__minio.GetObjectOptions__

| 字段                        | 类型                       | 描述                                                         |
| :-------------------------- | :------------------------- | :----------------------------------------------------------- |
| `opts.ServerSideEncryption` | *encrypt.ServerSide*       | “加密”包提供的指定服务器端加密的接口。(详情见 https://godoc.org/github.com/minio/minio-go/v7) |
| `opts.Internal`             | *minio.AdvancedGetOptions* | 此选项用于 MinIO 服务器的内部使用。除非应用程序知道预期的用途，否则不应设置此选项。 |

__返回值__


|参数   |类型   |描述   |
|:---|:---| :---|
|`object`  | _*minio.Object_ |_minio.Object_代表了一个object reader。它实现了io.Reader, io.Seeker, io.ReaderAt and io.Closer接口。 |


__示例__


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
### FGetObject(bucketName, objectName, filePath string, opts GetObjectOptions) error
下载并将文件保存到本地文件系统。

__参数__


|参数   |类型   |描述   |
|:---|:---| :---|
|`ctx` |_context.Context_ |用于超时/取消调用的自定义上下文 |
|`bucketName`  | _string_  |存储桶名称 |
|`objectName` | _string_  |对象的名称  |
|`filePath` | _string_  |下载后保存的路径 |
|`opts` | _minio.GetObjectOptions_ | GET请求的一些额外参数，像encryption，If-Match |


__示例__


```go
err = minioClient.FGetObject(context.Background(), "mybucket", "myobject", "/tmp/myobject", minio.GetObjectOptions{})
if err != nil {
    fmt.Println(err)
    return
}
```
### PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64,opts PutObjectOptions) (info UploadInfo, err error)
当对象小于128MiB时，直接在一次PUT请求里进行上传。当大于128MiB时，根据文件的实际大小，PutObject会自动地将对象进行拆分成128MiB一块或更大一些进行上传。对象的最大大小是5TB。

__参数__


|参数   |类型   |描述   |
|:---|:---| :---|
|`ctx` |_context.Context_ |用于超时/取消调用的自定义上下文 |
|`bucketName`  | _string_  |存储桶名称  |
|`objectName` | _string_  |对象的名称   |
|`reader` | _io.Reader_  |任意实现了io.Reader的GO类型 |
|`objectSize`| _int64_ |上传的对象的大小，-1代表未知。 |
|`opts` | _minio.PutObjectOptions_  |  允许用户设置可选的自定义元数据，内容标题，加密密钥和用于分段上传操作的线程数量。 |

__minio.PutObjectOptions__

| 字段                           | 类型                       | 描述                                                         |
| :----------------------------- | :------------------------- | :----------------------------------------------------------- |
| `opts.UserMetadata`            | *map[string]string*        | 用户元数据的Map                                              |
| `opts.UserTags`                | *map[string]string*        | 用户对象标签的Map                                            |
| `opts.Progress`                | *io.Reader*                | 获取上传进度的Reader                                         |
| `opts.ContentType`             | *string*                   | 对象的Content type， 例如"application/text"                  |
| `opts.ContentEncoding`         | *string*                   | 对象的Content encoding，例如"gzip"                           |
| `opts.ContentDisposition`      | *string*                   | 对象的Content disposition, "inline"                          |
| `opts.ContentLanguage`         | *string*                   | 对象的Content language ，例如 "French"                       |
| `opts.CacheControl`            | *string*                   | 指定针对请求和响应的缓存机制，例如"max-age=600"              |
| `opts.Mode`                    | **minio.RetentionMode*     | 将要被设置的模式, 例如 "COMPLIANCE"                          |
| `opts.RetainUntilDate`         | **time.Time*               | 应用保留的有效期限                                           |
| `opts.ServerSideEncryption`    | *encrypt.ServerSide*       | “加密”包提供的指定服务器端加密的接口。 (详情见 https://godoc.org/github.com/minio/minio-go/v7) |
| `opts.StorageClass`            | *string*                   | 指定对象的存储类。 MinIO 服务器支持的值为“ REDUCED _ REDUNDANCY”和“ STANDARD” |
| `opts.WebsiteRedirectLocation` | *string*                   | 为该对象指定一个重定向，指向同一存储桶中的另一个对象或外部 URL。 |
| `opts.SendContentMd5`          | *bool*                     | 指定是否要使用 PutObject 操作发送“ content-md5”头。请注意，由于内存中的“ md5sum”计算，设置此标志将导致更高的内存使用率。 |
| `opts.PartSize`                | *uint64*                   | 指定用于上载对象的自定义部件大小                             |
| `opts.Internal`                | *minio.AdvancedPutOptions* | 这个选项是供 MinIO 服务器内部使用的，除非应用程序知道预期的用途，否则不应该设置该选项。 |


__示例__


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

API方法在minio-go SDK版本v3.0.3中提供的PutObjectWithSize，PutObjectWithMetadata，PutObjectStreaming和PutObjectWithProgress被替换为接受指向PutObjectOptions struct的指针的新的PutObject调用变体。

<a name="CopyObject"></a>

### CopyObject(ctx context.Context, dst CopyDestOptions, src CopySrcOptions) (UploadInfo, error)
通过在服务端对已存在的对象进行拷贝，实现新建或者替换对象。它支持有条件的拷贝，拷贝对象的一部分，以及在服务端的加解密。请查看`SourceInfo`和`DestinationInfo`两个类型来了解更多细节。 

拷贝多个源文件到一个目标对象，请查看`ComposeObject` API。

__参数__


|参数   |类型   |描述   |
|:---|:---| :---|
|`ctx` |_context.Context_ |用于超时/取消调用的自定义上下文 |
|`dst`  | _minio.CopyDestOptions_ |目标对象 |
|`src` | _minio.CopySrcOptions_ |源对象 |


__示例__


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
通过使用服务端拷贝实现钭多个源对象合并创建成一个新的对象。

__参数__


|参数   |类型   |描述   |
|:---|:---|:---|
|`ctx` |_context.Context_ |用于超时/取消调用的自定义上下文 |
|`dst`  | _minio.DestinationInfo_  |要被创建的目标对象 |
|`srcs` | _[]minio.SourceInfo_  |要合并的多个源对象 |


__示例__


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
将内容从文件上传到 objectName。



FPutObject 在单个 PUT 操作中上传小于128MiB 的对象。对于大于128MiB 大小的对象，FPutObject 根据实际文件大小以128MiB 或更大的数据块无缝地上传对象。对象的最大上载大小为5TB。

__参数__

| 参数         | 类型                     | 描述                                                         |
| :----------- | :----------------------- | :----------------------------------------------------------- |
| `ctx`        | *context.Context*        | 用于超时/取消调用的自定义上下文                              |
| `bucketName` | *string*                 | 存储桶名称                                                   |
| `objectName` | *string*                 | 对象名称                                                     |
| `filePath`   | *string*                 | 要上载的文件路径                                             |
| `opts`       | *minio.PutObjectOptions* | 指向 struct 的指针，允许用户设置可选的自定义元数据、内容类型、内容编码、内容配置、内容语言和缓存控制头，传递加密对象的加密模块，并可选地配置多部分放置操作的线程数。 |

__示例__

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

### StatObject(ctx context.Context, bucketName, objectName string, opts StatObjectOptions) (ObjectInfo, error)
获取对象的元数据。

__参数__


|参数   |类型   |描述   |
|:---|:---| :---|
|`ctx` |_context.Context_ |用于超时/取消调用的自定义上下文 |
|`bucketName`  | _string_  |存储桶名称  |
|`objectName` | _string_  |对象的名称   |
|`opts` | _minio.StatObjectOptions_ | GET info/stat请求的一些额外参数，像encryption，If-Match |


__返回值__

|参数   |类型   |描述   |
|:---|:---| :---|
|`objInfo`  | _minio.ObjectInfo_  |对象stat信息 |


__minio.ObjectInfo__

|属性   |类型   |描述   |
|:---|:---| :---|
|`objInfo.LastModified`  | _time.Time_  |对象的最后修改时间 |
|`objInfo.ETag` | _string_ |对象的MD5校验码|
|`objInfo.ContentType` | _string_ |对象的Content type|
|`objInfo.Size` | _int64_ |对象的大小|


__示例__


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

删除一个对象。

__参数__

| 参数         | 类型                        | 描述                            |
| :----------- | :-------------------------- | :------------------------------ |
| `ctx`        | *context.Context*           | 用于超时/取消调用的自定义上下文 |
| `bucketName` | *string*                    | 存储桶名称                      |
| `objectName` | *string*                    | 对象的名称                      |
| `opts`       | *minio.RemoveObjectOptions* | 允许用户设置的配置选项          |

__minio.RemoveObjectOptions__

| 字段                    | 类型                          | 描述                                                         |
| :---------------------- | :---------------------------- | :----------------------------------------------------------- |
| `opts.GovernanceBypass` | _bool_                        | 设置旁路治理标头以删除以 GOVERNANCE 模式锁定的对象           |
| `opts.VersionID`        | _string_                      | 要删除的对象的版本 ID                                        |
| `opts.Internal`         | _minio.AdvancedRemoveOptions_ | 这个选项是供 MinIO 服务器内部使用的，除非应用程序知道预期的用途，否则不应该设置该选项。 |


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

对对象应用对象保留锁。

__参数	__


| 参数         | 类型                              | 描述                                           |
| :----------- | :-------------------------------- | :--------------------------------------------- |
| `ctx`        | _context.Context_                 | 用于超时/取消调用的自定义上下文                |
| `bucketName` | _string_                          | 存储桶名称                                     |
| `objectName` | _string_                          | 对象名称                                       |
| `opts`       | _minio.PutObjectRetentionOptions_ | 允许用户设置保留模式、到期日期和版本 ID 等选项 |

<a name="RemoveObjects"></a>

### RemoveObjects(ctx context.Context, bucketName string, objectsCh <-chan ObjectInfo, opts RemoveObjectsOptions) <-chan RemoveObjectError

从一个input channel里删除一个对象集合。一次发送到服务端的删除请求最多可删除1000个对象。通过error channel返回的错误信息。

__参数__

|参数   |类型   |描述   |
|:---|:---| :---|
|`ctx` |_context.Contex_ |用于超时/取消调用的自定义上下文 |
|`bucketName`  | _string_  |存储桶名称  |
|`objectsCh` | _chan string_  | 要删除的对象的channel   |
|`opts` | _minio.RemoveObjectsOptions_ | 运行用户设置的配置选项 |

__minio.RemoveObjectsOptions__

| 字段                    | 类型   | 描述                                               |
| :---------------------- | :----- | :------------------------------------------------- |
| `opts.GovernanceBypass` | _bool_ | 设置旁路治理标头以删除以 GOVERNANCE 模式锁定的对象 |

__返回值__

|参数   |类型   |描述   |
|:---|:---| :---|
|`errorCh` | _<-chan minio.RemoveObjectError_  | 删除时观察到的错误的Receive-only channel。 |


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

返回给定对象的保持集。

__参数__


| 参数         | 类型              | 描述                            |
| :----------- | :---------------- | :------------------------------ |
| `ctx`        | _context.Context_ | 用于超时/取消调用的自定义上下文 |
| `bucketName` | _string_          | 存储桶名称                      |
| `objectName` | _string_          | 对象名称                        |
| `versionID`  | _string_          | 对象的版本ID                    |

```go
err = minioClient.PutObjectRetention(context.Background(), "mybucket", "myobject", "")
if err != nil {
    fmt.Println(err)
    return
}
```

<a name="PutObjectLegalHold"></a>

### PutObjectLegalHold(ctx context.Context, bucketName, objectName string, opts minio.PutObjectLegalHoldOptions) error

应用合法-保持一个对象。

__参数	__


| 参数         | 参数                              | 描述                                 |
| :----------- | :-------------------------------- | :----------------------------------- |
| `ctx`        | _context.Context_                 | 用于超时/取消调用的自定义上下文      |
| `bucketName` | _string_                          | 存储桶名称                           |
| `objectName` | _string_                          | 对象名称                             |
| `opts`       | _minio.PutObjectLegalHoldOptions_ | 像版本 ID 一样允许用户设置的配置选项 |

_minio.PutObjectLegalHoldOptions_

| 字段             | 类型                     | 描述                          |
| :--------------- | :----------------------- | :---------------------------- |
| `opts.Status`    | _*minio.LegalHoldStatus_ | Legal-Hold 状态设置           |
| `opts.VersionID` | _string_                 | 要对其应用保留的对象的版本 ID |

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

返回给定对象的合法持有状态

__参数__

| 参数         | 类型                              | 描述                                 |
| :----------- | :-------------------------------- | :----------------------------------- |
| `ctx`        | _context.Context_                 | 用于超时/取消调用的自定义上下文      |
| `bucketName` | _string_                          | 存储桶名称                           |
| `objectName` | _string_                          | 对象名称                             |
| `opts`       | _minio.GetObjectLegalHoldOptions_ | 像版本 ID 一样允许用户设置的配置选项 |

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

__参数	__

| 参数         | 类型                  | 描述                            |
| :----------- | :-------------------- | :------------------------------ |
| `ctx`        | _context.Context_     | 用于超时/取消调用的自定义上下文 |
| `bucketName` | _string_              | 存储桶名称                      |
| `objectName` | _string_              | 对象名称                        |
| `options`    | _SelectObjectOptions_ | 请求配置选项                    |

__返回值__

| 参数            | 类型            | 描述                                                         |
| :-------------- | :-------------- | :----------------------------------------------------------- |
| `SelectResults` | _SelectResults_ | 是一个 io.ReadCclose 对象，可以直接传递给 csv.NewReader 以处理输出。 |

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

将新对象标记设置为给定对象，替换/覆盖任何现有标记。

__参数	__


| 参数         | 类型              | 描述                            |
| :----------- | :---------------- | :------------------------------ |
| `ctx`        | _context.Context_ | 用于超时/取消调用的自定义上下文 |
| `bucketName` | _string_          | 存储桶名称                      |
| `objectName` | _string_          | 对象名称                        |
| `objectTags` | _*tags.Tags_      | 标签的Map                       |

__例子__


```go
err = minioClient.PutObjectTagging(context.Background(), bucketName, objectName, objectTags)
if err != nil {
    fmt.Println(err)
    return
}
```

<a name="GetObjectTagging"></a>

### GetObjectTagging(ctx context.Context, bucketName, objectName string) (*tags.Tags, error)

从给定对象获取对象标记

__参数__


| 参数         | 类型              | 描述                            |
| :----------- | :---------------- | :------------------------------ |
| `ctx`        | _context.Context_ | 用于超时/取消调用的自定义上下文 |
| `bucketName` | _string_          | 存储桶名称                      |
| `objectName` | _string_          | 对象名称                        |

__例子__


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

从给定对象中删除对象标记

__参数__


| 参数         | 类型              | 描述                            |
| :----------- | :---------------- | :------------------------------ |
| `ctx`        | _context.Context_ | 用于超时/取消调用的自定义上下文 |
| `bucketName` | _string_          | 存储桶名称                      |
| `objectName` | _string_          | 对象名称                        |

__例子__


```go
err = minioClient.RemoveObjectTagging(context.Background(), bucketName, objectName)
if err != nil {
    fmt.Println(err)
    return
}
```

<a name="RestoreObject"></a>

### RestoreObject(ctx context.Context, bucketName, objectName, versionID string, opts minio.RestoreRequest) error

对归档对象还原或执行 SQL 操作

__参数__

| 参数         | 类型                    | 描述                            |
| :----------- | :---------------------- | :------------------------------ |
| `ctx`        | _context.Context_       | 用于超时/取消调用的自定义上下文 |
| `bucketName` | _string_                | 存储桶名称                      |
| `objectName` | _string_                | 对象名称                        |
| `versionID`  | _string_                | 对象的版本 ID                   |
| `opts`       | _minio.RestoreRequest)_ | Restore请求配置选项             |

__例子	__

```go
opts := minio.RestoreRequest{}
opts.SetDays(1)
opts.SetGlacierJobParameters(minio.GlacierJobParameters{Tier: minio.TierStandard})

err = s3Client.RestoreObject(context.Background(), "your-bucket", "your-object", "", opts)
if err != nil {
    log.Fatalln(err)
}
```

<a name="RemoveIncompleteUpload"></a>

### RemoveIncompleteUpload(ctx context.Context, bucketName, objectName string) error
删除一个未完整上传的对象。

__参数__


|参数   |类型   |描述   |
|:---|:---| :---|
|`ctx` |_context.Context_ |用于超时/取消调用的自定义上下文 |
|`bucketName`  | _string_  |存储桶名称   |
|`objectName` | _string_  |对象的名称   |

__示例__


```go
err = minioClient.RemoveIncompleteUpload(context.Background(), "mybucket", "myobject")
if err != nil {
    fmt.Println(err)
    return
}
```


## 4. Presigned操作

<a name="PresignedGetObject"></a>

### PresignedGetObject(ctx context.Context, bucketName, objectName string, expiry time.Duration, reqParams url.Values) (*url.URL, error)
生成一个用于HTTP GET操作的presigned URL。浏览器/移动客户端可以在即使存储桶为私有的情况下也可以通过这个URL进行下载。这个presigned URL可以有一个过期时间，默认是7天。

__参数__


|参数   |类型   |描述   |
|:---|:---| :---|
|`ctx` |_context.Context_ |用于超时/取消调用的自定义上下文 |
|`bucketName`  | _string_  |存储桶名称   |
|`objectName` | _string_  |对象的名称   |
|`expiry` | _time.Duration_  |presigned URL的过期时间，单位是秒   |
|`reqParams` | _url.Values_  |额外的响应头，支持_response-expires_， _response-content-type_， _response-cache-control_， _response-content-disposition_。  |


__示例__


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
生成一个用于HTTP GET操作的presigned URL。浏览器/移动客户端可以在即使存储桶为私有的情况下也可以通过这个URL进行下载。这个presigned URL可以有一个过期时间，默认是7天。

注意：你可以通过只指定对象名称上传到S3。

__参数__


|参数   |类型   |描述   |
|:---|:---| :---|
|`ctx` |_context.Context_ |用于超时/取消调用的自定义上下文 |
|`bucketName`  | _string_  |存储桶名称   |
|`objectName` | _string_  |对象的名称   |
|`expiry` | _time.Duration_  |presigned URL的过期时间，单位是秒 |


__示例__


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
生成一个用于HTTP GET操作的presigned URL。浏览器/移动客户端可以在即使存储桶为私有的情况下也可以通过这个URL进行下载。这个presigned URL可以有一个过期时间，默认是7天。

__参数__

|参数   |类型   |描述   |
|:---|:---| :---|
|`ctx` |_context.Contex_ |用于超时/取消调用的自定义上下文 |
|`bucketName`  | _string_  |存储桶名称   |
|`objectName` | _string_  |对象的名称   |
|`expiry` | _time.Duration_  |presigned URL的过期时间，单位是秒   |
|`reqParams` | _url.Values_  |额外的响应头，支持_response-expires_， _response-content-type_， _response-cache-control_， _response-content-disposition_。  |


__示例__


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
允许给POST操作的presigned URL设置策略条件。这些策略包括比如，接收对象上传的存储桶名称，名称前缀，过期策略。

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

## 5. 存储桶策略/通知

<a name="SetBucketPolicy"></a>
### SetBucketPolicy(ctx context.Context, bucketname, policy string) error
给存储桶或者对象前缀设置访问权限。

__参数__


|参数   |类型   |描述   |
|:---|:---| :---|
|`ctx` |_context.Context_ |用于超时/取消调用的自定义上下文 |
|`bucketName` | _string_  |存储桶名称|
|`objectPrefix` | _string_  |对象的名称前缀|
|`policy` | _policy.BucketPolicy_  |要设置的policy |


__返回值__


|参数   |类型   |描述   |
|:---|:---| :---|
|`err` | _error_  |标准Error   |


__示例__


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
获取存储桶或者对象前缀的访问权限。

__参数__


|参数   |类型   |描述   |
|:---|:---| :---|
|`ctx` |_context.Context_ |用于超时/取消调用的自定义上下文 |
|`bucketName`  | _string_  |存储桶名称   |

__返回值__


|参数   |类型   |描述   |
|:---|:---| :---|
|`policy`  | _string_ |返回的policy   |
|`err` | _error_  |标准Error  |

__示例__


```go
policy, err := minioClient.GetBucketPolicy(context.Background(), "my-bucketname")
if err != nil {
    log.Fatalln(err)
}
```

<a name="GetBucketNotification"></a>
### GetBucketNotification(ctx context.Context, bucketName string) (notification.Configuration, error)
获取存储桶的通知配置

__参数__


|参数   |类型   |描述   |
|:---|:---| :---|
|`ctx` |_context.Context_ |用于超时/取消调用的自定义上下文 |
|`bucketName`  | _string_  |存储桶名称 |

__返回值__


|参数   |类型   |描述   |
|:---|:---| :---|
|`config`  | _notification.Configuration_ |含有所有通知配置的数据结构|
|`err` | _error_  |标准Error  |

__示例__


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
给存储桶设置新的通知

__参数__


|参数   |类型   |描述   |
|:---|:---| :---|
|`ctx` |_context.Context_ |用于超时/取消调用的自定义上下文 |
|`bucketName`  | _string_  |存储桶名称   |
|`bucketNotification`  | _minio.BucketNotification_  |发送给配置的web service的XML  |

__返回值__


|参数   |类型   |描述   |
|:---|:---| :---|
|`err` | _error_  |标准Error  |

__示例__


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
删除存储桶上所有配置的通知

__参数__


|参数   |类型   |描述   |
|:---|:---| :---|
|`ctx` |_context.Context_ |用于超时/取消调用的自定义上下文 |
|`bucketName`  | _string_  |存储桶名称   |

__返回值__


|参数   |类型   |描述   |
|:---|:---| :---|
|`err` | _error_  |标准Error  |

__示例__


```go
err = minioClient.RemoveAllBucketNotification(context.Background(), "mybucket")
if err != nil {
    fmt.Println("Unable to remove bucket notifications.", err)
    return
}
```

<a name="ListenBucketNotification"></a>
### ListenBucketNotification(context context.Context, bucketName, prefix, suffix string, events []string) <-chan notification.Info
ListenBucketNotification API通过notification channel接收存储桶通知事件。返回的notification channel有两个属性，'Records'和'Err'。

- 'Records'持有从服务器返回的通知信息。
- 'Err'表示的是处理接收到的通知时报的任何错误。

注意：一旦报错，notification channel就会关闭。

__参数__


|参数   |类型   |描述   |
|:---|:---| :---|
|`ctx` |_context.Context_ |用于超时/取消调用的自定义上下文 |
|`bucketName`  | _string_  | 被监听通知的存储桶   |
|`prefix`  | _string_ | 过滤通知的对象前缀  |
|`suffix`  | _string_ | 过滤通知的对象后缀  |
|`events`  | _[]string_ | 开启指定事件类型的通知 |
|`doneCh`  | _chan struct{}_ | 在该channel上结束ListenBucketNotification iterator的一个message。  |

__返回值__

|参数   |类型   |描述   |
|:---|:---| :---|
|`notificationInfo` | _chan minio.NotificationInfo_ | 存储桶通知的channel |

__minio.NotificationInfo__

|属性   |类型   |描述   |
|`notificationInfo.Records` | _[]minio.NotificationEvent_ | 通知事件的集合 |
|`notificationInfo.Err` | _error_ | 操作时报的任何错误(标准Error) |


__示例__


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

## 6. 客户端自定义设置

<a name="SetAppInfo"></a>
### SetAppInfo(appName, appVersion string)
给User-Agent添加的自定义应用信息。

__参数__

| 参数  | 类型  | 描述  |
|---|---|---|
|`appName`  | _string_  | 发请求的应用名称 |
| `appVersion`| _string_ | 发请求的应用版本 |


__示例__


```go
// Set Application name and version to be used in subsequent API requests.
minioClient.SetAppInfo("myCloudApp", "1.0.0")
```

<a name="TraceOn"></a>
### TraceOn(outputStream io.Writer)
开启HTTP tracing。追踪信息输出到io.Writer，如果outputstream为nil，则trace写入到os.Stdout标准输出。

__参数__

| 参数  | 类型  | 描述  |
|---|---|---|
|`outputStream`  | _io.Writer_  | HTTP trace写入到outputStream |


<a name="TraceOff"></a>
### TraceOff()
关闭HTTP tracing。

<a name="SetS3TransferAccelerate"></a>
### SetS3TransferAccelerate(acceleratedEndpoint string)
给后续所有API请求设置ASW S3传输加速endpoint。
注意：此API仅对AWS S3有效，对其它S3兼容的对象存储服务不生效。

__参数__

| 参数  | 类型  | 描述  |
|---|---|---|
|`acceleratedEndpoint`  | _string_  | 设置新的S3传输加速endpoint。|
