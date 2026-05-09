package biz

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

// =============================================================================
// File — 文件业务模型
//
// 字段含义：
//   - ID: 数据库自增主键，用作 API 标识（对外暴露）
//   - UserID: 上传者主键，用于所有权校验
//   - URL: 文件的下载/访问地址（MinIO 公开地址或预签名 URL）
// =============================================================================

type File struct {
	ID        uint      // 主键
	UserID    uint      // 上传者
	Filename  string    // 原始文件名（经安全清理后）
	MimeType  string    // MIME 类型（从文件魔数和上传声明交叉验证）
	Size      int64     // 文件大小（字节）
	URL       string    // 访问地址
	CreatedAt time.Time // 上传时间
}

// =============================================================================
// FileRepo — 文件数据访问接口（依赖倒置原则）
//
// data 层负责：
//   - Create: 存储文件到 MinIO + 写入数据库记录
//   - Delete: 从 MinIO 删除 + 软删除数据库记录
//   - GetPresignedPutURL: 生成 MinIO 预签名上传 URL（让客户端直传）
// =============================================================================

type FileRepo interface {
	Create(ctx context.Context, file *File, content io.Reader) error
	FindByID(ctx context.Context, id uint) (*File, error)
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, userID uint, page, pageSize int) ([]*File, int64, error)
	GetPresignedPutURL(ctx context.Context, key, mimeType string, expireSeconds int64) (string, error)
}

// =============================================================================
// 安全校验常量
// =============================================================================

const (
	MaxFileSize   = 10 * 1024 * 1024 // 10MB — 最大文件大小
	MaxAvatarSize = 2 * 1024 * 1024  // 2MB  — 头像专用限制
	MaxAttachSize = 50 * 1024 * 1024 // 50MB — 附件专用限制
)

// =============================================================================
// allowedMimeTypes — MIME 类型白名单
//
// 仅允许图片、文档和压缩包。任何不在白名单中的类型都会被拒绝。
// 即便 MIME 类型在白名单中，仍需通过魔数校验（MIME 伪造检测）。
// =============================================================================

var allowedMimeTypes = map[string]bool{
	// 图片类型
	"image/jpeg":    true,
	"image/png":     true,
	"image/gif":     true,
	"image/webp":    true,
	"image/svg+xml": true,
	// 文档类型
	"application/pdf": true,
	"text/plain":      true,
	"text/markdown":   true,
	"text/csv":        true,
	// 压缩包
	"application/zip":  true,
	"application/gzip": true,
	"application/x-tar": true,
}

// =============================================================================
// allowedExtensions — 文件扩展名白名单
//
// 即使用户上传时声称某个 MIME 类型，我们仍独立检查扩展名。
// 双重校验（MIME + 扩展名）防止伪装。
// =============================================================================

var allowedExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
	".webp": true, ".svg": true, ".bmp": true,
	".pdf": true, ".txt": true, ".md": true, ".csv": true,
	".zip": true, ".gz": true, ".tar": true,
	".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
}

// =============================================================================
// magicNumbers — 文件魔数签名表（前8字节）
//
// 用于验证上传文件的实际内容是否与声明的 MIME 类型一致。
// 例如：攻击者声称 image/png 但文件实际是 .exe → 魔数不匹配 → 拒绝。
// 只有定义了魔数的 MIME 类型会进行此校验，未定义的类型直接放行。
// =============================================================================

var magicNumbers = map[string][]byte{
	"image/jpeg":      {0xFF, 0xD8, 0xFF},
	"image/png":       {0x89, 0x50, 0x4E, 0x47},
	"image/gif":       {0x47, 0x49, 0x46, 0x38},
	"image/webp":      {0x52, 0x49, 0x46, 0x46},
	"application/pdf": {0x25, 0x50, 0x44, 0x46},
	"application/zip": {0x50, 0x4B, 0x03, 0x04},
}

// =============================================================================
// 错误定义
// =============================================================================

