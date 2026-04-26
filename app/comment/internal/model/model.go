package model

import (
	"time"

	"gorm.io/gorm"
)

// Comment table: "comment".comments
type Comment struct {
	ID        int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UUID      string         `gorm:"column:uuid;type:varchar(36);uniqueIndex;not null" json:"uuid"`
	PostID    int64          `gorm:"column:post_id;type:bigint;not null;index:idx_comments_post,where:deleted_at IS NULL" json:"post_id"`
	AuthorID  int64          `gorm:"column:author_id;type:bigint;not null" json:"author_id"`
	ParentID  *int64         `gorm:"column:parent_id;type:bigint;index:idx_comments_parent,where:deleted_at IS NULL" json:"parent_id"`
	Content   string         `gorm:"column:content;type:text;not null" json:"content"`
	Status    CommentStatus  `gorm:"column:status;type:smallint;not null;default:0" json:"status"`
	CreatedAt time.Time      `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;type:timestamptz;index" json:"-"`

	Children []*Comment `gorm:"-" json:"children,omitempty"` // 内存构建评论树，非数据库字段
}

func (Comment) TableName() string {
	return "comment.comments"
}

type CommentStatus int8

const (
	CommentStatusPending  CommentStatus = 0 // 待审核
	CommentStatusApproved CommentStatus = 1 // 已通过
	CommentStatusSpam     CommentStatus = 2 // 垃圾评论
	CommentStatusDeleted  CommentStatus = 3 // 已删除
)

func (s CommentStatus) String() string {
	switch s {
	case CommentStatusPending:
		return "pending"
	case CommentStatusApproved:
		return "approved"
	case CommentStatusSpam:
		return "spam"
	case CommentStatusDeleted:
		return "deleted"
	default:
		return "unknown"
	}
}
