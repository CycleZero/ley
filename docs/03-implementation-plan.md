# 实现计划

## 1. 总体实施路线图

```
Phase 0: 基础设施搭建 (1-2 天)
  ├── Docker Compose 开发环境
  ├── Proto 定义规范与代码生成
  ├── 配置管理框架 (etcd)
  └── 公共代码库 (pkg/) 补全

Phase 1: 用户服务 (3-4 天)
  ├── Proto 定义与生成
  ├── 数据模型与 GORM 迁移
  ├── 注册/登录/Token 业务逻辑
  └── gRPC + HTTP 服务暴露

Phase 2: 文章服务 (4-5 天)
  ├── Proto 定义与生成
  ├── 数据模型与全文搜索索引
  ├── CRUD + 标签/分类管理
  └── 缓存与搜索功能

Phase 3: 评论服务 (2-3 天)
  ├── Proto 定义与生成
  ├── 评论 CRUD + 树形构建
  ├── 敏感词过滤
  └── 事件集成

Phase 4: 文件服务 (2-3 天)
  ├── Proto 定义与生成
  ├── MinIO 集成
  ├── 上传/下载/预签名
  └── 文件校验与安全

Phase 5: 网关服务 (1-2 天)
  ├── HTTP 路由映射
  ├── JWT 验证中间件
  ├── gRPC 客户端管理
  └── CORS 与限流

Phase 6: 事件驱动集成 (2-3 天)
  ├── NATS JetStream 初始化
  ├── 事件生产者/消费者
  └── 跨服务事件编排

Phase 7: 集成测试与调优 (2-3 天)
  ├── 端到端测试
  ├── 性能压测
  └── 文档补全
```

**总预计工期：16-25 天（单人全职）**

---

## 2. 项目目录结构

```
ley/
├── api/                              # Proto 定义（共享契约）
│   ├── user/
│   │   └── v1/
│   │       ├── user.proto            # 用户服务接口
│   │       └── user_error.proto      # 用户错误码
│   ├── post/
│   │   └── v1/
│   │       ├── post.proto
│   │       └── post_error.proto
│   ├── comment/
│   │   └── v1/
│   │       ├── comment.proto
│   │       └── comment_error.proto
│   └── file/
│       └── v1/
│           ├── file.proto
│           └── file_error.proto
│
├── cmd/                              # 服务入口
│   ├── ley-gateway/
│   │   ├── main.go
│   │   ├── wire.go
│   │   └── wire_gen.go
│   ├── ley-user/
│   │   ├── main.go
│   │   ├── wire.go
│   │   └── wire_gen.go
│   ├── ley-post/
│   │   ├── main.go
│   │   ├── wire.go
│   │   └── wire_gen.go
│   ├── ley-comment/
│   │   ├── main.go
│   │   ├── wire.go
│   │   └── wire_gen.go
│   └── ley-file/
│       ├── main.go
│       ├── wire.go
│       └── wire_gen.go
│
├── internal/                         # 私有应用代码
│   ├── gateway/                      # 网关专用代码
│   │   ├── biz/
│   │   │   ├── biz.go                # ProviderSet
│   │   │   └── router.go             # 路由编排
│   │   ├── conf/
│   │   │   ├── conf.proto
│   │   │   └── conf.pb.go
│   │   ├── server/
│   │   │   └── http.go
│   │   └── service/
│   │       ├── service.go
│   │       └── gateway.go            # HTTP Handler Mux + 转发
│   │
│   ├── user/                         # 用户服务
│   │   ├── biz/
│   │   │   ├── biz.go                # ProviderSet
│   │   │   ├── user.go               # User UseCase
│   │   │   └── auth.go               # Auth UseCase
│   │   ├── conf/
│   │   │   ├── conf.proto
│   │   │   └── conf.pb.go
│   │   ├── data/
│   │   │   ├── data.go               # Data 层初始化
│   │   │   ├── user.go               # UserRepo 实现
│   │   │   └── user_cache.go         # User Redis 缓存
│   │   ├── server/
│   │   │   ├── server.go
│   │   │   ├── grpc.go
│   │   │   └── http.go
│   │   └── service/
│   │       ├── service.go
│   │       ├── user.go               # UserService 实现
│   │       └── auth.go               # Auth 接口实现
│   │
│   ├── post/                         # 文章服务（同上结构）
│   ├── comment/                      # 评论服务（同上结构）
│   └── file/                         # 文件服务（同上结构）
│
├── pkg/                              # 公共库（已有，按需补充）
│   ├── cache/        # Redis 缓存抽象 ✓
│   ├── common/       # 通用工具
│   ├── constant/     # 常量定义 ✓
│   ├── infra/        # 基础设施 (DB/Redis/NATS/MinIO/etcd) ✓
│   ├── jwt/          # JWT 认证 ✓
│   ├── log/          # Zap 日志 ✓
│   ├── meta/         # 元数据传递 ✓
│   ├── middleware/    # 中间件 ✓
│   ├── mq/           # 消息队列抽象 (NATS) ✓
│   ├── oss/          # 对象存储 (MinIO/COS) ✓
│   ├── security/     # 密码/安全 ✓
│   ├── task/         # 本地任务队列 ✓
│   ├── testutil/     # 测试工具 ✓
│   ├── trace/        # 链路追踪 ✓
│   └── util/         # 工具集 ✓
│
├── configs/                          # 本地配置文件
│   ├── gateway.yaml
│   ├── user.yaml
│   ├── post.yaml
│   ├── comment.yaml
│   └── file.yaml
│
├── docker/                           # Docker 相关
│   ├── docker-compose.yml
│   ├── docker-compose.infra.yml     # 仅基础设施
│   └── Dockerfile
│
├── docs/                             # 文档
│   ├── 01-technical-selection.md     # 技术选型 ✓
│   ├── 02-system-design.md           # 系统设计 ✓
│   └── 03-implementation-plan.md     # 实现计划（本文档）
│
├── Makefile                          # 构建命令 ✓
├── go.mod
└── go.sum
```