var (
	ErrFileNotFound = kerrors.NotFound("FILE_NOT_FOUND", "文件不存在")
	ErrFileTooLarge = kerrors.BadRequest("FILE_TOO_LARGE", "文件大小超过限制")
	ErrMimeNotAllowed = kerrors.BadRequest("MIME_NOT_ALLOWED", "不支持的文件类型")
	ErrExtensionNotAllowed = kerrors.BadRequest("EXT_NOT_ALLOWED", "不支持的文件扩展名")
	ErrInvalidFilename = kerrors.BadRequest("INVALID_FILENAME", "非法的文件名")
	ErrMimeMismatch = kerrors.BadRequest("MIME_MISMATCH", "文件类型与实际内容不匹配")
	ErrInvalidImage = kerrors.BadRequest("INVALID_IMAGE", "图片文件损坏或非真实图片")
	ErrFilePermissionDenied = kerrors.Forbidden("FILE_PERMISSION_DENIED", "无权操作此文件")
)

// =============================================================================
// FileUseCase — 文件业务用例
//
// 封装文件上传、下载、删除、列表等全部业务逻辑。
// 核心职责：
//   - 上传安全校验链（文件名 → 扩展名 → MIME → 大小 → 魔数 → 图片完整性）
//   - 所有权校验（GetFile / DeleteFile 时验证操作者是否为上传者）
// =============================================================================

type FileUseCase struct {
	repo FileRepo     // 文件数据访问
	log  *log.Helper  // 结构化日志
}

func NewFileUseCase(repo FileRepo, logger log.Logger) *FileUseCase {
	return &FileUseCase{repo: repo, log: log.NewHelper(logger)}
}

// =============================================================================
// Upload — 上传文件（含完整安全校验链）
//
// 校验流程（6步，任意一步失败立即返回）：
//  1. 文件名安全清理 — 路径穿越防护（filepath.Clean + Base）
//  2. 扩展名白名单 — 检查文件后缀是否在允许列表内
//  3. MIME 类型白名单 — 检查声明的 Content-Type 是否在允许列表内
//  4. 文件大小限制 — 超过 MaxFileSize (10MB) 拒绝
//  5. 魔数校验 — 读取文件头部字节与声明的 MIME 类型签名比对
//  6. 图片完整性校验 — 对 image/* 类型额外解析文件头确认是真实图片
//
// 全部通过后方才委托 data 层存储。
// =============================================================================

