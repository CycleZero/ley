package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ley/app/user/internal/biz"
	"ley/app/user/internal/model"
	"ley/pkg/cache"
	"ley/pkg/util"

	"go.opentelemetry.io/otel/attribute"
	"gorm.io/gorm"
)

// 用户缓存常量
const (
	cacheKeyUser       = "user:detail:%s"  // 用户详情缓存 key 模板（按 UUID）：user:detail:<uuid>
	cacheKeyUserID     = "user:id:%d"      // 用户 ID 索引缓存 key 模板（按 ID）：user:id:<id> → 存 UUID
	cacheTTLUserDetail = 30 * time.Minute  // 用户详情缓存 TTL：30 分钟（完整用户对象）
	cacheTTLUserStale  = 5 * time.Minute   // 空值标记缓存 TTL：5 分钟（防缓存穿透，值 "null" 表示用户不存在）
)

// 业务错误别名，指向 biz 层统一定义的哨兵错误
// 避免 data 层直接依赖 biz 包的细节
var (
	ErrUserNotFound  = biz.ErrUserNotFound  // 用户不存在
	ErrUserDuplicate = biz.ErrUserDuplicate // 用户名或邮箱重复
)

// userRepo UserRepo 接口实现
// 持有 Data 聚合对象，通过 Data 访问 DB、缓存、日志、链路追踪
type userRepo struct {
	data *Data
}

// 编译期接口实现检查：确保 *userRepo 实现了 biz.UserRepo 接口
var _ biz.UserRepo = (*userRepo)(nil)

// Create 创建用户
// 将用户对象写入 PostgreSQL，若发生唯一约束冲突返回 ErrUserDuplicate
// 成功时记录用户创建日志（Info 级别）
func (r *userRepo) Create(ctx context.Context, user *model.User) error {
	// 创建链路追踪 Span，用于追踪本次数据库写入操作
	ctx, span := r.data.StartSpan(ctx, "UserRepo.Create")
	defer span.End()

	// 将用户关键属性注入 Span，便于排查问题
	span.SetAttributes(
		attribute.String("user.username", user.Username),
		attribute.String("user.email", user.Email),
	)

	r.data.Log(ctx).Debugw("开始创建用户",
		"username", user.Username,
		"email", user.Email,
		"nickname", user.Nickname,
	)

	// 通过 GORM WithContext 执行 INSERT（链路追踪信息随 context 传播到 DB）
	err := r.data.db.WithContext(ctx).Create(user).Error
	if err != nil {
		// 检查是否为唯一约束冲突（PostgreSQL 错误码 23505）
		if util.IsUniqueViolation(err) {
			r.data.Log(ctx).Warnw("创建用户重复",
				"username", user.Username,
				"email", user.Email,
				"error", err,
			)
			return ErrUserDuplicate
		}
		r.data.Log(ctx).Errorw("创建用户失败",
			"username", user.Username,
			"error", err,
		)
		return fmt.Errorf("create user: %w", err)
	}

	r.data.Log(ctx).Infow("用户创建成功",
		"id", user.ID,
		"uuid", user.UUID,
		"username", user.Username,
	)
	return nil
}

// Update 更新用户字段（使用 map 避免零值跳过）
// GORM Updates 默认跳过零值字段，使用 map[string]interface{} 可确保空字符串也被更新
// 更新成功后自动删除相关缓存，保证缓存一致性
func (r *userRepo) Update(ctx context.Context, user *model.User) error {
	ctx, span := r.data.StartSpan(ctx, "UserRepo.Update")
	defer span.End()

	span.SetAttributes(attribute.Int("user.id", int(user.ID)))

	r.data.Log(ctx).Debugw("开始更新用户",
		"id", user.ID,
		"uuid", user.UUID,
		"username", user.Username,
		"email", user.Email,
	)

	// 使用 map 更新字段，避免 GORM 跳过零值（如空字符串 nickname）
	result := r.data.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", user.ID).
		Updates(map[string]interface{}{
			"username": user.Username,
			"email":    user.Email,
			"nickname": user.Nickname,
			"avatar":   user.Avatar,
			"bio":      user.Bio,
			"status":   user.Status,
			"role":     user.Role,
		})

	if result.Error != nil {
		r.data.Log(ctx).Errorw("更新用户失败",
			"id", user.ID,
			"error", result.Error,
		)
		return fmt.Errorf("update user: %w", result.Error)
	}
	// 影响行数为 0 表示目标用户不存在
	if result.RowsAffected == 0 {
		r.data.Log(ctx).Warnw("更新用户：目标用户不存在", "id", user.ID)
		return ErrUserNotFound
	}

	// 删除用户相关缓存（详情缓存 + ID 索引缓存），确保后续读取获取最新数据
	r.deleteCache(ctx, user.UUID, user.ID)

	r.data.Log(ctx).Infow("用户更新成功", "id", user.ID)
	return nil
}

