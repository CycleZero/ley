# Ley — 个人博客平台

Ley 是一个采用 Go/Kratos 微服务 + Nuxt 4 全栈构建的个人博客平台。后端遵循 Kratos DDD 四层架构，前端使用 Vue 3 组合式 API 与 TailwindCSS 语义化设计系统，支持暗色模式、Markdown 渲染、嵌套评论、文件直传等现代博客特性。

---

## 特性

- **文章管理**：创建、编辑、发布、归档、搜索、点赞；支持 Markdown 渲染与代码高亮
- **标签 & 分类**：树形分类结构，灵活的文章组织方式
- **评论系统**：嵌套回复，最大支持 5 级深度
- **用户认证**：注册、登录、JWT 双令牌（15 分钟 Access + 7 天 Refresh）、Token 黑名单、个人资料管理
- **文件上传**：集成 MinIO / 阿里云 OSS，支持服务端上传与预签名 URL 直传
- **站点配置**：后台可动态管理站点信息、背景、音乐播放列表
- **暗色模式**：基于 `@nuxtjs/color-mode` 的系统/亮色/暗色切换
- **可观测性**：OpenTelemetry 链路追踪、Prometheus 指标、结构化日志

---

## 技术栈

### 后端

| 技术 | 说明 |
|------|------|
| [Kratos](https://go-kratos.dev/) v2.9 | Go 微服务框架（gRPC + HTTP） |
| Protocol Buffers | API 定义与代码生成 |
| Google Wire | 编译期依赖注入 |
| GORM | ORM（PostgreSQL / MySQL） |
| Redis | 缓存、Token 黑名单、分布式锁 |
| NATS JetStream | 事件总线与消息队列 |
| MinIO / 阿里云 OSS | 对象存储（S3 兼容 / OSS SDK v2） |
| OpenTelemetry | 链路追踪 |

### 前端

| 技术 | 说明 |
|------|------|
| [Nuxt](https://nuxt.com/) 4 | Vue 3 全栈框架 |
| [Pinia](https://pinia.vuejs.org/) | 状态管理 |
| [TailwindCSS](https://tailwindcss.com/) 3.4 | 原子化 CSS + 自定义语义化设计系统 |
| [@nuxtjs/mdc](https://github.com/nuxt-modules/mdc) | Markdown 渲染与代码高亮 |
| [VueUse](https://vueuse.org/) | 组合式工具集 |
| [Lucide Vue](https://lucide.dev/) | 图标库 |

---

## 架构

```
客户端（Web） ──HTTP──> Gateway (:8000) ──gRPC──> Auth (:9001)
                                                ──gRPC──> Blog (:9002)
```

- **Gateway**：HTTP 入口网关，负责 JWT 校验、CORS、限流、熔断、链路追踪、请求路由转发
- **Auth**：用户注册/登录/JWT 管理/个人资料 CRUD
- **Blog**：文章/评论/标签/分类/文件/站点配置管理

---

## 目录结构

```
ley/
├── api/                    # Proto API 定义（gRPC + HTTP/REST）
│   ├── auth/v1/
│   ├── blog/v1/
│   └── gateway/
├── app/                    # 微服务实现
│   ├── auth/               # 认证服务
│   ├── blog/               # 博客服务
│   └── gateway/            # API 网关
├── web/                    # Nuxt 4 前端
│   ├── app/pages/          # 文件路由页面
│   ├── app/stores/         # Pinia 状态管理
│   ├── app/composables/    # API 组合函数（统一请求入口）
│   └── app/components/     # 原子化 UI + 布局 + 文章组件
├── pkg/                    # 共享库（JWT、缓存、元数据传递、安全、事件总线等）
├── conf/                   # 共享 Proto 配置定义
├── configs/                # 运行时 YAML 引导配置
├── docs/                   # 设计文档与部署指南
├── docker-compose.yml      # Docker Compose 编排
├── Dockerfile              # 构建基础镜像
├── runtime.Dockerfile      # 运行时基础镜像
└── Makefile                # 构建、生成、测试命令集
```

---

## 快速开始

### 前置依赖

- Go 1.26+
- Node.js 22+ & pnpm 10+
- Protocol Buffers (protoc) v25+
- Docker & Docker Compose（可选，用于容器化部署）
- PostgreSQL / MySQL、Redis、etcd、NATS、MinIO（用于运行时依赖）

### 安装构建工具

```bash
make init
```

这会安装 `protoc-gen-go`、`protoc-gen-go-grpc`、`protoc-gen-go-http`、`protoc-gen-openapi`、`wire` 等工具。

### 构建后端

```bash
# 构建 auth + blog 服务
make build

# 构建网关（独立模块）
make build-gateway

# 完整重新构建：proto 生成 + wire + 编译
make rebuild
```

### 构建前端

```bash
cd web
pnpm install
pnpm build
```

### 运行服务

各服务通过本地 YAML 配置启动，默认从 `./data/{service}/configs/config.yaml` 加载引导配置，再从 etcd 加载业务配置。

```bash
# 单独运行（开发模式）
./bin/auth -conf ./data/auth/configs
./bin/blog -conf ./data/blog/configs
./bin/gateway -conf ./data/gateway/configs
```

或一键启动全部（Docker）：

```bash
make docker-build
make docker-up
```

---

## 开发工作流

### 常用 Make 命令

```bash
make api              # 从 api/ 生成 pb.go、http、grpc、openapi
make config           # 从 conf/ 和 app/**/internal/conf/ 生成配置 proto 代码
make internal_proto   # 生成 app/ 内部 proto 代码
make wire             # 为所有服务运行 Wire 依赖注入生成
make wire-all         # 所有服务 + gateway 运行 wire
make rebuild          # api + config + internal_proto + wire + build（完整重新构建）

make test-unit        # 运行单元测试（含覆盖率）
make test-integration # 运行集成测试（需要 Docker 基础设施）
make test-coverage    # 在浏览器中打开覆盖率报告
```

### 前端开发

```bash
cd web
pnpm dev        # 启动 Nuxt 开发服务器（:3000）
```

前端 API 调用规范：所有 HTTP 请求必须通过 `composables/useXxxApi.ts` 和 `stores/*.ts`，禁止在页面/组件中直接使用 `$fetch`。参见 [CLAUDE.md](./CLAUDE.md)。

---

## 部署

### Docker Compose（推荐）

项目已提供 `docker-compose.yml`，使用 **host 网络模式** 编排 Gateway、Auth、Blog 三个服务：

```bash
docker-compose build
docker-compose up -d
```

- Gateway 暴露 `:8000`
- Auth 暴露 `:9001`（gRPC）/ `:8001`（HTTP）
- Blog 暴露 `:9002`（gRPC）/ `:8002`（HTTP）

详见 [docs/deployment.md](./docs/deployment.md)。

### 配置体系

1. **引导配置**（本地 YAML `./data/{service}/configs/config.yaml`）：服务器地址、etcd 端点、日志、链路追踪等。
2. **业务配置**（来自 etcd）：数据库/Redis/JWT/MinIO/NATS 等参数，启动时通过 Kratos etcd config source 动态加载。

---

## 测试

```bash
# 单元测试（-short 标志，含覆盖率）
make test-unit

# 集成测试（需要 Docker 基础设施）
make test-integration

# 查看覆盖率报告
make test-coverage
```

---

## 相关文档

- [CLAUDE.md](./CLAUDE.md) — 项目开发规范与架构约定
- [docs/design.md](./docs/design.md) — 系统设计文档（架构、API、数据库、缓存、事件流）
- [docs/deployment.md](./docs/deployment.md) — 部署指南
- [docs/frontend-design.md](./docs/frontend-design.md) — 前端设计文档
- [docs/nuxt-generate-deployment-lessons.md](./docs/nuxt-generate-deployment-lessons.md) — Nuxt 部署经验

---

## License

[MIT License](./LICENSE)
