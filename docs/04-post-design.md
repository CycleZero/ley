# Post 服务详设

## 1. 服务定位

```
ley-post 是博客内容核心服务，负责文章全生命周期管理、分类体系、标签体系、全文搜索、
内容计数维护，并向其他服务提供文章查询的内部 RPC。
```

### 边界定义

| 归属 Post 服务 | 归属其他服务 |
|---------------|-------------|
| 文章 CRUD | 用户认证（ley-user） |
| 分类管理 | 评论内容（ley-comment） |
| 标签管理 | 文件上传（ley-file） |
| 全文搜索 | 通知推送（ley-notification） |
| 点赞/取消点赞 | |
| 浏览计数（写入） | |
| 评论计数（最终一致维护） | |

---

## 2. 数据模型

### 2.1 ER 图

```
┌──────────────┐       ┌──────────────────┐       ┌──────────┐
│   Category   │       │       Post       │       │    Tag   │
├──────────────┤       ├──────────────────┤       ├──────────┤
│ ID       uint│  1:N  │ ID         uint  │  M:N  │ ID   uint│
│ Name  string │◄──────│ UUID     string  │──────►│ Name     │
│ Slug  string │       │ Title    string  │       │ Slug     │
│ Desc  string │       │ Slug     string  │       │ PostCount│
│ Parent *uint │       │ Content  string  │       │ gorm.Model│
│ Sort   int   │       │ Excerpt  string  │       └──────────┘
│ PostCount   │       │ Cover    string  │            │
│ gorm.Model  │       │ Status   int8    │            │
└──────────────┘       │ AuthorID  uint  │       ┌────┴─────┐
                       │ CateID   *uint  │       │ PostTag   │
                       │ ViewCnt  int64  │       ├───────────┤
                       │ LikeCnt  int64  │       │ PostID    │
                       │ CmtCnt   int64  │       │ TagID     │
                       │ IsTop    bool  │       │ gorm.Model│
                       │ Search   tsvec │       └───────────┘
                       │ PublishAt *time │
                       │ gorm.Model      │
                       └─────────────────┘
```

### 2.2 完整 DDL

```sql
CREATE SCHEMA IF NOT EXISTS "post";

-- ==================== categories ====================
CREATE TABLE "post".categories (
    id          BIGSERIAL PRIMARY KEY,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ,
    name        VARCHAR(64) NOT NULL,
    slug        VARCHAR(64) NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    parent_id   BIGINT,
    sort_order  INT NOT NULL DEFAULT 0,
    post_count  BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX idx_categories_deleted ON "post".categories(deleted_at);

-- ==================== tags ====================
CREATE TABLE "post".tags (
    id          BIGSERIAL PRIMARY KEY,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ,
    name        VARCHAR(64) NOT NULL UNIQUE,
    slug        VARCHAR(64) NOT NULL UNIQUE,
    post_count  BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX idx_tags_deleted ON "post".tags(deleted_at);

-- ==================== posts ====================
CREATE TABLE "post".posts (
    id              BIGSERIAL PRIMARY KEY,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    uuid            VARCHAR(36) NOT NULL UNIQUE,
    title           VARCHAR(200) NOT NULL,
    slug            VARCHAR(200) NOT NULL,
    content         TEXT NOT NULL DEFAULT '',
    excerpt         TEXT NOT NULL DEFAULT '',
    cover_image     VARCHAR(512) NOT NULL DEFAULT '',
    status          SMALLINT NOT NULL DEFAULT 0,     -- 0=draft 1=published 2=archived
    author_id       BIGINT NOT NULL,
    category_id     BIGINT,
    view_count      BIGINT NOT NULL DEFAULT 0,
    like_count      BIGINT NOT NULL DEFAULT 0,
    comment_count   BIGINT NOT NULL DEFAULT 0,
    is_top          BOOLEAN NOT NULL DEFAULT FALSE
);

-- search_vector is PostgreSQL GENERATED column, not stored in Go model
ALTER TABLE "post".posts ADD COLUMN search_vector tsvector
  GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(title, '')), 'A') ||
    setweight(to_tsvector('simple', coalesce(content, '')), 'B')
  ) STORED;

-- 索引
CREATE UNIQUE INDEX idx_posts_slug ON "post".posts(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_posts_deleted ON "post".posts(deleted_at);
CREATE INDEX idx_posts_status ON "post".posts(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_posts_author ON "post".posts(author_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_posts_category ON "post".posts(category_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_posts_search ON "post".posts USING GIN(search_vector);
CREATE INDEX idx_posts_published ON "post".posts(published_at DESC) WHERE status = 1 AND deleted_at IS NULL;
CREATE INDEX idx_posts_created ON "post".posts(created_at DESC);
CREATE INDEX idx_posts_uuid ON "post".posts(uuid);

-- ==================== posts_tags ====================
CREATE TABLE "post".posts_tags (
    id          BIGSERIAL PRIMARY KEY,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ,
    post_id     BIGINT NOT NULL,
    tag_id      BIGINT NOT NULL,
    UNIQUE (post_id, tag_id)
);

CREATE INDEX idx_posts_tags_post ON "post".posts_tags(post_id);
CREATE INDEX idx_posts_tags_tag ON "post".posts_tags(tag_id);
CREATE INDEX idx_posts_tags_deleted ON "post".posts_tags(deleted_at);
```

### 2.3 关键设计决策

**search_vector 用数据库生成列而非 Go 写入：**

```sql
GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(title, '')), 'A') ||   -- 标题权重 A
    setweight(to_tsvector('simple', coalesce(content, '')), 'B')    -- 正文权重 B
) STORED
```

> `search_vector` 字段在 Go model 中标记 `json:"-"`，由数据库自动维护，Go 侧不写、不读，仅 GORM AutoMigrate 时建列，实际 DDL 需手动执行 `ALTER TABLE ADD COLUMN`（GORM 不支持 GENERATED ALWAYS AS 语法）。

**slug 唯一索引条件化：**

```sql
CREATE UNIQUE INDEX idx_posts_slug ON "post".posts(slug) WHERE deleted_at IS NULL;
```

> 软删除的文章不占用 slug，允许同 slug 重新发布。

**PostTag 双重保障：**

- `gorm.Model.ID` 作为主键（满足 ORM 约定）
- `UNIQUE (post_id, tag_id)` 作为业务唯一约束

---

## 3. 分层架构

