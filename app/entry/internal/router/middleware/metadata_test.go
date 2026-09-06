package middleware

import (
	"net/http"
	"testing"

	"github.com/CycleZero/ley/app/entry/internal/common"

	"github.com/gin-gonic/gin"
)

// newMetaEngine 构建挂载 AddMetaData 的测试引擎：
// /meta 路由回显 RequestID（gin ctx）与种子元数据（common 请求元数据）。
func newMetaEngine() *gin.Engine {
	e := gin.New()
	e.Use(AddMetaData())
	e.GET("/meta", func(c *gin.Context) {
		rid := c.GetString(requestIDKey)
		m := common.GetRequestMeta(c)
		hasMeta := m != nil
		var uid uint64
		if m != nil {
			uid = m.Auth.UserID
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": gin.H{
			"rid":      rid,
			"has_meta": hasMeta,
			"uid":      uid, // 种子 meta 应为匿名（0），认证信息由 JWT 中间件后续覆盖
		}})
	})
	return e
}

// TestAddMetaData 种子元数据 + RequestID 注入：
//  1. 响应头 X-Request-ID 存在且与 gin 上下文一致；
//  2. common 请求元数据已种入且为匿名（Auth 零值）。
func TestAddMetaData(t *testing.T) {
	e := newMetaEngine()

	rec := perform(t, e, http.MethodGet, "/meta", nil)
	wantStatus(t, rec, http.StatusOK)
	headerRID := rec.Header().Get(xRequestIDHeader)
	if headerRID == "" {
		t.Fatalf("响应头 X-Request-ID 不应为空")
	}
	env := decodeEnvelope(t, rec)
	if str(env.Data, "rid") != headerRID {
		t.Fatalf("gin 上下文 RequestID 与响应头不一致：ctx=%q header=%q", str(env.Data, "rid"), headerRID)
	}
	if has, _ := env.Data["has_meta"].(bool); !has {
		t.Fatalf("请求元数据应已种入 gin 上下文")
	}
	if uid := num(env.Data, "uid"); uid != 0 {
		t.Fatalf("种子元数据应为匿名（UserID=0），实际 %v", uid)
	}
}

// TestAddMetaDataRequestIDUnique RequestID 每次请求唯一（两次调用不同）。
func TestAddMetaDataRequestIDUnique(t *testing.T) {
	e := newMetaEngine()

	first := perform(t, e, http.MethodGet, "/meta", nil).Header().Get(xRequestIDHeader)
	second := perform(t, e, http.MethodGet, "/meta", nil).Header().Get(xRequestIDHeader)
	if first == "" || second == "" {
		t.Fatalf("两次请求的 X-Request-ID 均不应为空")
	}
	if first == second {
		t.Fatalf("两次请求的 RequestID 不应相同：%q", first)
	}
}