// Delete 软删除用户
// GORM 的 Delete 在 model.User 配置了 DeletedAt 字段时执行软删除（设置 deleted_at 时间戳）
// 删除前先查询用户获取 UUID，以便清理缓存
func (r *userRepo) Delete(ctx context.Context, id uint) error {
	ctx, span := r.data.StartSpan(ctx, "UserRepo.Delete")
	defer span.End()

	span.SetAttributes(attribute.Int("user.id", int(id)))

	r.data.Log(ctx).Debugw("开始删除用户", "id", id)

	// 删除前先查询用户，获取 UUID 用于后续缓存清理
	user, err := r.FindByID(ctx, id)
	if err != nil {
		r.data.Log(ctx).Warnw("删除用户：查询用户失败，无法删除", "id", id, "error", err)
		return err
	}

	// 执行软删除（GORM 自动设置 deleted_at 字段）
	result := r.data.db.WithContext(ctx).Where("id = ?", id).Delete(&model.User{})
	if result.Error != nil {
		r.data.Log(ctx).Errorw("删除用户失败", "id", id, "error", result.Error)
		return fmt.Errorf("delete user: %w", result.Error)
	}

	// 删除缓存，防止脏数据残留
	r.deleteCache(ctx, user.UUID, id)
	r.data.Log(ctx).Infow("用户删除成功", "id", id)
	return nil
}

// FindByID 按主键 ID 查询（Cache-Aside 模式）
// 缓存策略：
//   1. 优先查 ID→UUID 索引缓存
//   2. 命中则用 UUID 查详情缓存
//   3. 未命中则查 DB，回写两个缓存
func (r *userRepo) FindByID(ctx context.Context, id uint) (*model.User, error) {
	ctx, span := r.data.StartSpan(ctx, "UserRepo.FindByID")
	defer span.End()

	span.SetAttributes(attribute.Int("user.id", int(id)))

	r.data.Log(ctx).Debugw("按ID查询用户开始", "id", id)

	// 第一步：查 ID→UUID 索引缓存
	idCacheKey := fmt.Sprintf(cacheKeyUserID, id)
	uuidBytes, err := r.data.cache.Get(ctx, idCacheKey)
	if err == nil {
		r.data.Log(ctx).Debugw("ID索引缓存命中", "id", id, "uuid", string(uuidBytes))
		// 索引缓存命中，用 UUID 查用户详情缓存
		user, err := r.FindByUUID(ctx, string(uuidBytes))
		if err == nil {
			r.data.Log(ctx).Debugw("通过ID索引缓存查用户详情成功", "id", id, "uuid", string(uuidBytes))
			return user, nil
		}
		if !errors.Is(err, ErrUserNotFound) {
			r.data.Log(ctx).Warnw("ID索引缓存命中但UUID回查失败", "id", id, "uuid", string(uuidBytes), "error", err)
		}
	} else {
		r.data.Log(ctx).Debugw("ID索引缓存未命中", "id", id, "cacheKey", idCacheKey)
	}

	// 第二步：缓存未命中，查 DB
	var user model.User
	dbErr := r.data.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if dbErr != nil {
		if errors.Is(dbErr, gorm.ErrRecordNotFound) {
			r.data.Log(ctx).Debugw("按ID查询用户：数据库中不存在", "id", id)
			return nil, ErrUserNotFound
		}
		r.data.Log(ctx).Errorw("按ID查询用户失败", "id", id, "error", dbErr)
		return nil, fmt.Errorf("find user by id: %w", dbErr)
	}

	// 第三步：回写缓存（详情缓存 + ID→UUID 索引缓存）
	detailKey := fmt.Sprintf(cacheKeyUser, user.UUID)
	if setErr := r.data.cache.Set(ctx, detailKey, &user, cacheTTLUserDetail); setErr != nil {
		r.data.Log(ctx).Warnw("写入用户详情缓存失败", "uuid", user.UUID, "error", setErr)
	}
	if setErr := r.data.cache.Set(ctx, idCacheKey, user.UUID, cacheTTLUserDetail); setErr != nil {
		r.data.Log(ctx).Warnw("写入用户ID索引缓存失败", "id", id, "uuid", user.UUID, "error", setErr)
	}

	r.data.Log(ctx).Debugw("按ID查询用户完成（来自DB）", "id", id, "uuid", user.UUID)
	return &user, nil
}

