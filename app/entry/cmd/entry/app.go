package main

import (
	"context"
	"net/http"
	"time"

	"github.com/CycleZero/ley/app/entry/internal/common"
	"github.com/CycleZero/ley/app/entry/internal/domain/proxy"
	"github.com/CycleZero/ley/app/entry/internal/router"
	commonconf "github.com/CycleZero/ley/conf"
	plog "github.com/CycleZero/ley/pkg/log"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// shutdownTimeout 优雅退出时等待存量请求处理完成的超时时间
const shutdownTimeout = 10 * time.Second

// MainApp 应用主结构：封装 Gin Engine 与 HTTP Server，是入口服务唯一应用对象。
type MainApp struct {
	Engine       *gin.Engine         // Gin 路由引擎
	ServiceHub   *proxy.ServiceHub   // 代理服务聚合（本 wave 空壳，后续 wave 挂载 handler）
	httpServer   *http.Server        // HTTP 服务器（支持优雅退出）
	RegisterFunc router.RegisterFunc // 路由注册函数
}

// NewMainApp 创建主应用实例（由 Wire 注入）。
//
// 组装顺序固定：gin.New → 基础中间件（Recovery 信封）→ RegisteredMiddleWire.Register()
// → 业务路由注册——保证中间件先于路由，且后续 wave 无需改动此骨架。
func NewMainApp(
	bc *commonconf.Bootstrap,
	hub *proxy.ServiceHub,
	registerFunc router.RegisterFunc,
	registeredMiddleWire router.RegisteredMiddleWire,
) *MainApp {
	// gin 运行模式跟随引导配置日志模式：Dev→DebugMode，Prod→ReleaseMode
	if bc.Log == nil || bc.Log.Mode == commonconf.LogMode_Dev {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	e := gin.New()
	// 开启 NoMethod 支持：方法不匹配时进入 NoMethod 处理器而非 NoRoute
	e.HandleMethodNotAllowed = true

	// 全局 Recovery：panic 统一返回 500 信封 JSON，避免堆栈泄漏给客户端。
	// AbortWithStatusJSON 可在 handler 已写出部分响应时强制覆盖状态码与报文
	e.Use(gin.CustomRecovery(func(c *gin.Context, err any) {
		plog.GetLogger().Error("请求处理发生 panic", zap.Any("panic", err))
		c.AbortWithStatusJSON(http.StatusInternalServerError, common.Response{
			Code: http.StatusInternalServerError,
			Msg:  "服务器内部错误",
			Data: nil,
		})
	}))

	// 注册自定义中间件（必须在路由注册前完成；本 wave 为空壳）
	registeredMiddleWire.Register()

	// 注册全部路由（healthz/readyz + NoRoute/NoMethod 兜底）
	registerFunc(e, hub)

	// 解析监听地址（server.http.addr），缺省回退 0.0.0.0:8000
	addr := "0.0.0.0:8000"
	if bc.Server != nil && bc.Server.Http != nil && bc.Server.Http.Addr != "" {
		addr = bc.Server.Http.Addr
	}

	app := &MainApp{
		Engine:       e,
		ServiceHub:   hub,
		RegisterFunc: registerFunc,
		httpServer: &http.Server{
			Addr:              addr,
			Handler:           e,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}

	app.printRoutes()
	return app
}

// printRoutes 打印已注册路由，便于开发期核对（日志消息走 pkg/log，中文）
func (a *MainApp) printRoutes() {
	logger := plog.GetLogger()
	routes := a.Engine.Routes()
	logger.Info("路由注册完成", zap.Int("total", len(routes)))
	for _, route := range routes {
		logger.Debug("已注册路由", zap.String("method", route.Method), zap.String("path", route.Path))
	}
}

// StartServer 启动 HTTP 服务（阻塞，直到服务退出）。
//
// 优雅退出路径（Close 触发 Shutdown）会返回 http.ErrServerClosed，
// 此处归一为 nil，由调用方按正常退出处理。
func (a *MainApp) StartServer() error {
	plog.GetLogger().Info("HTTP 服务启动", zap.String("addr", a.httpServer.Addr))
	err := a.httpServer.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// Close 优雅关闭 HTTP 服务：等待进行中的请求处理完成，超时后强制退出。
func (a *MainApp) Close() error {
	if a.httpServer == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return a.httpServer.Shutdown(ctx)
}
