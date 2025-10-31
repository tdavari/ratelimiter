package ratelimiter

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func getenvInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if v, err := strconv.Atoi(val); err == nil {
			return v
		}
	}
	return fallback
}

// setupRedis initializes Redis client for tests
func setupRedis() *redis.Client {
	err := godotenv.Load("../../.env")
	if err != nil {
		panic("Error loading .env file")
	}

	addr := os.Getenv("REDIS_ADDR")
	db := getenvInt("REDIS_TEST_DB", 1) // use test DB
	poolSize := getenvInt("REDIS_POOLSIZE", 100)

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		DB:       db,
		PoolSize: poolSize,
	})
	// Dealing with lazy connection for redis driver
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		panic(fmt.Sprintf("Failed to connect to Redis at %s: %v", addr, err))
	}

	// Clean slate for tests
	if err := rdb.FlushDB(context.Background()).Err(); err != nil {
		panic(fmt.Sprintf("Failed to flush Redis: %v", err))
	}
	return rdb
}

func TestRateLimiter_SlidingWindow(t *testing.T) {
	rdb := setupRedis()
	limiter := NewLimiterSlidingWindow(rdb, 1*time.Second)

	userID := "user123"
	limit := 5

	// 5 requests allowed
	for i := 0; i < limit; i++ {
		if !limiter.RateLimit(userID, limit) {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// wait 2.1 seconds so the window slides
	time.Sleep(2100 * time.Millisecond)

	// request should be allowed again
	for i := 0; i < limit-1; i++ {
		if !limiter.RateLimit(userID, limit) {
			t.Errorf("request after sliding window should be allowed")
		}
	}

}

func TestRateLimiter_ConcurrentMultipleUsers(t *testing.T) {
	rdb := setupRedis()
	limiter := NewLimiterSlidingWindow(rdb, time.Second)

	users := map[string]int{
		"userA": 5,
		"userB": 10,
		"userC": 3,
	}

	var wg sync.WaitGroup
	allowedCounts := make(map[string]int)
	mutex := sync.Mutex{}

	// spawn goroutines for each user
	for userID, limit := range users {
		for i := 0; i < limit*2; i++ { // attempt twice the limit
			wg.Add(1)
			go func(uid string, lim int) {
				defer wg.Done()
				if limiter.RateLimit(uid, lim) {
					mutex.Lock()
					allowedCounts[uid]++
					mutex.Unlock()
				}
			}(userID, limit)
		}
	}

	wg.Wait()

	// check that each user was allowed only up to their limit
	for uid, limit := range users {
		if allowedCounts[uid] != limit {
			t.Errorf("expected %d allowed for %s, got %d", limit, uid, allowedCounts[uid])
		} else {
			t.Logf("user %s: allowed %d requests (limit %d)", uid, allowedCounts[uid], limit)
		}
	}
}

func BenchmarkRateLimiter_ManyUsers(b *testing.B) {
	rdb := setupRedis()
	limiter := NewLimiterSlidingWindow(rdb, time.Second)

	users := []struct {
		id    string
		limit int
	}{
		{"userA", 5},
		{"userB", 20},
		{"userC", 50},
		{"userD", 100},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			u := users[rand.Intn(len(users))]
			limiter.RateLimit(u.id, u.limit)
		}
	})
}
