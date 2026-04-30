package biz

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ley/app/user/internal/model"
	"ley/pkg/meta"
	"ley/pkg/security"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

// 输入校验常量
const (
	minPasswordLength = 8    // 密码最小长度
	maxPasswordLength = 64   // 密码最大长度
	minUsernameLength = 3    // 用户名最小长度
	maxUsernameLength = 32   // 用户名最大长度
	maxNicknameLength = 64   // 昵称最大长度
)

// 业务层哨兵错误定义
// 各 UseCase 统一使用这些错误，service 层据此映射 gRPC 状态码
var (
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong  = errors.New("password must be at most 64 characters")
	ErrPasswordWeak     = errors.New("password must contain uppercase, lowercase, and digits")
	ErrUsernameInvalid  = errors.New("username must be 3-32 alphanumeric characters, underscores or hyphens")
	ErrNicknameTooLong  = errors.New("nickname must be at most 64 characters")
	ErrBioTooLong       = errors.New("bio must be at most 500 characters")
	ErrCannotEditOthers = errors.New("cannot modify another user's profile")

	ErrUserNotFound    = errors.New("user not found")
	ErrUserDuplicate   = errors.New("username or email already exists")
	ErrUsernameTaken   = errors.New("username already taken")
	ErrEmailTaken      = errors.New("email already registered")
	ErrBadCredentials  = errors.New("invalid username/email or password")
	ErrAccountDisabled = errors.New("account has been disabled")
)

// UserRepo 用户数据访问接口（依赖倒置原则）
// biz 层定义接口，data 层实现接口
// biz 层不依赖 data 层的具体实现，便于单元测试 Mock
type UserRepo interface {
	Create(ctx context.Context, user *model.User) error
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*model.User, error)
	FindByUUID(ctx context.Context, uuid string) (*model.User, error)
	FindByUsername(ctx context.Context, username string) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByAccount(ctx context.Context, account string) (*model.User, error)
	List(ctx context.Context, page, pageSize int) ([]*model.User, int64, error)
	UpdateStatus(ctx context.Context, id uint, status model.UserStatus) error
}

// UserUseCase 用户业务用例
// 封装用户资料相关的所有业务逻辑：注册、登录、资料管理
// 持有 repo 接口（依赖倒置）和 logger（日志记录）
type UserUseCase struct {
	repo   UserRepo   // 用户数据访问接口（data 层实现）
	logger log.Logger // Kratos 日志接口
}

// NewUserUseCase 创建 UserUseCase
// repo: UserRepo 接口实现（data.NewUserRepo 提供）
// logger: Kratos 日志接口（main 注入）
func NewUserUseCase(repo UserRepo, logger log.Logger) *UserUseCase {
	return &UserUseCase{repo: repo, logger: logger}
}

// log 创建携带链路追踪上下文信息的日志 Helper
func (uc *UserUseCase) log(ctx context.Context) *log.Helper {
	return log.NewHelper(log.WithContext(ctx, uc.logger))
}

// GetProfile 获取当前用户资料
// 从上下文提取 userID，查询用户完整信息
// 若上下文中无用户信息（未认证），返回 ErrBadCredentials
func (uc *UserUseCase) GetProfile(ctx context.Context) (*model.User, error) {
	// 从 context 中提取当前登录用户 ID
	userID, err := getCurrentUserID(ctx)
	if err != nil {
		uc.log(ctx).Warnw("获取用户资料：上下文中无用户信息", "error", err)
		return nil, ErrBadCredentials
	}

	uc.log(ctx).Debugw("获取用户资料开始", "userID", userID)

	user, err := uc.repo.FindByID(ctx, uint(userID))
	if err != nil {
		uc.log(ctx).Warnw("获取用户资料：查询用户失败", "userID", userID, "error", err)
		return nil, ErrUserNotFound
	}

	uc.log(ctx).Debugw("获取用户资料成功", "id", user.ID, "username", user.Username)
	return user, nil
}