```
request → server/http.go → service/post.go → biz/post.go → data/post.go → PostgreSQL
                                 │                            │
                                 │ 依赖方向 →                  │ 实现 biz/post.go
                                 │ (biz 定义接口)              │ 定义的 PostRepo
                                 │                            │
                                 ├→ biz/tag.go                ├→ data/tag.go
                                 ├→ biz/category.go           ├→ data/category.go
                                 └→ biz/search.go             └→ data/search.go
```

### 3.1 各层职责

| 层 | 文件 | 职责 |
|----|------|------|
| **service** | `post.go`, `tag.go`, `category.go` | Proto 消息 ↔ 业务模型转换、参数校验（Proto validate）、错误码映射 |
| **biz** | `post.go`, `tag.go`, `category.go`, `search.go` | 业务规则、领域模型、Repo 接口定义、事件发布、事务编排 |
| **data** | `post.go`, `tag.go`, `category.go`, `search.go`, `cache.go` | GORM DB 操作、Redis 缓存、NATS 事件发布、搜索查询 |

---

## 4. API 设计（Proto）

### 4.1 文章接口

```protobuf
service PostService {
  // ---- 公开接口 ----
  rpc GetPost(GetPostRequest) returns (GetPostReply) {
    option (google.api.http) = { get: "/api/v1/posts/{slug_or_id}" };
  }
  rpc ListPosts(ListPostsRequest) returns (ListPostsReply) {
    option (google.api.http) = { get: "/api/v1/posts" };
  }
  rpc SearchPosts(SearchPostsRequest) returns (SearchPostsReply) {
    option (google.api.http) = { get: "/api/v1/posts/search" };
  }

  // ---- 作者接口 ----
  rpc CreatePost(CreatePostRequest) returns (CreatePostReply) {
    option (google.api.http) = { post: "/api/v1/posts" body: "*" };
  }
  rpc UpdatePost(UpdatePostRequest) returns (UpdatePostReply) {
    option (google.api.http) = { put: "/api/v1/posts/{id}" body: "*" };
  }
  rpc DeletePost(DeletePostRequest) returns (DeletePostReply) {
    option (google.api.http) = { delete: "/api/v1/posts/{id}" };
  }
  rpc PublishPost(PublishPostRequest) returns (PublishPostReply) {
    option (google.api.http) = { post: "/api/v1/posts/{id}/publish" body: "*" };
  }
  rpc LikePost(LikePostRequest) returns (LikePostReply) {
    option (google.api.http) = { post: "/api/v1/posts/{id}/like" body: "*" };
  }
  rpc UnlikePost(UnlikePostRequest) returns (UnlikePostReply) {
    option (google.api.http) = { delete: "/api/v1/posts/{id}/like" };
  }

  // ---- 标签接口 ----
  rpc CreateTag(CreateTagRequest) returns (CreateTagReply);
  rpc ListTags(ListTagsRequest) returns (ListTagsReply);
  rpc DeleteTag(DeleteTagRequest) returns (DeleteTagReply);

  // ---- 分类接口 ----
  rpc CreateCategory(CreateCategoryRequest) returns (CreateCategoryReply);
  rpc ListCategories(ListCategoriesRequest) returns (ListCategoriesReply);
  rpc UpdateCategory(UpdateCategoryRequest) returns (UpdateCategoryReply);
  rpc DeleteCategory(DeleteCategoryRequest) returns (DeleteCategoryReply);

  // ---- 内部 RPC（供其他服务调用） ----
  rpc IncrementViewCount(IncrementViewCountRequest) returns (IncrementViewCountReply);
  rpc BatchGetPosts(BatchGetPostsRequest) returns (BatchGetPostsReply);
}
```

### 4.2 关键消息定义

```protobuf
message CreatePostRequest {
  string title       = 1 [(validate.rules).string = {min_len: 2, max_len: 200}];
  string content     = 2 [(validate.rules).string = {min_len: 1}];
  string excerpt     = 3 [(validate.rules).string = {max_len: 500}];
  string cover_image = 4;
  uint64 category_id = 5;
  repeated string tag_names = 6;      // 按名称传标签，服务端自动查找或创建
}
message CreatePostReply {
  PostInfo post = 1;
}

message UpdatePostRequest {
  uint64 id         = 1;
  string title      = 2;
  string content    = 3;
  string excerpt    = 4;
  string cover_image = 5;
  uint64 category_id = 6;
  repeated string tag_names = 7;
}
message UpdatePostReply {
  PostInfo post = 1;
}

message ListPostsRequest {
  int32 page       = 1 [(validate.rules).int32 = {gte: 1}];
  int32 page_size  = 2 [(validate.rules).int32 = {gte: 1, lte: 50}];
  string status    = 3;  // draft | published | archived (空=全部)
  uint64 category_id = 4;
  repeated string tags = 5;
  string sort_by   = 6;  // created_at | updated_at | published_at | view_count
  string sort_order = 7; // asc | desc
}
message ListPostsReply {
  repeated PostInfo posts = 1;
  int32 total             = 2;    // 总数
  int32 page              = 3;
  int32 page_size         = 4;
}

message SearchPostsRequest {
  string keyword = 1 [(validate.rules).string = {min_len: 1, max_len: 200}];
  int32 page      = 2;
  int32 page_size = 3;
}
message SearchPostsReply {
  repeated PostInfo posts = 1;
  int32 total = 2;
}

message PostInfo {
  uint64 id              = 1;
  string uuid            = 2;
  string title           = 3;
  string slug            = 4;
  string excerpt         = 5;
  string cover_image     = 6;
  string status          = 7;
  uint64 author_id       = 8;
  uint64 category_id     = 9;
  repeated TagInfo tags  = 10;
  int64 view_count       = 11;
  int64 like_count       = 12;
  int64 comment_count    = 13;
  bool is_top            = 14;
  string published_at    = 15;
  string created_at      = 16;
  string updated_at      = 17;
  bool is_liked          = 18;  // 当前用户是否点赞（需登录）
}

message TagInfo {
  uint64 id    = 1;
  string name  = 2;
  string slug  = 3;
  int64 post_count = 4;
}

message CategoryInfo {
  uint64 id          = 1;
  string name        = 2;
  string slug        = 3;
  string description = 4;
  uint64 parent_id   = 5;
  int32 sort_order   = 6;
  int64 post_count   = 7;
  repeated CategoryInfo children = 8;  // 子分类树
}
```

---

## 5. 核心业务流程

### 5.1 CreatePost — 创建文章（Draft）

