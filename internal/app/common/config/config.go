package config

import (
	"fmt"
	"github.com/go-redis/redis"
	_ "github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"gorm.io/gorm/schema"
	"strings"

	//"github.com/gofiber/fiber/v2/middleware/logger"
	"fiber-web-api/internal/app/common/mylog"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"time"
	//"fiber-web-api/internal/app/common/middleware"
	//api "fiber-web-api/internal/app/controller/sys"
)

var (
	Config         *viper.Viper
	HTTPPort       int
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	DB             *gorm.DB
	RedisConn      *redis.Client
	AuthHost       []string
	AllowCorsApi   string
	AllowedOrigins []string
	FilePath       string
)

func InitConfig() (*viper.Viper, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")              // 配置文件的类型
	viper.AddConfigPath("./manifest/config") //配置文件所在的路径
	err := viper.ReadInConfig()
	if err != nil {
		log.Panic("read config file error: %v", err)
	}
	Config = viper.GetViper()
	log.Info("config:", Config)
	//load
	LoadServer()
	LoadMySql()
	LoadRedis()
	LoadIP()
	return Config, nil
}

func LoadMySql() {
	Config.Get("database")
	Config.Get("database.username")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=%s",
		Config.Get("database.username"),
		Config.Get("database.password"),
		Config.Get("database.host"),
		Config.GetInt("database.port"),
		Config.Get("database.dbname"),
		Config.Get("database.timeout"))
	log.Info("", Config.Get("database.timeout"))
	log.Info("", dsn)
	// 设置操作数据库的日志输出到文件
	mylogger := logger.New(
		Writer{},
		logger.Config{
			SlowThreshold: time.Second, // 慢 SQL 阈值
			LogLevel:      logger.Info, // Log level
			Colorful:      true,        // 允许彩色打印
		},
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: mylogger,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})
	if err != nil {
		log.Error("connect mysql error: ", err)
	}
	DB = db
	log.Info("mysql connect success")
}

func LoadRedis() {
	RedisConn = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", Config.Get("redis.host"), Config.GetInt("redis.port")),
		Password: Config.GetString("redis.pass"),
		DB:       Config.GetInt("redis.db"),
	})
}

func LoadIP() {
	ips := Config.GetString("ip.auth_host")
	AuthHost = strings.Split(ips, ";")
	AllowCorsApi = Config.GetString("ip.allow_cors_api")
	origins := Config.GetString("ip.allowed_origins")
	AllowedOrigins = strings.Split(origins, ";")
}

func LoadServer() {
	HTTPPort = Config.GetInt("server.port")
	ReadTimeout = time.Duration(Config.GetInt("server.read_timeout")) * time.Second
	WriteTimeout = time.Duration(Config.GetInt("server.write_timeout")) * time.Second
	FilePath = Config.GetString("filePath")
}

type Writer struct {
}

func (w Writer) Printf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...) + "\n"
	fmt.Println(msg)
	mylog.LogOut(msg)
}
