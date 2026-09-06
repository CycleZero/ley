package middleware

import (
	"net/http"
	"testing"
	"time"

	"github.com/CycleZero/ley/app/entry/internal/common"

	"github.com/gin-gonic/gin"
)

// TestJWTAuth JWT 认证中间件表驱动测试。
//
// 覆盖矩阵（对应任务验收点）：
//   - 未携带 token × optional true/false；
//   - 裸 token（无 Bearer 前缀）× optional true/false；
//   - 无效 token、过期 token；
//   - 黑名单命中 token × optional true/false（吊销令牌必须 401，不得按匿名放行）。
func TestJWTAuth(t *testing.T) {
	jwt := newTestJWT(15 * time.Minute)
	expiredJWT := newTestJWT(-time.Minute) // 负有效期 → 签发出即过期
	valid := issueToken(t, jwt, 42, "bob", "admin")
	expired := issueToken(t, expiredJWT, 42, "bob", "admin")

	// 预置黑名单：吊销 valid（黑名单启用 + 命中两条件齐备）
	blacklist := enabledBlacklist(t)
	if err := blacklist.Add(valid); err != nil {
		t.Fatalf("预置黑名单失败: %v", err)
	}
	if !blacklist.IsTokenBlackListed(valid) {
		t.Fatalf("测试前置失败：令牌应已入黑名单")
	}

	cases := []struct {
		name       string
		optional   bool
		authHeader string // "" = 不携带 Authorization 头
		wantStatus int
		wantMsg    string // 仅 401 场景校验
	}{
		{
			name:       "未携带令牌_强制认证_401未提供",
			optional:   false,
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
			wantMsg:    "未提供认证令牌",
		},
		{
			name:       "未携带令牌_可选认证_匿名放行",
			optional:   true,
			authHeader: "",
			wantStatus: http.StatusOK,
		},
		{
			name:       "裸token无Bearer前缀_强制认证_401无效",
			optional:   false,
			authHeader: valid, // 故意不带 Bearer 前缀
			wantStatus: http.StatusUnauthorized,
			wantMsg:    "无效的认证令牌",
		},
		{
			name:       "裸token无Bearer前缀_可选认证_匿名放行",
			optional:   true,
			authHeader: valid,
			wantStatus: http.StatusOK,
		},
		{
			name:       "无效token_强制认证_401无效",
			optional:   false,
			authHeader: "Bearer not-a-jwt",
			wantStatus: http.StatusUnauthorized,
			wantMsg:    "无效的认证令牌",
		},
		{
			name:       "无效token_可选认证_匿名放行",
			optional:   true,
			authHeader: "Bearer not-a-jwt",
			wantStatus: http.StatusOK,
		},
		{
			name:       "过期token_强制认证_401无效",
			optional:   false,
			authHeader: "Bearer " + expired,
			wantStatus: http.StatusUnauthorized,
			wantMsg:    "无效的认证令牌",
		},
		{
			// 不可中断核心：黑名单检查必须先于解析——本用例 token 本身合法可解析，
			// 若中间件先解析后查黑名单，用例将落入「无效令牌」而非「令牌已吊销」。
			name:       "黑名单token_强制认证_401已吊销",
			optional:   false,
			authHeader: "Bearer " + valid,
			wantStatus: http.StatusUnauthorized,
			wantMsg:    "令牌已吊销",
		},
		{
			// optional 语义边界：吊销令牌是明确的安全信号，optional 也不得放行
			name:       "黑名单token_可选认证_仍401",
			optional:   true,
			authHeader: "Bearer " + valid,
			wantStatus: http.StatusUnauthorized,
			wantMsg:    "令牌已吊销",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newEngine(NewJWTAuth(jwt, blacklist)(tc.optional))
			headers := map[string]string{}
			if tc.authHeader != "" {
				headers["Authorization"] = tc.authHeader
			}
			rec := perform(t, e, http.MethodGet, "/ping", headers)

			if tc.wantStatus == http.StatusOK {
				// 放行断言：探测 handler 正常执行并返回成功信封
				wantStatus(t, rec, http.StatusOK)
				if env := decodeEnvelope(t, rec); env.Code != 0 {
					t.Fatalf("放行请求应返回成功信封，实际 code=%d msg=%q", env.Code, env.Msg)
				}
				return
			}
			wantFail(t, rec, tc.wantStatus, tc.wantMsg)
		})
	}
}

// TestJWTAuthInjectsUserMeta 有效 token 通过认证后，用户身份必须注入 gin 上下文
// （c.Set user_id/user_name/role，供 RBAC 读取）与 common 请求元数据（供代理 handler
// BuildRequestMeta 透传下游 gRPC）。
func TestJWTAuthInjectsUserMeta(t *testing.T) {
	jwt := newTestJWT(15 * time.Minute)
	token := issueToken(t, jwt, 42, "bob", "admin")

	e := gin.New()
	// 使用「启用且未命中」的黑名单：证明黑名单检查开启时正常令牌仍可通行
	e.Use(NewJWTAuth(jwt, enabledBlacklist(t))(false))
	e.GET("/me", func(c *gin.Context) {
		m := common.GetRequestMeta(c)
		ctxUID, _ := c.Get(userIDKey)
		ctxName, _ := c.Get(userNameKey)
		ctxRole, _ := c.Get(roleKey)
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": gin.H{
			"meta_uid":   m.Auth.UserID,
			"meta_name":  m.Auth.UserName,
			"meta_role":  m.Auth.Role,
			"ctx_uid":    ctxUID,
			"ctx_name":   ctxName,
			"ctx_role":   ctxRole,
			"has_meta":   m != nil,
			"meta_user0": m.Auth.UserID == 0, // false：必须携带真实身份
		}})
	})

	rec := perform(t, e, http.MethodGet, "/me", map[string]string{"Authorization": "Bearer " + token})
	wantStatus(t, rec, http.StatusOK)
	env := decodeEnvelope(t, rec)
	if env.Code != 0 {
		t.Fatalf("有效令牌应通过认证，实际 code=%d msg=%q", env.Code, env.Msg)
	}
	data := env.Data
	// common 请求元数据（下游 gRPC 身份来源）
	if uid := num(data, "meta_uid"); uid != 42 {
		t.Fatalf("meta.UserID 注入不符：期望 42，实际 %v", uid)
	}
	if str(data, "meta_name") != "bob" || str(data, "meta_role") != "admin" {
		t.Fatalf("meta 用户名/角色注入不符: %v", data)
	}
	// gin 上下文键（RBAC 角色来源）
	if uid := num(data, "ctx_uid"); uid != 42 {
		t.Fatalf("ctx user_id 注入不符：期望 42，实际 %v", uid)
	}
	if str(data, "ctx_role") != "admin" || str(data, "ctx_name") != "bob" {
		t.Fatalf("ctx 角色/用户名注入不符: %v", data)
	}
}

// num 安全读取信封 data 中的数值字段（JSON 数字解码为 float64）。
func num(data map[string]any, key string) float64 {
	v, _ := data[key].(float64)
	return v
}

// str 安全读取信封 data 中的字符串字段。
func str(data map[string]any, key string) string {
	v, _ := data[key].(string)
	return v
}
