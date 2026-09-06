package proxy

// ===================== 分类代理测试（category_test.go） =====================
//
// 覆盖 category.go 四个 handler 的主链路：
//  1. CreateCategory：body（snake_case：parent_id/sort_order）→ gRPC 请求字段，
//     成功信封 data 回显 snake_case；
//  2. ListCategories：空请求 GET 正常转发（含 children 嵌套树序列化）；
//  3. UpdateCategory：路径 id 与 body 合并（id 取自路径）→ gRPC 请求字段；
//  4. DeleteCategory：路径参数合并 + 错误映射。
//
// fake 服务与环境见 tag_category_test.go。

import (
	"net/http"
	"strings"
	"testing"

	kerrors "github.com/go-kratos/kratos/v2/errors"
)

// TestCategoryHandler_CreateCategory_BodyToReqAndSuccessReply：Given 携带全部字段
// （snake_case 键 parent_id/sort_order）的合法 body；When POST 转发；
// Then 200 成功信封、fake 收到绑定后的全字段、data 回显 snake_case 契约。
func TestCategoryHandler_CreateCategory_BodyToReqAndSuccessReply(t *testing.T) {
	// Given
	env := newTagCategoryEnv(t)
	registerCategoryRoutes(env)
	body := `{"name":"后端","slug":"backend","description":"后端技术","parent_id":3,"sort_order":1}`
	// When
	w := doRequest(t, env.engine, http.MethodPost, "/api/v1/categories", body)
	// Then：信封 + snake_case 回显
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	resp := decodeEnvelope(t, w)
	if resp.Code != 0 || resp.Msg != "ok" {
		t.Errorf("信封 = {code:%d msg:%q}, 期望 {code:0 msg:ok}", resp.Code, resp.Msg)
	}
	raw := string(resp.Data)
	for _, want := range []string{`"category":`, `"id":"9"`, `"name":"后端"`, `"parent_id":"3"`, `"sort_order":1`, `"article_count":"2"`} {
		if !strings.Contains(raw, want) {
			t.Errorf("data 缺少 snake_case 片段 %s，data=%s", want, raw)
		}
	}
	// Then：fake server 收到请求体绑定结果（snake_case 键正确映射到 proto 字段）
	snap, ok := env.fakeCat.lastCatCreate()
	if !ok {
		t.Fatal("fake category 未收到 CreateCategory 调用")
	}
	if snap.name != "后端" || snap.slug != "backend" || snap.description != "后端技术" {
		t.Errorf("fake 收到 name=%q slug=%q description=%q, 期望 后端/backend/后端技术",
			snap.name, snap.slug, snap.description)
	}
	if snap.parentID != 3 {
		t.Errorf("fake 收到 parentID = %d, 期望 3（snake_case parent_id 绑定失败）", snap.parentID)
	}
	if snap.sortOrder != 1 {
		t.Errorf("fake 收到 sortOrder = %d, 期望 1（snake_case sort_order 绑定失败）", snap.sortOrder)
	}
}

// TestCategoryHandler_ListCategories_EmptyRequestOK：Given 无请求体的 GET；
// When ListCategories 转发空请求；Then 200 成功信封、data 含 categories 树
// （含 children 嵌套）、fake 恰好收到一次调用。
func TestCategoryHandler_ListCategories_EmptyRequestOK(t *testing.T) {
	// Given
	env := newTagCategoryEnv(t)
	registerCategoryRoutes(env)
	// When
	w := doRequest(t, env.engine, http.MethodGet, "/api/v1/categories", "")
	// Then
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	resp := decodeEnvelope(t, w)
	if resp.Code != 0 {
		t.Errorf("业务码 = %d, 期望 0", resp.Code)
	}
	raw := string(resp.Data)
	for _, want := range []string{`"categories":`, `"name":"后端"`, `"children":`, `"name":"Go"`} {
		if !strings.Contains(raw, want) {
			t.Errorf("data 缺少片段 %s，data=%s", want, raw)
		}
	}
	if got := env.fakeCat.catListCount(); got != 1 {
		t.Errorf("ListCategories 调用次数 = %d, 期望 1", got)
	}
}

