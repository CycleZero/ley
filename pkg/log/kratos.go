package log

import (
	"fmt"

	klog "github.com/go-kratos/kratos/v2/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// correlationKeys 关联字段键集合：值为空时丢弃，避免无链路/无用户上下文时输出空字段。
var correlationKeys = map[string]struct{}{
	FieldTraceID: {},
	FieldSpanID:  {},
	FieldUserID:  {},
	FieldRole:    {},
}

// kratosLogger 将 zap 适配为 Kratos log.Logger。
//
// 相比官方 kratos/contrib/log/zap，额外做了两件事：
//  1. 丢弃空值的关联字段（trace_id/span_id/user_id/role），保持日志简洁；
//  2. 按 Kratos 语义显式映射级别，避免 Fatal 被误映射为 zap 的 DPanic。
type kratosLogger struct {
	log    *zap.Logger
	msgKey string
}

// newKratosLogger 构造 Kratos 日志适配器。
func newKratosLogger(zl *zap.Logger) *kratosLogger {
	return &kratosLogger{log: zl, msgKey: klog.DefaultMessageKey}
}

// Log 实现 Kratos log.Logger：把 keyvals 键值对转换为 zap 结构化字段。
func (l *kratosLogger) Log(level klog.Level, keyvals ...any) error {
	if !l.log.Core().Enabled(toZapLevel(level)) {
		return nil
	}
	if len(keyvals) == 0 || len(keyvals)%2 != 0 {
		l.log.Warn(fmt.Sprint("Kratos 日志键值必须成对出现：", keyvals))
		return nil
	}

	msg := ""
	fields := make([]zap.Field, 0, len(keyvals)/2)
	for i := 0; i < len(keyvals); i += 2 {
		key, _ := keyvals[i].(string)
		val := keyvals[i+1]
		if key == l.msgKey {
			if s, ok := val.(string); ok {
				msg = s
			}
			continue
		}
		if _, ok := correlationKeys[key]; ok && isEmptyValue(val) {
			continue
		}
		fields = append(fields, zap.Any(key, val))
	}

	switch level {
	case klog.LevelDebug:
		l.log.Debug(msg, fields...)
	case klog.LevelInfo:
		l.log.Info(msg, fields...)
	case klog.LevelWarn:
		l.log.Warn(msg, fields...)
	case klog.LevelError:
		l.log.Error(msg, fields...)
	case klog.LevelFatal:
		l.log.Fatal(msg, fields...)
	default:
		l.log.Info(msg, fields...)
	}
	return nil
}

// toZapLevel 将 Kratos 日志级别映射为 zap 级别（显式映射，避免枚举值巧合错位）。
func toZapLevel(level klog.Level) zapcore.Level {
	switch level {
	case klog.LevelDebug:
		return zapcore.DebugLevel
	case klog.LevelWarn:
		return zapcore.WarnLevel
	case klog.LevelError:
		return zapcore.ErrorLevel
	case klog.LevelFatal:
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// isEmptyValue 判断关联字段值是否为空（空串/nil 视为空）。
func isEmptyValue(val any) bool {
	if val == nil {
		return true
	}
	if s, ok := val.(string); ok {
		return s == ""
	}
	return false
}

// GetKratosLogger 返回带关联字段自动注入的 Kratos 日志器。
// 通过 log.Valuer 机制，Kratos log.Helper 在 WithContext(ctx) 后会自动带上
// trace_id/span_id/user_id/role，无需每个调用点手工取字段。
func GetKratosLogger() klog.Logger {
	return klog.With(newKratosLogger(GetLogger().Logger),
		FieldTraceID, traceIDValuer(),
		FieldSpanID, spanIDValuer(),
		FieldUserID, userIDValuer(),
		FieldRole, roleValuer(),
	)
}

// GetKratosLogHelper 返回 Kratos 日志 Helper（中文消息用 Sprint 渲染）。
func GetKratosLogHelper() *klog.Helper {
	return klog.NewHelper(GetKratosLogger(), klog.WithSprint(Sprint))
}
