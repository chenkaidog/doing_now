package ratelimit

import (
	"context"
	db_redis "doing_now/be/biz/db/redis"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/bytedance/mockey"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestInterceptor_Allow_LuaScript(t *testing.T) {
	// Setup miniredis
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	// Setup redis client
	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	ctx := context.Background()
	key := "test_ip"
	redisKey := "rate_limit:" + key

	// Mock GetRedisClient using mockey
	mockey.PatchConvey("TestInterceptor_Allow_LuaScript", t, func() {
		mockey.Mock(db_redis.GetRedisClient).Return(rdb).Build()

		t.Run("Normal Flow", func(t *testing.T) {
			mr.FlushAll()
			// Limit 2 requests per 1 second
			interceptor := NewInterceptor(1, 2)

			// 1st request: Allowed
			allowed, err := interceptor.Allow(ctx, key)
			assert.NoError(t, err)
			assert.True(t, allowed)

			// Check TTL is set (approx 1s)
			ttl := mr.TTL(redisKey)
			assert.True(t, ttl > 0 && ttl <= time.Second, "TTL should be set")

			// 2nd request: Allowed
			allowed, err = interceptor.Allow(ctx, key)
			assert.NoError(t, err)
			assert.True(t, allowed)

			// 3rd request: Denied
			allowed, err = interceptor.Allow(ctx, key)
			assert.NoError(t, err)
			assert.False(t, allowed)
		})

		t.Run("Window Expiration", func(t *testing.T) {
			mr.FlushAll()
			interceptor := NewInterceptor(1, 1)

			// 1st request: Allowed
			allowed, err := interceptor.Allow(ctx, key)
			assert.True(t, allowed)
			assert.NoError(t, err)

			// 2nd request: Denied
			allowed, err = interceptor.Allow(ctx, key)
			assert.False(t, allowed)
			assert.NoError(t, err)

			// Fast forward time by 2 seconds
			mr.FastForward(2 * time.Second)

			// Should be allowed again
			allowed, err = interceptor.Allow(ctx, key)
			assert.True(t, allowed)
			assert.NoError(t, err)
		})

		t.Run("Self Healing (Zombie Key)", func(t *testing.T) {
			mr.FlushAll()
			interceptor := NewInterceptor(10, 5)

			// Simulate a zombie key: Exists but has no TTL
			// We manually set the key to value 2 with NO expiration
			err := rdb.Set(ctx, redisKey, 2, 0).Err()
			assert.NoError(t, err)

			// Verify no TTL
			assert.Equal(t, time.Duration(0), mr.TTL(redisKey))

			// Execute Allow
			allowed, err := interceptor.Allow(ctx, key)
			assert.NoError(t, err)
			assert.True(t, allowed)

			// Verify TTL was healed (set to window)
			// Note: miniredis might return slightly less than window, but should be > 0
			ttl := mr.TTL(redisKey)
			assert.True(t, ttl > 0, "TTL should be healed")

			// Value should be incremented to 3
			val, _ := mr.Get(redisKey)
			assert.Equal(t, "3", val)
		})
	})
}
