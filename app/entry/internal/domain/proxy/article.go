package proxy

// ===================== 文章域代理 handler（article.go） =====================
//
// 覆盖博客文章服务 11 个 REST 端点（Create/Update/Delete/Get/List/Search/
// Publish/Archive/Like/Unlike/View），路由由后续 wave 在 router 层按本文件方法
// 挂载，方法签名统一为 gin.HandlerFunc。
//
// 每个 handler 只做三件事：分配具体请求消息 → 路径参数合并（BindPath* 辅助，
// 显式逐字段）→ callProto 骨架转发（请求体绑定/元数据透传/信封封装都在骨架内
// 完成，见 handle.go 顶部设计约定）。路径参数合并须在 callProto 之前完成——
// body 若携带与路径同名字段会覆盖（REST 契约下资源标识只应经路径传递，
// 请求体不得携带路径同名字段）。
//
// GET 端点（Get/List/Search）无请求体：callProto 的 bindRequestBody 对 GET
// 直接跳过；query 参数在本文件显式逐字段绑定（bindQueryUint64/bindQueryInt32/
// queryTags，与 BindPath* 同一哲学：不引入反射、编译期可查）。这些辅助先放
// 本文件就近使用——若后续 tag/category/file 域出现第二个使用者再提升至
// handle.go 共享（YAGNI）。

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	blogv1 "github.com/CycleZero/ley/api/blog/v1"
	"github.com/CycleZero/ley/app/entry/internal/common"
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"
)

// ArticleHandler 文章域代理 handler：hub 持有下游 ArticleService gRPC client，
// 方法各自对应权威路由表中的一个 REST 端点（路径见方法注释 @Router）。
type ArticleHandler struct {
	hub *ServiceHub
}

// NewArticleHandler 构造文章代理 handler（hub 由 wire 注入 ServiceHub）。
func NewArticleHandler(hub *ServiceHub) *ArticleHandler {
	return &ArticleHandler{hub: hub}
}

// CreateArticle 创建文章：POST /api/v1/articles，body 即 CreateArticleRequest。
//
// @Summary 创建文章
// @Description 创建一篇新文章（草稿状态），可选择关联标签和分类
// @Tags 文章
// @Accept json
// @Produce json
// @Param request body blogv1.CreateArticleRequest true "创建文章请求体"
// @Success 200 {object} common.Response{data=blogv1.CreateArticleReply}
// @Router /api/v1/articles [post]
func (h *ArticleHandler) CreateArticle(c *gin.Context) {
	req := &blogv1.CreateArticleRequest{}
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.Article.CreateArticle(ctx, m.(*blogv1.CreateArticleRequest))
	})
}

// UpdateArticle 更新文章：PUT /api/v1/articles/{id}，路径 id 合并进请求体字段。
//
// @Summary 更新文章
// @Description 部分更新文章字段，nil 字段保持不变，可全量替换标签
// @Tags 文章
// @Accept json
// @Produce json
// @Param id path integer true "文章 ID"
// @Param request body blogv1.UpdateArticleRequest true "更新文章请求体"
// @Success 200 {object} common.Response{data=blogv1.UpdateArticleReply}
// @Router /api/v1/articles/{id} [put]
func (h *ArticleHandler) UpdateArticle(c *gin.Context) {
	req := &blogv1.UpdateArticleRequest{}
	// 路径参数合并（显式逐字段）：路由 :id → proto Id 字段
	if !BindPathUint64(c, "id", &req.Id) {
		return
	}
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.Article.UpdateArticle(ctx, m.(*blogv1.UpdateArticleRequest))
	})
}

// DeleteArticle 删除文章：DELETE /api/v1/articles/{id}（软删除，下游处理计数）。
//
// @Summary 删除文章
// @Description 软删除文章，已发布文章会同步调整分类和标签的计数
// @Tags 文章
// @Param id path integer true "文章 ID"
// @Success 200 {object} common.Response{data=blogv1.DeleteArticleReply}
// @Router /api/v1/articles/{id} [delete]
func (h *ArticleHandler) DeleteArticle(c *gin.Context) {
	req := &blogv1.DeleteArticleRequest{}
	if !BindPathUint64(c, "id", &req.Id) {
		return
	}
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.Article.DeleteArticle(ctx, m.(*blogv1.DeleteArticleRequest))
	})
}

// GetArticle 获取文章：GET /api/v1/articles/{identifier}。
//
// identifier 为字符串（ID 或 Slug），经 BindPathString 原样透传，不做数字猜测
// ——ID 形态由下游解析，Slug 形态（含连字符）不会被误伤。
//
// @Summary 获取文章
// @Description 按 ID 或 Slug 获取文章详情，认证用户会填充点赞状态
// @Tags 文章
// @Param identifier path string true "文章 ID 或 Slug"
// @Success 200 {object} common.Response{data=blogv1.GetArticleReply}
// @Router /api/v1/articles/{identifier} [get]
func (h *ArticleHandler) GetArticle(c *gin.Context) {
	req := &blogv1.GetArticleRequest{}
	BindPathString(c, "identifier", &req.Identifier)
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.Article.GetArticle(ctx, m.(*blogv1.GetArticleRequest))
	})
}

