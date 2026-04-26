package model

import (
	"time"

	"gorm.io/gorm"
)

// User table: "user".users
type User struct {
	ID        int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UUID      string         `gorm:"column:uuid;type:varchar(36);uniqueIndex;not null" json:"uuid"`
	Username  string         `gorm:"column:username;type:varchar(32);uniqueIndex:idx_users_username,where:deleted_at IS NULL;not null" json:"username"`
	Email     string         `gorm:"column:email;type:varchar(255);uniqueIndex:idx_users_email,where:deleted_at IS NULL;not null" json:"email"`
	Password  string         `gorm:"column:password;type:varchar(255);not null" json:"-"` // bcrypt hash, json:"-" won't serialize
	Nickname  string         `gorm:"column:nickname;type:varchar(64);not null;default:''" json:"nickname"`
	Avatar    string         `gorm:"column:avatar;type:varchar(512);not null;default:''" json:"avatar"`
	Bio       string         `gorm:"column:bio;type:text;not null;default:''" json:"bio"`
	Status    UserStatus     `gorm:"column:status;type:smallint;not null;default:0;index" json:"status"`
	Role      UserRole       `gorm:"column:role;type:varchar(16);not null;default:'reader'" json:"role"`
	CreatedAt time.Time      `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;type:timestamptz;index" json:"-"`
}

func (User) TableName() string {
	return "user.users"
}

type UserStatus int8

const (
	UserStatusActive   UserStatus = 0
	UserStatusDisabled UserStatus = 1
)

func (s UserStatus) String() string {
	switch s {
	case UserStatusActive:
		return "active"
	case UserStatusDisabled:
		return "disabled"
	default:
		return "unknown"
	}
}

type UserRole string

const (
	RoleReader UserRole = "reader"
	RoleAuthor UserRole = "author"
	RoleAdmin  UserRole = "admin"
)

func (r UserRole) Valid() bool {
	switch r {
	case RoleReader, RoleAuthor, RoleAdmin:
		return true
	default:
		return false
	}
}
