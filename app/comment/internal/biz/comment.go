package biz

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"ley/app/comment/internal/model"
	"ley/pkg/meta"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

// =============================================================================
// 评论业务约束常量
// =============================================================================

const (
	minCommentLength = 1     // 评论最小字符数（UTF-8 字符）
	maxCommentLength = 2000  // 评论最大字符数（UTF-8 字符）
	maxCommentDepth  = 5     // 评论最大嵌套深度（顶级评论深度为 0）
)

// =============================================================================
// 业务错误定义
// =============================================================================

var (
	ErrCommentNotFound  = errors.New("comment not found")                      // 评论未找到
	ErrCommentTooShort  = errors.New("comment must be at least 1 character")   // 评论内容过短
	ErrCommentTooLong   = errors.New("comment must be at most 2000 characters") // 评论内容过长
	ErrNotCommentOwner  = errors.New("you can only modify your own comments")  // 非评论作者无权操作
	ErrMaxDepthExceeded = errors.New("maximum nesting depth exceeded")         // 嵌套深度超限
	ErrParentNotFound   = errors.New("parent comment not found")               // 父评论未找到
)

// =============================================================================
// CommentRepo — 评论数据访问接口（依赖倒置原则）
// =============================================================================

// CommentRepo 评论数据访问接口
// biz 层仅依赖此接口而非具体实现，实现依赖倒置原则（DIP）
// data 层负责提供具体实现
type CommentRepo interface {
	Create(ctx context.Context, comment *model.Comment) error
	Update(ctx context.Context, comment *model.Comment) error
	Delete(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*model.Comment, error)
	FindByUUID(ctx context.Context, uuid string) (*model.Comment, error)
	ListByPost(ctx context.Context, postID uint, page, pageSize int) ([]*model.Comment, int64, error)
	ListChildren(ctx context.Context, parentID uint) ([]*model.Comment, error)
	UpdateStatus(ctx context.Context, id uint, status model.CommentStatus) error
	CountByPost(ctx context.Context, postID uint) (int64, error)
	BatchDeleteByPost(ctx context.Context, postID uint) error
}

// =============================================================================
// CommentUseCase — 评论业务用例
// =============================================================================

// CommentUseCase 评论业务用例
// 封装评论相关的业务规则、权限校验和内容验证逻辑
type CommentUseCase struct {
	repo   CommentRepo // 数据访问接口（依赖倒置）
	logger log.Logger  // Kratos 日志器
}

// NewCommentUseCase 创建 CommentUseCase
func NewCommentUseCase(repo CommentRepo, logger log.Logger) *CommentUseCase {
	return &CommentUseCase{repo: repo, logger: logger}
}

// log 返回携带上下文信息的日志辅助器
// 自动从 ctx 中提取 Span 信息注入日志，实现日志与链路追踪关联
func (uc *CommentUseCase) log(ctx context.Context) *log.Helper {
	return log.NewHelper(log.WithContext(ctx, uc.logger))
}

// =============================================================================
// 评论 CRUD 业务方法
// =============================================================================