func (uc *FileUseCase) Upload(ctx context.Context, filename, mimeType string, content []byte) (*File, error) {
	userID, err := getCurrentUserID(ctx)
	if err != nil {
		uc.log.WithContext(ctx).Debugf("[Upload] 未认证")
		return nil, ErrUserNotAuthenticated
	}

	contentLen := int64(len(content))
	uc.log.WithContext(ctx).Debugf("[Upload] 开始 filename=%q mime=%q size=%d user_id=%d",
		filename, mimeType, contentLen, userID)

	// ===================================================================
	// 校验1: 文件名安全清理
	// filepath.Clean 消除 ".." 和多余斜杠 → filepath.Base 取纯文件名
	// 若清理后为空、"."".." → 拒绝（路径穿越攻击）
	// ===================================================================
	baseName := filepath.Base(filepath.Clean(filename))
	if baseName == "." || baseName == ".." || baseName == "" {
		uc.log.WithContext(ctx).Warnf("[Upload] 校验1失败: 非法文件名 filename=%q cleaned=%q", filename, baseName)
		return nil, ErrInvalidFilename
	}
	uc.log.WithContext(ctx).Debugf("[Upload] 校验1通过: 文件名清理 original=%q → %q", filename, baseName)

	// ===================================================================
	// 校验2: 扩展名白名单
	// 统一转小写后检查，防止 .JPG / .PNG 等变体绕过
	// ===================================================================
	ext := strings.ToLower(filepath.Ext(baseName))
	if !allowedExtensions[ext] {
		uc.log.WithContext(ctx).Warnf("[Upload] 校验2失败: 不允许的扩展名 ext=%q filename=%q", ext, baseName)
		return nil, ErrExtensionNotAllowed
	}
	uc.log.WithContext(ctx).Debugf("[Upload] 校验2通过: 扩展名=%q", ext)

	// ===================================================================
	// 校验3: MIME 类型白名单
	// 检查 HTTP Content-Type 声明是否在允许列表中
	// ===================================================================
	if !allowedMimeTypes[mimeType] {
		uc.log.WithContext(ctx).Warnf("[Upload] 校验3失败: 不允许的MIME mime=%q filename=%q", mimeType, baseName)
		return nil, ErrMimeNotAllowed
	}
	uc.log.WithContext(ctx).Debugf("[Upload] 校验3通过: MIME=%q", mimeType)

	// ===================================================================
	// 校验4: 文件大小限制
	// 超过 MaxFileSize (10MB) 拒绝，防止恶意占用存储空间
	// ===================================================================
	if contentLen > MaxFileSize {
		uc.log.WithContext(ctx).Warnf("[Upload] 校验4失败: 文件过大 size=%d limit=%d filename=%q",
			contentLen, MaxFileSize, baseName)
		return nil, ErrFileTooLarge
	}
	uc.log.WithContext(ctx).Debugf("[Upload] 校验4通过: size=%d (max=%d)", contentLen, MaxFileSize)

	// ===================================================================
	// 校验5: 魔数校验（MIME 伪造检测）
	// 读取文件头部字节与声明的 MIME 类型签名比对
	// 例如：上传者声称 image/png，但文件头是 0xFFD8FF（JPEG）→ 拒绝
	// 只有定义了魔数的 MIME 类型才进行此校验（未定义则直接放行）
	// ===================================================================
	if !verifyMagicNumber(content, mimeType) {
		uc.log.WithContext(ctx).Warnf("[Upload] 校验5失败: MIME与文件内容不匹配 claimed=%q filename=%q",
			mimeType, baseName)
		return nil, ErrMimeMismatch
	}
	uc.log.WithContext(ctx).Debugf("[Upload] 校验5通过: 魔数匹配")

	// ===================================================================
	// 校验6: 图片完整性校验
	// 对 image/* 类型额外解析文件头，确保是可识别的图片格式
	// 防止把随机文件重命名为 .png 后上传
	// ===================================================================
	if strings.HasPrefix(mimeType, "image/") {
		if !isImageContent(content) {
			uc.log.WithContext(ctx).Warnf("[Upload] 校验6失败: 图片文件损坏 mime=%q filename=%q", mimeType, baseName)
			return nil, ErrInvalidImage
		}
		uc.log.WithContext(ctx).Debugf("[Upload] 校验6通过: 图片文件头验证通过")
	}

	// ===================================================================
	// 全部校验通过 → 委托 data 层存储
	// data 层负责：
	//   1. 生成 object key 并上传到 MinIO
	//   2. 写入数据库 files 表
	//   3. 返回填充了 ID、URL 的 File 对象
	// ===================================================================
	uc.log.WithContext(ctx).Debugf("[Upload] 全部安全校验通过, 委托data层存储")

	file := &File{
		UserID:   uint(userID),
		Filename: baseName,
		MimeType: mimeType,
		Size:     contentLen,
	}

	uc.log.WithContext(ctx).Debugf("[Upload] 调用 repo.Create filename=%q mime=%q size=%d", baseName, mimeType, contentLen)
	if err := uc.repo.Create(ctx, file, bytes.NewReader(content)); err != nil {
		uc.log.WithContext(ctx).Errorf("[Upload] data层存储失败 filename=%q err=%v", baseName, err)
		return nil, fmt.Errorf("upload: %w", err)
	}
	uc.log.WithContext(ctx).Debugf("[Upload] repo.Create 返回 id=%d url=%q", file.ID, file.URL)

	uc.log.WithContext(ctx).Infof("[Upload] 上传成功 id=%d filename=%q mime=%q size=%d",
		file.ID, file.Filename, file.MimeType, file.Size)
	return file, nil
}

// =============================================================================
// GetFile — 获取文件信息（含所有权校验）
//
// 流程：
//  1. 查询文件是否存在
//  2. 若用户已认证且不是文件上传者 → 返回 403
//  3. 未认证用户可访问（公开访问场景，如文章配图）
//
// IDOR 防护：已认证用户只能访问自己的文件。
// =============================================================================

