package middlewares

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

var (
	corsRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_cors_requests_total",
			Help: "Total number of CORS requests by origin",
		},
		[]string{"origin"},
	)
)

func init() {
	prometheus.MustRegister(corsRequestsTotal)
}

type SecurityConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
	Environment      string
}

func NewSecurityConfig() *SecurityConfig {
	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	config := &SecurityConfig{
		Environment:    env,
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{
			"Authorization",
			"Content-Type",
			"X-Request-ID",
			"X-Real-IP",
		},
		ExposedHeaders:   []string{"X-Request-ID"},
		MaxAge:           3600,
		AllowCredentials: true,
	}

	// Set allowed origins based on environment
	if env == "production" {
		config.AllowedOrigins = strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",")
		if len(config.AllowedOrigins) == 0 {
			log.Fatal().Msg("ALLOWED_ORIGINS must be set in production")
		}
	} else {
		config.AllowedOrigins = []string{"http://localhost:3000"}
	}

	return config
}

func (c *SecurityConfig) SecurityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers first
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		if c.Environment == "production" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		origin := r.Header.Get("Origin")

		// CORS headers
		if origin != "" {
			if !c.isAllowedOrigin(origin) {
				log.Warn().
					Str("origin", origin).
					Str("path", r.URL.Path).
					Str("ip", r.RemoteAddr).
					Msg("Invalid origin attempt")
				http.Error(w, "Invalid origin", http.StatusForbidden)
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(c.AllowedMethods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(c.AllowedHeaders, ", "))
			w.Header().Set("Access-Control-Expose-Headers", strings.Join(c.ExposedHeaders, ", "))

			if c.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				requestMethod := r.Header.Get("Access-Control-Request-Method")
				requestHeaders := r.Header.Get("Access-Control-Request-Headers")

				// Validate requested method
				methodAllowed := false
				for _, method := range c.AllowedMethods {
					if method == requestMethod {
						methodAllowed = true
						break
					}
				}

				if !methodAllowed {
					log.Warn().
						Str("origin", origin).
						Str("method", requestMethod).
						Msg("Invalid preflight method request")
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
					return
				}

				// Validate requested headers
				if requestHeaders != "" {
					headers := strings.Split(strings.ToLower(requestHeaders), ",")
					for _, header := range headers {
						header = strings.TrimSpace(header)
						headerAllowed := false
						for _, allowed := range c.AllowedHeaders {
							if strings.ToLower(allowed) == header {
								headerAllowed = true
								break
							}
						}
						if !headerAllowed {
							log.Warn().
								Str("origin", origin).
								Str("header", header).
								Msg("Invalid preflight header request")
							http.Error(w, "Header not allowed", http.StatusForbidden)
							return
						}
					}
				}

				log.Debug().
					Str("origin", origin).
					Str("method", requestMethod).
					Str("headers", requestHeaders).
					Msg("Successful preflight request")

				w.Header().Set("Access-Control-Max-Age", strconv.Itoa(c.MaxAge))
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		// Metrics
		if origin != "" {
			corsRequestsTotal.WithLabelValues(origin).Inc()
		}

		next.ServeHTTP(w, r)
	})
}

func (c *SecurityConfig) isAllowedOrigin(origin string) bool {
	if c.Environment != "production" {
		return true // Allow all origins in non-production environments
	}

	for _, allowed := range c.AllowedOrigins {
		if allowed == origin {
			return true
		}
	}
	return false
}