**标注 ✓ 的文件已存在或已在现有项目中实现。**

---

## 3. Phase 0: 基础设施搭建

### 3.1 Docker Compose 开发环境

**目标：** 一键启动所有依赖的基础设施组件。

**文件：** `docker/docker-compose.infra.yml`

需要包含以下服务：

| 服务 | 镜像 | 端口 | 挂载 | 健康检查 |
|------|------|------|------|---------|
| postgres | postgres:16-alpine | 5432 | 数据卷 | pg_isready |
| redis | redis:7-alpine | 6379 | 数据卷 | redis-cli ping |
| nats | nats:2-alpine | 4222/8222 | 数据卷 | /healthz |
| etcd | bitnami/etcd:3.6 | 2379 | 数据卷 | etcdctl |
| jaeger | jaegertracing/all-in-one | 4318/16686 | 无 | HTTP / |
| minio | minio/minio:latest | 9000/9001 | 数据卷 | HTTP /minio/health/live |

**初始化脚本需求：**

1. PostgreSQL 初始化脚本：创建 `ley` 数据库和 4 个 Schema（user/post/comment/file）
2. MinIO 初始化：创建 4 个 Bucket（ley-images/ley-attachments/ley-avatars/ley-temp）
3. NATS 初始化：创建 JetStream 和 Stream

```bash
# 启动命令
make infra-up     # 启动全部基础设施
make infra-down   # 停止并清理
make infra-reset  # 清理数据卷并重启
```

### 3.2 Proto 代码生成规范

**文件组织规范：**

```
api/{service}/v1/{service}.proto       # 服务 RPC 定义
api/{service}/v1/{service}_model.proto # 公共消息模型（可选）
api/{service}/v1/{service}_error.proto # 错误码枚举
```

**Makefile 扩展：**

```makefile
api-user:
	protoc --proto_path=./api \
		--proto_path=./third_party \
		--go_out=paths=source_relative:./api \
		--go-http_out=paths=source_relative:./api \
		--go-grpc_out=paths=source_relative:./api \
		--openapi_out=fq_schema_naming=true,default_response=false:. \
		api/user/v1/user.proto

api-post:
	# 同上

api-all: api-user api-post api-comment api-file
```

### 3.3 公共代码库补全 (pkg/)

现有 `pkg/` 已较完善，需补全/修改的部分：

| 模块 | 当前状态 | 需做工作 |
|------|---------|---------|
| `pkg/infra/minio.go` | 全部注释 | 取消注释，完善 MinIO 客户端初始化（复用 `pkg/oss/minio.go`） |
| `pkg/infra/provider.go` | 全部注释 | 定义 Wire ProviderSet，暴露 infra 单例 |
| `pkg/infra/cron.go` | 缺少 import | 修复全局 cron 实现 |
| `pkg/infra/milvus.go` | 全部注释 | 移除或保留注释（当前项目不需要） |
| `pkg/util/tools.go` | 存在 | 补充 `Paginate` 的 Offset/Limit 辅助函数 |
| `pkg/constant/app.go` | 存在 | 更新服务名称列表，对齐实际微服务 |

---

## 4. Phase 1: 用户服务实现

### 4.1 文件创建清单

```
创建 api/user/v1/user.proto
创建 api/user/v1/user_error.proto
创建 cmd/ley-user/main.go
创建 cmd/ley-user/wire.go
创建 internal/user/biz/biz.go
创建 internal/user/biz/user.go
创建 internal/user/biz/auth.go
创建 internal/user/conf/conf.proto
创建 internal/user/data/data.go
创建 internal/user/data/user.go
创建 internal/user/data/user_cache.go
创建 internal/user/server/server.go
创建 internal/user/server/grpc.go
创建 internal/user/server/http.go
创建 internal/user/service/service.go
创建 internal/user/service/auth.go
创建 configs/user.yaml
```

### 4.2 Proto 定义

