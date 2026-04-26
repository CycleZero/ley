# 系统设计方案

## 1. 系统架构总览

### 1.1 微服务拓扑

```
                              ┌─────────────┐
                              │   Browser    │
                              │   Client     │
                              └──────┬──────┘
                                     │ HTTP/REST
                                     │
                         ┌───────────┴───────────┐
                         │    API Gateway (BFF)   │
                         │    ley-gateway         │
                         │    :8000               │
                         │  ┌──────────────────┐  │
                         │  │ Auth Middleware   │  │
                         │  │ RateLimit         │  │
                         │  │ Router            │  │
                         │  │ gRPC Client Pool  │  │
                         │  └──────────────────┘  │
                         └───┬────┬─────┬─────┬──┘
                    gRPC    │    │     │     │
          ┌────────────────┘    │     │     └────────────────┐
          ▼                     ▼     ▼                      ▼
┌─────────────────┐  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐
│   ley-user      │  │  ley-post     │  │  ley-comment  │  │  ley-file     │
│   :9001 (gRPC)  │  │  :9002 (gRPC) │  │  :9003 (gRPC) │  │  :9004 (gRPC) │
│   :8001 (HTTP)  │  │  :8002 (HTTP) │  │  :8003 (HTTP) │  │  :8004 (HTTP) │
│                 │  │               │  │               │  │               │
│ ┌─────────────┐ │  │ ┌───────────┐ │  │ ┌───────────┐ │  │ ┌───────────┐ │
│ │ Auth        │ │  │ │ Post CRUD │ │  │ │ Comment   │ │  │ │ Upload    │ │
│ │ Profile     │ │  │ │ Tag/Cat   │ │  │ │ List/Post │ │  │ │ Download  │ │
│ │ Role        │ │  │ │ Search    │ │  │ │ Moderate  │ │  │ │ Presigned │ │
│ └─────────────┘ │  │ └───────────┘ │  │ └───────────┘ │  │ └───────────┘ │
└────────┬───────┘  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘
         │                 │                 │                 │
         └──────────┬──────┴────────┬────────┴────────┬────────┘
                    │               │                 │
              ┌─────┴─────┐   ┌────┴────┐     ┌──────┴──────┐
              │ PostgreSQL │   │  Redis  │     │    MinIO    │
              │            │   │         │     │             │
              └─────┬──────┘   └────┬────┘     └──────┬──────┘
                    │               │                 │
              ┌─────┴──────┐  ┌────┴────┐     ┌──────┴──────┐
              │    etcd    │  │   NATS  │     │   Jaeger    │
              │ (Registry) │  │ (Events)│     │  (Tracing)  │
              └────────────┘  └─────────┘     └─────────────┘
```

### 1.2 服务分解原则

| 原则 | 描述 |
|------|------|
| **高内聚低耦合** | 每个服务拥有独立的数据存储，通过 API/事件通信 |
| **领域驱动** | 按业务领域边界划分服务（用户域/内容域/评论域/文件域） |
| **数据主权** | 各服务独享数据库 Schema，禁止跨服务直连数据库 |
| **弹性设计** | 服务间通过 gRPC + 超时/重试/熔断，异步通过 NATS 事件驱动 |

---

## 2. 分层架构设计（Kratos Clean Architecture）

每个微服务统一遵循四层架构：

```
┌─────────────────────────────────────────────────────────────┐
│                    api/   (Proto 定义)                        │
│              Service Interface (gRPC + HTTP)                  │
├─────────────────────────────────────────────────────────────┤
│                 internal/service/                            │
│         ┌──────────────────────────────────┐                │
│         │  协议适配层                        │                │
│         │  - Proto 消息 ↔ 业务模型 转换       │                │
│         │  - 参数校验 (Validate)              │                │
│         │  - 错误码映射                       │                │
│         └─────────────┬────────────────────┘                │
├───────────────────────┼─────────────────────────────────────┤
│                 internal/biz/                                │
│         ┌─────────────┴────────────────────┐                │
│         │  业务逻辑层 (UseCase)               │                │
│         │  - 业务规则校验                     │                │
│         │  - 领域模型定义                     │                │
│         │  - Repo 接口定义（依赖倒置）          │                │
│         │  - 事务编排 (Unit of Work)          │                │
│         │  - 事件发布 (Event Publisher)       │                │
│         └─────────────┬────────────────────┘                │
├───────────────────────┼─────────────────────────────────────┤
│                 internal/data/                               │
│         ┌─────────────┴────────────────────┐                │
│         │  数据访问层 (Repo 实现)             │                │
│         │  - GORM DB 操作                    │                │
│         │  - Redis 缓存操作                   │                │
│         │  - NATS 消息发布/订阅               │                │
│         │  - MinIO 文件操作                   │                │
│         └──────────────────────────────────┘                │
├─────────────────────────────────────────────────────────────┤
│                 internal/conf/    (配置)                      │
│                 internal/server/ (传输层)                     │
└─────────────────────────────────────────────────────────────┘
```

