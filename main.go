package main

import (
	"fmt"
	"log"
	"os"
	"ratelimiter/internal/config"
	"ratelimiter/internal/db"
	"ratelimiter/internal/ratelimiter"
	"strconv"
	"sync"
	"time"
)

// Example usage of the distributed sliding window rate limiter.
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	rdb, err := db.NewRedisClient(cfg)

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