```protobuf
// api/user/v1/user.proto
syntax = "proto3";
package user.v1;
option go_package = "ley/api/user/v1;userv1";

import "google/api/annotations.proto";
import "validate/validate.proto";
import "openapi/v3/annotations.proto";

service UserService {
  // 注册
  rpc Register(RegisterRequest) returns (RegisterReply) {
    option (google.api.http) = {
      post: "/api/v1/auth/register"
      body: "*"
    };
  }
  // 登录
  rpc Login(LoginRequest) returns (LoginReply) {
    option (google.api.http) = {
      post: "/api/v1/auth/login"
      body: "*"
    };
  }
  // 刷新令牌
  rpc RefreshToken(RefreshTokenRequest) returns (LoginReply) {
    option (google.api.http) = {
      post: "/api/v1/auth/refresh"
      body: "*"
    };
  }
  // 获取当前用户信息
  rpc GetProfile(GetProfileRequest) returns (GetProfileReply) {
    option (google.api.http) = {
      get: "/api/v1/users/me"
    };
  }
  // 更新当前用户信息
  rpc UpdateProfile(UpdateProfileRequest) returns (UpdateProfileReply) {
    option (google.api.http) = {
      put: "/api/v1/users/me"
      body: "*"
    };
  }
  // 内部 RPC：获取用户信息
  rpc GetUser(GetUserRequest) returns (GetUserReply);
  // 内部 RPC：批量获取用户信息
  rpc BatchGetUsers(BatchGetUsersRequest) returns (BatchGetUsersReply);
}

message RegisterRequest {
  string username = 1 [(validate.rules).string = {min_len: 3, max_len: 32, pattern: "^[a-zA-Z0-9_-]+$"}];
  string email    = 2 [(validate.rules).string.email = true];
  string password = 3 [(validate.rules).string = {min_len: 8, max_len: 64}];
  string nickname = 4 [(validate.rules).string = {max_len: 64}];
}

message RegisterReply {
  int64  id       = 1;
  string username = 2;
  string email    = 3;
  string nickname = 4;
  string created_at = 5;
}

message LoginRequest {
  string account  = 1; // username or email
  string password = 2;
}

message LoginReply {
  UserInfo user          = 1;
  TokenPair token_pair   = 2;
}

message RefreshTokenRequest {
  string refresh_token = 1;
}

message GetProfileRequest {}

message GetProfileReply {
  UserInfo user = 1;
}

message UpdateProfileRequest {
  string nickname = 1 [(validate.rules).string = {max_len: 64}];
  string avatar   = 2;
  string bio      = 3;
}

message UpdateProfileReply {
  UserInfo user = 1;
}

message GetUserRequest {
  string user_id = 1;
}

message GetUserReply {
  UserInfo user = 1;
}

message BatchGetUsersRequest {
  repeated string user_ids = 1;
}

message BatchGetUsersReply {
  repeated UserInfo users = 1;
}

message UserInfo {
  string id        = 1;
  string username  = 2;
  string email     = 3;
  string nickname  = 4;
  string avatar    = 5;
  string bio       = 6;
  string role      = 7;
  string created_at = 8;
}

message TokenPair {
  string access_token  = 1;
  string refresh_token = 2;
  int64  expires_in    = 3; // access token TTL in seconds
}
```

### 4.3 业务逻辑核心实现

**`internal/user/biz/user.go` - UserUseCase：**

```
Register(ctx, username, email, password, nickname) → (User, error)
  ├── 校验用户名/邮箱唯一性 (Repo)
  ├── 校验密码强度
  ├── bcrypt 哈希密码
  ├── 生成 UUID v7
  ├── 创建用户 (Repo)
  ├── 发布 user.registered 事件 (EventBus)
  └── 返回用户信息

Login(ctx, account, password) → (User, TokenPair, error)
  ├── 查询用户（用户名或邮箱）(Repo + Cache)
  ├── 验证密码 (bcrypt.CompareHashAndPassword)
  ├── 检查账号状态
  ├── 检查登录频率限制 (Redis)
  ├── 生成 TokenPair (JWT)
  ├── 存储 RefreshToken (Redis)
  ├── 发布 user.logged_in 事件
  └── 返回用户 + TokenPair

RefreshToken(ctx, refreshToken) → (TokenPair, error)
  ├── 解析 RefreshToken
  ├── 校验 Redis 中是否存在（未被撤销）
  ├── 生成新的 TokenPair
  ├── 旧 RefreshToken 加入黑名单
  ├── 存储新 RefreshToken
  └── 返回新 TokenPair

GetProfile(ctx) → (User, error)
  ├── 从 Context 获取 UserID
  ├── 查询用户信息 (Repo + Cache)
  └── 返回用户信息

UpdateProfile(ctx, nickname, avatar, bio) → (User, error)
  ├── 从 Context 获取 UserID
  ├── 查询用户 (Repo)
  ├── 更新字段
  ├── 保存 (Repo)
  ├── 清除缓存
  ├── 发布 user.updated 事件
  └── 返回更新后用户
```

### 4.4 数据层实现要点

**`internal/user/data/user.go`：**
- 使用 `gorm.io/gorm` 操作 `"user".users` 表
- 实现 `biz.UserRepo` 接口
- 查询加入软删除过滤 `WHERE deleted_at IS NULL`
- 使用 `clause.OnConflict` 处理唯一约束冲突

**`internal/user/data/user_cache.go`：**
- 使用 `pkg/cache` 的 `Cache` 接口
- Key: `cache:user:{id}`, TTL: 30min
- 采用 Cache-Aside 模式
- 更新后主动删除缓存

### 4.5 Wire 依赖注入链

```
NewConfig      → *conf.Bootstrap
NewData        → *data.Data (DB, Redis)
NewUserRepo    → biz.UserRepo
NewUserUseCase → *biz.UserUseCase
NewUserService → *service.UserService
NewGRPCServer  → *grpc.Server (register UserService)
NewHTTPServer  → *http.Server (register UserHTTPServer)
wireApp        → *kratos.App
```

### 4.6 验证标准

- [ ] `POST /api/v1/auth/register` 返回 201 + 用户信息
- [ ] `POST /api/v1/auth/login` 返回 200 + TokenPair
- [ ] `GET /api/v1/users/me` (Bearer Token) 返回用户信息
- [ ] 重复用户名/邮箱注册返回 409
- [ ] 错误密码登录返回 401
- [ ] 禁用账号登录返回 403
- [ ] Refresh Token 能获取新 Token
- [ ] 单元测试覆盖率 > 70%

---

## 5. Phase 2: 文章服务实现

### 5.1 文件创建清单

