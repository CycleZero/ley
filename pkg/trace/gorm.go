package trace

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

// gormSpanKey 是在 GORM Statement.Settings 中暂存 span 状态的键：
// before 回调写入，after 回调取出并结束，从而让一次查询对应一个完整 span。
const gormSpanKey = "ley:otel:gorm:span"

// gormSpanState 保存一次数据库操作的 span 与起始时间（用于统计耗时）。
type gormSpanState struct {
	span  trace.Span
	start time.Time
}

// GormOption 配置 GORM 插桩。
type GormOption func(*gormOptions)

type gormOptions struct {
	tp trace.TracerProvider
}

// WithGormTracerProvider 显式指定 TracerProvider（不指定则用全局）。
func WithGormTracerProvider(tp trace.TracerProvider) GormOption {
	return func(o *gormOptions) { o.tp = tp }
}

// GormPlugin 是 GORM 的 OTel 插桩插件：
//   - 为 create/query/update/delete 四类操作各产生一个 client span；
//   - span 属性遵循 OTel 数据库语义约定（db.system/db.operation.name/db.collection.name）；
//   - 失败时记录错误并置 span 状态为 Error；
//   - 同时记录 db.client.operation.duration 指标（若已初始化 metrics Provider）。
//
// 使用：db.Use(NewGormPlugin())，务必以 db.WithContext(ctx) 执行查询，
// 否则 span 无法挂到请求链路上。
type GormPlugin struct {
	opts gormOptions
}

// NewGormPlugin 创建 GORM 插桩插件。
func NewGormPlugin(opts ...GormOption) *GormPlugin {
	o := gormOptions{}
	for _, opt := range opts {
		opt(&o)
	}
	return &GormPlugin{opts: o}
}

// Name 返回插件名（GORM 要求唯一）。
func (p *GormPlugin) Name() string { return "ley:otel:gorm" }

// Initialize 注册四类操作的 before/after 回调。
func (p *GormPlugin) Initialize(db *gorm.DB) error {
	cb := db.Callback()
	steps := []struct {
		op     string
		before func(string) error
		after  func(string) error
	}{
		{"create",
			func(name string) error { return cb.Create().Before("gorm:create").Register(name, p.before("create")) },
			func(name string) error { return cb.Create().After("gorm:create").Register(name, p.after()) }},
		{"query",
			func(name string) error { return cb.Query().Before("gorm:query").Register(name, p.before("query")) },
			func(name string) error { return cb.Query().After("gorm:query").Register(name, p.after()) }},
		{"update",
			func(name string) error { return cb.Update().Before("gorm:update").Register(name, p.before("update")) },
			func(name string) error { return cb.Update().After("gorm:update").Register(name, p.after()) }},
		{"delete",
			func(name string) error { return cb.Delete().Before("gorm:delete").Register(name, p.before("delete")) },
			func(name string) error { return cb.Delete().After("gorm:delete").Register(name, p.after()) }},
	}
	for _, s := range steps {
		if err := s.before("ley:otel:before:" + s.op); err != nil {
			return err
		}
		if err := s.after("ley:otel:after:" + s.op); err != nil {
			return err
		}
	}
	return nil
}

// tracer 返回插件使用的 tracer（显式注入优先）。
func (p *GormPlugin) tracer() trace.Tracer {
	if p.opts.tp != nil {
		return p.opts.tp.Tracer(tracerName)
	}
	return otel.GetTracerProvider().Tracer(tracerName)
}

// before 在 SQL 执行前启动 span，并写入当前语句上下文供 after 取出。
func (p *GormPlugin) before(op string) func(*gorm.DB) {
	return func(db *gorm.DB) {
		if db.Error != nil || db.Statement == nil {
			return
		}
		ctx := db.Statement.Context
		if ctx == nil {
			ctx = context.Background()
		}

		table := db.Statement.Table
		spanName := "db." + op
		attrs := make([]attribute.KeyValue, 0, 3)
		attrs = append(attrs,
			attribute.String("db.system", db.Dialector.Name()),
			attribute.String("db.operation.name", op),
		)
		if table != "" {
			spanName += " " + table
			attrs = append(attrs, attribute.String("db.collection.name", table))
		}

		ctx, span := p.tracer().Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(attrs...),
		)
		db.Statement.Context = ctx
		db.Statement.Settings.Store(gormSpanKey, &gormSpanState{span: span, start: time.Now()})
	}
}

// after 结束 span：记录错误并统计耗时指标。
func (p *GormPlugin) after() func(*gorm.DB) {
	return func(db *gorm.DB) {
		if db.Statement == nil {
			return
		}
		value, ok := db.Statement.Settings.LoadAndDelete(gormSpanKey)
		if !ok {
			return
		}
		state, ok := value.(*gormSpanState)
		if !ok {
			return
		}

		if db.Error != nil {
			state.span.RecordError(db.Error)
			state.span.SetStatus(codes.Error, db.Error.Error())
		}
		state.span.End()
		recordDBOperation(db.Statement.Context, db.Dialector.Name(), db.Statement.Table, db.Error, time.Since(state.start))
	}
}

// dbMetricsOnce 保证指标仪器仅创建一次（基于全局 MeterProvider）。
var (
	dbMetricsOnce sync.Once
	dbDuration    metric.Float64Histogram
)

// recordDBOperation 记录数据库操作耗时直方图（metrics 未初始化时自动降级为空操作）。
func recordDBOperation(ctx context.Context, dbSystem, operation string, err error, elapsed time.Duration) {
	dbMetricsOnce.Do(func() {
		dbDuration, _ = otel.GetMeterProvider().Meter("ley/db").Float64Histogram(
			"db.client.operation.duration",
			metric.WithUnit("s"),
			metric.WithDescription("数据库操作耗时"),
		)
	})
	if dbDuration == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	dbDuration.Record(ctx, elapsed.Seconds(),
		metric.WithAttributes(
			attribute.String("db.system", dbSystem),
			attribute.String("db.operation.name", operation),
			attribute.String("result", resultLabel(err)),
		),
	)
}

// resultLabel 将错误转为有界的指标标签值。
func resultLabel(err error) string {
	if err != nil {
		return "error"
	}
	return "success"
}
