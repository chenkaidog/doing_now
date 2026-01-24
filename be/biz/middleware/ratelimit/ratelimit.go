package ratelimit

import (
	"context"
	"doing_now/be/biz/config"
	"doing_now/be/biz/model/dto"
	"doing_now/be/biz/model/errs"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/hertz-contrib/sessions"
)

type rule struct {
	interceptor *Interceptor
	hasSession  bool
}

// NewWithConfig creates a rate limit middleware that uses configuration from config.GetRateLimitConf()
func New() app.HandlerFunc {
	confList := config.GetRateLimitConf()
	rules := make(map[string]*rule)

	for _, conf := range confList {
		if conf.Path != "" && conf.WindowSeconds > 0 && conf.Limit > 0 {
			rules[conf.Path] = &rule{
				interceptor: NewInterceptor(conf.WindowSeconds, conf.Limit),
				hasSession:  conf.HasSession,
			}
		}
	}

	// Default rule: window=1, limit=2, has_session=false
	defaultRule := &rule{
		interceptor: NewInterceptor(1, 2),
		hasSession:  false,
	}

	return func(ctx context.Context, c *app.RequestContext) {
		path := string(c.Request.URI().Path())

		// If no rate limit configured for this path, use default rule
		r, ok := rules[path]
		if !ok {
			r = defaultRule
		}

		var key string
		if r.hasSession {
			key = sessions.Default(c).ID()
		} else {
			key = c.ClientIP()
		}

		allowed, err := r.interceptor.Allow(ctx, key)
		if err != nil {
			// Fail open strategy: Log error and allow request on Redis failure
			hlog.CtxErrorf(ctx, "Rate limit error for key %s: %v", key, err)
			c.Next(ctx)
			return
		}

		if !allowed {
			c.AbortWithStatusJSON(consts.StatusTooManyRequests, dto.CommonResp{
				Success: false,
				Code:    int(errs.TooManyRequest.Code()),
				Message: errs.TooManyRequest.Msg(),
			})
			return
		}

		c.Next(ctx)
	}
}
