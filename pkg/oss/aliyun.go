package oss

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	aliyun "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)

// =============================================================================
// 错误映射
// =============================================================================

// aliyunError 包装阿里云 ServiceError，通过 Is 映射到统一哨兵错误
type aliyunError struct {
	code    string // OSS 错误码（NoSuchKey / NoSuchBucket / AccessDenied ...）
	message string
	err     error
}

func (e *aliyunError) Error() string {
	return fmt.Sprintf("oss/aliyun: code=%s message=%s (%v)", e.code, e.message, e.err)
}

func (e *aliyunError) Unwrap() error { return e.err }

// Is 将 OSS 错误码映射为 oss 包统一哨兵错误
func (e *aliyunError) Is(target error) bool {
	switch e.code {
	case "NoSuchKey":
		return errors.Is(target, ErrObjectNotFound)
	case "NoSuchBucket":
		return errors.Is(target, ErrBucketNotFound)
	case "AccessDenied", "InvalidAccessKeyId", "SignatureDoesNotMatch":
		return errors.Is(target, ErrAccessDenied)
	}
	return false
}

// toAliyunErr 将 OSS SDK 错误包装为带哨兵语义的错误；非服务端响应错误原样返回
func toAliyunErr(err error) error {
	if err == nil {
		return nil
	}
	var se *aliyun.ServiceError
	if errors.As(err, &se) {
		return &aliyunError{code: se.Code, message: se.Message, err: err}
	}
	return err
}

// =============================================================================
// AliyunOSS 实现（官方 SDK v2：github.com/aliyun/alibabacloud-oss-go-sdk-v2）
// =============================================================================

// AliyunConfig 阿里云 OSS 客户端初始化配置
type AliyunConfig struct {
	Region          string // 地域，如 cn-hangzhou
	AccessKeyID     string // AccessKey ID
	AccessKeySecret string // AccessKey Secret
	BucketName      string // 默认操作的 Bucket 名称
}

// AliyunOSS 阿里云 OSS 实现
type AliyunOSS struct {
	client *aliyun.Client
	bucket string
}

// NewAliyunOSS 创建阿里云 OSS 客户端实例
func NewAliyunOSS(cfg AliyunConfig) (OSSWithBucket, error) {
	if cfg.Region == "" || cfg.AccessKeyID == "" || cfg.AccessKeySecret == "" || cfg.BucketName == "" {
		return nil, fmt.Errorf("%w: region/accessKeyID/accessKeySecret/bucketName 均必填", ErrInvalidParam)
	}
	client := aliyun.NewClient(
		aliyun.LoadDefaultConfig().
			WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.AccessKeySecret)).
			WithRegion(cfg.Region),
	)
	return &AliyunOSS{client: client, bucket: cfg.BucketName}, nil
}

// PutObject 上传对象
// size >= 0 时显式指定 Content-Length（服务端单请求上限 5GiB，超大文件应由调用方走 Multipart）
func (a *AliyunOSS) PutObject(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	req := &aliyun.PutObjectRequest{
		Bucket: aliyun.Ptr(a.bucket),
		Key:    aliyun.Ptr(key),
		Body:   reader,
	}
	if contentType != "" {
		req.ContentType = aliyun.Ptr(contentType)
	}
	if size >= 0 {
		req.ContentLength = aliyun.Ptr(size)
	}
	_, err := a.client.PutObject(ctx, req)
	return toAliyunErr(err)
}

// GetObject 获取对象（完整内容）
func (a *AliyunOSS) GetObject(ctx context.Context, key string) (io.ReadCloser, *ObjectInfo, error) {
	return a.getObject(ctx, key, "")
}