// FindByUUID 按对外 UUID 查询（Cache-Aside 模式）
// 缓存策略：
//   1. 读详情缓存（cacheKey: user:detail:<uuid>）
//   2. 命中则直接返回
//   3. 缓存为空值标记（"null"）表示用户不存在，直接返回 ErrUserNotFound
//   4. 未命中则查 DB，回写缓存
//   5. DB 中不存在时写入空值标记（TTL 5 分钟），防缓存穿透
func (r *userRepo) FindByUUID(ctx context.Context, uuid string) (*model.User, error) {
	ctx, span := r.data.StartSpan(ctx, "UserRepo.FindByUUID")
	defer span.End()

	span.SetAttributes(attribute.String("user.uuid", uuid))
	cacheKey := fmt.Sprintf(cacheKeyUser, uuid)

	r.data.Log(ctx).Debugw("按UUID查询用户开始", "uuid", uuid, "cacheKey", cacheKey)

	// 第一步：读缓存
	user, err := r.readCache(ctx, cacheKey)
	if err == nil && user != nil {
		r.data.Log(ctx).Debugw("用户缓存命中", "uuid", uuid)
		return user, nil
	}
	// 缓存异常（非键不存在、非空值标记）记录 Warn，降级查 DB
	if err != nil && !errors.Is(err, cache.ErrKeyNotFound) && !errors.Is(err, ErrUserNotFound) {
		r.data.Log(ctx).Warnw("用户缓存读取异常，回源数据库", "uuid", uuid, "error", err)
	} else {
		r.data.Log(ctx).Debugw("用户缓存未命中", "uuid", uuid)
	}
	// 空值标记命中，直接返回用户不存在
	if errors.Is(err, ErrUserNotFound) {
		r.data.Log(ctx).Debugw("用户为空值标记，直接返回不存在", "uuid", uuid)
		return nil, ErrUserNotFound
	}

	// 第二步：缓存未命中，查 DB
	var m model.User
	dbErr := r.data.db.WithContext(ctx).Where("uuid = ?", uuid).First(&m).Error
	if dbErr != nil {
		if errors.Is(dbErr, gorm.ErrRecordNotFound) {
			// DB 中不存在，写入空值标记防缓存穿透（短 TTL）
			r.data.Log(ctx).Debugw("按UUID查询：数据库中不存在，写入空值标记", "uuid", uuid)
			if setErr := r.data.cache.Set(ctx, cacheKey, []byte("null"), cacheTTLUserStale); setErr != nil {
				r.data.Log(ctx).Warnw("写入空值标记缓存失败", "uuid", uuid, "error", setErr)
			}
			return nil, ErrUserNotFound
		}
		r.data.Log(ctx).Errorw("按UUID查询用户失败", "uuid", uuid, "error", dbErr)
		return nil, fmt.Errorf("find user by uuid: %w", dbErr)
	}

	// 第三步：回写详情缓存
	if setErr := r.data.cache.Set(ctx, cacheKey, &m, cacheTTLUserDetail); setErr != nil {
		r.data.Log(ctx).Warnw("写入用户缓存失败", "uuid", uuid, "error", setErr)
	}

	span.SetAttributes(attribute.String("user.username", m.Username))
	r.data.Log(ctx).Debugw("按UUID查询用户完成（来自DB）", "uuid", uuid, "username", m.Username)
	return &m, nil
}

