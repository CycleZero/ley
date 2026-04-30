package model

import "gorm.io/gorm"

// User table: "user".users
type User struct {
	gorm.Model
	UUID     string   `gorm:"column:uuid;type:varchar(36);uniqueIndex;not null" json:"uuid"`
	Username string   `gorm:"column:username;type:varchar(32);uniqueIndex:idx_users_username,where:deleted_at IS NULL;not null" json:"username"`
	Email    string   `gorm:"column:email;type:varchar(255);uniqueIndex:idx_users_email,where:deleted_at IS NULL;not null" json:"email"`
	Password string   `gorm:"column:password;type:varchar(255);not null" json:"-"`
	Nickname string   `gorm:"column:nickname;type:varchar(64);not null;default:''" json:"nickname"`
	Avatar   string   `gorm:"column:avatar;type:varchar(512);not null;default:''" json:"avatar"`
	Bio      string   `gorm:"column:bio;type:text;not null;default:''" json:"bio"`
	Status   UserStatus `gorm:"column:status;type:smallint;not null;default:0;index" json:"status"`
	Role     UserRole `gorm:"column:role;type:varchar(16);not null;default:'reader'" json:"role"`
}

func (User) TableName() string {
	return "user.users"
}

type UserStatus int8

const (
	UserStatusActive   UserStatus = 0
	UserStatusDisabled UserStatus = 1
)

type UserRole string

const (
	RoleReader UserRole = "reader"
	RoleAuthor UserRole = "author"
	RoleAdmin  UserRole = "admin"
)
