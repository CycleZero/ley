package data

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ley/app/auth/internal/biz"
	"ley/pkg/cache"
	"ley/pkg/util"

	"go.opentelemetry.io/otel/attribute"
	"gorm.io/gorm"
)

// =============================================================================
// UserPO — 用户持久化对象（GORM 模型）
// =============================================================================

type UserPO struct {
	gorm.Model
	Username string `gorm:"column:username;type:varchar(32);uniqueIndex:idx_users_username,where:deleted_at IS NULL;not null"`
	Email    string `gorm:"column:email;type:varchar(255);uniqueIndex:idx_users_email,where:deleted_at IS NULL;not null"`
	Password string `gorm:"column:password;type:varchar(255);not null" json:"-"`
	Avatar   string `gorm:"column:avatar;type:varchar(512);default:''"`
	Bio      string `gorm:"column:bio;type:text;default:''"`
	Status   int8   `gorm:"column:status;type:smallint;default:0"`
	Role     string `gorm:"column:role;type:varchar(16);default:'reader'"`
}

func (UserPO) TableName() string { return "user.users" }

// =============================================================================
// 缓存常量
// =============================================================================

const (
	cacheKeyUser       = "auth:user:%d"     // 用户缓存键模板: auth:user:{id}
	cacheTTLUser       = 30 * time.Minute   // 正常数据缓存 TTL
	cacheTTLUserStale  = 5 * time.Minute    // 空值标记缓存 TTL（防缓存穿透）
)

// =============================================================================
// userRepo — biz.UserRepo 接口实现
// =============================================================================

type userRepo struct {
	data *Data   // 数据层聚合（持有 DB、Cache、Tracer、Logger）
}

// 编译期接口实现检查
var _ biz.UserRepo = (*userRepo)(nil)

// =============================================================================
// Create — 创建用户
//
// GORM 通过 WithContext 传递 trace 信息到 DB。插入失败时：
//   - 唯一约束冲突 → 返回 ErrUserDuplicate（由服务层映射为 409）
//   - 其他错误 → 包装原始错误返回
// 成功时不写缓存（首次查询时会回写），避免缓存冷数据。
// =============================================================================

func (r *userRepo) Create(ctx context.Context, user *biz.User) error {
	ctx, span := r.data.StartSpan(ctx, "UserRepo.Create")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.username", user.Username),
		attribute.String("user.email", user.Email),
	)
	r.data.logger.WithContext(ctx).Debugf("[UserRepo.Create] 开始 username=%s email=%s", user.Username, user.Email)

	// 将领域实体转换为持久化对象
	po := toUserPO(user)

	// 执行 INSERT，GORM 会自动填充 PO 的 ID 和 CreatedAt
	if err := r.data.db.WithContext(ctx).Create(po).Error; err != nil {
		// 检测 PostgreSQL 唯一约束冲突（错误码 23505）
		if util.IsUniqueViolation(err) {
			r.data.logger.WithContext(ctx).Warnf("[UserRepo.Create] 唯一冲突 username=%s email=%s", user.Username, user.Email)
			return biz.ErrUserDuplicate
		}
		r.data.logger.WithContext(ctx).Errorf("[UserRepo.Create] 插入失败 username=%s err=%v", user.Username, err)
		return fmt.Errorf("create user: %w", err)
	}

	// 回填自增 ID 到业务模型
	user.ID = po.ID
	user.CreatedAt = po.CreatedAt
	user.UpdatedAt = po.UpdatedAt

	span.SetAttributes(attribute.Int("user.id", int(user.ID)))
	r.data.logger.WithContext(ctx).Infof("[UserRepo.Create] 创建成功 id=%d username=%s", user.ID, user.Username)
	return nil
}

// =============================================================================
// Update — 更新用户（使用 map 避免 GORM 零值跳过）
//
// GORM 的 struct Updates 会跳过零值字段（如空字符串 ""），
// 因此使用 map[string]interface{} 确保零值字段也能被更新。
// 更新成功后立即删除缓存，使下次查询从 DB 重新加载。
// =============================================================================

