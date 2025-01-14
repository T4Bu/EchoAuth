package middlewares

import (
	"EchoAuth/services"
	"EchoAuth/utils/metrics"
	"fmt"
	"net/http"
	"strings"
)

func RateLimitMiddleware(limiter services.RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get IP address
			ip := getIP(r)

			// Create rate limit key based on IP and path
			key := fmt.Sprintf("rate_limit:%s:%s", ip, r.URL.Path)

			// Check rate limit
			allowed, err := limiter.Allow(key)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			if !allowed {
				// Record rate limit hit
				metrics.RecordRateLimitHit()
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getIP extracts the client IP address from the request
func getIP(r *http.Request) string {
	// Check X-Forwarded-For header
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return strings.Split(forwarded, ",")[0]
	}

	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	return strings.Split(r.RemoteAddr, ":")[0]
}
