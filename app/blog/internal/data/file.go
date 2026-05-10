package data

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/CycleZero/ley/app/blog/internal/biz"
	"github.com/CycleZero/ley/pkg/oss"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// =============================================================================
// FilePO — files 表（放在 public schema 或复用现有）
// =============================================================================

type FilePO struct {
	gorm.Model
	Filename string `gorm:"column:filename;type:varchar(255);not null"`
	MimeType string `gorm:"column:mime_type;type:varchar(128);not null"`
	Size     int64  `gorm:"column:size;type:bigint;default:0"`
	URL      string `gorm:"column:url;type:varchar(1024);not null"`
	UserID   uint   `gorm:"column:user_id;type:bigint;not null"`
}

func (FilePO) TableName() string { return "files" }

// =============================================================================
// fileRepo — biz.FileRepo 接口实现
// =============================================================================

type fileRepo struct {
	data *Data
	oss  oss.OSS
}

var _ biz.FileRepo = (*fileRepo)(nil)

func (r *fileRepo) Create(ctx context.Context, file *biz.File, content io.Reader) error {
	key := uuid.Must(uuid.NewV7()).String()
	if err := r.oss.PutObject(ctx, key, content, file.Size, file.MimeType); err != nil {
		return fmt.Errorf("upload to oss: %w", err)
	}

	file.URL = key

	// 写数据库
	po := &FilePO{Filename: file.Filename, MimeType: file.MimeType, Size: file.Size, URL: file.URL, UserID: file.UserID}
	if err := r.data.db.WithContext(ctx).Create(po).Error; err != nil {
		return fmt.Errorf("create file record: %w", err)
	}
	file.ID = po.ID
	file.CreatedAt = po.CreatedAt
	r.data.log.WithContext(ctx).Infof("[FileRepo.Create] 成功 id=%d key=%s", file.ID, key)
	return nil
}

func (r *fileRepo) FindByID(ctx context.Context, id uint) (*biz.File, error) {
	var po FilePO
	if err := r.data.db.WithContext(ctx).Where("id = ?", id).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, biz.ErrFileNotFound
		}
		return nil, err
	}
	return &biz.File{ID: po.ID, UserID: po.UserID, Filename: po.Filename, MimeType: po.MimeType, Size: po.Size, URL: po.URL, CreatedAt: po.CreatedAt}, nil
}

func (r *fileRepo) Delete(ctx context.Context, id uint) error {
	var po FilePO
	if err := r.data.db.WithContext(ctx).Where("id = ?", id).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return biz.ErrFileNotFound
		}
		return err
	}
	// 从 MinIO 删除（best effort）
	_ = r.oss.DeleteObject(ctx, po.URL)
	return r.data.db.WithContext(ctx).Delete(&po).Error
}

func (r *fileRepo) List(ctx context.Context, userID uint, page, pageSize int) ([]*biz.File, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	query := r.data.db.WithContext(ctx).Where("user_id = ?", userID)
	var total int64
	if err := query.Model(&FilePO{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var pos []FilePO
	if err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}
	files := make([]*biz.File, 0, len(pos))
	for i := range pos {
		po := &pos[i]
		files = append(files, &biz.File{ID: po.ID, UserID: po.UserID, Filename: po.Filename, MimeType: po.MimeType, Size: po.Size, URL: po.URL, CreatedAt: po.CreatedAt})
	}
	return files, total, nil
}

func (r *fileRepo) GetPresignedPutURL(ctx context.Context, key, mimeType string, expireSeconds int64) (string, error) {
	return r.oss.GetPresignedPutURL(ctx, key, mimeType, expireSeconds)
}