```
service.CreatePost(req)
  │
  ├── service 层
  │   ├── 从 ctx 提取 author_id（JWT 注入的 uid）
  │   └── 参数校验（Proto validate 自动）
  │
  ├── biz 层
  │   ├── 生成 UUID v7
  │   ├── 生成 slug = GenerateSlug(req.Title)
  │   ├── 检查 slug 唯一性 → 冲突则追加 "-2", "-3" ...
  │   ├── 查找或创建标签 ← req.TagNames
  │   │   └── 事务：INSERT INTO tags ... ON CONFLICT (name) DO NOTHING RETURNING id
  │   ├── 构建 Post 实体
  │   │   ├── Status = Draft
  │   │   ├── AuthorID = author_id
  │   │   ├── CategoryID = req.CategoryID
  │   │   └── Tags = resolvedTags
  │   ├── repo.Create(post)
  │   │   └── INSERT INTO "post".posts ... RETURNING id
  │   ├── repo.AssociateTags(post.ID, tagIDs)
  │   │   └── INSERT INTO "post".posts_tags (post_id, tag_id) ...
  │   └── 更新 tag.post_count、category.post_count
  │
  └── 返回 PostInfo
```

### 5.2 PublishPost — 发布文章

```
service.PublishPost(req)
  │
  ├── 校验：仅 author 或 admin 可操作
  ├── 校验：当前状态必须为 Draft
  ├── 设置 Status = Published
  ├── 设置 PublishedAt = now()
  ├── repo.Update(post)
  ├── 主动删除缓存 key
  ├── 发布事件: post.published {post_id, author_id, title, slug, tags}
  └── 返回 PostInfo
```

### 5.3 UpdatePost — 更新文章

```
service.UpdatePost(req)
  │
  ├── 从缓存/DB 读取现有 Post
  ├── 校验：仅 author 或 admin 可操作
  ├── 允许更新的字段：Title, Content, Excerpt, CoverImage, CategoryID, Tags
  │   ├── 若 Title 变更 → 重新生成 Slug
  │   └── 若 Tags 变更 → 重新维护关联关系
  ├── repo.Update(post)
  ├── 删除缓存
  ├── 发布事件: post.updated {post_id, ...}
  └── 返回 PostInfo
```

### 5.4 LikePost / UnlikePost — 点赞/取消点赞

```
LikePost(req):
  │
  ├── 从 ctx 获取 user_id
  ├── Redis 检查: ZSCORE post:like:{post_id} {user_id}
  │   ├── 已存在 → 幂等返回
  │   └── 不存在 → 继续
  ├── repo.InsertLike(post_id, user_id)  → INSERT INTO "post".posts_likes ...
  ├── repo.IncrementLikeCount(post_id)    → UPDATE posts SET like_count = like_count + 1
  ├── Redis ZADD post:like:{post_id} {timestamp} {user_id}
  └── 发布事件: post.liked {post_id, user_id}

UnlikePost(req):
  │
  ├── 从 ctx 获取 user_id
  ├── repo.DeleteLike(post_id, user_id)  → DELETE FROM "post".posts_likes ...
  ├── repo.DecrementLikeCount(post_id)   → UPDATE posts SET like_count = like_count - 1
  ├── Redis ZREM post:like:{post_id} {user_id}
  └── 发布事件: post.unliked {post_id, user_id}
```

### 5.5 GetPost — 获取文章（Cache-Aside）

```
                      ┌──────────────┐
      请求 ──────────►│  Redis GET   │── miss ──► GORM Query ──► Redis SET ──► 返回
                      │ cache:post:  │               │
                      │ detail:{slug}│               ├── SELECT posts.*, tags.*
                      └──────┬───────┘               │   FROM "post".posts
                             │ hit                   │   LEFT JOIN posts_tags ...
                             ▼                       │   WHERE slug = ?
                      ┌──────────────┐               │
                      │  直接返回    │               └── AutoMigrate 生成的
                      └──────────────┘                   search_vector 列...
                                                        (使用 json:"-" 在 Go 中屏蔽)
```

**缓存策略矩阵：**

| Key Pattern | 内容 | TTL | 触发更新 |
|------------|------|-----|---------|
| `post:detail:{uuid}` | PostInfo JSON | 10min | 文章更新/删除时删除 |
| `post:list:{hash}` | ListPostsReply JSON | 5min | 新文章发布时 BulkDelete `post:list:*` |
| `post:like:{id}` | Sorted Set (user_id→timestamp) | 永久 | 点赞/取消时 ZADD/ZREM |
| `tag:all` | TagInfo[] JSON | 60min | 标签增删时删除 |
| `category:tree` | CategoryInfo[] JSON（含 children） | 60min | 分类增删改时删除 |
| `post:view:{id}` | 计数 int64 | 永久 | INCR，定时批量刷 DB |

### 5.6 浏览计数：Redis INCR + 定时刷新

```
写路径:
  IncrementViewCount(postID):
    Redis INCR post:view:{postID}
    (异步、无阻塞)

定时刷新（pkg/task 每 5min）:
  TickViewFlush():
    keys = Redis KEYS post:view:*
    for key in keys:
        postID  = extractID(key)
        delta   = Redis GETDEL(key)       // 原子读取并删除
        DB UPDATE posts SET view_count = view_count + delta WHERE id = postID
```

> 设计要点：`GETDEL` 原子操作确保计数不丢失；刷新失败则下轮重试（delta 累积在下一个 5min 周期）。

---

## 6. 全文搜索设计

### 6.1 PostgreSQL tsvector 方案

**架构：**

```
Client                    ley-post                       PostgreSQL
  │                         │                               │
  │ GET /posts/search       │                               │
  │ ?keyword=kratos&page=1  │                               │
  │ ──────────────────────► │                               │
  │                         │ 不走缓存（搜索结果多变）         │
  │                         │                               │
  │                         │ SELECT id, title, excerpt,     │
  │                         │   ts_rank(search_vector,       │
  │                         │     plainto_tsquery(           │
  │                         │       'simple', 'kratos'))     │
  │                         │   AS rank                      │
  │                         │ FROM "post".posts              │
  │                         │ WHERE search_vector @@         │
  │                         │   plainto_tsquery(             │
  │                         │     'simple', 'kratos')        │
  │                         │   AND status = 1               │
  │                         │   AND deleted_at IS NULL       │
  │                         │ ORDER BY rank DESC             │
  │                         │ LIMIT 20 OFFSET 0;             │
  │                         │ ──────────────────────────────►│
  │                         │                               │
  │                         │ ← 结果 + total count           │
  │ ← JSON 结果             │                               │
```

### 6.2 中文分词方案

