package service

import (
	"context"
	"time"

	blogv1 "github.com/CycleZero/ley/api/blog/v1"
	"github.com/CycleZero/ley/app/blog/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type FileService struct {
	blogv1.UnimplementedFileServiceServer
	uc  *biz.FileUseCase
	log *log.Helper
}

const presignedExpiresIn = int64(3600)

func NewFileService(uc *biz.FileUseCase, logger log.Logger) *FileService {
	return &FileService{uc: uc, log: log.NewHelper(logger)}
}

func (s *FileService) UploadFile(ctx context.Context, req *blogv1.UploadFileRequest) (*blogv1.UploadFileReply, error) {
	f, err := s.uc.Upload(ctx, req.Filename, req.MimeType, req.Content)
	if err != nil {
		return nil, err
	}
	return &blogv1.UploadFileReply{File: toFileInfo(f)}, nil
}

func (s *FileService) GetFile(ctx context.Context, req *blogv1.GetFileRequest) (*blogv1.GetFileReply, error) {
	f, err := s.uc.GetFile(ctx, uint(req.Id))
	if err != nil {
		return nil, err
	}
	return &blogv1.GetFileReply{File: toFileInfo(f)}, nil
}

func (s *FileService) DeleteFile(ctx context.Context, req *blogv1.DeleteFileRequest) (*blogv1.DeleteFileReply, error) {
	return &blogv1.DeleteFileReply{}, s.uc.DeleteFile(ctx, uint(req.Id))
}

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

func (s *FileService) GetPresignedPutURL(ctx context.Context, req *blogv1.GetPresignedPutURLRequest) (*blogv1.GetPresignedPutURLReply, error) {
	url, key, err := s.uc.GetPresignedPutURL(ctx, req.Filename, req.MimeType)
	if err != nil {
		return nil, err
	}
	return &blogv1.GetPresignedPutURLReply{Url: url, ObjectKey: key}, nil
}

func (s *FileService) CreatePresignedUpload(ctx context.Context, req *blogv1.CreatePresignedUploadRequest) (*blogv1.CreatePresignedUploadReply, error) {
	url, key, err := s.uc.CreatePresignedUpload(ctx, req.Filename, req.MimeType, req.Size)
	if err != nil {
		return nil, err
	}
	return &blogv1.CreatePresignedUploadReply{PresignedUrl: url, ObjectKey: key, ExpiresIn: presignedExpiresIn}, nil
}

func (s *FileService) CompletePresignedUpload(ctx context.Context, req *blogv1.CompletePresignedUploadRequest) (*blogv1.CompletePresignedUploadReply, error) {
	f, err := s.uc.CompletePresignedUpload(ctx, req.ObjectKey, req.Filename, req.MimeType)
	if err != nil {
		return nil, err
	}
	return &blogv1.CompletePresignedUploadReply{File: toFileInfo(f)}, nil
}

func toFileInfo(f *biz.File) *blogv1.FileInfo {
	if f == nil {
		return nil
	}
	return &blogv1.FileInfo{
		Id: uint64(f.ID), Filename: f.Filename, MimeType: f.MimeType, Size: f.Size, Url: f.URL,
		CreatedAt: f.CreatedAt.UTC().Format(time.RFC3339),
	}
}
