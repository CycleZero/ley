package biz

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

// =============================================================================
// SiteSetting — 站点配置
// =============================================================================

type SiteSetting struct {
	SiteTitle           string         `json:"site_title"`           // 站点标题（浏览器标签页显示）
	SiteSubtitle        string         `json:"site_subtitle"`        // 站点副标题（首页标语）
	SiteDescription     string         `json:"site_description"`     // 站点简介（SEO description）
	SiteLogo            string         `json:"site_logo"`            // 站点 Logo URL
	SiteFavicon         string         `json:"site_favicon"`         // 浏览器 Favicon URL
	SeoKeywords         string         `json:"seo_keywords"`         // SEO 关键词（meta keywords）
	SeoDescription      string         `json:"seo_description"`      // SEO 描述（meta description，优先于 site_description）
	SocialGithub        string         `json:"social_github"`        // GitHub 主页链接
	SocialTwitter       string         `json:"social_twitter"`       // Twitter 主页链接
	SocialEmail         string         `json:"social_email"`         // 联系邮箱
	FooterText          string         `json:"footer_text"`          // 页脚文案（版权声明等）
	ICPNumber           string         `json:"icp_number"`           // ICP 备案号
	EnableComments      bool           `json:"enable_comments"`      // 全站评论开关
	EnableLikes         bool           `json:"enable_likes"`         // 全站点赞开关
	AutoApproveComments bool           `json:"auto_approve_comments"` // 评论自动审核通过（false=需管理员审核）
	MusicPlaylist       *MusicPlaylist `json:"music_playlist,omitempty"` // 歌单
}

// =============================================================================
// SiteBackground — 背景图片
// =============================================================================

type SiteBackground struct {
	ID        uint
	Filename  string
	URL       string
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
	repo SiteRepo
	log  *log.Helper
}

func NewSiteUseCase(repo SiteRepo, logger log.Logger) *SiteUseCase {
	return &SiteUseCase{repo: repo, log: log.NewHelper(logger)}
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

// UpdateConfig 更新站点配置。合并策略：仅覆盖传入的非零值字段，保留未传入字段的原值。
func (uc *SiteUseCase) UpdateConfig(ctx context.Context, newCfg *SiteSetting) (*SiteSetting, error) {
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

// AddBackground 上传背景图片。校验 MIME 类型后再存储。
func (uc *SiteUseCase) AddBackground(ctx context.Context, filename string, content []byte) (*SiteBackground, error) {
	uc.log.WithContext(ctx).Debugf("[SiteUseCase.AddBackground] filename=%q size=%d", filename, len(content))

	if !isImageContent(content) {
		uc.log.WithContext(ctx).Warnf("[SiteUseCase.AddBackground] 非图片文件 filename=%q", filename)
		return nil, ErrInvalidImageFormat
	}

	bg := &SiteBackground{Filename: filename, SortOrder: 0}
	if err := uc.repo.CreateBackground(ctx, bg, ioReader(content)); err != nil {
		return nil, fmt.Errorf("add background: %w", err)
	}
	uc.log.WithContext(ctx).Infof("[SiteUseCase.AddBackground] 成功 id=%d", bg.ID)
	return bg, nil
}

// DeleteBackground 删除背景图片。
func (uc *SiteUseCase) DeleteBackground(ctx context.Context, id uint) error {
	return uc.repo.DeleteBackground(ctx, id)
}

// SetActiveBackground 激活指定背景图片。
func (uc *SiteUseCase) SetActiveBackground(ctx context.Context, id uint) error {
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

// UpdatePlaylist 更新歌单（写入站点配置 JSON）。
func (uc *SiteUseCase) UpdatePlaylist(ctx context.Context, playlist *MusicPlaylist) (*MusicPlaylist, error) {
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
	r.EnableComments = new.EnableComments
	r.EnableLikes = new.EnableLikes
	r.AutoApproveComments = new.AutoApproveComments
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