```
┌──────────────────────────────────────────────────────┐
│  方案 A: pg_jieba 扩展（推荐）                        │
│                                                      │
│  CREATE EXTENSION pg_jieba;                          │
│  -- 配置字典: jieba.dict_path = '/usr/share/jieba/'  │
│                                                      │
│  ALTER TABLE posts                                   │
│    ALTER search_vector TYPE tsvector                 │
│    USING to_tsvector('jieba', title || ' ' || content)│
│                                                      │
│  搜索: to_tsquery('jieba', '微服务框架')              │
├──────────────────────────────────────────────────────┤
│  方案 B: zhparser 扩展                               │
│                                                      │
│  CREATE EXTENSION zhparser;                          │
│  CREATE TEXT SEARCH CONFIGURATION chinese (          │
│    PARSER = zhparser                                 │
│  );                                                  │
│  ALTER TEXT SEARCH CONFIGURATION chinese             │
│    ADD MAPPING FOR n,v,a,i,e,l WITH simple;          │
└──────────────────────────────────────────────────────┘
```

### 6.3 hit 高亮（应用层实现）

```go
func Highlight(content string, keyword string) string {
    // 对 content 中匹配 keyword 的部分包裹 <mark> 标签
    re := regexp.MustCompile(`(?i)(` + regexp.QuoteMeta(keyword) + `)`)
    return re.ReplaceAllString(content, `<mark>$1</mark>`)
}
```

---

## 7. 事件驱动设计

### 7.1 发布的事件

| 事件 | 触发时机 | Payload | 消费者 |
|------|---------|---------|--------|
| `post.created` | 创建文章 | `{post_id, author_id, title, status}` | 无（仅日志记录） |
| `post.published` | 发布文章 | `{post_id, author_id, title, slug, tags, published_at}` | notification → 邮件通知订阅者 |
| `post.updated` | 更新文章 | `{post_id, title, slug, diff_fields[]}` | cache → 清理 CDN 缓存 |
| `post.deleted` | 删除文章 | `{post_id, author_id}` | comment → 批量软删除关联评论 |
| `post.liked` | 点赞 | `{post_id, user_id}` | notification → 通知作者 |
| `post.unliked` | 取消点赞 | `{post_id, user_id}` | 无 |

### 7.2 订阅的事件

| 事件 | 来源 | 处理逻辑 |
|------|------|---------|
| `comment.created` | ley-comment | `UPDATE posts SET comment_count = comment_count + 1 WHERE id = ?` |
| `comment.deleted` | ley-comment | `UPDATE posts SET comment_count = comment_count - 1 WHERE id = ?` |
| `user.deleted` | ley-user | 转移/删除用户所有文章（或标记 author_id=0） |

### 7.3 幂等性保障

```go
// data 层事件处理器
func (r *postRepo) OnCommentCreated(ctx context.Context, event CommentCreatedEvent) error {
    idempotentKey := fmt.Sprintf("idempotent:comment:%s:%d", event.EventID, event.PostID)
    
    // Redis SET NX 原子防重
    ok, err := r.cache.SetNX(ctx, idempotentKey, "1", 1*time.Hour)
    if err != nil || !ok {
        return nil // 已处理或错误，跳过
    }
    
    return r.DB.WithContext(ctx).
        Model(&model.Post{}).
        Where("id = ?", event.PostID).
        Update("comment_count", gorm.Expr("comment_count + 1")).Error
}
```

---

## 8. 缓存设计详解

### 8.1 缓存 Key 设计

```go
const (
    // 文章详情:{uuid}
    cacheKeyPostDetail = "post:detail:%s"          // → post:detail:01JQXYZ...
    
    // 文章列表:{pageSize}_{page}_{status}_{category}_{tags}_{sort}
    cacheKeyPostList   = "post:list:{%d}_{%d}_{%s}_{%d}_{%s}_{%s}"
    
    // 标签全量
    cacheKeyTags       = "tag:all"
    
    // 分类树
    cacheKeyCategories = "category:tree"
    
    // 浏览计数
    cacheKeyViews      = "post:view:%d"            // → post:view:42
    
    // 点赞用户集合
    cacheKeyLikes      = "post:like:%d"            // → post:like:42
)
```

### 8.2 列表缓存命中策略

```
ListPosts 请求流程:

  hash = MD5(req.Status + req.CategoryID + req.Tags + req.SortBy + req.SortOrder)
  key  = post:list:{pageSize}_{page}_{hash}
  
  val = Redis GET(key)
  if val != nil:
      return deserialize(val)           // Cache Hit
  
  // Cache Miss
  posts, total = DB Query
  reply = {posts, total, page, pageSize}
  Redis SETEX(key, 300, serialize(reply))   // 5min TTL
  return reply
```

**列表缓存失效策略：**

```go
// 任何文章 CUD 操作后
func (r *postRepo) invalidateListCache(ctx context.Context) {
    // 方案 A（简单粗暴）：删除所有列表缓存
    keys, _ := r.rdb.Keys(ctx, "post:list:*").Result()
    if len(keys) > 0 {
        r.rdb.Del(ctx, keys...)
    }
    
    // 方案 B（精确）：按标签/分类针对性失效
    // 实际用方案 A，5min TTL 足够短，删除开销可控
}
```

### 8.3 缓存穿透预防

| 攻击方式 | 对策 |
|---------|------|
| 查询不存在的 slug | 空值也缓存：`SETEX post:detail:{slug} 60 "null"` |
| 大量并发 Miss | singleflight：同一 key 只有一个 goroutine 查 DB |
| 大量随机 key 扫描 | Bloom filter 预判存在性（后续优化，一期不实现） |

**singleflight 示例：**

```go
var sf singleflight.Group

func (r *postRepo) GetBySlug(ctx context.Context, slug string) (*model.Post, error) {
    cacheKey := fmt.Sprintf(cacheKeyPostDetail, slug)
    
    val, err, _ := sf.Do(cacheKey, func() (interface{}, error) {
        // 1. 先查 Redis
        data, err := r.cache.Get(ctx, cacheKey)
        if err == nil {
            var post model.Post
            json.Unmarshal(data, &post)
            return &post, nil
        }
        
        // 2. 查 DB
        var post model.Post
        err = r.DB.WithContext(ctx).
            Preload("Tags").
            Where("slug = ?", slug).
            First(&post).Error
        if err != nil {
            if errors.Is(err, gorm.ErrRecordNotFound) {
                // 缓存空值
                r.cache.Set(ctx, cacheKey, []byte("null"), 60*time.Second)
                return nil, ErrNotFound
            }
            return nil, err
        }
        
        // 3. 写缓存
        data, _ := json.Marshal(post)
        r.cache.Set(ctx, cacheKey, data, 10*time.Minute)
        return &post, nil
    })
    
    if err != nil {
        return nil, err
    }
    return val.(*model.Post), nil
}
```

