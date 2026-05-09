# Ley Blog — 系统设计文档

> 版本: v2.1 | 日期: 2026-05 | 状态: 设计中

---

## 1. 项目概述

Ley 是一个个人博客平台，支持文章发布、评论互动、文件管理、邮件通知、访问统计等核心功能。

**设计目标**：

- 合理的服务拆分，每个服务对应一个独立的 Bounded Context
- 数据一致性优先，同一 Context 内使用 DB 事务保证原子性
- API 优先设计，Proto 作为服务契约
- 纯异步服务（Notification、Analytics）通过 NATS 事件驱动，不阻塞核心流程
- 共享基础设施复用 `pkg/` 已有代码

---

## 2. 系统架构

### 2.1 架构图

```
                        ┌──────────────────────────────────┐
                        │         Client (Web/App)          │
                        └──────────────┬───────────────────┘
                                       │ HTTPS
                                       ▼
                        ┌──────────────────────────────┐
                        │        Gateway :8000          │
                        │  HTTP REST + gRPC-Gateway     │
                        │                              │
                        │  JWT 验证 · CORS · 限流      │
                        │  路由转发 · 静态资源代理      │
                        └──┬───────┬───────┬───────────┘
                           │gRPC   │gRPC   │gRPC
                           ▼       ▼       ▼
              ┌────────────┐ ┌──────────┐ ┌───────────────┐
              │Auth :9001  │ │Blog :9002│ │Analytics :9004│
              │            │ │          │ │               │
              │ ○ 注册     │ │ ○ 文章   │ │ ○ PV/UV 追踪  │
              │ ○ 登录     │ │ ○ 评论   │ │ ○ 热门文章    │
              │ ○ Token    │ │ ○ 标签   │ │ ○ 访问来源    │
              │ ○ 个人资料 │ │ ○ 分类   │ │ ○ 仪表盘 API  │
              │            │ │ ○ 搜索（预留）│ │               │
              │ PostgreSQL │ │ ○ 文件   │ │ PostgreSQL     │
              │   user.*   │ │          │ │   analytics.* │
              │            │ │PostgreSQL│ └───────────────┘
              │   Redis    │ │ article.*│
              │ 缓存+黑名单│ │ comment.*│       NATS
              └────────────┘ │          │ ───────────────┐
                             │  MinIO   │                │
                             │ 文件存储  │                ▼
                             │          │     ┌──────────────────┐
                             │  Redis   │     │Notification :9003│
                             │  缓存    │     │                  │
                             └──────────┘     │ ○ 评论回复通知   │
                                              │ ○ 新文章推送     │
        ┌──────────────────────────────────┐  │ ○ 注册欢迎邮件   │
        │          NATS JetStream           │  │ ○ 通知历史      │
        │  auth.* / article.* / comment.*  │  │                  │
        │  notification.* / analytics.*     │  │ PostgreSQL       │
        └──────────────────────────────────┘  │   notify.*      │
                                              │                  │
 ┌──────────────────────────────────────────┐ │  SMTP/SendGrid   │
 │              基础设施                      │ └──────────────────┘
 │                                          │
 │  etcd —— 服务注册与配置中心                │
 │  OpenTelemetry + Jaeger —— 链路追踪       │
 │  Prometheus —— 指标采集                   │
 └──────────────────────────────────────────┘
```

### 2.2 服务职责

| 服务 | gRPC | HTTP | 数据库 Schema | 外部依赖 | 职责 |
|------|------|------|--------------|---------|------|
| **Gateway** | — | 8000 | — | — | HTTP 入口，JWT 验证，路由转发，CORS，限流 |
| **Auth** | 9001 | 8001 | user | Redis（黑名单） | 注册/登录，JWT 签发，Token 刷新/黑名单，用户资料 |
| **Blog** | 9002 | 8002 | article, comment | Redis, MinIO | 文章CRUD、评论CRUD、标签/分类管理、搜索（预留）、点赞、文件上传 |
| **Notification** | 9003 | — | notify | SMTP/SendGrid | NATS 消费者：评论回复邮件、新文章推送、欢迎邮件、通知历史 |
| **Analytics** | 9004 | 8004 | analytics | — | 前端埋点采集 + NATS 消费者：PV/UV、热门文章、访问来源、仪表盘 |

### 2.3 服务拆分原则

| 服务 | 拆出理由 | 通信方式 |
|------|---------|---------|
| Auth | JWT 密钥隔离，用户数据与博客内容天然分离 | Gateway→Auth (gRPC) |
| Blog | 文章/评论/标签/分类高度耦合，同一事务保证计数一致性 | Gateway→Blog (gRPC) |
| Notification | 纯异步 NATS 消费者，依赖外部 SMTP，与核心业务零耦合 | NATS 消费 |
| Analytics | 写入密集（每次页面访问一条记录），与核心业务负载特征不同，可独立扩缩容 | HTTP 埋点 + NATS 消费 |

---

## 3. API 设计

### 3.1 Proto 目录结构