func (r *userRepo) Update(ctx context.Context, user *biz.User) error {
	ctx, span := r.data.StartSpan(ctx, "UserRepo.Update")
	defer span.End()

	span.SetAttributes(attribute.Int("user.id", int(user.ID)))
	r.data.logger.WithContext(ctx).Debugf("[UserRepo.Update] 开始 id=%d username=%s", user.ID, user.Username)

	// 使用 map 执行部分更新，避免 GORM 零值跳过问题
	result := r.data.db.WithContext(ctx).
		Model(&UserPO{}).
		Where("id = ?", user.ID).
		Updates(map[string]interface{}{
			"username": user.Username,
			"email":    user.Email,
			"password": user.Password,
			"avatar":   user.Avatar,
			"bio":      user.Bio,
			"status":   int8(user.Status),
			"role":     string(user.Role),
		})

	if result.Error != nil {
		r.data.logger.WithContext(ctx).Errorf("[UserRepo.Update] 更新失败 id=%d err=%v", user.ID, result.Error)
		return fmt.Errorf("update user: %w", result.Error)
	}

	// RowsAffected = 0 表示目标行不存在（可能已被软删除）
	if result.RowsAffected == 0 {
		r.data.logger.WithContext(ctx).Warnf("[UserRepo.Update] 目标不存在 id=%d", user.ID)
		return biz.ErrUserNotFound
	}

	// 删除缓存，下次查询回源 DB
	r.deleteCache(ctx, user.ID)

	r.data.logger.WithContext(ctx).Infof("[UserRepo.Update] 更新成功 id=%d", user.ID)
	return nil
}

// =============================================================================
// Delete — 软删除用户
//
// GORM 在模型包含 gorm.DeletedAt 字段时自动执行软删除（设置 deleted_at）。
// 删除前先查用户确认存在，删除后清理缓存。
// =============================================================================

func (r *userRepo) Delete(ctx context.Context, id uint) error {
	ctx, span := r.data.StartSpan(ctx, "UserRepo.Delete")
	defer span.End()

	span.SetAttributes(attribute.Int("user.id", int(id)))
	r.data.logger.WithContext(ctx).Debugf("[UserRepo.Delete] 开始 id=%d", id)

	// 确认用户存在（GORM 自动过滤软删除记录）
	var po UserPO
	if err := r.data.db.WithContext(ctx).Where("id = ?", id).First(&po).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			r.data.logger.WithContext(ctx).Debugf("[UserRepo.Delete] 用户不存在 id=%d", id)
			return biz.ErrUserNotFound
		}
		return fmt.Errorf("delete user: %w", err)
	}

	// 执行软删除（GORM 自动设置 deleted_at）
	if err := r.data.db.WithContext(ctx).Where("id = ?", id).Delete(&UserPO{}).Error; err != nil {
		r.data.logger.WithContext(ctx).Errorf("[UserRepo.Delete] 删除失败 id=%d err=%v", id, err)
		return fmt.Errorf("delete user: %w", err)
	}

	r.deleteCache(ctx, id)
	r.data.logger.WithContext(ctx).Infof("[UserRepo.Delete] 删除成功 id=%d", id)
	return nil
}

// =============================================================================
// FindByID — 按主键查询（Cache-Aside 模式）
//
// 缓存策略：
//  1. 读缓存 auth:user:{id} → 命中直接返回
//  2. 缓存为空值标记 "null" → 返回 ErrUserNotFound（防缓存穿透）
//  3. 缓存未命中 → 查 DB → 回写缓存
//  4. DB 也不存在 → 写入空值标记（短 TTL）
// =============================================================================