**依赖方向：service → biz → data**，biz 层定义接口，data 层实现接口。（依赖倒置原则）

---

## 3. 服务详细设计

### 3.1 ley-gateway（API 网关）

**职责：** 请求路由、认证验证、限流、CORS、请求/响应聚合

| 配置项 | 值 |
|--------|-----|
| HTTP 端口 | 8000 |
| gRPC 客户端连接池 | 4 服务 × 4 连接 = 16 |

**路由表：**

| 路径 | 方法 | 目标服务 | gRPC Method | 认证 |
|------|------|---------|-------------|------|
| `/api/v1/auth/register` | POST | ley-user | UserService.Register | 否 |
| `/api/v1/auth/login` | POST | ley-user | UserService.Login | 否 |
| `/api/v1/auth/refresh` | POST | ley-user | UserService.RefreshToken | 否 |
| `/api/v1/users/me` | GET | ley-user | UserService.GetProfile | 是 |
| `/api/v1/users/me` | PUT | ley-user | UserService.UpdateProfile | 是 |
| `/api/v1/posts` | GET | ley-post | PostService.ListPosts | 否 |
| `/api/v1/posts` | POST | ley-post | PostService.CreatePost | 是 |
| `/api/v1/posts/{id}` | GET | ley-post | PostService.GetPost | 否 |
| `/api/v1/posts/{id}` | PUT | ley-post | PostService.UpdatePost | 是 |
| `/api/v1/posts/{id}` | DELETE | ley-post | PostService.DeletePost | 是 |
| `/api/v1/posts/{id}/comments` | GET | ley-comment | CommentService.ListComments | 否 |
| `/api/v1/posts/{id}/comments` | POST | ley-comment | CommentService.CreateComment | 是 |
| `/api/v1/comments/{id}` | DELETE | ley-comment | CommentService.DeleteComment | 是 |
| `/api/v1/files/upload` | POST | ley-file | FileService.Upload | 是 |
| `/api/v1/tags` | GET | ley-post | PostService.ListTags | 否 |
| `/api/v1/categories` | GET | ley-post | PostService.ListCategories | 否 |

---

### 3.2 ley-user（用户服务）

**职责：** 用户注册/登录、Token 管理、用户资料 CRUD、角色管理

**API 定义（Proto）：**

```protobuf
service UserService {
  rpc Register(RegisterRequest) returns (RegisterReply);
  rpc Login(LoginRequest) returns (LoginReply);
  rpc RefreshToken(RefreshTokenRequest) returns (LoginReply);
  rpc GetProfile(GetProfileRequest) returns (GetProfileReply);
  rpc UpdateProfile(UpdateProfileRequest) returns (UpdateProfileReply);
  rpc GetUser(GetUserRequest) returns (GetUserReply);           // 内部
  rpc BatchGetUsers(BatchGetUsersRequest) returns (BatchGetUsersReply); // 内部
}
```

**领域模型：**

```
User {
  ID        int64     // 自增主键
  UUID      string    // 对外 ID（UUID v7）
  Username  string    // 唯一，3-32 字符
  Email     string    // 唯一
  Password  string    // bcrypt hash
  Nickname  string    // 显示名称
  Avatar    string    // MinIO URL
  Bio       string    // 个人简介
  Status    int       // 0=正常 1=禁用 2=未验证
  Role      string    // admin/author/reader
  CreatedAt time.Time
  UpdatedAt time.Time
  DeletedAt gorm.DeletedAt
}
```

**密码策略：**
- 最小长度 8 位，必须包含大小写字母+数字
- bcrypt cost=12 哈希
- 5 次失败后账号临时锁定 15 分钟（Redis 计数器）

**Token 策略：**

