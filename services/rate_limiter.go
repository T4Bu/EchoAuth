package services

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter interface {
	Allow(key string) (bool, error)
	Reset(key string) error
}

type RateLimiterConfig struct {
	MaxAttempts int
	Window      time.Duration
}

type redisRateLimiter struct {
	client      *redis.Client
	maxAttempts int
	window      time.Duration
}

func NewRateLimiter(client *redis.Client, config RateLimiterConfig) RateLimiter {
	return &redisRateLimiter{
		client:      client,
		maxAttempts: config.MaxAttempts,
		window:      config.Window,
	}
}

// Allow checks if the action is allowed for the given key
func (r *redisRateLimiter) Allow(key string) (bool, error) {
	ctx := context.Background()
	now := time.Now().Unix()
	windowStart := now - int64(r.window.Seconds())

	// Use Lua script for atomic operation
	script := `
		-- Remove old attempts
		redis.call('ZREMRANGEBYSCORE', KEYS[1], 0, ARGV[1])
		
		-- Count attempts in window
		local count = redis.call('ZCARD', KEYS[1])
		
		-- Check if under limit
		if count >= tonumber(ARGV[3]) then
			return 0
		end
		
		-- Add new attempt with unique member (timestamp:nanos)
		redis.call('ZADD', KEYS[1], ARGV[2], ARGV[2] .. ':' .. ARGV[5])
		
		-- Set expiration
		redis.call('EXPIRE', KEYS[1], ARGV[4])
		
		return 1
	`

	// Run the script
	result, err := r.client.Eval(ctx, script, []string{key},
		windowStart,             // ARGV[1] - window start
		now,                     // ARGV[2] - current timestamp
		r.maxAttempts,           // ARGV[3] - max attempts
		int(r.window.Seconds()), // ARGV[4] - window in seconds
		time.Now().UnixNano(),   // ARGV[5] - nanoseconds for unique member
	).Int()

	if err != nil {
		return false, fmt.Errorf("failed to execute rate limit check: %w", err)
	}

	return result == 1, nil
}

// Reset resets the rate limit for the given key
func (r *redisRateLimiter) Reset(key string) error {
	ctx := context.Background()
	return r.client.Del(ctx, key).Err()
}
