# PROJECT KNOWLEDGE BASE

**最近更新:** 前端重构后刷新（Nuxt 4 → React SPA）
**分支:** master

## 概述

Ley 是一个个人博客平台，采用 Go/Kratos 微服务单体仓库 + React 19 SPA 前端（`ley-web/`）。Gateway 作为 HTTP 入口网关，gRPC 转发到 Auth（认证）和 Blog（博客）服务。

## 结构

```
ley/
├── api/                    # Proto API 定义（gRPC + HTTP/REST 注解）
│   ├── auth/v1/            # 认证服务 API
│   ├── blog/v1/            # 博客服务 API
│   ├── common/v1/          # 共享类型（TokenPair, UserInfo, AuthorInfo）
│   └── gateway/            # 网关中间件 Proto 配置
├── app/
│   ├── auth/               # 认证服务 — 标准 Kratos DDD（见下方分层）
│   ├── blog/               # 博客服务 — 标准 Kratos DDD（见下方分层）
│   └── gateway/            # API 网关 — 非 DDD 结构（见 gateway/AGENTS.md）
├── ley-web/                # React 19 SPA 前端（Vite 8，2026-07 由 Nuxt 4 重构而来）
│   └── src/
│       ├── pages/          # 页面（前台 + /admin 后台）
│       ├── components/     # 布局 + 业务组件（layout/、RequireAuth、RequireAdmin）
│       ├── hooks/          # TanStack Query 数据获取（use-articles、use-tags 等）
│       ├── stores/         # Zustand 纯客户端状态（auth、ui、draft）
│       ├── lib/            # api-client.ts（ofetch 封装）、i18n.ts
│       ├── i18n/           # zh-CN / en-US 语言包
│       ├── styles/         # globals.css（设计 token）
│       └── routes.tsx      # React Router v7 路由树 + 布局嵌套
├── pkg/                    # 共享 Go 库（17 个子包）
│   ├── meta/               # 用户上下文跨服务传递（x-md-global- 前缀）
│   ├── jwt/                # JWT 生成与校验（etcd 动态密钥）
│   ├── infra/              # 基础设施初始化（DB/Redis/MinIO，NATS 段为遗留死代码）
│   ├── eventbus/           # 事件总线（生产接线为内存通道，NATS 实现就绪未启用）
│   ├── cache/              # Redis 缓存（接口 + 实现）
│   ├── oss/                # 对象存储（MinIO / 腾讯云 COS）
│   ├── mq/                 # 消息队列抽象（NATS JetStream + 内存实现；两服务当前用内存连接）
│   ├── security/           # 密码哈希（bcrypt）
│   ├── trace/              # OpenTelemetry 链路追踪
│   ├── task/               # 异步任务队列
│   ├── middleware/          # HTTP 中间件（EdgeOne、通用）
│   ├── testutil/           # 集成测试套件 + InMemoryCache
│   ├── util/               # 通用工具（DB 错误处理、哈希等）
│   ├── log/                # 结构化日志
│   ├── common/             # 共享数据对象
│   └── constant/           # 应用常量
├── conf/                   # 共享引导配置 Proto 定义
├── configs/                # 开发用 YAML 模板配置（部署时复制到 data/）
├── data/                   # 运行时数据：每服务 config.yaml + 日志
├── docs/                   # 设计文档、部署指南
│   ├── design.md           # 系统设计（架构、API、数据库、缓存、事件流）
│   ├── frontend-design.md  # 旧 Nuxt 前端设计（已废弃，仅作需求参考）
│   └── frontend-rewrite-tech-selection.md # 前端重写技术选型（现行规范）
├── third_party/            # 第三方 Proto（google、validate）
└── bin/                    # 编译输出（gitignored）
```

## 去哪儿找