// CreateComment 创建评论
// 流程：获取当前用户 → 校验评论内容 → 检查嵌套深度 → 创建评论记录
func (uc *CommentUseCase) CreateComment(ctx context.Context, postID uint, parentID *uint, content string) (*model.Comment, error) {
	uc.log(ctx).Debugw("业务层开始创建评论", "post_id", postID, "parent_id", parentID)

	// 1) 获取当前登录用户 ID
	authorID, err := getCurrentUserID(ctx)
	if err != nil {
		uc.log(ctx).Warnw("创建评论时获取用户身份失败", "错误", err)
		return nil, err
	}
	uc.log(ctx).Debugw("当前用户ID获取成功", "author_id", authorID)

	// 2) 验证评论内容长度
	if err := validateCommentContent(content); err != nil {
		uc.log(ctx).Debugw("评论内容校验失败", "content_length", utf8.RuneCountInString(content), "错误", err)
		return nil, err
	}
	uc.log(ctx).Debugw("评论内容校验通过", "content_length", utf8.RuneCountInString(content))

	// 3) 如果有父评论，递归检查嵌套深度
	if parentID != nil && *parentID > 0 {
		uc.log(ctx).Debugw("开始检查嵌套深度", "parent_id", *parentID)
		if err := uc.checkDepth(ctx, *parentID, 1); err != nil {
			uc.log(ctx).Warnw("嵌套深度检查失败", "parent_id", *parentID, "错误", err)
			return nil, err
		}
		uc.log(ctx).Debugw("嵌套深度检查通过", "parent_id", *parentID)
	}

	// 4) 构造评论实体
	comment := &model.Comment{
		UUID:     uuid.Must(uuid.NewV7()).String(), // 使用 UUID v7（时间排序友好）
		PostID:   postID,
		AuthorID: uint(authorID),
		ParentID: parentID,                          // nil 表示顶级评论
		Content:  strings.TrimSpace(content),        // 去除首尾空白
		Status:   model.CommentStatusPending,        // 默认状态：待审核
	}
	uc.log(ctx).Debugw("评论实体构造完成", "uuid", comment.UUID, "post_id", postID, "author_id", comment.AuthorID, "status", "pending")

	// 5) 通过 repo 写入数据库
	if err := uc.repo.Create(ctx, comment); err != nil {
		uc.log(ctx).Errorw("创建评论时数据层操作失败", "post_id", postID, "错误", err)
		return nil, fmt.Errorf("create comment: %w", err)
	}

	uc.log(ctx).Infow("评论创建成功", "id", comment.ID, "post_id", postID, "author_id", comment.AuthorID)
	return comment, nil
}

// UpdateComment 更新评论内容
// 流程：权限校验 → 内容验证 → 更新
func (uc *CommentUseCase) UpdateComment(ctx context.Context, commentID uint, content string) (*model.Comment, error) {
	uc.log(ctx).Debugw("业务层开始更新评论", "comment_id", commentID)

	// 1) 权限校验：必须是评论作者本人
	comment, err := uc.authorizeComment(ctx, commentID)
	if err != nil {
		uc.log(ctx).Warnw("更新评论时权限校验失败", "comment_id", commentID, "错误", err)
		return nil, err
	}
	uc.log(ctx).Debugw("评论权限校验通过", "comment_id", commentID, "author_id", comment.AuthorID)

	// 2) 验证新内容长度
	if err := validateCommentContent(content); err != nil {
		uc.log(ctx).Debugw("更新评论时内容校验失败", "comment_id", commentID, "错误", err)
		return nil, err
	}

	// 3) 更新评论内容（去除首尾空白）
	comment.Content = strings.TrimSpace(content)
	uc.log(ctx).Debugw("开始执行评论内容持久化更新", "comment_id", commentID)
	if err := uc.repo.Update(ctx, comment); err != nil {
		uc.log(ctx).Errorw("更新评论时数据层操作失败", "id", commentID, "错误", err)
		return nil, fmt.Errorf("update comment: %w", err)
	}

	uc.log(ctx).Infow("评论更新成功", "id", commentID)
	return comment, nil
}

// DeleteComment 删除评论
// 流程：权限校验 → 删除
func (uc *CommentUseCase) DeleteComment(ctx context.Context, commentID uint) error {
	uc.log(ctx).Debugw("业务层开始删除评论", "comment_id", commentID)

	// 1) 权限校验：必须是评论作者本人
	comment, err := uc.authorizeComment(ctx, commentID)
	if err != nil {
		uc.log(ctx).Warnw("删除评论时权限校验失败", "comment_id", commentID, "错误", err)
		return err
	}
	uc.log(ctx).Debugw("删除评论权限校验通过", "comment_id", commentID)

	// 2) 执行删除
	uc.log(ctx).Debugw("开始执行评论删除操作", "comment_id", commentID)
	if err := uc.repo.Delete(ctx, commentID); err != nil {
		uc.log(ctx).Errorw("删除评论时数据层操作失败", "id", commentID, "错误", err)
		return fmt.Errorf("delete comment: %w", err)
	}

	uc.log(ctx).Infow("评论删除成功", "id", commentID, "post_id", comment.PostID)
	return nil
}

// =============================================================================
// 评论查询与审核
// =============================================================================

