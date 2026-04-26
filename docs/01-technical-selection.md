# 技术选型方案

## 1. 项目概述

### 1.1 项目背景

本系统是一个**分布式个人博客平台**，采用微服务架构，旨在提供高可用、可扩展、高性能的个人内容发布与管理平台。系统需支持博客文章的发布、编辑、删除、分类、标签管理，用户认证与授权，文件上传与存储，评论互动，全文搜索，通知推送等核心功能。

### 1.2 架构目标

| 目标 | 指标 |
|------|------|
| 可用性 | 99.9%（全年停机时间 < 8.76 小时） |
| 可扩展性 | 支持水平扩展，单服务峰值 QPS ≥ 5000 |
| 响应时间 | P99 < 200ms（读），P99 < 500ms（写） |
| 数据一致性 | 最终一致性（事件驱动），关键路径强一致 |
| 可观测性 | 全链路追踪、指标监控、日志聚合 |

---

## 2. 技术栈全景

### 2.1 核心技术栈一览

```
┌─────────────────────────────────────────────────────────────────┐
│                        技术栈架构全景                              │
├───────────────┬─────────────────────────────────────────────────┤
│   层次         │   技术选型                                        │
├───────────────┼─────────────────────────────────────────────────┤
│   编程语言     │   Go 1.23+                                      │
│   微服务框架   │   Kratos v2 (go-kratos/kratos/v2)               │
│   协议定义     │   Protocol Buffers 3 (proto3)                    │
│   API 风格     │   gRPC (服务间) + HTTP/REST (对外)               │
│   RPC 传输     │   gRPC (HTTP/2)                                 │
│   HTTP 传输    │   net/http + gorilla/mux (Kratos 内置)          │
│   链路追踪     │   OpenTelemetry + Jaeger                        │
│   服务注册发现  │   etcd v3                                       │
│   配置中心     │   etcd v3 + 本地 YAML                           │
│   消息队列     │   NATS JetStream                                │
│   关系数据库   │   PostgreSQL 15+                                │
│   ORM          │   GORM v2                                      │
│   缓存         │   Redis 7+ (go-redis/v9)                        │
│   对象存储     │   MinIO (minio-go/v7)                           │
│   依赖注入     │   Google Wire                                   │
│   日志         │   Zap (uber-go/zap)                             │
│   认证鉴权     │   JWT (golang-jwt/jwt)                          │
│   容器化       │   Docker + Docker Compose                        │
│   编排         │   Kubernetes (可选)                              │
│   监控         │   Prometheus + Grafana (可选)                    │
│   网关         │   Kratos HTTP Server + 自定义中间件              │
│   限流         │   Kratos 内置 ratelimit + 令牌桶                │
└───────────────┴─────────────────────────────────────────────────┘
```

---

## 3. 各项技术选型详述

### 3.1 编程语言：Go

| 维度 | 分析 |
|------|------|
| **选型理由** | 天然支持高并发（goroutine + channel），编译为单一二进制，部署极简；静态类型，IDE 友好；GC 延迟低，适合微服务场景 |
| **版本要求** | Go 1.23+（泛型支持，结构化日志 slog，net/http 路由增强） |
| **现有基础** | 项目已使用 Go 1.25.0，生态依赖成熟 |
| **备选方案** | Rust（学习曲线陡峭，生态不如 Go 成熟）、Java/Spring（资源占用高，启动慢） |

---

### 3.2 微服务框架：Kratos v2

| 维度 | 分析 |
|------|------|
| **选型理由** | B站开源的生产级 Go 微服务框架，内置中间件体系（recovery/tracing/logging/ratelimit/circuitbreaker），完美集成 OpenTelemetry、gRPC、服务发现；规范化项目布局（api/internal/pkg/cmd/configs） |
| **核心能力** | gRPC + HTTP 双协议支持、Proto 代码生成、中间件链、Wire 依赖注入集成、优雅关机、健康检查 |
| **版本** | v2.9.2 |
| **备选方案** | Go-Zero（七牛云，社区活跃度略低）、Go-Micro（过于重，侵入性强）、自建（维护成本高） |

**Kratos 分层架构在项目中的落地：**