```
Access Token:
  ┌────────────────────────────────────────────┐
  │ Header: {alg: HS256, typ: JWT}            │
  │ Payload: {                                 │
  │   sub: user_id,                            │
  │   usn: username,                           │
  │   role: role,                              │
  │   iat: issued_at,                          │
  │   exp: issued_at + 15min                   │
  │   jti: unique_token_id                     │
  │ }                                          │
  │ Signature: HMAC-SHA256                     │
  └────────────────────────────────────────────┘

Refresh Token:
  ┌────────────────────────────────────────────┐
  │ Same structure, exp: issued_at + 7d        │
  │ Additionally stored in Redis for revocation │
  └────────────────────────────────────────────┘
```

---

### 3.3 ley-post（文章服务）

**职责：** 文章 CRUD、分类管理、标签管理、草稿管理、全文搜索

**API 定义：**

```protobuf
service PostService {
  rpc CreatePost(CreatePostRequest) returns (CreatePostReply);
  rpc UpdatePost(UpdatePostRequest) returns (UpdatePostReply);
  rpc DeletePost(DeletePostRequest) returns (DeletePostReply);
  rpc GetPost(GetPostRequest) returns (GetPostReply);
  rpc ListPosts(ListPostsRequest) returns (ListPostsReply);
  rpc SearchPosts(SearchPostsRequest) returns (SearchPostsReply);
  rpc PublishPost(PublishPostRequest) returns (PublishPostReply);
  rpc LikePost(LikePostRequest) returns (LikePostReply);
  rpc UnlikePost(UnlikePostRequest) returns (UnlikePostReply);
  
  rpc CreateTag(CreateTagRequest) returns (CreateTagReply);
  rpc ListTags(ListTagsRequest) returns (ListTagsReply);
  rpc DeleteTag(DeleteTagRequest) returns (DeleteTagReply);
  
  rpc CreateCategory(CreateCategoryRequest) returns (CreateCategoryReply);
  rpc ListCategories(ListCategoriesRequest) returns (ListCategoriesReply);
  rpc DeleteCategory(DeleteCategoryRequest) returns (DeleteCategoryReply);
  
  rpc IncrementViewCount(IncrementViewCountRequest) returns (IncrementViewCountReply); // 内部
}
```

**领域模型：**

```
Post {
  ID           int64
  UUID         string       // 对外 ID
  Title        string       // 标题，2-200 字符
  Slug         string       // URL 友好标识，唯一
  Content      string       // Markdown 正文
  Excerpt      string       // 摘要（自动生成或手动填写）
  CoverImage   string       // 封面图 MinIO URL
  Status       PostStatus   // draft/published/archived
  AuthorID     int64        // 作者 (User.ID)
  CategoryID   *int64       // 分类（可选）
  Tags         []Tag        // 多对多
  ViewCount    int64        // 浏览数
  LikeCount    int64        // 点赞数
  CommentCount int64        // 评论数
  IsTop        bool         // 是否置顶
  PublishedAt  *time.Time   // 发布时间
  CreatedAt    time.Time
  UpdatedAt    time.Time
  DeletedAt    gorm.DeletedAt
}

Tag {
  ID   int64
  Name string               // 标签名，唯一
  Slug string               // URL 友好标识
}

Category {
  ID          int64
  Name        string        // 分类名
  Slug        string        // URL 友好标识
  Description string        // 分类描述
  ParentID    *int64        // 父分类（支持层级）
  SortOrder   int           // 排序
  PostCount   int64         // 文章数
}
```

**文章状态流转：**

```
  ┌──────┐    Save     ┌──────────┐    Publish    ┌─────────────┐
  │ Draft │ ────────→  │ Draft    │ ───────────→  │ Published   │
  │ (new) │            │ (editing)│               │             │
  └──────┘            └──────────┘               └──────┬──────┘
                                                        │
                                              Archive ←─┘
                                                        │
                                              ┌─────────┴──────┐
                                              │ Archived       │
                                              └────────────────┘
```

**全文搜索方案：**

使用 PostgreSQL 内置的 `tsvector` + `tsquery`，配合 GIN 索引：

```sql
-- 生成搜索向量列（自动更新）
ALTER TABLE posts ADD COLUMN search_vector tsvector
  GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(title, '')), 'A') ||
    setweight(to_tsvector('simple', coalesce(content, '')), 'B')
  ) STORED;

-- GIN 索引（加速搜索）
CREATE INDEX idx_posts_search ON posts USING GIN(search_vector);

-- 查询示例
SELECT * FROM posts
WHERE search_vector @@ plainto_tsquery('simple', 'kratos framework')
ORDER BY ts_rank(search_vector, plainto_tsquery('simple', 'kratos framework')) DESC;
```

**缓存策略：**

