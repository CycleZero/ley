# Entry 服务路由表（权威文档）

> 依据：`app/entry/internal/router/routes.go`（**单点真相**，本文件由该表生成）。
> 任何路由增删改：先改 routes.go + routes_test.go，再同步本文档。
> 冻结于 Wave 5 T11，分类依据 proto `google.api.http` 注解 + blog 侧 biz 鉴权盘点 + `web/src/pages/admin` 后台使用面三方核对。

## 鉴权分级

| 分类 | 中间件链 | 语义 |
|---|---|---|
| `PUBLIC` | `AuthMiddleWire(true)` | 公开：匿名可访问；携带有效 token 时注入用户身份（供下游个性化，如点赞状态）；坏 token 按匿名放行 |
| `AUTH` | `AuthMiddleWire(false)` | 登录用户：缺失/无效/吊销 token 一律 401 |
| `AUTHOR_OR_ADMIN` | `AuthMiddleWire(false)` + `RequireRole("author","admin")` | 作者或管理员；其他角色 403 |
| `ADMIN` | `AuthMiddleWire(false)` + `RequireRole("admin")` | 仅管理员；其他角色 403 |

角色取值来自 JWT claims 的 `role` 字段（user/author/admin）。

## 路由总表（39 条业务路由，公共前缀 `/api/v1`）

### PUBLIC（12）

| 方法 | 路径 | 代理 handler | 说明 |
|---|---|---|---|
| POST | `/api/v1/auth/register` | `AuthHandler.Register` | 注册 |
| POST | `/api/v1/auth/login` | `AuthHandler.Login` | 登录（返回双 token） |
| POST | `/api/v1/auth/refresh` | `AuthHandler.RefreshToken` | 刷新 token 对 |
| GET | `/api/v1/articles` | `ArticleHandler.ListArticles` | 文章列表（分页/过滤） |
| GET | `/api/v1/articles/search` | `ArticleHandler.SearchArticles` | 全文搜索（仅已发布） |
| GET | `/api/v1/articles/{identifier}` | `ArticleHandler.GetArticle` | 文章详情（ID 或 Slug；有 token 填充点赞状态） |
| GET | `/api/v1/tags` | `TagHandler.ListTags` | 标签列表 |
| GET | `/api/v1/categories` | `CategoryHandler.ListCategories` | 分类树 |
| GET | `/api/v1/site/config` | `SiteHandler.GetSiteConfig` | 站点配置（公开只读） |
| GET | `/api/v1/site/backgrounds` | `SiteHandler.ListBackgrounds` | 背景列表 |
| GET | `/api/v1/site/music/playlist` | `SiteHandler.GetMusicPlaylist` | 音乐列表 |
| GET | `/api/v1/files/presigned-upload` | `FileHandler.GetPresignedPutURL` | 预签名上传 URL（blog 侧无鉴权） |

### AUTH（17）

| 方法 | 路径 | 代理 handler | 说明 |
|---|---|---|---|
| POST | `/api/v1/auth/logout` | `AuthHandler.Logout` | 登出（吊销 token） |
| GET | `/api/v1/users/me` | `AuthHandler.GetProfile` | 当前用户资料 |
| PUT | `/api/v1/users/me` | `AuthHandler.UpdateProfile` | 更新当前用户资料 |
| POST | `/api/v1/articles` | `ArticleHandler.CreateArticle` | 创建文章（草稿） |
| POST | `/api/v1/articles/{id}/view` | `ArticleHandler.ViewArticle` | 浏览计数 |
| PUT | `/api/v1/articles/{id}` | `ArticleHandler.UpdateArticle` | 更新文章（作者/管理员，下游校验） |
| DELETE | `/api/v1/articles/{id}` | `ArticleHandler.DeleteArticle` | 删除文章 |
| POST | `/api/v1/articles/{id}/publish` | `ArticleHandler.PublishArticle` | 发布 |
| POST | `/api/v1/articles/{id}/archive` | `ArticleHandler.ArchiveArticle` | 归档 |
| POST | `/api/v1/articles/{id}/like` | `ArticleHandler.LikeArticle` | 点赞 |
| DELETE | `/api/v1/articles/{id}/like` | `ArticleHandler.UnlikeArticle` | 取消点赞 |
| POST | `/api/v1/files/upload` | `FileHandler.UploadFile` | 服务端上传 |
| GET | `/api/v1/files` | `FileHandler.ListFiles` | 文件列表 |
| GET | `/api/v1/files/{id}` | `FileHandler.GetFile` | 文件详情 |
| DELETE | `/api/v1/files/{id}` | `FileHandler.DeleteFile` | 删除文件 |
| POST | `/api/v1/files/presigned-uploads` | `FileHandler.CreatePresignedUpload` | 创建预签名上传记录 |
| POST | `/api/v1/files/presigned-uploads/complete` | `FileHandler.CompletePresignedUpload` | 完成预签名上传 |

