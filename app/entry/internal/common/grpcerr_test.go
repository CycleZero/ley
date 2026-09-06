package common

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestMapGRPCError_ExhaustiveMapping 穷举映射表：每个标准 gRPC 状态码（空消息构造）
// 断言 HTTP 状态 / 业务码与中文默认消息 —— 业务码必须镜像 HTTP 状态码。
func TestMapGRPCError_ExhaustiveMapping(t *testing.T) {
	cases := []struct {
		grpcCode codes.Code // 输入 gRPC 状态码
		httpCode int        // 期望 HTTP 状态码（业务码与其一致）
		defMsg   string     // 期望的中文默认消息
	}{
		{codes.InvalidArgument, http.StatusBadRequest, "参数错误"},
		{codes.NotFound, http.StatusNotFound, "资源不存在"},
		{codes.AlreadyExists, http.StatusConflict, "资源已存在"},
		// Aborted：kratos Conflict(409) 在 gRPC wire 上的编码（google httpstatus），须映射回 409
		{codes.Aborted, http.StatusConflict, "请求冲突"},
		{codes.PermissionDenied, http.StatusForbidden, "无权限访问"},
		{codes.Unauthenticated, http.StatusUnauthorized, "未认证或令牌失效"},
		{codes.DeadlineExceeded, http.StatusGatewayTimeout, "服务处理超时，请稍后重试"},
		{codes.Unavailable, http.StatusServiceUnavailable, "服务暂不可用，请稍后重试"},
		{codes.ResourceExhausted, http.StatusTooManyRequests, "请求过于频繁，请稍后再试"},
		{codes.Internal, http.StatusInternalServerError, "服务内部错误"},
	}
	for _, tc := range cases {
		t.Run(tc.grpcCode.String(), func(t *testing.T) {
			// Given: 下游返回空消息的 gRPC 错误
			err := status.Error(tc.grpcCode, "")
			// When: 调用错误映射
			httpStatus, code, msg := MapGRPCError(err)
			// Then: 状态与业务码镜像且命中期望值，空消息回落中文默认提示
			if httpStatus != tc.httpCode {
				t.Errorf("httpStatus = %d, 期望 %d", httpStatus, tc.httpCode)
			}
			if code != tc.httpCode {
				t.Errorf("code = %d, 期望镜像 httpStatus = %d", code, tc.httpCode)
			}
			if msg != tc.defMsg {
				t.Errorf("msg = %q, 期望中文默认 %q", msg, tc.defMsg)
			}
		})
	}
}

// TestMapGRPCError_MessagePriority 消息三级优先级：Kratos 业务消息 → gRPC 原始消息 → 中文默认。
func TestMapGRPCError_MessagePriority(t *testing.T) {
	kratosErr := kerrors.New(http.StatusNotFound, "ARTICLE_NOT_FOUND", "文章不存在")
	cases := []struct {
		name string
		err  error
		http int    // 期望 HTTP 状态码（业务码镜像）
		msg  string // 期望消息
	}{
		{
			name: "gRPC 原始消息透传（不回落中文默认）",
			err:  status.Error(codes.NotFound, "文章不存在"),
			http: http.StatusNotFound,
			msg:  "文章不存在",
		},
		{
			name: "英文 gRPC 消息原样透传",
			err:  status.Error(codes.Internal, "cache unavailable"),
			http: http.StatusInternalServerError,
			msg:  "cache unavailable",
		},
		{
			name: "Kratos 错误直传取业务消息",
			err:  kratosErr,
			http: http.StatusNotFound,
			msg:  "文章不存在",
		},
		{
			name: "Kratos 错误经 gRPC 传输后仍取业务消息",
			err:  kratosErr.GRPCStatus().Err(),
			http: http.StatusNotFound,
			msg:  "文章不存在",
		},
		{
			name: "Kratos 错误被 fmt 包装仍优先取业务消息",
			err:  fmt.Errorf("查询失败: %w", kratosErr),
			http: http.StatusNotFound,
			msg:  "文章不存在",
		},
		{
			name: "Kratos Conflict 经 gRPC 传输映射回 409（wire 编码为 Aborted）",
			err:  kerrors.Conflict("USER_DUPLICATE", "用户名或邮箱已存在").GRPCStatus().Err(),
			http: http.StatusConflict,
			msg:  "用户名或邮箱已存在",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// When: 调用错误映射
			httpStatus, code, msg := MapGRPCError(tc.err)
			// Then
			if httpStatus != tc.http || code != tc.http {
				t.Errorf("(httpStatus, code) = (%d, %d), 期望 (%d, %d)", httpStatus, code, tc.http, tc.http)
			}
			if msg != tc.msg {
				t.Errorf("msg = %q, 期望 %q", msg, tc.msg)
			}
		})
	}
}

// TestMapGRPCError_Fallback 兜底路径：nil / 普通 Go 错误 / 未列入映射表的状态码，
// 一律映射为 500 + 通用中文提示，不泄露内部错误细节。
func TestMapGRPCError_Fallback(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{name: "nil 错误", err: nil},
		{name: "普通 Go 错误（非 gRPC 状态）", err: errors.New("数据库连接失败")},
		{name: "未映射状态码 Canceled", err: status.Error(codes.Canceled, "")},
		{name: "未映射状态码 Unknown", err: status.Error(codes.Unknown, "")},
	}
	const wantMsg = "服务内部错误"
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// When: 调用错误映射
			httpStatus, code, msg := MapGRPCError(tc.err)
			// Then: 500 + 通用中文提示兜底
			if httpStatus != http.StatusInternalServerError || code != http.StatusInternalServerError {
				t.Errorf("(httpStatus, code) = (%d, %d), 期望 (500, 500)", httpStatus, code)
			}
			if msg != wantMsg {
				t.Errorf("msg = %q, 期望通用兜底 %q", msg, wantMsg)
			}
		})
	}
}
