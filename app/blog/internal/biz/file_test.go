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
	file       *File
}

func (s *stubFileRepo) Create(ctx context.Context, f *File, content io.Reader) error { return nil }
func (s *stubFileRepo) FindByID(ctx context.Context, id uint) (*File, error) {
	if s.file != nil {
		return s.file, nil
	}
	return nil, ErrFileNotFound
}
func (s *stubFileRepo) Delete(ctx context.Context, id uint) error { return nil }
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

// FIX-3: GetFile 须登录且仅能访问自己的文件（entry 侧该路由为 AUTH；
// blog 侧必须同样收口，防止绕过 entry 直连 gRPC/HTTP 的 IDOR 读取）
func TestGetFileRequiresAuthAndOwnership(t *testing.T) {
	uc, repo := setupFileUseCase()
	repo.file = &File{ID: 7, UserID: 2, Filename: "private.pdf", MimeType: "application/pdf", URL: "objects/private.pdf"}

	t.Run("anonymous rejected", func(t *testing.T) {
		if _, err := uc.GetFile(context.Background(), 7); err != ErrUserNotAuthenticated {
			t.Errorf("匿名应拒绝: got %v", err)
		}
	})
	t.Run("owner allowed", func(t *testing.T) {
		f, err := uc.GetFile(ctxWithRole(2, "reader"), 7)
		if err != nil {
			t.Fatalf("所有者应可访问: %v", err)
		}
		if f.ID != 7 {
			t.Errorf("文件 ID 不符: %d", f.ID)
		}
	})
	t.Run("non-owner denied", func(t *testing.T) {
		if _, err := uc.GetFile(ctxWithRole(3, "reader"), 7); err != ErrFilePermissionDenied {
			t.Errorf("非所有者应拒绝: got %v", err)
		}
	})
}

// FIX-2: 旧版 GetPresignedPutURL 须登录——匿名签发预签名 PUT URL 等于开放
// 任意上传入口（无大小约束、对象不入 files 表）
func TestGetPresignedPutURLRequiresAuth(t *testing.T) {
	uc, repo := setupFileUseCase()

	t.Run("anonymous rejected", func(t *testing.T) {
		if _, _, err := uc.GetPresignedPutURL(context.Background(), "a.jpg", "image/jpeg"); err != ErrUserNotAuthenticated {
			t.Errorf("匿名应拒绝: got %v", err)
		}
	})
	t.Run("authenticated ok", func(t *testing.T) {
		url, key, err := uc.GetPresignedPutURL(ctxWithRole(1, "reader"), "a.jpg", "image/jpeg")
		if err != nil {
			t.Fatalf("登录用户应可签发: %v", err)
		}
		if url != repo.url || key == "" {
			t.Errorf("返回值异常: url=%q key=%q", url, key)
		}
	})
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
