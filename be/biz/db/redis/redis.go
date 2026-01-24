package redis

import (
	"doing_now/be/biz/config"
	"fmt"

	"github.com/redis/go-redis/v9"
)

var rdbClient *redis.Client

func Init() {
	rdbClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.GetRedisConf().IP, config.GetRedisConf().Port),
		Password: config.GetRedisConf().Password,
		DB:       config.GetRedisConf().DB,
	})

	rdbClient.AddHook(new(loggerHook))
}

func GetRedisClient() *redis.Client {
	if rdbClient == nil {
		panic("redis client is nil")
	}
	return rdbClient
}