// ListArticles 文章列表：GET /api/v1/articles，过滤/分页全部经 query 绑定。
//
// @Summary 文章列表
// @Description 分页查询文章列表，支持按状态、分类、标签、作者过滤，支持排序
// @Tags 文章
// @Param status query string false "状态过滤（published/archived/draft，缺省不过滤）"
// @Param category_id query integer false "分类 ID"
// @Param tags query []string false "标签过滤（支持 ?tags=a&tags=b 或逗号分隔 ?tags=a,b）"
// @Param author_id query integer false "作者 ID"
// @Param sort_by query string false "排序字段（缺省由下游决定）"
// @Param sort_order query string false "排序方向（asc/desc）"
// @Param page query integer false "页码（从 1 起）"
// @Param page_size query integer false "每页数量"
// @Success 200 {object} common.Response{data=blogv1.ListArticlesReply}
// @Router /api/v1/articles [get]
func (h *ArticleHandler) ListArticles(c *gin.Context) {
	req := &blogv1.ListArticlesRequest{}
	// GET 无请求体：query 显式逐字段绑定（解析失败时内部已写 400 信封并短路）
	if !bindListQuery(c, req) {
		return
	}
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.Article.ListArticles(ctx, m.(*blogv1.ListArticlesRequest))
	})
}

// SearchArticles 全文搜索：GET /api/v1/articles/search。
//
// @Summary 全文搜索
// @Description 关键词搜索文章标题/摘要/正文，仅返回已发布文章
// @Tags 文章
// @Param keyword query string true "搜索关键词"
// @Param page query integer false "页码（从 1 起）"
// @Param page_size query integer false "每页数量"
// @Success 200 {object} common.Response{data=blogv1.SearchArticlesReply}
// @Router /api/v1/articles/search [get]
func (h *ArticleHandler) SearchArticles(c *gin.Context) {
	req := &blogv1.SearchArticlesRequest{}
	if !bindSearchQuery(c, req) {
		return
	}
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.Article.SearchArticles(ctx, m.(*blogv1.SearchArticlesRequest))
	})
}

// PublishArticle 发布文章：POST /api/v1/articles/{id}/publish。
//
// @Summary 发布文章
// @Description 将草稿发布为公开状态，记录发布时间，更新分类和标签计数
// @Tags 文章
// @Param id path integer true "文章 ID"
// @Success 200 {object} common.Response{data=blogv1.PublishArticleReply}
// @Router /api/v1/articles/{id}/publish [post]
func (h *ArticleHandler) PublishArticle(c *gin.Context) {
	req := &blogv1.PublishArticleRequest{}
	if !BindPathUint64(c, "id", &req.Id) {
		return
	}
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.Article.PublishArticle(ctx, m.(*blogv1.PublishArticleRequest))
	})
}

// ArchiveArticle 归档文章：POST /api/v1/articles/{id}/archive。
//
// @Summary 归档文章
// @Description 将文章归档（不展示在默认列表），减少分类和标签计数
// @Tags 文章
// @Param id path integer true "文章 ID"
// @Success 200 {object} common.Response{data=blogv1.ArchiveArticleReply}
// @Router /api/v1/articles/{id}/archive [post]
func (h *ArticleHandler) ArchiveArticle(c *gin.Context) {
	req := &blogv1.ArchiveArticleRequest{}
	if !BindPathUint64(c, "id", &req.Id) {
		return
	}
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.Article.ArchiveArticle(ctx, m.(*blogv1.ArchiveArticleRequest))
	})
}

// LikeArticle 点赞文章：POST /api/v1/articles/{id}/like（幂等）。
//
// @Summary 点赞文章
// @Description 为文章点赞（幂等，重复点赞不报错）
// @Tags 文章
// @Param id path integer true "文章 ID"
// @Success 200 {object} common.Response{data=blogv1.LikeArticleReply}
// @Router /api/v1/articles/{id}/like [post]
func (h *ArticleHandler) LikeArticle(c *gin.Context) {
	req := &blogv1.LikeArticleRequest{}
	if !BindPathUint64(c, "id", &req.Id) {
		return
	}
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.Article.LikeArticle(ctx, m.(*blogv1.LikeArticleRequest))
	})
}

