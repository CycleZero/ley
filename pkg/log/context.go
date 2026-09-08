package log

import (
	"context"

	"github.com/CycleZero/ley/pkg/meta"
	klog "github.com/go-kratos/kratos/v2/log"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// 结构化日志关联字段键名：采集侧据此把日志与链路、用户关联起来。
const (
	FieldTraceID = "trace_id" // 链路追踪 ID（OTel）
	FieldSpanID  = "span_id"  // 当前 span ID（OTel）
	FieldUserID  = "user_id"  // 认证用户 ID（来自 pkg/meta）
	FieldRole    = "role"     // 用户角色（reader/author/admin）
)

// Ctx 返回携带请求关联字段的日志器：
//   - trace_id / span_id 来自上下文中的 OTel span；
//   - user_id / role 来自 pkg/meta 请求元数据。
//
// 用法：在持有 ctx 的业务代码中使用 log.Ctx(ctx).Info("...")，
// 使日志自动与链路、用户关联。全局日志器未初始化（单测/启动早期）时降级为 Nop，
// 避免 panic；无关联信息时返回全局日志器本身（零额外开销）。
func Ctx(ctx context.Context) *Logger {
	l := globalLogger
	if l == nil || l.Logger == nil {
		l = &Logger{Logger: zap.NewNop()}
	}
	if ctx == nil {
		return l
	}
	return l.WithContext(ctx)
}

// WithContext 返回注入了关联字段的派生日志器。
// 仅在字段值有效时注入，避免无链路上下文时出现 trace_id="" 之类空值噪声。
func (l *Logger) WithContext(ctx context.Context) *Logger {
	if l == nil || l.Logger == nil || ctx == nil {
		return l
	}
	fields := correlationFields(ctx)
	if len(fields) == 0 {
		return l
	}
	return &Logger{Logger: l.Logger.With(fields...)}
}

// correlationFields 从上下文提取关联字段（仅在值有效时生成）。
func correlationFields(ctx context.Context) []zap.Field {
	fields := make([]zap.Field, 0, 4)
	if sc := trace.SpanContextFromContext(ctx); sc.HasTraceID() {
		fields = append(fields, zap.String(FieldTraceID, sc.TraceID().String()))
		if sc.HasSpanID() {
			fields = append(fields, zap.String(FieldSpanID, sc.SpanID().String()))
		}
	}
	if md := meta.GetRequestMetaData(ctx); md != nil {
		if md.Auth.UserID > 0 {
			fields = append(fields, zap.Uint64(FieldUserID, md.Auth.UserID))
		}
		if md.Auth.Role != "" {
			fields = append(fields, zap.String(FieldRole, md.Auth.Role))
		}
	}
	return fields
}

// ===================== Kratos log.Valuer（供 Kratos log.Helper 自动注入关联字段） =====================

// traceIDValuer 从 Kratos 传入的 ctx 中解析 trace_id；无 span 时返回空串。
func traceIDValuer() klog.Valuer {
	return func(ctx context.Context) any {
		if sc := trace.SpanContextFromContext(ctx); sc.HasTraceID() {
			return sc.TraceID().String()
		}
		return ""
	}
}

// spanIDValuer 从 Kratos 传入的 ctx 中解析 span_id；无 span 时返回空串。
func spanIDValuer() klog.Valuer {
	return func(ctx context.Context) any {
		if sc := trace.SpanContextFromContext(ctx); sc.HasSpanID() {
			return sc.SpanID().String()
		}
		return ""
	}
}

// userIDValuer 从 Kratos 传入的 ctx 中解析认证用户 ID；无用户时返回空串。
func userIDValuer() klog.Valuer {
	return func(ctx context.Context) any {
		if md := meta.GetRequestMetaData(ctx); md != nil && md.Auth.UserID > 0 {
			return md.Auth.UserID
		}
		return ""
	}
}

// roleValuer 从 Kratos 传入的 ctx 中解析用户角色；无角色时返回空串。
func roleValuer() klog.Valuer {
	return func(ctx context.Context) any {
		if md := meta.GetRequestMetaData(ctx); md != nil && md.Auth.Role != "" {
			return md.Auth.Role
		}
		return ""
	}
}