| 缓存 Key | 内容 | TTL | 更新策略 |
|----------|------|-----|---------|
| `post:detail:{slug}` | 文章详情 | 10min | 发布/更新时主动删除 |
| `post:list:page:{n}:size:{s}:{filters}` | 文章列表 | 5min | 发布新文章时主动删除 |
| `post:tag:all` | 全部标签 | 60min | 增删标签时主动删除 |
| `post:category:all` | 全部分类 | 60min | 增删分类时主动删除 |
| `post:view:{id}` | 浏览计数 | 持久 | 定时批量刷 DB（每 5min） |
| `post:like:{id}:{user_id}` | 用户点赞状态 | 30min | 点赞/取消时更新 |

---

### 3.4 ley-comment（评论服务）

**职责：** 评论 CRUD、回复嵌套、内容审核、垃圾过滤

**API 定义：**

```protobuf
service CommentService {
  rpc CreateComment(CreateCommentRequest) returns (CreateCommentReply);
  rpc GetComment(GetCommentRequest) returns (GetCommentReply);
  rpc ListComments(ListCommentsRequest) returns (ListCommentsReply);
  rpc DeleteComment(DeleteCommentRequest) returns (DeleteCommentReply);
  rpc ApproveComment(ApproveCommentRequest) returns (ApproveCommentReply); // 管理员
}
```

**领域模型：**

```
Comment {
  ID        int64
  UUID      string
  PostID    int64        // 所属文章
  AuthorID  int64        // 评论者
  ParentID  *int64       // 父评论（NULL=顶级评论）
  Content   string       // 评论内容
  Status    CommentStatus // pending/approved/spam/deleted
  CreatedAt time.Time
  UpdatedAt time.Time
  DeletedAt gorm.DeletedAt
}
```

**评论树构建：**

采用 `parent_id` 方案，在应用层（biz）通过一次查询获取所有评论，内存中构建树结构。对于评论量大的文章（>100 条），采用懒加载子评论。

**内容审核策略：**

| 策略 | 实现 |
|------|------|
| 敏感词过滤 | Trie 树 + AC 自动机（内置词库） |
| 垃圾评论检测 | 频率限制（同一 IP 每分钟 ≤ 3 条）+ 内容相似度 |
| 链接检测 | 正则匹配 URL，含链接的评论自动 pending |
| 审核模式 | 首次评论审核 / 全量审核 / 关闭审核（配置化） |

**事件发布：**

- `comment.created`：评论创建 → 通知文章作者 + 更新文章评论计数
- `comment.approved`：评论通过审核 → 通知评论者

---

### 3.5 ley-file（文件服务）

**职责：** 文件上传、下载、缩略图生成、预签名 URL

**API 定义：**

```protobuf
service FileService {
  rpc Upload(UploadRequest) returns (UploadReply);
  rpc GetDownloadURL(GetDownloadURLRequest) returns (GetDownloadURLReply);
  rpc GetPresignedPutURL(GetPresignedPutURLRequest) returns (GetPresignedPutURLReply);
  rpc DeleteFile(DeleteFileRequest) returns (DeleteFileReply);
  rpc ListFiles(ListFilesRequest) returns (ListFilesReply);
}
```

**领域模型：**

```
File {
  ID         int64
  UUID       string
  UserID     int64        // 上传者
  Bucket     string       // MinIO Bucket
  ObjectKey  string       // MinIO Object Key
  Filename   string       // 原始文件名
  MimeType   string       // Content-Type
  Size       int64        // 文件大小（字节）
  Width      int          // 图片宽度（仅图片）
  Height     int          // 图片高度（仅图片）
  MD5Hash    string       // 文件 MD5
  URL        string       // 访问 URL
  CreatedAt  time.Time
  DeletedAt  gorm.DeletedAt
}
```

**上传流程：**

```
Client                          ley-file                        MinIO
  │                                │                              │
  │  POST /api/v1/files/upload     │                              │
  │  (multipart/form-data)         │                              │
  │ ─────────────────────────────→ │                              │
  │                                │  校验文件大小(<10MB)           │
  │                                │  校验 MIME Type               │
  │                                │  计算 MD5                     │
  │                                │  生成 ObjectKey               │
  │                                │  (bucket/YYYY/MM/DD/uuid.ext) │
  │                                │                              │
  │                                │  PutObject(bucket,key,reader) │
  │                                │ ───────────────────────────→ │
  │                                │                              │
  │                                │  ← 成功 (etag)               │
  │                                │                              │
  │                                │  INSERT INTO files            │
  │                                │  发布 file.uploaded 事件      │
  │                                │                              │
  │  ← {file_id, url, ...}        │                              │
```