```
┌──────────────────────────────────────────────────────────────┐
│  cmd/ley/       → 入口，依赖注入组装                          │
├──────────────────────────────────────────────────────────────┤
│  internal/                                                   │
│  ├── service/   → gRPC/HTTP Handler 层（协议适配）            │
│  ├── biz/       → 业务逻辑层（UseCase，领域模型）              │
│  ├── data/      → 数据访问层（Repo 实现，DB/Cache/MQ 操作）    │
│  ├── server/    → 传输服务器初始化                            │
│  └── conf/      → 配置定义（Protobuf 驱动）                   │
├──────────────────────────────────────────────────────────────┤
│  api/           → Proto 接口定义（服务契约）                   │
├──────────────────────────────────────────────────────────────┤
│  pkg/           → 通用工具库（可跨服务复用）                   │
└──────────────────────────────────────────────────────────────┘
```

---

### 3.3 通信协议：gRPC + HTTP/REST

| 维度 | 分析 |
|------|------|
| **服务间通信** | gRPC（基于 HTTP/2，Protobuf 序列化，支持 streaming、deadline、cancellation、interceptors） |
| **对外 API** | HTTP/REST（通过 Kratos gRPC-Gateway 或 HTTP Server 自动生成） |
| **API 文档** | Proto 文件自动生成 OpenAPI v3 规范 |
| **负载均衡** | 客户端负载均衡（通过 etcd 服务发现获取端点列表） |
| **序列化** | Protocol Buffers 3（强类型，向前向后兼容，序列化速度快，体积小） |

**协议选型对比：**

| 特性 | gRPC | HTTP/REST (JSON) | Thrift | Dubbo |
|------|------|-------------------|--------|-------|
| 性能 | 高 | 中 | 高 | 高 |
| 浏览器友好 | 否 | 是 | 否 | 否 |
| 工具链 | 丰富 | 极丰富 | 一般 | 少 |
| 流式支持 | 原生 | SSE/WS | 有限 | 有限 |
| Proto 规范 | 强 | 无 | 强 | 定制 |
| **结论** | **服务间首选** | **对外首选** | 淘汰 | 生态局限 |

---

### 3.4 服务注册与发现：etcd v3

| 维度 | 分析 |
|------|------|
| **选型理由** | 强一致（Raft）、高可用集群、Watch 机制（实时感知变化）、TTL 自动清理失效节点、KVS + Lease 模型简洁 |
| **使用场景** | 服务注册（Register/Discover）、配置管理（WatchConfig）、分布式锁、Leader 选举 |
| **客户端** | `go.etcd.io/etcd/client/v3`（v3.6.10，已对齐 GO-ETCD 3.6.x） |
| **备选方案** | Consul（功能更多但更重，Go 版本已停止维护）、Nacos（中文生态好但 Go SDK 不成熟）、ZooKeeper（Java 生态老牌，Go 客户端弱） |

**使用模式：**

```
服务注册:  每个实例启动时 PUT /services/{name}/{instanceID}  +  Lease (TTL=10s) + KeepAlive
服务发现:  WATCH /services/{name}/ 实时获取在线实例  +  GET /services/{name}/ 全量
配置管理:  WATCH /configs/{service}/ 监听配置变更，热更新
```

---

### 3.5 消息队列：NATS JetStream

| 维度 | 分析 |
|------|------|
| **选型理由** | CNCF 毕业项目，Go 原生实现，单节点百万级 QPS，支持 At-Least-Once / Exactly-Once，JetStream 提供持久化流、消费者组、重试、死信队列；部署运维极简（单二进制） |
| **使用场景** | 异步事件通知、数据同步、削峰填谷、最终一致性保证 |
| **客户端** | `github.com/nats-io/nats.go` v1.51.0 |
| **备选方案** | Kafka（重，需 Zookeeper/Kraft，运维成本高）、RabbitMQ（Erlang 生态，性能上限低）、Redis Streams（缺少 ACK/重试/死信，不专业） |

**Stream 设计：**

| Stream | 主题 | 用途 |
|--------|------|------|
| EVENTS.POST | `post.created`, `post.updated`, `post.deleted` | 文章生命周期事件 |
| EVENTS.COMMENT | `comment.created`, `comment.deleted` | 评论事件 |
| EVENTS.USER | `user.registered`, `user.updated` | 用户事件 |
| EVENTS.NOTIFICATION | `notification.email`, `notification.push` | 通知发送事件 |
| EVENTS.FILE | `file.uploaded`, `file.deleted` | 文件事件 |

---

### 3.6 数据库：PostgreSQL 15+

