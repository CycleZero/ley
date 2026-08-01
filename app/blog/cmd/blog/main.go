package main

import (
	"flag"

	"github.com/CycleZero/ley/app/blog/internal/conf"
	commonconf "github.com/CycleZero/ley/conf"
	"github.com/CycleZero/ley/pkg/infra"
	locallog "github.com/CycleZero/ley/pkg/log"
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
		kratos.ID(id), 		kratos.Name(util.DisServiceName(Name)), kratos.Version(Version),
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
	l, err := locallog.NewLogger(logMode, logLevel, logPath, Name)
	if err != nil {
		panic(err)
	}
	locallog.SetGlobalLogger(l)
	logger := log.With(locallog.GetKratosLogger(),
		"ts", log.DefaultTimestamp, "caller", log.DefaultCaller,
		"service.id", id, "service.name", Name, "service.version", Version,
		"trace.id", tracing.TraceID(), "span.id", tracing.SpanID(),
	)

	if bc.Etcd == nil || len(bc.Etcd.Endpoints) == 0 {
		panic("config error: etcd.endpoints is required in bootstrap config")
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

	if bc.Trace != nil && bc.Trace.Endpoint != "" {
		_ = trace.InitTracer(bc.Trace.Endpoint, util.DisServiceName(conf.ServiceName))
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
