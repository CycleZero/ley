package common

import (
	"net/http"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// grpcToHTTP gRPC 状态码 → HTTP 状态码标准映射。
// 业务码与 HTTP 状态码数值一致（前后端约定 code === httpStatus），故一次映射复用两处。
var grpcToHTTP = map[codes.Code]int{
	codes.InvalidArgument: http.StatusBadRequest,
	codes.NotFound:        http.StatusNotFound,
	codes.AlreadyExists:   http.StatusConflict,
	// Aborted：kratos 业务错误（*errors.Error，如 kerrors.Conflict）经 gRPC wire
	// 传输时按 google httpstatus 规则把 HTTP 409 编码为 codes.Aborted（而非
	// AlreadyExists），故须映射回 409——否则下游 Conflict 语义会落入 500 兜底。
	codes.Aborted:           http.StatusConflict,
	codes.PermissionDenied:  http.StatusForbidden,
	codes.Unauthenticated:   http.StatusUnauthorized,
	codes.DeadlineExceeded:  http.StatusGatewayTimeout,
	codes.Unavailable:       http.StatusServiceUnavailable,
	codes.ResourceExhausted: http.StatusTooManyRequests,
	codes.Internal:          http.StatusInternalServerError,
}

// grpcDefaultMsg 各状态码的中文默认提示：下游 gRPC 错误未携带任何消息时兜底使用。
var grpcDefaultMsg = map[codes.Code]string{
	codes.InvalidArgument:   "参数错误",
	codes.NotFound:          "资源不存在",
	codes.AlreadyExists:     "资源已存在",
	codes.Aborted:           "请求冲突",
	codes.PermissionDenied:  "无权限访问",
	codes.Unauthenticated:   "未认证或令牌失效",
	codes.DeadlineExceeded:  "服务处理超时，请稍后重试",
	codes.Unavailable:       "服务暂不可用，请稍后重试",
	codes.ResourceExhausted: "请求过于频繁，请稍后再试",
	codes.Internal:          "服务内部错误",
}

// defaultInternalMsg 未列入映射表的状态码（如 Canceled/Unknown）以及
// nil / 非 gRPC 错误的统一兜底文案：对外只暴露通用提示，不泄露内部细节。
const defaultInternalMsg = "服务内部错误"

// MapGRPCError 将 gRPC 调用错误映射为 (HTTP 状态码, 信封业务码, 中文提示消息)，
// 供 handler 直接配合 Fail 写出失败信封：
//
//	httpStatus, code, msg := MapGRPCError(err)
//	Fail(c, httpStatus, code, msg)
//
// 规则：
//   - 业务码 code 镜像 HTTP 状态码（前后端约定 code === httpStatus），
//     标准映射见 grpcToHTTP，未覆盖的状态码按 500 兜底；
//   - msg 优先级：Kratos 业务消息（FromError 可解码且非空，通常为中文）
//     → gRPC 原始状态消息 → 对应状态码的中文默认提示（见 grpcDefaultMsg）；
//   - nil / 非 gRPC 错误统一按 500 兜底并返回通用中文提示，
//     避免向上游泄露内部错误细节。
func MapGRPCError(err error) (httpStatus, code int, msg string) {
	// nil 错误：无状态可取，按内部错误兜底（grpc status.FromError(nil) 返回 nil 状态，不可解引用）
	if err == nil {
		return http.StatusInternalServerError, http.StatusInternalServerError, defaultInternalMsg
	}
	// 提取 gRPC 状态；普通 Go 错误（非 gRPC 状态）同样按内部错误兜底
	st, ok := status.FromError(err)
	if !ok {
		return http.StatusInternalServerError, http.StatusInternalServerError, defaultInternalMsg
	}

	// 业务码镜像 HTTP 状态
	httpStatus = httpStatusOf(st.Code())
	code = httpStatus

	// 消息三级优先级：
	// 1) Kratos 业务消息（语义最准确，多为下游业务的中文提示）
	if ke := kerrors.FromError(err); ke != nil && ke.Message != "" {
		msg = ke.Message
		return
	}
	// 2) gRPC 原始状态消息
	if st.Message() != "" {
		msg = st.Message()
		return
	}
	// 3) 中文默认提示兜底
	msg = grpcDefaultMsg[st.Code()]
	if msg == "" {
		msg = defaultInternalMsg
	}
	return
}

// httpStatusOf 将 gRPC 状态码映射为 HTTP 状态码；未列入映射表的按 500 兜底。
func httpStatusOf(c codes.Code) int {
	if hs, ok := grpcToHTTP[c]; ok {
		return hs
	}
	return http.StatusInternalServerError
}
