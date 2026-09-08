package service

import (
	"context"
	"time"

	blogv1 "github.com/CycleZero/ley/api/blog/v1"
	"github.com/CycleZero/ley/app/blog/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

// SiteService 站点配置服务传输层：站点信息/背景图/音乐播放列表的 proto ↔ biz DTO 转换。
type SiteService struct {
	blogv1.UnimplementedSiteServiceServer
	uc  *biz.SiteUseCase
	log *log.Helper
}

// NewSiteService 构造站点配置服务（依赖由 Wire 注入）。
func NewSiteService(uc *biz.SiteUseCase, logger log.Logger) *SiteService {
	return &SiteService{uc: uc, log: log.NewHelper(logger)}
}

// GetSiteConfig 获取站点配置（公开接口，未配置时返回默认值）。
func (s *SiteService) GetSiteConfig(ctx context.Context, _ *blogv1.GetSiteConfigRequest) (*blogv1.GetSiteConfigReply, error) {
	c, err := s.uc.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &blogv1.GetSiteConfigReply{Config: toSiteConfig(c)}, nil
}

// UpdateSiteConfig 更新站点配置（需管理员权限）。
func (s *SiteService) UpdateSiteConfig(ctx context.Context, req *blogv1.UpdateSiteConfigRequest) (*blogv1.UpdateSiteConfigReply, error) {
	s.log.WithContext(ctx).Debug("收到更新站点配置请求")
	c, err := s.uc.UpdateConfig(ctx, fromSiteConfig(req.Config))
	if err != nil {
		return nil, err
	}
	return &blogv1.UpdateSiteConfigReply{Config: toSiteConfig(c)}, nil
}

// ListBackgrounds 查询背景图列表（公开接口）。
func (s *SiteService) ListBackgrounds(ctx context.Context, _ *blogv1.ListBackgroundsRequest) (*blogv1.ListBackgroundsReply, error) {
	bgs, err := s.uc.ListBackgrounds(ctx)
	if err != nil {
		return nil, err
	}
	infos := make([]*blogv1.SiteBackground, 0, len(bgs))
	for _, b := range bgs {
		infos = append(infos, toSiteBackground(b))
	}
	return &blogv1.ListBackgroundsReply{Backgrounds: infos}, nil
}

// UploadBackground 上传背景图（需管理员权限；魔数校验真实图片类型）。
func (s *SiteService) UploadBackground(ctx context.Context, req *blogv1.UploadBackgroundRequest) (*blogv1.UploadBackgroundReply, error) {
	s.log.WithContext(ctx).Debug("收到上传背景图请求")
	bg, err := s.uc.AddBackground(ctx, req.Filename, req.Content)
	if err != nil {
		return nil, err
	}
	return &blogv1.UploadBackgroundReply{Background: toSiteBackground(bg)}, nil
}

// DeleteBackground 删除背景图（需管理员权限）。
func (s *SiteService) DeleteBackground(ctx context.Context, req *blogv1.DeleteBackgroundRequest) (*blogv1.DeleteBackgroundReply, error) {
	s.log.WithContext(ctx).Debug("收到删除背景图请求")
	return &blogv1.DeleteBackgroundReply{}, s.uc.DeleteBackground(ctx, uint(req.Id))
}

// SetActiveBackground 设置启用中的背景图（需管理员权限）。
func (s *SiteService) SetActiveBackground(ctx context.Context, req *blogv1.SetActiveBackgroundRequest) (*blogv1.SetActiveBackgroundReply, error) {
	s.log.WithContext(ctx).Debug("收到设置背景图请求")
	return &blogv1.SetActiveBackgroundReply{}, s.uc.SetActiveBackground(ctx, uint(req.Id))
}

// GetMusicPlaylist 获取音乐播放列表（公开接口）。
func (s *SiteService) GetMusicPlaylist(ctx context.Context, _ *blogv1.GetMusicPlaylistRequest) (*blogv1.GetMusicPlaylistReply, error) {
	p, err := s.uc.GetPlaylist(ctx)
	if err != nil {
		return nil, err
	}
	return &blogv1.GetMusicPlaylistReply{Playlist: toMusicPlaylist(p)}, nil
}

// UpdateMusicPlaylist 更新音乐播放列表（需管理员权限）。
func (s *SiteService) UpdateMusicPlaylist(ctx context.Context, req *blogv1.UpdateMusicPlaylistRequest) (*blogv1.UpdateMusicPlaylistReply, error) {
	s.log.WithContext(ctx).Debug("收到更新音乐播放列表请求")
	p, err := s.uc.UpdatePlaylist(ctx, fromMusicPlaylist(req.Playlist))
	if err != nil {
		return nil, err
	}
	return &blogv1.UpdateMusicPlaylistReply{Playlist: toMusicPlaylist(p)}, nil
}

// toSiteConfig 将 biz 站点配置转为 proto SiteConfig（EnableLikes 解引用，默认 false）。
func toSiteConfig(c *biz.SiteSetting) *blogv1.SiteConfig {
	if c == nil {
		return nil
	}
	return &blogv1.SiteConfig{
		SiteTitle: c.SiteTitle, SiteSubtitle: c.SiteSubtitle, SiteDescription: c.SiteDescription,
		SiteLogo: c.SiteLogo, SiteFavicon: c.SiteFavicon, SeoKeywords: c.SeoKeywords, SeoDescription: c.SeoDescription,
		SocialGithub: c.SocialGithub, SocialTwitter: c.SocialTwitter, SocialEmail: c.SocialEmail,
		FooterText: c.FooterText, IcpNumber: c.ICPNumber,
		EnableLikes: derefBool(c.EnableLikes),
	}
}

// fromSiteConfig 将 proto SiteConfig 转为 biz 站点配置（EnableLikes 取指针以区分未设置）。
func fromSiteConfig(c *blogv1.SiteConfig) *biz.SiteSetting {
	if c == nil {
		return nil
	}
	return &biz.SiteSetting{
		SiteTitle: c.SiteTitle, SiteSubtitle: c.SiteSubtitle, SiteDescription: c.SiteDescription,
		SiteLogo: c.SiteLogo, SiteFavicon: c.SiteFavicon, SeoKeywords: c.SeoKeywords, SeoDescription: c.SeoDescription,
		SocialGithub: c.SocialGithub, SocialTwitter: c.SocialTwitter, SocialEmail: c.SocialEmail,
		FooterText: c.FooterText, ICPNumber: c.IcpNumber,
		EnableLikes: boolPtr(c.EnableLikes),
	}
}

// toSiteBackground 将 biz 背景图对象转为 proto SiteBackground（时间为 UTC RFC3339）。
func toSiteBackground(b *biz.SiteBackground) *blogv1.SiteBackground {
	if b == nil {
		return nil
	}
	return &blogv1.SiteBackground{
		Id: uint64(b.ID), Filename: b.Filename, Url: b.URL, IsActive: b.IsActive,
		SortOrder: int32(b.SortOrder), CreatedAt: b.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// toMusicPlaylist 将 biz 音乐播放列表转为 proto MusicPlaylist。
func toMusicPlaylist(p *biz.MusicPlaylist) *blogv1.MusicPlaylist {
	if p == nil {
		return nil
	}
	tracks := make([]*blogv1.MusicTrack, 0, len(p.Tracks))
	for _, t := range p.Tracks {
		tracks = append(tracks, &blogv1.MusicTrack{Title: t.Title, Artist: t.Artist, Url: t.URL, CoverUrl: t.CoverURL})
	}
	return &blogv1.MusicPlaylist{Tracks: tracks}
}

// fromMusicPlaylist 将 proto MusicPlaylist 转为 biz 音乐播放列表。
func fromMusicPlaylist(p *blogv1.MusicPlaylist) *biz.MusicPlaylist {
	if p == nil {
		return nil
	}
	tracks := make([]biz.MusicTrack, 0, len(p.Tracks))
	for _, t := range p.Tracks {
		tracks = append(tracks, biz.MusicTrack{Title: t.Title, Artist: t.Artist, URL: t.Url, CoverURL: t.CoverUrl})
	}
	return &biz.MusicPlaylist{Tracks: tracks}
}