```
api/
├── common/v1/
│   └── common.proto          # 共享消息: TokenPair, UserInfo, AuthorInfo, Pagination
├── auth/v1/
│   ├── auth.proto             # AuthService RPC
│   └── auth_error.proto       # AuthErrorReason
├── blog/v1/
│   ├── blog.proto              # BlogService RPC
│   └── blog_error.proto        # BlogErrorReason
├── analytics/v1/
│   └── analytics.proto         # AnalyticsService RPC
└── notification/v1/
    └── notification.proto      # NotificationService RPC（内部查询接口）
```

### 3.2 AuthService — 认证服务 API

```
POST   /api/v1/auth/register     # 用户注册 → TokenPair + UserInfo（注册即登录）
POST   /api/v1/auth/login        # 登录 → TokenPair + UserInfo
POST   /api/v1/auth/refresh      # 刷新令牌 → 新 TokenPair
POST   /api/v1/auth/logout       # 登出（令牌加入黑名单）
GET    /api/v1/users/me          # 获取当前用户资料    [需认证]
PUT    /api/v1/users/me          # 更新当前用户资料    [需认证]
```

**请求/响应消息**：

```protobuf
message RegisterRequest {
  string username = 1;  // 3-32 字母数字下划线连字符，同时作为登录标识和显示名
  string email = 2;     // 合法邮箱
  string password = 3;  // 8-64 含大小写+数字
}
message RegisterReply {
  UserInfo user = 1;
  TokenPair token_pair = 2;
}

message LoginRequest {
  string account = 1;   // 用户名或邮箱
  string password = 2;
}
message LoginReply {
  UserInfo user = 1;
  TokenPair token_pair = 2;
}

message RefreshTokenRequest { string refresh_token = 1; }
message RefreshTokenReply {
  UserInfo user = 1;
  TokenPair token_pair = 2;
}

message LogoutRequest { string refresh_token = 1; }
message LogoutReply {}

message GetProfileRequest {}
message GetProfileReply { UserInfo user = 1; }

message UpdateProfileRequest {
  string avatar = 2;    // 最长 512
  string bio = 3;       // 最长 500
}
message UpdateProfileReply { UserInfo user = 1; }
```

### 3.3 BlogService — 博客服务 API

```
# === 文章 ===
POST   /api/v1/articles                       # 创建文章（草稿）      [需认证]
PUT    /api/v1/articles/{id}                  # 更新文章              [需认证]
DELETE /api/v1/articles/{id}                  # 软删除文章            [需认证]
GET    /api/v1/articles/{identifier}          # 获取文章（uuid或slug）
GET    /api/v1/articles                       # 文章列表（分页+过滤+排序）
POST   /api/v1/articles/{id}/publish          # 发布文章              [需认证]
POST   /api/v1/articles/{id}/archive          # 归档文章              [需认证]
GET    /api/v1/articles/search                # 全文搜索（预留，后续接入搜索引擎）

# === 点赞 ===
POST   /api/v1/articles/{id}/like             # 点赞（幂等）          [需认证]
DELETE /api/v1/articles/{id}/like             # 取消点赞（幂等）      [需认证]

# === 评论 ===
POST   /api/v1/articles/{id}/comments         # 发表评论              [需认证]
PUT    /api/v1/comments/{id}                  # 编辑评论              [需认证]
DELETE /api/v1/comments/{id}                  # 删除评论              [需认证]
GET    /api/v1/articles/{id}/comments         # 评论列表（树形）
POST   /api/v1/comments/{id}/approve          # 审核通过              [管理员]
POST   /api/v1/comments/{id}/spam             # 标记垃圾              [管理员]

# === 标签 ===
POST   /api/v1/tags                           # 创建标签              [管理员]
GET    /api/v1/tags                           # 全量标签列表
DELETE /api/v1/tags/{id}                      # 删除标签              [管理员]

# === 分类 ===
POST   /api/v1/categories                     # 创建分类              [管理员]
GET    /api/v1/categories                     # 分类树
PUT    /api/v1/categories/{id}                # 更新分类              [管理员]
DELETE /api/v1/categories/{id}                # 删除分类              [管理员]

# === 文件 ===
POST   /api/v1/files/upload                   # 上传文件              [需认证]
GET    /api/v1/files/{uuid}                   # 获取文件信息
DELETE /api/v1/files/{uuid}                   # 删除文件              [需认证]
GET    /api/v1/files                          # 文件列表              [需认证]
GET    /api/v1/files/presigned-upload         # 获取预签名上传URL     [需认证]

# === 站点配置 ===
GET    /api/v1/site/config                    # 获取站点配置（公开）
PUT    /api/v1/site/config                    # 更新站点配置           [管理员]
GET    /api/v1/site/backgrounds               # 背景图片列表（公开）
POST   /api/v1/site/backgrounds               # 上传背景图片           [管理员]
DELETE /api/v1/site/backgrounds/{id}          # 删除背景图片           [管理员]
PUT    /api/v1/site/backgrounds/{id}/active   # 设为当前背景           [管理员]
GET    /api/v1/site/music/playlist            # 获取歌单（公开）
PUT    /api/v1/site/music/playlist            # 更新歌单              [管理员]
```

