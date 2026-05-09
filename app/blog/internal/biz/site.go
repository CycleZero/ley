package biz

import (
	"context"
	"io"
	"time"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

// =============================================================================
// SiteSetting — 站点配置
// =============================================================================

type SiteSetting struct {
	SiteTitle          string `json:"site_title"`
	SiteSubtitle       string `json:"site_subtitle"`
	SiteDescription    string `json:"site_description"`
	SiteLogo           string `json:"site_logo"`
	SiteFavicon        string `json:"site_favicon"`
	SeoKeywords        string `json:"seo_keywords"`
	SeoDescription     string `json:"seo_description"`
	SocialGithub       string `json:"social_github"`
	SocialTwitter      string `json:"social_twitter"`
	SocialEmail        string `json:"social_email"`
	FooterText         string `json:"footer_text"`
	ICPNumber          string `json:"icp_number"`
	EnableComments     bool   `json:"enable_comments"`
	EnableLikes        bool   `json:"enable_likes"`
	AutoApproveComments bool  `json:"auto_approve_comments"`
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