// UpdateProfile 更新当前用户资料
// 校验输入字段长度，仅允许编辑本人的资料
// 更新成功后返回最新的用户对象
func (uc *UserUseCase) UpdateProfile(ctx context.Context, nickname, avatar, bio string) (*model.User, error) {
	// 从上下文提取当前用户 ID，防止越权编辑他人资料
	userID, err := getCurrentUserID(ctx)
	if err != nil {
		uc.log(ctx).Warnw("更新用户资料：上下文中无用户信息", "error", err)
		return nil, ErrBadCredentials
	}

	uc.log(ctx).Debugw("更新用户资料开始", "userID", userID, "nickname", nickname)

	// 字段长度校验
	if len(nickname) > maxNicknameLength {
		uc.log(ctx).Warnw("更新用户资料：昵称太长", "length", len(nickname), "max", maxNicknameLength)
		return nil, ErrNicknameTooLong
	}
	if len(bio) > 500 {
		uc.log(ctx).Warnw("更新用户资料：个人简介太长", "length", len(bio))
		return nil, ErrBioTooLong
	}

	// 查询用户确保存在
	user, err := uc.repo.FindByID(ctx, uint(userID))
	if err != nil {
		uc.log(ctx).Warnw("更新用户资料：用户不存在", "userID", userID)
		return nil, ErrUserNotFound
	}

	// 更新用户字段
	user.Nickname = nickname
	user.Avatar = avatar
	user.Bio = bio

	if err := uc.repo.Update(ctx, user); err != nil {
		uc.log(ctx).Errorw("更新用户资料失败", "id", user.ID, "error", err)
		return nil, fmt.Errorf("update profile: %w", err)
	}

	uc.log(ctx).Infow("用户资料更新成功", "id", user.ID, "username", user.Username)
	return user, nil
}

// Register 用户注册
// 流程：校验输入 → 检查用户名/邮箱唯一性 → 密码哈希 → 创建用户
// 返回创建成功的用户对象
func (uc *UserUseCase) Register(ctx context.Context, username, email, password, nickname string) (*model.User, error) {
	uc.log(ctx).Debugw("用户注册开始", "username", username, "email", email)

	// 校验用户名格式
	if err := validateUsername(username); err != nil {
		uc.log(ctx).Warnw("用户注册：用户名格式无效", "username", username, "error", err)
		return nil, err
	}
	// 校验密码强度
	if err := validatePassword(password); err != nil {
		uc.log(ctx).Warnw("用户注册：密码强度不足", "error", err)
		return nil, err
	}

	// 检查用户名是否已存在
	if _, err := uc.repo.FindByUsername(ctx, username); err == nil {
		uc.log(ctx).Warnw("用户注册：用户名已被占用", "username", username)
		return nil, ErrUsernameTaken
	}
	// 检查邮箱是否已被注册
	if _, err := uc.repo.FindByEmail(ctx, email); err == nil {
		uc.log(ctx).Warnw("用户注册：邮箱已被注册", "email", email)
		return nil, ErrEmailTaken
	}

	// 密码哈希处理（bcrypt 加盐哈希）
	hashedPassword, err := security.HashPassword(password)
	if err != nil {
		uc.log(ctx).Errorw("用户注册：密码哈希失败", "error", err)
		return nil, fmt.Errorf("register: %w", err)
	}

	// 构造用户对象，设置默认值
	user := &model.User{
		UUID:     uuid.Must(uuid.NewV7()).String(), // 生成 UUID v7（时间有序）
		Username: username,
		Email:    email,
		Password: hashedPassword,
		Nickname: nickname,
		Status:   model.UserStatusActive,  // 默认激活状态
		Role:     model.RoleReader,         // 默认读者角色
	}

	uc.log(ctx).Debugw("用户注册：准备创建用户", "uuid", user.UUID, "username", username)

	// 写入数据库（含唯一性约束检查）
	if err := uc.repo.Create(ctx, user); err != nil {
		if errors.Is(err, ErrUserDuplicate) {
			uc.log(ctx).Warnw("用户注册：用户名或邮箱冲突", "username", username, "email", email)
			return nil, err
		}
		uc.log(ctx).Errorw("用户注册：创建用户失败", "username", username, "error", err)
		return nil, fmt.Errorf("register: %w", err)
	}

	uc.log(ctx).Infow("用户注册成功", "id", user.ID, "username", user.Username)
	return user, nil
}

