package proxy

// ===================== 分类代理 handler（category.go） =====================
//
// 承接 /api/v1/categories 的 4 个 REST 接口（Create/List/Update/Delete），全部经
// callProto 骨架转发到 blog 微服务 CategoryService（设计约定见 handle.go 顶部
// 注释）。分类为树形结构（parent_id/children），树组装与循环引用校验等业务规则
// 全部留在下游 blog 服务，本文件只做「绑定 → 转发 → 信封」。
//
// 路由表（权威，路由注册见 router wave）：
//
//	POST   /api/v1/categories      → CreateCategory
//	GET    /api/v1/categories      → ListCategories
//	PUT    /api/v1/categories/:id  → UpdateCategory
//	DELETE /api/v1/categories/:id  → DeleteCategory

import (
	"context"
	"net/http"

	blogv1 "github.com/CycleZero/ley/api/blog/v1"
	"github.com/CycleZero/ley/app/entry/internal/common"
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"
)

// CategoryHandler 分类代理 handler：持有 ServiceHub，逐方法转发 blog CategoryService。
type CategoryHandler struct {
	hub *ServiceHub
}

// NewCategoryHandler 构造分类代理 handler。
func NewCategoryHandler(hub *ServiceHub) *CategoryHandler {
	return &CategoryHandler{hub: hub}
}

// CreateCategory 创建分类。
//
//	@Summary 创建分类
//	@Description 创建博客分类（parent_id 为空表示根分类；名称/层级规则下游校验）
//	@Tags 分类
//	@Accept json
//	@Produce json
//	@Param body body blogv1.CreateCategoryRequest true "分类信息"
//	@Success 200 {object} common.Response{data=blogv1.CreateCategoryReply}
//	@Failure 400 {object} common.Response
//	@Failure 409 {object} common.Response
//	@Router /api/v1/categories [post]
func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	req := &blogv1.CreateCategoryRequest{}
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.Category.CreateCategory(ctx, m.(*blogv1.CreateCategoryRequest))
	})
}

// ListCategories 分类列表。
//
//	@Summary 分类列表
//	@Description 返回树形分类（含文章计数与 children 子树；公开接口）
//	@Tags 分类
//	@Produce json
//	@Success 200 {object} common.Response{data=blogv1.ListCategoriesReply}
//	@Failure 500 {object} common.Response
//	@Router /api/v1/categories [get]
func (h *CategoryHandler) ListCategories(c *gin.Context) {
	req := &blogv1.ListCategoriesRequest{}
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.Category.ListCategories(ctx, m.(*blogv1.ListCategoriesRequest))
	})
}

// UpdateCategory 更新分类。
//
//	@Summary 更新分类
//	@Description 按 ID 更新分类信息（REST 契约下资源标识只经路径传递，
//	请求体不得携带与路径同名的 id 字段，见 handle.go 顶部约定）
//	@Tags 分类
//	@Accept json
//	@Produce json
//	@Param id path uint64 true "分类 ID"
//	@Param body body blogv1.UpdateCategoryRequest true "分类信息"
//	@Success 200 {object} common.Response{data=blogv1.UpdateCategoryReply}
//	@Failure 400 {object} common.Response
//	@Failure 404 {object} common.Response
//	@Router /api/v1/categories/{id} [put]
func (h *CategoryHandler) UpdateCategory(c *gin.Context) {
	req := &blogv1.UpdateCategoryRequest{}
	// 绑定顺序与 DELETE 类不同（必须「先 body 后路径」）：protojson.Unmarshal 会先
	// Reset 目标消息（protobuf encoding/protojson/decode.go），若先合并路径再让
	// callProto 绑 body，Reset 会把路径 Id 一并清掉，PUT 转发将携带零值 id。
	// 故此处先用 bindRequestBody（callProto 内部同款解析：DiscardUnknown 容忍 +
	// 空 body 跳过）解析 body，再按路径覆盖 id——路径参数恒胜出，符合 REST 契约
	// 「资源标识只经路径传递」；随后 callProto 因 body 已被读空而跳过二次绑定。
	if err := bindRequestBody(c, req); err != nil {
		common.Fail(c, http.StatusBadRequest, http.StatusBadRequest, "参数解析失败")
		return
	}
	// 路径参数合并（显式逐字段）：路由 :id → proto Id 字段
	if !BindPathUint64(c, "id", &req.Id) {
		return
	}
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.Category.UpdateCategory(ctx, m.(*blogv1.UpdateCategoryRequest))
	})
}

// DeleteCategory 删除分类。
//
//	@Summary 删除分类
//	@Description 按 ID 删除分类（存在子分类或文章时下游拒绝删除）
//	@Tags 分类
//	@Produce json
//	@Param id path uint64 true "分类 ID"
//	@Success 200 {object} common.Response{data=blogv1.DeleteCategoryReply}
//	@Failure 400 {object} common.Response
//	@Failure 404 {object} common.Response
//	@Router /api/v1/categories/{id} [delete]
func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	req := &blogv1.DeleteCategoryRequest{}
	if !BindPathUint64(c, "id", &req.Id) {
		return
	}
	callProto(c, req, func(ctx context.Context, m proto.Message) (proto.Message, error) {
		return h.hub.Category.DeleteCategory(ctx, m.(*blogv1.DeleteCategoryRequest))
	})
}
