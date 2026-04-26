package oss

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// 定义错误类型
var (
	ErrInvalidConfig     = errors.New("cos: invalid configuration, secretID/secretKey/region/bucketName is required")
	ErrInvalidExpiryTime = errors.New("cos: invalid expiry time, must be between 1 second and 7 days")
	ErrEmptyKey          = errors.New("cos: object key cannot be empty")
	ErrBucketNotFound    = errors.New("cos: bucket not found")
	ErrObjectNotFound    = errors.New("cos: object not found")
	ErrAccessDenied      = errors.New("cos: access denied")
)

// CosError 包装COS错误，提供更详细的错误信息
type CosError struct {
	Code      string // COS错误码
	Message   string // 错误消息
	RequestID string // 请求ID，用于问题追踪
	Resource  string // 资源路径
	Err       error  // 原始错误
}

func (e *CosError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("cos error: code=%s, message=%s, requestID=%s, resource=%s", e.Code, e.Message, e.RequestID, e.Resource)
	}
	return fmt.Sprintf("cos error: code=%s, message=%s, resource=%s", e.Code, e.Message, e.Resource)
}

func (e *CosError) Unwrap() error {
	return e.Err
}

// Is 判断错误类型
func (e *CosError) Is(target error) bool {
	switch e.Code {
	case "NoSuchBucket":
		return errors.Is(target, ErrBucketNotFound)
	case "NoSuchKey":
		return errors.Is(target, ErrObjectNotFound)
	case "AccessDenied", "UnauthorizedAccess":
		return errors.Is(target, ErrAccessDenied)
	}
	return false
}

// Cos COS客户端实现
type Cos struct {
	bucketName string
	region     string
	appID      string // 从bucketName中提取的appid
	secretID   string // 用于生成预签名URL
	secretKey  string // 用于生成预签名URL
	client     *cos.Client
}

// CosConfig COS客户端初始化配置
type CosConfig struct {
	SecretID   string        // 腾讯云SecretID
	SecretKey  string        // 腾讯云SecretKey
	Region     string        // COS地域（如ap-guangzhou）
	BucketName string        // Bucket名称（如examplebucket-1250000000）
	Timeout    time.Duration // 客户端超时时间（默认30s）
	MaxRetries int           // 最大重试次数（默认3次）
	RetryDelay time.Duration // 重试间隔（默认100ms）
}

// NewCos 创建COS客户端实例
func NewCos(config CosConfig) (OSS, error) {
	// 校验必要参数
	if config.SecretID == "" || config.SecretKey == "" || config.Region == "" || config.BucketName == "" {
		return nil, ErrInvalidConfig
	}

	// 设置默认超时时间
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	// 设置默认重试次数
	maxRetries := config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	// 设置默认重试间隔
	retryDelay := config.RetryDelay
	if retryDelay <= 0 {
		retryDelay = 100 * time.Millisecond
	}

	// 提取appid（bucketName格式为 bucketname-appid）
	appID := extractAppID(config.BucketName)

	// 构建COS Bucket URL
	bucketURL, err := url.Parse(fmt.Sprintf("https://%s.cos.%s.myqcloud.com", config.BucketName, config.Region))
	if err != nil {
		return nil, fmt.Errorf("cos: failed to parse bucket URL: %w", err)
	}

	baseURL := &cos.BaseURL{
		BucketURL: bucketURL,
	}

	// 初始化http.Client（带超时配置和重试）
	httpClient := &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  config.SecretID,
			SecretKey: config.SecretKey,
			Transport: &retryTransport{
				base:       http.DefaultTransport,
				maxRetries: maxRetries,
				retryDelay: retryDelay,
			},
		},
		Timeout: timeout,
	}

	// 创建COS Client
	client := cos.NewClient(baseURL, httpClient)

	return &Cos{
		bucketName: config.BucketName,
		region:     config.Region,
		appID:      appID,
		secretID:   config.SecretID,
		secretKey:  config.SecretKey,
		client:     client,
	}, nil
}

// extractAppID 从bucketName中提取appid
// bucketName格式为 bucketname-appid
func extractAppID(bucketName string) string {
	parts := strings.Split(bucketName, "-")
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}
	return ""
}