---

## 9. biz 层接口定义

```go
// biz/post.go — 业务接口（依赖倒置）

// PostRepo 数据访问接口（由 data 层实现）
type PostRepo interface {
    Create(ctx context.Context, post *model.Post) error
    Update(ctx context.Context, post *model.Post) error
    SoftDelete(ctx context.Context, id uint) error
    FindByID(ctx context.Context, id uint) (*model.Post, error)
    FindByUUID(ctx context.Context, uuid string) (*model.Post, error)
    FindBySlug(ctx context.Context, slug string) (*model.Post, error)
    List(ctx context.Context, query PostListQuery) ([]*model.Post, int64, error)
    Search(ctx context.Context, keyword string, page, pageSize int) ([]*model.Post, int64, error)
    InsertLike(ctx context.Context, postID, userID uint) error
    DeleteLike(ctx context.Context, postID, userID uint) error
    IsLiked(ctx context.Context, postID, userID uint) (bool, error)
}

type PostListQuery struct {
    Status     string
    CategoryID *uint
    Tags       []string
    AuthorID   *uint
    SortBy     string
    SortOrder  string
    Page       int
    PageSize   int
}

// PostUseCase 业务用例
type PostUseCase struct {
    repo     PostRepo
    eventBus EventBus
    cache    PostCache
}

func (uc *PostUseCase) CreatePost(ctx context.Context, req *CreatePostReq) (*model.Post, error)
func (uc *PostUseCase) PublishPost(ctx context.Context, id uint) (*model.Post, error)
func (uc *PostUseCase) UpdatePost(ctx context.Context, req *UpdatePostReq) (*model.Post, error)
func (uc *PostUseCase) DeletePost(ctx context.Context, id uint) error
func (uc *PostUseCase) GetPost(ctx context.Context, slugOrID string) (*model.Post, error)
func (uc *PostUseCase) ListPosts(ctx context.Context, query PostListQuery) ([]*model.Post, int64, error)
func (uc *PostUseCase) SearchPosts(ctx context.Context, kw string, page, size int) ([]*model.Post, int64, error)
func (uc *PostUseCase) LikePost(ctx context.Context, postID, userID uint) error
func (uc *PostUseCase) UnlikePost(ctx context.Context, postID, userID uint) error
```

---

## 10. Slug 生成策略

```go
import (
    "regexp"
    "strings"
    "github.com/mozillazg/go-pinyin"  // 中文拼音转换
)

var (
    reNonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)
    reDash        = regexp.MustCompile(`-+`)
)

func GenerateSlug(title string) string {
    // 1. 中文 → 全拼首字母
    pyArgs := pinyin.NewArgs()
    pyArgs.Style = pinyin.FirstLetter  // 首字母模式
    
    var builder strings.Builder
    for _, r := range title {
        if r >= 0x4e00 && r <= 0x9fff {  // 中文字符
            py := pinyin.Pinyin(string(r), pyArgs)
            if len(py) > 0 && len(py[0]) > 0 {
                builder.WriteString(py[0])
            }
        } else {
            builder.WriteRune(r)
        }
    }
    
    // 2. 转小写
    s := strings.ToLower(builder.String())
    
    // 3. 非字母数字 → "-"
    s = reNonAlphaNum.ReplaceAllString(s, "-")
    
    // 4. 合并连续 "-"
    s = reDash.ReplaceAllString(s, "-")
    
    // 5. 去除首尾 "-"
    s = strings.Trim(s, "-")
    
    // 6. 截断至 200
    if len(s) > 200 {
        s = s[:200]
    }
    
    // 7. 空值兜底
    if s == "" {
        s = "post"
    }
    
    return s
}

// 唯一性保证：若 slug 已存在，追加 "-2", "-3" ...
func (uc *PostUseCase) ensureUniqueSlug(ctx context.Context, slug string) (string, error) {
    candidate := slug
    for i := 2; i <= 100; i++ {
        _, err := uc.repo.FindBySlug(ctx, candidate)
        if errors.Is(err, ErrNotFound) {
            return candidate, nil
        }
        candidate = fmt.Sprintf("%s-%d", slug, i)
    }
    return "", fmt.Errorf("slug collision exhausted: %s", slug)
}
```

---

## 11. 安全与权限

| 操作 | reader | author | admin |
|------|--------|--------|-------|
| ListPosts / GetPost | ✓ | ✓ | ✓ |
| SearchPosts | ✓ | ✓ | ✓ |
| CreatePost | ✗ | ✓ | ✓ |
| UpdatePost | ✗ | ✓ (仅自己) | ✓ (所有) |
| DeletePost | ✗ | ✓ (仅自己) | ✓ (所有) |
| PublishPost | ✗ | ✓ (仅自己) | ✓ (所有) |
| LikePost / UnlikePost | ✓ | ✓ | ✓ |
| CreateTag | ✗ | ✓ | ✓ |
| DeleteTag | ✗ | ✗ | ✓ |
| CreateCategory | ✗ | ✗ | ✓ |

> Biz 层在操作前校验 `author_id` 与 ctx 中 `user_id` 一致性，admin 跳过该检查。

---

## 12. 内容格式设计

### 12.1 存储格式：Markdown

`posts.content` 字段存储**纯文本 Markdown**（非 HTML，非二进制），理由：

| 考量 | Markdown | HTML 富文本 |
|------|----------|------------|
| 安全性 | 天然无 XSS，渲染侧沙箱 | 需服务端清洗 + CSP |
| 全文搜索 | 纯文本，`tsvector` 索引直接命中 | HTML 标签噪音干扰搜索质量 |
| 体积 | 紧凑，无样式冗余 | 携带大量 DOM/样式，膨胀 3-10x |
| 可移植 | Git 友好，可 diff/merge | 不可 diff |
| 编辑体验 | 任意编辑器，不绑定前端 | 强依赖 WYSIWYG 编辑器 |
| 版本化 | 天然支持（纯文本 diff） | 需额外序列化逻辑 |

### 12.2 Markdown 方言：GFM + 扩展

基于 GitHub Flavored Markdown（GFM），扩展以下特性：