**核心消息结构**：

```protobuf
message ArticleInfo {
  uint64 id = 1;
  string uuid = 2;
  string title = 3;
  string slug = 4;
  string content = 5;
  string excerpt = 6;
  string cover_image = 7;
  string status = 8;          // draft / published / archived
  uint64 author_id = 9;
  AuthorInfo author = 10;
  uint64 category_id = 11;
  CategoryInfo category = 12;
  repeated TagInfo tags = 13;
  int64 view_count = 14;
  int64 like_count = 15;
  int64 comment_count = 16;
  bool is_top = 17;
  bool is_liked = 18;
  string published_at = 19;   // RFC3339, 仅已发布文章有值
  string created_at = 20;
  string updated_at = 21;
}

message CommentNode {
  CommentInfo comment = 1;
  repeated CommentNode children = 2;
}

message TagInfo {
  uint64 id = 1;
  string name = 2;
  string slug = 3;
  int64 article_count = 4;
}

message CategoryInfo {
  uint64 id = 1;
  string name = 2;
  string slug = 3;
  string description = 4;
  uint64 parent_id = 5;
  int32 sort_order = 6;
  int64 article_count = 7;
  repeated CategoryInfo children = 8;
}

message FileInfo {
  string uuid = 1;
  string filename = 2;
  string mime_type = 3;
  int64 size = 4;
  string url = 5;
  string created_at = 6;
}

// === 站点配置 ===

message SiteConfig {
  // 站点信息
  string site_title = 1;
  string site_subtitle = 2;
  string site_description = 3;
  string site_logo = 4;
  string site_favicon = 5;
  // SEO
  string seo_keywords = 6;
  string seo_description = 7;
  // 社交链接
  string social_github = 8;
  string social_twitter = 9;
  string social_email = 10;
  // 页脚
  string footer_text = 11;
  string icp_number = 12;
  // 功能开关
  bool enable_comments = 13;
  bool enable_likes = 14;
  bool auto_approve_comments = 15;
}

message SiteBackground {
  uint64 id = 1;
  string uuid = 2;
  string filename = 3;
  string url = 4;
  bool is_active = 5;
  int32 sort_order = 6;
  string created_at = 7;
}

message MusicTrack {
  string title = 1;
  string artist = 2;
  string url = 3;         // 外链地址
  string cover_url = 4;   // 封面图
}

message MusicPlaylist {
  repeated MusicTrack tracks = 1;
}
```

### 3.4 NotificationService — 通知服务 API

Notification 服务主要作为 NATS 消费者运行，对外仅暴露通知历史的查询接口。

```
GET    /api/v1/notifications                  # 当前用户的通知列表（分页）  [需认证]
PUT    /api/v1/notifications/{id}/read        # 标记单条已读              [需认证]
PUT    /api/v1/notifications/read-all         # 全部标记已读              [需认证]
GET    /api/v1/notifications/unread-count     # 未读数量                  [需认证]
```

**消息结构**：

```protobuf
message NotificationInfo {
  uint64 id = 1;
  string type = 2;          // comment_reply / new_article / welcome
  string title = 3;
  string content = 4;
  bool is_read = 5;
  string target_url = 6;    // 点击后跳转的地址
  string created_at = 7;
}
```

### 3.5 AnalyticsService — 统计服务 API

```
# 前端埋点上报
POST   /api/v1/analytics/track                # 上报单次访问事件（无需认证）
POST   /api/v1/analytics/track-batch          # 批量上报

# 仪表盘查询
GET    /api/v1/analytics/dashboard            # 仪表盘概览（PV/UV/热门文章）  [管理员]
GET    /api/v1/analytics/articles/{id}/stats  # 单篇文章统计                  [原作者]
GET    /api/v1/analytics/referrers            # 访问来源                      [管理员]
```

**消息结构**：

```protobuf
message TrackRequest {
  string article_uuid = 1;   // 可为空（非文章页面）
  string path = 2;           // 页面路径
  string referrer = 3;       // 来源 URL
  string user_agent = 4;     // UA
  int32  duration = 5;       // 页面停留时长（秒），heartbeat 上报
}

message DashboardReply {
  int64 today_pv = 1;
  int64 today_uv = 2;
  int64 total_pv = 3;
  int64 total_uv = 4;
  repeated ArticleStat top_articles = 5;
}

message ArticleStat {
  string article_uuid = 1;
  string title = 2;
  int64 pv = 3;
  int64 uv = 4;
}
```

### 3.6 错误码枚举

**AuthErrorReason**：

| 枚举值 | HTTP | 说明 |
|--------|------|------|
| USERNAME_TAKEN | 409 | 用户名已被占用 |
| EMAIL_TAKEN | 409 | 邮箱已注册 |
| BAD_CREDENTIALS | 401 | 用户名或密码错误 |
| ACCOUNT_DISABLED | 403 | 账号已被禁用 |
| USER_NOT_FOUND | 404 | 用户不存在 |
| TOKEN_INVALID | 401 | 令牌无效或过期 |
| TOKEN_BLACKLISTED | 401 | 令牌已登出 |