**文件大小限制：**

| 类型 | 限制 | 允许扩展名 |
|------|------|-----------|
| 图片 | 10 MB | jpg/jpeg/png/gif/webp/svg |
| 附件 | 50 MB | pdf/doc/docx/txt/md/zip |
| 头像 | 2 MB | jpg/jpeg/png/webp |

---

## 4. 数据存储设计

### 4.1 PostgreSQL Schema 设计

```
Database: ley
├── SCHEMA user
│   ├── users
│   ├── roles
│   └── user_roles
├── SCHEMA post
│   ├── posts
│   ├── posts_tags (关联表)
│   ├── tags
│   ├── categories
│   └── post_likes
├── SCHEMA comment
│   └── comments
└── SCHEMA file
    └── files
```

### 4.2 核心表 DDL

```sql
-- ===================== User Schema =====================
CREATE SCHEMA IF NOT EXISTS "user";

CREATE TABLE "user".users (
    id          BIGSERIAL PRIMARY KEY,
    uuid        VARCHAR(36) NOT NULL UNIQUE,
    username    VARCHAR(32) NOT NULL UNIQUE,
    email       VARCHAR(255) NOT NULL UNIQUE,
    password    VARCHAR(255) NOT NULL,          -- bcrypt hash
    nickname    VARCHAR(64) NOT NULL DEFAULT '',
    avatar      VARCHAR(512) NOT NULL DEFAULT '',
    bio         TEXT NOT NULL DEFAULT '',
    status      SMALLINT NOT NULL DEFAULT 0,    -- 0=正常 1=禁用
    role        VARCHAR(16) NOT NULL DEFAULT 'reader',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX idx_users_username ON "user".users(username) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_email ON "user".users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_uuid ON "user".users(uuid);

-- ===================== Post Schema =====================
CREATE SCHEMA IF NOT EXISTS "post";

CREATE TABLE "post".categories (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(64) NOT NULL,
    slug        VARCHAR(64) NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    parent_id   BIGINT,
    sort_order  INT NOT NULL DEFAULT 0,
    post_count  BIGINT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE "post".tags (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(64) NOT NULL UNIQUE,
    slug        VARCHAR(64) NOT NULL UNIQUE,
    post_count  BIGINT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE "post".posts (
    id              BIGSERIAL PRIMARY KEY,
    uuid            VARCHAR(36) NOT NULL UNIQUE,
    title           VARCHAR(200) NOT NULL,
    slug            VARCHAR(200) NOT NULL UNIQUE,
    content         TEXT NOT NULL DEFAULT '',
    excerpt         TEXT NOT NULL DEFAULT '',
    cover_image     VARCHAR(512) NOT NULL DEFAULT '',
    status          SMALLINT NOT NULL DEFAULT 0,    -- 0=draft 1=published 2=archived
    author_id       BIGINT NOT NULL,                -- 不设 FK，不跨库约束
    category_id     BIGINT,
    view_count      BIGINT NOT NULL DEFAULT 0,
    like_count      BIGINT NOT NULL DEFAULT 0,
    comment_count   BIGINT NOT NULL DEFAULT 0,
    is_top          BOOLEAN NOT NULL DEFAULT FALSE,
    search_vector   TSVECTOR,
    published_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

-- 搜索向量自动生成列
ALTER TABLE "post".posts ADD COLUMN search_vector tsvector
  GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(title, '')), 'A') ||
    setweight(to_tsvector('simple', coalesce(content, '')), 'B')
  ) STORED;

CREATE INDEX idx_posts_slug ON "post".posts(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_posts_status ON "post".posts(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_posts_author ON "post".posts(author_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_posts_category ON "post".posts(category_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_posts_search ON "post".posts USING GIN(search_vector);
CREATE INDEX idx_posts_published ON "post".posts(published_at DESC) WHERE status = 1 AND deleted_at IS NULL;
CREATE INDEX idx_posts_created ON "post".posts(created_at DESC);

CREATE TABLE "post".posts_tags (
    post_id  BIGINT NOT NULL,
    tag_id   BIGINT NOT NULL,
    PRIMARY KEY (post_id, tag_id)
);

CREATE TABLE "post".post_likes (
    post_id   BIGINT NOT NULL,
    user_id   BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (post_id, user_id)
);

-- ===================== Comment Schema =====================
CREATE SCHEMA IF NOT EXISTS "comment";

CREATE TABLE "comment".comments (
    id          BIGSERIAL PRIMARY KEY,
    uuid        VARCHAR(36) NOT NULL UNIQUE,
    post_id     BIGINT NOT NULL,
    author_id   BIGINT NOT NULL,
    parent_id   BIGINT,
    content     TEXT NOT NULL,
    status      SMALLINT NOT NULL DEFAULT 0,    -- 0=pending 1=approved 2=spam 3=deleted
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX idx_comments_post ON "comment".comments(post_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_comments_parent ON "comment".comments(parent_id) WHERE deleted_at IS NULL;

-- ===================== File Schema =====================
CREATE SCHEMA IF NOT EXISTS "file";

CREATE TABLE "file".files (
    id          BIGSERIAL PRIMARY KEY,
    uuid        VARCHAR(36) NOT NULL UNIQUE,
    user_id     BIGINT NOT NULL,
    bucket      VARCHAR(64) NOT NULL,
    object_key  VARCHAR(512) NOT NULL,
    filename    VARCHAR(255) NOT NULL,
    mime_type   VARCHAR(128) NOT NULL,
    size        BIGINT NOT NULL,
    width       INT NOT NULL DEFAULT 0,
    height      INT NOT NULL DEFAULT 0,
    md5_hash    VARCHAR(32) NOT NULL DEFAULT '',
    url         VARCHAR(512) NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX idx_files_user ON "file".files(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_files_bucket ON "file".files(bucket, created_at DESC) WHERE deleted_at IS NULL;
```

