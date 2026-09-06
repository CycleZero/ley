package common

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// ginTestCtx 构造隔离的 gin 测试上下文（纯逻辑，不依赖任何外部服务）。
func ginTestCtx() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode) // 关闭 gin 调试输出，保证测试输出干净
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

// decodeEnvelope 解析响应体为统一信封并返回。
func decodeEnvelope(t *testing.T, w *httptest.ResponseRecorder) Response {
	t.Helper()
	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应体失败: %v，body=%s", err, w.Body.String())
	}
	return resp
}

// TestOK_成功信封：Given 任意业务数据；When 调用 OK；
// Then 断言 HTTP 200、业务码 CodeOK(0)、data 原样返回。
func TestOK_SuccessEnvelope(t *testing.T) {
	// Given
	c, w := ginTestCtx()
	// When
	OK(c, map[string]any{"id": 1})
	// Then
	if w.Code != http.StatusOK {
		t.Errorf("HTTP 状态 = %d, 期望 %d", w.Code, http.StatusOK)
	}
	resp := decodeEnvelope(t, w)
	if resp.Code != CodeOK {
		t.Errorf("业务码 = %d, 期望 CodeOK(%d)", resp.Code, CodeOK)
	}
	if resp.Msg != "ok" {
		t.Errorf("msg = %q, 期望 %q", resp.Msg, "ok")
	}
	data, ok := resp.Data.(map[string]any)
	if !ok || data["id"] != float64(1) {
		t.Errorf("data = %#v, 期望原样返回 {id:1}", resp.Data)
	}
}

// TestFail_FailureEnvelope：Given HTTP 400 + 业务码 400 + 中文消息；
// When 调用 Fail；Then 断言状态/业务码镜像、data 为 null。
func TestFail_FailureEnvelope(t *testing.T) {
	// Given
	c, w := ginTestCtx()
	// When
	Fail(c, http.StatusBadRequest, http.StatusBadRequest, "参数错误")
	// Then
	if w.Code != http.StatusBadRequest {
		t.Errorf("HTTP 状态 = %d, 期望 %d", w.Code, http.StatusBadRequest)
	}
	resp := decodeEnvelope(t, w)
	if resp.Code != http.StatusBadRequest {
		t.Errorf("业务码 = %d, 期望镜像 HTTP 状态 %d", resp.Code, http.StatusBadRequest)
	}
	if resp.Msg != "参数错误" {
		t.Errorf("msg = %q, 期望 %q", resp.Msg, "参数错误")
	}
	if resp.Data != nil {
		t.Errorf("data = %#v, 失败信封期望为 null", resp.Data)
	}
}

// TestJSON_CustomEnvelope：Given 自定义 HTTP 状态/业务码/消息/数据；
// When 调用 JSON；Then 断言信封各字段逐一符合。
func TestJSON_CustomEnvelope(t *testing.T) {
	// Given
	c, w := ginTestCtx()
	// When
	JSON(c, http.StatusCreated, 7, "自定义消息", map[string]any{"id": 1})
	// Then
	if w.Code != http.StatusCreated {
		t.Errorf("HTTP 状态 = %d, 期望 %d", w.Code, http.StatusCreated)
	}
	resp := decodeEnvelope(t, w)
	if resp.Code != 7 || resp.Msg != "自定义消息" {
		t.Errorf("信封 = {code:%d msg:%q}, 期望 {code:7 msg:%q}", resp.Code, resp.Msg, "自定义消息")
	}
	data, ok := resp.Data.(map[string]any)
	if !ok || data["id"] != float64(1) {
		t.Errorf("data = %#v, 期望 {id:1}", resp.Data)
	}
}
