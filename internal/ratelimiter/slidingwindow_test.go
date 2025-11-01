package ratelimiter

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"ratelimiter/internal/config"
	"ratelimiter/internal/db"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

var rdb *redis.Client // shared by all tests in this package

// TestMain runs once before and after all tests.
func TestMain(m *testing.M) {
	fmt.Println("🔧 Setting up Redis for tests...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	rdb, err = db.NewRedisClient(cfg)
	if err != nil {
		log.Fatalf("Failed to connect Redis: %v", err)
	}

	// Optional: flush before starting all tests
	rdb.FlushDB(context.Background())

	// Run all test functions in this package
	code := m.Run()

	fmt.Println("🧹 Cleaning up Redis...")
	rdb.Close()

	os.Exit(code)
}

func TestRateLimiter_SlidingWindow(t *testing.T) {
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