| 任务 | 位置 | 备注 |
|------|------|------|
| API 定义（Proto） | `api/{service}/v1/` | 修改后运行 `make api` 重新生成 |
| 身份认证逻辑 | `app/auth/internal/biz/auth.go` | Register/Login/JWT/Refresh |
| 用户 CRUD | `app/auth/internal/biz/user.go` | 密码规则、GORM 零值跳过 |
| 文章 CRUD | `app/blog/internal/biz/article.go` | 最大文件，核心业务逻辑 |
| 标签/分类 | `app/blog/internal/biz/tag.go` | 树形分类，循环引用检测 |
| 文件上传 | `app/blog/internal/biz/file.go` | MinIO 直传、MIME 校验 |
| 网关路由/中间件 | `app/gateway/proxy/`、`app/gateway/middleware/` | JWT/CORS/限流/熔断/链路追踪 |
| 前端 API 客户端 | `ley-web/src/lib/api-client.ts` | ofetch：Token 注入、401 单飞刷新、`{code,msg,data}` 解包 |
| 前端数据获取 | `ley-web/src/hooks/use-*.ts` | TanStack Query（tags/categories staleTime 60s） |
| 前端页面 | `ley-web/src/pages/` | 文件路由，含 `/admin` 后台 |
| 前端状态管理 | `ley-web/src/stores/` | Zustand（auth、ui、draft） |
| 前端路由/守卫 | `ley-web/src/routes.tsx` | React Router v7，RequireAuth/RequireAdmin |
| 设计 token | `ley-web/src/styles/globals.css` | 现代极简（Cloudreve 参考）CSS 变量 |
| 配置 Proto | `conf/common.proto`、`app/*/internal/conf/` | 引导配置结构定义 |
| Docker 编排 | `docker-compose.yml` | host 网络模式，3 服务 |
| GitHub Actions | `.github/workflows/` | CI（构建+测试）+ CD（前后端独立部署） |

## Kratos DDD 分层（Auth / Blog）

```
app/{service}/
├── cmd/{service}/          # Wire 依赖注入入口
│   ├── main.go             # 启动入口
│   ├── wire.go             # ProviderSet 声明
│   └── wire_gen.go         # 生成代码（gitignored）
└── internal/
    ├── conf/               # 服务专用 proto 配置结构 + conf.proto
    ├── biz/                # 业务逻辑层：UseCase + 仓储接口定义
    ├── data/               # 数据层：仓储实现（GORM + Redis + OSS）
    ├── service/             # 传输层：gRPC/HTTP handler
    └── server/              # gRPC + HTTP server 构建

依赖方向: cmd → service → biz ← data
```

**Gateway 不遵循此分层** — 它是上游 go-kratos/gateway 的嵌入，结构完全不同。参见 `app/gateway/AGENTS.md`。

## 配置体系

两级配置：

1. **引导配置**（本地 YAML）：`./data/{service}/configs/config.yaml`
   - 服务器地址、etcd 端点、日志设置、链路追踪端点
   - 结构由 `conf/common.proto` 定义
   - 开发模板在 `configs/` 目录

2. **业务配置**（来自 etcd）：
   - 数据库/Redis/JWT/NATS/MinIO 等运行时参数
   - 启动时通过 Kratos etcd config source 动态加载（路径 `ley/configs/{service}/config.yaml`）
   - 网关 JWT 密钥支持 etcd watch 热更新

## 用户上下文传递

跨服务传递用户信息，使用 Kratos metadata（`x-md-global-` 前缀）：

```go
// 读取
meta := pkg/meta.GetRequestMetaData(ctx)

// 注入到下游调用
ctx = pkg/meta.NewClientCtx(ctx, meta)
```

**禁止使用原生 `ctx.Value`。**

## 反模式（本项目特有）

### 🔴 致命级

| 规则 | 说明 |
|------|------|
| **禁止 `ctx.Value` 获取用户信息** | 必须用 `pkg/meta.GetRequestMetaData(ctx)` |
| **禁止页面/组件直接裸调 `ofetch`/`fetch`** | 必须走 `hooks/use-*.ts → lib/api-client.ts`；**auth store 例外**（login/register/refresh/logout 内联避免循环依赖）|
| **JWT Secret 必须 256 位随机** | 生产环境部署前必须替换 |
| **生产环境关闭 Debug 日志** | `log.level: info` |

### 🟠 严重级

| 规则 | 说明 |
|------|------|
| **GORM 零值跳过** | Update 用 `map[string]interface{}`，不用 struct |
| **并发更新用 SQL 表达式** | `gorm.Expr("view_count + ?")`，避免读-改-写竞态 |
| **唯一性检查用 ON CONFLICT** | `INSERT ... ON CONFLICT DO NOTHING` 兜底，不靠"检查再操作" |
| **`useCookie` + Pinia 是危险组合** | ~~旧 Nuxt 遗留~~ 前端现状：Token 存 Cookie（`ley_at`/`ley_rt`），Zustand 不作为 token 真实来源 |
| **避免 KEYS 命令** | 用 SCAN 迭代，避免阻塞 Redis |