```
创建 api/post/v1/post.proto
创建 api/post/v1/post_error.proto
创建 cmd/ley-post/main.go
创建 cmd/ley-post/wire.go
创建 internal/post/biz/biz.go
创建 internal/post/biz/post.go
创建 internal/post/biz/tag.go
创建 internal/post/biz/category.go
创建 internal/post/biz/search.go
创建 internal/post/conf/conf.proto
创建 internal/post/data/data.go
创建 internal/post/data/post.go
创建 internal/post/data/tag.go
创建 internal/post/data/category.go
创建 internal/post/data/post_cache.go
创建 internal/post/data/search.go
创建 internal/post/server/server.go
创建 internal/post/server/grpc.go
创建 internal/post/server/http.go
创建 internal/post/service/service.go
创建 internal/post/service/post.go
创建 internal/post/service/tag.go
创建 internal/post/service/category.go
创建 configs/post.yaml
```

### 5.2 Proto 定义核心

```protobuf
// api/post/v1/post.proto
service PostService {
  rpc CreatePost(CreatePostRequest) returns (CreatePostReply);
  rpc UpdatePost(UpdatePostRequest) returns (UpdatePostReply);
  rpc DeletePost(DeletePostRequest) returns (DeletePostReply);
  rpc GetPost(GetPostRequest) returns (GetPostReply);
  rpc ListPosts(ListPostsRequest) returns (ListPostsReply);
  rpc SearchPosts(SearchPostsRequest) returns (SearchPostsReply);
  rpc PublishPost(PublishPostRequest) returns (PublishPostReply);
  
  rpc CreateTag(CreateTagRequest) returns (CreateTagReply);
  rpc ListTags(ListTagsRequest) returns (ListTagsReply);
  rpc DeleteTag(DeleteTagRequest) returns (DeleteTagReply);
  
  rpc CreateCategory(CreateCategoryRequest) returns (CreateCategoryReply);
  rpc ListCategories(ListCategoriesRequest) returns (ListCategoriesReply);
  rpc DeleteCategory(DeleteCategoryRequest) returns (DeleteCategoryReply);
  
  // 内部 RPC
  rpc IncrementViewCount(IncrementViewCountRequest) returns (IncrementViewCountReply);
  rpc GetPostID(GetPostIDRequest) returns (GetPostIDReply);
}

message CreatePostRequest {
  string title       = 1;
  string content     = 2;
  string excerpt     = 3;
  string cover_image = 4;
  int64  category_id = 5;
  repeated string tags = 6;
}

message ListPostsRequest {
  int32  page        = 1;
  int32  page_size   = 2;
  string status      = 3;  // draft/published/archived
  int64  category_id = 4;
  repeated string tags = 5;
  string sort_by     = 6;  // created_at/published_at/view_count
  string sort_order  = 7;  // asc/desc
}

message PostInfo {
  string id            = 1;
  string title         = 2;
  string slug          = 3;
  string content       = 4;
  string excerpt       = 5;
  string cover_image   = 6;
  string status        = 7;
  int64  author_id     = 8;
  int64  category_id   = 9;
  repeated TagInfo tags = 10;
  int64  view_count    = 11;
  int64  like_count    = 12;
  int64  comment_count = 13;
  bool   is_top        = 14;
  string published_at  = 15;
  string created_at    = 16;
  string updated_at    = 17;
}

message TagInfo {
  int64  id         = 1;
  string name       = 2;
  string slug       = 3;
  int64  post_count = 4;
}

message CategoryInfo {
  int64  id          = 1;
  string name        = 2;
  string slug        = 3;
  string description = 4;
  int64  parent_id   = 5;
  int32  sort_order  = 6;
  int64  post_count  = 7;
}
```

### 5.3 自增 ID 与数据隔离

使用 PostgreSQL `BIGSERIAL` 作为主键（内部使用），对外暴露 UUID v7。

```go
// 雪花算法替代方案：UUID v7（时间排序友好）
import "github.com/google/uuid"

func NewPostID() string {
    id, _ := uuid.NewV7()
    return id.String()
}
```

> 注：项目现有 `pkg/task/id.go` 实现了基于原子计数器 + 纳秒时间戳的 ID 生成，如需无依赖方案可复用。

### 5.4 全文搜索实现

```go
// internal/post/data/search.go
func (r *postRepo) Search(ctx context.Context, query string, page, pageSize int) ([]*biz.Post, int64, error) {
    db := r.data.DB.WithContext(ctx)
    
    var posts []*PostModel
    var total int64
    
    // 计数
    db.Table(`"post".posts`).
        Where("search_vector @@ plainto_tsquery('simple', ?)", query).
        Where("status = ?", 1).  // only published
        Where("deleted_at IS NULL").
        Count(&total)
    
    // 搜索 + 排序
    db.Table(`"post".posts`).
        Select("*, ts_rank(search_vector, plainto_tsquery('simple', ?)) AS rank", query).
        Where("search_vector @@ plainto_tsquery('simple', ?)", query).
        Where("status = ?", 1).
        Where("deleted_at IS NULL").
        Order("rank DESC").
        Offset((page - 1) * pageSize).
        Limit(pageSize).
        Find(&posts)
    
    return toBizPosts(posts), total, nil
}
```

### 5.5 Slug 生成策略

