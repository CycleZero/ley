// Package common 存放 entry 入口服务的跨层通用组件：
// 统一响应信封（Response）、gRPC 错误映射（MapGRPCError）与请求元数据传递（RequestMetaData）。
package common

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CodeOK 成功业务码：与前端契约 code === 0 表示成功。
const CodeOK = 0

// Response 统一响应信封：{code, msg, data}
//
// code 为业务码：0 表示成功，非 0 表示失败；失败时 code 与 HTTP 状态码数值一致
// （见 MapGRPCError），便于前端统一契约判断与错误提示。
type Response struct {
	Code int    `json:"code"` // 业务码：0 成功，非 0 失败
	Msg  string `json:"msg"`  // 提示消息（失败时为中文错误提示）
	Data any    `json:"data"` // 业务数据；失败时为 null
}

// JSON 以统一信封格式写出 JSON 响应。
//
// httpStatus 为 HTTP 状态码，code 为信封业务码；两者约定数值一致（code 镜像 httpStatus），
// 保留独立参数以便特殊场景微调；msg 建议为中文提示。
func JSON(c *gin.Context, httpStatus, code int, msg string, data any) {
	c.JSON(httpStatus, Response{Code: code, Msg: msg, Data: data})
}

// OK 快捷方式：写出 200 + 成功信封（业务码 CodeOK）。
func OK(c *gin.Context, data any) {
	JSON(c, http.StatusOK, CodeOK, "ok", data)
}

// Fail 快捷方式：写出失败信封（data 为 null）。
// 典型用法：gRPC 调用错误先经 MapGRPCError 映射为三元组后再写出：
//
//	httpStatus, code, msg := MapGRPCError(err)
//	Fail(c, httpStatus, code, msg)
func Fail(c *gin.Context, httpStatus, code int, msg string) {
	JSON(c, httpStatus, code, msg, nil)
}
