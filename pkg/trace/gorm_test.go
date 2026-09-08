package trace

import (
	"context"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// traceWidget 仅用于 GORM 干跑（DryRun）测试的临时模型。
type traceWidget struct {
	ID   uint `gorm:"primaryKey"`
	Name string
}

// TableName 固定表名，便于断言 span 属性中的集合名。
func (traceWidget) TableName() string { return "trace_widgets" }

// newTestDB 构造不连库的 GORM 实例（DisableAutomaticPing + DryRun），
// 使回调链真实执行但不产生网络 I/O，用于单元测试 data 层插桩。
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(
		mysql.New(mysql.Config{
			DSN:                       "user:pass@tcp(127.0.0.1:3306)/ley",
			SkipInitializeWithVersion: true,
		}),
		&gorm.Config{
			DisableAutomaticPing: true,
			Logger:               logger.Discard,
		},
	)
	if err != nil {
		t.Fatalf("打开 GORM 干跑实例失败：%v", err)
	}
	return db
}

// 场景 S4（data 层 trace）：GORM 查询必须产生一个 data 层 span，
// 且带 db.system / db.operation.name / db.collection.name 语义属性。
func TestGormPluginEmitsQuerySpan(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tp := tracesdk.NewTracerProvider(tracesdk.WithSampler(tracesdk.AlwaysSample()), tracesdk.WithSyncer(exp))

	db := newTestDB(t)
	if err := db.Use(NewGormPlugin(WithGormTracerProvider(tp))); err != nil {
		t.Fatalf("注册 GORM trace 插件失败：%v", err)
	}

	var widgets []traceWidget
	db.WithContext(context.Background()).
		Session(&gorm.Session{DryRun: true}).
		Find(&widgets)

	spans := exp.GetSpans()
	got := findSpan(spans, "trace_widgets")
	if got == nil {
		t.Fatalf("未产生包含 trace_widgets 的 span，实际 span：%v", spanNames(spans))
	}
	attrs := attrMap(got.Attributes)
	if attrs["db.system"] != "mysql" {
		t.Errorf("db.system 应为 mysql，实际 %q", attrs["db.system"])
	}
	if attrs["db.operation.name"] != "query" {
		t.Errorf("db.operation.name 应为 query，实际 %q", attrs["db.operation.name"])
	}
	if attrs["db.collection.name"] != "trace_widgets" {
		t.Errorf("db.collection.name 应为 trace_widgets，实际 %q", attrs["db.collection.name"])
	}
}

// findSpan 返回名称包含 name 的第一个 span。
func findSpan(spans tracetest.SpanStubs, name string) *tracetest.SpanStub {
	for i := range spans {
		if strings.Contains(spans[i].Name, name) {
			return &spans[i]
		}
	}
	return nil
}

// spanNames 提取 span 名称列表，用于失败断言输出。
func spanNames(spans tracetest.SpanStubs) []string {
	names := make([]string, 0, len(spans))
	for i := range spans {
		names = append(names, spans[i].Name)
	}
	return names
}

// attrMap 将属性切片转为 map，便于按键断言。
func attrMap(attrs []attribute.KeyValue) map[string]string {
	m := make(map[string]string, len(attrs))
	for _, kv := range attrs {
		m[string(kv.Key)] = kv.Value.Emit()
	}
	return m
}
