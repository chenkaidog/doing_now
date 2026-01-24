package ratelimit

import (
	"context"
	"doing_now/be/biz/config"
	db_redis "doing_now/be/biz/db/redis"
	"os"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/bytedance/mockey"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/hertz-contrib/sessions"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

type MockSession struct {
	sessions.Session
	IDVal string
}

func (m *MockSession) ID() string {
	return m.IDVal
}

func TestNewWithConfig(t *testing.T) {
	// 1. Setup Config
	configFile := "test_config.yaml"
	configContent := `
rate_limit:
  - path: "/limited"
    window_seconds: 1
    limit: 2
  - path: "/unlimited"
    window_seconds: 1
    limit: 100
  - path: "/session_limited"
    window_seconds: 1
    limit: 2
    has_session: true
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(configFile)

	// Initialize config
	config.Init(configFile)

	// 2. Setup Redis Mock
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// 3. Test Middleware
	mockey.PatchConvey("TestNewWithConfig", t, func() {
		mockey.Mock(db_redis.GetRedisClient).Return(rdb).Build()

		mw := New()
		ctx := context.Background()

		t.Run("Path with Limit", func(t *testing.T) {
			mr.FlushAll()
			c := app.NewContext(0)
			c.Request.SetRequestURI("/limited")

			// Request 1: Allowed
			mw(ctx, c)
			assert.False(t, c.IsAborted())

			// Request 2: Allowed
			c.Reset()
			c.Request.SetRequestURI("/limited")
			mw(ctx, c)
			assert.False(t, c.IsAborted())

			// Request 3: Blocked
			c.Reset()
			c.Request.SetRequestURI("/limited")
			mw(ctx, c)
			assert.True(t, c.IsAborted())
			assert.Equal(t, consts.StatusTooManyRequests, c.Response.StatusCode())
		})

		t.Run("Path without Config (Default Limit)", func(t *testing.T) {
			mr.FlushAll()
			c := app.NewContext(0)
			c.Request.SetRequestURI("/other")

			// Request 1: Allowed
			mw(ctx, c)
			assert.False(t, c.IsAborted())

			// Request 2: Allowed
			c.Reset()
			c.Request.SetRequestURI("/other")
			mw(ctx, c)
			assert.False(t, c.IsAborted())

			// Request 3: Blocked (Default Limit is 2)
			c.Reset()
			c.Request.SetRequestURI("/other")
			mw(ctx, c)
			assert.True(t, c.IsAborted())
			assert.Equal(t, consts.StatusTooManyRequests, c.Response.StatusCode())
		})

		t.Run("Path with High Limit", func(t *testing.T) {
			mr.FlushAll()
			c := app.NewContext(0)
			c.Request.SetRequestURI("/unlimited")

			// Should allow many requests
			for i := 0; i < 10; i++ {
				c.Reset()
				c.Request.SetRequestURI("/unlimited")
				mw(ctx, c)
				assert.False(t, c.IsAborted())
			}
		})

		t.Run("Session Limit", func(t *testing.T) {
			mr.FlushAll()
			c := app.NewContext(0)
			c.Request.SetRequestURI("/session_limited")

			// Mock sessions.Default with dynamic ID
			currentSessionID := "user_session_1"
			mockey.Mock(sessions.Default).To(func(c *app.RequestContext) sessions.Session {
				return &MockSession{IDVal: currentSessionID}
			}).Build()

			// Request 1: Allowed
			mw(ctx, c)
			assert.False(t, c.IsAborted())

			// Request 2: Allowed
			c.Reset()
			c.Request.SetRequestURI("/session_limited")
			mw(ctx, c)
			assert.False(t, c.IsAborted())

			// Request 3: Blocked
			c.Reset()
			c.Request.SetRequestURI("/session_limited")
			mw(ctx, c)
			assert.True(t, c.IsAborted())
			assert.Equal(t, consts.StatusTooManyRequests, c.Response.StatusCode())

			// Change session ID
			currentSessionID = "user_session_2"

			// Different session should be allowed
			c.Reset()
			c.Request.SetRequestURI("/session_limited")
			mw(ctx, c)
			assert.False(t, c.IsAborted())
		})
	})
}
