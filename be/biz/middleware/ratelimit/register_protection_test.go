package ratelimit

import (
	"context"
	"doing_now/be/biz/config"
	db_redis "doing_now/be/biz/db/redis"
	"doing_now/be/biz/model/dto"
	"encoding/json"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/bytedance/mockey"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestRegisterProtection(t *testing.T) {
	// Setup Redis Mock
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer rdb.Close()

	mockey.PatchConvey("TestRegisterProtection", t, func() {
		mockey.Mock(db_redis.GetRedisClient).Return(rdb).Build()
		mockey.Mock(config.GetRegisterProtectionConf).Return(config.RegisterProtectionConf{
			BlockMinutes: 10,
		}).Build()

		mw := NewRegisterProtection()
		ctx := context.Background()
		clientIP := "127.0.0.1"

		// Helper to make a register request
		makeRegisterReq := func(ip string, success bool) *app.RequestContext {
			c := app.NewContext(0)
			c.Request.SetRequestURI("/api/v1/user/register")
			// Set Client IP
			c.Request.Header.Set("X-Forwarded-For", ip)

			if success {
				resp := dto.CommonResp{Success: true, Code: 0}
				respBytes, _ := json.Marshal(resp)
				c.Response.SetBody(respBytes)
			} else {
				resp := dto.CommonResp{Success: false, Code: 10002}
				respBytes, _ := json.Marshal(resp)
				c.Response.SetBody(respBytes)
			}

			return c
		}

		t.Run("Block Logic", func(t *testing.T) {
			mr.FlushAll()

			// 1. First Register Success
			c := makeRegisterReq(clientIP, true)
			mw(ctx, c)
			assert.False(t, c.IsAborted())

			// Verify Block Key exists
			exists, _ := rdb.Exists(ctx, "rate_limit:"+keyRegisterBlock+clientIP).Result()
			assert.Equal(t, int64(1), exists)

			// 2. Second Register (Should be Blocked)
			c = app.NewContext(0)
			c.Request.SetRequestURI("/api/v1/user/register")
			c.Request.Header.Set("X-Forwarded-For", clientIP)
			mw(ctx, c)

			assert.True(t, c.IsAborted())
			assert.Equal(t, consts.StatusForbidden, c.Response.StatusCode())
			assert.Contains(t, string(c.Response.Body()), "10 minutes")
		})

		t.Run("Fail Should Not Block", func(t *testing.T) {
			mr.FlushAll()

			// 1. Register Fail
			c := makeRegisterReq(clientIP, false)
			mw(ctx, c)
			assert.False(t, c.IsAborted())

			// Verify Block Key NOT exists
			exists, _ := rdb.Exists(ctx, "rate_limit:"+keyRegisterBlock+clientIP).Result()
			assert.Equal(t, int64(0), exists)

			// 2. Second Register (Should Pass)
			c = makeRegisterReq(clientIP, true)
			mw(ctx, c)
			assert.False(t, c.IsAborted())
		})

		t.Run("Different IP", func(t *testing.T) {
			mr.FlushAll()

			// IP A Success
			c := makeRegisterReq("1.1.1.1", true)
			mw(ctx, c)

			// IP B Should Pass
			c = makeRegisterReq("2.2.2.2", true)
			mw(ctx, c)
			assert.False(t, c.IsAborted())
		})
	})
}