```go
// 中文/英文 slug 处理
func GenerateSlug(title string) string {
    // 1. 转小写
    // 2. 中文转拼音首字母（可选），英文保留
    // 3. 替换空格为 "-"
    // 4. 移除特殊字符，仅保留 [a-z0-9-]
    // 5. 连续 "-" 合并为一个
    // 6. 去除首尾 "-"
    // 7. 截断至 200 字符
    // 8. 若重复，追加 "-{n}" 后缀
}
```

### 5.6 验证标准

- [ ] 文章完整 CRUD 通过
- [ ] 列表接口支持分页、排序、按状态/分类/标签筛选
- [ ] Slug 自动生成且唯一
- [ ] 全文搜索返回相关排序结果
- [ ] 文章详情走缓存（Cache-Aside）
- [ ] 标签/分类增删改正常
- [ ] 作者只能操作自己的文章
- [ ] 单元测试覆盖率 > 70%

---

## 6. Phase 3: 评论服务实现

### 6.1 文件创建清单

```
创建 api/comment/v1/comment.proto
创建 api/comment/v1/comment_error.proto
创建 cmd/ley-comment/main.go & wire.go
创建 internal/comment/biz/{biz,comment,filter}.go
创建 internal/comment/conf/conf.proto
创建 internal/comment/data/{data,comment}.go
创建 internal/comment/server/{server,grpc,http}.go
创建 internal/comment/service/{service,comment}.go
创建 configs/comment.yaml
```

### 6.2 评论树构建算法

```go
// internal/comment/biz/comment.go
type CommentNode struct {
    Comment  *Comment
    Children []*CommentNode
}

func BuildCommentTree(comments []*Comment) []*CommentNode {
    nodeMap := make(map[int64]*CommentNode, len(comments))
    var roots []*CommentNode
    
    for _, c := range comments {
        nodeMap[c.ID] = &CommentNode{Comment: c}
    }
    
    for _, c := range comments {
        node := nodeMap[c.ID]
        if c.ParentID != nil {
            if parent, ok := nodeMap[*c.ParentID]; ok {
                parent.Children = append(parent.Children, node)
            }
        } else {
            roots = append(roots, node)
        }
    }
    
    sortByCreatedAt(roots) // 递归排序
    
    return roots
}
```

### 6.3 敏感词过滤

```go
// internal/comment/biz/filter.go
type SensitiveWordFilter struct {
    trie *Trie // Trie 树 + AC 自动机
}

func (f *SensitiveWordFilter) Filter(text string) (filtered string, hasSensitive bool) {
    matches := f.trie.Search(text)
    if len(matches) > 0 {
        hasSensitive = true
        filtered = f.trie.Replace(text, "***")
        return
    }
    return text, false
}
```

敏感词库来源：内置基础词库 + 支持远程配置更新（etcd Watch）。

### 6.4 验证标准

- [ ] 评论 CRUD 通过
- [ ] 评论树结构正确（嵌套回复）
- [ ] 敏感词过滤生效
- [ ] 评论频率限制生效
- [ ] 包含链接的评论自动待审核
- [ ] 非文章作者只能删除自己的评论
- [ ] 单元测试覆盖率 > 70%

---

## 7. Phase 4: 文件服务实现

### 7.1 文件创建清单

```
创建 api/file/v1/file.proto
创建 api/file/v1/file_error.proto
创建 cmd/ley-file/main.go & wire.go
创建 internal/file/biz/{biz,file}.go
创建 internal/file/conf/conf.proto
创建 internal/file/data/{data,file}.go
创建 internal/file/server/{server,grpc,http}.go
创建 internal/file/service/{service,file}.go
创建 configs/file.yaml
```

### 7.2 上传流程伪代码

```go
// internal/file/biz/file.go
func (uc *FileUseCase) Upload(ctx context.Context, reader io.Reader, filename string, mimeType string, userID int64) (*File, error) {
    // 1. 校验 MIME Type 白名单
    if !uc.allowedTypes.Contains(mimeType) {
        return nil, ErrInvalidFileType
    }
    
    // 2. 校验文件大小（读取时限制）
    limitReader := io.LimitReader(reader, uc.maxSize)
    
    // 3. 读取到临时 buffer，计算 MD5
    var buf bytes.Buffer
    tee := io.TeeReader(limitReader, &buf)
    hash := md5.New()
    io.Copy(hash, tee)
    md5hex := hex.EncodeToString(hash.Sum(nil))
    
    // 4. 检测是否重复上传（按 MD5）
    if existing, _ := uc.repo.FindByMD5(ctx, userID, md5hex); existing != nil {
        return existing, nil
    }
    
    // 5. 生成 ObjectKey: {bucket}/YYYY/MM/DD/{uuid}.{ext}
    objectKey := uc.generateObjectKey(filename)
    
    // 6. 上传到 MinIO
    err := uc.oss.PutObject(ctx, bucket, objectKey, &buf, int64(buf.Len()), mimeType)
    if err != nil {
        return nil, err
    }
    
    // 7. 组装 URL
    fileURL := fmt.Sprintf("%s/%s/%s", uc.endpoint, bucket, objectKey)
    
    // 8. 写入元数据库
    file := &File{
        UserID:    userID,
        Bucket:    bucket,
        ObjectKey: objectKey,
        Filename:  filename,
        MimeType:  mimeType,
        Size:      int64(buf.Len()),
        MD5Hash:   md5hex,
        URL:       fileURL,
    }
    err = uc.repo.Create(ctx, file)
    if err != nil {
        // 回滚：删除 MinIO 对象
        _ = uc.oss.DeleteObject(ctx, bucket, objectKey)
        return nil, err
    }
    
    // 9. 发布 file.uploaded 事件
    uc.eventBus.Publish(ctx, "file.uploaded", file)
    
    return file, nil
}
```