func (r *userRepo) FindByID(ctx context.Context, id uint) (*biz.User, error) {
	ctx, span := r.data.StartSpan(ctx, "UserRepo.FindByID")
	defer span.End()

	span.SetAttributes(attribute.Int("user.id", int(id)))
	r.data.logger.WithContext(ctx).Debugf("[UserRepo.FindByID] 开始 id=%d", id)

	cacheKey := fmt.Sprintf(cacheKeyUser, id)

	// 第一步：读缓存
	user, err := r.readCache(ctx, cacheKey)
	if err == nil && user != nil {
		r.data.logger.WithContext(ctx).Debugf("[UserRepo.FindByID] 缓存命中 id=%d", id)
		return user, nil
	}

	// 空值标记命中，直接返回不存在
	if err != nil && err.Error() == "null sentinel" {
		r.data.logger.WithContext(ctx).Debugf("[UserRepo.FindByID] 空值标记命中 id=%d", id)
		return nil, biz.ErrUserNotFound
	}

	// 缓存读取异常（非键不存在、非空值标记）→ 记录 Warn，降级查 DB
	if err != nil && !isCacheMiss(err) {
		r.data.logger.WithContext(ctx).Warnf("[UserRepo.FindByID] 缓存异常，回源DB id=%d err=%v", id, err)
	} else {
		r.data.logger.WithContext(ctx).Debugf("[UserRepo.FindByID] 缓存未命中 id=%d", id)
	}

	// 第二步：查 DB
	var po UserPO
	dbErr := r.data.db.WithContext(ctx).Where("id = ?", id).First(&po).Error
	if dbErr != nil {
		if dbErr == gorm.ErrRecordNotFound {
			// DB 也不存在 → 写入空值标记防缓存穿透
			r.data.logger.WithContext(ctx).Debugf("[UserRepo.FindByID] DB无记录 id=%d, 写入空值标记", id)
			r.writeNullSentinel(ctx, cacheKey)
			return nil, biz.ErrUserNotFound
		}
		r.data.logger.WithContext(ctx).Errorf("[UserRepo.FindByID] DB查询失败 id=%d err=%v", id, dbErr)
		return nil, fmt.Errorf("find user by id: %w", dbErr)
	}

	// 第三步：回写缓存
	user = toUser(&po)
	r.writeCache(ctx, cacheKey, user, cacheTTLUser)

	r.data.logger.WithContext(ctx).Debugf("[UserRepo.FindByID] DB命中 id=%d username=%s", id, user.Username)
	return user, nil
}

// =============================================================================
// FindByUsername — 按用户名查询（不走缓存）
//
// 用户名唯一性检查对实时性要求高，且查询频率低，直接查 DB。
// GORM 默认过滤软删除记录（WHERE deleted_at IS NULL）。
// =============================================================================

func (r *userRepo) FindByUsername(ctx context.Context, username string) (*biz.User, error) {
	ctx, span := r.data.StartSpan(ctx, "UserRepo.FindByUsername")
	defer span.End()

	span.SetAttributes(attribute.String("user.username", username))
	r.data.logger.WithContext(ctx).Debugf("[UserRepo.FindByUsername] 开始 username=%s", username)

	var po UserPO
	err := r.data.db.WithContext(ctx).Where("username = ?", username).First(&po).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			r.data.logger.WithContext(ctx).Debugf("[UserRepo.FindByUsername] 未找到 username=%s", username)
			return nil, biz.ErrUserNotFound
		}
		r.data.logger.WithContext(ctx).Errorf("[UserRepo.FindByUsername] DB失败 username=%s err=%v", username, err)
		return nil, fmt.Errorf("find user by username: %w", err)
	}

	r.data.logger.WithContext(ctx).Debugf("[UserRepo.FindByUsername] 命中 id=%d", po.ID)
	return toUser(&po), nil
}

// =============================================================================
// FindByEmail — 按邮箱查询（不走缓存）
//
// 与 FindByUsername 同理，邮箱唯一性检查直接查 DB。
// =============================================================================

func (r *userRepo) FindByEmail(ctx context.Context, email string) (*biz.User, error) {
	ctx, span := r.data.StartSpan(ctx, "UserRepo.FindByEmail")
	defer span.End()

	span.SetAttributes(attribute.String("user.email", email))
	r.data.logger.WithContext(ctx).Debugf("[UserRepo.FindByEmail] 开始 email=%s", email)

	var po UserPO
	err := r.data.db.WithContext(ctx).Where("email = ?", email).First(&po).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			r.data.logger.WithContext(ctx).Debugf("[UserRepo.FindByEmail] 未找到 email=%s", email)
			return nil, biz.ErrUserNotFound
		}
		r.data.logger.WithContext(ctx).Errorf("[UserRepo.FindByEmail] DB失败 email=%s err=%v", email, err)
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	r.data.logger.WithContext(ctx).Debugf("[UserRepo.FindByEmail] 命中 id=%d", po.ID)
	return toUser(&po), nil
}