| 维度 | 分析 |
|------|------|
| **选型理由** | ACID 事务，MVCC 高并发读，JSONB 支持（文章元数据），全文搜索（`tsvector` + GIN 索引），窗口函数与 CTE，丰富的索引类型（B-Tree/Hash/GIN/GiST/BRIN），成熟稳定 |
| **驱动** | `gorm.io/driver/postgres` v1.6.0 |
| **ORM** | GORM v2（自动迁移、关联预加载、事务、Hook、Scope） |
| **备选方案** | MySQL（JSON/FullText 不如 PG 强大，License 风险 Oracle 收购）、MongoDB（缺乏事务强支持，关联查询弱）、TiDB（分布式 PG 兼容，运维复杂，个人博客过重） |

**分库策略：**

| 数据库 | 用途 |
|--------|------|
| `ley_user` | 用户、认证、角色 |
| `ley_post` | 文章、分类、标签 |
| `ley_comment` | 评论、回复 |
| `ley_notification` | 通知记录、消息模板 |

> 注：当前阶段采用逻辑分库（同一 PG 实例不同 Schema），后续可按需迁移至独立实例。

---

### 3.7 缓存：Redis 7+

| 维度 | 分析 |
|------|------|
| **选型理由** | 内存数据库，亚毫秒级响应，丰富数据结构（String/Hash/List/Set/SortedSet/Stream/Bitmap/Geo/JSON），Lua 脚本原子操作，Pub/Sub 轻量消息，Sentinel/Cluster 高可用 |
| **客户端** | `github.com/redis/go-redis/v9` v9.18.0 |
| **备选方案** | Dragonfly（Drop-in Redis 替代，多线程，但生态新有风险）、Memcached（只有 KV，功能弱）、Hazelcast（Java 生态重） |

**缓存策略：**

| 场景 | 策略 | TTL |
|------|------|-----|
| 文章内容（热门） | Cache-Aside + 主动失效 | 10min |
| 文章列表（首页） | Cache-Aside + 定时预热 | 5min |
| 用户 Session | 写入即缓存 | 30min（滑动过期） |
| 标签/分类 | 预加载缓存 | 60min |
| 点赞/浏览计数 | Write-Behind（定时刷 DB） | 持久 |
| 限流计数器 | 滑动窗口 + Lua | 按窗口 |
| 分布式锁 | SET NX + Lua 释放 | 30s |

---

### 3.8 对象存储：MinIO

| 维度 | 分析 |
|------|------|
| **选型理由** | 完全兼容 AWS S3 API，Go 原生实现，性能优异，支持纠删码、多副本、SSE 加密、WORM、版本控制，单机/集群部署灵活；AGPLv3 开源许可 |
| **客户端** | `github.com/minio/minio-go/v7` v7.0.100 |
| **使用场景** | 博客图片、附件、头像、静态资源存储 |
| **备选方案** | 阿里云 OSS/腾讯云 COS（公有云方案，已有 COS 实现）、SeaweedFS（S3 API 兼容性差）、Ceph（运维极重） |

**Bucket 规划：**

| Bucket | 用途 | 访问策略 | 生命周期 |
|--------|------|----------|----------|
| `ley-images` | 文章配图 | 公开读 | 永久 |
| `ley-attachments` | 文章附件 | 公开读 | 永久 |
| `ley-avatars` | 用户头像 | 公开读 | 永久 |
| `ley-temp` | 临时文件 | 私有 | 24h 过期 |

---

### 3.9 链路追踪：OpenTelemetry + Jaeger

| 维度 | 分析 |
|------|------|
| **选型理由** | OpenTelemetry 是 CNCF 毕业的观测标准，提供统一 API/SDK；Jaeger 是 Uber 开源的成熟追踪后端，支持 OTLP 协议，UI 友好 |
| **集成方式** | OpenTelemetry OTLP HTTP Exporter → Jaeger Collector → Jaeger Query UI |
| **中间件植入** | Kratos 内置 tracing 中间件，自动注入 HTTP/gRPC Span；数据库查询通过 GORM 插件打点；Redis 通过自定义 Hook；NATS 通过消息头传递 Trace Context |
| **采样策略** | 生产环境：10% 尾部采样 + 错误全采样；开发环境：100% |
| **备选方案** | Zipkin（社区不活跃）、SigNoz（新兴，不稳定）、SkyWalking（Java 生态优先） |