| 特性 | 语法 | 实现 |
|------|------|------|
| **GFM 基础** | 标题、列表、链接、图片、代码块、表格、任务列表 | goldmark-gfm |
| **数学公式** | `$E=mc^2$` 行内 / `$$...$$` 块级 | goldmark-mathjax (KaTeX/MathJax) |
| **脚注** | `[^1]` ... `[^1]: 注释内容` | goldmark-footnote |
| **定义列表** | `Term` / `: definition` | goldmark-extensions |
| **高亮标记** | `==highlight==` | goldmark-extensions |
| **流程图/图表** | ````mermaid` ... ```` | 前端 mermaid.js 渲染 |
| **自定义容器** | `:::tip` / `:::warning` / `:::danger` | 前端 CSS 组件 |
| **Emoji** | `:smile:` → 😄 | goldmark-emoji |
| **媒体嵌入** | `@[youtube](video_id)` | 自定义 parser 转换为 iframe |

### 12.3 渲染管道

```
 ┌──────────────────────────────────────────────────────────┐
 │                    内容渲染管道                            │
 │                                                          │
 │   存储 (DB)         服务端可选渲染        前端渲染         │
 │                                                          │
 │                    ┌──────────────┐                      │
 │   Markdown ──────►│ goldmark     │──► HTML (服务端)      │
 │   (纯文本)         │ + extensions │                      │
 │                    └──────┬───────┘                      │
 │                           │                              │
 │                           │ 或跳过服务端渲染               │
 │                           ▼                              │
 │                    ┌──────────────┐                      │
 │                    │ markdown-it  │──► HTML (客户端)     │
 │                    │ + plugins    │                      │
 │                    └──────────────┘                      │
 │                                                          │
 │   策略: 存储 Markdown，服务端不渲染，前端负责渲染         │
 │   优势: 保持原始内容纯净，支持多客户端 (Web/App/API)      │
 └──────────────────────────────────────────────────────────┘
```

**决策：服务端不渲染 Markdown。** API 返回原始 Markdown 文本，由前端渲染。

只有在以下场景服务端才渲染：
- **搜索摘要**（excerpt 字段）：截取纯文本前 200 字，去除 Markdown 标记
- **RSS/Atom 输出**：渲染为 HTML 嵌入 XML
- **邮件通知**：渲染为 HTML 发送

### 12.4 富文本支持策略

**不支持直接存储 HTML 富文本**，但通过以下机制覆盖富文本场景：

| 需求 | 方案 |
|------|------|
| 用户从外部编辑器粘贴 | 前端粘贴时自动转换 HTML → Markdown（turndown.js） |
| 图片/附件插入 | 先上传到 ley-file，获取 URL，自动拼接 `![]()` 语法 |
| 表格 | GFM 表格语法原生支持 |
| 代码高亮 | 前端 highlight.js / Prism.js 渲染 |
| 自定义样式 | 前端 CSS 变量 + `.blog-content` 作用域样式 |

### 12.5 安全措施

```
                       ┌──────────────┐
                       │  XSS 防御链   │
                       ├──────────────┤
写作时                   │                      │
  ├── 服务端不做内容校验                    │
  │   (Markdown 纯文本无 XSS 风险)          │
  │                                        │
存储时                   │                      │
  ├── 直接写入 PostgreSQL TEXT 列          │
  │   (参数化查询防 SQL 注入)               │
  │                                        │
前端渲染时               │                      │
  ├── markdown-it 配置:                    │
  │   html: false       ← 禁止原始 HTML     │
  │   linkify: true     ← 自动识别链接      │
  │   typographer: true ← 智能引号          │
  ├── DOMPurify 二次清洗（兜底）            │
  ├── CSP Header:                          │
  │   Content-Security-Policy:             │
  │   default-src 'self'                   │
  └────────────────────────────────────────┘
```

### 12.6 内容自动摘要

```go
// Excerpt 生成策略（优先级递减）
func GenerateExcerpt(content string, maxLen int) string {
    // 1. 用户手动填写 → 直接使用
    // 2. 自动生成：
    content = stripMarkdown(content)  // 去除 # * _ []( ) ``` `` 等标记
    content = strings.Join(strings.Fields(content), " ") // 合并空白
    
    runes := []rune(content)
    if len(runes) <= maxLen {
        return content
    }
    return string(runes[:maxLen]) + "..."
}

func stripMarkdown(s string) string {
    // 移除标题标记: ### 
    s = reHeading.ReplaceAllString(s, "")
    // 移除加粗斜体: ** __ * _
    s = reBoldItalic.ReplaceAllString(s, "$1")
    // 移除链接: [text](url) → text
    s = reLink.ReplaceAllString(s, "$1")
    // 移除图片: ![alt](url) → alt
    s = reImage.ReplaceAllString(s, "$1")
    // 移除代码块: ```...```
    s = reCodeBlock.ReplaceAllString(s, "")
    // 移除行内代码: `code`
    s = reInlineCode.ReplaceAllString(s, "$1")
    // 移除多余空白
    s = strings.Join(strings.Fields(s), " ")
    return s
}
```

### 12.7 内容版本化（可选，Phase 2+）

```
 ┌─────────────────────────────────────┐
 │          内容版本控制                 │
 │                                     │
 │   posts 表                          │
 │   ├── content (当前版本)              │
 │                                     │
 │   post_revisions 表（可选）           │
 │   ├── post_id                       │
 │   ├── version  INT                  │
 │   ├── content  TEXT                 │
 │   ├── diff_from_prev  TEXT (可选)    │
 │   ├── created_at                    │
 │   └── created_by                    │
 │                                     │
 │   使用: git apply 风格的增量 diff     │
 └─────────────────────────────────────┘
```

暂不实现，预留扩展点。内容存储在 `posts.content` 中，`updated_at` 记录最后修改时间。

### 12.8 Markdown vs HTML 富文本详细对比

#### 总览

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    Markdown vs HTML 全景对比                               │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│   维度          │  Markdown ★ 赢            │  HTML                       │
│   ──            │  ───────                  │  ────                       │
│   安全性        │  纯文本天然安全            │  必须清洗，漏洞高发区        │
│   搜索质量      │  无标签噪音，精确索引      │  DOM 噪音污染搜索结果        │
│   存储体积      │  紧凑（~10KB/篇）          │  膨胀 3-10x                 │
│   版本化        │  Git diff 行级精准        │  diff 无意义（无换行）       │
│   API 友好      │  raw text，多端通用        │  强耦合 Web 渲染            │
│   可移植性      │  零依赖迁移                │  绑定编辑器产出的 DOM 结构   │
│   ──            │  ───────                  │  ────                       │
│   排版能力      │  有限（受限语法集）        │  ★ 无限（任意 CSS）          │
│   所见即所得    │  需分屏预览                │  ★ WYSIWYG 原生支持          │
│   非技术用户    │  入门有学习成本            │  ★ 像 Word 一样编辑          │
│   复杂表格      │  手写对齐困难              │  ★ 拖动拽生成                │
│   嵌入内容      │  语法简陋 (![]())          │  ★ 任意 iframe/web component │
│                                                                          │
├──────────────────────────────────────────────────────────────────────────┤
│   结论: 技术博客 / 开发者向 → Markdown 完胜                               │
│         大众内容平台 / 运营向 → HTML 有优势，但可转换                      │
└──────────────────────────────────────────────────────────────────────────┘
```

