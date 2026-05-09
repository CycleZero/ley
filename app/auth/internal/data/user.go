package data

import (
	"gorm.io/gorm"
)

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

func (UserPO) TableName() string {
	return "user.users"
}
