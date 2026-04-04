package security

import (
	"context"
	"doing_now/be/biz/config"
	"doing_now/be/biz/db/redis"
	"doing_now/be/biz/model/errs"
	"doing_now/be/biz/util/interceptor"
	"fmt"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

const (
	keyLoginBlockHour   = "login_block_h:"
	keyLoginBlockMinute = "login_block_m:"
	keyLoginFailLvl     = "login_fail_level:"
	keyLoginFail        = "login_fail:"

	keyLoginSuccess = "login_success:"

	rateLimitPrefix = "rate_limit:"

	defaultWindowSeconds        = 300
	defaultLimit                = 3
	defaultBlockMinDuration     = 5 * time.Minute
	defaultBlockHourDuration    = 24 * time.Hour
	defaultLevelDuration        = 30 * time.Minute
	defaultSuccessWindowSeconds = 60
	defaultSuccessLimit         = 10

	msgLoginFailuresHoursFmt   = "Too many login failures, please try again after %v hours"
	msgLoginFailuresMinutesFmt = "Too many login failures, please try again after %v minutes"
	msgLoginLimitReached       = "Login limit reached, please try again later"

	logFailInterceptorErrFmt   = "FailInterceptor error: %v"
	logSuccessRecorderErrFmt   = "LoginProtection success recorder error: %v"
	logBlockedLevel1Fmt        = "Login protection: IP %s blocked for %v (Level 1)"
	logBlockedLevel2Fmt        = "Login protection: IP %s blocked for %v (Level 2)"
	logSetLoginBlockKeysErrFmt = "Failed to set login block keys: %v"

	redisValueOne = "1"
)

type loginProtectionSettings struct {
	failInterceptor *interceptor.Interceptor
	successRecorder *interceptor.Interceptor

	durationBlockMin  time.Duration
	durationBlockHour time.Duration
	durationFailLvl   time.Duration
}

var (
	loginSettingsOnce sync.Once
	loginSettings     loginProtectionSettings
)

func CheckLoginAllowed(ctx context.Context, ip string, account string) errs.Error {
	settings := getLoginProtectionSettings()
	if err := checkLoginBlocked(ctx, ip, settings.durationBlockMin, settings.durationBlockHour); err != nil {
		return err
	}
	if settings.successRecorder.ReachLimit(ctx, keyLoginSuccess+account) {
		return errs.LoginReachLimit.SetMsg(msgLoginLimitReached)
	}
	return nil
}

func RecordLoginSuccess(ctx context.Context, account string) {
	settings := getLoginProtectionSettings()
	_, err := settings.successRecorder.Allow(ctx, keyLoginSuccess+account)
	if err != nil {
		hlog.CtxErrorf(ctx, logSuccessRecorderErrFmt, err)
	}
}

func ShouldHandleLoginFailure(bizErr errs.Error) bool {
	if bizErr == nil {
		return false
	}
	switch bizErr.Code() {
	case errs.UserNotExist.Code(), errs.PasswordIncorrect.Code():
		return true
	default:
		return false
	}
}

func HandleLoginFailure(ctx context.Context, ip string) {
	settings := getLoginProtectionSettings()
	rdb := redis.GetRedisClient()
	allowed, err := settings.failInterceptor.Allow(ctx, keyLoginFail+ip)
	if err != nil {
		hlog.CtxErrorf(ctx, logFailInterceptorErrFmt, err)
		return
	}
	if allowed {
		return
	}
	lvlExists, _ := rdb.Exists(ctx, keyLoginFailLvl+ip).Result()
	if lvlExists > 0 {
		rdb.Set(ctx, rateLimitPrefix+keyLoginBlockHour+ip, redisValueOne, settings.durationBlockHour)
		hlog.CtxInfof(ctx, logBlockedLevel2Fmt, ip, settings.durationBlockHour)
		return
	}
	pipe := rdb.Pipeline()
	pipe.Set(ctx, rateLimitPrefix+keyLoginBlockMinute+ip, redisValueOne, settings.durationBlockMin)
	pipe.Set(ctx, keyLoginFailLvl+ip, redisValueOne, settings.durationFailLvl)
	_, err = pipe.Exec(ctx)
	if err != nil {
		hlog.CtxErrorf(ctx, logSetLoginBlockKeysErrFmt, err)
	}
	hlog.CtxInfof(ctx, logBlockedLevel1Fmt, ip, settings.durationBlockMin)
}

func getLoginProtectionSettings() loginProtectionSettings {
	loginSettingsOnce.Do(func() {
		loginSettings = newLoginProtectionSettings(config.GetLoginProtectionConf())
	})
	return loginSettings
}

func newLoginProtectionSettings(conf config.LoginProtectionConf) loginProtectionSettings {
	windowSeconds := conf.WindowSeconds
	if windowSeconds <= 0 {
		windowSeconds = defaultWindowSeconds
	}
	limit := conf.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	durationBlockMin := time.Duration(conf.BlockMinDuration) * time.Minute
	if durationBlockMin <= 0 {
		durationBlockMin = defaultBlockMinDuration
	}
	durationBlockHour := time.Duration(conf.BlockHourDuration) * time.Hour
	if durationBlockHour <= 0 {
		durationBlockHour = defaultBlockHourDuration
	}
	durationFailLvl := time.Duration(conf.LevelDuration) * time.Second
	if durationFailLvl <= 0 {
		durationFailLvl = defaultLevelDuration
	}
	successWindowSeconds := conf.SuccessWindowSeconds
	if successWindowSeconds <= 0 {
		successWindowSeconds = defaultSuccessWindowSeconds
	}
	successLimit := conf.SuccessLimit
	if successLimit <= 0 {
		successLimit = defaultSuccessLimit
	}
	return loginProtectionSettings{
		failInterceptor:   interceptor.NewInterceptor(windowSeconds, int64(limit-1)),
		successRecorder:   interceptor.NewInterceptor(successWindowSeconds, int64(successLimit-1)),
		durationBlockMin:  durationBlockMin,
		durationBlockHour: durationBlockHour,
		durationFailLvl:   durationFailLvl,
	}
}

func checkLoginBlocked(ctx context.Context, ip string, durationBlockMin time.Duration, durationBlockHour time.Duration) errs.Error {
	rdb := redis.GetRedisClient()
	if n, _ := rdb.Exists(ctx, rateLimitPrefix+keyLoginBlockHour+ip).Result(); n > 0 {
		return errs.RequestBlocked.SetMsg(fmt.Sprintf(msgLoginFailuresHoursFmt, durationBlockHour.Hours()))
	}
	if n, _ := rdb.Exists(ctx, rateLimitPrefix+keyLoginBlockMinute+ip).Result(); n > 0 {
		return errs.RequestBlocked.SetMsg(fmt.Sprintf(msgLoginFailuresMinutesFmt, durationBlockMin.Minutes()))
	}
	return nil
}
