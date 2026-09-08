// routes.go —— 权威路由表（单点真相）。
//
// 本文件是 entry 全部业务路由的**唯一事实来源**：路由分类常量 + 39 条路由规则
// （方法/路径/分类/绑定的代理 handler 方法）。router.go 的 RegisterRouter 依据
// 本表完成注册，routes_test.go 的契约测试以本表为期望核对引擎注册表——
// 任何 handler 方法新增/删除、路径改动、分类调整都会让注册与测试同步失效（红）。
//
// 路径约定：表中 Path 为完整路径（含 /api/v1 前缀，与 swag @Router 注释及
// gateway 时代对外契约一致）；注册时按 apiV1BasePath 剥离前缀挂到分组。
//
// 分类与鉴权链（中间件声明顺序即执行顺序）：
//
//	RouteClassPublic        → AuthMiddleWire(true)                      可选认证：有 token 注入身份，无 token 放行
//	RouteClassAuth          → AuthMiddleWire(false)                     强制认证：无效/缺失 token 一律 401
//	RouteClassAuthorOrAdmin → AuthMiddleWire(false)+RequireRole("author","admin")
//	RouteClassAdmin         → AuthMiddleWire(false)+RequireRole("admin")
//
// 分类冻结依据：api/*/v1 proto google.api.http 注解 + blog 侧 biz 鉴权盘点 +
// web/src/pages/admin 后台使用面三方核对（Wave 5 T11 决策，勿单独调整——
// 如需变更请同步修订 docs/entry-routing.md 与 routes_test.go 期望）。
package router

import (
	"net/http"

	"github.com/CycleZero/ley/app/entry/internal/domain/proxy"

	"github.com/gin-gonic/gin"
)

// apiV1BasePath 全部业务路由的公共前缀（与 proto 注解 /api/v1 一致）。
const apiV1BasePath = "/api/v1"

// RouteClass 业务路由分类：决定该路由挂载的鉴权中间件链。
type RouteClass int

const (
	// RouteClassPublic 公开路由：可选认证（AuthMiddleWire(true)）。
	// 匿名可访问；携带有效 token 时注入用户身份，供下游按需个性化（如点赞状态）。
	RouteClassPublic RouteClass = iota
	// RouteClassAuth 登录用户路由：强制认证（AuthMiddleWire(false)）。
	RouteClassAuth
	// RouteClassAuthorOrAdmin 作者/管理员路由：强制认证 + RequireRole("author","admin")。
	// 分类能力保留；当前 39 条路由无使用方——tag/category 写操作已随 blog 侧
	// requireAdmin 落地（FIX-4）收紧为 ADMIN。
	RouteClassAuthorOrAdmin
	// RouteClassAdmin 管理员路由：强制认证 + RequireRole("admin")。
	RouteClassAdmin
)

// String RouteClass 的中文可读名（路由表文档/测试断言用）。
func (c RouteClass) String() string {
	switch c {
	case RouteClassPublic:
		return "PUBLIC"
	case RouteClassAuth:
		return "AUTH"
	case RouteClassAuthorOrAdmin:
		return "AUTHOR_OR_ADMIN"
	case RouteClassAdmin:
		return "ADMIN"
	default:
		return "UNKNOWN"
	}
}

// RouteRule 权威路由表条目：方法 + 完整路径 + 分类 + 绑定的代理 handler 方法。
type RouteRule struct {
	Method  string
	Path    string
	Class   RouteClass
	Handler gin.HandlerFunc
}

