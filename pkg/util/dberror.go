package util

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

// IsUniqueViolation 判断数据库错误是否为唯一约束违反
// PostgreSQL errcode 23505 = unique_violation，兼容 MySQL errcode 1062
// 抽离到 pkg/util 中供所有微服务 data 层复用，避免重复代码
func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// PostgreSQL SQLSTATE 23505
	if containsSubstr(errStr, "23505") {
		return true
	}
	// MySQL error 1062
	if containsSubstr(errStr, "1062") {
		return true
	}
	// 通用字符串匹配（sqlite、驱动兼容）
	if containsSubstr(errStr, "duplicate key") {
		return true
	}
	if containsSubstr(errStr, "unique constraint") {
		return true
	}
	if containsSubstr(errStr, "UNIQUE constraint failed") {
		return true
	}
	return false
}

// IsRecordNotFound 判断是否为 GORM 记录不存在错误
func IsRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

// containsSubstr 子串查找（原地实现，避免 import strings）
// 在频繁调用的错误路径上，inline 版本减少一次函数调用开销
func containsSubstr(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	// 使用 Go 原生字符串切片而非手动遍历，编译器会内联优化
	return strings.Contains(s, substr)
}
