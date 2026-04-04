package security

import (
	"context"
	"doing_now/be/biz/config"
	"doing_now/be/biz/db/redis"
	"doing_now/be/biz/model/errs"
	"fmt"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

const (
	keyRegisterBlock = "register_block:"

	defaultRegisterBlockMinutes = 10
	msgRegisterBlockedFmt       = "Registration is temporarily blocked. Please try again after %v minutes"
	logRegisterSetBlockErrFmt   = "Failed to set register block key: %v"
	logRegisterBlockedInfoFmt   = "Register protection: IP %s blocked for %v after successful registration"
)

func CheckRegisterAllowed(ctx context.Context, ip string) errs.Error {
	rdb := redis.GetRedisClient()
	blockMinutes := getRegisterBlockMinutes()
	if n, _ := rdb.Exists(ctx, rateLimitPrefix+keyRegisterBlock+ip).Result(); n > 0 {
		return errs.RequestBlocked.SetMsg(fmt.Sprintf(msgRegisterBlockedFmt, blockMinutes))
	}
	return nil
}

func MarkRegisterSuccess(ctx context.Context, ip string) {
	blockDuration := time.Duration(getRegisterBlockMinutes()) * time.Minute
	rdb := redis.GetRedisClient()
	err := rdb.Set(ctx, rateLimitPrefix+keyRegisterBlock+ip, redisValueOne, blockDuration).Err()
	if err != nil {
		hlog.CtxErrorf(ctx, logRegisterSetBlockErrFmt, err)
		return
	}
	hlog.CtxInfof(ctx, logRegisterBlockedInfoFmt, ip, blockDuration)
}

func getRegisterBlockMinutes() int {
	blockMinutes := config.GetRegisterProtectionConf().BlockMinutes
	if blockMinutes <= 0 {
		blockMinutes = defaultRegisterBlockMinutes
	}
	return blockMinutes
}
