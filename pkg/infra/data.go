package infra

import (
	"context"
	"strconv"

	"ley/pkg/log"

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
	return masterDB
}

func GetDsn(engine, host, port, user, password, dbname string) string {
	dsn := ""
	log.GetLogger().Info("引擎", zap.String("engine", engine))
	switch engine {
	case EngineMysql:
		dsn = user + ":" + password + "@tcp(" + host + ":" + port + ")/" + dbname + "?charset=utf8&parseTime=True&loc=Local"
		log.GetLogger().Info("生成DSN: " + dsn)
	case EnginePostgres:
		//host=localhost user=gorm password=gorm dbname=gorm port=9920 sslmode=disable TimeZone=Asia/Shanghai
		dsn = "host=" + host + " user=" + user + " password=" + password + " dbname=" + dbname + " port=" + port + " sslmode=disable TimeZone=Asia/Shanghai"
		log.GetLogger().Info("生成DSN: " + dsn)
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