### 4.3 Redis 数据结构设计

```
┌──────────────────────────────────────────────────────────────────┐
│ Key Pattern                        │ Type     │ TTL      │ 用途    │
├────────────────────────────────────┼──────────┼──────────┼─────────┤
│ cache:post:detail:{slug}           │ String   │ 10min    │ 文章缓存 │
│ cache:post:list:{page}:{size}:{f}  │ String   │ 5min     │ 列表缓存 │
│ cache:post:view:{id}               │ String   │ 持久      │ 浏览计数 │
│ cache:post:like:set:{id}           │ Set      │ 持久      │ 点赞用户 │
│ cache:tag:all                      │ String   │ 60min    │ 标签缓存 │
│ cache:category:all                 │ String   │ 60min    │ 分类缓存 │
│ session:access:{user_id}           │ String   │ 15min    │ 在线状态 │
│ session:refresh:{user_id}:{jti}    │ String   │ 7d       │ 刷新令牌 │
│ jwt:blacklist:{jti}                │ String   │ 15min    │ JWT 黑名单│
│ rate:login:{ip}                    │ String   │ 15min    │ 登录限流 │
│ rate:comment:{ip}                  │ String   │ 1min     │ 评论限流 │
│ rate:api:{ip}:{path}               │ String   │ 1min     │ API 限流 │
│ lock:{resource}:{id}               │ String   │ 30s      │ 分布式锁 │
│ view:flush:queue                   │ List     │ 持久      │ 浏览计数队列│
└──────────────────────────────────────────────────────────────────┘
```

---

## 5. 事件驱动架构设计

### 5.1 NATS JetStream 流设计

```
┌─────────────────────────────────────────────────────────────────┐
│ Stream: EVENTS-POST                                              │
│ Subjects: post.created, post.updated, post.published,            │
│           post.deleted, post.liked, post.unliked                 │
│ Retention: Limits (MaxAge=7d, MaxMsgs=1M)                       │
│ Storage: File                                                   │
│ Replicas: 1 (dev) / 3 (prod)                                    │
│ ACK Policy: Explicit                                            │
├─────────────────────────────────────────────────────────────────┤
│ Stream: EVENTS-COMMENT                                           │
│ Subjects: comment.created, comment.approved, comment.deleted     │
│ Retention: Limits (MaxAge=7d, MaxMsgs=1M)                       │
├─────────────────────────────────────────────────────────────────┤
│ Stream: EVENTS-USER                                              │
│ Subjects: user.registered, user.updated                          │
│ Retention: Limits (MaxAge=30d, MaxMsgs=100K)                    │
├─────────────────────────────────────────────────────────────────┤
│ Stream: EVENTS-FILE                                              │
│ Subjects: file.uploaded, file.deleted                            │
│ Retention: Limits (MaxAge=7d, MaxMsgs=100K)                     │
├─────────────────────────────────────────────────────────────────┤
│ Stream: EVENTS-NOTIFICATION                                      │
│ Subjects: notification.email, notification.push                  │
│ Retention: Limits (MaxAge=3d, MaxMsgs=500K)                     │
│ Consumer: notification-worker (Durable Pull)                    │
└─────────────────────────────────────────────────────────────────┘
```

