package service

import (
	"context"
	"time"

	blogv1 "github.com/CycleZero/ley/api/blog/v1"
	"github.com/CycleZero/ley/app/blog/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

// FileService 文件服务传输层：上传/下载/预签名直传的 proto ↔ biz DTO 转换。
type FileService struct {
	blogv1.UnimplementedFileServiceServer
	uc  *biz.FileUseCase
	log *log.Helper
}

// presignedExpiresIn 预签名 URL 有效期（秒）。
const presignedExpiresIn = int64(3600)

// NewFileService 构造文件服务（依赖由 Wire 注入）。
func NewFileService(uc *biz.FileUseCase, logger log.Logger) *FileService {
	return &FileService{uc: uc, log: log.NewHelper(logger)}
}

// UploadFile 服务端直传文件（经 entry 转发 base64 内容）。
func (s *FileService) UploadFile(ctx context.Context, req *blogv1.UploadFileRequest) (*blogv1.UploadFileReply, error) {
	s.log.WithContext(ctx).Debug("收到文件上传请求")
	f, err := s.uc.Upload(ctx, req.Filename, req.MimeType, req.Content)
	if err != nil {
		return nil, err
	}
	return &blogv1.UploadFileReply{File: toFileInfo(f)}, nil
}

// GetFile 获取文件元信息（需登录且仅本人，IDOR 防护在 biz）。
func (s *FileService) GetFile(ctx context.Context, req *blogv1.GetFileRequest) (*blogv1.GetFileReply, error) {
	f, err := s.uc.GetFile(ctx, uint(req.Id))
	if err != nil {
		return nil, err
	}
	return &blogv1.GetFileReply{File: toFileInfo(f)}, nil
}

// DeleteFile 删除文件（仅本人，归属校验在 biz）。
func (s *FileService) DeleteFile(ctx context.Context, req *blogv1.DeleteFileRequest) (*blogv1.DeleteFileReply, error) {
	s.log.WithContext(ctx).Debug("收到文件删除请求")
	return &blogv1.DeleteFileReply{}, s.uc.DeleteFile(ctx, uint(req.Id))
}

// ListFiles 分页查询当前用户的文件列表。
func (s *FileService) ListFiles(ctx context.Context, req *blogv1.ListFilesRequest) (*blogv1.ListFilesReply, error) {
	files, total, err := s.uc.ListFiles(ctx, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, err
	}
	infos := make([]*blogv1.FileInfo, 0, len(files))
	for _, f := range files {
		infos = append(infos, toFileInfo(f))
	}
	return &blogv1.ListFilesReply{Files: infos, Total: total}, nil
}

// GetPresignedPutURL 获取预签名 PUT URL（需登录；客户端据此直传对象存储）。
func (s *FileService) GetPresignedPutURL(ctx context.Context, req *blogv1.GetPresignedPutURLRequest) (*blogv1.GetPresignedPutURLReply, error) {
	s.log.WithContext(ctx).Debug("收到预签名直传请求")
	url, key, err := s.uc.GetPresignedPutURL(ctx, req.Filename, req.MimeType)
	if err != nil {
		return nil, err
	}
	return &blogv1.GetPresignedPutURLReply{Url: url, ObjectKey: key}, nil
}

// CreatePresignedUpload 创建预签名上传记录（先登记，再返回直传 URL）。
func (s *FileService) CreatePresignedUpload(ctx context.Context, req *blogv1.CreatePresignedUploadRequest) (*blogv1.CreatePresignedUploadReply, error) {
	s.log.WithContext(ctx).Debug("收到创建预签名上传请求")
	url, key, err := s.uc.CreatePresignedUpload(ctx, req.Filename, req.MimeType, req.Size)
	if err != nil {
		return nil, err
	}
	return &blogv1.CreatePresignedUploadReply{PresignedUrl: url, ObjectKey: key, ExpiresIn: presignedExpiresIn}, nil
}

// CompletePresignedUpload 确认直传完成（校验对象已落存储后登记文件记录）。
func (s *FileService) CompletePresignedUpload(ctx context.Context, req *blogv1.CompletePresignedUploadRequest) (*blogv1.CompletePresignedUploadReply, error) {
	s.log.WithContext(ctx).Debug("收到确认预签名上传请求")
	f, err := s.uc.CompletePresignedUpload(ctx, req.ObjectKey, req.Filename, req.MimeType)
	if err != nil {
		return nil, err
	}
	return &blogv1.CompletePresignedUploadReply{File: toFileInfo(f)}, nil
}

// toFileInfo 将 biz 文件对象转为 proto FileInfo（时间为 UTC RFC3339）。
func toFileInfo(f *biz.File) *blogv1.FileInfo {
	if f == nil {
		return nil
	}
	return &blogv1.FileInfo{
		Id: uint64(f.ID), Filename: f.Filename, MimeType: f.MimeType, Size: f.Size, Url: f.URL,
		CreatedAt: f.CreatedAt.UTC().Format(time.RFC3339),
	}
}