// FindByUsername 按用户名查询（不走缓存）
// 用户名唯一性约束由 DB 保证，不走缓存避免用户名冲突场景
func (r *userRepo) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	ctx, span := r.data.StartSpan(ctx, "UserRepo.FindByUsername")
	defer span.End()

	span.SetAttributes(attribute.String("user.username", username))

	r.data.Log(ctx).Debugw("按用户名查询用户开始", "username", username)

	var user model.User
	err := r.data.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.data.Log(ctx).Debugw("按用户名查询：用户不存在", "username", username)
			return nil, ErrUserNotFound
		}
		r.data.Log(ctx).Errorw("按用户名查询用户失败", "username", username, "error", err)
		return nil, fmt.Errorf("find user by username: %w", err)
	}

	r.data.Log(ctx).Debugw("按用户名查询用户成功", "username", username, "id", user.ID)
	return &user, nil
}

// FindByEmail 按邮箱查询（不走缓存）
// 邮箱唯一性约束由 DB 保证，不走缓存避免邮箱冲突场景
func (r *userRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	ctx, span := r.data.StartSpan(ctx, "UserRepo.FindByEmail")
	defer span.End()

	span.SetAttributes(attribute.String("user.email", email))

	r.data.Log(ctx).Debugw("按邮箱查询用户开始", "email", email)

	var user model.User
	err := r.data.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.data.Log(ctx).Debugw("按邮箱查询：用户不存在", "email", email)
			return nil, ErrUserNotFound
		}
		r.data.Log(ctx).Errorw("按邮箱查询用户失败", "email", email, "error", err)
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	r.data.Log(ctx).Debugw("按邮箱查询用户成功", "email", email, "id", user.ID)
	return &user, nil
}

// FindByAccount 按用户名或邮箱查询（登录使用）
// 使用 OR 条件匹配，支持用户以用户名或邮箱登录
func (r *userRepo) FindByAccount(ctx context.Context, account string) (*model.User, error) {
	ctx, span := r.data.StartSpan(ctx, "UserRepo.FindByAccount")
	defer span.End()

	span.SetAttributes(attribute.String("account", account))

	r.data.Log(ctx).Debugw("按账号查询用户开始", "account", account)

	var user model.User
	err := r.data.db.WithContext(ctx).
		Where("username = ? OR email = ?", account, account).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.data.Log(ctx).Debugw("按账号查询：用户不存在", "account", account)
			return nil, ErrUserNotFound
		}
		r.data.Log(ctx).Errorw("按账号查询用户失败", "account", account, "error", err)
		return nil, fmt.Errorf("find user by account: %w", err)
	}

	r.data.Log(ctx).Debugw("按账号找到用户", "account", account, "id", user.ID)
	return &user, nil
}

// List 分页查询用户列表
// 参数校验：page 默认为 1，pageSize 受 util.MaxPageSize 上限约束
// 返回用户列表和总记录数，按创建时间倒序排列
func (r *userRepo) List(ctx context.Context, page, pageSize int) ([]*model.User, int64, error) {
	ctx, span := r.data.StartSpan(ctx, "UserRepo.List")
	defer span.End()

	span.SetAttributes(attribute.Int("page", page), attribute.Int("page_size", pageSize))

	// 分页参数兜底校验
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > util.MaxPageSize {
		pageSize = 20
	}

	r.data.Log(ctx).Debugw("分页查询用户列表开始", "page", page, "pageSize", pageSize)

	var users []*model.User
	var total int64
	query := r.data.db.WithContext(ctx)

	// 先统计总记录数
	if err := query.Model(&model.User{}).Count(&total).Error; err != nil {
		r.data.Log(ctx).Errorw("统计用户总数失败", "error", err)
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	r.data.Log(ctx).Debugw("用户总数统计完成", "total", total)

	// 计算偏移量，执行分页查询并按创建时间倒序
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&users).Error; err != nil {
		r.data.Log(ctx).Errorw("分页查询用户列表失败", "page", page, "pageSize", pageSize, "error", err)
		return nil, 0, fmt.Errorf("list users: %w", err)
	}

	r.data.Log(ctx).Debugw("分页查询用户列表完成", "page", page, "pageSize", pageSize, "count", len(users), "total", total)
	return users, total, nil
}