### 5.2 事件流编排示例

**场景：用户发布文章**

```
ley-post (CreatePost)
  │
  ├── 1. 保存文章到 PostgreSQL (post schema)
  ├── 2. 写入 Redis 缓存
  ├── 3. 发布事件到 NATS:
  │
  ├── post.created {post_id, author_id, title, tags, published_at}
  │
  ▼

ley-comment 订阅 post.created
  │ (初始化文章评论计数)

ley-notification 订阅 post.created
  │ (给订阅者发送邮件通知)
```

**场景：文章被评论**

```
ley-comment (CreateComment)
  │
  ├── 1. 保存评论到 PostgreSQL (comment schema)
  ├── 2. 发布事件到 NATS:
  │
  ├── comment.created {comment_id, post_id, author_id, content}
  │
  ▼

ley-post 订阅 comment.created
  ├── UPDATE posts SET comment_count = comment_count + 1

ley-notification 订阅 comment.created
  ├── 推送通知给文章作者
  └── 推送邮件通知
```

### 5.3 消费语义保证

| 场景 | 语义 | 实现 |
|------|------|------|
| 文章创建 | At-Least-Once | NATS ACK + 幂等 Key（post_id+version） |
| 评论计数更新 | At-Least-Once | UPDATE ... WHERE version = ? (乐观锁) |
| 通知发送 | At-Least-Once | NATS ACK + 数据库中已发送标记 |
| 浏览计数更新 | At-Most-Once | Redis 批量 + 定时刷 DB（可丢失） |

---

## 6. 安全设计

### 6.1 认证流程

```
┌──────────┐         ┌──────────┐         ┌──────────┐
│  Client  │         │ Gateway  │         │ ley-user │
└────┬─────┘         └────┬─────┘         └────┬─────┘
     │                    │                     │
     │ POST /auth/login   │                     │
     │ {username,password}│                     │
     │───────────────────→│                     │
     │                    │ gRPC Login          │
     │                    │────────────────────→│
     │                    │                     │ 验证密码
     │                    │                     │ 生成 TokenPair
     │                    │                     │ 存储 RefreshToken → Redis
     │                    │←────────────────────│
     │← {access,refresh}  │                     │
     │                    │                     │
     │ GET /users/me      │                     │
     │ Authorization:     │                     │
     │ Bearer <access>    │                     │
     │───────────────────→│                     │
     │                    │ 验证 AccessToken    │
     │                    │ (本地 JWT 解析)      │
     │                    │ 检查黑名单 (Redis)   │
     │                    │ gRPC GetProfile     │
     │                    │────────────────────→│
     │                    │←────────────────────│
     │← {user_profile}    │                     │
```

### 6.2 授权模型

采用 **RBAC**（基于角色的访问控制）：

| 角色 | 权限 |
|------|------|
| **reader** | 浏览文章、发表评论、点赞 |
| **author** | reader 权限 + 发布文章、管理自有文章 |
| **admin** | author 权限 + 管理所有文章/评论/用户/标签/分类 |

中间件层面基于 Kratos Middleware 注入 `UserID`、`UserName`、`Role` 到 Context，Biz 层进行权限校验。

### 6.3 安全防护清单

| 防护项 | 实现手段 |
|--------|---------|
| SQL 注入 | GORM 参数化查询 |
| XSS | 前端渲染侧转义，后端存储 Markdown（非 HTML） |
| CSRF | Bearer Token 认证天然免疫（非 Cookie 模式） |
| 暴力破解 | Redis 计数器，IP + 用户名维度限流 |
| DDoS | Nginx 反向代理限流 + Kratos Ratelimit 中间件 |
| 敏感数据加密 | bcrypt 存储密码，TLS 传输加密 |
| 文件上传漏洞 | MIME Type 白名单校验，File Magic Number 检测 |
| IDOR | 非公开资源查询绑定 `author_id` 或 `user_id` |
| CORS | Gateway 层面控制允许的 Origin |

---

## 7. 可观测性设计

### 7.1 三大支柱