// GetObjectRange 范围读取对象
func (a *AliyunOSS) GetObjectRange(ctx context.Context, key string, offset, length int64) (io.ReadCloser, *ObjectInfo, error) {
	if offset < 0 {
		return nil, nil, fmt.Errorf("%w: offset=%d", ErrInvalidParam, offset)
	}
	// 构造 Range 头（闭区间）；length <= 0 表示读到对象结尾
	var rangeHeader string
	if length <= 0 {
		rangeHeader = fmt.Sprintf("bytes=%d-", offset)
	} else {
		rangeHeader = fmt.Sprintf("bytes=%d-%d", offset, offset+length-1)
	}
	return a.getObject(ctx, key, rangeHeader)
}

// getObject 底层获取对象（rangeHeader 为空表示完整获取）
func (a *AliyunOSS) getObject(ctx context.Context, key string, rangeHeader string) (io.ReadCloser, *ObjectInfo, error) {
	req := &aliyun.GetObjectRequest{
		Bucket: aliyun.Ptr(a.bucket),
		Key:    aliyun.Ptr(key),
	}
	if rangeHeader != "" {
		req.Range = aliyun.Ptr(rangeHeader)
	}
	result, err := a.client.GetObject(ctx, req)
	if err != nil {
		return nil, nil, toAliyunErr(err)
	}
	info := &ObjectInfo{
		Key:          key,
		Size:         result.ContentLength,
		ContentType:  derefStr(result.ContentType),
		LastModified: derefTime(result.LastModified),
		ETag:         derefStr(result.ETag),
	}
	return result.Body, info, nil
}

// StatObject 获取对象元信息
func (a *AliyunOSS) StatObject(ctx context.Context, key string) (*ObjectInfo, error) {
	result, err := a.client.HeadObject(ctx, &aliyun.HeadObjectRequest{
		Bucket: aliyun.Ptr(a.bucket),
		Key:    aliyun.Ptr(key),
	})
	if err != nil {
		return nil, toAliyunErr(err)
	}
	return &ObjectInfo{
		Key:          key,
		Size:         result.ContentLength,
		ContentType:  derefStr(result.ContentType),
		LastModified: derefTime(result.LastModified),
		ETag:         derefStr(result.ETag),
	}, nil
}

// DeleteObject 删除单个对象（对象不存在不视为错误，与 S3 语义一致）
func (a *AliyunOSS) DeleteObject(ctx context.Context, key string) error {
	_, err := a.client.DeleteObject(ctx, &aliyun.DeleteObjectRequest{
		Bucket: aliyun.Ptr(a.bucket),
		Key:    aliyun.Ptr(key),
	})
	return toAliyunErr(err)
}

// DeleteObjects 批量删除对象
// OSS 单请求上限 1000 个，超出自动分块；删除不存在的对象不报错
func (a *AliyunOSS) DeleteObjects(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	const maxBatch = 1000
	for start := 0; start < len(keys); start += maxBatch {
		end := min(start+maxBatch, len(keys))
		objects := make([]aliyun.ObjectIdentifier, 0, end-start)
		for _, k := range keys[start:end] {
			objects = append(objects, aliyun.ObjectIdentifier{Key: aliyun.Ptr(k)})
		}
		_, err := a.client.DeleteMultipleObjects(ctx, &aliyun.DeleteMultipleObjectsRequest{
			Bucket: aliyun.Ptr(a.bucket),
			Delete: &aliyun.Delete{
				Objects: objects,
				Quiet:   true,
			},
		})
		if err != nil {
			return toAliyunErr(err)
		}
	}
	return nil
}

// CopyObject 复制对象（同 bucket 服务端复制）
func (a *AliyunOSS) CopyObject(ctx context.Context, sourceKey, destKey string) error {
	_, err := a.client.CopyObject(ctx, &aliyun.CopyObjectRequest{
		Bucket:       aliyun.Ptr(a.bucket),
		Key:          aliyun.Ptr(destKey),
		SourceBucket: aliyun.Ptr(a.bucket),
		SourceKey:    aliyun.Ptr(sourceKey),
	})
	return toAliyunErr(err)
}