func (uc *FileUseCase) GetFile(ctx context.Context, id uint) (*File, error) {
	uc.log.WithContext(ctx).Debugf("[GetFile] 开始 id=%d", id)

	// 步骤1: 查询文件
	file, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		uc.log.WithContext(ctx).Debugf("[GetFile] 文件不存在 id=%d", id)
		return nil, ErrFileNotFound
	}
	uc.log.WithContext(ctx).Debugf("[GetFile] 文件找到 id=%d owner=%d", id, file.UserID)

	// 步骤2: 所有权校验（IDOR 防护）
	// 若用户已认证但不是文件所有者 → 拒绝
	// 若用户未认证 → 放行（公开访问，如前端渲染文章中的图片）
	userID, err := getCurrentUserID(ctx)
	if err == nil && file.UserID != uint(userID) {
		uc.log.WithContext(ctx).Warnf("[GetFile] 权限拒绝 id=%d owner=%d requester=%d",
			id, file.UserID, userID)
		return nil, ErrFilePermissionDenied
	}

	uc.log.WithContext(ctx).Debugf("[GetFile] 成功 id=%d filename=%q", id, file.Filename)
	return file, nil
}

// =============================================================================
// DeleteFile — 删除文件（含所有权校验）
//
// 流程：
//  1. 查询文件是否存在
//  2. 验证操作者是否为上传者（非上传者返回 403）
//  3. 委托 data 层删除（MinIO + 数据库）
// =============================================================================

func (uc *FileUseCase) DeleteFile(ctx context.Context, id uint) error {
	uc.log.WithContext(ctx).Debugf("[DeleteFile] 开始 id=%d", id)

	// 步骤1: 查询文件
	file, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		uc.log.WithContext(ctx).Debugf("[DeleteFile] 文件不存在 id=%d", id)
		return ErrFileNotFound
	}
	uc.log.WithContext(ctx).Debugf("[DeleteFile] 文件找到 id=%d owner=%d", id, file.UserID)

	// 步骤2: 所有权校验 — 只有上传者可以删除
	userID, err := getCurrentUserID(ctx)
	if err != nil {
		uc.log.WithContext(ctx).Warnf("[DeleteFile] 未认证 id=%d", id)
		return ErrUserNotAuthenticated
	}
	if file.UserID != uint(userID) {
		uc.log.WithContext(ctx).Warnf("[DeleteFile] 权限拒绝 id=%d owner=%d requester=%d",
			id, file.UserID, userID)
		return ErrFilePermissionDenied
	}
	uc.log.WithContext(ctx).Debugf("[DeleteFile] 权限验证通过")

	// 步骤3: 委托 data 层删除
	uc.log.WithContext(ctx).Debugf("[DeleteFile] 调用 repo.Delete id=%d", id)
	if err := uc.repo.Delete(ctx, id); err != nil {
		uc.log.WithContext(ctx).Errorf("[DeleteFile] 删除失败 id=%d err=%v", id, err)
		return fmt.Errorf("delete file: %w", err)
	}

	uc.log.WithContext(ctx).Infof("[DeleteFile] 删除成功 id=%d filename=%q", id, file.Filename)
	return nil
}

// =============================================================================
// ListFiles — 分页查询文件列表（按用户）
//
// 只返回当前用户上传的文件。
// =============================================================================

func (uc *FileUseCase) ListFiles(ctx context.Context, page, pageSize int) ([]*File, int64, error) {
	uc.log.WithContext(ctx).Debugf("[ListFiles] 开始 page=%d page_size=%d", page, pageSize)

	// 提取当前用户 ID — 文件列表是用户私有的
	userID, err := getCurrentUserID(ctx)
	if err != nil {
		uc.log.WithContext(ctx).Warnf("[ListFiles] 未认证")
		return nil, 0, ErrUserNotAuthenticated
	}
	uc.log.WithContext(ctx).Debugf("[ListFiles] user_id=%d", userID)

	// 委托 data 层分页查询
	files, total, err := uc.repo.List(ctx, uint(userID), page, pageSize)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("[ListFiles] 查询失败 user_id=%d err=%v", userID, err)
		return nil, 0, fmt.Errorf("list files: %w", err)
	}

	uc.log.WithContext(ctx).Debugf("[ListFiles] 完成 total=%d returned=%d", total, len(files))
	return files, total, nil
}

