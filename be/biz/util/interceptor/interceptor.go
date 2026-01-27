package interceptor

import (
	"context"
	"doing_now/be/biz/db/redis"
	"time"
)

// luaScript ensures atomicity of INCR + EXPIRE and provides self-healing for keys without TTL.
// KEYS[1]: The rate limit key
// ARGV[1]: Window duration in seconds
// ARGV[2]: Max limit count
const luaScript = `
local key = KEYS[1]
local window = ARGV[1]
local limit = tonumber(ARGV[2])

local current = redis.call("INCR", key)

if current == 1 then
    redis.call("EXPIRE", key, window)
else
    if redis.call("TTL", key) == -1 then
        redis.call("EXPIRE", key, window)
    end
end

if current > limit then
    return 0 -- Denied
end
return 1 -- Allowed
`

type Interceptor struct {
	window time.Duration
	limit  int64
}

func NewInterceptor(windowSeconds int, limit int64) *Interceptor {
	return &Interceptor{
		window: time.Duration(windowSeconds) * time.Second,
		limit:  limit,
	}
}

func (i *Interceptor) Allow(ctx context.Context, key string) (bool, error) {
	redisKey := "rate_limit:" + key

	// Execute Lua script
	// Pass window in seconds
	result, err := redis.GetRedisClient().
		Eval(ctx, luaScript, []string{redisKey}, int(i.window.Seconds()), i.limit).Result()
	if err != nil {
		return false, err
	}

	// Result is 1 (Allowed) or 0 (Denied)
	return result.(int64) == 1, nil
}
