package oss

import (
	"context"
	"errors"
	"io"
	"time"
)

// =============================================================================
// 统一哨兵错误
// =============================================================================
//
// 所有实现（MinIO / 阿里云 OSS）必须将各自的底层错误映射到下列哨兵错误之一，
// 调用方用 errors.Is(err, oss.ErrXxx) 即可跨实现判断错误类型，无需感知具体 SDK。

var (
	// ErrObjectNotFound 对象不存在
	ErrObjectNotFound = errors.New("对象不存在")
	// ErrBucketNotFound 存储桶不存在
	ErrBucketNotFound = errors.New("存储桶不存在")
	// ErrAccessDenied 访问被拒绝（密钥错误 / 无权限）
	ErrAccessDenied = errors.New("访问被拒绝")
	// ErrEmptyKey 对象键不能为空
	ErrEmptyKey = errors.New("对象键不能为空")
	// ErrInvalidParam 参数不合法
	ErrInvalidParam = errors.New("参数不合法")
)

// =============================================================================
// 基础类型
// =============================================================================

// ObjectInfo 存储对象的基本信息
type ObjectInfo struct {
	Key          string
	Size         int64
	ContentType  string
	LastModified time.Time // 统一为 time.Time，各实现自行解析底层格式
	ETag         string
}

// Part 分片上传的一个分片结果（CompleteMultipartUpload 入参）
type Part struct {
	PartNumber int32
	ETag       string
}

// BucketInfo 桶的基本信息
type BucketInfo struct {
	Name         string
	CreationDate string
}

// =============================================================================
// OSS 通用存储接口
// =============================================================================

// OSS 通用对象存储接口，覆盖 S3 兼容存储的常见能力面：
//   - 基础对象操作：上传 / 下载 / 范围读 / 元信息 / 删除 / 批量删除 / 复制
//   - 分页列举
//   - 预签名 URL（GET / PUT 直传）
//   - Multipart 分片上传（大文件 / 断点续传，各实现内部对接自有 SDK）
//
// 支持多种后端（MinIO、阿里云 OSS 等），业务层依赖本接口编程。
type OSS interface {
	// PutObject 上传对象
	// key: 对象键名
	// reader: 数据读取器
	// size: 数据大小（-1 表示未知，实现应支持流式上传；超过单请求上限的大文件由实现内部自动转分片）
	// contentType: 内容类型
	PutObject(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error

	// GetObject 获取对象（完整内容）
	// 返回可读取的对象内容和对象信息
	GetObject(ctx context.Context, key string) (io.ReadCloser, *ObjectInfo, error)

	// GetObjectRange 范围读取对象
	// offset: 起始字节偏移（从 0 开始）
	// length: 读取长度；length <= 0 表示读到对象结尾
	// 用于图片裁剪、视频字节流、断点下载等场景
	GetObjectRange(ctx context.Context, key string, offset, length int64) (io.ReadCloser, *ObjectInfo, error)

	// StatObject 获取对象元信息
	StatObject(ctx context.Context, key string) (*ObjectInfo, error)

	// DeleteObject 删除单个对象
	DeleteObject(ctx context.Context, key string) error

	// DeleteObjects 批量删除对象
	// 实现负责分块（S3 单请求上限 1000 个）；keys 为空时直接返回 nil
	DeleteObjects(ctx context.Context, keys []string) error

	// CopyObject 复制对象（同桶）
	CopyObject(ctx context.Context, sourceKey, destKey string) error

	// ListObjects 分页列举对象
	// prefix: 对象前缀过滤
	// limit: 单页数量上限（<=0 时使用 1000）
	// token: 上一页返回的下一页令牌；第一页传 ""
	// 返回: 本页对象列表 + 下一页令牌（"" 表示已列举完毕）
	ListObjects(ctx context.Context, prefix string, limit int, token string) ([]ObjectInfo, string, error)

	// GetPresignedURL 获取预签名 GET URL（用于客户端直接下载）
	// expirySeconds: 过期时间（秒），实现应限制在 1s ~ 7 天内
	GetPresignedURL(ctx context.Context, key string, expirySeconds int64) (string, error)

	// GetPresignedPutURL 获取预签名 PUT URL（用于客户端直传上传）
	// key: 对象键名
	// contentType: 期望的内容类型（仅作提示，不强制校验；客户端以该类型 PUT 即可）
	// expirySeconds: 过期时间（秒），实现应限制在 1s ~ 7 天内
	GetPresignedPutURL(ctx context.Context, key string, contentType string, expirySeconds int64) (string, error)

	// InitiateMultipartUpload 初始化分片上传，返回 uploadID
	// 分片上传流程：Initiate → 多次 UploadPart → Complete / Abort
	InitiateMultipartUpload(ctx context.Context, key, contentType string) (string, error)

	// UploadPart 上传一个分片
	// partNumber: 分片序号，从 1 开始递增
	// size: 本分片数据大小；除最后一片外，单片大小不应小于各存储的块下限（S3 为 5MiB）
	// 返回本分片的 ETag（CompleteMultipartUpload 时需要）
	UploadPart(ctx context.Context, key, uploadID string, partNumber int32, reader io.Reader, size int64) (Part, error)

	// CompleteMultipartUpload 完成分片上传，合并为最终对象
	// parts: 各分片结果，须按 partNumber 升序排列
	CompleteMultipartUpload(ctx context.Context, key, uploadID string, parts []Part) error

	// AbortMultipartUpload 取消分片上传，释放已上传的分片
	AbortMultipartUpload(ctx context.Context, key, uploadID string) error
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
