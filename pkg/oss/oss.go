package oss

import (
	"context"
	"io"
)

// ObjectInfo 存储对象的基本信息
type ObjectInfo struct {
	Key          string
	Size         int64
	ContentType  string
	LastModified string
	ETag         string
}

// OSS 通用存储接口
// 支持多种OSS实现（如MinIO、阿里云OSS、AWS S3等）
type OSS interface {
	// PutObject 上传对象
	// key: 对象键名
	// reader: 数据读取器
	// size: 数据大小（-1表示未知）
	// contentType: 内容类型
	PutObject(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error

	// GetObject 获取对象
	// 返回可读取的对象内容和对象信息
	GetObject(ctx context.Context, key string) (io.ReadCloser, *ObjectInfo, error)

	// StatObject 获取对象元信息
	StatObject(ctx context.Context, key string) (*ObjectInfo, error)

	// DeleteObject 删除对象
	DeleteObject(ctx context.Context, key string) error

	// GetPresignedURL 获取预签名URL
	// expirySeconds: 过期时间（秒）
	GetPresignedURL(ctx context.Context, key string, expirySeconds int64) (string, error)

	// CopyObject 复制对象
	CopyObject(ctx context.Context, sourceKey, destKey string) error

	// ListObjects 列举对象
	// prefix: 对象前缀过滤
	ListObjects(ctx context.Context, prefix string) ([]ObjectInfo, error)

	// GetPresignedPutURL 获取预签名 PUT URL（用于客户端直传上传）
	// key: 对象键名
	// contentType: 内容类型
	// expirySeconds: 过期时间（秒）
	GetPresignedPutURL(ctx context.Context, key string, contentType string, expirySeconds int64) (string, error)
}

// BucketInfo 桶的基本信息
type BucketInfo struct {
	Name         string
	CreationDate string
}

// OSSWithBucket 支持桶操作的 OSS 接口扩展
// 提供创建、删除、列举桶等管理功能
// 注意：如果没有在业务中操作桶的需求，则应该优先使用 OSS 接口，而不是此接口
type OSSWithBucket interface {
	OSS

	// BucketExists 检查桶是否存在
	BucketExists(ctx context.Context, bucket string) (bool, error)

	// MakeBucket 创建桶
	MakeBucket(ctx context.Context, bucket string) error

	// RemoveBucket 删除桶
	RemoveBucket(ctx context.Context, bucket string) error

	// ListBuckets 列出所有桶
	ListBuckets(ctx context.Context) ([]BucketInfo, error)
}
