package ratelimiter

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type LimiterSlidingWindow struct {
	DB     *redis.Client
	Window time.Duration
}

func NewLimiterSlidingWindow(db *redis.Client, window time.Duration) *LimiterSlidingWindow {
	return &LimiterSlidingWindow{DB: db, Window: window}
}

// I use redis server time as a score so that if there is clock mismatch
// between several instance it dose not back and forth the requests in redis
var luaScript = redis.NewScript(`
local key = KEYS[1]
local window = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])
local member = ARGV[3]
local allowed

-- Get Redis server time
local redis_time = redis.call('TIME')
local now = tonumber(redis_time[1]) + tonumber(redis_time[2])/1000000

-- Remove old requests outside the window
redis.call('ZREMRANGEBYSCORE', key, 0, now - window)

-- Count current requests
local count = redis.call('ZCARD', key)

-- Add new request only if under limit
if count < limit then
    redis.call('ZADD', key, now, member)
    allowed = 1
else
    allowed = 0
end

-- Set expiration to 1 hour (3600 seconds) from now
redis.call('EXPIRE', key, 3600)

return allowed
`)

func (l *LimiterSlidingWindow) RateLimit(userId string, limit int) bool {
	if l == nil || l.DB == nil {
		log.Println("RateLimiter not initialized or Redis client is nil")
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	key := fmt.Sprintf("ratelimiter:user:%s", userId)
	requestId := uuid.New().String()

	res, err := luaScript.Run(ctx, l.DB, []string{key}, l.Window.Seconds(), limit, requestId).Result()
	if err != nil {
		log.Printf("Error running Lua script for user %s: %v\n", userId, err)
		return false
	}

	allowed, ok := res.(int64)
	return ok && allowed == 1
}