// =============================================================================
// FindByAccount — 按用户名或邮箱查询（登录专用）
//
// 使用 OR 条件匹配用户名或邮箱，支持用户以任意一种方式登录。
// =============================================================================

func (r *userRepo) FindByAccount(ctx context.Context, account string) (*biz.User, error) {
	ctx, span := r.data.StartSpan(ctx, "UserRepo.FindByAccount")
	defer span.End()

	span.SetAttributes(attribute.String("account", account))
	r.data.logger.WithContext(ctx).Debugf("[UserRepo.FindByAccount] 开始 account=%s", account)

	var po UserPO
	err := r.data.db.WithContext(ctx).
		Where("username = ? OR email = ?", account, account).
		First(&po).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			r.data.logger.WithContext(ctx).Debugf("[UserRepo.FindByAccount] 未找到 account=%s", account)
			return nil, biz.ErrUserNotFound
		}
		r.data.logger.WithContext(ctx).Errorf("[UserRepo.FindByAccount] DB失败 account=%s err=%v", account, err)
		return nil, fmt.Errorf("find user by account: %w", err)
	}

	r.data.logger.WithContext(ctx).Debugf("[UserRepo.FindByAccount] 命中 id=%d username=%s", po.ID, po.Username)
	return toUser(&po), nil
}

// =============================================================================
// List — 分页查询用户列表
//
// 分页参数兜底校验（page < 1 取 1，pageSize 超出 MaxPageSize 取默认值）。
// 先 COUNT 获取总数，再 OFFSET/LIMIT 查询，按创建时间倒序排列。
// =============================================================================

func (r *userRepo) List(ctx context.Context, page, pageSize int) ([]*biz.User, int64, error) {
	ctx, span := r.data.StartSpan(ctx, "UserRepo.List")
	defer span.End()

	// 分页参数兜底校验
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	span.SetAttributes(
		attribute.Int("page", page),
		attribute.Int("page_size", pageSize),
	)
	r.data.logger.WithContext(ctx).Debugf("[UserRepo.List] 开始 page=%d pageSize=%d", page, pageSize)

	query := r.data.db.WithContext(ctx).Model(&UserPO{})

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		r.data.logger.WithContext(ctx).Errorf("[UserRepo.List] COUNT失败 err=%v", err)
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	r.data.logger.WithContext(ctx).Debugf("[UserRepo.List] 总数=%d", total)

	if total == 0 {
		return nil, 0, nil
	}

	// 分页查询，按创建时间倒序
	offset := (page - 1) * pageSize
	var pos []UserPO
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&pos).Error; err != nil {
		r.data.logger.WithContext(ctx).Errorf("[UserRepo.List] 查询失败 err=%v", err)
		return nil, 0, fmt.Errorf("list users: %w", err)
	}

	users := make([]*biz.User, 0, len(pos))
	for i := range pos {
		users = append(users, toUser(&pos[i]))
	}

	r.data.logger.WithContext(ctx).Debugf("[UserRepo.List] 完成 total=%d returned=%d", total, len(users))
	return users, total, nil
}

// =============================================================================
// UpdateStatus — 更新用户状态（启用/禁用）
//
// 更新成功后删除缓存，使下次查询回源 DB。
// =============================================================================