**BlogErrorReason**：

| 枚举值 | HTTP | 说明 |
|--------|------|------|
| ARTICLE_NOT_FOUND | 404 | 文章不存在 |
| NOT_ARTICLE_OWNER | 403 | 非文章作者无权操作 |
| ALREADY_PUBLISHED | 409 | 文章已发布，不可重复操作 |
| SLUG_ALREADY_EXISTS | 409 | Slug 已被占用 |
| COMMENT_NOT_FOUND | 404 | 评论不存在 |
| NOT_COMMENT_OWNER | 403 | 非评论作者无权操作 |
| MAX_DEPTH_EXCEEDED | 422 | 嵌套深度超过限制 |
| TAG_NOT_FOUND | 404 | 标签不存在 |
| TAG_NAME_EXISTS | 409 | 标签名称已存在 |
| CATEGORY_NOT_FOUND | 404 | 分类不存在 |
| CATEGORY_HAS_CHILDREN | 409 | 存在子分类，不允许删除 |
| CATEGORY_HAS_ARTICLES | 409 | 分类下有文章，不允许删除 |
| FILE_NOT_FOUND | 404 | 文件不存在 |
| FILE_TOO_LARGE | 422 | 超过文件大小限制 |
| MIME_NOT_ALLOWED | 422 | 不允许的文件类型 |

---

## 4. 数据库设计

### 4.1 Schema 划分

所有表通过 GORM AutoMigrate 自动创建和维护，无需手动迁移脚本。Schema 通过 GORM TableName 指定。

| Schema | 所属服务 | 包含表 |
|--------|---------|--------|
| `user` | Auth | users |
| `article` | Blog | articles, articles_tags, articles_likes, tags, categories |
| `comment` | Blog | comments |
| `notify` | Notification | notifications |
| `analytics` | Analytics | page_views, daily_stats, article_stats |
| `audit` | 共享(可选) | audit_logs |

### 4.2 核心表结构

#### `user.users` — 用户表

| 列 | 类型 | 说明 |
|----|------|------|
| id | BIGSERIAL PK | 自增主键（内部使用） |
| uuid | VARCHAR(36) UNIQUE NOT NULL | 对外唯一标识（UUID v7） |
| username | VARCHAR(32) UNIQUE NOT NULL | 唯一索引 where deleted_at IS NULL |
| email | VARCHAR(255) UNIQUE NOT NULL | 唯一索引 where deleted_at IS NULL |
| password | VARCHAR(255) NOT NULL | bcrypt 哈希 |
| avatar | VARCHAR(512) DEFAULT '' | |
| bio | TEXT DEFAULT '' | |
| status | SMALLINT DEFAULT 0 | 0=正常 1=禁用 |
| role | VARCHAR(16) DEFAULT 'reader' | reader / author / admin |
| created_at | TIMESTAMPTZ DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ DEFAULT NOW() | |
| deleted_at | TIMESTAMPTZ | |

#### `article.articles` — 文章表

| 列 | 类型 | 说明 |
|----|------|------|
| id | BIGSERIAL PK | |
| uuid | VARCHAR(36) UNIQUE NOT NULL | UUID v7 |
| title | VARCHAR(200) NOT NULL | |
| slug | VARCHAR(200) UNIQUE NOT NULL | 唯一索引 where deleted_at IS NULL |
| content | TEXT NOT NULL | |
| excerpt | TEXT DEFAULT '' | |
| cover_image | VARCHAR(512) DEFAULT '' | |
| status | SMALLINT DEFAULT 0 | 0=draft 1=published 2=archived |
| author_id | BIGINT NOT NULL | |
| category_id | BIGINT | |
| view_count | BIGINT DEFAULT 0 | |
| like_count | BIGINT DEFAULT 0 | |
| comment_count | BIGINT DEFAULT 0 | |
| is_top | BOOLEAN DEFAULT false | |
| published_at | TIMESTAMPTZ | 首次发布时间（发布时写入，后续编辑不更新） |
| created_at | TIMESTAMPTZ DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ DEFAULT NOW() | |
| deleted_at | TIMESTAMPTZ | |

---

#### `article.tags` — 标签表

| 列 | 类型 | 说明 |
|----|------|------|
| id | BIGSERIAL PK | |
| name | VARCHAR(64) UNIQUE NOT NULL | 唯一索引 where deleted_at IS NULL |
| slug | VARCHAR(64) UNIQUE NOT NULL | |
| article_count | BIGINT DEFAULT 0 | 关联文章数 |
| created_at / updated_at / deleted_at | TIMESTAMPTZ | |

#### `article.categories` — 分类表

