package security

import (
	"context"
	db_redis "doing_now/be/biz/db/redis"
	"doing_now/be/biz/model/dto"
	"doing_now/be/biz/model/errs"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/bytedance/mockey"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestLoginProtection(t *testing.T) {
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

	mockey.PatchConvey("TestLoginProtection", t, func() {
		mockey.Mock(db_redis.GetRedisClient).Return(rdb).Build()

		mw := NewLoginProtection()
		ctx := context.Background()
		clientIP := "127.0.0.1"

		// Helper to make a login request
		makeLoginReq := func(ip string, success bool) *app.RequestContext {
			c := app.NewContext(0)
			c.Request.SetRequestURI("/api/v1/user/login")
			// Set Client IP
			c.Request.Header.Set("X-Forwarded-For", ip)

			if success {
				resp := dto.CommonResp{Success: true, Code: 0}
				respBytes, _ := json.Marshal(resp)
				c.Response.SetBody(respBytes)
			} else {
				resp := dto.CommonResp{Success: false, Code: int(errs.PasswordIncorrect.Code())}
				respBytes, _ := json.Marshal(resp)
				c.Response.SetBody(respBytes)
			}

			return c
		}

		t.Run("Normal Flow", func(t *testing.T) {
			mr.FlushAll()
			c := makeLoginReq(clientIP, true)
			mw(ctx, c)
			assert.False(t, c.IsAborted())
		})

		t.Run("Block 5m Logic", func(t *testing.T) {
			mr.FlushAll()

			// Fail 1
			c := makeLoginReq(clientIP, false)
			mw(ctx, c)
			assert.False(t, c.IsAborted())

			// Fail 2
			c = makeLoginReq(clientIP, false)
			mw(ctx, c)
			assert.False(t, c.IsAborted())

			// Fail 3 (Should Trigger Block 5m)
			c = makeLoginReq(clientIP, false)
			mw(ctx, c)
			assert.False(t, c.IsAborted()) // The request itself is not aborted, but post-check sets block

			// Verify Block Key exists
			exists, _ := rdb.Exists(ctx, "rate_limit:"+keyLoginBlockMinute+clientIP).Result()
			assert.Equal(t, int64(1), exists)

			// Verify Fail Level exists
			exists, _ = rdb.Exists(ctx, keyLoginFailLvl+clientIP).Result()
			assert.Equal(t, int64(1), exists)

			// Next Request (Should be blocked)
			c = app.NewContext(0)
			c.Request.SetRequestURI("/api/v1/user/login")
			c.Request.Header.Set("X-Forwarded-For", clientIP)
			mw(ctx, c)
			assert.True(t, c.IsAborted())
			assert.Equal(t, consts.StatusForbidden, c.Response.StatusCode())
			assert.Contains(t, string(c.Response.Body()), "5 minutes")
		})

		t.Run("Block 24h Logic", func(t *testing.T) {
			mr.FlushAll()

			// Pre-condition: User is in "Level 1" (Level key exists)
			// But Block 5m key might have expired (simulating "5 mins later")
			rdb.Set(ctx, keyLoginFailLvl+clientIP, "1", time.Hour)

			// Fail 1
			c := makeLoginReq(clientIP, false)
			mw(ctx, c)

			// Fail 2
			c = makeLoginReq(clientIP, false)
			mw(ctx, c)

			// Fail 3 (Should Trigger Block 24h)
			c = makeLoginReq(clientIP, false)
			mw(ctx, c)

			// Verify Block 24h Key exists
			exists, _ := rdb.Exists(ctx, "rate_limit:"+keyLoginBlockHour+clientIP).Result()
			assert.Equal(t, int64(1), exists)

			// Verify Level Key still exists (implementation does not remove it)
			exists, _ = rdb.Exists(ctx, keyLoginFailLvl+clientIP).Result()
			assert.Equal(t, int64(1), exists)

			// Next Request (Should be blocked 24h)
			c = app.NewContext(0)
			c.Request.SetRequestURI("/api/v1/user/login")
			c.Request.Header.Set("X-Forwarded-For", clientIP)
			mw(ctx, c)
			assert.True(t, c.IsAborted())
			assert.Contains(t, string(c.Response.Body()), "24 hours")
		})

		t.Run("System Error Should Not Count", func(t *testing.T) {
			mr.FlushAll()
			// Fail with System Error (Code 10001) 3 times
			for i := 0; i < 3; i++ {
				c := makeLoginReq(clientIP, false)
				// Override response body to be SystemError
				resp := dto.CommonResp{Success: false, Code: 10001} // ServerError
				respBytes, _ := json.Marshal(resp)
				c.Response.SetBody(respBytes)
				mw(ctx, c)
				assert.False(t, c.IsAborted())
			}

			// 4th request should still be allowed
			c := makeLoginReq(clientIP, true)
			mw(ctx, c)
			assert.False(t, c.IsAborted())

			// Verify NO Block Key
			exists, _ := rdb.Exists(ctx, "rate_limit:"+keyLoginBlockMinute+clientIP).Result()
			assert.Equal(t, int64(0), exists)
		})
	})
}