// ListComments 分页查询评论并构建树形结构
// 返回树形结构的评论节点列表，方便前端渲染嵌套评论
func (uc *CommentUseCase) ListComments(ctx context.Context, postID uint, page, pageSize int) ([]*CommentNode, int64, error) {
	uc.log(ctx).Debugw("业务层开始查询评论列表",
		"post_id", postID, "page", page, "page_size", pageSize,
	)

	// 从 data 层获取分页评论列表
	comments, total, err := uc.repo.ListByPost(ctx, postID, page, pageSize)
	if err != nil {
		uc.log(ctx).Errorw("查询评论列表时数据层操作失败", "post_id", postID, "错误", err)
		return nil, 0, fmt.Errorf("list comments: %w", err)
	}
	uc.log(ctx).Debugw("评论列表数据获取成功",
		"post_id", postID, "total", total, "returned", len(comments),
	)

	// 构建评论树形结构
	tree := BuildCommentTree(comments)
	uc.log(ctx).Debugw("评论树构建完成", "post_id", postID, "root_nodes", len(tree), "total_comments", total)

	return tree, total, nil
}

// ApproveComment 审核通过评论
// 将评论状态更新为已通过审核
func (uc *CommentUseCase) ApproveComment(ctx context.Context, commentID uint) error {
	uc.log(ctx).Infow("审核通过评论", "comment_id", commentID)
	return uc.repo.UpdateStatus(ctx, commentID, model.CommentStatusApproved)
}

// MarkSpam 标记评论为垃圾
// 将评论状态更新为垃圾评论
func (uc *CommentUseCase) MarkSpam(ctx context.Context, commentID uint) error {
	uc.log(ctx).Infow("标记评论为垃圾", "comment_id", commentID)
	return uc.repo.UpdateStatus(ctx, commentID, model.CommentStatusSpam)
}

// BatchDeleteByPost 按文章 ID 批量软删除评论
// 通常用于文章删除时的级联操作
func (uc *CommentUseCase) BatchDeleteByPost(ctx context.Context, postID uint) error {
	uc.log(ctx).Infow("业务层开始批量删除文章评论", "post_id", postID)
	err := uc.repo.BatchDeleteByPost(ctx, postID)
	if err != nil {
		uc.log(ctx).Errorw("批量删除文章评论失败", "post_id", postID, "错误", err)
	}
	return err
}

// UpdateStatus 更新评论审核状态（通用方法）
func (uc *CommentUseCase) UpdateStatus(ctx context.Context, id uint, status model.CommentStatus) error {
	uc.log(ctx).Debugw("业务层更新评论状态", "id", id, "status", status)
	return uc.repo.UpdateStatus(ctx, id, status)
}

// CountByPost 统计文章已通过审核的评论数
func (uc *CommentUseCase) CountByPost(ctx context.Context, postID uint) (int64, error) {
	uc.log(ctx).Debugw("业务层统计文章评论数", "post_id", postID)
	count, err := uc.repo.CountByPost(ctx, postID)
	if err != nil {
		uc.log(ctx).Errorw("统计文章评论数失败", "post_id", postID, "错误", err)
	}
	return count, err
}

// =============================================================================
// 权限与校验辅助方法
// =============================================================================

// authorizeComment 权限校验
// 验证当前请求用户是否为评论作者本人
// 返回评论实体（已从数据库加载）供后续更新/删除使用
func (uc *CommentUseCase) authorizeComment(ctx context.Context, commentID uint) (*model.Comment, error) {
	// 获取当前登录用户 ID
	userID, err := getCurrentUserID(ctx)
	if err != nil {
		uc.log(ctx).Debugw("权限校验时获取用户身份失败", "comment_id", commentID, "错误", err)
		return nil, err
	}

	// 从数据库加载评论
	uc.log(ctx).Debugw("权限校验：加载评论信息", "comment_id", commentID)
	comment, err := uc.repo.FindByID(ctx, commentID)
	if err != nil {
		uc.log(ctx).Debugw("权限校验时查询评论失败", "comment_id", commentID, "错误", err)
		return nil, ErrCommentNotFound
	}

	// 比对作者 ID
	if comment.AuthorID != uint(userID) {
		uc.log(ctx).Warnw("非评论作者尝试越权操作",
			"comment_id", commentID, "author_id", comment.AuthorID, "requester_id", userID,
		)
		return nil, ErrNotCommentOwner
	}

	uc.log(ctx).Debugw("评论权限校验通过",
		"comment_id", commentID, "author_id", comment.AuthorID,
	)
	return comment, nil
}