| 列 | 类型 | 说明 |
|----|------|------|
| id | BIGSERIAL PK | |
| name | VARCHAR(64) NOT NULL | |
| slug | VARCHAR(64) UNIQUE NOT NULL | |
| description | TEXT DEFAULT '' | |
| parent_id | BIGINT | NULL=根分类 |
| sort_order | INT DEFAULT 0 | |
| article_count | BIGINT DEFAULT 0 | |
| created_at / updated_at / deleted_at | TIMESTAMPTZ | |

#### `article.articles_tags` — 文章-标签关联表

| 列 | 类型 |
|----|------|
| id | BIGSERIAL PK |
| article_id | BIGINT NOT NULL |
| tag_id | BIGINT NOT NULL |
| UNIQUE(article_id, tag_id) WHERE deleted_at IS NULL | |
| created_at / updated_at / deleted_at | TIMESTAMPTZ |

#### `article.articles_likes` — 点赞记录表

| 列 | 类型 |
|----|------|
| id | BIGSERIAL PK |
| article_id | BIGINT NOT NULL |
| user_id | BIGINT NOT NULL |
| UNIQUE(article_id, user_id) WHERE deleted_at IS NULL | |
| created_at / updated_at / deleted_at | TIMESTAMPTZ |

#### `article.site_settings` — 站点配置表（单行）

| 列 | 类型 | 说明 |
|----|------|------|
| id | BIGSERIAL PK | 始终为 1（单行 upsert） |
| config | JSONB NOT NULL DEFAULT '{}' | 完整 SiteConfig JSON |
| updated_at | TIMESTAMPTZ DEFAULT NOW() | |

#### `article.site_backgrounds` — 背景图片表

| 列 | 类型 | 说明 |
|----|------|------|
| id | BIGSERIAL PK | |
| uuid | VARCHAR(36) UNIQUE NOT NULL | |
| filename | VARCHAR(255) NOT NULL | |
| url | VARCHAR(1024) NOT NULL | |
| is_active | BOOLEAN DEFAULT false | 同一时间最多一个 true |
| sort_order | INT DEFAULT 0 | |
| created_at | TIMESTAMPTZ DEFAULT NOW() | |

#### `comment.comments` — 评论表

| 列 | 类型 | 说明 |
|----|------|------|
| id | BIGSERIAL PK | |
| uuid | VARCHAR(36) UNIQUE NOT NULL | |
| article_id | BIGINT NOT NULL | |
| author_id | BIGINT NOT NULL | |
| parent_id | BIGINT | NULL=顶级评论 |
| depth | SMALLINT DEFAULT 0 | 预计算嵌套深度 |
| content | TEXT NOT NULL | |
| status | SMALLINT DEFAULT 0 | 0=pending 1=approved 2=spam 3=deleted |
| created_at / updated_at / deleted_at | TIMESTAMPTZ | |

索引：`idx_comments_article`(article_id, created_at ASC WHERE deleted_at IS NULL), `idx_comments_parent`(parent_id WHERE deleted_at IS NULL)

#### `notify.notifications` — 通知表

| 列 | 类型 | 说明 |
|----|------|------|
| id | BIGSERIAL PK | |
| uuid | VARCHAR(36) UNIQUE NOT NULL | |
| user_id | BIGINT NOT NULL | 接收者 |
| type | VARCHAR(32) NOT NULL | comment_reply / new_article / welcome |
| title | VARCHAR(256) NOT NULL | |
| content | TEXT DEFAULT '' | |
| is_read | BOOLEAN DEFAULT false | |
| target_url | VARCHAR(512) DEFAULT '' | 点击跳转 |
| created_at | TIMESTAMPTZ DEFAULT NOW() | |

索引：`idx_notify_user`(user_id, is_read, created_at DESC)

#### `analytics.page_views` — 页面访问明细表

| 列 | 类型 | 说明 |
|----|------|------|
| id | BIGSERIAL PK | |
| article_uuid | VARCHAR(36) | 可空（非文章页） |
| visitor_id | VARCHAR(64) NOT NULL | 匿名访客标识（UUID/指纹） |
| path | VARCHAR(512) NOT NULL | |
| referrer | VARCHAR(1024) DEFAULT '' | |
| duration | INT DEFAULT 0 | 停留秒数 |
| created_at | TIMESTAMPTZ DEFAULT NOW() | |

索引：`idx_pv_article`(article_uuid, created_at DESC), `idx_pv_time`(created_at DESC)

#### `analytics.daily_stats` — 每日聚合统计表

| 列 | 类型 | 说明 |
|----|------|------|
| date | DATE PK | |
| pv | BIGINT DEFAULT 0 | |
| uv | BIGINT DEFAULT 0 | |
| top_articles | JSONB DEFAULT '[]' | 当日热门文章 Top 10 |

#### `analytics.article_stats` — 文章累计统计表

| 列 | 类型 | 说明 |
|----|------|------|
| article_uuid | VARCHAR(36) PK | |
| pv | BIGINT DEFAULT 0 | |
| uv | BIGINT DEFAULT 0 | |
| last_viewed_at | TIMESTAMPTZ | |

### 4.3 关键改进点

