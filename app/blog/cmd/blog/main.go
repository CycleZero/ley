package main

import (
	"context"
	"flag"

	"github.com/CycleZero/ley/app/blog/internal/conf"
	commonconf "github.com/CycleZero/ley/conf"
	"github.com/CycleZero/ley/pkg/infra"
	locallog "github.com/CycleZero/ley/pkg/log"
	"github.com/CycleZero/ley/pkg/metrics"
	"github.com/CycleZero/ley/pkg/trace"
	"github.com/CycleZero/ley/pkg/util"

	"github.com/go-kratos/kratos/contrib/config/etcd/v2"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	kratosjson "github.com/go-kratos/kratos/v2/encoding/json"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
)

var (
	Name     string = conf.ServiceName
	Version  string = "v0.0.1"
	flagconf string = conf.LocalConfigDir
	id              = util.ServiceId(Name)
)

func init() {
	flag.StringVar(&flagconf, "conf", conf.LocalConfigDir, "config path, eg: -conf config.yaml")

	// JSON 输出使用 proto 字段原名（snake_case），与前端 API 契约对齐
	kratosjson.MarshalOptions.UseProtoNames = true
}

func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server, rr registry.Registrar) *kratos.App {
	return kratos.New(
		kratos.ID(id), kratos.Name(util.DisServiceName(Name)), kratos.Version(Version),
		kratos.Metadata(map[string]string{}), kratos.Logger(logger),
		kratos.Server(gs, hs), kratos.Registrar(rr),
	)
}

func main() {
	flag.Parse()
	c := config.New(config.WithSource(file.NewSource(flagconf)))
	if err := c.Load(); err != nil {
		panic(err)
	}
	var bc commonconf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}
	c.Close()

	logMode := int(commonconf.LogMode_Dev.Number())
	logLevel := commonconf.LogLevel_Debug.String()
	logPath := "./data/logs"
	if bc.Log != nil {
		logMode = int(bc.Log.Mode.Number())
		logLevel = bc.Log.Level.String()
		logPath = bc.Log.Path
	}
	// OTLP 端点统一取自引导配置 trace.endpoint（trace/metrics/logs 共用）
	traceEndpoint := ""
	if bc.Trace != nil {
		traceEndpoint = bc.Trace.Endpoint
	}
	l, err := locallog.NewLogger(logMode, logLevel, logPath, Name, locallog.WithOTLP(util.DisServiceName(conf.ServiceName), traceEndpoint))
	if err != nil {
		panic(err)
	}
	locallog.SetGlobalLogger(l)
	defer func() { _ = locallog.ShutdownOTLP(context.Background()) }()
	logger := log.With(locallog.GetKratosLogger(),
		"ts", log.DefaultTimestamp, "caller", log.DefaultCaller,
		"service.id", id, "service.name", Name, "service.version", Version,
		"trace.id", tracing.TraceID(), "span.id", tracing.SpanID(),
	)

	if bc.Etcd == nil || len(bc.Etcd.Endpoints) == 0 {
		panic("config error: etcd.endpoints is required in bootstrap config")
	}
	// 初始化可观测指标（Prometheus /metrics + OTLP 推送；与追踪共用端点）
	metricsProvider, metricsErr := metrics.New(util.DisServiceName(conf.ServiceName), metrics.WithOTLPEndpoint(traceEndpoint))
	if metricsErr != nil {
		logger.Log(log.LevelError, "init metrics error", metricsErr)
	} else {
		defer func() { _ = metricsProvider.Shutdown(context.Background()) }()
	}
	etcdClient := infra.NewEtcdClient(bc.Etcd.Endpoints)
	defer etcdClient.Close()

	var serviceConf conf.Config
	remoteCfg, err := etcd.New(etcdClient, etcd.WithPath(conf.RemoteConfigPath))
	if err != nil {
		panic(err)
	}
	finalCfg := config.New(config.WithSource(remoteCfg))
	defer finalCfg.Close()
	if err = finalCfg.Load(); err != nil {
		panic(err)
	}
	if err = finalCfg.Scan(&serviceConf); err != nil {
		panic(err)
	}

	// 初始化链路追踪（OTLP/HTTP + W3C 上下文传播；退出时刷出缓冲 span）
	if bc.Trace != nil && bc.Trace.Endpoint != "" {
		if _, err = trace.Init(trace.Config{
			Endpoint:    bc.Trace.Endpoint,
			ServiceName: util.DisServiceName(conf.ServiceName),
			Insecure:    true,
		}); err != nil {
			logger.Log(log.LevelError, "init tracer error", err)
		}
		defer func() { _ = trace.Shutdown(context.Background()) }()
	}

	app, cleanup, err := wireApp(&bc, &serviceConf, bc.Server, serviceConf.Data, logger, etcdClient)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	if err := app.Run(); err != nil {
		panic(err)
	}
}
