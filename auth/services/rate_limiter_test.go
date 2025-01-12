package services

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestRateLimiter(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Connect to Redis
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})
	defer client.Close()

	// Ping Redis to ensure connection
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	// Clear any existing data
	client.FlushDB(context.Background())

	// Create rate limiter with 3 attempts per 1 second window
	limiter := NewRateLimiter(client, RateLimiterConfig{
		MaxAttempts: 3,
		Window:      time.Second,
	})

	// Test key
	key := "test:rate:limit"
	defer client.Del(context.Background(), key)

	// First 3 attempts should be allowed
	for i := 0; i < 3; i++ {
		allowed, err := limiter.Allow(key)
		if err != nil {
			t.Fatalf("Failed to check rate limit: %v", err)
		}
		if !allowed {
			t.Errorf("Attempt %d should be allowed", i+1)
		}
		t.Logf("Attempt %d: allowed=%v", i+1, allowed)

		// Print current count
		count, err := client.ZCard(context.Background(), key).Result()
		if err != nil {
			t.Fatalf("Failed to get count: %v", err)
		}
		t.Logf("Current count after attempt %d: %d", i+1, count)
	}

	// 4th attempt should be blocked
	allowed, err := limiter.Allow(key)
	if err != nil {
		t.Fatalf("Failed to check rate limit: %v", err)
	}
	if allowed {
		count, _ := client.ZCard(context.Background(), key).Result()
		t.Errorf("4th attempt should be blocked (current count: %d)", count)
	}

	// Print all attempts
	attempts, err := client.ZRange(context.Background(), key, 0, -1).Result()
	if err != nil {
		t.Fatalf("Failed to get attempts: %v", err)
	}
	t.Logf("All attempts: %v", attempts)

	// Wait for window to expire
	time.Sleep(time.Second)

	// Should be allowed again
	allowed, err = limiter.Allow(key)
	if err != nil {
		t.Fatalf("Failed to check rate limit: %v", err)
	}
	if !allowed {
		t.Error("Attempt after window expiry should be allowed")
	}

	// Test reset
	if err := limiter.Reset(key); err != nil {
		t.Fatalf("Failed to reset rate limit: %v", err)
	}

	// Should be allowed after reset
	allowed, err = limiter.Allow(key)
	if err != nil {
		t.Fatalf("Failed to check rate limit: %v", err)
	}
	if !allowed {
		t.Error("Attempt after reset should be allowed")
	}
}