| 改进 | 说明 |
|------|------|
| 新增 `article.published_at` | 首次发布时写入，后续编辑不更新 |
| 新增 `comment.depth` | 预计算嵌套深度，避免 N+1 递归查询 |
| 所有计数在事务中更新 | `article_count`、`like_count`、`comment_count` 与主操作在同一 DB 事务 |
| 全文搜索预留 | 不依赖 PostgreSQL 内置全文搜索，`search` 接口预留后续接入外部搜索引擎 |
| `notify.*` 独立 Schema | 通知数据与核心业务隔离 |
| `analytics.*` 独立 Schema | 统计写入密集，独立管理 |

---

## 5. 核心业务流程

### 5.1 用户注册

```
Client → Gateway → Auth.Register
  1. 校验输入（用户名格式、密码强度）
  2. 检查用户名/邮箱唯一性（应用层 + DB 唯一约束）
  3. bcrypt 哈希密码
  4. INSERT INTO user.users
  5. 返回 UserInfo
  6. [NATS] 发布 user.registered 事件 → Notification 消费 → 发送欢迎邮件
```

### 5.2 登录与 Token 刷新

```
登录:
  Client → Gateway → Auth.Login
    1. 按用户名或邮箱查找用户
    2. bcrypt 验证密码
    3. 检查账号状态 (active/disabled)
    4. 生成 TokenPair (AccessToken 15min + RefreshToken 7d)
    5. 返回 TokenPair + UserInfo

刷新令牌 (Token Rotation):
  Client → Gateway → Auth.RefreshToken
    1. 验证 RefreshToken 签名和类型
    2. 检查黑名单（防重放攻击）
    3. 查询用户状态
    4. 生成新的 TokenPair
    5. 旧 RefreshToken 加入黑名单（TTL = 剩余有效期）
    6. 返回新 TokenPair + UserInfo

登出:
  Client → Gateway → Auth.Logout
    1. 提取 Bearer token（来自请求 header）
    2. 验证 AccessToken 有效性
    3. AccessToken + RefreshToken 加入黑名单
```

### 5.3 创建并发布文章（关键流程）

```
创建草稿:
  Client → Gateway → Blog.CreateArticle
    1. Gateway 验证 JWT，注入 user_id 到 gRPC metadata
    2. Blog 从 context 提取 author_id
    3. 校验标题/内容长度
    4. 生成唯一 slug（拼音 + 数字后缀）
    5. 解析标签（FindOrCreate）
    6. DB 事务:
       BEGIN
         INSERT INTO articles (status=draft, ...)
         INSERT INTO articles_tags (article_id, tag_id)
         -- 草稿不更新 article_count
       COMMIT
    7. 返回 ArticleInfo

发布:
  Client → Gateway → Blog.PublishArticle
    1. 权限校验：当前用户 == 作者
    2. 状态校验：当前状态 != published
    3. DB 事务:
       BEGIN
         UPDATE articles SET status=published, published_at=NOW() WHERE id=?
         UPDATE categories SET article_count = article_count + 1 WHERE id=?
         UPDATE tags SET article_count = article_count + 1 WHERE id IN (?)
       COMMIT
    4. [NATS] 发布 article.published 事件
       → Notification 消费 → 发送新文章推送邮件

归档:
  Client → Gateway → Blog.ArchiveArticle
    DB 事务:
      BEGIN
        UPDATE articles SET status=archived WHERE id=?
        UPDATE categories SET article_count = GREATEST(article_count - 1, 0) WHERE id=?
        UPDATE tags SET article_count = GREATEST(article_count - 1, 0) WHERE id IN (?)
      COMMIT

删除:
  Client → Gateway → Blog.DeleteArticle
    DB 事务:
      BEGIN
        DELETE FROM articles_tags WHERE article_id=?
        DELETE FROM articles WHERE id=? (软删除)
        UPDATE categories SET article_count = GREATEST(article_count - 1, 0) WHERE id=?
        UPDATE tags SET article_count = GREATEST(article_count - 1, 0) WHERE id IN (?)
      COMMIT
```

### 5.4 创建评论

```
Client → Gateway → Blog.CreateComment
  1. 校验内容长度 (1-2000 字符，UTF-8)
  2. 若有 parent_id:
     a. 查询父评论获取 depth
     b. 校验 depth + 1 <= 5（最大嵌套深度）
     c. 新评论 depth = parent.depth + 1 （直接计算，不递归）
  3. DB 事务:
     BEGIN
       INSERT INTO comments (article_id, author_id, parent_id, depth, content, status=pending)
       UPDATE articles SET comment_count = comment_count + 1 WHERE id=?
     COMMIT
  4. [NATS] 发布 comment.created 事件
     → Notification 消费 → 发送文章作者评论通知
```

### 5.5 通知事件流

```
NATS 消费者 —— Notification 服务:

  user.registered  → 发送欢迎邮件
  article.published → 发送新文章通知给订阅者
  comment.created   → 发送"新评论"通知给文章作者
  comment.created (reply) → 发送"评论回复"通知给被回复者

通知持久化到 notify.notifications 表（可查询历史）
```

### 5.6 统计分析事件流

