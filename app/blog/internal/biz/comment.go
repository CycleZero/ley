package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// =============================================================================
// CommentStatus
// =============================================================================

type CommentStatus int8

const (
	CommentStatusPending  CommentStatus = 0
	CommentStatusApproved CommentStatus = 1
	CommentStatusSpam     CommentStatus = 2
	CommentStatusDeleted  CommentStatus = 3
)

// =============================================================================
// Comment — 评论业务模型
// =============================================================================

type Comment struct {
	ID           uint
	ArticleID    uint
	AuthorID     uint
	AuthorName   string
	AuthorAvatar string
	ParentID     *uint
	Depth        int
	Content      string
	Status       CommentStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// =============================================================================
// CommentNode — 评论树节点
// =============================================================================

type CommentNode struct {
	Comment  *Comment
	Children []*CommentNode
}

// =============================================================================
// CommentRepo — 评论数据访问接口
// =============================================================================

type CommentRepo interface {
	Create(ctx context.Context, comment *Comment) error
	Update(ctx context.Context, comment *Comment) error
	Delete(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*Comment, error)
	ListByArticle(ctx context.Context, articleID uint, page, pageSize int) ([]*Comment, int64, error)
	UpdateStatus(ctx context.Context, id uint, status CommentStatus) error
	CountByArticle(ctx context.Context, articleID uint) (int64, error)
	BatchDeleteByArticle(ctx context.Context, articleID uint) error
}

// =============================================================================
// CommentUseCase
// =============================================================================

type CommentUseCase struct {
	repo     CommentRepo
	articleRepo ArticleRepo
	log      *log.Helper
}

func NewCommentUseCase(repo CommentRepo, articleRepo ArticleRepo, logger log.Logger) *CommentUseCase {
	return &CommentUseCase{repo: repo, articleRepo: articleRepo, log: log.NewHelper(logger)}
}
