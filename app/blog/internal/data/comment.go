package data

import (
	"context"
	"errors"
	"fmt"

	"ley/app/blog/internal/biz"

	"go.opentelemetry.io/otel/attribute"
	"gorm.io/gorm"
)

// =============================================================================
// CommentPO — comment.comments 表
// =============================================================================

type CommentPO struct {
	gorm.Model
	ArticleID uint   `gorm:"column:article_id;type:bigint;not null"`
	AuthorID  uint   `gorm:"column:author_id;type:bigint;not null"`
	ParentID  *uint  `gorm:"column:parent_id;type:bigint"`
	Depth     int    `gorm:"column:depth;type:smallint;default:0"`
	Content   string `gorm:"column:content;type:text;not null"`
	Status    int8   `gorm:"column:status;type:smallint;default:0"`
}

func (CommentPO) TableName() string { return "comment.comments" }

// =============================================================================
// commentRepo — biz.CommentRepo 接口实现
// =============================================================================

type commentRepo struct{ data *Data }

var _ biz.CommentRepo = (*commentRepo)(nil)

// Create 创建评论。
func (r *commentRepo) Create(ctx context.Context, c *biz.Comment) error {
	ctx, span := r.data.startSpan(ctx, "CommentRepo.Create")
	defer span.End()
	span.SetAttributes(attribute.Int("comment.article_id", int(c.ArticleID)))
	r.data.log.WithContext(ctx).Debugf("[CommentRepo.Create] article_id=%d depth=%d", c.ArticleID, c.Depth)

	po := &CommentPO{ArticleID: c.ArticleID, AuthorID: c.AuthorID, ParentID: c.ParentID, Depth: c.Depth, Content: c.Content, Status: int8(c.Status)}
	if err := r.data.db.WithContext(ctx).Create(po).Error; err != nil {
		r.data.log.WithContext(ctx).Errorf("[CommentRepo.Create] 失败 err=%v", err)
		return fmt.Errorf("create comment: %w", err)
	}
	c.ID = po.ID
	c.CreatedAt = po.CreatedAt
	c.UpdatedAt = po.UpdatedAt
	r.data.log.WithContext(ctx).Infof("[CommentRepo.Create] 成功 id=%d", c.ID)
	return nil
}

// Update 更新评论内容。
func (r *commentRepo) Update(ctx context.Context, c *biz.Comment) error {
	result := r.data.db.WithContext(ctx).Model(&CommentPO{}).Where("id = ?", c.ID).Update("content", c.Content)
	if result.RowsAffected == 0 {
		return biz.ErrCommentNotFound
	}
	return result.Error
}

// Delete 软删除。
func (r *commentRepo) Delete(ctx context.Context, id uint) error {
	result := r.data.db.WithContext(ctx).Where("id = ?", id).Delete(&CommentPO{})
	if result.RowsAffected == 0 {
		return biz.ErrCommentNotFound
	}
	return result.Error
}

// FindByID 按主键查询。
func (r *commentRepo) FindByID(ctx context.Context, id uint) (*biz.Comment, error) {
	var po CommentPO
	if err := r.data.db.WithContext(ctx).Where("id = ?", id).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, biz.ErrCommentNotFound
		}
		return nil, fmt.Errorf("find comment: %w", err)
	}
	return commentPOToBiz(&po), nil
}

// ListByArticle 分页查询文章的顶级评论。
func (r *commentRepo) ListByArticle(ctx context.Context, articleID uint, page, pageSize int) ([]*biz.Comment, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	query := r.data.db.WithContext(ctx).Where("article_id = ? AND status = 1", articleID)
	var total int64
	if err := query.Model(&CommentPO{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return nil, 0, nil
	}
	var pos []CommentPO
	if err := query.Order("created_at ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}
	comments := make([]*biz.Comment, 0, len(pos))
	for i := range pos {
		comments = append(comments, commentPOToBiz(&pos[i]))
	}
	return comments, total, nil
}

// UpdateStatus 更新评论状态。
func (r *commentRepo) UpdateStatus(ctx context.Context, id uint, status biz.CommentStatus) error {
	return r.data.db.WithContext(ctx).Model(&CommentPO{}).Where("id = ?", id).Update("status", int8(status)).Error
}

// CountByArticle 统计文章已通过审核的评论数。
func (r *commentRepo) CountByArticle(ctx context.Context, articleID uint) (int64, error) {
	var count int64
	err := r.data.db.WithContext(ctx).Model(&CommentPO{}).Where("article_id = ? AND status = 1", articleID).Count(&count).Error
	return count, err
}

// BatchDeleteByArticle 批量软删除文章下的所有评论。
func (r *commentRepo) BatchDeleteByArticle(ctx context.Context, articleID uint) error {
	return r.data.db.WithContext(ctx).Where("article_id = ?", articleID).Delete(&CommentPO{}).Error
}

// =============================================================================
// PO ↔ biz 转换
// =============================================================================

func commentPOToBiz(po *CommentPO) *biz.Comment {
	return &biz.Comment{
		ID: po.ID, ArticleID: po.ArticleID, AuthorID: po.AuthorID, ParentID: po.ParentID,
		Depth: po.Depth, Content: po.Content, Status: biz.CommentStatus(po.Status),
		CreatedAt: po.CreatedAt, UpdatedAt: po.UpdatedAt,
	}
}

var _ = context.Background