```
┌─────────────────────────────────────────────────────────────────┐
│  可观测性三支柱                                                   │
├──────────────┬──────────────────┬───────────────────────────────┤
│  日志 (Logs) │ 指标 (Metrics)   │ 追踪 (Traces)                  │
├──────────────┼──────────────────┼───────────────────────────────┤
│ Zap          │ OpenTelemetry    │ OpenTelemetry + Jaeger        │
│ JSON 格式     │ Metrics (可选)   │ OTLP HTTP Export              │
│ 文件滚动      │ Prometheus +     │ 100% 开发 / 10% 生产采样      │
│ TraceID 关联  │ Grafana (可选)   │ GORM/Redis/NATS 自动埋点      │
└──────────────┴──────────────────┴───────────────────────────────┘
```

### 7.2 日志规范

```json
{
  "level": "info",
  "ts": "2026-04-26T10:30:00.000Z",
  "caller": "biz/post.go:45",
  "msg": "post created",
  "trace_id": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
  "span_id": "abcdef0123456789",
  "service": "ley-post",
  "instance": "ley-post-7d4f8b9c-1",
  "post_id": "01JT8X...",
  "author_id": "01JT8Y...",
  "duration_ms": 12
}
```

### 7.3 关键指标

| 指标 | 类型 | 描述 |
|------|------|------|
| `http_request_duration_ms` | Histogram | HTTP 请求延迟 |
| `grpc_request_duration_ms` | Histogram | gRPC 请求延迟 |
| `http_requests_total` | Counter | HTTP 请求总数 |
| `db_query_duration_ms` | Histogram | 数据库查询延迟 |
| `cache_hit_ratio` | Gauge | 缓存命中率 |
| `nats_publish_total` | Counter | NATS 发布消息数 |
| `nats_consume_lag` | Gauge | 消费者积压数 |
| `task_queue_size` | Gauge | 本地任务队列大小 |

---

## 8. 部署架构

### 8.1 Docker Compose 开发环境

```yaml
# docker-compose.yml 组件清单
services:
  postgres:     # PostgreSQL 16, port 5432
  redis:        # Redis 7, port 6379
  nats:         # NATS 2.x, port 4222
  etcd:         # etcd 3.6, port 2379
  jaeger:       # Jaeger all-in-one, port 16686 (UI) / 4318 (OTLP)
  minio:        # MinIO, port 9000 (API) / 9001 (Console)
  ley-gateway:  # 网关服务, port 8000
  ley-user:     # 用户服务, port 8001/9001
  ley-post:     # 文章服务, port 8002/9002
  ley-comment:  # 评论服务, port 8003/9003
  ley-file:     # 文件服务, port 8004/9004
```

### 8.2 服务资源配置

| 服务 | CPU | 内存 | 实例数 | gRPC 端口 | HTTP 端口 |
|------|-----|------|--------|-----------|-----------|
| ley-gateway | 0.5 | 256M | 1 | - | 8000 |
| ley-user | 0.25 | 128M | 1 | 9001 | 8001 |
| ley-post | 0.5 | 256M | 1 | 9002 | 8002 |
| ley-comment | 0.25 | 128M | 1 | 9003 | 8003 |
| ley-file | 0.25 | 128M | 1 | 9004 | 8004 |

---

## 9. 设计决策记录 (ADR)

### ADR-001: 为什么用 PostgreSQL 全文搜索而不是 Elasticsearch？

**决策：** 使用 PostgreSQL `tsvector` + GIN 索引。

**理由：**
1. 个人博客搜索需求简单（标题+内容关键词搜索），PG 内置方案完全足够
2. 减少运维组件（Elasticsearch 内存消耗大，需 ≥2GB heap）
3. 中文分词可通过 `zhparser` 扩展或 `jieba` 分词插件实现
4. 后续如需增强搜索，可无缝迁移到 Elasticsearch（保留 PG 作为主存）

### ADR-002: 为什么浏览计数用 Redis + 定时刷 DB？

**决策：** 浏览计数先写 Redis，定时批量刷新到 PostgreSQL。

**理由：**
1. 高并发浏览场景下，直接 UPDATE 数据库会造成写热点
2. Redis INCR 原子操作性能极佳
3. 允许少量计数损失（At-Most-Once 语义，定时刷新失败时）
4. 后续可用 Redis Stream 或 NATS 做更可靠的异步刷新

### ADR-003: 为什么评论计数维护在 Post 表？

**决策：** 采用反范式设计，在 posts 表维护 `comment_count` 冗余字段。

**理由：**
1. 避免每次展示文章列表时都要 JOIN + COUNT（性能优化）
2. 通过 NATS 事件驱动异步更新，确保最终一致性
3. 评论区需要总数时直接读取，无需跨服务调用
