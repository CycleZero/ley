package service

import (
	"errors"
	"testing"

	"ley/app/post/internal/biz"

	klog "github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
)

func testPostService() *PostService {
	return &PostService{logger: klog.NewStdLogger(io.Discard)}
}

func TestPostService_MapError(t *testing.T) {
	svc := testPostService()

	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
	}{
		{name: "文章不存在→NotFound", err: biz.ErrPostNotFound, wantCode: codes.NotFound},
		{name: "标题空→InvalidArgument", err: biz.ErrPostTitleEmpty, wantCode: codes.InvalidArgument},
		{name: "内容空→InvalidArgument", err: biz.ErrPostContentEmpty, wantCode: codes.InvalidArgument},
		{name: "内容过大→InvalidArgument", err: biz.ErrPostContentTooBig, wantCode: codes.InvalidArgument},
		{name: "非作者→PermissionDenied", err: biz.ErrNotPostOwner, wantCode: codes.PermissionDenied},
		{name: "已发布→FailedPrecondition", err: biz.ErrPostAlreadyPublished, wantCode: codes.FailedPrecondition},
		{name: "标签不存在→NotFound", err: biz.ErrTagNotFound, wantCode: codes.NotFound},
		{name: "标签名存在→AlreadyExists", err: biz.ErrTagNameExists, wantCode: codes.AlreadyExists},
		{name: "未知→Internal", err: errors.New("boom"), wantCode: codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, ok := status.FromError(svc.mapError(tt.err))
			if !ok {
				t.Fatalf("not gRPC status: %v", tt.err)
			}
			if s.Code() != tt.wantCode {
				t.Errorf("%s: code = %s, want %s", tt.name, s.Code(), tt.wantCode)
			}
		})
	}
}
