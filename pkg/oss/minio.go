package oss

import (
	"bytes"
	"context"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
)

// MinioOSS MinIO实现
type MinioOSS struct {
	client *minio.Client
	bucket string
}

// NewMinioOSS 创建MinIO OSS客户端
func NewMinioOSS(client *minio.Client, bucket string) OSSWithBucket {
	return &MinioOSS{
		client: client,
		bucket: bucket,
	}
}

// PutObject 上传对象
func (m *MinioOSS) PutObject(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	opts := minio.PutObjectOptions{
		ContentType: contentType,
	}
	_, err := m.client.PutObject(ctx, m.bucket, key, reader, size, opts)
	return err
}

// GetObject 获取对象
func (m *MinioOSS) GetObject(ctx context.Context, key string) (io.ReadCloser, *ObjectInfo, error) {
	obj, err := m.client.GetObject(ctx, m.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, nil, err
	}
	stat, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, nil, err
	}
	info := &ObjectInfo{
		Key:          key,
		Size:         stat.Size,
		ContentType:  stat.ContentType,
		LastModified: stat.LastModified.Format(time.RFC3339),
		ETag:         stat.ETag,
	}
	return obj, info, nil
}

// StatObject 获取对象元信息
func (m *MinioOSS) StatObject(ctx context.Context, key string) (*ObjectInfo, error) {
	stat, err := m.client.StatObject(ctx, m.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return nil, err
	}
	return &ObjectInfo{
		Key:          key,
		Size:         stat.Size,
		ContentType:  stat.ContentType,
		LastModified: stat.LastModified.Format(time.RFC3339),
		ETag:         stat.ETag,
	}, nil
}

// DeleteObject 删除对象
func (m *MinioOSS) DeleteObject(ctx context.Context, key string) error {
	return m.client.RemoveObject(ctx, m.bucket, key, minio.RemoveObjectOptions{})
}

// GetPresignedURL 获取预签名URL
func (m *MinioOSS) GetPresignedURL(ctx context.Context, key string, expirySeconds int64) (string, error) {
	reqParams := make(url.Values)
	presignedURL, err := m.client.PresignedGetObject(ctx, m.bucket, key, time.Duration(expirySeconds)*time.Second, reqParams)
	if err != nil {
		return "", err
	}
	return presignedURL.String(), nil
}

// CopyObject 复制对象
func (m *MinioOSS) CopyObject(ctx context.Context, sourceKey, destKey string) error {
	_, err := m.client.CopyObject(ctx, minio.CopyDestOptions{
		Bucket: m.bucket,
		Object: destKey,
	}, minio.CopySrcOptions{
		Bucket: m.bucket,
		Object: sourceKey,
	})
	return err
}

// ListObjects 列举对象
func (m *MinioOSS) ListObjects(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	objects := m.client.ListObjects(ctx, m.bucket, minio.ListObjectsOptions{
		Prefix: prefix,
	})
	var result []ObjectInfo
	for obj := range objects {
		if obj.Err != nil {
			return nil, obj.Err
		}
		result = append(result, ObjectInfo{
			Key:          obj.Key,
			Size:         obj.Size,
			ContentType:  obj.ContentType,
			LastModified: obj.LastModified.Format(time.RFC3339),
			ETag:         obj.ETag,
		})
	}
	return result, nil
}

// GetObjectBytes 获取对象全部内容（适合小文件）
func (m *MinioOSS) GetObjectBytes(ctx context.Context, key string) ([]byte, *ObjectInfo, error) {
	obj, info, err := m.GetObject(ctx, key)
	if err != nil {
		return nil, nil, err
	}
	defer obj.Close()
	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, nil, err
	}
	return data, info, nil
}

// PutObjectBytes 上传字节数组（适合小文件）
func (m *MinioOSS) PutObjectBytes(ctx context.Context, key string, data []byte, contentType string) error {
	return m.PutObject(ctx, key, bytes.NewReader(data), int64(len(data)), contentType)
}

// GetPresignedPutURL 获取预签名 PUT URL（用于客户端直传上传）
func (m *MinioOSS) GetPresignedPutURL(ctx context.Context, key string, contentType string, expirySeconds int64) (string, error) {
	reqParams := make(url.Values)
	reqParams.Set("Content-Type", contentType)
	presignedURL, err := m.client.PresignedPutObject(ctx, m.bucket, key, time.Duration(expirySeconds)*time.Second)
	if err != nil {
		return "", err
	}
	// 添加 Content-Type 到 URL 查询参数
	return presignedURL.String(), nil
}

// ====== 桶操作 (Bucket Operations) ======

// BucketExists 检查桶是否存在
func (m *MinioOSS) BucketExists(ctx context.Context, bucket string) (bool, error) {
	return m.client.BucketExists(ctx, bucket)
}

// MakeBucket 创建桶
func (m *MinioOSS) MakeBucket(ctx context.Context, bucket string) error {
	return m.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
}

// RemoveBucket 删除桶
func (m *MinioOSS) RemoveBucket(ctx context.Context, bucket string) error {
	return m.client.RemoveBucket(ctx, bucket)
}

// ListBuckets 列出所有桶
func (m *MinioOSS) ListBuckets(ctx context.Context) ([]BucketInfo, error) {
	buckets, err := m.client.ListBuckets(ctx)
	if err != nil {
		return nil, err
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
