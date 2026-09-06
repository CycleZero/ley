package oss

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
)

// =============================================================================
// 错误映射
// =============================================================================

// minioError 包装 minio.ErrorResponse，通过 Is 映射到统一哨兵错误
type minioError struct {
	code    string // S3 错误码（NoSuchKey / NoSuchBucket / AccessDenied ...）
	message string
	err     error
}

func (e *minioError) Error() string {
	return fmt.Sprintf("oss/minio: code=%s message=%s (%v)", e.code, e.message, e.err)
}

func (e *minioError) Unwrap() error { return e.err }

// Is 将 S3 错误码映射为 oss 包统一哨兵错误
func (e *minioError) Is(target error) bool {
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

// toOssErr 将 minio-go 错误包装为带哨兵语义的错误；非 S3 响应错误原样返回
func toOssErr(err error) error {
	if err == nil {
		return nil
	}
	var respErr minio.ErrorResponse
	if errors.As(err, &respErr) {
		return &minioError{code: respErr.Code, message: respErr.Message, err: err}
	}
	return err
}

// =============================================================================
// MinioOSS 实现
// =============================================================================
//
// 基于 minio.Core 包装：Core 内嵌 *minio.Client（全部高层方法可用），
// 并额外暴露低层 API（分页列举 ListObjectsV2 / Multipart / 范围读）。
// 注意 Core 自身定义了 PutObject/GetObject/CopyObject 的低层版本，
// 会遮蔽 Client 同名高层方法，故高层调用显式走 core.Client.*。

// MinioOSS MinIO/S3 兼容实现
type MinioOSS struct {
	core   *minio.Core // 低层 API：分页列举 / multipart / 范围读
	bucket string
}

// NewMinioOSS 创建 MinIO OSS 客户端实例
func NewMinioOSS(core *minio.Core, bucket string) OSSWithBucket {
	return &MinioOSS{core: core, bucket: bucket}
}

// PutObject 上传对象（高层 API，SDK 内部对超大/未知大小文件自动转 multipart）
func (m *MinioOSS) PutObject(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	_, err := m.core.Client.PutObject(ctx, m.bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return toOssErr(err)
}

// GetObject 获取对象（单次请求同时返回内容与元信息，无需额外 Stat）
func (m *MinioOSS) GetObject(ctx context.Context, key string) (io.ReadCloser, *ObjectInfo, error) {
	reader, obj, _, err := m.core.GetObject(ctx, m.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, nil, toOssErr(err)
	}
	return reader, toObjectInfo(key, obj), nil
}

// GetObjectRange 范围读取对象
func (m *MinioOSS) GetObjectRange(ctx context.Context, key string, offset, length int64) (io.ReadCloser, *ObjectInfo, error) {
	if offset < 0 {
		return nil, nil, fmt.Errorf("%w: offset=%d", ErrInvalidParam, offset)
	}
	opts := minio.GetObjectOptions{}
	if length <= 0 {
		// 读到对象结尾
		if err := opts.SetRange(offset, 0); err != nil {
			return nil, nil, toOssErr(err)
		}
	} else {
		// S3 Range 头为闭区间，end = offset + length - 1
		if err := opts.SetRange(offset, offset+length-1); err != nil {
			return nil, nil, toOssErr(err)
		}
	}
	reader, obj, _, err := m.core.GetObject(ctx, m.bucket, key, opts)
	if err != nil {
		return nil, nil, toOssErr(err)
	}
	return reader, toObjectInfo(key, obj), nil
}

// StatObject 获取对象元信息
func (m *MinioOSS) StatObject(ctx context.Context, key string) (*ObjectInfo, error) {
	stat, err := m.core.StatObject(ctx, m.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return nil, toOssErr(err)
	}
	return toObjectInfo(key, stat), nil
}

// DeleteObject 删除单个对象（对象不存在不视为错误，与 S3 语义一致）
func (m *MinioOSS) DeleteObject(ctx context.Context, key string) error {
	return toOssErr(m.core.RemoveObject(ctx, m.bucket, key, minio.RemoveObjectOptions{}))
}

// DeleteObjects 批量删除对象（SDK 内部按 S3 单请求 1000 上限自动分块；
// 删除不存在的对象不报错，与 S3 语义一致）
func (m *MinioOSS) DeleteObjects(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	objCh := make(chan minio.ObjectInfo, len(keys))
	go func() {
		defer close(objCh)
		for _, k := range keys {
			objCh <- minio.ObjectInfo{Key: k}
		}
	}()
	errCh := m.core.RemoveObjects(ctx, m.bucket, objCh, minio.RemoveObjectsOptions{})
	for e := range errCh {
		if e.Err != nil {
			return toOssErr(e.Err)
		}
	}
	return nil
}

// CopyObject 复制对象（同桶服务端复制）
func (m *MinioOSS) CopyObject(ctx context.Context, sourceKey, destKey string) error {
	// Core.CopyObject 是低层版本，这里显式走高层 Client 方法
	_, err := m.core.Client.CopyObject(ctx, minio.CopyDestOptions{
		Bucket: m.bucket,
		Object: destKey,
	}, minio.CopySrcOptions{
		Bucket: m.bucket,
		Object: sourceKey,
	})
	return toOssErr(err)
}

// ListObjects 分页列举对象
// 注意：minio.Core.ListObjectsV2 不接受 context（SDK 限制），
// 长列表中途取消需依赖请求超时，调用方应限制单页大小。
func (m *MinioOSS) ListObjects(ctx context.Context, prefix string, limit int, token string) ([]ObjectInfo, string, error) {
	_ = ctx // Core.ListObjectsV2 无 ctx 参数，见上方注释
	if limit <= 0 || limit > 1000 {
		limit = 1000 // S3 ListObjectsV2 单页上限
	}
	result, err := m.core.ListObjectsV2(m.bucket, prefix, "", token, "", limit)
	if err != nil {
		return nil, "", toOssErr(err)
	}
	objects := make([]ObjectInfo, 0, len(result.Contents))
	for _, obj := range result.Contents {
		objects = append(objects, *toObjectInfo(obj.Key, obj))
	}
	nextToken := ""
	if result.IsTruncated {
		nextToken = result.NextContinuationToken
	}
	return objects, nextToken, nil
}

// GetPresignedURL 获取预签名 GET URL
func (m *MinioOSS) GetPresignedURL(ctx context.Context, key string, expirySeconds int64) (string, error) {
	reqParams := make(url.Values)
	u, err := m.core.PresignedGetObject(ctx, m.bucket, key, time.Duration(expirySeconds)*time.Second, reqParams)
	if err != nil {
		return "", toOssErr(err)
	}
	return u.String(), nil
}

// GetPresignedPutURL 获取预签名 PUT URL（客户端直传）
// Content-Type 仅作提示，不参与签名——客户端 PUT 时可携带任意（或省略）该头
func (m *MinioOSS) GetPresignedPutURL(ctx context.Context, key string, contentType string, expirySeconds int64) (string, error) {
	u, err := m.core.PresignedPutObject(ctx, m.bucket, key, time.Duration(expirySeconds)*time.Second)
	if err != nil {
		return "", toOssErr(err)
	}
	return u.String(), nil
}

// ====== Multipart 分片上传 ======

// InitiateMultipartUpload 初始化分片上传
func (m *MinioOSS) InitiateMultipartUpload(ctx context.Context, key, contentType string) (string, error) {
	uploadID, err := m.core.NewMultipartUpload(ctx, m.bucket, key, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", toOssErr(err)
	}
	return uploadID, nil
}

// UploadPart 上传单个分片
func (m *MinioOSS) UploadPart(ctx context.Context, key, uploadID string, partNumber int32, reader io.Reader, size int64) (Part, error) {
	result, err := m.core.PutObjectPart(ctx, m.bucket, key, uploadID, int(partNumber), reader, size, minio.PutObjectPartOptions{})
	if err != nil {
		return Part{}, toOssErr(err)
	}
	return Part{PartNumber: partNumber, ETag: result.ETag}, nil
}

// CompleteMultipartUpload 完成分片上传
func (m *MinioOSS) CompleteMultipartUpload(ctx context.Context, key, uploadID string, parts []Part) error {
	completeParts := make([]minio.CompletePart, 0, len(parts))
	for _, p := range parts {
		completeParts = append(completeParts, minio.CompletePart{
			PartNumber: int(p.PartNumber),
			ETag:       p.ETag,
		})
	}
	_, err := m.core.CompleteMultipartUpload(ctx, m.bucket, key, uploadID, completeParts, minio.PutObjectOptions{})
	return toOssErr(err)
}

// AbortMultipartUpload 取消分片上传
func (m *MinioOSS) AbortMultipartUpload(ctx context.Context, key, uploadID string) error {
	return toOssErr(m.core.AbortMultipartUpload(ctx, m.bucket, key, uploadID))
}

// ====== 桶操作 ======

// BucketExists 检查桶是否存在
func (m *MinioOSS) BucketExists(ctx context.Context, bucket string) (bool, error) {
	exists, err := m.core.BucketExists(ctx, bucket)
	return exists, toOssErr(err)
}

// MakeBucket 创建桶
func (m *MinioOSS) MakeBucket(ctx context.Context, bucket string) error {
	return toOssErr(m.core.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}))
}

// RemoveBucket 删除桶
func (m *MinioOSS) RemoveBucket(ctx context.Context, bucket string) error {
	return toOssErr(m.core.RemoveBucket(ctx, bucket))
}

// ListBuckets 列出所有桶
func (m *MinioOSS) ListBuckets(ctx context.Context) ([]BucketInfo, error) {
	buckets, err := m.core.ListBuckets(ctx)
	if err != nil {
		return nil, toOssErr(err)
	}
	result := make([]BucketInfo, 0, len(buckets))
	for _, b := range buckets {
		result = append(result, BucketInfo{
			Name:         b.Name,
			CreationDate: b.CreationDate.Format(time.RFC3339),
		})
	}
	return result, nil
}

// toObjectInfo 将 minio.ObjectInfo 转换为统一 ObjectInfo
func toObjectInfo(key string, obj minio.ObjectInfo) *ObjectInfo {
	return &ObjectInfo{
		Key:          key,
		Size:         obj.Size,
		ContentType:  obj.ContentType,
		LastModified: obj.LastModified,
		ETag:         obj.ETag,
	}
}
