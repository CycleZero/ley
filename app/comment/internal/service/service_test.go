package service

import (
	"errors"
	"testing"

	"ley/app/comment/internal/biz"

	klog "github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
)

func testCommentService() *CommentService {
	return &CommentService{logger: klog.NewStdLogger(io.Discard)}
}

func TestCommentService_MapError(t *testing.T) {
	svc := testCommentService()

	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
	}{
		{name: "评论不存在→NotFound", err: biz.ErrCommentNotFound, wantCode: codes.NotFound},
		{name: "超短→InvalidArgument", err: biz.ErrCommentTooShort, wantCode: codes.InvalidArgument},
		{name: "超长→InvalidArgument", err: biz.ErrCommentTooLong, wantCode: codes.InvalidArgument},
		{name: "非作者→PermissionDenied", err: biz.ErrNotCommentOwner, wantCode: codes.PermissionDenied},
		{name: "超深→InvalidArgument", err: biz.ErrMaxDepthExceeded, wantCode: codes.InvalidArgument},
		{name: "父评论不存在→NotFound", err: biz.ErrParentNotFound, wantCode: codes.NotFound},
		{name: "未知→Internal", err: errors.New("unknown"), wantCode: codes.Internal},
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
