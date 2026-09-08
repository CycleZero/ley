package data

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/CycleZero/ley/pkg/testutil/datatest"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// FIX-8: GetConfig 遇 DB 故障须上抛错误，而非静默返回空配置（掩盖故障）
func TestGetConfigPropagatesDBError(t *testing.T) {
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       "root:invalid@tcp(127.0.0.1:1)/none?timeout=1s",
		SkipInitializeWithVersion: true,
	}), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Fatalf("构造 gorm.DB 失败: %v", err)
	}
	repo := &siteRepo{data: &Data{db: db, cache: datatest.NewInMemoryCache(), log: log.NewHelper(log.DefaultLogger)}}

	if _, err := repo.GetConfig(context.Background()); err == nil {
		t.Error("DB 错误应向上传播，而非静默返回空配置")
	}
}