### 🟡 业务约束

| 规则 | 说明 |
|------|------|
| 不允许删除存在子分类的父分类 | `app/blog/internal/biz/tag.go` |
| 不允许删除存在文章的分类 | `app/blog/internal/biz/tag.go` |
| 分类不能构成循环引用 | `app/blog/internal/biz/tag.go` |
| 密码必须包含大写、小写、数字 | `app/auth/internal/biz/user.go` |
| 文章内容 ≤ 100000 字符 | `app/blog/internal/biz/article.go` |
| 文件上传校验扩展名 + MIME + 魔数 | `app/blog/internal/biz/file.go` |

## 独有风格

- **中文为主**：后端所有错误消息、代码注释、日志输出使用中文
- **Proto 描述中文**：`summary`、`description`、`tags` 均为中文
- **前端现代极简**（2026-07 重构，Cloudreve 参考）：灰底白卡、12px 统一圆角、扁平无阴影按钮、自定义滚动条；TailwindCSS v4 + CSS 变量；组件变体用 CVA；表单 react-hook-form + zod；Toast 用 sonner；动画 framer-motion
- **前端状态分层**：服务端数据 → TanStack Query（缓存/后台刷新），纯客户端状态 → Zustand（auth/ui/draft）
- **测试纯标准库**：不用 testify，所有断言 `t.Fatalf` / `t.Errorf`
- **Mock 内联**：无 `mocks/` 目录，mock 直接写在 `helpers_test.go` 中
- **TestMain + sync.Once**：共享真实数据库连接，避免重复初始化
- **唯一 ID 生成**：`uniSlug(t, "prefix")` / `uniq(t, "prefix")` 用纳秒后缀去重

## 命令

```bash
# === 构建 ===
make build              # 构建 auth + blog 到 bin/
make build-auth          # 仅构建 auth
make build-blog          # 仅构建 blog
make build-gateway       # 构建 gateway（独立目录）
make build-all           # 构建全部（含 gateway）
make rebuild             # proto + wire + build 完整重建

# === Proto 生成 ===
make api                 # api/ proto → pb.go + http + grpc + openapi
make config              # conf/ + app/**/conf/ → 配置 proto 代码
make internal_proto      # app/ 内部 proto
make wire                # Wire 依赖注入代码生成
make wire-all            # 含 gateway

# === 测试 ===
make test-unit           # 单元测试（data 层集成测试已用 //go:build integration 隔离，连库需 LEY_TEST_MYSQL_DSN + make test-integration）
make test-integration    # 集成测试，需 Docker 基础设施
make test-coverage       # 浏览器打开覆盖率报告

# === Docker ===
make docker-build        # docker-compose build（需先构建基础镜像）
make docker-up           # docker-compose up -d
make docker-down         # docker-compose down

# === 前端 ===
cd ley-web && pnpm dev       # Vite 开发服务器 :3000（/api 代理到 :8000）
cd ley-web && pnpm build     # tsc -b && vite build（生产构建，输出 dist/）
cd ley-web && pnpm lint      # oxlint
```

## 注意事项

- **前端是纯 SPA**：CD 部署 `pnpm build` 静态产物 → Nginx `try_files $uri /index.html`（非 SSR）
- **开发代理**：`/api` → `http://localhost:8000`（ley-web/vite.config.ts）
- **评论系统已删除**：后端评论模块已移除（commit 8da75c36），前端不规划评论功能；`docs/design.md` 中评论设计已过时
- **AGENTS.md 已更新**：旧 Nuxt 4 描述作废，`docs/frontend-design.md` 仅作需求参考
- **基础镜像依赖**：Docker 构建前必须运行 `./build-deploy-image.sh` 构建 `ley-builder:v1` 和 `ley-runtime:v1`，或从 GHCR 拉取
- **docker-compose 无基础设施**：不含 MySQL/Redis/etcd/MinIO 容器，需单独部署；数据库为 MySQL（configs/ 模板已对齐）
- **Host 网络模式**：docker-compose 使用 `network_mode: host`，无端口映射，不能多实例
- **Gateway proto**：api/gateway 由根 `make api` 统一生成（`app/gateway/Makefile` 已失效：`find api` 指向不存在的目录）
- **中国镜像**：Go 代理 `goproxy.cn`，Debian 源 `mirrors.ustc.edu.cn`，非中国网络需调整
- **空目录 pkg/dtm/**：遗留空目录，建议清理