// newRouteRules 构建权威路由表（39 条业务路由，顺序即注册顺序）。
//
// 字段顺序：Method, Path, Class, Handler——与 docs/entry-routing.md 表格一致。
// 行序约定（同方法树内静态段先于通配段注册，如 articles/search 先于
// articles/{identifier}、files/presigned-upload 先于 files/{id}）；
// gin v1.12 静态段与通配段可同层共存（实测顺序无关），但保持静态在前
// 仍是防御性写法，避免未来 gin 行为变更造成注册 panic。
func newRouteRules(hub *proxy.ServiceHub) []RouteRule {
	authH := proxy.NewAuthHandler(hub)
	articleH := proxy.NewArticleHandler(hub)
	tagH := proxy.NewTagHandler(hub)
	categoryH := proxy.NewCategoryHandler(hub)
	fileH := proxy.NewFileHandler(hub)
	siteH := proxy.NewSiteHandler(hub)

	return []RouteRule{
		// ── PUBLIC：公开（可选认证）──
		{http.MethodPost, apiV1BasePath + "/auth/register", RouteClassPublic, authH.Register},
		{http.MethodPost, apiV1BasePath + "/auth/login", RouteClassPublic, authH.Login},
		{http.MethodPost, apiV1BasePath + "/auth/refresh", RouteClassPublic, authH.RefreshToken},
		{http.MethodGet, apiV1BasePath + "/articles", RouteClassPublic, articleH.ListArticles},
		{http.MethodGet, apiV1BasePath + "/articles/search", RouteClassPublic, articleH.SearchArticles},
		{http.MethodGet, apiV1BasePath + "/articles/{identifier}", RouteClassPublic, articleH.GetArticle},
		{http.MethodGet, apiV1BasePath + "/tags", RouteClassPublic, tagH.ListTags},
		{http.MethodGet, apiV1BasePath + "/categories", RouteClassPublic, categoryH.ListCategories},
		{http.MethodGet, apiV1BasePath + "/site/config", RouteClassPublic, siteH.GetSiteConfig},
		{http.MethodGet, apiV1BasePath + "/site/backgrounds", RouteClassPublic, siteH.ListBackgrounds},
		{http.MethodGet, apiV1BasePath + "/site/music/playlist", RouteClassPublic, siteH.GetMusicPlaylist},
		{http.MethodPost, apiV1BasePath + "/auth/logout", RouteClassPublic, authH.Logout},

		// ── AUTH：登录用户（强制认证）──
		{http.MethodGet, apiV1BasePath + "/users/me", RouteClassAuth, authH.GetProfile},
		{http.MethodPut, apiV1BasePath + "/users/me", RouteClassAuth, authH.UpdateProfile},
		{http.MethodPost, apiV1BasePath + "/articles", RouteClassAuth, articleH.CreateArticle},
		{http.MethodPost, apiV1BasePath + "/articles/{id}/view", RouteClassAuth, articleH.ViewArticle},
		{http.MethodPut, apiV1BasePath + "/articles/{id}", RouteClassAuth, articleH.UpdateArticle},
		{http.MethodDelete, apiV1BasePath + "/articles/{id}", RouteClassAuth, articleH.DeleteArticle},
		{http.MethodPost, apiV1BasePath + "/articles/{id}/publish", RouteClassAuth, articleH.PublishArticle},
		{http.MethodPost, apiV1BasePath + "/articles/{id}/archive", RouteClassAuth, articleH.ArchiveArticle},
		{http.MethodPost, apiV1BasePath + "/articles/{id}/like", RouteClassAuth, articleH.LikeArticle},
		{http.MethodDelete, apiV1BasePath + "/articles/{id}/like", RouteClassAuth, articleH.UnlikeArticle},
		{http.MethodPost, apiV1BasePath + "/files/upload", RouteClassAuth, fileH.UploadFile},
		{http.MethodGet, apiV1BasePath + "/files", RouteClassAuth, fileH.ListFiles},
		{http.MethodGet, apiV1BasePath + "/files/presigned-upload", RouteClassAuth, fileH.GetPresignedPutURL},
		{http.MethodGet, apiV1BasePath + "/files/{id}", RouteClassAuth, fileH.GetFile},
		{http.MethodDelete, apiV1BasePath + "/files/{id}", RouteClassAuth, fileH.DeleteFile},
		{http.MethodPost, apiV1BasePath + "/files/presigned-uploads", RouteClassAuth, fileH.CreatePresignedUpload},
		{http.MethodPost, apiV1BasePath + "/files/presigned-uploads/complete", RouteClassAuth, fileH.CompletePresignedUpload},

		// ── ADMIN：管理员（tag/category 写操作 + 站点配置/背景/音乐列表）──
		{http.MethodPost, apiV1BasePath + "/tags", RouteClassAdmin, tagH.CreateTag},
		{http.MethodDelete, apiV1BasePath + "/tags/{id}", RouteClassAdmin, tagH.DeleteTag},
		{http.MethodPost, apiV1BasePath + "/categories", RouteClassAdmin, categoryH.CreateCategory},
		{http.MethodPut, apiV1BasePath + "/categories/{id}", RouteClassAdmin, categoryH.UpdateCategory},
		{http.MethodDelete, apiV1BasePath + "/categories/{id}", RouteClassAdmin, categoryH.DeleteCategory},
		{http.MethodPut, apiV1BasePath + "/site/config", RouteClassAdmin, siteH.UpdateSiteConfig},
		{http.MethodPost, apiV1BasePath + "/site/backgrounds", RouteClassAdmin, siteH.UploadBackground},
		{http.MethodDelete, apiV1BasePath + "/site/backgrounds/{id}", RouteClassAdmin, siteH.DeleteBackground},
		{http.MethodPut, apiV1BasePath + "/site/backgrounds/{id}/active", RouteClassAdmin, siteH.SetActiveBackground},
		{http.MethodPut, apiV1BasePath + "/site/music/playlist", RouteClassAdmin, siteH.UpdateMusicPlaylist},
	}
}