// UnlikeArticle 取消点赞：DELETE /api/v1/articles/{id}/like（幂等）。
//
// @Summary 取消点赞
// @Description 取消对文章的点赞（幂等）
// @Tags 文章
// @Param id path integer true "文章 ID"
// @Success 200 {object} common.Response{data=blogv1.UnlikeArticleReply}
// @Router /api/v1/articles/{id}/like [delete]
func (h *ArticleHandler) UnlikeArticle(c *gin.Context) {
	req := &blogv1.UnlikeArticleRequest{}
	if !BindPathUint64(c, "id", &req.Id) {
		return
	}
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.Article.UnlikeArticle(ctx, m.(*blogv1.UnlikeArticleRequest))
	})
}

// ViewArticle 浏览文章：POST /api/v1/articles/{id}/view（记浏览量）。
//
// @Summary 浏览文章
// @Description 记录文章浏览量。同一 IP 1 小时内重复浏览同一篇文章会被去重
// @Tags 文章
// @Param id path integer true "文章 ID"
// @Success 200 {object} common.Response{data=blogv1.ViewArticleReply}
// @Router /api/v1/articles/{id}/view [post]
func (h *ArticleHandler) ViewArticle(c *gin.Context) {
	req := &blogv1.ViewArticleRequest{}
	if !BindPathUint64(c, "id", &req.Id) {
		return
	}
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.Article.ViewArticle(ctx, m.(*blogv1.ViewArticleRequest))
	})
}

// ===================== query 参数绑定辅助（本文件本地） =====================
//
// handle.go 的 BindPath* 只服务路径参数；callProto 的 bindRequestBody 对 GET 直接
// 跳过（GET 无 body 语义）。这里按 BindPath* 同款约定补充 query 侧绑定：参数缺失
// 时跳过（零值即"不过滤/默认值"），解析失败时写 400 信封并返回 false——调用方
// 应立即 return，禁止携带零值调用下游。

// bindQueryUint64 读取 query 参数 name 并按 uint64 解析写入 dst（语义同
// BindPathUint64；失败消息措辞区分「查询参数」）。参数缺失或为空串时跳过。
func bindQueryUint64(c *gin.Context, name string, dst *uint64) bool {
	v := c.Query(name)
	if v == "" {
		return true
	}
	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		common.Fail(c, http.StatusBadRequest, http.StatusBadRequest, "查询参数 "+name+" 必须为无符号整数")
		return false
	}
	*dst = n
	return true
}

// bindQueryInt32 读取 query 参数 name 并按 int32 解析写入 dst（语义同
// BindPathInt32，服务于 page/page_size 等 int32 分页字段）。
func bindQueryInt32(c *gin.Context, name string, dst *int32) bool {
	v := c.Query(name)
	if v == "" {
		return true
	}
	n, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		common.Fail(c, http.StatusBadRequest, http.StatusBadRequest, "查询参数 "+name+" 必须为整数")
		return false
	}
	*dst = int32(n)
	return true
}

// queryTags 读取 tags 查询参数为字符串切片（repeated string 语义）。
//
// 兼容两种序列化形态，可混用：
//  1. 重复键：?tags=go&tags=gin——URLSearchParams/ofetch（web 前端 query 序列化）
//     的原生输出；
//  2. 逗号分隔：?tags=go,gin——kratos HTTP client binding.EncodeURL 对 repeated
//     标量参数的默认编码（旧网关对接客户端的形态）。
//
// 空白项剔除；tag 名不含逗号（下游命名约束），逗号仅作分隔符。
func queryTags(c *gin.Context) []string {
	vals, _ := c.GetQueryArray("tags")
	var tags []string
	for _, v := range vals {
		for _, part := range strings.Split(v, ",") {
			if part = strings.TrimSpace(part); part != "" {
				tags = append(tags, part)
			}
		}
	}
	return tags
}

// bindListQuery ListArticles 的 GET query → 请求字段（显式逐字段绑定）。
//
// 字符串字段（status/sort_by/sort_order）缺失时保持零值即可表达"不过滤/默认
// 排序"，无需判断存在性；数值字段经 bindQuery* 校验。
func bindListQuery(c *gin.Context, req *blogv1.ListArticlesRequest) bool {
	req.Status = c.Query("status")
	req.SortBy = c.Query("sort_by")
	req.SortOrder = c.Query("sort_order")
	req.Tags = queryTags(c)
	return bindQueryUint64(c, "category_id", &req.CategoryId) &&
		bindQueryUint64(c, "author_id", &req.AuthorId) &&
		bindQueryInt32(c, "page", &req.Page) &&
		bindQueryInt32(c, "page_size", &req.PageSize)
}

// bindSearchQuery SearchArticles 的 GET query → 请求字段。
func bindSearchQuery(c *gin.Context, req *blogv1.SearchArticlesRequest) bool {
	req.Keyword = c.Query("keyword")
	return bindQueryInt32(c, "page", &req.Page) &&
		bindQueryInt32(c, "page_size", &req.PageSize)
}