// ListObjects 分页列举对象
func (a *AliyunOSS) ListObjects(ctx context.Context, prefix string, limit int, token string) ([]ObjectInfo, string, error) {
	if limit <= 0 || limit > 1000 {
		limit = 1000 // OSS ListObjectsV2 单页上限
	}
	req := &aliyun.ListObjectsV2Request{
		Bucket:  aliyun.Ptr(a.bucket),
		MaxKeys: int32(limit),
	}
	if prefix != "" {
		req.Prefix = aliyun.Ptr(prefix)
	}
	if token != "" {
		req.ContinuationToken = aliyun.Ptr(token)
	}
	result, err := a.client.ListObjectsV2(ctx, req)
	if err != nil {
		return nil, "", toAliyunErr(err)
	}
	objects := make([]ObjectInfo, 0, len(result.Contents))
	for _, obj := range result.Contents {
		objects = append(objects, ObjectInfo{
			Key:          derefStr(obj.Key),
			Size:         obj.Size,
			ContentType:  "", // ListObjectsV2 不返回 ContentType（S3 协议一致），需要时用 StatObject
			LastModified: derefTime(obj.LastModified),
			ETag:         derefStr(obj.ETag),
		})
	}
	nextToken := ""
	if result.IsTruncated && result.NextContinuationToken != nil {
		nextToken = *result.NextContinuationToken
	}
	return objects, nextToken, nil
}

// GetPresignedURL 获取预签名 GET URL
func (a *AliyunOSS) GetPresignedURL(ctx context.Context, key string, expirySeconds int64) (string, error) {
	if err := validateExpiry(expirySeconds); err != nil {
		return "", err
	}
	result, err := a.client.Presign(ctx, &aliyun.GetObjectRequest{
		Bucket: aliyun.Ptr(a.bucket),
		Key:    aliyun.Ptr(key),
	}, aliyun.PresignExpires(time.Duration(expirySeconds)*time.Second))
	if err != nil {
		return "", toAliyunErr(err)
	}
	return result.URL, nil
}

// GetPresignedPutURL 获取预签名 PUT URL（客户端直传）
// Content-Type 仅作提示，不参与签名——客户端 PUT 时可携带任意（或省略）该头
func (a *AliyunOSS) GetPresignedPutURL(ctx context.Context, key string, contentType string, expirySeconds int64) (string, error) {
	if err := validateExpiry(expirySeconds); err != nil {
		return "", err
	}
	req := &aliyun.PutObjectRequest{
		Bucket: aliyun.Ptr(a.bucket),
		Key:    aliyun.Ptr(key),
	}
	if contentType != "" {
		req.ContentType = aliyun.Ptr(contentType)
	}
	result, err := a.client.Presign(ctx, req, aliyun.PresignExpires(time.Duration(expirySeconds)*time.Second))
	if err != nil {
		return "", toAliyunErr(err)
	}
	return result.URL, nil
}

// ====== Multipart 分片上传 ======

// InitiateMultipartUpload 初始化分片上传
func (a *AliyunOSS) InitiateMultipartUpload(ctx context.Context, key, contentType string) (string, error) {
	req := &aliyun.InitiateMultipartUploadRequest{
		Bucket: aliyun.Ptr(a.bucket),
		Key:    aliyun.Ptr(key),
	}
	if contentType != "" {
		req.ContentType = aliyun.Ptr(contentType)
	}
	result, err := a.client.InitiateMultipartUpload(ctx, req)
	if err != nil {
		return "", toAliyunErr(err)
	}
	if result.UploadId == nil {
		return "", fmt.Errorf("%w: 响应缺少 uploadId", ErrInvalidParam)
	}
	return *result.UploadId, nil
}

// UploadPart 上传单个分片
func (a *AliyunOSS) UploadPart(ctx context.Context, key, uploadID string, partNumber int32, reader io.Reader, size int64) (Part, error) {
	result, err := a.client.UploadPart(ctx, &aliyun.UploadPartRequest{
		Bucket:     aliyun.Ptr(a.bucket),
		Key:        aliyun.Ptr(key),
		UploadId:   aliyun.Ptr(uploadID),
		PartNumber: partNumber,
		Body:       reader,
	})
	if err != nil {
		return Part{}, toAliyunErr(err)
	}
	return Part{PartNumber: partNumber, ETag: derefStr(result.ETag)}, nil
}

