package security

import (
	"context"
	"doing_now/be/biz/config"
	"doing_now/be/biz/db/redis"
	"doing_now/be/biz/model/dto"
	"doing_now/be/biz/model/errs"
	"doing_now/be/biz/util/interceptor"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
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

	unknownIP     = "unknown"
	redisValueOne = "1"

	msgLoginFailuresHoursFmt   = "Too many login failures, please try again after %v hours"
	msgLoginFailuresMinutesFmt = "Too many login failures, please try again after %v minutes"
	msgLoginLimitReached       = "Login limit reached, please try again later"

	logParseRespErrFmt         = "Failed to parse response body in LoginProtection: %v"
	logFailInterceptorErrFmt   = "FailInterceptor error: %v"
	logSuccessRecorderErrFmt   = "LoginProtection success recorder error: %v"
	logBlockedLevel1Fmt        = "Login protection: IP %s blocked for %v (Level 1)"
	logBlockedLevel2Fmt        = "Login protection: IP %s blocked for %v (Level 2)"
	logSetLoginBlockKeysErrFmt = "Failed to set login block keys: %v"
)

type loginProtectionSettings struct {
	failInterceptor *interceptor.Interceptor
	successRecorder *interceptor.Interceptor

	durationBlockMin  time.Duration
	durationBlockHour time.Duration
	durationFailLvl   time.Duration
}

func NewLoginProtection() app.HandlerFunc {
	conf := config.GetLoginProtectionConf()
	settings := newLoginProtectionSettings(conf)

	return func(ctx context.Context, c *app.RequestContext) {
		ip := loginProtectionClientIP(c)

		if loginProtectionAbortIfBlocked(ctx, c, ip, settings.durationBlockMin, settings.durationBlockHour) {
			return
		}

		req, ok := loginProtectionBindReq(c)
		if !ok {
			return
		}

		if loginProtectionAbortIfReachSuccessLimit(ctx, c, settings.successRecorder, req.Account) {
			return
		}

		c.Next(ctx)

		resp, ok := loginProtectionParseResp(ctx, c)
		if !ok {
			return
		}

		if resp.Success {
			loginProtectionRecordSuccess(ctx, settings.successRecorder, req.Account)
			return
		}

		if !loginProtectionShouldHandleFailure(resp) {
			return
		}

		loginProtectionHandleFailure(ctx, ip, settings.failInterceptor, settings.durationBlockMin, settings.durationBlockHour, settings.durationFailLvl)
	}
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

func loginProtectionClientIP(c *app.RequestContext) string {
	ip := c.ClientIP()
	if ip == "" {
		return unknownIP
	}
	return ip
}

func loginProtectionAbortIfBlocked(ctx context.Context, c *app.RequestContext, ip string, durationBlockMin time.Duration, durationBlockHour time.Duration) bool {
	rdb := redis.GetRedisClient()

	if n, _ := rdb.Exists(ctx, rateLimitPrefix+keyLoginBlockHour+ip).Result(); n > 0 {
		c.AbortWithStatusJSON(http.StatusForbidden, dto.CommonResp{
			Code:    int(errs.RequestBlocked.Code()),
			Message: fmt.Sprintf(msgLoginFailuresHoursFmt, durationBlockHour.Hours()),
			Success: false,
		})
		return true
	}

	if n, _ := rdb.Exists(ctx, rateLimitPrefix+keyLoginBlockMinute+ip).Result(); n > 0 {
		c.AbortWithStatusJSON(http.StatusForbidden, dto.CommonResp{
			Code:    int(errs.RequestBlocked.Code()),
			Message: fmt.Sprintf(msgLoginFailuresMinutesFmt, durationBlockMin.Minutes()),
			Success: false,
		})
		return true
	}

	return false
}

func loginProtectionBindReq(c *app.RequestContext) (dto.LoginReq, bool) {
	var req dto.LoginReq
	if err := c.BindAndValidate(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, dto.CommonResp{
			Code:    int(errs.ParamError.Code()),
			Message: err.Error(),
			Success: false,
		})
		return dto.LoginReq{}, false
	}
	return req, true
}

func loginProtectionAbortIfReachSuccessLimit(ctx context.Context, c *app.RequestContext, recorder *interceptor.Interceptor, account string) bool {
	if recorder.ReachLimit(ctx, keyLoginSuccess+account) {
		c.AbortWithStatusJSON(http.StatusForbidden, dto.CommonResp{
			Code:    int(errs.LoginReachLimit.Code()),
			Message: msgLoginLimitReached,
			Success: false,
		})
		return true
	}
	return false
}

func loginProtectionParseResp(ctx context.Context, c *app.RequestContext) (dto.CommonResp, bool) {
	respBody := c.Response.Body()
	var resp dto.CommonResp
	if err := json.Unmarshal(respBody, &resp); err != nil {
		hlog.CtxErrorf(ctx, logParseRespErrFmt, err)
		return dto.CommonResp{}, false
	}
	return resp, true
}

func loginProtectionRecordSuccess(ctx context.Context, recorder *interceptor.Interceptor, account string) {
	_, err := recorder.Allow(ctx, keyLoginSuccess+account)
	if err != nil {
		hlog.CtxErrorf(ctx, logSuccessRecorderErrFmt, err)
	}
}

func loginProtectionShouldHandleFailure(resp dto.CommonResp) bool {
	switch int32(resp.Code) {
	case errs.UserNotExist.Code(), errs.PasswordIncorrect.Code():
		return true
	default:
		return false
	}
}

func loginProtectionHandleFailure(
	ctx context.Context,
	ip string,
	failInterceptor *interceptor.Interceptor,
	durationBlockMin time.Duration,
	durationBlockHour time.Duration,
	durationFailLvl time.Duration,
) {
	rdb := redis.GetRedisClient()

	allowed, err := failInterceptor.Allow(ctx, keyLoginFail+ip)
	if err != nil {
		hlog.CtxErrorf(ctx, logFailInterceptorErrFmt, err)
		return
	}

	if allowed {
		return
	}

	lvlExists, _ := rdb.Exists(ctx, keyLoginFailLvl+ip).Result()
	if lvlExists > 0 {
		rdb.Set(ctx, rateLimitPrefix+keyLoginBlockHour+ip, redisValueOne, durationBlockHour)
		hlog.CtxInfof(ctx, logBlockedLevel2Fmt, ip, durationBlockHour)
		return
	}

	pipe := rdb.Pipeline()
	pipe.Set(ctx, rateLimitPrefix+keyLoginBlockMinute+ip, redisValueOne, durationBlockMin)
	pipe.Set(ctx, keyLoginFailLvl+ip, redisValueOne, durationFailLvl)
	_, err = pipe.Exec(ctx)
	if err != nil {
		hlog.CtxErrorf(ctx, logSetLoginBlockKeysErrFmt, err)
	}
	hlog.CtxInfof(ctx, logBlockedLevel1Fmt, ip, durationBlockMin)
}
