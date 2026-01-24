package mysql

import (
	"database/sql"
	"doing_now/be/biz/config"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var gormDB *gorm.DB

func Init() {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.GetMySQLConf().Username,
		config.GetMySQLConf().Password,
		config.GetMySQLConf().IP,
		config.GetMySQLConf().Port,
		config.GetMySQLConf().DBName,
	)

	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}

	gormDB, err = gorm.Open(
		mysql.New(mysql.Config{Conn: sqlDB}),
		&gorm.Config{
			SkipDefaultTransaction: true,
			Logger: &GormLogger{
				SlowThreshold: 2 * time.Second,
				LogLevel:      logger.Info,
			},
		})
	if err != nil {
		panic(err)
	}
}

func GetDbConn() *gorm.DB {
	return gormDB
}

type DbConn struct {
	*gorm.DB
}

func NewDbConn(tx ...*gorm.DB) *DbConn {
	if len(tx) > 0 {
		return &DbConn{
			DB: tx[0],
		}
	}

	return &DbConn{
		DB: gormDB,
	}
}
