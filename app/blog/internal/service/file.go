package service

import (
	"context"

	blogv1 "ley/api/blog/v1"
	"ley/app/blog/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type FileService struct {
	blogv1.UnimplementedFileServiceServer
	uc  *biz.FileUseCase
	log *log.Helper
}

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

func toFileInfo(f *biz.File) *blogv1.FileInfo {
	if f == nil {
		return nil
	}
	return &blogv1.FileInfo{
		Id: uint64(f.ID), Filename: f.Filename, MimeType: f.MimeType, Size: f.Size, Url: f.URL,
		CreatedAt: f.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
