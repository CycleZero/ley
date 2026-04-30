package main

import (
	"flag"
	"fmt"
	"ley/app/user/internal/conf"
	commonconf "ley/conf"
	"ley/pkg/infra"
	locallog "ley/pkg/log"
	"ley/pkg/trace"
	"ley/pkg/util"

	"github.com/go-kratos/kratos/contrib/config/etcd/v2"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
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
}

func newApp(
	logger log.Logger,
	gs *grpc.Server,
	hs *http.Server,
	rr registry.Registrar,
) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
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
	l, err := locallog.NewLogger(int(bc.Log.Mode.Number()), bc.Log.Level.String(), bc.Log.Path, Name)
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

	// 初始化Tracer
	err = trace.InitTracer(bc.Trace.Endpoint, conf.ServiceName)
	if err != nil {
		logger.Log(log.LevelError, "init tracer error", err)
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
