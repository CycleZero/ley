# Gateway 服务 — 开发指引

**生成:** 2026-06-03 | **提交:** e9c6a42

## 概述

HTTP 入口网关，监听 `:8000`，gRPC 转发 Auth（`:9001`）和 Blog（`:9002`）。**不遵循 Kratos DDD**，基于 [go-kratos/gateway](https://github.com/go-kratos/gateway) 嵌入，无 `internal/` 目录。

**顶层包：** `cmd/gateway/main.go`（入口）、`config/`（YAML + 热更新）、`client/`（gRPC 工厂）、`discovery/`（etcd/consul）、`middleware/`（cors、jwt、logging、tracing、ratelimit、bbr、circuitbreaker、rewrite、transcoder、wrapresp、streamrecorder）、`proxy/`（路由构建/重试/熔断）、`router/`（mux）、`server/`（HTTP server）。

## 请求处理流程

```
HTTP → Proxy.ServeHTTP → Router → Middleware Chain → Client → Backend (gRPC)
```

路由由配置定义 `Endpoint`（path、method、protocol、middlewares、backends）。Proxy 在 `Update()` 时构建路由表，`atomic.Value` 热切换。

## 中间件链

通过 `init()` + 空白导入注册到全局 registry。链顺序由配置 `middlewares` 列表决定，从外到内：

```
cors → jwt → logging → tracing → ratelimit → bbr → circuitbreaker → rewrite → transcoder → wrapresp
```

- **cors** 跨域 | **jwt** Token 校验 + metadata 注入 | **logging** 请求日志 | **tracing** 链路追踪
- **ratelimit** 限流 | **bbr** 自适应过载保护 | **circuitbreaker** 熔断（Aegis SRE）
- **rewrite** URL 重写 | **transcoder** HTTP ↔ gRPC 协议转换 | **wrapresp** 响应包装

**添加中间件：** 新建目录 → 实现 `Factory` → `init()` 调 `middleware.Register()` → `main.go` 空白导入。

## 响应格式

`wrapresp` 包装所有 JSON 响应：`{"code": 0, "msg": "success", "data": <原始体>}`。错误时 `code` = HTTP 状态码，`data` = null。非 JSON 跳过。

## 配置体系

1. **本地** `./data/gateway/configs/config.yaml`：server、discovery DSN、日志、追踪
2. **路由**（同文件或 etcd）：Endpoint 列表
3. **控制面**（可选）：`--ctrl.service` 从 etcd 动态加载
4. **热更新**：文件变更或 ctrl service 推送 → 自动重载路由表

## 关键文件

| 文件 | 说明 |
|------|------|
| `cmd/gateway/main.go` | 启动入口 |
| `proxy/proxy.go` | 核心：路由构建、请求处理、重试、熔断 |
| `proxy/retry.go` | 重试策略 |
| `middleware/registry.go` | 中间件注册中心 |
| `middleware/wrapresp/wrapresp.go` | 响应包装实现 |
| `client/factory.go` | gRPC 客户端工厂 |
| `config/config.go` | 文件加载 + Watch |
| `discovery/registry.go` | DSN 解析 |

## Proto 生成

独立 Makefile，根目录 `make api` 不处理：

```bash
cd app/gateway && make api
```

## 反模式（Gateway 特有）

| 规则 | 说明 |
|------|------|
| **中间件必须注册** | `init()` 调 `Register()`，否则报 `ErrNotFound` |
| **空白导入在 main.go** | 新中间件需在 `main.go` 加 `_ import` |
| **路由变更必须 Update** | 通过 `Proxy.Update()` 或 config loader 重建 |
| **禁止直接操作 router** | 路由由 Proxy 统一管理 |
| **wrapresp 仅包装 JSON** | 非 JSON 响应不受影响 |