// retryTransport 带重试功能的Transport
type retryTransport struct {
	base       http.RoundTripper
	maxRetries int
	retryDelay time.Duration
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var lastErr error
	var resp *http.Response

	for i := 0; i <= t.maxRetries; i++ {
		if i > 0 {
			select {
			case <-req.Context().Done():
				return nil, req.Context().Err()
			case <-time.After(t.retryDelay):
			}
		}

		resp, lastErr = t.base.RoundTrip(req)
		if lastErr == nil && resp.StatusCode < 500 {
			return resp, nil
		}

		// 对于5xx错误或网络错误，进行重试
		if resp != nil && resp.StatusCode >= 500 {
			resp.Body.Close()
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return resp, nil
}

// PutObject 上传对象
func (c *Cos) PutObject(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	if key == "" {
		return ErrEmptyKey
	}

	opt := &cos.ObjectPutOptions{
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
			ContentType:   contentType,
			ContentLength: size,
		},
	}

	_, err := c.client.Object.Put(ctx, key, reader, opt)
	if err != nil {
		return c.wrapCosError(err)
	}
	return nil
}

// GetObject 获取对象
func (c *Cos) GetObject(ctx context.Context, key string) (io.ReadCloser, *ObjectInfo, error) {
	if key == "" {
		return nil, nil, ErrEmptyKey
	}

	resp, err := c.client.Object.Get(ctx, key, nil)
	if err != nil {
		return nil, nil, c.wrapCosError(err)
	}

	info := &ObjectInfo{
		Key:          key,
		Size:         resp.ContentLength,
		ContentType:  resp.Header.Get("Content-Type"),
		LastModified: resp.Header.Get("Last-Modified"),
		ETag:         resp.Header.Get("ETag"),
	}

	return resp.Body, info, nil
}

// StatObject 获取对象元信息
func (c *Cos) StatObject(ctx context.Context, key string) (*ObjectInfo, error) {
	if key == "" {
		return nil, ErrEmptyKey
	}

	resp, err := c.client.Object.Head(ctx, key, nil)
	if err != nil {
		return nil, c.wrapCosError(err)
	}

	return &ObjectInfo{
		Key:          key,
		Size:         resp.ContentLength,
		ContentType:  resp.Header.Get("Content-Type"),
		LastModified: resp.Header.Get("Last-Modified"),
		ETag:         resp.Header.Get("ETag"),
	}, nil
}

// DeleteObject 删除对象
func (c *Cos) DeleteObject(ctx context.Context, key string) error {
	if key == "" {
		return ErrEmptyKey
	}

	_, err := c.client.Object.Delete(ctx, key)
	if err != nil {
		return c.wrapCosError(err)
	}
	return nil
}

// GetPresignedURL 生成GET预签名URL
func (c *Cos) GetPresignedURL(ctx context.Context, key string, expirySeconds int64) (string, error) {
	if key == "" {
		return "", ErrEmptyKey
	}

	// 校验过期时间合法性（COS预签名URL最大有效期7天）
	if expirySeconds <= 0 || expirySeconds > 60*60*24*7 {
		return "", ErrInvalidExpiryTime
	}

	presignedURL, err := c.client.Object.GetPresignedURL(
		ctx,
		http.MethodGet,
		key,
		c.secretID,
		c.secretKey,
		time.Duration(expirySeconds)*time.Second,
		nil,
	)
	if err != nil {
		return "", c.wrapCosError(err)
	}

	return presignedURL.String(), nil
}

// GetPresignedPutURL 生成PUT预签名URL
func (c *Cos) GetPresignedPutURL(ctx context.Context, key string, contentType string, expirySeconds int64) (string, error) {
	if key == "" {
		return "", ErrEmptyKey
	}

	// 校验过期时间合法性
	if expirySeconds <= 0 || expirySeconds > 60*60*24*7 {
		return "", ErrInvalidExpiryTime
	}

	opt := &cos.PresignedURLOptions{
		Header: &http.Header{},
	}
	if contentType != "" {
		opt.Header.Set("Content-Type", contentType)
	}

	presignedURL, err := c.client.Object.GetPresignedURL(
		ctx,
		http.MethodPut,
		key,
		c.secretID,
		c.secretKey,
		time.Duration(expirySeconds)*time.Second,
		opt,
	)
	if err != nil {
		return "", c.wrapCosError(err)
	}

	return presignedURL.String(), nil
}

// CopyObject 复制对象
// 源对象格式为同一bucket内的key，或者跨bucket的完整路径
func (c *Cos) CopyObject(ctx context.Context, sourceKey, destKey string) error {
	if sourceKey == "" || destKey == "" {
		return ErrEmptyKey
	}

	// 构建源对象URL
	// 格式: bucketname-appid.cos.region.myqcloud.com/sourceKey
	sourceURL := fmt.Sprintf("%s.cos.%s.myqcloud.com/%s", c.bucketName, c.region, sourceKey)

	_, _, err := c.client.Object.Copy(ctx, destKey, sourceURL, nil)
	if err != nil {
		return c.wrapCosError(err)
	}
	return nil
}

// ListObjects 列举对象
func (c *Cos) ListObjects(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	var result []ObjectInfo
	var marker string

	for {
		opt := &cos.BucketGetOptions{
			Prefix:  prefix,
			Marker:  marker,
			MaxKeys: 1000,
		}

		resp, _, err := c.client.Bucket.Get(ctx, opt)
		if err != nil {
			return nil, c.wrapCosError(err)
		}

		for _, obj := range resp.Contents {
			result = append(result, ObjectInfo{
				Key:          obj.Key,
				Size:         obj.Size,
				LastModified: obj.LastModified,
				ETag:         obj.ETag,
				// 注意：COS ListObjects接口不返回ContentType
			})
		}

		// 如果没有更多数据，退出循环
		if !resp.IsTruncated {
			break
		}
		marker = resp.NextMarker
	}

	return result, nil
}

// wrapCosError 包装COS错误，便于上层区分错误类型
func (c *Cos) wrapCosError(err error) error {
	if err == nil {
		return nil
	}

	// 尝试解析响应体中的错误信息
	if respErr, ok := err.(*cos.ErrorResponse); ok {
		return &CosError{
			Code:      respErr.Code,
			Message:   respErr.Message,
			RequestID: respErr.RequestID,
			Resource:  respErr.Resource,
			Err:       err,
		}
	}

	// 检查是否为url.Error类型
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return &CosError{
			Code:    "NetworkError",
			Message: urlErr.Error(),
			Err:     err,
		}
	}

	return err
}

// GetBucketName 获取当前bucket名称
func (c *Cos) GetBucketName() string {
	return c.bucketName
}

// GetRegion 获取当前region
func (c *Cos) GetRegion() string {
	return c.region
}
