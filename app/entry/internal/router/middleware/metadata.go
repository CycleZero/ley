// metadata.go —— 请求元数据中间件（种子 meta + RequestID，合并为单个 AddMetaData）。
//
// 形态决策（对比两处参考实现）：
//   - gin-template / reimbee 均为单函数 AddMetaData：同时完成「种子请求元数据」与
//     「生成 RequestID」两件事，中间件顺序天然一致，故沿用单函数形态；
//   - 本项目内部元数据载体是 common.RequestMetaData 结构体（非模板的
//     map/struct 混合形态），此处种子 UserID=0 的匿名 meta，认证信息由
//     JWT 认证中间件（按路由组挂载，晚于全局链执行）经 common.SetRequestMeta 覆盖。
//
// 全局链中本中间件位于 RequestLogger 之后：请求日志在完成时（c.Next 返回后）
// 可从 gin 上下文读取到本中间件写入的 RequestID，一并记入日志。
package middleware

import (
	"crypto/rand"
	"math/big"
	"strconv"
	"time"

	"github.com/CycleZero/ley/app/entry/internal/common"

	"github.com/gin-gonic/gin"
)

const (
	// requestIDKey RequestID 在 gin.Context 中的存储键（同包请求日志中间件读取）。
	requestIDKey = "request_id"
	// xRequestIDHeader 响应头：回传 RequestID，便于客户端/运维按 ID 追踪请求。
	xRequestIDHeader = "X-Request-ID"
)

// AddMetaData 为每个请求注入请求元数据：
//  1. 生成唯一 RequestID：种入 gin 上下文 + 写入响应头 X-Request-ID；
//  2. 种子 common.RequestMetaData（匿名：Auth 零值）——JWT 认证中间件在其后覆盖。
func AddMetaData() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := generateRequestID()
		c.Set(requestIDKey, requestID)
		c.Header(xRequestIDHeader, requestID)

		common.SetRequestMeta(c, &common.RequestMetaData{})

		c.Next()
	}
}

// generateRequestID 生成请求 ID：纳秒时间戳 + 8 位随机字母后缀（reimbee 模式）。
// 随机源失败时回退纯时间戳（同纳秒内理论可碰撞，概率极低且不阻塞请求）。
func generateRequestID() string {
	now := strconv.FormatInt(time.Now().UnixNano(), 10)
	suffix, err := randomLetters(8)
	if err != nil {
		return now
	}
	return now + suffix
}

// randomLetters 从 crypto/rand 生成指定长度的随机字母串。
func randomLetters(length int) (string, error) {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	result := make([]byte, length)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		result[i] = letters[n.Int64()]
	}
	return string(result), nil
}