### 7.3 验证标准

- [ ] 图片上传成功并返回 URL
- [ ] MIME 白名单校验生效
- [ ] 文件大小限制生效
- [ ] 重复 MD5 文件秒传
- [ ] 预签名上传 URL 有效
- [ ] 文件删除成功（DB 软删除 + MinIO 对象删除）
- [ ] 上传失败时 MinIO 对象回滚

---

## 8. Phase 5: 网关服务实现

### 8.1 文件创建清单

```
创建 cmd/ley-gateway/main.go & wire.go
创建 internal/gateway/biz/{biz,router}.go
创建 internal/gateway/conf/conf.proto
创建 internal/gateway/server/http.go
创建 internal/gateway/service/{service,gateway}.go
创建 configs/gateway.yaml
```

### 8.2 路由映射设计

```go
// internal/gateway/biz/router.go
type Route struct {
    Path    string
    Method  string
    Target  string  // gRPC method full name
    Service string  // 目标服务名（服务发现用）
    Auth    bool    // 是否需要认证
}

var Routes = []Route{
    // Auth
    {"/api/v1/auth/register", "POST", "user.v1.UserService/Register", "ley-user", false},
    {"/api/v1/auth/login", "POST", "user.v1.UserService/Login", "ley-user", false},
    {"/api/v1/auth/refresh", "POST", "user.v1.UserService/RefreshToken", "ley-user", false},
    
    // User
    {"/api/v1/users/me", "GET", "user.v1.UserService/GetProfile", "ley-user", true},
    {"/api/v1/users/me", "PUT", "user.v1.UserService/UpdateProfile", "ley-user", true},
    
    // Post
    {"/api/v1/posts", "GET", "post.v1.PostService/ListPosts", "ley-post", false},
    {"/api/v1/posts", "POST", "post.v1.PostService/CreatePost", "ley-post", true},
    {"/api/v1/posts/{id}", "GET", "post.v1.PostService/GetPost", "ley-post", false},
    {"/api/v1/posts/{id}", "PUT", "post.v1.PostService/UpdatePost", "ley-post", true},
    {"/api/v1/posts/{id}", "DELETE", "post.v1.PostService/DeletePost", "ley-post", true},
    
    // Comment
    {"/api/v1/posts/{id}/comments", "GET", "comment.v1.CommentService/ListComments", "ley-comment", false},
    {"/api/v1/posts/{id}/comments", "POST", "comment.v1.CommentService/CreateComment", "ley-comment", true},
    
    // File
    {"/api/v1/files/upload", "POST", "file.v1.FileService/Upload", "ley-file", true},
    
    // Tag / Category (read-only for all)
    {"/api/v1/tags", "GET", "post.v1.PostService/ListTags", "ley-post", false},
    {"/api/v1/categories", "GET", "post.v1.PostService/ListCategories", "ley-post", false},
}
```

### 8.3 中间件链

```go
// 请求进来顺序
HTTP Request
  → Recovery Middleware (panic recovery)
  → Tracing Middleware (OpenTelemetry span start)
  → Logging Middleware (request log)
  → CORS Middleware
  → RateLimit Middleware
  → Meta Middleware (metadata extraction/propagation)
  → Auth Middleware (JWT verification, inject user info)
  → Route Handler (gRPC client call to target service)
```

### 8.4 验证标准

- [ ] 所有 API 路由正确转发到对应服务
- [ ] JWT 认证中间件正确拦截未认证请求
- [ ] 白名单路径（注册/登录）无需认证可访问
- [ ] 限流中间件生效
- [ ] 请求 Tracing ID 正确传递

---

## 9. Phase 6: 事件驱动集成

### 9.1 NATS JetStream 初始化

```go
// 在 main.go 中或通过 infra 层统一初始化
func InitJetStream(conn *nats.Conn) (nats.JetStreamContext, error) {
    js, err := conn.JetStream()
    if err != nil {
        return nil, err
    }
    
    streams := []*nats.StreamConfig{
        {
            Name:      "EVENTS-POST",
            Subjects:  []string{"post.created", "post.updated", "post.published", "post.deleted", "post.liked", "post.unliked"},
            MaxAge:    7 * 24 * time.Hour,
            MaxMsgs:   1_000_000,
            Storage:   nats.FileStorage,
            Replicas:  1,
        },
        {
            Name:      "EVENTS-COMMENT",
            Subjects:  []string{"comment.created", "comment.approved", "comment.deleted"},
            MaxAge:    7 * 24 * time.Hour,
            MaxMsgs:   1_000_000,
            Storage:   nats.FileStorage,
        },
        {
            Name:      "EVENTS-USER",
            Subjects:  []string{"user.registered", "user.updated"},
            MaxAge:    30 * 24 * time.Hour,
            MaxMsgs:   100_000,
            Storage:   nats.FileStorage,
        },
        {
            Name:      "EVENTS-FILE",
            Subjects:  []string{"file.uploaded", "file.deleted"},
            MaxAge:    7 * 24 * time.Hour,
            MaxMsgs:   100_000,
            Storage:   nats.FileStorage,
        },
    }
    
    for _, cfg := range streams {
        _, err := js.AddStream(cfg)
        if err != nil && !errors.Is(err, nats.ErrStreamNameAlreadyInUse) {
            return nil, err
        }
    }
    
    return js, nil
}
```

### 9.2 事件消费者注册

每个服务在启动时注册自己关心的事件消费者：

