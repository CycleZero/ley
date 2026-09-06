package proxy

// ===================== 标签代理测试（tag_test.go） =====================
//
// 覆盖 tag.go 三个 handler 的主链路：
//  1. CreateTag：body → gRPC 请求字段，成功信封 data 为 snake_case 原始 JSON；
//  2. ListTags：空请求 GET 正常转发；
//  3. DeleteTag：路径参数合并（含非法值 400 短路）+ 元数据透传 + 错误映射。
//
// fake 服务与环境见 tag_category_test.go。

import (
	"net/http"
	"strings"
	"testing"

	pkgmeta "github.com/CycleZero/ley/pkg/meta"
	kerrors "github.com/go-kratos/kratos/v2/errors"
)

// TestTagHandler_CreateTag_BodyToReqAndSuccessReply：Given 合法 body；
// When POST /api/v1/tags 经 CreateTag 转发；Then 200 成功信封、data 为
// snake_case 的 tag 对象（tag 字段名而非 Tag），fake server 收到 name。
func TestTagHandler_CreateTag_BodyToReqAndSuccessReply(t *testing.T) {
	// Given
	env := newTagCategoryEnv(t)
	registerTagRoutes(env)
	// When
	w := doRequest(t, env.engine, http.MethodPost, "/api/v1/tags", `{"name":"golang"}`)
	// Then：信封 + snake_case 契约
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	resp := decodeEnvelope(t, w)
	if resp.Code != 0 || resp.Msg != "ok" {
		t.Errorf("信封 = {code:%d msg:%q}, 期望 {code:0 msg:ok}", resp.Code, resp.Msg)
	}
	raw := string(resp.Data)
	for _, want := range []string{`"tag":`, `"id":"11"`, `"name":"golang"`, `"slug":"go-lang"`, `"article_count":"5"`} {
		if !strings.Contains(raw, want) {
			t.Errorf("data 缺少 snake_case 片段 %s，data=%s", want, raw)
		}
	}
	if strings.Contains(raw, `"Tag"`) {
		t.Errorf("data 出现 Go 结构体字段名 Tag（期望 proto 字段名 tag），data=%s", raw)
	}
	// Then：fake server 收到请求体绑定结果
	snap, ok := env.fakeTag.lastTagCreate()
	if !ok {
		t.Fatal("fake tag 未收到 CreateTag 调用")
	}
	if snap.name != "golang" {
		t.Errorf("fake 收到 name=%q, 期望 golang", snap.name)
	}
}

