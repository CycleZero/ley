//go:build integration

package data

import (
	"os"
	"sync"
	"testing"

	"github.com/CycleZero/ley/pkg/cache"
	"github.com/CycleZero/ley/pkg/testutil/datatest"
	"github.com/go-kratos/kratos/v2/log"
	"go.opentelemetry.io/otel"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	testDB    *gorm.DB
	testCache cache.Cache
	testOnce  sync.Once
)

// testMain sets up a shared MySQL connection and cache for all tests.
func testMain(m *testing.M) {
	testOnce.Do(func() {
		dsn := os.Getenv("LEY_TEST_MYSQL_DSN")
		if dsn == "" {
			panic("LEY_TEST_MYSQL_DSN 未设置：data 集成测试需指向测试 MySQL，" +
				"例 root:pass@tcp(127.0.0.1:3306)/ley?charset=utf8mb4&parseTime=True&loc=Local")
		}
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
			DisableForeignKeyConstraintWhenMigrating: true,
			Logger:                                   logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			panic("failed to connect mysql: " + err.Error())
		}
		sqlDB, _ := db.DB()
		sqlDB.SetMaxOpenConns(10)
		testDB = db
		testCache = datatest.NewInMemoryCache()
	})
	os.Exit(m.Run())
}

func newTestData(t *testing.T) *Data {
	t.Helper()
	if testDB == nil {
		t.Fatal("testMain not called")
	}
	return &Data{
		db:     testDB,
		cache:  testCache,
		log:    log.NewHelper(log.DefaultLogger),
		tracer: otel.GetTracerProvider().Tracer("blog-service.data.test"),
	}
}

func TestMain(m *testing.M) {
	testMain(m)
}
