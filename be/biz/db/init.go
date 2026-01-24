package db

import (
	"doing_now/be/biz/db/mysql"
	"doing_now/be/biz/db/redis"
)

func Init() {
	mysql.Init()
	redis.Init()
}
