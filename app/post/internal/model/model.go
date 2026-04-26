package model

import (
	"time"

	"gorm.io/gorm"
)

// Post table: "post".posts
type Post struct {
	ID           int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UUID         string         `gorm:"column:uuid;type:varchar(36);uniqueIndex;not null" json:"uuid"`
	Title        string         `gorm:"column:title;type:varchar(200);not null" json:"title"`
	Slug         string         `gorm:"column:slug;type:varchar(200);uniqueIndex:idx_posts_slug,where:deleted_at IS NULL;not null" json:"slug"`
	Content      string         `gorm:"column:content;type:text;not null;default:''" json:"content"`
	Excerpt      string         `gorm:"column:excerpt;type:text;not null;default:''" json:"excerpt"`
	CoverImage   string         `gorm:"column:cover_image;type:varchar(512);not null;default:''" json:"cover_image"`
	Status       PostStatus     `gorm:"column:status;type:smallint;not null;default:0;index:idx_posts_status,where:deleted_at IS NULL" json:"status"`
	AuthorID     int64          `gorm:"column:author_id;type:bigint;not null;index:idx_posts_author,where:deleted_at IS NULL" json:"author_id"`
	CategoryID   *int64         `gorm:"column:category_id;type:bigint;index:idx_posts_category,where:deleted_at IS NULL" json:"category_id"`
	ViewCount    int64          `gorm:"column:view_count;type:bigint;not null;default:0" json:"view_count"`
	LikeCount    int64          `gorm:"column:like_count;type:bigint;not null;default:0" json:"like_count"`
	CommentCount int64          `gorm:"column:comment_count;type:bigint;not null;default:0" json:"comment_count"`
	IsTop        bool           `gorm:"column:is_top;type:boolean;not null;default:false" json:"is_top"`
	SearchVector string         `gorm:"column:search_vector;type:tsvector;index:idx_posts_search,type:gin" json:"-"` // PostgreSQL full-text search
	PublishedAt  *time.Time     `gorm:"column:published_at;type:timestamptz;index:idx_posts_published,where:status = 1 AND deleted_at IS NULL,sort:desc" json:"published_at"`
	CreatedAt    time.Time      `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at;type:timestamptz;index" json:"-"`

	Tags []Tag `gorm:"many2many:post.posts_tags;" json:"tags,omitempty"`
}

func (Post) TableName() string {
	return "post.posts"
}

type PostStatus int8

const (
	PostStatusDraft     PostStatus = 0
	PostStatusPublished PostStatus = 1
	PostStatusArchived  PostStatus = 2
)

func (s PostStatus) String() string {
	switch s {
	case PostStatusDraft:
		return "draft"
	case PostStatusPublished:
		return "published"
	case PostStatusArchived:
		return "archived"
	default:
		return "unknown"
	}
}

// Tag table: "post".tags
type Tag struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"column:name;type:varchar(64);uniqueIndex;not null" json:"name"`
	Slug      string    `gorm:"column:slug;type:varchar(64);uniqueIndex;not null" json:"slug"`
	PostCount int64     `gorm:"column:post_count;type:bigint;not null;default:0" json:"post_count"`
	CreatedAt time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
}

func (Tag) TableName() string {
	return "post.tags"
}

// PostTag table: "post".posts_tags (M:N join table)
type PostTag struct {
	PostID int64 `gorm:"column:post_id;type:bigint;primaryKey;not null" json:"post_id"`
	TagID  int64 `gorm:"column:tag_id;type:bigint;primaryKey;not null" json:"tag_id"`
}

func (PostTag) TableName() string {
	return "post.posts_tags"
}

// Category table: "post".categories
type Category struct {
	ID          int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"column:name;type:varchar(64);not null" json:"name"`
	Slug        string    `gorm:"column:slug;type:varchar(64);uniqueIndex;not null" json:"slug"`
	Description string    `gorm:"column:description;type:text;not null;default:''" json:"description"`
	ParentID    *int64    `gorm:"column:parent_id;type:bigint;index" json:"parent_id"`
	SortOrder   int       `gorm:"column:sort_order;type:int;not null;default:0" json:"sort_order"`
	PostCount   int64     `gorm:"column:post_count;type:bigint;not null;default:0" json:"post_count"`
	CreatedAt   time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
}

func (Category) TableName() string {
	return "post.categories"
}
