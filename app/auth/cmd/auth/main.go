package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/CycleZero/ley/app/auth/internal/conf"
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
	//_ "go.uber.org/automaxprocs"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name string = conf.ServiceName
	// Version is the version of the compiled software.
	Version string = "v0.0.1"
	// flagconf is the config flag.
	flagconf string = conf.LocalConfigDir

	id = util.ServiceId(Name)
)

func init() {
	flag.StringVar(&flagconf, "conf", conf.LocalConfigDir, "config path, eg: -conf config.yaml")

	// JSON 输出使用 proto 字段原名（snake_case），与前端 API 契约对齐
	kratosjson.MarshalOptions.UseProtoNames = true
}

func newApp(
	logger log.Logger,
	gs *grpc.Server,
	hs *http.Server,
	rr registry.Registrar,
) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(util.DisServiceName(Name)),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			gs,
			hs,
		),
		kratos.Registrar(rr),
	)
}

func main() {
	flag.Parse()
	configFile := flagconf
	//fi, err := os.Stat(configFile)
	//if err != nil {
	//	panic(err)
	//}

	var sources []config.Source
	//if fi.IsDir() {
	//	sources = append(sources, file.NewSource(configFile+"/"+Name))
	//}
	sources = append(sources, file.NewSource(configFile))
	c := config.New(
		config.WithSource(
			sources...,
		),
	)

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc commonconf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}
	c.Close()
	fmt.Println("配置项", bc.String())
	//fmt.Println()
	logMode := int(commonconf.LogMode_Dev.Number())
	logLevel := commonconf.LogLevel_Debug.String()
	logPath := "./data/logs"
	if bc.Log != nil {
		logMode = int(bc.Log.Mode.Number())
		logLevel = bc.Log.Level.String()
		logPath = bc.Log.Path
	}
	l, err := locallog.NewLogger(logMode, logLevel, logPath, Name)
	if err != nil {
		panic(err)
	}
	locallog.SetGlobalLogger(l)
	logger := log.With(locallog.GetKratosLogger(),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", id,
		"service.name", Name,
		"service.version", Version,
		"trace.id", tracing.TraceID(),
		"span.id", tracing.SpanID(),
	)
	if bc.Etcd == nil || len(bc.Etcd.Endpoints) == 0 {
		panic("config error: etcd.endpoints is required in bootstrap config")
	}
	// 初始化可观测指标（Prometheus /metrics，OTel 兼容；与追踪共用 MeterProvider）
	if _, err := metrics.New(util.DisServiceName(conf.ServiceName)); err != nil {
		logger.Log(log.LevelError, "init metrics error", err)
	}
	etcdClient := infra.NewEtcdClient(bc.Etcd.Endpoints)
	defer etcdClient.Close()
	logger.Log(log.LevelInfo, "init logger success")
	//res, err := etcdClient.Get(context.Background(), "config/rhea/common.yaml")
	//if err != nil {
	//	logger.Log(log.LevelError, "init etcd client error", zap.Error(err))
	//	panic(err)
	//}
	//fmt.Println("etcd 配置项", res.Kvs)
	//fmt.Println("etcd 配置项长度", len(res.Kvs), " ", res.Count)
	//for _, kv := range res.Kvs {
	//	fmt.Println(string(kv.Key), string(kv.Value))
	//}

	var serviceConf conf.Config
	remoteCfg, err := etcd.New(
		etcdClient,
		etcd.WithPath(conf.RemoteConfigPath), // etcd 里的配置 key
	)
	if err != nil {
		panic(err)
	}
	//fmt.Println("etcd service配置项", conf.RemoteConfigPath)

	//sources = append(sources, remoteCommonCfg, remoteServiceConfig)
	finalCfg := config.New(
		config.WithSource(remoteCfg),
		//config.WithSource(remoteCfg),
	)

	defer finalCfg.Close()
	err = finalCfg.Load()
	if err != nil {
		panic(err)
	}
	err = finalCfg.Scan(&serviceConf)
	if err != nil {
		panic(err)
	}

	// 初始化Tracer（OTLP/HTTP + W3C 上下文传播；退出时刷出缓冲 span）
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
	//logger.Log(log.LevelInfo, "最终配置内容", bc.String())
	app, cleanup, err := wireApp(
		&bc,
		&serviceConf,
		bc.Server,
		serviceConf.Data,
		logger,
		etcdClient,
	)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	logger.Log(log.LevelInfo, "init wireApp success")
	// start and wait for stop signal
	if err := app.Run(); err != nil {
		panic(err)
	}
}
