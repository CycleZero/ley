package infra

import (
	"context"
	"strconv"
	"time"

	"github.com/CycleZero/ley/pkg/log"
	"github.com/CycleZero/ley/pkg/trace"

	"github.com/minio/minio-go/v7"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	EngineMysql    = "mysql"
	EnginePostgres = "postgres"
)

type Data struct {
	DB          *gorm.DB
	logger      *zap.Logger
	RedisClient *RedisClient
	MongoDb     *mongo.Client
	MinioClient *minio.Client
	NatsMQ      *NatsMQ
}

type NewDbParams struct {
	Driver string
	Host   string
	Port   int
	User   string
	Pass   string
	DBName string
}

func NewDB(
	params NewDbParams,
) *gorm.DB {
	dsn := GetDsn(
		params.Driver,
		params.Host,
		strconv.FormatInt(int64(params.Port), 10),
		params.User,
		params.Pass,
		params.DBName,
	)
	var masterDB *gorm.DB
	var err error
	var conn gorm.Dialector
	switch params.Driver {
	case EngineMysql:
		conn = mysql.Open(dsn)
	case EnginePostgres:
		conn = postgres.Open(dsn)
	default:
		panic("不支持的数据库引擎" + params.Driver)
	}
	masterDB, err = gorm.Open(conn, &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		panic(err)
	}

	// 配置数据库连接池参数（生产级）
	sqlDB, err := masterDB.DB()
	if err != nil {
		panic(err)
	}
	// 最大打开连接数：根据 PostgreSQL 规格和预期并发量设定
	sqlDB.SetMaxOpenConns(50)
	// 最大空闲连接数：保持一定数量的预热连接，减少连接建立开销
	sqlDB.SetMaxIdleConns(10)
	// 连接最大存活时间：超时后自动重建，防止长连接被中间网络设备断开
	sqlDB.SetConnMaxLifetime(1 * time.Hour)
	// 空闲连接最大存活时间：超时后关闭，释放数据库服务端资源
	sqlDB.SetConnMaxIdleTime(30 * time.Minute)

	// 注册 GORM OTel 插件：每条 SQL 自动产生 data 层 span 与耗时指标。
	if err := masterDB.Use(trace.NewGormPlugin()); err != nil {
		panic("注册 GORM 追踪插件失败：" + err.Error())
	}

	return masterDB
}

func GetDsn(engine, host, port, user, password, dbname string) string {
	dsn := ""
	log.GetLogger().Info("初始化数据库",
		zap.String("engine", engine),
		zap.String("host", host),
		zap.String("port", port),
		zap.String("dbname", dbname),
	)
	switch engine {
	case EngineMysql:
		dsn = user + ":" + password + "@tcp(" + host + ":" + port + ")/" + dbname + "?charset=utf8&parseTime=True&loc=Local"
	case EnginePostgres:
		//host=localhost user=gorm password=gorm dbname=gorm port=9920 sslmode=disable TimeZone=Asia/Shanghai
		dsn = "host=" + host + " user=" + user + " password=" + password + " dbname=" + dbname + " port=" + port + " sslmode=disable TimeZone=Asia/Shanghai"
	default:
		panic("不支持的数据库引擎")
	}
	//gorm:gorm@tcp(localhost:9910)/gorm?charset=utf8&parseTime=True&loc=Local

	return dsn
}

// WithTransaction 从Context中获取父事务（父DB），子事务需自行调用db.Begin().无论是否成功，都会绑定context
func (d *Data) WithTransaction(c context.Context) *gorm.DB {
	if c == nil {
		return d.DB.WithContext(c)
	}
	db := GetTransaction(c)
	if db != nil {
		return db
	}
	return d.DB.WithContext(c)

}

func SetTransaction(c context.Context, tran *gorm.DB) context.Context {
	t := tran.WithContext(c)
	return context.WithValue(c, "transaction", t)
}

func GetTransaction(c context.Context) *gorm.DB {
	db, ok := c.Value("transaction").(*gorm.DB)
	if ok {
		return db
	}
	return nil
}

// DBSystemName 返回 GORM 方言名（如 "mysql"/"postgres"），供追踪属性标注真实数据库类型。
func DBSystemName(db *gorm.DB) string {
	if db != nil && db.Dialector != nil {
		return db.Dialector.Name()
	}
	return "unknown"
}
