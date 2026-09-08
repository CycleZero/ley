package oss

import (
	"context"
	"io"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// tracerName 对象存储插桩使用的 tracer 名称。
const tracerName = "ley/otel"

// OSSTraceOption 配置 OSS 插桩。
type OSSTraceOption func(*ossTraceOptions)

type ossTraceOptions struct {
	tp trace.TracerProvider
}

// WithOSSTracerProvider 显式指定 TracerProvider（不指定则用全局）。
func WithOSSTracerProvider(tp trace.TracerProvider) OSSTraceOption {
	return func(o *ossTraceOptions) { o.tp = tp }
}

// tracedOSS 是 OSSWithBucket 的追踪装饰器：
// 为每次对象/桶操作产生一个 client span（含操作名与对象键）并记录耗时指标，
// 从而让 pkg/oss 的所有实现（MinIO / 阿里云）统一获得 data 层链路，无需改业务代码。
type tracedOSS struct {
	inner OSSWithBucket
	opts  ossTraceOptions
}

// NewTracedOSS 用追踪能力包装任意 OSSWithBucket 实现。
func NewTracedOSS(inner OSSWithBucket, opts ...OSSTraceOption) OSSWithBucket {
	o := ossTraceOptions{}
	for _, opt := range opts {
		opt(&o)
	}
	return &tracedOSS{inner: inner, opts: o}
}

// tracer 返回装饰器使用的 tracer（显式注入优先）。
func (t *tracedOSS) tracer() trace.Tracer {
	if t.opts.tp != nil {
		return t.opts.tp.Tracer(tracerName)
	}
	return otel.GetTracerProvider().Tracer(tracerName)
}

// begin 启动 span 并返回结束回调；key 非空时记录对象键。
func (t *tracedOSS) begin(ctx context.Context, op, key string) (context.Context, func(error)) {
	if ctx == nil {
		ctx = context.Background()
	}
	attrs := []attribute.KeyValue{attribute.String("oss.operation", op)}
	if key != "" {
		attrs = append(attrs, attribute.String("object.key", key))
	}
	ctx, span := t.tracer().Start(ctx, "oss."+op,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attrs...),
	)
	start := time.Now()
	return ctx, func(err error) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
		recordOSSOperation(ctx, op, err, time.Since(start))
	}
}

// ossMetricsOnce 保证指标仪器仅创建一次。
var (
	ossMetricsOnce sync.Once
	ossDuration    metric.Float64Histogram
)

// recordOSSOperation 记录对象存储操作耗时直方图（metrics 未初始化时为空操作）。
func recordOSSOperation(ctx context.Context, op string, err error, elapsed time.Duration) {
	ossMetricsOnce.Do(func() {
		ossDuration, _ = otel.GetMeterProvider().Meter("ley/oss").Float64Histogram(
			"oss.client.operation.duration",
			metric.WithUnit("s"),
			metric.WithDescription("对象存储操作耗时"),
		)
	})
	if ossDuration == nil {
		return
	}
	result := "success"
	if err != nil {
		result = "error"
	}
	ossDuration.Record(ctx, elapsed.Seconds(),
		metric.WithAttributes(
			attribute.String("oss.operation", op),
			attribute.String("result", result),
		),
	)
}

// ===== OSS 接口方法（逐一插桩后转发给 inner） =====