```go
// ley-post 启动时
js.QueueSubscribe("comment.created", "post-service", func(msg *nats.Msg) {
    var event CommentCreatedEvent
    json.Unmarshal(msg.Data, &event)
    
    // 幂等更新评论数
    svc.updateCommentCount(ctx, event.PostID, +1)
    
    msg.Ack()
})
```

### 9.3 事件结构规范

```go
// 所有事件统一的信封结构
type EventEnvelope struct {
    EventID   string    `json:"event_id"`   // UUID v7
    EventType string    `json:"event_type"` // "post.created"
    Timestamp time.Time `json:"timestamp"`
    TraceID   string    `json:"trace_id"`   // 关联追踪
    Source    string    `json:"source"`      // "ley-post"
    Version   string    `json:"version"`     // 事件版本
    Payload   json.RawMessage `json:"payload"`
    IdempotencyKey string `json:"idempotency_key"` // 幂等 Key
}
```

### 9.4 幂等性保证

所有消费者在收到事件后，以 `IdempotencyKey` 写入 Redis：
- `SET NX idempotent:{key} 1 EX 3600` → 成功则处理，失败则跳过
- 避免 NATS 重试导致重复消费

### 9.5 验证标准

- [ ] 文章发布后，事件发布成功
- [ ] 评论创建后，文章评论计数正确更新
- [ ] 消费者重连后不会丢失消息（JetStream Persist）
- [ ] 幂等 Key 机制防止重复消费
- [ ] 错误消息正确 NAK/Term，进入死信队列

---

## 10. Phase 7: 测试与调优

### 10.1 测试策略

```
┌─────────────────────────────────────────────────────────────────┐
│ 测试层级                                                         │
├───────────────┬─────────────────────────────────────────────────┤
│ 单元测试       │ go test ./... (每个 internal/*/biz/)             │
│               │ 使用 mock 替换 data 层 (testify/mock 或 gomock)   │
│               │ 覆盖率目标 > 70% (核心 biz 逻辑 > 80%)            │
├───────────────┼─────────────────────────────────────────────────┤
│ 集成测试       │ 使用真实 PostgreSQL/Redis/NATS (docker-compose)   │
│               │ 测试完整的 service → biz → data 链路              │
│               │ 关键场景：注册→登录→发布→评论→搜索                 │
├───────────────┼─────────────────────────────────────────────────┤
│ 端到端测试     │ 通过 HTTP Client 模拟用户操作                     │
│               │ 使用 testutil.Suite 框架                          │
│               │ 场景：用户完整旅程                                 │
├───────────────┼─────────────────────────────────────────────────┤
│ 性能测试       │ go-wrk / vegeta 压测                             │
│               │ 关注 QPS、P99 延迟、缓存命中率                     │
└───────────────┴─────────────────────────────────────────────────┘
```

### 10.2 单元测试示例

```go
// internal/post/biz/post_test.go
func TestPostUseCase_CreatePost(t *testing.T) {
    // 1. 创建 mock repo
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()
    mockRepo := NewMockPostRepo(ctrl)
    mockEventBus := NewMockEventBus(ctrl)
    
    // 2. 配置期望
    mockRepo.EXPECT().
        FindBySlug(gomock.Any(), "hello-world").
        Return(nil, ErrNotFound)
    mockRepo.EXPECT().
        Create(gomock.Any(), gomock.Any()).
        Return(nil)
    mockEventBus.EXPECT().
        Publish(gomock.Any(), "post.created", gomock.Any()).
        Return(nil)
    
    // 3. 创建 usecase 并执行
    uc := NewPostUseCase(mockRepo, mockEventBus, nil)
    post, err := uc.CreatePost(context.Background(), 
        "Hello World", "Content", "hello-world", 1, nil, nil)
    
    // 4. 断言
    assert.NoError(t, err)
    assert.Equal(t, "hello-world", post.Slug)
}
```

### 10.3 性能基准

```go
// Benchmark 测试
func BenchmarkCreatePost(b *testing.B) {
    // ... setup ...
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        uc.CreatePost(ctx, title, content, slug, authorID, tags, categoryID)
    }
}
```

### 10.4 性能调优清单

| 优化项 | 预期提升 | 实施位置 |
|--------|---------|---------|
| PostgreSQL 连接池 (PgBouncer) | 减少连接开销 | data 层 |
| Redis Pipeline 批量操作 | 减少 RTT x 10 倍 | cache 层 |
| gRPC Keepalive 连接复用 | 减少握手开销 | server 层 |
| GORM Preload 按需加载 | 减少 N+1 查询 | data 层 |
| 静态文件 CDN 化 | 减少应用服务器压力 | OSS 层 |
| 首页文章列表预热 | 减少首次访问延迟 | task 定时任务 |

### 10.5 Makefile 测试命令

```makefile
test: test-unit test-integration

test-unit:
	@echo "Running unit tests..."
	go test -v -short -count=1 -coverprofile=coverage.out ./internal/... ./pkg/...

test-integration:
	@echo "Running integration tests..."
	go test -v -count=1 -tags=integration ./internal/...

test-coverage:
	go tool cover -html=coverage.out

test-bench:
	go test -bench=. -benchmem ./internal/...

test-e2e:
	go test -v -count=1 -tags=e2e ./test/...
```

---

## 11. 开发工作流

### 11.1 分支策略

