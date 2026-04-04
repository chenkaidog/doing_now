package security

import (
	"context"
	"sync"
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

func resetLoginSettingsForTest() {
	loginSettingsOnce = sync.Once{}
	loginSettings = loginProtectionSettings{}
}

func TestLoginProtection_CheckAndSuccessLimit(t *testing.T) {
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	mockey.PatchConvey("check and success limit", t, func() {
		mockey.Mock(db_redis.GetRedisClient).Return(rdb).Build()
		mockey.Mock(config.GetLoginProtectionConf).Return(config.LoginProtectionConf{
			WindowSeconds:        60,
			Limit:                2,
			BlockMinDuration:     5,
			BlockHourDuration:    24,
			LevelDuration:        1800,
			SuccessWindowSeconds: 60,
			SuccessLimit:         2,
		}).Build()
		resetLoginSettingsForTest()

		ctx := context.Background()
		ip := "127.0.0.1"
		account := "acc1"

		bizErr := CheckLoginAllowed(ctx, ip, account)
		assert.Nil(t, bizErr)

		RecordLoginSuccess(ctx, account)
		bizErr = CheckLoginAllowed(ctx, ip, account)
		assert.NotNil(t, bizErr)
		assert.Equal(t, errs.LoginReachLimit.Code(), bizErr.Code())
	})
}

func TestLoginProtection_FailureBlockFlow(t *testing.T) {
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	mockey.PatchConvey("failure block flow", t, func() {
		mockey.Mock(db_redis.GetRedisClient).Return(rdb).Build()
		mockey.Mock(config.GetLoginProtectionConf).Return(config.LoginProtectionConf{
			WindowSeconds:        60,
			Limit:                2,
			BlockMinDuration:     5,
			BlockHourDuration:    24,
			LevelDuration:        1800,
			SuccessWindowSeconds: 60,
			SuccessLimit:         10,
		}).Build()
		resetLoginSettingsForTest()

		ctx := context.Background()
		ip := "127.0.0.2"
		account := "acc2"

		HandleLoginFailure(ctx, ip)
		exists, err := rdb.Exists(ctx, rateLimitPrefix+keyLoginBlockMinute+ip).Result()
		assert.NoError(t, err)
		assert.Equal(t, int64(0), exists)

		HandleLoginFailure(ctx, ip)
		exists, err = rdb.Exists(ctx, rateLimitPrefix+keyLoginBlockMinute+ip).Result()
		assert.NoError(t, err)
		assert.Equal(t, int64(1), exists)

		bizErr := CheckLoginAllowed(ctx, ip, account)
		assert.NotNil(t, bizErr)
		assert.Equal(t, errs.RequestBlocked.Code(), bizErr.Code())
		assert.Contains(t, bizErr.Msg(), "5 minutes")

		err = rdb.Del(ctx, rateLimitPrefix+keyLoginBlockMinute+ip).Err()
		assert.NoError(t, err)

		HandleLoginFailure(ctx, ip)
		HandleLoginFailure(ctx, ip)
		exists, err = rdb.Exists(ctx, rateLimitPrefix+keyLoginBlockHour+ip).Result()
		assert.NoError(t, err)
		assert.Equal(t, int64(1), exists)
	})
}

func TestLoginProtection_HelperFunctions(t *testing.T) {
	assert.True(t, ShouldHandleLoginFailure(errs.UserNotExist))
	assert.True(t, ShouldHandleLoginFailure(errs.PasswordIncorrect))
	assert.False(t, ShouldHandleLoginFailure(errs.ServerError))
	assert.False(t, ShouldHandleLoginFailure(nil))
}

func TestLoginProtection_DefaultDurationMessage(t *testing.T) {
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	mockey.PatchConvey("default duration message", t, func() {
		mockey.Mock(db_redis.GetRedisClient).Return(rdb).Build()
		mockey.Mock(config.GetLoginProtectionConf).Return(config.LoginProtectionConf{
			WindowSeconds:        0,
			Limit:                0,
			BlockMinDuration:     0,
			BlockHourDuration:    0,
			LevelDuration:        0,
			SuccessWindowSeconds: 0,
			SuccessLimit:         0,
		}).Build()
		resetLoginSettingsForTest()

		ctx := context.Background()
		ip := "127.0.0.3"
		account := "acc3"
		err := rdb.Set(ctx, rateLimitPrefix+keyLoginBlockMinute+ip, redisValueOne, time.Minute).Err()
		assert.NoError(t, err)

		bizErr := CheckLoginAllowed(ctx, ip, account)
		assert.NotNil(t, bizErr)
		assert.Contains(t, bizErr.Msg(), "5 minutes")
	})
}
