package ratelimiter

import (
	"context"
	_ "embed"
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
//
//go:embed sliding_window.lua
var slidingWindowScript string
var luaScript = redis.NewScript(slidingWindowScript)

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