```
main                    # 稳定分支，随时可部署
  ├── develop           # 开发分支
  │   ├── feature/phase0-infra
  │   ├── feature/phase1-user
  │   ├── feature/phase2-post
  │   ├── feature/phase3-comment
  │   ├── feature/phase4-file
  │   ├── feature/phase5-gateway
  │   └── feature/phase6-events
  └── hotfix/xxx        # 紧急修复
```

### 11.2 开发流程

```
1. 从 develop 拉取分支: git checkout -b feature/phaseN-xxx develop
2. Proto 定义 → 代码生成: make api-{service}
3. 编写代码: go 分层实现 (biz → data → service → server → cmd)
4. 单元测试: make test-unit
5. 集成测试: make test-integration (确保 Docker 基础设施已启动)
6. 提交 PR → Code Review → 合并 develop
```

### 11.3 代码规范

| 规范 | 工具 | 说明 |
|------|------|------|
| 代码格式化 | `gofmt` / `goimports` | 强制性 |
| Linter | `golangci-lint` | 提交前检查 |
| Proto 规范 | `buf` | Proto Lint + Breaking Change |
| 提交信息 | Conventional Commits | `feat(user): add register API` |
| 错误处理 | 自定义错误码 + Kratos Error API | 使用 proto 定义错误枚举 |

### 11.4 Wire 依赖注入规范

```go
// 每个服务定义自己的 ProviderSet
var ProviderSet = wire.NewSet(
    NewUserUseCase,
    NewUserRepo,
    // ...
)
```

运行 `wire gen ./cmd/ley-user/...` 生成 `wire_gen.go`。

---

## 12. 配置文件模板

### 12.1 configs/user.yaml

```yaml
server:
  http:
    addr: 0.0.0.0:8001
    timeout: 30s
  grpc:
    addr: 0.0.0.0:9001
    timeout: 30s

data:
  database:
    driver: postgres
    source: host=localhost user=ley password=ley123 dbname=ley port=5432 sslmode=disable TimeZone=Asia/Shanghai
    max_open_conns: 50
    max_idle_conns: 10
    conn_max_lifetime: 3600s
  redis:
    addr: localhost:6379
    password: ""
    db: 0
    read_timeout: 0.2s
    write_timeout: 0.2s
  nats:
    host: localhost
    port: 4222
  etcd:
    endpoints:
      - localhost:2379

jwt:
  secret: "your-256-bit-secret-key-here-change-in-production"
  access_ttl: 900s        # 15 minutes
  refresh_ttl: 604800s    # 7 days

trace:
  endpoint: localhost:4318
  service_name: ley-user

service:
  name: ley-user
  version: 0.1.0
```

### 12.2 etcd 远程配置（可选）

```
路径: /configs/ley-user/jwt.secret
路径: /configs/ley-user/rate.limit
路径: /configs/ley-post/search.enabled
```

通过 `etcd Watch` 实现热更新，无需重启服务。

---

## 13. 阶段交付检查清单

### Phase 0 完成标准
- [ ] `make infra-up` 一键启动全部基础设施
- [ ] `make api-all` 生成所有 Proto 代码
- [ ] 数据库 Schema 自动创建
- [ ] MinIO Bucket 自动创建
- [ ] NATS Stream 自动创建

### Phase 1 完成标准
- [ ] 用户注册/登录/Token刷新链路通过
- [ ] 单元测试覆盖率 > 70%
- [ ] 集成测试通过
- [ ] 服务注册到 etcd 并能被发现

### Phase 2 完成标准
- [ ] 文章完整 CRUD + 发布流程通过
- [ ] 全文搜索功能可用
- [ ] 标签/分类管理通过
- [ ] 缓存策略生效

### Phase 3 完成标准
- [ ] 评论创建/审核/嵌套回复通过
- [ ] 敏感词过滤生效
- [ ] 评论事件发布成功

### Phase 4 完成标准
- [ ] 文件上传/下载/删除通过
- [ ] MD5 秒传生效
- [ ] 预签名 URL 功能可用

### Phase 5 完成标准
- [ ] 前端通过 Gateway 可访问全部 API
- [ ] JWT 认证拦截正确
- [ ] 限流/追踪正常

### Phase 6 完成标准
- [ ] 文章发布 → 缓存更新 → 计数同步事件链路正常
- [ ] 评论创建 → 文章评论计数 + 1 正确
- [ ] 幂等消费机制验证通过

### Phase 7 完成标准
- [ ] 端到端测试场景全部通过
- [ ] P99 延迟 < 200ms（读）/ < 500ms（写）
- [ ] 缓存命中率 > 80%
- [ ] 无内存泄漏、Goroutine 泄漏

---

## 14. 附录：关键依赖版本锁定

| 组件 | 版本 | 说明 |
|------|------|------|
| Go | 1.23+ | 编译器 |
| Kratos | v2.9.2 | 微服务框架 |
| GORM | v1.31.1 | ORM |
| PostgreSQL Driver | v1.6.0 | PG 驱动 |
| go-redis | v9.18.0 | Redis 客户端 |
| nats.go | v1.51.0 | NATS 客户端 |
| etcd/client | v3.6.10 | etcd 客户端 |
| minio-go | v7.0.100 | MinIO 客户端 |
| Zap | v1.27.1 | 日志库 |
| Wire | v0.6.0 | 依赖注入 |
| UUID | v1.6.0 | UUID 生成 |
| OpenTelemetry | v1.43.0 | 可观测性 |
| bcrypt (x/crypto) | v0.49.0 | 密码哈希 |
| lumberjack | v2.2.1 | 日志滚动 |

---

> 本文档将随开发迭代持续更新。每个 Phase 结束时更新完成状态。
