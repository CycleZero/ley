package biz

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/CycleZero/ley/pkg/metrics"
	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"go.opentelemetry.io/otel/metric"
)

// =============================================================================
// SiteSetting — 站点配置
// =============================================================================

type SiteSetting struct {
	SiteTitle       string         `json:"site_title"`               // 站点标题（浏览器标签页显示）
	SiteSubtitle    string         `json:"site_subtitle"`            // 站点副标题（首页标语）
	SiteDescription string         `json:"site_description"`         // 站点简介（SEO description）
	SiteLogo        string         `json:"site_logo"`                // 站点 Logo URL
	SiteFavicon     string         `json:"site_favicon"`             // 浏览器 Favicon URL
	SeoKeywords     string         `json:"seo_keywords"`             // SEO 关键词（meta keywords）
	SeoDescription  string         `json:"seo_description"`          // SEO 描述（meta description，优先于 site_description）
	SocialGithub    string         `json:"social_github"`            // GitHub 主页链接
	SocialTwitter   string         `json:"social_twitter"`           // Twitter 主页链接
	SocialEmail     string         `json:"social_email"`             // 联系邮箱
	FooterText      string         `json:"footer_text"`              // 页脚文案（版权声明等）
	ICPNumber       string         `json:"icp_number"`               // ICP 备案号
	EnableLikes     *bool          `json:"enable_likes,omitempty"`   // 全站点赞开关
	MusicPlaylist   *MusicPlaylist `json:"music_playlist,omitempty"` // 歌单
}

// =============================================================================
// SiteBackground — 背景图片
// =============================================================================

type SiteBackground struct {
	ID        uint
	Filename  string
	URL       string
	MimeType  string
	IsActive  bool
	SortOrder int
	CreatedAt time.Time
}

// =============================================================================
// MusicTrack / MusicPlaylist — 歌单
// =============================================================================

type MusicTrack struct {
	Title    string
	Artist   string
	URL      string
	CoverURL string
}

type MusicPlaylist struct {
	Tracks []MusicTrack
}

// =============================================================================
// SiteRepo — 站点配置数据访问接口
// =============================================================================

type SiteRepo interface {
	GetConfig(ctx context.Context) (*SiteSetting, error)
	SaveConfig(ctx context.Context, config *SiteSetting) error
	// IsLikesEnabled 全站点赞开关（供 ArticleUseCase 读取 enable_likes）
	IsLikesEnabled(ctx context.Context) (bool, error)

	CreateBackground(ctx context.Context, bg *SiteBackground, file io.Reader) error
	DeleteBackground(ctx context.Context, id uint) error
	ListBackgrounds(ctx context.Context) ([]*SiteBackground, error)
	SetActiveBackground(ctx context.Context, id uint) error
}

// =============================================================================
// 站点配置错误
// =============================================================================

var (
	ErrBackgroundNotFound = kerrors.NotFound("BACKGROUND_NOT_FOUND", "背景图片不存在")
	ErrInvalidImageFormat = kerrors.BadRequest("INVALID_IMAGE", "仅支持 JPEG/PNG/GIF/WEBP 格式")
	ErrInvalidMusicURL    = kerrors.BadRequest("INVALID_MUSIC_URL", "音乐链接格式不正确")
)

// =============================================================================
// SiteUseCase
// =============================================================================

type SiteUseCase struct {
	repo           SiteRepo
	log            *log.Helper
	updateConfig   metric.Int64Counter
	addBackground  metric.Int64Counter
	delBackground  metric.Int64Counter
	setActive      metric.Int64Counter
	updatePlaylist metric.Int64Counter
}

// NewSiteUseCase 构造站点配置用例；业务指标在构造期创建（Wire 阶段，metrics.New 已完成）。
func NewSiteUseCase(repo SiteRepo, logger log.Logger) *SiteUseCase {
	return &SiteUseCase{
		repo:           repo,
		log:            log.NewHelper(logger),
		updateConfig:   metrics.Counter("blog_site_config_update_total", "站点配置更新次数"),
		addBackground:  metrics.Counter("blog_site_background_add_total", "背景图上传次数"),
		delBackground:  metrics.Counter("blog_site_background_delete_total", "背景图删除次数"),
		setActive:      metrics.Counter("blog_site_background_set_active_total", "背景图启用次数"),
		updatePlaylist: metrics.Counter("blog_site_playlist_update_total", "音乐播放列表更新次数"),
	}
}

// GetConfig 获取站点配置。优先读缓存，未命中查 DB。
func (uc *SiteUseCase) GetConfig(ctx context.Context) (*SiteSetting, error) {
	uc.log.WithContext(ctx).Debugf("[SiteUseCase.GetConfig] 开始")
	cfg, err := uc.repo.GetConfig(ctx)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("[SiteUseCase.GetConfig] 失败 err=%v", err)
		return nil, fmt.Errorf("get site config: %w", err)
	}
	return cfg, nil
}

// UpdateConfig 更新站点配置（仅管理员）。合并策略：仅覆盖传入的非零值字段，保留未传入字段的原值。
func (uc *SiteUseCase) UpdateConfig(ctx context.Context, newCfg *SiteSetting) (cfg *SiteSetting, err error) {
	defer func() { recordResult(ctx, uc.updateConfig, err) }()
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	uc.log.WithContext(ctx).Debugf("[SiteUseCase.UpdateConfig] 开始")

	// 读取当前配置
	old, err := uc.repo.GetConfig(ctx)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("[SiteUseCase.UpdateConfig] 读取旧配置失败 err=%v", err)
		return nil, err
	}

	// 合并：非零值覆盖，零值保留旧值
	merged := mergeConfig(old, newCfg)

	// 持久化
	if err := uc.repo.SaveConfig(ctx, merged); err != nil {
		uc.log.WithContext(ctx).Errorf("[SiteUseCase.UpdateConfig] 保存失败 err=%v", err)
		return nil, fmt.Errorf("update site config: %w", err)
	}

	uc.log.WithContext(ctx).Infof("[SiteUseCase.UpdateConfig] 成功")
	return merged, nil
}