// TestCategoryHandler_UpdateCategory_PathIDAndBodyMerge：Given PUT :id=5 + 变更 body
// （不含 id 字段，REST 契约约定见 handle.go）；When 转发；Then fake 收到 Id=5 与
// body 各字段合并结果、成功信封 data 回显 id=5。
func TestCategoryHandler_UpdateCategory_PathIDAndBodyMerge(t *testing.T) {
	// Given
	env := newTagCategoryEnv(t)
	registerCategoryRoutes(env)
	body := `{"name":"改名","slug":"renamed","description":"新描述","parent_id":2,"sort_order":7}`
	// When
	w := doRequest(t, env.engine, http.MethodPut, "/api/v1/categories/5", body)
	// Then：信封
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	resp := decodeEnvelope(t, w)
	if resp.Code != 0 || resp.Msg != "ok" {
		t.Errorf("信封 = {code:%d msg:%q}, 期望 {code:0 msg:ok}", resp.Code, resp.Msg)
	}
	// Then：路径 id 与 body 合并结果到达 fake
	snap, ok := env.fakeCat.lastCatUpdate()
	if !ok {
		t.Fatal("fake category 未收到 UpdateCategory 调用")
	}
	if snap.id != 5 {
		t.Errorf("fake 收到 Id = %d, 期望 5（路径参数合并失败）", snap.id)
	}
	if snap.name != "改名" || snap.slug != "renamed" || snap.description != "新描述" {
		t.Errorf("fake 收到 name=%q slug=%q description=%q, 期望 改名/renamed/新描述",
			snap.name, snap.slug, snap.description)
	}
	if snap.parentID != 2 || snap.sortOrder != 7 {
		t.Errorf("fake 收到 parentID=%d sortOrder=%d, 期望 2/7", snap.parentID, snap.sortOrder)
	}
	// Then：data 回显（id 来自路径参数；uint64 经 protojson 序列化为字符串）
	if raw := string(resp.Data); !strings.Contains(raw, `"id":"5"`) {
		t.Errorf("data 缺少回显 id=5，data=%s", raw)
	}
}

// TestCategoryHandler_DeleteCategory_PathParam：Given DELETE :id=7；
// When 转发；Then fake 收到 Id=7，空回复 data 为 {}。
func TestCategoryHandler_DeleteCategory_PathParam(t *testing.T) {
	// Given
	env := newTagCategoryEnv(t)
	registerCategoryRoutes(env)
	// When
	w := doRequest(t, env.engine, http.MethodDelete, "/api/v1/categories/7", "")
	// Then
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	resp := decodeEnvelope(t, w)
	if resp.Code != 0 {
		t.Errorf("业务码 = %d, 期望 0", resp.Code)
	}
	if raw := string(resp.Data); raw != "{}" {
		t.Errorf("data = %s, 期望 {}（空回复序列化结果）", raw)
	}
	snap, ok := env.fakeCat.lastCatDelete()
	if !ok {
		t.Fatal("fake category 未收到 DeleteCategory 调用")
	}
	if snap.id != 7 {
		t.Errorf("fake 收到 Id = %d, 期望 7（路径参数合并失败）", snap.id)
	}
}

// TestCategoryHandler_DeleteCategory_ErrorMapping：Given fake 注入 kratos 权限错误
// （PermissionDenied → gRPC PermissionDenied，映射表覆盖）；When 转发失败；
// Then 403 信封、msg 中文透出。
func TestCategoryHandler_DeleteCategory_ErrorMapping(t *testing.T) {
	// Given
	env := newTagCategoryEnv(t)
	env.fakeCat.err = kerrors.Forbidden("CATEGORY_DELETE_FORBIDDEN", "无权限删除该分类")
	registerCategoryRoutes(env)
	// When
	w := doRequest(t, env.engine, http.MethodDelete, "/api/v1/categories/7", "")
	// Then
	if w.Code != http.StatusForbidden {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusForbidden, w.Body.String())
	}
	resp := decodeEnvelope(t, w)
	if resp.Code != http.StatusForbidden || resp.Msg != "无权限删除该分类" {
		t.Errorf("信封 = {code:%d msg:%q}, 期望 {code:403 msg:无权限删除该分类}", resp.Code, resp.Msg)
	}
}
