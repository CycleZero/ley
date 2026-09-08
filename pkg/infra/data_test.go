package infra

import (
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// FIX-9: 追踪属性 db.system 应取真实方言名，而非硬编码 postgresql
func TestDBSystemName(t *testing.T) {
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       "root:invalid@tcp(127.0.0.1:1)/none?timeout=1s",
		SkipInitializeWithVersion: true,
	}), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Fatalf("构造 gorm.DB 失败: %v", err)
	}
	if got := DBSystemName(db); got != EngineMysql {
		t.Errorf("DBSystemName = %q, want %q", got, EngineMysql)
	}
	if got := DBSystemName(nil); got != "unknown" {
		t.Errorf("nil db 应返回 unknown，实际 %q", got)
	}
}