// Login 用户登录（验证凭证）
// 流程：按账号查找用户 → 检查账号状态 → 验证密码
// 返回通过认证的用户对象
func (uc *UserUseCase) Login(ctx context.Context, account, password string) (*model.User, error) {
	uc.log(ctx).Debugw("用户登录开始", "account", account)

	// 按用户名或邮箱查找用户
	user, err := uc.repo.FindByAccount(ctx, account)
	if err != nil {
		uc.log(ctx).Warnw("用户登录：账号不存在", "account", account)
		return nil, ErrBadCredentials
	}

	// 检查账号状态是否正常（禁用账号不允许登录）
	if user.Status != model.UserStatusActive {
		uc.log(ctx).Warnw("用户登录：账号已被禁用", "id", user.ID, "username", user.Username)
		return nil, ErrAccountDisabled
	}

	// 验证密码（bcrypt 比对）
	if !security.VerifyPassword(password, user.Password) {
		uc.log(ctx).Warnw("用户登录：密码错误", "id", user.ID, "username", user.Username)
		return nil, ErrBadCredentials
	}

	uc.log(ctx).Infow("用户登录成功", "id", user.ID, "username", user.Username)
	return user, nil
}

// FindByUUID 按 UUID 查询用户（供内部 RPC 使用）
// 直接调用 repo 层查询，若不存在返回 ErrUserNotFound
func (uc *UserUseCase) FindByUUID(ctx context.Context, uuid string) (*model.User, error) {
	uc.log(ctx).Debugw("按UUID查询用户开始", "uuid", uuid)
	user, err := uc.repo.FindByUUID(ctx, uuid)
	if err != nil {
		uc.log(ctx).Warnw("按UUID查询用户失败", "uuid", uuid, "error", err)
		return nil, ErrUserNotFound
	}
	uc.log(ctx).Debugw("按UUID查询用户成功", "uuid", uuid, "id", user.ID)
	return user, nil
}

// validateUsername 校验用户名格式
// 要求：3-32 位，仅允许字母、数字、下划线、连字符
func validateUsername(username string) error {
	if len(username) < minUsernameLength || len(username) > maxUsernameLength {
		return ErrUsernameInvalid
	}
	for _, r := range username {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-') {
			return ErrUsernameInvalid
		}
	}
	return nil
}

// validatePassword 校验密码强度
// 要求：8-64 位，必须包含大写字母、小写字母、数字
func validatePassword(password string) error {
	if len(password) < minPasswordLength {
		return ErrPasswordTooShort
	}
	if len(password) > maxPasswordLength {
		return ErrPasswordTooLong
	}
	// 检查是否包含大写、小写、数字
	var hasUpper, hasLower, hasDigit bool
	for _, r := range password {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= '0' && r <= '9':
			hasDigit = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit {
		return ErrPasswordWeak
	}
	return nil
}

// getCurrentUserID 从 context 中提取当前登录用户 ID
// 支持两种来源：
//   1. gRPC meta metadata（由上游 middleware 注入到 meta.RequestMetaData）
//   2. context value（直接注入，用于内部调用）
func getCurrentUserID(ctx context.Context) (uint64, error) {
	// 优先从 meta.RequestMetaData 中读取（gRPC 请求元数据）
	reqMeta := meta.GetRequestMetaData(ctx)
	if reqMeta != nil && reqMeta.Auth.UserID > 0 {
		return reqMeta.Auth.UserID, nil
	}
	// 其次从 context value 读取（直接注入场景）
	if userID, ok := ctx.Value("user_id").(uint64); ok && userID > 0 {
		return userID, nil
	}
	return 0, errors.New("user not authenticated")
}

// 确保导入 strings 包被使用（保留以防零值引用）
var _ = strings.Join
