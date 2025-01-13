package middlewares

import (
	"auth/services"
	"auth/utils/response"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	limiter services.RateLimiter
}

func NewRateLimiter(redisClient *redis.Client) *RateLimiter {
	config := services.RateLimiterConfig{
		MaxAttempts: 100,
		Window:      time.Minute,
	}
	return &RateLimiter{
		limiter: services.NewRateLimiter(redisClient, config),
	}
}

func (rl *RateLimiter) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		key := "rate_limit:" + ip

		allowed, err := rl.limiter.Allow(key)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		if !allowed {
			response.JSONError(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
