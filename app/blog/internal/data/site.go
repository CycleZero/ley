package data

import (
	"context"
	"encoding/json"
	"io"

	"github.com/CycleZero/ley/app/blog/internal/biz"
	"github.com/CycleZero/ley/pkg/oss"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// =============================================================================
// SiteSettingPO — 站点配置（单行 JSONB）
// =============================================================================

type SiteSettingPO struct {
	ID     uint            `gorm:"primaryKey"`
	Config json.RawMessage `gorm:"column:config;type:jsonb;default:'{}'"`
}

func (SiteSettingPO) TableName() string { return "article.site_settings" }

// SiteBackgroundPO — 背景图片表
type SiteBackgroundPO struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	Filename  string `gorm:"column:filename;type:varchar(255);not null"`
	URL       string `gorm:"column:url;type:varchar(1024);not null"`
	IsActive  bool   `gorm:"column:is_active;type:boolean;default:false"`
	SortOrder int    `gorm:"column:sort_order;type:int;default:0"`
}

func (SiteBackgroundPO) TableName() string { return "article.site_backgrounds" }

// =============================================================================
// siteRepo — biz.SiteRepo 接口实现
// =============================================================================

type siteRepo struct {
	data *Data
	oss  oss.OSS
}

var _ biz.SiteRepo = (*siteRepo)(nil)

// GetConfig 获取站点配置（单行 JSONB）。缓存 Key = "site:config"。
func (r *siteRepo) GetConfig(ctx context.Context) (*biz.SiteSetting, error) {
	// Cache-Aside
	cfgKey := "site:config"
	var cfg biz.SiteSetting
	if err := r.data.cache.GetObject(ctx, cfgKey, &cfg); err == nil {
		return &cfg, nil
	}

	// DB 查询
	var po SiteSettingPO
	if err := r.data.db.WithContext(ctx).Where("id = 1").First(&po).Error; err != nil {
		// 首次访问，表中无数据 → 返回空配置
		return &biz.SiteSetting{}, nil
	}

	json.Unmarshal(po.Config, &cfg)
	_ = r.data.cache.Set(ctx, cfgKey, &cfg, 10*60*1000*1000*1000) // 10min (nanoseconds)
	return &cfg, nil
}

// SaveConfig 保存站点配置。合并策略由 biz 层实现，data 层仅负责持久化。
func (r *siteRepo) SaveConfig(ctx context.Context, cfg *biz.SiteSetting) error {
	data, _ := json.Marshal(cfg)
	result := r.data.db.WithContext(ctx).Exec(
		`INSERT INTO "article".site_settings (id, config) VALUES (1, ?::jsonb) ON CONFLICT (id) DO UPDATE SET config = EXCLUDED.config`, data)
	if result.Error != nil {
		return result.Error
	}
	// 删除缓存
	_ = r.data.cache.Delete(ctx, "site:config")
	return nil
}

// CreateBackground 上传背景图片到 MinIO 并写入 DB。
func (r *siteRepo) CreateBackground(ctx context.Context, bg *biz.SiteBackground, file io.Reader) error {
	key := uuid.Must(uuid.NewV7()).String()
	if err := r.oss.PutObject(ctx, key, file, -1, "image/jpeg"); err != nil {
		return err
	}
	bg.URL = key
	bg.Filename = key
	po := &SiteBackgroundPO{Filename: bg.Filename, URL: bg.URL, IsActive: false, SortOrder: bg.SortOrder}
	if err := r.data.db.WithContext(ctx).Create(po).Error; err != nil {
		return err
	}
	bg.ID = po.ID
	_ = r.data.cache.Delete(ctx, "site:backgrounds")
	return nil
}

// DeleteBackground 删除背景图片。
func (r *siteRepo) DeleteBackground(ctx context.Context, id uint) error {
	var po SiteBackgroundPO
	if err := r.data.db.WithContext(ctx).Where("id = ?", id).First(&po).Error; err != nil {
		return biz.ErrBackgroundNotFound
	}
	_ = r.oss.DeleteObject(ctx, po.URL)
	_ = r.data.cache.Delete(ctx, "site:backgrounds")
	return r.data.db.WithContext(ctx).Delete(&po).Error
}

// ListBackgrounds 查询背景图片列表。Cache-Aside。
func (r *siteRepo) ListBackgrounds(ctx context.Context) ([]*biz.SiteBackground, error) {
	var bgs []*biz.SiteBackground
	if err := r.data.cache.GetObject(ctx, "site:backgrounds", &bgs); err == nil {
		return bgs, nil
	}
	var pos []SiteBackgroundPO
	if err := r.data.db.WithContext(ctx).Order("sort_order ASC").Find(&pos).Error; err != nil {
		return nil, err
	}
	bgs = make([]*biz.SiteBackground, 0, len(pos))
	for i := range pos {
		bgs = append(bgs, &biz.SiteBackground{ID: pos[i].ID, Filename: pos[i].Filename, URL: pos[i].URL, IsActive: pos[i].IsActive, SortOrder: pos[i].SortOrder})
	}
	_ = r.data.cache.Set(ctx, "site:backgrounds", bgs, 30*60*1000*1000*1000)
	return bgs, nil
}

// SetActiveBackground 激活背景图（事务：取消所有 → 激活目标）。
func (r *siteRepo) SetActiveBackground(ctx context.Context, id uint) error {
	return r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		tx.Model(&SiteBackgroundPO{}).Where("is_active = true").Update("is_active", false)
		result := tx.Model(&SiteBackgroundPO{}).Where("id = ?", id).Update("is_active", true)
		if result.RowsAffected == 0 {
			return biz.ErrBackgroundNotFound
		}
		return result.Error
	})
}
