package middleware

import (
	"net/http"
	"testing"

	"github.com/CycleZero/ley/pkg/log"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// attachObserverLogger 将全局日志器替换为内存观察器（zap observer），
// 使测试能断言日志级别与消息；返回的日志句柄按时间序记录全部条目。
// 调用方须 defer detachNopLogger() 恢复丢弃日志。
func attachObserverLogger() *observer.ObservedLogs {
	core, logs := observer.New(zapcore.DebugLevel)
	log.SetGlobalLogger(&log.Logger{Logger: zap.New(core)})
	return logs
}

// detachNopLogger 恢复全局日志器为丢弃日志（对齐 TestMain 初始态）。
func detachNopLogger() {
	log.SetGlobalLogger(&log.Logger{Logger: zap.NewNop()})
}

// TestRequestLoggerLevels 分级日志纪律验证（不可中断核心）：
//
//	2xx/3xx → Info；4xx → Warn；5xx → Error；
//	热路径禁止用 Error 刷屏：2xx/4xx 路径不得产生任何 Error 条目。
func TestRequestLoggerLevels(t *testing.T) {
	cases := []struct {
		name       string
		path       string
		wantStatus int
		wantLevel  zapcore.Level // 完成日志的期望级别
	}{
		{"200_Info路径", "/ping", http.StatusOK, zapcore.InfoLevel},
		{"404_Warn路径", "/missing", http.StatusNotFound, zapcore.WarnLevel},
		{"500_Error路径", "/boom", http.StatusInternalServerError, zapcore.ErrorLevel},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			logs := attachObserverLogger()
			defer detachNopLogger()

			e := gin.New()
			e.Use(RequestLogger())
			// Recovery 兜底 panic，保证 500 用例不炸测试进程
			e.Use(gin.CustomRecovery(func(c *gin.Context, _ any) {
				c.AbortWithStatusJSON(http.StatusInternalServerError,
					gin.H{"code": http.StatusInternalServerError, "msg": "服务器内部错误", "data": nil})
			}))
			e.GET("/ping", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": gin.H{"echo": "pong"}})
			})
			e.GET("/boom", func(c *gin.Context) { panic("测试用 panic") })

			rec := perform(t, e, http.MethodGet, tc.path, nil)
			wantStatus(t, rec, tc.wantStatus)

			// 日志分级断言（FilterLevelExact：精确匹配级别）
			exact := func(lvl zapcore.Level) int { return logs.FilterLevelExact(lvl).Len() }
			infos, warns, errs := exact(zapcore.InfoLevel), exact(zapcore.WarnLevel), exact(zapcore.ErrorLevel)
			switch tc.wantLevel {
			case zapcore.InfoLevel:
				if infos != 1 || warns != 0 || errs != 0 {
					t.Fatalf("2xx 分级不符：Info=%d Warn=%d Error=%d（期望 1/0/0）", infos, warns, errs)
				}
			case zapcore.WarnLevel:
				if infos != 0 || warns != 1 || errs != 0 {
					t.Fatalf("4xx 分级不符：Info=%d Warn=%d Error=%d（期望 0/1/0，4xx 不得刷 Error）", infos, warns, errs)
				}
			case zapcore.ErrorLevel:
				if infos != 0 || warns != 0 || errs != 1 {
					t.Fatalf("5xx 分级不符：Info=%d Warn=%d Error=%d（期望 0/0/1）", infos, warns, errs)
				}
			}
		})
	}
}

// TestRequestLoggerNotRewriteResponse 请求日志中间件不得改写响应（状态码/报文）。
func TestRequestLoggerNotRewriteResponse(t *testing.T) {
	defer detachNopLogger()
	attachObserverLogger()

	e := newEngine(RequestLogger())
	rec := perform(t, e, http.MethodGet, "/ping", nil)
	wantStatus(t, rec, http.StatusOK)
	env := decodeEnvelope(t, rec)
	if env.Code != 0 || str(env.Data, "echo") != "pong" {
		t.Fatalf("日志中间件改写响应：code=%d data=%v", env.Code, env.Data)
	}
}