func (r *userRepo) UpdateStatus(ctx context.Context, id uint, status biz.UserStatus) error {
	ctx, span := r.data.StartSpan(ctx, "UserRepo.UpdateStatus")
	defer span.End()

	span.SetAttributes(
		attribute.Int("user.id", int(id)),
		attribute.Int("status", int(status)),
	)
	r.data.logger.WithContext(ctx).Debugf("[UserRepo.UpdateStatus] 开始 id=%d status=%d", id, status)

	result := r.data.db.WithContext(ctx).
		Model(&UserPO{}).
		Where("id = ?", id).
		Update("status", int8(status))

	if result.Error != nil {
		r.data.logger.WithContext(ctx).Errorf("[UserRepo.UpdateStatus] 更新失败 id=%d err=%v", id, result.Error)
		return fmt.Errorf("update user status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		r.data.logger.WithContext(ctx).Warnf("[UserRepo.UpdateStatus] 目标不存在 id=%d", id)
		return biz.ErrUserNotFound
	}

	r.deleteCache(ctx, id)
	r.data.logger.WithContext(ctx).Infof("[UserRepo.UpdateStatus] 更新成功 id=%d status=%d", id, status)
	return nil
}

// =============================================================================
// 缓存辅助方法
// =============================================================================

// readCache 从 Redis 读取缓存，反序列化为 biz.User。
//
// 返回值：
//   - (user, nil): 缓存命中，反序列化成功
//   - (nil, "null sentinel" error): 空值标记（DB 中不存在该用户）
//   - (nil, other error): 缓存未命中或读取异常
func (r *userRepo) readCache(ctx context.Context, key string) (*biz.User, error) {
	if r.data.cache == nil {
		return nil, cache.ErrKeyNotFound
	}

	data, err := r.data.cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	// 空值标记："null" → DB 确认该 ID 不存在
	if string(data) == "null" {
		return nil, fmt.Errorf("null sentinel")
	}

	var user biz.User
	if err := json.Unmarshal(data, &user); err != nil {
		r.data.logger.WithContext(ctx).Warnf("缓存反序列化失败 key=%s err=%v", key, err)
		return nil, fmt.Errorf("cache deserialize: %w", err)
	}

	return &user, nil
}

// writeCache 写入缓存。若缓存不可用（nil），静默跳过。
func (r *userRepo) writeCache(ctx context.Context, key string, user *biz.User, ttl time.Duration) {
	if r.data.cache == nil {
		return
	}
	if err := r.data.cache.Set(ctx, key, user, ttl); err != nil {
		r.data.logger.WithContext(ctx).Warnf("缓存写入失败 key=%s err=%v", key, err)
	}
}

// writeNullSentinel 写入空值标记防缓存穿透（短 TTL）。
// 当 DB 确认某个 ID 不存在时写入 "null"，下次查询直接返回 NotFound。
func (r *userRepo) writeNullSentinel(ctx context.Context, key string) {
	if r.data.cache == nil {
		return
	}
	if err := r.data.cache.Set(ctx, key, []byte("null"), cacheTTLUserStale); err != nil {
		r.data.logger.WithContext(ctx).Warnf("空值标记写入失败 key=%s err=%v", key, err)
	}
}

// deleteCache 删除用户缓存。删除失败仅记录 Warn，不阻塞主流程。
func (r *userRepo) deleteCache(ctx context.Context, id uint) {
	if r.data.cache == nil {
		return
	}
	key := fmt.Sprintf(cacheKeyUser, id)
	r.data.logger.WithContext(ctx).Debugf("[UserRepo] 删除缓存 key=%s", key)
	if err := r.data.cache.Delete(ctx, key); err != nil {
		r.data.logger.WithContext(ctx).Warnf("缓存删除失败 key=%s err=%v", key, err)
	}
}

// isCacheMiss 判断错误是否为缓存未命中（键不存在）。
func isCacheMiss(err error) bool {
	return err == cache.ErrKeyNotFound
}

// =============================================================================
// 实体转换: UserPO ↔ biz.User
//
// 遵循 Clean Architecture：biz 层定义领域实体（biz.User），
// data 层定义持久化对象（UserPO），二者在 data 层互相转换。
// =============================================================================

// toUser 将持久化对象转换为领域实体。
func toUser(po *UserPO) *biz.User {
	return &biz.User{
		ID:        po.ID,
		Username:  po.Username,
		Email:     po.Email,
		Password:  po.Password,
		Avatar:    po.Avatar,
		Bio:       po.Bio,
		Status:    biz.UserStatus(po.Status),
		Role:      biz.UserRole(po.Role),
		CreatedAt: po.CreatedAt,
		UpdatedAt: po.UpdatedAt,
	}
}

// toUserPO 将领域实体转换为持久化对象。
func toUserPO(user *biz.User) *UserPO {
	return &UserPO{
		Username: user.Username,
		Email:    user.Email,
		Password: user.Password,
		Avatar:   user.Avatar,
		Bio:      user.Bio,
		Status:   int8(user.Status),
		Role:     string(user.Role),
	}
}