// ListBackgrounds 获取背景图片列表。
func (uc *SiteUseCase) ListBackgrounds(ctx context.Context) ([]*SiteBackground, error) {
	return uc.repo.ListBackgrounds(ctx)
}

// AddBackground 上传背景图片（仅管理员）。校验 MIME 类型后再存储。
func (uc *SiteUseCase) AddBackground(ctx context.Context, filename string, content []byte) (bg *SiteBackground, err error) {
	defer func() { recordResult(ctx, uc.addBackground, err) }()
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	uc.log.WithContext(ctx).Debugf("[SiteUseCase.AddBackground] filename=%q size=%d", filename, len(content))

	if !isImageContent(content) {
		uc.log.WithContext(ctx).Warnf("[SiteUseCase.AddBackground] 非图片文件 filename=%q", filename)
		return nil, ErrInvalidImageFormat
	}

	bg = &SiteBackground{Filename: filename, MimeType: detectImageMimeType(content), SortOrder: 0}
	if err := uc.repo.CreateBackground(ctx, bg, ioReader(content)); err != nil {
		return nil, fmt.Errorf("add background: %w", err)
	}
	uc.log.WithContext(ctx).Infof("[SiteUseCase.AddBackground] 成功 id=%d", bg.ID)
	return bg, nil
}

// DeleteBackground 删除背景图片（仅管理员）。
func (uc *SiteUseCase) DeleteBackground(ctx context.Context, id uint) (err error) {
	defer func() { recordResult(ctx, uc.delBackground, err) }()
	if err := requireAdmin(ctx); err != nil {
		return err
	}
	return uc.repo.DeleteBackground(ctx, id)
}

// SetActiveBackground 激活指定背景图片（仅管理员）。
func (uc *SiteUseCase) SetActiveBackground(ctx context.Context, id uint) (err error) {
	defer func() { recordResult(ctx, uc.setActive, err) }()
	if err := requireAdmin(ctx); err != nil {
		return err
	}
	uc.log.WithContext(ctx).Debugf("[SiteUseCase.SetActiveBackground] id=%d", id)
	if err := uc.repo.SetActiveBackground(ctx, id); err != nil {
		return fmt.Errorf("set active background: %w", err)
	}
	return nil
}

// GetPlaylist 获取歌单（从站点配置 JSON 中读取）。
func (uc *SiteUseCase) GetPlaylist(ctx context.Context) (*MusicPlaylist, error) {
	cfg, err := uc.repo.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	return cfg.MusicPlaylist, nil
}

// UpdatePlaylist 更新歌单（仅管理员，写入站点配置 JSON）。
func (uc *SiteUseCase) UpdatePlaylist(ctx context.Context, playlist *MusicPlaylist) (p *MusicPlaylist, err error) {
	defer func() { recordResult(ctx, uc.updatePlaylist, err) }()
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	uc.log.WithContext(ctx).Debugf("[SiteUseCase.UpdatePlaylist] 开始 track_count=%d", len(playlist.Tracks))

	// 校验每个 track 的 URL
	for _, t := range playlist.Tracks {
		if !strings.HasPrefix(t.URL, "http://") && !strings.HasPrefix(t.URL, "https://") {
			return nil, ErrInvalidMusicURL
		}
	}

	// 读取当前完整配置
	cfg, err := uc.repo.GetConfig(ctx)
	if err != nil {
		return nil, err
	}

	// 更新歌单字段
	cfg.MusicPlaylist = playlist

	// 持久化
	if err := uc.repo.SaveConfig(ctx, cfg); err != nil {
		return nil, fmt.Errorf("update playlist: %w", err)
	}

	uc.log.WithContext(ctx).Infof("[SiteUseCase.UpdatePlaylist] 成功")
	return cfg.MusicPlaylist, nil
}

func mergeConfig(old, new *SiteSetting) *SiteSetting {
	r := *old
	if new.SiteTitle != "" {
		r.SiteTitle = new.SiteTitle
	}
	if new.SiteSubtitle != "" {
		r.SiteSubtitle = new.SiteSubtitle
	}
	if new.SiteDescription != "" {
		r.SiteDescription = new.SiteDescription
	}
	if new.SiteLogo != "" {
		r.SiteLogo = new.SiteLogo
	}
	if new.SiteFavicon != "" {
		r.SiteFavicon = new.SiteFavicon
	}
	if new.SeoKeywords != "" {
		r.SeoKeywords = new.SeoKeywords
	}
	if new.SeoDescription != "" {
		r.SeoDescription = new.SeoDescription
	}
	if new.SocialGithub != "" {
		r.SocialGithub = new.SocialGithub
	}
	if new.SocialTwitter != "" {
		r.SocialTwitter = new.SocialTwitter
	}
	if new.SocialEmail != "" {
		r.SocialEmail = new.SocialEmail
	}
	if new.FooterText != "" {
		r.FooterText = new.FooterText
	}
	if new.ICPNumber != "" {
		r.ICPNumber = new.ICPNumber
	}
	if new.EnableLikes != nil {
		r.EnableLikes = new.EnableLikes
	}
	if new.MusicPlaylist != nil {
		r.MusicPlaylist = new.MusicPlaylist
	}
	return &r
}

// ioReader 将 []byte 包装为 io.Reader。
func ioReader(data []byte) io.Reader {
	return &sliceReader{data: data}
}

type sliceReader struct {
	data []byte
	pos  int
}

func (r *sliceReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