// CompleteMultipartUpload 完成分片上传（parts 须按 partNumber 升序）
func (a *AliyunOSS) CompleteMultipartUpload(ctx context.Context, key, uploadID string, parts []Part) error {
	uploadParts := make([]aliyun.UploadPart, 0, len(parts))
	for _, p := range parts {
		uploadParts = append(uploadParts, aliyun.UploadPart{
			PartNumber: p.PartNumber,
			ETag:       aliyun.Ptr(p.ETag),
		})
	}
	_, err := a.client.CompleteMultipartUpload(ctx, &aliyun.CompleteMultipartUploadRequest{
		Bucket:   aliyun.Ptr(a.bucket),
		Key:      aliyun.Ptr(key),
		UploadId: aliyun.Ptr(uploadID),
		CompleteMultipartUpload: &aliyun.CompleteMultipartUpload{
			Parts: uploadParts,
		},
	})
	return toAliyunErr(err)
}

// AbortMultipartUpload 取消分片上传
func (a *AliyunOSS) AbortMultipartUpload(ctx context.Context, key, uploadID string) error {
	_, err := a.client.AbortMultipartUpload(ctx, &aliyun.AbortMultipartUploadRequest{
		Bucket:   aliyun.Ptr(a.bucket),
		Key:      aliyun.Ptr(key),
		UploadId: aliyun.Ptr(uploadID),
	})
	return toAliyunErr(err)
}

// ====== 桶操作 ======

// BucketExists 检查桶是否存在
func (a *AliyunOSS) BucketExists(ctx context.Context, bucket string) (bool, error) {
	_, err := a.client.GetBucketInfo(ctx, &aliyun.GetBucketInfoRequest{
		Bucket: aliyun.Ptr(bucket),
	})
	if err == nil {
		return true, nil
	}
	if errors.Is(toAliyunErr(err), ErrBucketNotFound) {
		return false, nil
	}
	return false, toAliyunErr(err)
}

// MakeBucket 创建桶
func (a *AliyunOSS) MakeBucket(ctx context.Context, bucket string) error {
	_, err := a.client.PutBucket(ctx, &aliyun.PutBucketRequest{
		Bucket: aliyun.Ptr(bucket),
	})
	return toAliyunErr(err)
}

// RemoveBucket 删除桶
func (a *AliyunOSS) RemoveBucket(ctx context.Context, bucket string) error {
	_, err := a.client.DeleteBucket(ctx, &aliyun.DeleteBucketRequest{
		Bucket: aliyun.Ptr(bucket),
	})
	return toAliyunErr(err)
}

// ListBuckets 列出所有桶
func (a *AliyunOSS) ListBuckets(ctx context.Context) ([]BucketInfo, error) {
	result, err := a.client.ListBuckets(ctx, &aliyun.ListBucketsRequest{})
	if err != nil {
		return nil, toAliyunErr(err)
	}
	buckets := make([]BucketInfo, 0, len(result.Buckets))
	for _, b := range result.Buckets {
		buckets = append(buckets, BucketInfo{
			Name:         derefStr(b.Name),
			CreationDate: derefTime(b.CreationDate).Format(time.RFC3339),
		})
	}
	return buckets, nil
}

// ====== 辅助函数 ======

// validateExpiry 校验预签名过期时间（S3/OSS 限制：1s ~ 7 天）
func validateExpiry(expirySeconds int64) error {
	if expirySeconds <= 0 || expirySeconds > 7*24*3600 {
		return fmt.Errorf("%w: 过期时间需在 1s ~ 7 天内，实际 %ds", ErrInvalidParam, expirySeconds)
	}
	return nil
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func derefTime(p *time.Time) time.Time {
	if p == nil {
		return time.Time{}
	}
	return *p
}
