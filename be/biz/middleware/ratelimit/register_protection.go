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
	keyRegisterBlock = "register_block:"
)

// NewRegisterProtection creates a middleware that prevents registration after a successful one for a certain duration
func NewRegisterProtection() app.HandlerFunc {
	conf := config.GetRegisterProtectionConf()
	blockMinutes := conf.BlockMinutes
	if blockMinutes <= 0 {
		blockMinutes = 10 // default 10 minutes
	}
	blockDuration := time.Duration(blockMinutes) * time.Minute

	return func(ctx context.Context, c *app.RequestContext) {
		ip := c.ClientIP()
		if ip == "" {
			ip = "unknown"
		}

		rdb := redis.GetRedisClient()

		// 1. Pre-check: Check if blocked
		if n, _ := rdb.Exists(ctx, "rate_limit:"+keyRegisterBlock+ip).Result(); n > 0 {
			c.JSON(http.StatusForbidden, dto.CommonResp{
				Code:    int(errs.RequestBlocked.Code()),
				Message: fmt.Sprintf("Registration is temporarily blocked. Please try again after %v minutes", blockMinutes),
				Success: false,
			})
			c.Abort()
			return
		}

		// 2. Process Request
		c.Next(ctx)

		// 3. Post-check: Check if registration was successful
		respBody := c.Response.Body()
		var resp dto.CommonResp
		if err := json.Unmarshal(respBody, &resp); err != nil {
			hlog.CtxErrorf(ctx, "Failed to parse response body in RegisterProtection: %v", err)
			return
		}

		// Only block if registration was successful
		if resp.Success {
			err := rdb.Set(ctx, "rate_limit:"+keyRegisterBlock+ip, "1", blockDuration).Err()
			if err != nil {
				hlog.CtxErrorf(ctx, "Failed to set register block key: %v", err)
			} else {
				hlog.CtxInfof(ctx, "Register protection: IP %s blocked for %v after successful registration", ip, blockDuration)
			}
		}
	}
}