// checkDepth 递归检查嵌套深度
// parentID: 父评论 ID
// currentDepth: 当前深度（从 1 开始计数）
// 若超过 maxCommentDepth 则返回 ErrMaxDepthExceeded
func (uc *CommentUseCase) checkDepth(ctx context.Context, parentID uint, currentDepth int) error {
	// 深度超限检查
	if currentDepth >= maxCommentDepth {
		uc.log(ctx).Debugw("评论嵌套深度已达上限",
			"parent_id", parentID, "current_depth", currentDepth, "max_depth", maxCommentDepth,
		)
		return ErrMaxDepthExceeded
	}

	// 加载父评论
	parent, err := uc.repo.FindByID(ctx, parentID)
	if err != nil {
		uc.log(ctx).Debugw("检查深度时父评论未找到", "parent_id", parentID, "错误", err)
		return ErrParentNotFound
	}

	// 如果父评论还有父评论，递归继续检查
	if parent.ParentID != nil && *parent.ParentID > 0 {
		uc.log(ctx).Debugw("递归检查更上层深度",
			"parent_id", parentID, "grandparent_id", *parent.ParentID, "current_depth", currentDepth+1,
		)
		return uc.checkDepth(ctx, *parent.ParentID, currentDepth+1)
	}

	uc.log(ctx).Debugw("嵌套深度检查完毕，未超限",
		"parent_id", parentID, "final_depth", currentDepth,
	)
	return nil
}

// =============================================================================
// CommentNode — 评论树节点
// =============================================================================

// CommentNode 评论树节点
// 用于将扁平的评论列表转换为层级树形结构
type CommentNode struct {
	Comment  *model.Comment `json:"comment"`            // 当前层的评论
	Children []*CommentNode `json:"children,omitempty"` // 子评论列表（仅直接子评论）
}

// BuildCommentTree 构建评论树
// 将扁平的评论列表转换为树形结构
// 算法：第一遍遍历构建节点映射表，第二遍遍历将每个节点挂载到其父节点的 Children 中
// 时间 O(n)，空间 O(n)
func BuildCommentTree(comments []*model.Comment) []*CommentNode {
	if len(comments) == 0 {
		return nil
	}

	// 第一遍：为每个评论创建 CommentNode，建立 ID → 节点 的映射
	nodeMap := make(map[uint]*CommentNode, len(comments))
	var roots []*CommentNode

	for _, c := range comments {
		nodeMap[c.ID] = &CommentNode{Comment: c}
	}

	// 第二遍：根据 ParentID 挂载到父节点的 Children 或添加到根节点列表
	for _, c := range comments {
		node := nodeMap[c.ID]
		// 如果有父评论且父评论存在于当前列表中
		if c.ParentID != nil && *c.ParentID > 0 {
			if parent, ok := nodeMap[*c.ParentID]; ok {
				// 挂载为父节点的子节点
				parent.Children = append(parent.Children, node)
				continue
			}
			// 父评论不在当前列表中（可能在其他页），作为根节点处理
		}
		// 无父评论或父评论不在列表中，作为根节点
		roots = append(roots, node)
	}

	return roots
}

// =============================================================================
// 通用校验与工具函数
// =============================================================================

// validateCommentContent 验证评论内容长度
// 使用 UTF-8 字符计数确保多字节字符（如中文）正确计数
func validateCommentContent(content string) error {
	trimmed := strings.TrimSpace(content)           // 去除首尾空白
	charCount := utf8.RuneCountInString(trimmed)    // UTF-8 字符数（非字节数）
	if charCount < minCommentLength {
		return ErrCommentTooShort
	}
	if charCount > maxCommentLength {
		return ErrCommentTooLong
	}
	return nil
}

// getCurrentUserID 从上下文中获取当前登录用户 ID
// 优先从请求元数据（meta.GetRequestMetaData）中获取
// 其次从 context.Value("user_id") 中获取
// 若均未获取到则返回认证错误
func getCurrentUserID(ctx context.Context) (uint64, error) {
	// 方式一：通过请求元数据获取（由中间件注入）
	reqMeta := meta.GetRequestMetaData(ctx)
	if reqMeta != nil && reqMeta.Auth.UserID > 0 {
		return reqMeta.Auth.UserID, nil
	}

	// 方式二：通过 context.Value 获取（直接注入场景）
	if userID, ok := ctx.Value("user_id").(uint64); ok && userID > 0 {
		return userID, nil
	}

	return 0, errors.New("user not authenticated")
}
