package security

import (
	"context"
	"testing"
	"time"

	"doing_now/be/biz/config"
	db_redis "doing_now/be/biz/db/redis"
	"doing_now/be/biz/model/errs"

	"github.com/alicebob/miniredis/v2"
	"github.com/bytedance/mockey"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestRegisterProtection(t *testing.T) {
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	mockey.PatchConvey("register protection", t, func() {
		mockey.Mock(db_redis.GetRedisClient).Return(rdb).Build()
		mockey.Mock(config.GetRegisterProtectionConf).Return(config.RegisterProtectionConf{BlockMinutes: 2}).Build()

		ctx := context.Background()
		ip := "127.0.0.1"

		bizErr := CheckRegisterAllowed(ctx, ip)
		assert.Nil(t, bizErr)

		MarkRegisterSuccess(ctx, ip)
		exists, err := rdb.Exists(ctx, rateLimitPrefix+keyRegisterBlock+ip).Result()
		assert.NoError(t, err)
		assert.Equal(t, int64(1), exists)

		bizErr = CheckRegisterAllowed(ctx, ip)
		assert.NotNil(t, bizErr)
		assert.Equal(t, errs.RequestBlocked.Code(), bizErr.Code())
		assert.Contains(t, bizErr.Msg(), "2 minutes")
	})
}

func TestRegisterProtection_DefaultBlockMinutes(t *testing.T) {
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	mockey.PatchConvey("default block minutes", t, func() {
		mockey.Mock(db_redis.GetRedisClient).Return(rdb).Build()
		mockey.Mock(config.GetRegisterProtectionConf).Return(config.RegisterProtectionConf{BlockMinutes: 0}).Build()

		ctx := context.Background()
		ip := "127.0.0.2"
		err := rdb.Set(ctx, rateLimitPrefix+keyRegisterBlock+ip, redisValueOne, time.Minute).Err()
		assert.NoError(t, err)

		bizErr := CheckRegisterAllowed(ctx, ip)
		assert.NotNil(t, bizErr)
		assert.Contains(t, bizErr.Msg(), "10 minutes")
	})
}
