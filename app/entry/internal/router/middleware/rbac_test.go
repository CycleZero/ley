package middleware

import (
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// TestRequireRole RBAC 角色鉴权表驱动测试。
//
// 前置注入方式：用「预置中间件」模拟 JWT 认证注入的 role（真实链路中 role 由
// auth.go 认证通过后 c.Set(roleKey, ...) 注入，本测试聚焦 RequireRole 自身判定）。
func TestRequireRole(t *testing.T) {
	// setRole 构造预置角色中间件（nil 表示不注入 → 模拟未挂认证/匿名请求）
	setRole := func(role any) gin.HandlerFunc {
		return func(c *gin.Context) { c.Set(roleKey, role) }
	}

	cases := []struct {
		name       string
		prepare    gin.HandlerFunc // 预置角色；nil = 不注入
		allowed    []string
		wantStatus int
		wantMsg    string
	}{
		{
			name:       "缺少角色信息_403",
			prepare:    nil, // 认证中间件未执行（或匿名路由误挂鉴权）
			allowed:    []string{"admin"},
			wantStatus: http.StatusForbidden,
			wantMsg:    "无权访问：缺少角色信息",
		},
		{
			name:       "角色类型错误_403",
			prepare:    setRole(123), // 非 string 注入（上下文被污染/注入异常）
			allowed:    []string{"admin"},
			wantStatus: http.StatusForbidden,
			wantMsg:    "无权访问：缺少角色信息",
		},
		{
			name:       "角色不在白名单_403",
			prepare:    setRole("reader"),
			allowed:    []string{"admin", "author"},
			wantStatus: http.StatusForbidden,
			wantMsg:    "无权访问",
		},
		{
			name:       "角色命中白名单_放行",
			prepare:    setRole("admin"),
			allowed:    []string{"author", "admin"},
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mws := make([]gin.HandlerFunc, 0, 2)
			if tc.prepare != nil {
				mws = append(mws, tc.prepare)
			}
			mws = append(mws, RequireRole(tc.allowed...))
			e := newEngine(mws...)

			rec := perform(t, e, http.MethodGet, "/ping", nil)
			if tc.wantStatus == http.StatusOK {
				wantStatus(t, rec, http.StatusOK)
				if env := decodeEnvelope(t, rec); env.Code != 0 {
					t.Fatalf("授权角色应放行，实际 code=%d msg=%q", env.Code, env.Msg)
				}
				return
			}
			wantFail(t, rec, tc.wantStatus, tc.wantMsg)
		})
	}
}

// TestRequireRoleChainWithAuth 端到端链路：JWT 认证（可选）→ RBAC 串联挂载，
// 验证 rbac 实际消费的是 auth 注入的 role（而非测试预置中间件）。
func TestRequireRoleChainWithAuth(t *testing.T) {
	jwt := newTestJWT(15 * time.Minute)
	adminToken := issueToken(t, jwt, 1, "root", "admin")
	readerToken := issueToken(t, jwt, 2, "visitor", "reader")

	e := newEngine(
		NewJWTAuth(jwt, nil)(false),
		RequireRole("admin"),
	)

	// 管理员令牌放行
	rec := perform(t, e, http.MethodGet, "/ping", map[string]string{"Authorization": "Bearer " + adminToken})
	wantStatus(t, rec, http.StatusOK)

	// 读者令牌被拒（身份有效但角色不足）
	rec = perform(t, e, http.MethodGet, "/ping", map[string]string{"Authorization": "Bearer " + readerToken})
	wantFail(t, rec, http.StatusForbidden, "无权访问")
}