```
前端埋点（HTTP）:
  Client → Analytics.Track (POST /api/v1/analytics/track)
    → INSERT INTO analytics.page_views

定时聚合（Cron，每日凌晨）:
  Cron → Analytics 内部
    → INSERT INTO analytics.daily_stats (SELECT aggregation)
    → INSERT INTO analytics.article_stats ON CONFLICT UPDATE

仪表盘查询（HTTP）:
  GET /api/v1/analytics/dashboard
    → 读取 daily_stats (今日) + article_stats (热门)
```

---

## 6. 缓存策略

### 6.1 Cache-Aside 模式（复用 pkg/cache）

| 缓存 Key | 服务 | 内容 | TTL | 失效时机 |
|----------|------|------|-----|---------|
| `user:detail:{uuid}` | Auth | 用户详情 JSON | 30min | Update/Delete/UpdateStatus |
| `user:id:{id}` | Auth | id→uuid 映射 | 30min | Update/Delete/UpdateStatus |
| `article:detail:{uuid}` | Blog | 文章详情（含 Tags） | 10min | Create/Update/Delete |
| `article:slug:{slug}` | Blog | slug→uuid 映射 | 10min | Create/Update/Delete |
| `tag:all` | Blog | 全量标签 JSON | 60min | Create/Delete Tag |
| `category:tree` | Blog | 分类树 JSON | 60min | Create/Update/Delete Category |

### 6.2 防缓存穿透

- 查询不存在的数据时写入 `null` 标记（短 TTL: 2-5min）
- 缓存读取失败时降级到 DB（记录 Warn 日志）
- 缓存写入失败不阻塞主流程

---

## 7. 事件驱动（NATS JetStream）

### 7.1 事件主题

| 主题 | 发布者 | 消费者 |
|------|--------|--------|
| `user.registered` | Auth | **Notification**（欢迎邮件） |
| `user.updated` | Auth | —（预留） |
| `article.created` | Blog | —（预留） |
| `article.published` | Blog | **Notification**（新文章推送） |
| `article.archived` | Blog | —（预留） |
| `article.liked` | Blog | —（预留） |
| `article.viewed` | Blog | **Analytics**（浏览统计） |
| `comment.created` | Blog | **Notification**（评论通知） |
| `comment.approved` | Blog | **Notification**（审核通过通知） |

### 7.2 事件消费原则

- 所有事件发布采用 fire-and-forget 异步模式（不阻塞主流程）
- 发布失败记录 Error 日志
- 消费者崩溃不影响生产者

---

## 8. 目录结构

```
ley/
├── api/                        # Proto API 定义
│   ├── common/v1/common.proto
│   ├── auth/v1/auth.proto + auth_error.proto
│   ├── blog/v1/blog.proto + blog_error.proto
│   ├── notification/v1/notification.proto
│   └── analytics/v1/analytics.proto
│
├── app/                        # 微服务实现
│   ├── auth/                   # Auth 服务
│   │   ├── cmd/main.go, wire.go
│   │   └── internal/{biz,data,model,service,server,conf}
│   ├── blog/                   # Blog 服务
│   │   ├── cmd/main.go, wire.go
│   │   └── internal/{biz,data,model,service,server,conf}
│   ├── notification/           # Notification 服务（NATS 消费者）
│   │   ├── cmd/main.go, wire.go
│   │   └── internal/{biz,data,service,conf}
│   ├── analytics/              # Analytics 服务
│   │   ├── cmd/main.go, wire.go
│   │   └── internal/{biz,data,service,conf}
│   └── gateway/                # API 网关
│       ├── cmd/main.go, wire.go
│       └── internal/{service,server,conf}
│
├── pkg/                        # 共享库
│   ├── cache/ eventbus/ infra/ jwt/ log/ meta/ middleware/
│   ├── mq/ oss/ security/ task/ trace/ util/ testutil/
│
├── conf/                       # 共享 Proto 配置
│   ├── common.proto
│   └── types.proto
│
├── configs/                    # 运行时配置文件
│   ├── config.yaml             # 引导配置
│   └── auth.yaml / blog.yaml / notification.yaml / analytics.yaml
│
├── third_party/                # 第三方 Proto 依赖
├── Makefile / Dockerfile
├── go.mod / go.sum
└── docs/design.md
```

---

## 9. 站点配置设计（biz 层）

站点配置在 Blog 服务内实现，由 `SiteUseCase` 统一管理。本节聚焦 biz 层的业务模型和规则。

### 9.1 业务模型

```
SiteUseCase
├── repo (SiteRepo 接口，data 层实现)
├── cache (pkg/cache.Cache)
├── oss  (pkg/oss.OSS)
└── logger
```

**SiteRepo 接口**（biz 层定义）：

```go
type SiteRepo interface {
    // 站点配置（单行 JSONB）
    GetConfig(ctx context.Context) (*model.SiteSetting, error)
    SaveConfig(ctx context.Context, config map[string]interface{}) error

    // 背景图片
    CreateBackground(ctx context.Context, bg *model.SiteBackground, file io.Reader) error
    DeleteBackground(ctx context.Context, id uint) error
    ListBackgrounds(ctx context.Context) ([]*model.SiteBackground, error)
    SetActiveBackground(ctx context.Context, id uint) error
}
```