### AUTHOR_OR_ADMIN（5）

> ⚠️ 过渡约定：tag/category 写操作 blog 侧暂缺 `requireAdmin` 校验（Wave 5 盘点缺口），
> entry 侧先按「作者或管理员」收口，待 blog 侧修复后收紧为 ADMIN。

| 方法 | 路径 | 代理 handler | 说明 |
|---|---|---|---|
| POST | `/api/v1/tags` | `TagHandler.CreateTag` | 创建标签 |
| DELETE | `/api/v1/tags/{id}` | `TagHandler.DeleteTag` | 删除标签 |
| POST | `/api/v1/categories` | `CategoryHandler.CreateCategory` | 创建分类 |
| PUT | `/api/v1/categories/{id}` | `CategoryHandler.UpdateCategory` | 更新分类 |
| DELETE | `/api/v1/categories/{id}` | `CategoryHandler.DeleteCategory` | 删除分类 |

### ADMIN（5）

| 方法 | 路径 | 代理 handler | 说明 |
|---|---|---|---|
| PUT | `/api/v1/site/config` | `SiteHandler.UpdateSiteConfig` | 更新站点配置 |
| POST | `/api/v1/site/backgrounds` | `SiteHandler.UploadBackground` | 上传背景图 |
| DELETE | `/api/v1/site/backgrounds/{id}` | `SiteHandler.DeleteBackground` | 删除背景 |
| PUT | `/api/v1/site/backgrounds/{id}/active` | `SiteHandler.SetActiveBackground` | 设为当前背景 |
| PUT | `/api/v1/site/music/playlist` | `SiteHandler.UpdateMusicPlaylist` | 更新音乐列表 |

## 非业务路由（不经鉴权链）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/healthz` | 存活探针（200 信封） |
| GET | `/readyz` | 就绪探针（本 wave 无依赖检查，恒 200；后续 wave 探测 etcd/下游 gRPC） |
| ANY | 未匹配 | 404 信封 `{code:404,msg:"接口不存在"}` |
| ANY | 方法不匹配 | 405 信封（需 `HandleMethodNotAllowed=true`，app.go 已开启） |

## 注册实现要点

- **注册入口**：`router.RegisterRouter`（app.go 经 wire 注入的 `RegisterFunc`）。流程：
  全局中间件链（RequestLogger → RequestID → CORS → [RateLimit]）→ 按分类建组注册 39 条业务路由 → healthz/readyz → NoRoute/NoMethod。
- **分组与中间件快照**：每分类一个 `root.Group("/api/v1", 鉴权链...)`，组懒创建；gin 在路由注册时把组中间件快照进该路由 handler 链。
- **通配段共存**：gin v1.12 同一段可同时存在静态与通配子路由（实测顺序无关）——
  如 `/articles/search` 与 `/articles/{identifier}`、`/files/presigned-upload` 与 `/files/{id}` 可并存，静态优先匹配。
  但**同一方法树同一位置禁止两个不同参数名**（如 GET `/articles/{identifier}` 与 GET `/articles/{id}` 会 panic），
  本表按方法分树各段仅一个参数名，天然规避。行序仍保持「同树静态段先于通配段」，属防御性写法。
- **响应信封**：统一 `{code, msg, data}`；4xx 失败码镜像 HTTP 状态；handler 层显式封装，不做中间件改写。

## 契约测试

`app/entry/internal/router/routes_test.go` 覆盖：

1. 39 条规则全部注册、无重复、无表外路由（总数 39 + 2 健康检查）；
2. 分类鉴权矩阵：匿名/坏令牌/author/user/admin 角色 × 全量路由的 401/403/放行断言；
3. 拦截响应为标准失败信封（code 镜像 HTTP 状态、消息精确）；
4. healthz/readyz 200 + NoRoute 404 信封。

路由表变更（增删 handler/路径/分类）而未同步测试 → 测试变红。
