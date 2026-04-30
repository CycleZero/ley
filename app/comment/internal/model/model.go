package model

import "gorm.io/gorm"

// Comment table: "comment".comments
type Comment struct {
	gorm.Model
	UUID     string        `gorm:"column:uuid;type:varchar(36);uniqueIndex;not null" json:"uuid"`
	PostID   uint          `gorm:"column:post_id;type:bigint;not null;index:idx_comments_post,where:deleted_at IS NULL" json:"post_id"`
	AuthorID uint          `gorm:"column:author_id;type:bigint;not null" json:"author_id"`
	ParentID *uint         `gorm:"column:parent_id;type:bigint;index:idx_comments_parent,where:deleted_at IS NULL" json:"parent_id"`
	Content  string        `gorm:"column:content;type:text;not null" json:"content"`
	Status   CommentStatus `gorm:"column:status;type:smallint;not null;default:0" json:"status"`

	Children []*Comment `gorm:"-" json:"children,omitempty"`
}

func (Comment) TableName() string {
	return "comment.comments"
}

type CommentStatus int8

const (
	CommentStatusPending  CommentStatus = 0
	CommentStatusApproved CommentStatus = 1
	CommentStatusSpam     CommentStatus = 2
	CommentStatusDeleted  CommentStatus = 3
)