// =============================================================================
// GetPresignedPutURL — 获取预签名上传 URL
//
// 用于客户端直传 MinIO 的场景（避免流量经过 Gateway/Blog 服务）。
//
// 流程：
//  1. 校验扩展名白名单
//  2. 生成唯一 object key（时间戳 + 随机数 + 扩展名）
//  3. 委托 data 层生成 MinIO 预签名 PUT URL（有效期 3600 秒）
//  4. 返回（url, object_key, nil）
//
// 客户端后续：
//   1. 使用返回的 URL 执行 HTTP PUT 直接上传文件到 MinIO
//   2. 上传成功后调用 Blog 服务确认（将 object_key 与元数据关联）
// =============================================================================

func (uc *FileUseCase) GetPresignedPutURL(ctx context.Context, filename, mimeType string) (string, string, error) {
	uc.log.WithContext(ctx).Debugf("[GetPresignedPutURL] 开始 filename=%q mime=%q", filename, mimeType)

	// 步骤1: 校验扩展名白名单
	ext := strings.ToLower(filepath.Ext(filename))
	if !allowedExtensions[ext] {
		uc.log.WithContext(ctx).Warnf("[GetPresignedPutURL] 不允许的扩展名 ext=%q", ext)
		return "", "", ErrExtensionNotAllowed
	}
	uc.log.WithContext(ctx).Debugf("[GetPresignedPutURL] 扩展名校验通过 ext=%q", ext)

	// 步骤2: 生成唯一 object key
	// 格式: {timestamp_nanos}_{random}_{ext}
	// 例如: 1715299200000000000_a1b2c3d4.jpg
	key := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), randomHex(8), ext)
	uc.log.WithContext(ctx).Debugf("[GetPresignedPutURL] 生成 object_key=%q", key)

	// 步骤3: 委托 data 层生成预签名 URL
	uc.log.WithContext(ctx).Debugf("[GetPresignedPutURL] 调用 repo.GetPresignedPutURL key=%q", key)
	url, err := uc.repo.GetPresignedPutURL(ctx, key, mimeType, 3600)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("[GetPresignedPutURL] 生成预签名URL失败 key=%q err=%v", key, err)
		return "", "", fmt.Errorf("presigned url: %w", err)
	}

	uc.log.WithContext(ctx).Infof("[GetPresignedPutURL] 成功 filename=%q key=%q", filename, key)
	return url, key, nil
}

// =============================================================================
// 文件安全校验辅助函数
// =============================================================================

// verifyMagicNumber 检查文件头部魔数是否与声明的 MIME 类型匹配
//
// 逻辑：
//   1. 查找该 MIME 类型对应的魔数字节序列
//   2. 未找到魔数定义 → 直接放行（如 text/plain 无固定文件头）
//   3. 文件长度不足 → 拒绝
//   4. 逐字节比对 → 全部匹配才通过
func verifyMagicNumber(data []byte, mimeType string) bool {
	signature, ok := magicNumbers[mimeType]
	if !ok {
		return true // 该 MIME 类型无魔数定义，放行
	}
	if len(data) < len(signature) {
		return false // 文件太小，不足以读取魔数
	}
	for i, b := range signature {
		if data[i] != b {
			return false // 魔数不匹配
		}
	}
	return true
}

// isImageContent 检查文件字节是否为可识别的图片格式
//
// 通过文件头魔数识别：PNG (89 50 4E 47), JPEG (FF D8 FF), GIF (47 49 46 38), WebP (52 49 46 46)
// 必须是至少 4 字节的文件头才能判断。
func isImageContent(data []byte) bool {
	if len(data) < 3 {
		return false
	}
	// PNG: 89 50 4E 47
	if len(data) >= 4 && data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return true
	}
	// JPEG: FF D8 FF
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return true
	}
	// GIF: 47 49 46 38 (GIF8)
	if len(data) >= 4 && data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x38 {
		return true
	}
	// WebP: 52 49 46 46 (RIFF)
	if len(data) >= 4 && data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 {
		return true
	}
	return false
}

// randomHex 生成 n 字节的随机十六进制字符串（用于 object key 去重）
func randomHex(n int) string {
	b := make([]byte, n)
	ts := uint64(time.Now().UnixNano())
	for i := 0; i < n; i++ {
		b[i] = hexChars[(ts>>uint(i*4))&0xF]
		ts = ts*1103515245 + 12345
	}
	return string(b)
}

var hexChars = "0123456789abcdef"
