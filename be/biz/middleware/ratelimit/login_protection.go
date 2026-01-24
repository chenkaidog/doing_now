package ratelimit

import (
	"context"
	"doing_now/be/biz/config"
	"doing_now/be/biz/db/redis"
	"doing_now/be/biz/model/dto"
	"doing_now/be/biz/model/errs"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
)

const (
	// Key prefixes
	keyLoginBlockHour   = "login_block_h:"
	keyLoginBlockMinute = "login_block_m:"
	keyLoginFailLvl     = "login_fail_level:"
	keyLoginFail        = "login_fail:"
)

// NewLoginProtection creates a middleware that protects login interface from brute force attacks
func NewLoginProtection() app.HandlerFunc {
	conf := config.GetLoginProtectionConf()

	// Defaults
	window := conf.WindowSeconds
	if window <= 0 {
		window = 300 // 5 minutes
	}

	limit := conf.Limit
	if limit <= 0 {
		limit = 3
	}

	durationBlockMin := time.Duration(conf.BlockMinDuration) * time.Minute
	if durationBlockMin <= 0 {
		durationBlockMin = 5 * time.Minute
	}

	durationBlockHour := time.Duration(conf.BlockHourDuration) * time.Hour
	if durationBlockHour <= 0 {
		durationBlockHour = 24 * time.Hour
	}

	durationFailLvl := time.Duration(conf.LevelDuration) * time.Second
	if durationFailLvl <= 0 {
		durationFailLvl = 30 * time.Minute
	}

	// Limit is set to limit-1 because Interceptor denies when current > limit.
	// We want to trigger block on the Nth failure (current=N).
	failInterceptor := NewInterceptor(window, int64(limit-1))

	return func(ctx context.Context, c *app.RequestContext) {
		// Use Client IP as key
		ip := c.ClientIP()
		if ip == "" {
			ip = "unknown"
		}

		// 1. Pre-check: Check if blocked
		rdb := redis.GetRedisClient()

		// 先校验小时拦截策略
		if n, _ := rdb.Exists(ctx, "rate_limit:"+keyLoginBlockHour+ip).Result(); n > 0 {
			c.JSON(http.StatusForbidden, dto.CommonResp{
				Code:    int(errs.RequestBlocked.Code()),
				Message: fmt.Sprintf("Too many login failures, please try again after %v hours", durationBlockHour.Hours()),
				Success: false,
			})
			c.Abort()
			return
		}

		// 再校验分钟拦截策略
		if n, _ := rdb.Exists(ctx, "rate_limit:"+keyLoginBlockMinute+ip).Result(); n > 0 {
			c.JSON(http.StatusForbidden, dto.CommonResp{
				Code:    int(errs.RequestBlocked.Code()),
				Message: fmt.Sprintf("Too many login failures, please try again after %v minutes", durationBlockMin.Minutes()),
				Success: false,
			})
			c.Abort()
			return
		}

		// 2. Process Request
		c.Next(ctx)

		// 3. Post-check: Check for failure
		// We need to parse the response to see if it was a failure.
		// Assuming handler writes JSON response.
		respBody := c.Response.Body()
		var resp dto.CommonResp
		if err := json.Unmarshal(respBody, &resp); err != nil {
			hlog.CtxErrorf(ctx, "Failed to parse response body in LoginProtection: %v", err)
			return
		}

		if resp.Success {
			return
		}

		switch int32(resp.Code) {
		case errs.UserNotExist.Code(), errs.PasswordIncorrect.Code():
		default:
			// 不是账户问题，直接退出
			return
		}

		// Record failure using Interceptor
		// Interceptor key will be "rate_limit:login_fail:<ip>"
		allowed, err := failInterceptor.Allow(ctx, keyLoginFail+ip)
		if err != nil {
			hlog.CtxErrorf(ctx, "FailInterceptor error: %v", err)
			return
		}

		if !allowed {
			lvlExists, _ := rdb.Exists(ctx, keyLoginFailLvl+ip).Result()

			if lvlExists > 0 {
				// Level 1 -> Level 2
				rdb.Set(ctx, "rate_limit:"+keyLoginBlockHour+ip, "1", durationBlockHour)
				hlog.CtxInfof(ctx, "Login protection: IP %s blocked for %v (Level 2)", ip, durationBlockHour)
			} else {
				// Level 0 -> Trigger Level 1
				pipe := rdb.Pipeline()
				pipe.Set(ctx, "rate_limit:"+keyLoginBlockMinute+ip, "1", durationBlockMin)
				pipe.Set(ctx, keyLoginFailLvl+ip, "1", durationFailLvl)
				_, err := pipe.Exec(ctx)
				if err != nil {
					hlog.CtxErrorf(ctx, "Failed to set login block keys: %v", err)
				}
				hlog.CtxInfof(ctx, "Login protection: IP %s blocked for %v (Level 1)", ip, durationBlockMin)
			}
		}
	}
}
