package infra

import (
	"strconv"
	"testing"

	"ley/pkg/cache"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// getTestDSN 生成数据库 DSN（不依赖全局 Logger，避免测试时 panic）
func getTestDSN(params TestDBParams) string {
	portStr := strconv.FormatInt(int64(params.Port), 10)
	switch params.Driver {
	case EngineMysql:
		return params.User + ":" + params.Password + "@tcp(" + params.Host + ":" + portStr + ")/" + params.DBName + "?charset=utf8&parseTime=True&loc=Local"
	case EnginePostgres:
		return "host=" + params.Host + " user=" + params.User + " password=" + params.Password + " dbname=" + params.DBName + " port=" + portStr + " sslmode=disable TimeZone=Asia/Shanghai"
	default:
		panic("不支持的数据库引擎: " + params.Driver)
	}
}

// TestDBParams MySQL 连接参数，由测试文件手动填入
type TestDBParams struct {
	Driver   string // "mysql"
	Host     string // "127.0.0.1"
	Port     int    // 3306
	User     string // "root"
	Password string // 你的密码
	DBName   string // 数据库名
}

// NewTestDB 创建测试用 MySQL 数据库连接
// 所有微服务的 data 层测试统一引用此函数，避免重复创建连接的代码。
//
// 用法：
//
//	func TestXxx(t *testing.T) {
//	    db := infra.NewTestDB(t, infra.TestDBParams{
//	        Driver:   "mysql",
//	        Host:     "127.0.0.1",
//	        Port:     3306,
//	        User:     "root",
//	        Password: "password",
//	        DBName:   "vcyuan",
//	    })
//	    defer func() {
//	        sqlDB, _ := db.DB()
//	        sqlDB.Close()
//	    }()
//	    repo := NewXxxRepo(&data.Data{DB: db})
//	    // ...
//	}
func NewTestDB(t *testing.T, params TestDBParams) *gorm.DB {
	t.Helper()
	dsn := getTestDSN(params)
	var dialector gorm.Dialector
	switch params.Driver {
	case EngineMysql:
		dialector = mysql.Open(dsn)
	case EnginePostgres:
		dialector = postgres.Open(dsn)
	default:
		panic("不支持的数据库引擎: " + params.Driver)
	}
	db, err := gorm.Open(dialector, &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("连接测试数据库失败: %v", err)
	}
	return db
}

// TestRedisParams Redis 连接参数
type TestRedisParams struct {
	Host     string // "127.0.0.1"
	Port     int    // 6379
	Password string // 密码，无密码传 ""
	DB       int    // 数据库编号，默认 0
}

// NewTestRedis 创建测试用 Redis 缓存实例
// 所有微服务的 data 层测试统一引用此函数。
//
// 用法：
//
//	func TestXxx(t *testing.T) {
//	    cache := infra.NewTestRedis(t, infra.TestRedisParams{
//	        Host: "127.0.0.1",
//	        Port: 6379,
//	    })
//	    defer cache.Close()
//	    repo := NewXxxRepo(&data.Data{Cache: cache})
//	    // ...
//	}
func NewTestRedis(t *testing.T, params TestRedisParams) cache.Cache {
	t.Helper()
	return cache.NewRedisCache(params.Host, params.Port, params.Password, params.DB)
}

// CloseDB 关闭数据库连接，通常在 defer 中调用
func CloseDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	if db == nil {
		return
	}
	sqlDB, err := db.DB()
	if err == nil && sqlDB != nil {
		sqlDB.Close()
	}
}
