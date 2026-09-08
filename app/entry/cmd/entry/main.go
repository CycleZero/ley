package main

import (
	"context"
	"flag"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/CycleZero/ley/app/entry/conf"
	commonconf "github.com/CycleZero/ley/conf"
	locallog "github.com/CycleZero/ley/pkg/log"
	"github.com/CycleZero/ley/pkg/metrics"
	"github.com/CycleZero/ley/pkg/trace"
	"github.com/CycleZero/ley/pkg/util"
	"go.uber.org/zap"
)

// Name 服务名：日志、目录、链路追踪服务名均基于它
var Name = conf.ServiceName

// flagconf -conf flag 的值：本地引导配置目录
var flagconf = conf.LocalConfigDir

func init() {
	flag.StringVar(&flagconf, "conf", conf.LocalConfigDir, "config path, eg: -conf ./data/entry/configs")

	// TODO(后续 wave)：entry 自身响应体序列化用 protojson 的策略在此落位，
	// 不需要像 auth 那样全局开启 kratosjson.MarshalOptions.UseProtoNames
}

func main() {
	flag.Parse()

	// 1. 加载本地引导配置（conf/common.proto Bootstrap）
	bc, err := conf.LoadBootstrap(flagconf)
	if err != nil {
		panic(err)
	}
	fmt.Println("引导配置", bc.String())

	// 2. 初始化日志（pkg/log 全局单例，供全服务复用）
	logMode := int(commonconf.LogMode_Dev.Number())
	logLevel := commonconf.LogLevel_Debug.String()
	logPath := conf.ServiceDataDir + "/logs"
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
	l, err := locallog.NewLogger(logMode, logLevel, logPath, Name, locallog.WithOTLP(util.DisServiceName(Name), traceEndpoint))
	if err != nil {
		panic(fmt.Sprintf("初始化日志失败：%v", err))
	}
	locallog.SetGlobalLogger(l)
	defer func() { _ = locallog.ShutdownOTLP(context.Background()) }()
	logger := locallog.GetLogger()
	logger.Info("日志初始化成功", zap.String("service", Name))

	// 3. 初始化可观测指标（Prometheus /metrics + OTLP 推送；与追踪共用端点）
	metricsProvider, metricsErr := metrics.New(util.DisServiceName(Name), metrics.WithOTLPEndpoint(traceEndpoint))
	if metricsErr != nil {
		logger.Error("初始化指标失败", zap.Error(metricsErr))
	} else {
		defer func() { _ = metricsProvider.Shutdown(context.Background()) }()
	}

	// 4. 初始化链路追踪（OTLP/HTTP + W3C 传播；退出时刷出缓冲 span）
	if bc.Trace != nil && bc.Trace.Endpoint != "" {
		if _, err := trace.Init(trace.Config{
			Endpoint:    bc.Trace.Endpoint,
			ServiceName: util.DisServiceName(Name),
			Insecure:    true,
		}); err != nil {
			logger.Error("初始化链路追踪失败", zap.Error(err))
		}
		defer func() { _ = trace.Shutdown(context.Background()) }()
	}

	// 5. Wire 注入应用（cleanup 预留给后续 wave 释放 etcd 等资源）
	app, cleanup, err := initApp(bc)
	if err != nil {
		panic(err)
	}
	defer cleanup()
	logger.Info("依赖注入初始化成功")

	// 5. 启动 HTTP 服务 + 优雅退出（SIGINT/SIGTERM）
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 服务异常退出时触发退出流程
	go func() {
		if err := app.StartServer(); err != nil {
			logger.Error("HTTP 服务异常退出", zap.Error(err))
			stop()
		}
	}()

	// 阻塞等待退出信号
	<-ctx.Done()
	logger.Info("收到退出信号，开始优雅退出 HTTP 服务")

	done := make(chan error, 1)
	go func() { done <- app.Close() }()
	select {
	case err := <-done:
		if err != nil {
			logger.Error("HTTP 服务优雅退出失败", zap.Error(err))
		} else {
			logger.Info("HTTP 服务已优雅退出")
		}
	case <-time.After(15 * time.Second):
		logger.Warn("优雅退出超时，强制退出")
	}
	logger.Info("entry 服务已退出")
}
