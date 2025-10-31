package main

import (
	"context"
	"fmt"
	"os"
	"ratelimiter/internal/ratelimiter"
	"strconv"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

type config struct {

	// rdb struct field holds the configuration settings for our redis connection pool.
	rdb struct {
		addr     string
		db       int
		poolSize int
	}
}

// Example usage of the distributed sliding window rate limiter.
func main() {
	err := godotenv.Load(".env")
	if err != nil {
		panic("Error loading .env file")
	}

	cfg := config{}
	cfg.rdb.addr = os.Getenv("REDIS_ADDR")
	cfg.rdb.db = getenvInt("REDIS_DB", 0)
	cfg.rdb.poolSize = getenvInt("REDIS_POOL_SIZE", 100)

	// Initialize Redis connection
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.rdb.addr,
		DB:       cfg.rdb.db,
		PoolSize: cfg.rdb.poolSize, // good for high concurrency
	})
	// Deal with lazy connection in redis driver
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		panic(fmt.Sprintf("Failed to connect to Redis at %s: %v", cfg.rdb.addr, err))
	}

	// Create limiter instance (sliding window of 1 second) as it was said in task
	limiter := ratelimiter.NewLimiterSlidingWindow(rdb, time.Second)

	// Define user configurations (user -> rate limit)
	userLimits := map[string]int{
		"userA": 5,
		"userB": 10,
		"userC": 3,
	}

	fmt.Println("Starting distributed rate limiter demo...")

	// Simulate concurrent requests from multiple users
	var wg sync.WaitGroup
	for user, limit := range userLimits {
		wg.Add(1)
		go func(user string, limit int) {
			defer wg.Done()
			for i := 1; i <= 8; i++ { // try 8 requests each
				allowed := limiter.RateLimit(user, limit)
				if allowed {
					fmt.Printf("[%s] Request %d: allowed\n", user, i)
				} else {
					fmt.Printf("[%s] Request %d: rate limited\n", user, i)
				}
				time.Sleep(150 * time.Millisecond)
			}
		}(user, limit)
	}

	wg.Wait()

	fmt.Println("\nDemo finished. Check Redis keys with:")
	fmt.Println("  redis-cli ZRANGE ratelimiter:user:<id> 0 -1 WITHSCORES")
}

func getenvInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if v, err := strconv.Atoi(val); err == nil {
			return v
		}
	}
	return fallback
}