// UpdateStatus 更新用户状态
// 更新指定用户的 status 字段（如 激活/禁用）
// 更新成功后删除相关缓存确保一致性
func (r *userRepo) UpdateStatus(ctx context.Context, id uint, status model.UserStatus) error {
	ctx, span := r.data.StartSpan(ctx, "UserRepo.UpdateStatus")
	defer span.End()

	span.SetAttributes(attribute.Int("user.id", int(id)), attribute.Int("status", int(status)))

	r.data.Log(ctx).Debugw("更新用户状态开始", "id", id, "status", status)

	// 用 Update 单个字段更新 status
	result := r.data.db.WithContext(ctx).
		Model(&model.User{}).Where("id = ?", id).Update("status", status)

	if result.Error != nil {
		r.data.Log(ctx).Errorw("更新用户状态失败", "id", id, "status", status, "error", result.Error)
		return fmt.Errorf("update user status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		r.data.Log(ctx).Warnw("更新用户状态：目标用户不存在", "id", id)
		return ErrUserNotFound
	}

	// 查询用户获取 UUID，清理缓存
	if user, err := r.FindByID(ctx, id); err == nil {
		r.deleteCache(ctx, user.UUID, id)
	}

	r.data.Log(ctx).Infow("用户状态更新成功", "id", id, "status", status)
	return nil
}

// readCache 从缓存读取用户对象，处理空值标记
// 返回值说明：
//   - (user, nil): 缓存命中且反序列化成功
//   - (nil, ErrUserNotFound): 空值标记（值 "null"），表示确定用户不存在
//   - (nil, other err): 缓存读取失败或反序列化失败
func (r *userRepo) readCache(ctx context.Context, cacheKey string) (*model.User, error) {
	// 从 Redis 读取原始字节
	data, err := r.data.cache.Get(ctx, cacheKey)
	if err != nil {
		r.data.Log(ctx).Debugw("缓存读取失败", "cacheKey", cacheKey, "error", err)
		return nil, err
	}
	// 检查空值标记（防缓存穿透）
	if string(data) == "null" {
		r.data.Log(ctx).Debugw("缓存返回空值标记", "cacheKey", cacheKey)
		return nil, ErrUserNotFound
	}
	// JSON 反序列化为 User 对象
	var user model.User
	if err := json.Unmarshal(data, &user); err != nil {
		r.data.Log(ctx).Warnw("缓存反序列化用户对象失败", "cacheKey", cacheKey, "error", err)
		return nil, fmt.Errorf("cache deserialize user: %w", err)
	}
	r.data.Log(ctx).Debugw("缓存读取并反序列化成功", "cacheKey", cacheKey, "userID", user.ID)
	return &user, nil
}

// deleteCache 删除用户相关缓存（详情缓存 + ID 索引缓存）
// 用于 Update/Delete/UpdateStatus 后清理缓存，保证缓存一致性
// 删除失败仅记录 Warn 日志，不阻塞主流程
func (r *userRepo) deleteCache(ctx context.Context, uuid string, id uint) {
	r.data.Log(ctx).Debugw("开始删除用户缓存", "uuid", uuid, "id", id)
	keys := []string{fmt.Sprintf(cacheKeyUser, uuid), fmt.Sprintf(cacheKeyUserID, id)}
	for _, key := range keys {
		if err := r.data.cache.Delete(ctx, key); err != nil {
			r.data.Log(ctx).Warnw("删除用户缓存失败", "key", key, "error", err)
		} else {
			r.data.Log(ctx).Debugw("用户缓存删除成功", "key", key)
		}
	}
}
