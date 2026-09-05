package biz

import (
	"context"
	"io"
	"strings"
	"testing"
)

// stubFileRepo 满足 FileRepo 的最小测试替身
type stubFileRepo struct {
	url        string
	registered *File
}

func (s *stubFileRepo) Create(ctx context.Context, f *File, content io.Reader) error { return nil }
func (s *stubFileRepo) FindByID(ctx context.Context, id uint) (*File, error)          { return nil, ErrFileNotFound }
func (s *stubFileRepo) Delete(ctx context.Context, id uint) error                     { return nil }
func (s *stubFileRepo) List(ctx context.Context, userID uint, page, pageSize int) ([]*File, int64, error) {
	return nil, 0, nil
}
func (s *stubFileRepo) GetPresignedPutURL(ctx context.Context, key, mimeType string, expire int64) (string, error) {
	return s.url, nil
}
func (s *stubFileRepo) RegisterPresignedObject(ctx context.Context, f *File) error {
	s.registered = f
	f.ID = 1
	return nil
}

func setupFileUseCase() (*FileUseCase, *stubFileRepo) {
	repo := &stubFileRepo{url: "https://minio.example.test/presigned"}
	uc := NewFileUseCase(repo, testLogger())
	return uc, repo
}

// B-111: 创建预签名直传须鉴权并校验元数据；对象键由服务端签发
func TestCreatePresignedUpload(t *testing.T) {
	uc, repo := setupFileUseCase()

	t.Run("anonymous rejected", func(t *testing.T) {
		if _, _, err := uc.CreatePresignedUpload(context.Background(), "a.jpg", "image/jpeg", 1024); err != ErrUserNotAuthenticated {
			t.Errorf("匿名应拒绝: got %v", err)
		}
	})
	t.Run("bad extension rejected", func(t *testing.T) {
		if _, _, err := uc.CreatePresignedUpload(ctxWithRole(1, "admin"), "evil.exe", "image/jpeg", 1024); err != ErrExtensionNotAllowed {
			t.Errorf("非法扩展名应拒绝: got %v", err)
		}
	})
	t.Run("oversize rejected", func(t *testing.T) {
		if _, _, err := uc.CreatePresignedUpload(ctxWithRole(1, "admin"), "big.jpg", "image/jpeg", int64(MaxAttachSize)+1); err != ErrFileTooLarge {
			t.Errorf("超限大小应拒绝: got %v", err)
		}
	})
	t.Run("happy path issues server-side key", func(t *testing.T) {
		url, key, err := uc.CreatePresignedUpload(ctxWithRole(1, "admin"), "photo.JPG", "image/jpeg", 2048)
		if err != nil {
			t.Fatalf("应成功: %v", err)
		}
		if url != repo.url {
			t.Errorf("url 异常: %q", url)
		}
		if !strings.HasPrefix(key, presignedKeyPrefix) || !strings.HasSuffix(key, ".jpg") {
			t.Errorf("对象键应由服务端生成且含白名单扩展名: %q", key)
		}
	})
}

// B-111: 完成直传须鉴权、校验对象键前缀与重传的文件名
func TestCompletePresignedUpload(t *testing.T) {
	uc, repo := setupFileUseCase()

	t.Run("anonymous rejected", func(t *testing.T) {
		if _, err := uc.CompletePresignedUpload(context.Background(), "uploads/1.jpg", "a.jpg", "image/jpeg"); err != ErrUserNotAuthenticated {
			t.Errorf("匿名应拒绝: got %v", err)
		}
	})
	t.Run("foreign object key rejected", func(t *testing.T) {
		if _, err := uc.CompletePresignedUpload(ctxWithRole(1, "reader"), "other/1.jpg", "a.jpg", "image/jpeg"); err != ErrInvalidFilename {
			t.Errorf("非本服务签发键应拒绝: got %v", err)
		}
	})
	t.Run("happy path registers file", func(t *testing.T) {
		f, err := uc.CompletePresignedUpload(ctxWithRole(1, "admin"), "uploads/20260906_abc.jpg", "../a.jpg", "image/jpeg")
		if err != nil {
			t.Fatalf("应成功: %v", err)
		}
		if repo.registered == nil {
			t.Fatal("应调用 repo.RegisterPresignedObject")
		}
		if repo.registered.Filename != "a.jpg" {
			t.Errorf("文件名应清理: got %q", repo.registered.Filename)
		}
		if repo.registered.UserID != 1 || f.ID == 0 {
			t.Errorf("登记字段异常: %+v", f)
		}
	})
}