// TestTagHandler_CreateTag_BadJSON：Given 非法 JSON body；When 转发；
// Then 400 信封「参数解析失败」且不触发下游调用。
func TestTagHandler_CreateTag_BadJSON(t *testing.T) {
	// Given
	env := newTagCategoryEnv(t)
	registerTagRoutes(env)
	// When
	w := doRequest(t, env.engine, http.MethodPost, "/api/v1/tags", `{"name":`)
	// Then
	if w.Code != http.StatusBadRequest {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
	resp := decodeEnvelope(t, w)
	if resp.Code != http.StatusBadRequest || resp.Msg != "参数解析失败" {
		t.Errorf("信封 = {code:%d msg:%q}, 期望 {code:400 msg:参数解析失败}", resp.Code, resp.Msg)
	}
	if _, ok := env.fakeTag.lastTagCreate(); ok {
		t.Error("非法 JSON 不应触发下游 CreateTag 调用")
	}
}

// TestTagHandler_ListTags_EmptyRequestOK：Given 无请求体的 GET；
// When ListTags 转发空请求；Then 200 成功信封、data 含 tags 数组、
// fake server 恰好收到一次调用。
func TestTagHandler_ListTags_EmptyRequestOK(t *testing.T) {
	// Given
	env := newTagCategoryEnv(t)
	registerTagRoutes(env)
	// When
	w := doRequest(t, env.engine, http.MethodGet, "/api/v1/tags", "")
	// Then
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	resp := decodeEnvelope(t, w)
	if resp.Code != 0 {
		t.Errorf("业务码 = %d, 期望 0", resp.Code)
	}
	raw := string(resp.Data)
	for _, want := range []string{`"tags":`, `"name":"go"`, `"name":"gin"`, `"article_count":"3"`} {
		if !strings.Contains(raw, want) {
			t.Errorf("data 缺少片段 %s，data=%s", want, raw)
		}
	}
	if got := env.fakeTag.tagListCount(); got != 1 {
		t.Errorf("ListTags 调用次数 = %d, 期望 1", got)
	}
}

// TestTagHandler_DeleteTag_PathParamAndMeta：Given 认证请求 DELETE :id=99；
// When BindPathUint64 合并后转发；Then fake 收到 Id=99，认证元数据（uid/name/role）
// 随 x-md-global-* 透传到达 server 端，删除类空回复 data 为 {}。
func TestTagHandler_DeleteTag_PathParamAndMeta(t *testing.T) {
	// Given
	env := newTagCategoryEnv(t)
	registerTagRoutes(env)
	// When
	w := doRequest(t, env.engine, http.MethodDelete, "/api/v1/tags/99", "")
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
	snap, ok := env.fakeTag.lastTagDelete()
	if !ok {
		t.Fatal("fake tag 未收到 DeleteTag 调用")
	}
	if snap.id != 99 {
		t.Errorf("fake 收到 Id = %d, 期望 99（路径参数合并失败）", snap.id)
	}
	if got := mdVal(t, snap.md, pkgmeta.AuthUserIDKey); got != "1" {
		t.Errorf("x-md-global-auth-user-id = %q, 期望 %q", got, "1")
	}
	if got := mdVal(t, snap.md, pkgmeta.AuthUserNameKey); got != "root" {
		t.Errorf("x-md-global-auth-user-name = %q, 期望 %q", got, "root")
	}
	if got := mdVal(t, snap.md, pkgmeta.AuthUserRoleKey); got != "admin" {
		t.Errorf("x-md-global-auth-user-role = %q, 期望 %q", got, "admin")
	}
}

// TestTagHandler_DeleteTag_BadPathParam：Given :id 非法（非整数）；
// When BindPathUint64 解析失败；Then 400 信封并短路——不触发下游调用。
func TestTagHandler_DeleteTag_BadPathParam(t *testing.T) {
	// Given
	env := newTagCategoryEnv(t)
	registerTagRoutes(env)
	// When
	w := doRequest(t, env.engine, http.MethodDelete, "/api/v1/tags/abc", "")
	// Then
	if w.Code != http.StatusBadRequest {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
	resp := decodeEnvelope(t, w)
	if resp.Code != http.StatusBadRequest || !strings.Contains(resp.Msg, "必须为无符号整数") {
		t.Errorf("信封 = {code:%d msg:%q}, 期望 {code:400 msg:含'必须为无符号整数'}", resp.Code, resp.Msg)
	}
	if _, ok := env.fakeTag.lastTagDelete(); ok {
		t.Error("非法路径参数不应触发下游 DeleteTag 调用")
	}
}

// TestTagHandler_DeleteTag_ErrorMapping：Given fake 注入 kratos NotFound 业务错误；
// When 转发失败；Then 404 信封、msg 取 kratos 业务消息（中文透出）。
func TestTagHandler_DeleteTag_ErrorMapping(t *testing.T) {
	// Given
	env := newTagCategoryEnv(t)
	env.fakeTag.err = kerrors.NotFound("TAG_NOT_FOUND", "标签不存在")
	registerTagRoutes(env)
	// When
	w := doRequest(t, env.engine, http.MethodDelete, "/api/v1/tags/404", "")
	// Then
	if w.Code != http.StatusNotFound {
		t.Fatalf("HTTP 状态 = %d, 期望 %d，body=%s", w.Code, http.StatusNotFound, w.Body.String())
	}
	resp := decodeEnvelope(t, w)
	if resp.Code != http.StatusNotFound || resp.Msg != "标签不存在" {
		t.Errorf("信封 = {code:%d msg:%q}, 期望 {code:404 msg:标签不存在}", resp.Code, resp.Msg)
	}
}