#### 维度 1: 安全性

| 攻击面 | Markdown | HTML |
|--------|----------|------|
| **XSS** | 不存在。文本就是文本，`<script>` 就是字面 6 个字符 | 核心威胁。必须服务端 HTML sanitize（如 bluemonday）+ CSP 头 + 前端 DOMPurify |
| **CSS 注入** | 不存在 | `<div style="position:fixed;top:0;left:0;width:100vw;height:100vh">` 可覆盖页面 |
| **点击劫持** | 不存在 | `<a>` 可伪装 UI 元素诱骗点击 |
| **图片外链追踪** | 可控。解析 `![](url)` 时可改写为代理 | 更难控制，DOM 中图片 URL 分散 |
| **iframe 嵌入** | 默认不支持 | `<iframe>` 可嵌入恶意站点，X-Frame-Options 防护 |
| **SVG 内嵌** | 不支持 | `<svg><script>` 可执行脚本 |

```
Markdown 安全模型:
  输入:  # Hello <script>alert(1)</script>
  安全分析: 没有"执行"路径，尖括号就是尖括号
  
HTML 安全模型:
  输入:  <p>Hello <script>alert(1)</script></p>
  安全分析: script 标签需要被白名单过滤掉
  防御链:  解析 → 白名单过滤 → 属性过滤 → URL 协议检查 → 输出编码
  每个环节都可能出错
```

**Markdown 也非绝对安全：** 如果渲染器允许 raw HTML（markdown-it 配置 `html: true`），攻击者可在 Markdown 中嵌入 HTML 脚本。本系统配置 `html: false` 杜绝此路径。

#### 维度 2: 搜索质量

```
Markdown 存储:  "Kratos 是一个 Go 微服务框架，起源于 B 站"
tsvector 索引:  'kratos':1 'go':3 '微服务':4 '框架':5 '起源于':6 'b':7 '站':8
搜索 "微服务框架": 精确命中

HTML 存储:      "<h1 id="kratos">Kratos</h1><p>是一个 <strong>Go</strong> 微服务框架，起源于 B 站</p>"
tsvector 索引:  'h1':1 'id':2 'kratos':3,4 'p':5,12 'strong':6 'go':7 'go':8 'strong':9 ...
搜索 "微服务框架": 命中但排名受 HTML 标签 noise 影响
```

| 影响 | Markdown | HTML |
|------|----------|------|
| `ts_rank` 准确度 | 高（每个词都来自正文） | 低（标签词稀释权重，`div`/`span`/`class` 出现在向量中） |
| GIN 索引大小 | 小 | 大 2-5x（多了大量标签词元） |
| 搜索结果摘要 | 直接截取可读 | 需先 strip_tags 再截取，或存储两个字段 |

> PostgreSQL `tsvector` 不支持"忽略 HTML 标签"模式。HTML 内所有文本（包括属性值、标签名）都进入搜索向量。

#### 维度 3: 存储体积

```
同一篇文章:
  Markdown:  10,242 bytes
  HTML:      42,183 bytes  (4.1x)
  
  含大量样式类名的 HTML (如 Quill 输出):
  Markdown:  10,242 bytes
  HTML:      87,560 bytes  (8.5x)
```

| 累计效应 | 100 篇文章 | 1000 篇文章 | 10万篇文章 |
|----------|-----------|------------|-----------|
| Markdown | ~1 MB | ~10 MB | ~1 GB |
| HTML | ~4-8 MB | ~40-80 MB | ~4-8 GB |
| DB 备份 | 快 | 慢 4-8x |
| 网络传输 | 快 (gzip 后差距更大) | 慢 |
| CDN 缓存 | 命中率高 | 命中率低（体积大） |

#### 维度 4: 版本控制与协作

```
Markdown Git Diff:
  - Kratos 是一个 Go 微服务框架，起源于 B 站。
  + Kratos 是 B 站开源的 Go 微服务框架，用于构建云原生应用。
  
  一目了然，行级对比。

HTML Git Diff:
  - <p>Kratos 是一个 <strong>Go</strong> 微服务框架，起源于 <a href="https://bilibili.com">B 站</a>。</p>
  + <div class="content-block"><p>Kratos 是 <a href="https://bilibili.com">B 站</a>开源的 <strong>Go</strong> 微服务框架，用于构建<span class="highlight">云原生</span>应用。</p></div>
  
  改了什么？diff 不可读。
```

| 能力 | Markdown | HTML |
|------|----------|------|
| 行级 diff | ✓ | ✗（编辑器一行写完所有 HTML） |
| merge 冲突解决 | ✓ 简单 | ✗ 灾难 |
| blame 追溯 | ✓ | ✗（每次编辑整行变更） |
| 历史回滚 | ✓ 精准 | ✓ 技术上可行，实际无意义 |
| AI 辅助改写 | ✓（纯文本，LLM 天然理解） | △（需要 parse → 改 → 序列化，容易破坏结构） |

#### 维度 5: 多端支持

```
                      Markdown                          HTML
                      ────────                          ────
   Web 渲染     markdown-it + CSS          直接 innerHTML（清洗后）
                可换任意渲染器              绑定输出格式
  
   iOS/Android  MarkdownKit (原生渲染)      WKWebView/WebView
                Compose Markdown            必须 WebView，启动慢
                
   RSS/Atom     服务端渲染为 HTML           直出（但需 strip 样式）
   Reader       RSS Reader 原生支持         混乱（自带样式覆盖 RSS Reader）

   API 消费方    返回 raw text               返回 HTML 片段
                客户端自由渲染               客户端必须理解 HTML

   微信/飞书     转 Markdown → 富文本消息    转 HTML → 纯文本（丢失格式）
   Bot 推送      天然适配                    需要 strip_tags 降级

   终端阅读      cat post.md 可读            curl | lynx 勉强
```

**结论：** Markdown 是内容格式的最小公约数，各端按需渲染。HTML 需要客户端具备浏览器级别的理解能力。

#### 维度 6: 编辑体验

