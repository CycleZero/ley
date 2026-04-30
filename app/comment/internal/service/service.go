// Package service — 评论服务 HTTP/gRPC Handler 层
//
// 实现 api/comment/v1 生成的 CommentServiceServer 接口。
// 全部业务逻辑委托给 biz 层 CommentUseCase。
// 本层仅负责：参数提取、类型转换、错误映射。
package service

import (
	"context"
	"errors"

	commentv1 "ley/api/comment/v1"
	"ley/app/comment/internal/biz"
	"ley/app/comment/internal/model"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// =============================================================================
// CommentService — 实现 api/comment/v1.CommentServiceServer
// =============================================================================

// CommentService 评论服务 gRPC Handler
// 嵌入 UnimplementedCommentServiceServer 确保向前兼容（后续 protobuf 新增方法不会编译报错）
type CommentService struct {
	commentv1.UnimplementedCommentServiceServer // gRPC 向前兼容兜底实现

	uc     *biz.CommentUseCase // 评论业务用例，处理核心业务逻辑
	logger log.Logger          // Kratos 日志器
}

// NewCommentService 创建 CommentService
func NewCommentService(uc *biz.CommentUseCase, logger log.Logger) *CommentService {
	log.NewHelper(logger).Debug("CommentService 服务层创建成功")
	return &CommentService{uc: uc, logger: logger}
}

// log 返回携带上下文信息的日志辅助器
func (s *CommentService) log(ctx context.Context) *log.Helper {
	return log.NewHelper(log.WithContext(ctx, s.logger))
}

// =============================================================================
// Comment CRUD Handlers
// =============================================================================

// CreateComment 创建评论
func (s *CommentService) CreateComment(ctx context.Context, req *commentv1.CreateCommentRequest) (*commentv1.CreateCommentReply, error) {
	s.log(ctx).Debugw("服务层收到创建评论请求",
		"post_id", req.PostId, "parent_id", req.ParentId, "content_length", len(req.Content),
	)

	// 处理可选的父评论 ID：uint64 → *uint
	var parentID *uint
	if req.ParentId > 0 {
		pid := uint(req.ParentId)
		parentID = &pid
		s.log(ctx).Debugw("创建评论包含父评论ID", "parent_id", pid)
	}

	// 委托 biz 层处理
	comment, err := s.uc.CreateComment(ctx, uint(req.PostId), parentID, req.Content)
	if err != nil {
		s.log(ctx).Warnw("创建评论失败", "post_id", req.PostId, "错误", err)
		return nil, s.mapError(err)
	}

	s.log(ctx).Infow("创建评论成功",
		"comment_id", comment.ID, "post_id", comment.PostID, "author_id", comment.AuthorID,
	)
	return &commentv1.CreateCommentReply{Comment: toCommentInfo(comment)}, nil
}

// UpdateComment 更新评论
func (s *CommentService) UpdateComment(ctx context.Context, req *commentv1.UpdateCommentRequest) (*commentv1.UpdateCommentReply, error) {
	s.log(ctx).Debugw("服务层收到更新评论请求",
		"comment_id", req.Id, "content_length", len(req.Content),
	)

	// 委托 biz 层处理
	comment, err := s.uc.UpdateComment(ctx, uint(req.Id), req.Content)
	if err != nil {
		s.log(ctx).Warnw("更新评论失败", "comment_id", req.Id, "错误", err)
		return nil, s.mapError(err)
	}

	s.log(ctx).Infow("更新评论成功", "comment_id", comment.ID)
	return &commentv1.UpdateCommentReply{Comment: toCommentInfo(comment)}, nil
}

// DeleteComment 删除评论
func (s *CommentService) DeleteComment(ctx context.Context, req *commentv1.DeleteCommentRequest) (*commentv1.DeleteCommentReply, error) {
	s.log(ctx).Debugw("服务层收到删除评论请求", "comment_id", req.Id)

	// 委托 biz 层处理
	if err := s.uc.DeleteComment(ctx, uint(req.Id)); err != nil {
		s.log(ctx).Warnw("删除评论失败", "comment_id", req.Id, "错误", err)
		return nil, s.mapError(err)
	}

	s.log(ctx).Infow("删除评论成功", "comment_id", req.Id)
	return &commentv1.DeleteCommentReply{}, nil
}

// =============================================================================
// Comment List Handler
// =============================================================================

// ListComments 查询评论列表（树形结构）
func (s *CommentService) ListComments(ctx context.Context, req *commentv1.ListCommentsRequest) (*commentv1.ListCommentsReply, error) {
	s.log(ctx).Debugw("服务层收到查询评论列表请求",
		"post_id", req.PostId, "page", req.Page, "page_size", req.PageSize,
	)

	// 委托 biz 层查询并构建树形结构
	nodes, total, err := s.uc.ListComments(ctx, uint(req.PostId), int(req.Page), int(req.PageSize))
	if err != nil {
		s.log(ctx).Warnw("查询评论列表失败", "post_id", req.PostId, "错误", err)
		return nil, s.mapError(err)
	}

	// 将 CommentNode 树转换为 protobuf CommentNode 树
	commentNodes := make([]*commentv1.CommentNode, 0, len(nodes))
	for _, n := range nodes {
		commentNodes = append(commentNodes, toCommentNode(n))
	}

	s.log(ctx).Debugw("评论列表查询成功",
		"post_id", req.PostId, "total", total, "root_nodes", len(commentNodes),
	)
	return &commentv1.ListCommentsReply{
		Comments: commentNodes,
		Total:    int32(total),
	}, nil
}

// =============================================================================
// Moderation Handlers
// =============================================================================

// ApproveComment 审核通过
func (s *CommentService) ApproveComment(ctx context.Context, req *commentv1.ApproveCommentRequest) (*commentv1.ApproveCommentReply, error) {
	s.log(ctx).Infow("服务层收到审核通过请求", "comment_id", req.Id)

	if err := s.uc.ApproveComment(ctx, uint(req.Id)); err != nil {
		s.log(ctx).Warnw("审核通过评论失败", "comment_id", req.Id, "错误", err)
		return nil, s.mapError(err)
	}

	s.log(ctx).Infow("评论审核通过成功", "comment_id", req.Id)
	return &commentv1.ApproveCommentReply{}, nil
}

// RejectComment 审核拒绝
// 拒绝后评论状态变更为 Deleted（软删除）
func (s *CommentService) RejectComment(ctx context.Context, req *commentv1.RejectCommentRequest) (*commentv1.RejectCommentReply, error) {
	s.log(ctx).Infow("服务层收到审核拒绝请求", "comment_id", req.Id)

	if err := s.uc.UpdateStatus(ctx, uint(req.Id), model.CommentStatusDeleted); err != nil {
		s.log(ctx).Warnw("审核拒绝评论失败", "comment_id", req.Id, "错误", err)
		return nil, s.mapError(err)
	}

	s.log(ctx).Infow("评论审核拒绝成功", "comment_id", req.Id)
	return &commentv1.RejectCommentReply{}, nil
}

// MarkSpam 标记垃圾
func (s *CommentService) MarkSpam(ctx context.Context, req *commentv1.MarkSpamRequest) (*commentv1.MarkSpamReply, error) {
	s.log(ctx).Infow("服务层收到标记垃圾评论请求", "comment_id", req.Id)

	if err := s.uc.MarkSpam(ctx, uint(req.Id)); err != nil {
		s.log(ctx).Warnw("标记垃圾评论失败", "comment_id", req.Id, "错误", err)
		return nil, s.mapError(err)
	}

	s.log(ctx).Infow("标记垃圾评论成功", "comment_id", req.Id)
	return &commentv1.MarkSpamReply{}, nil
}

// =============================================================================
// Internal RPC Handlers
// =============================================================================

// CountByPost 统计文章评论数
func (s *CommentService) CountByPost(ctx context.Context, req *commentv1.CountByPostRequest) (*commentv1.CountByPostReply, error) {
	s.log(ctx).Debugw("服务层收到统计评论数请求", "post_id", req.PostId)

	// 委托 biz 层统计
	count, err := s.uc.CountByPost(ctx, uint(req.PostId))
	if err != nil {
		s.log(ctx).Warnw("统计文章评论数失败", "post_id", req.PostId, "错误", err)
		return nil, s.mapError(err)
	}

	s.log(ctx).Debugw("文章评论数统计完成", "post_id", req.PostId, "count", count)
	return &commentv1.CountByPostReply{Count: count}, nil
}

// BatchDeleteByPost 级联删除文章评论
func (s *CommentService) BatchDeleteByPost(ctx context.Context, req *commentv1.BatchDeleteByPostRequest) (*commentv1.BatchDeleteByPostReply, error) {
	s.log(ctx).Infow("服务层收到批量删除文章评论请求", "post_id", req.PostId)

	if err := s.uc.BatchDeleteByPost(ctx, uint(req.PostId)); err != nil {
		s.log(ctx).Warnw("批量删除文章评论失败", "post_id", req.PostId, "错误", err)
		return nil, s.mapError(err)
	}

	s.log(ctx).Infow("批量删除文章评论成功", "post_id", req.PostId)
	return &commentv1.BatchDeleteByPostReply{}, nil
}

// =============================================================================
// Helpers: 类型转换
// =============================================================================

// toCommentInfo 将 model.Comment 转换为 protobuf CommentInfo
// 处理 nil 指针、可选字段 ParentID 的转换、时间格式化
func toCommentInfo(comment *model.Comment) *commentv1.CommentInfo {
	if comment == nil {
		return nil
	}
	// 转换 ParentID（*uint → uint64）
	var parentID uint64
	if comment.ParentID != nil {
		parentID = uint64(*comment.ParentID)
	}
	return &commentv1.CommentInfo{
		Id:        uint64(comment.ID),
		Uuid:      comment.UUID,
		PostId:    uint64(comment.PostID),
		AuthorId:  uint64(comment.AuthorID),
		ParentId:  parentID,
		Content:   comment.Content,
		Status:    statusToString(comment.Status),
		CreatedAt: comment.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: comment.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// toCommentNode 将 biz.CommentNode 递归转换为 protobuf CommentNode
func toCommentNode(node *biz.CommentNode) *commentv1.CommentNode {
	if node == nil {
		return nil
	}
	// 递归转换子节点
	children := make([]*commentv1.CommentNode, 0, len(node.Children))
	for _, ch := range node.Children {
		children = append(children, toCommentNode(ch))
	}
	return &commentv1.CommentNode{
		Comment:  toCommentInfo(node.Comment),
		Children: children,
	}
}

// statusToString 将评论状态枚举转换为字符串
func statusToString(status model.CommentStatus) string {
	switch status {
	case model.CommentStatusPending:
		return "pending"
	case model.CommentStatusApproved:
		return "approved"
	case model.CommentStatusSpam:
		return "spam"
	case model.CommentStatusDeleted:
		return "deleted"
	default:
		return "unknown"
	}
}

// =============================================================================
// Error Mapping
// =============================================================================

// mapError 将业务层错误映射为 gRPC 状态错误
// biz 层定义的各类错误对应不同的 gRPC 状态码：
//   - ErrCommentNotFound     → NotFound（评论不存在）
//   - ErrCommentTooShort     → InvalidArgument（内容过短）
//   - ErrCommentTooLong      → InvalidArgument（内容过长）
//   - ErrNotCommentOwner     → PermissionDenied（无权操作）
//   - ErrMaxDepthExceeded    → InvalidArgument（嵌套过多）
//   - ErrParentNotFound      → NotFound（父评论不存在）
//   - 其他未知错误           → Internal（内部错误）
func (s *CommentService) mapError(err error) error {
	switch {
	case errors.Is(err, biz.ErrCommentNotFound):
		return status.Error(codes.NotFound, "comment not found")
	case errors.Is(err, biz.ErrCommentTooShort), errors.Is(err, biz.ErrCommentTooLong):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, biz.ErrNotCommentOwner):
		return status.Error(codes.PermissionDenied, "not comment owner")
	case errors.Is(err, biz.ErrMaxDepthExceeded):
		return status.Error(codes.InvalidArgument, "maximum nesting depth exceeded")
	case errors.Is(err, biz.ErrParentNotFound):
		return status.Error(codes.NotFound, "parent comment not found")
	default:
		// 未识别的错误：记录日志并返回通用内部错误
		s.logger.Log(log.LevelError, "服务层遇到未映射的错误", "错误", err)
		return status.Error(codes.Internal, "internal server error")
	}
}