// PutObject 上传对象。
func (t *tracedOSS) PutObject(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (err error) {
	ctx, done := t.begin(ctx, "PutObject", key)
	defer func() { done(err) }()
	return t.inner.PutObject(ctx, key, reader, size, contentType)
}

// GetObject 获取对象（完整内容）。
func (t *tracedOSS) GetObject(ctx context.Context, key string) (rc io.ReadCloser, info *ObjectInfo, err error) {
	ctx, done := t.begin(ctx, "GetObject", key)
	defer func() { done(err) }()
	return t.inner.GetObject(ctx, key)
}

// GetObjectRange 范围读取对象。
func (t *tracedOSS) GetObjectRange(ctx context.Context, key string, offset, length int64) (rc io.ReadCloser, info *ObjectInfo, err error) {
	ctx, done := t.begin(ctx, "GetObjectRange", key)
	defer func() { done(err) }()
	return t.inner.GetObjectRange(ctx, key, offset, length)
}

// StatObject 获取对象元信息。
func (t *tracedOSS) StatObject(ctx context.Context, key string) (info *ObjectInfo, err error) {
	ctx, done := t.begin(ctx, "StatObject", key)
	defer func() { done(err) }()
	return t.inner.StatObject(ctx, key)
}

// DeleteObject 删除单个对象。
func (t *tracedOSS) DeleteObject(ctx context.Context, key string) (err error) {
	ctx, done := t.begin(ctx, "DeleteObject", key)
	defer func() { done(err) }()
	return t.inner.DeleteObject(ctx, key)
}

// DeleteObjects 批量删除对象。
func (t *tracedOSS) DeleteObjects(ctx context.Context, keys []string) (err error) {
	ctx, done := t.begin(ctx, "DeleteObjects", "")
	defer func() { done(err) }()
	return t.inner.DeleteObjects(ctx, keys)
}

// CopyObject 复制对象。
func (t *tracedOSS) CopyObject(ctx context.Context, sourceKey, destKey string) (err error) {
	ctx, done := t.begin(ctx, "CopyObject", sourceKey)
	defer func() { done(err) }()
	return t.inner.CopyObject(ctx, sourceKey, destKey)
}

// ListObjects 分页列举对象。
func (t *tracedOSS) ListObjects(ctx context.Context, prefix string, limit int, token string) (objects []ObjectInfo, next string, err error) {
	ctx, done := t.begin(ctx, "ListObjects", prefix)
	defer func() { done(err) }()
	return t.inner.ListObjects(ctx, prefix, limit, token)
}

// GetPresignedURL 获取预签名 GET URL。
func (t *tracedOSS) GetPresignedURL(ctx context.Context, key string, expirySeconds int64) (url string, err error) {
	ctx, done := t.begin(ctx, "GetPresignedURL", key)
	defer func() { done(err) }()
	return t.inner.GetPresignedURL(ctx, key, expirySeconds)
}

// GetPresignedPutURL 获取预签名 PUT URL。
func (t *tracedOSS) GetPresignedPutURL(ctx context.Context, key string, contentType string, expirySeconds int64) (url string, err error) {
	ctx, done := t.begin(ctx, "GetPresignedPutURL", key)
	defer func() { done(err) }()
	return t.inner.GetPresignedPutURL(ctx, key, contentType, expirySeconds)
}

// InitiateMultipartUpload 初始化分片上传。
func (t *tracedOSS) InitiateMultipartUpload(ctx context.Context, key, contentType string) (uploadID string, err error) {
	ctx, done := t.begin(ctx, "InitiateMultipartUpload", key)
	defer func() { done(err) }()
	return t.inner.InitiateMultipartUpload(ctx, key, contentType)
}

// UploadPart 上传单个分片。
func (t *tracedOSS) UploadPart(ctx context.Context, key, uploadID string, partNumber int32, reader io.Reader, size int64) (part Part, err error) {
	ctx, done := t.begin(ctx, "UploadPart", key)
	defer func() { done(err) }()
	return t.inner.UploadPart(ctx, key, uploadID, partNumber, reader, size)
}

// CompleteMultipartUpload 完成分片上传。
func (t *tracedOSS) CompleteMultipartUpload(ctx context.Context, key, uploadID string, parts []Part) (err error) {
	ctx, done := t.begin(ctx, "CompleteMultipartUpload", key)
	defer func() { done(err) }()
	return t.inner.CompleteMultipartUpload(ctx, key, uploadID, parts)
}

// AbortMultipartUpload 取消分片上传。
func (t *tracedOSS) AbortMultipartUpload(ctx context.Context, key, uploadID string) (err error) {
	ctx, done := t.begin(ctx, "AbortMultipartUpload", key)
	defer func() { done(err) }()
	return t.inner.AbortMultipartUpload(ctx, key, uploadID)
}

// BucketExists 检查桶是否存在。
func (t *tracedOSS) BucketExists(ctx context.Context, bucket string) (exists bool, err error) {
	ctx, done := t.begin(ctx, "BucketExists", "")
	defer func() { done(err) }()
	return t.inner.BucketExists(ctx, bucket)
}

// MakeBucket 创建桶。
func (t *tracedOSS) MakeBucket(ctx context.Context, bucket string) (err error) {
	ctx, done := t.begin(ctx, "MakeBucket", "")
	defer func() { done(err) }()
	return t.inner.MakeBucket(ctx, bucket)
}

// RemoveBucket 删除桶。
func (t *tracedOSS) RemoveBucket(ctx context.Context, bucket string) (err error) {
	ctx, done := t.begin(ctx, "RemoveBucket", "")
	defer func() { done(err) }()
	return t.inner.RemoveBucket(ctx, bucket)
}

// ListBuckets 列出所有桶。
func (t *tracedOSS) ListBuckets(ctx context.Context) (buckets []BucketInfo, err error) {
	ctx, done := t.begin(ctx, "ListBuckets", "")
	defer func() { done(err) }()
	return t.inner.ListBuckets(ctx)
}