```
Markdown 编辑:
  ┌──────────────────────────────────────────────────┐
  │ ## 标题                                          │
  │                                                  │
  │ 正文内容，**加粗**，*斜体*。                       │
  │                                                  │
  │ - 列表项 1                                       │
  │ - 列表项 2                                       │
  │                                                  │
  │ ```go                                            │
  │ func main() {                                    │
  │     fmt.Println("hello")                         │
  │ }                                                │
  │ ```                                              │
  └──────────────────────────────────────────────────┘
  
  优势: 键盘流，不碰鼠标，专注内容
  劣势: 学习语法（30 分钟入门），需要分屏预览查看效果

HTML 富文本编辑:
  ┌──────────────────────────────────────────────────┐
  │ [B] [I] [U] [H1] [H2] [🔗] [🖼️] [📋] ...        │
  ├──────────────────────────────────────────────────┤
  │ 标题（点击选中文字点 H2 设置）                      │
  │ 正文内容，加粗（选中 + Ctrl+B），斜体               │
  │ · 列表项 1（点击列表按钮）                          │
  │ · 列表项 2                                        │
  └──────────────────────────────────────────────────┘
  
  优势: 所见即所得，非技术人员友好
  劣势: 鼠标操作慢，排版不可控（不同编辑器产出的 HTML 完全不同）
```

| 用户群体 | 推荐格式 | 理由 |
|---------|---------|------|
| 开发者 | Markdown | 已经会、效率高、版本可控 |
| 技术作者 | Markdown | 代码块/数学公式/TOC 比富文本编辑器强 |
| 普通用户 | HTML 富文本 | 不需要学语法 |
| 运营/编辑 | HTML 富文本 | 需要精准控制排版，所见即所得 |

**本系统定位：个人技术博客 → Markdown 正确选择。**

若未来需要支持非技术用户，可前端集成 TipTap/Quill，编辑器内部用 Markdown 作为序列化格式（turndown.js 转换），对外透明。

#### 维度 7: 可移植性与锁定风险

```
Markdown 迁移路径:
  当前: psql TEXT 列
    → WordPress: 插件导入 Markdown
    → Ghost: 原生 Markdown 支持
    → Hugo/Hexo: 文件复制即可（.md 文件）
    → Notion: 导入 Markdown
    → 静态 HTML: 一行命令 goldmark → HTML
  风险: 零锁定

HTML 迁移路径:
  当前: psql TEXT 列 (带 Quill 生成的 class 名)
    → WordPress: 格式兼容较好
    → Ghost: 需转换
    → Hugo/Hexo: 不兼容，需全部转换
    → Notion: 会丢失样式
    → 静态 HTML: 携带原始编辑器的 DOM 结构 (quill-* / tiptap-* 等 class)
  风险: 与编辑器锁定。换一个编辑器 = 全量数据迁移
```

#### 维度 8: 性能对比

```
场景: API 返回单篇文章 (content + metadata)

              请求 → 响应全链路延迟 (ms)
              ──────    ─────
              50KB Markdown              50KB HTML (压缩结构)
DB 读取:      2ms                         3ms         (体积差异)
网络传输:     5ms (gzip → 8KB)           12ms (gzip → 18KB)
Redis 缓存:   1ms (内存占用小)            2ms (内存占用大)
前端渲染:     8ms (markdown-it parse)     2ms (innerHTML)
─────────────────────────────────────────────────────────
总计:         16ms                        19ms

但是极端场景 (200KB 富文本 HTML):
DB 读取:      2ms                         8ms
网络传输:     20ms                        60ms
Redis 缓存:   3ms                         10ms
前端渲染:     12ms                        3ms
─────────────────────────────────────────────────────────
总计:         37ms                        81ms   (2.2x)
```

**关键发现：** 小文章差异不大。长文章（图文并茂）HTML 传输和缓存开销显著增大。

#### 维度 9: 文章导出

```
Markdown 导出:
  → PDF:    pandoc post.md -o post.pdf (完美，保留代码高亮)
  → Word:   pandoc post.md -o post.docx
  → ePub:   pandoc post.md -o post.epub
  → 纯文本:  cat post.md
  → 静态站:  goldmark → HTML → 任意模板

HTML 导出:
  → PDF:    浏览器打印 (依赖浏览器，CSS 打印样式需额外维护)
  → Word:   格式丢失严重，需手动调整
  → ePub:   需 strip 内联样式，转换复杂
  → 纯文本: lynx -dump -stdin < post.html
  → 静态站: 可直接嵌（但带有 blog-platform 特定 class）
```

#### 最终决策矩阵

```
┌────────────────────────────────────────────────────────────────┐
│  权重  │  维度         │  Markdown  │  HTML    │  对本项目影响  │
├────────┼───────────────┼────────────┼──────────┼───────────────┤
│  ★★★   │  安全性       │  ★★★★★    │  ★★      │  核心考量      │
│  ★★★   │  搜索质量     │  ★★★★★    │  ★★      │  核心功能      │
│  ★★★   │  API/多端     │  ★★★★★    │  ★★      │  微服务架构    │
│  ★★    │  存储成本     │  ★★★★★    │  ★★      │  长期运营      │
│  ★★    │  可移植性     │  ★★★★★    │  ★★      │  避免锁定      │
│  ★★    │  版本控制     │  ★★★★★    │  ★       │  开发流程      │
│  ★     │  编辑体验     │  ★★★★     │  ★★★★★   │  个人使用      │
│  ★     │  排版能力     │  ★★★      │  ★★★★★   │  非必需        │
│  ★     │  非技术用户   │  ★★       │  ★★★★★   │  目标用户是开发者│
├────────┼───────────────┼────────────┼──────────┼───────────────┤
│        │  加权总分     │  4.7/5     │  2.3/5   │               │
└────────┴───────────────┴────────────┴──────────┴───────────────┘
```

**结论：Markdown 在安全、搜索、API 友好、存储、可移植性等关键维度全面领先。HTML 仅在"所见即所得"和"精准排版控制"上有优势，这两个需求在个人技术博客场景中权重很低。**

## 13. 性能指标

| 指标 | 目标 | 监控方式 |
|------|------|---------|
| 文章详情 P99 | < 50ms（命中缓存）/ < 150ms（miss） | Jaeger Span |
| 文章列表 P99 | < 100ms（缓存）/ < 300ms（miss） | Jaeger Span |
| 搜索 P99 | < 200ms | Jaeger Span |
| 写操作 P99 | < 300ms | Jaeger Span |
| 缓存命中率 | > 85%（详情）/ > 70%（列表） | Redis INFO |
| DB 连接池使用率 | < 80% | GORM Stats |
| 浏览计数刷新延迟 | < 6min | task metrics |