### 9.2 SiteUseCase — 核心业务规则

#### GetConfig

```
1. 读缓存 site:config
2. 命中返回缓存的 SiteConfig
3. 未命中 → 查 DB (site_settings WHERE id=1 的唯一行)
4. 回写缓存 (TTL 10min)
5. 返回配置
```

如果 `auto_approve_comments=true`，Blog 服务的 `CreateComment` 应直接设置 `status=approved` 而非 `pending`。

#### SaveConfig

```
1. 输入校验：URL 字段验证格式、长度限制
2. JSON Patch 合并到已有 config（不覆盖未传入的字段）
3. UPDATE site_settings SET config = ?, updated_at = NOW() WHERE id = 1
4. 删除缓存 site:config
```

**合并策略**：不直接替换整个 JSON，而是 `old_config merge new_fields`。例如只传 `{"site_title": "新标题"}` 时，其他 14 个字段保持不变。

```
old: {"site_title":"A", "enable_likes": true}
new: {"site_title":"B"}
   → {"site_title":"B", "enable_likes": true}
```

#### AddBackground

```
1. 校验文件类型（仅允许图片 MIME）
2. 上传到 MinIO → 获得 URL
3. INSERT INTO site_backgrounds (uuid, filename, url, sort_order)
4. 更新 sort_order = max(sort_order) + 1
5. 删除缓存 site:backgrounds
```

#### SetActiveBackground

```
1. DB 事务:
   BEGIN
     UPDATE site_backgrounds SET is_active = false WHERE is_active = true
     UPDATE site_backgrounds SET is_active = true WHERE id = ?
   COMMIT
2. 删除缓存 site:backgrounds
```

同一时间只有一个背景图为 active。事务保证原子切换。

#### ListBackgrounds

```
1. 读缓存 site:backgrounds (TTL 30min)
2. 命中返回
3. 未命中 → SELECT * FROM site_backgrounds ORDER BY sort_order
4. 回写缓存
```

#### UpdatePlaylist

歌单仅存储外部链接的 URL 数组，不做文件上传。

```
1. 校验每个 track.url 是合法 HTTP(S) 地址
2. 将 playlist ([]MusicTrack) 序列化为 JSON
3. 写入 site_settings.config.music_playlist 字段
4. 删除缓存 site:config
```

### 9.3 缓存键

| 键 | 内容 | TTL | 失效 |
|----|------|-----|------|
| `site:config` | SiteConfig JSON | 10min | SaveConfig / UpdatePlaylist |
| `site:backgrounds` | []SiteBackground JSON | 30min | AddBackground / DeleteBackground / SetActiveBackground |

### 9.4 Gateway 中间件

Gateway 对 `/api/v1/site/*` 的写方法（POST/PUT/DELETE）增加角色校验：

```
中间件栈: JWT 验证 → RoleCheck("admin") → 路由转发
```

---

## 10. 关键设计决策

### 决策 1：Auth 和 Blog 使用不同的 Redis DB

- Auth → Redis DB 0（JWT 黑名单 + 用户缓存）
- Blog → Redis DB 1（文章/标签/分类缓存）

### 决策 2：文章发布后才更新 article_count

- 草稿不计数 → 发布时 +1 → 归档/删除 -1

### 决策 3：评论默认 pending 状态

- 新评论需管理员 approve 后方可公开显示
- 可配置 `auto_approve: true` 跳过审核

### 决策 4：文件上传

- 推荐方案：客户端通过预签名 URL 直传 MinIO，上传完成后通知 Blog 服务关联文件记录
- 备选方案：Gateway 层处理 multipart，转发 bytes 给 Blog gRPC

### 决策 5：Notification 是纯 NATS 消费者

- 不暴露 HTTP 管理 API（仅通知历史查询）
- 邮件发送失败记录日志，不重试（避免邮件轰炸）

### 决策 6：Analytics 双写入路径

- 前端埋点（HTTP POST）：实时写入，用于热门文章计数
- NATS 消费（article.viewed）：来自 Blog 服务的事件，用于精确 view_count 更新

---

## 11. 实施计划

| 阶段 | 内容 | 产出 |
|------|------|---------|
| Phase 1 | Proto 定义 + 生成代码 | api/ 目录完整，`make api` 通过 |
| Phase 2 | Auth 服务 | 注册/登录/Token/资料 |
| Phase 3 | Blog 服务 | 文章+评论+标签+分类+文件（搜索预留） |
| Phase 4 | Gateway | HTTP 路由转发 + JWT 中间件 |
| Phase 5 | Notification 服务 | NATS 消费者 + 邮件发送 + 通知历史 |
| Phase 6 | Analytics 服务 | 埋点采集 + 定时聚合 + 仪表盘 |
| Phase 7 | 集成测试 + docker-compose | 端到端验证 |
