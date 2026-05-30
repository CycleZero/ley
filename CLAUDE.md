# CLAUDE.md

本文件为 Claude Code (claude.ai/code) 在此仓库中工作时提供指引。

## 语言规范

项目中所有错误消息、代码注释、日志输出均使用中文。

## 常用命令

```bash
# 构建
make build              # 构建 auth + blog 服务到 bin/
make build-auth          # 仅构建 auth
make build-blog          # 仅构建 blog
make build-gateway       # 构建 gateway（独立模块）
make build-all           # 构建所有服务（含 gateway）

# Proto 与代码生成
make api                 # 从 api/ 目录的 proto 文件生成 Go 代码（pb.go, http, grpc, openapi）
make config              # 从 conf/ 和 app/**/internal/conf/ 生成配置 proto 代码
make internal_proto      # 生成 app/ 内部 proto 代码
make all                 # api + config + generate
make wire                # 为所有服务运行 wire 依赖注入代码生成
make wire-all            # 为所有服务 + gateway 运行 wire
make rebuild             # api + config + internal_proto + wire + build（完整重新构建）

# 测试
make test                # 运行单元测试 + 集成测试
make test-unit           # 运行单元测试（含覆盖率，-short 标志）
make test-integration    # 运行集成测试（需要 Docker 基础设施）
make test-coverage       # 在浏览器中打开覆盖率报告

# Docker
make docker-build        # docker-compose 构建所有镜像
make docker-up           # docker-compose 启动所有服务
make docker-down         # docker-compose 停止所有服务

# 前端
cd web && pnpm dev       # 启动 Nuxt 开发服务器（:3000）
cd web && pnpm build     # 生产构建
```

## 架构概览

**Ley** 是一个个人博客平台，采用 Go/Kratos 微服务单体仓库 + Nuxt 4/Vue 3 前端。

```
客户端（Web） --> Gateway (:8000) --gRPC--> Auth (:9001)
                                   --gRPC--> Blog (:9002)
```

- **Gateway**（`app/gateway`）：HTTP 入口网关。负责 JWT 校验、CORS、限流、熔断、链路追踪、请求重写、路由到后端 gRPC 服务。基于 Kratos gateway。
- **Auth**（`app/auth`）：用户注册、登录、JWT token 管理（15分钟访问令牌 + 7天刷新令牌）、个人信息 CRUD、Redis token 黑名单。
- **Blog**（`app/blog`）：文章 CRUD、评论（支持嵌套回复）、标签、分类、文件上传（MinIO）、站点配置管理。
- **web/**（`web/`）：Nuxt 4 前端。使用 Pinia 状态管理、TailwindCSS、`@nuxtjs/mdc` 渲染 Markdown。

共享库位于 `pkg/`：`meta`（用户上下文跨服务传递）、`jwt`、`infra`、`eventbus`（NATS JetStream）、`cache`、`oss`、`mq`、`security`、`trace`、`util`、`testutil`。

## Kratos DDD 分层约定

每个服务严格遵循 4 层 Kratos DDD 结构：

```
app/{service}/
  cmd/{service}/main.go    -- 入口，Wire 依赖注入容器
                wire.go    -- Wire provider set 定义
  internal/
    conf/                  -- 服务专用 protobuf 配置结构体
    biz/                   -- 业务逻辑层（UseCase、领域模型、repo 接口定义）
    data/                  -- 数据层（repo 实现：数据库、缓存、外部 API 调用）
    service/               -- gRPC/HTTP 处理层（传输层，实现 proto 生成的 server 接口）
    server/                -- 服务器构建（gRPC + HTTP server 创建）
```

依赖方向：`cmd` -> `service` -> `biz` <- `data`。`biz` 层定义仓储接口，`data` 层实现接口。Google Wire 在 `cmd/` 层编译时将各部分组装注入。

## 用户上下文传递

用户信息通过 Kratos metadata 跨服务传递，使用 `x-md-global-` 前缀的键名（Kratos 中间件会自动在服务间转发这些键值）。

- 获取用户信息：`pkg/meta.GetRequestMetaData(ctx)`
- 注入用户信息到下游调用：`pkg/meta.NewClientCtx(ctx, meta)`
- **禁止**使用原生 `ctx.Value` 获取用户信息。

## 配置体系

两级配置系统：
1. **引导配置**（本地文件 `./data/{service}/configs/config.yaml`）：服务器地址、etcd 端点、日志设置、链路追踪端点等。
2. **业务配置**（来自 etcd）：数据库/Redis/JWT/NATS/MinIO 等业务参数，启动时通过 Kratos etcd config source 加载。

## Proto 代码生成

API proto 文件位于 `api/{service}/`，使用 `google.api.http` 注解同时生成 gRPC 和 HTTP/REST 端点。修改 proto 文件后运行 `make api` 重新生成。`app/` 目录下的内部 proto 使用 `make internal_proto`。`conf/` 和 `app/**/internal/conf/` 下的配置 proto 使用 `make config`。

## 前端 API 调用规范（铁律）

**所有 HTTP API 请求必须通过统一的 API composables 和 Pinia stores，禁止在页面/布局/组件中直接使用 `$fetch` 或 `fetch`。**

正确路径：
```
pages/*.vue → stores/*.ts → composables/useXxxApi.ts → useApiClient.ts → 后端
```

- **页面/组件**：只调用 `store.action()`，不直接发起 HTTP 请求
- **stores**：使用 `useXxxApi()` composables，不直接写 `$fetch`
- **API composables**：基于 `useApiClient()`（已封装 Token 注入、401 刷新、响应解包）
- **唯一例外**：`auth store` 中的 `login/register/refresh/logout` 为避免循环依赖，内联使用 `$fetch`

违反此规则会导致：
- 响应解包逻辑分散，后端 `{ code, msg, data }` 格式处理不一致
- 401 刷新重试失效
- Toast 错误提示不统一
- 代码不可维护
