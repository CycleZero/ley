package biz

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// recordResult 记录一次业务操作结果（result=success/error），标签基数有界。
func recordResult(ctx context.Context, counter metric.Int64Counter, result string) {
	if counter == nil {
		return
	}
	counter.Add(ctx, 1, metric.WithAttributes(attribute.String("result", result)))
}

// resultOf 将错误映射为指标标签值。
func resultOf(err error) string {
	if err != nil {
		return "error"
	}
	return "success"
}