**追踪埋点层级：**

```
┌─────────────────────────────────────────────────────────────┐
│  HTTP Request (traceparent header)                          │
│  ├── Gateway Middleware (root span)                         │
│  ├── JWT Auth Middleware                                    │
│  ├── gRPC Client (propagate trace)                          │
│  │   ├── PostService.GetPost                                │
│  │   │   ├── Redis GET (cache check)                        │
│  │   │   ├── GORM Query (cache miss)                        │
│  │   │   └── Redis SET (cache write)                        │
│  │   └── CommentService.ListComments                        │
│  │       └── GORM Query                                     │
│  └── Response Assembly                                      │
└─────────────────────────────────────────────────────────────┘
```

---

### 3.10 日志：Zap

| 维度 | 分析 |
|------|------|
| **选型理由** | 高性能结构化日志库（0 内存分配），生产验证广泛，支持 Console/JSON 双格式输出，lumberjack 支持日志滚动与自动清理 |
| **组件** | `go.uber.org/zap` v1.27.1 + `gopkg.in/natefinch/lumberjack.v2` v2.2.1 |
| **特点** | 彩色控制台输出（开发）+ JSON 文件输出（采集），自动注入 TraceID/SpanID，异步写入不阻塞 |
| **备选方案** | Zerolog（API 不如 Zap 优雅）、Go slog（1.21 标准库新增但生态不成熟） |

---

### 3.11 认证鉴权：JWT

| 维度 | 分析 |
|------|------|
| **选型理由** | 无状态认证，适合分布式架构，无需集中 Session 存储；Access Token + Refresh Token 双令牌机制；支持黑名单（Redis）实现主动注销 |
| **令牌策略** | Access Token 15min 过期 + Refresh Token 7d 过期 + Redis 黑名单 |
| **现有实现** | `pkg/jwt/` 已实现完整的 JWT 中间件（Server/Client）、令牌对生成解析、黑名单 |

---

### 3.12 依赖注入：Google Wire

| 维度 | 分析 |
|------|------|
| **选型理由** | 编译时 DI（非运行时反射），类型安全，IDE 友好，自动生成初始化代码，性能无损 |
| **备选方案** | Uber FX（运行时 DI，复杂度过大）、手动注入（维护成本高） |

---

### 3.13 容器化与部署

| 组件 | 选型 | 说明 |
|------|------|------|
| 构建 | Docker Multi-stage Build | 第一阶段 Go 编译，第二阶段 alpine/debian-slim 运行，镜像 < 30MB |
| 编排 | Docker Compose（开发/测试） | 一键拉起所有基础设施（PG/Redis/NATS/etcd/Jaeger/MinIO） |
| 编排 | Kubernetes（生产可选） | 配合 Kustomize/Helm 管理部署 |
| CI/CD | GitHub Actions / GitLab CI | 自动构建、测试、镜像推送 |

---

## 4. 技术选型风险与对策

| 风险 | 影响 | 对策 |
|------|------|------|
| NATS 集群网络分区 | 消息丢失 | 启用 JetStream Exactly-Once，多副本 |
| PostgreSQL 单点故障 | 服务不可用 | 主从复制 + PgBouncer 连接池 + 自动 Failover（Patroni） |
| Redis 内存溢出 | 缓存雪崩 | 集群模式 + 内存淘汰策略（allkeys-lru）+ 熔断降级 |
| etcd 集群脑裂 | 服务发现异常 | 部署奇数节点（≥3），配置合理的 Election Timeout |
| MinIO 磁盘损坏 | 文件丢失 | 纠删码（EC）模式，最少 4 盘 |
| JWT 泄露 | 未授权访问 | 短 TTL + 黑名单机制 + IP 绑定（可选） |

---

## 5. 选型总结

本技术选型基于以下原则：

1. **Go 优先**：单一技术栈降低团队认知负担，Go 天然适合微服务
2. **CNCF 生态**：Kratos/NATS/etcd/Jaeger/OpenTelemetry 均为 CNCF 相关项目，社区活跃、文档丰富
3. **生产验证**：所选组件均在大型互联网公司（B站/Uber/腾讯/CloudFlare）有大规模生产实践
4. **渐进式**：可从单机 Docker Compose 起步，平滑演进到 Kubernetes 集群部署
5. **现有基础复用**：充分利用项目已有的 `pkg/` 基础设施代码，避免重复建设
