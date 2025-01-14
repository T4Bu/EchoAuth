package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path", "method", "status_code"},
	)

	DatabaseOperations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "database_operations_total",
			Help: "Total number of database operations",
		},
		[]string{"operation", "status"},
	)

	AuthenticationAttempts = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "authentication_attempts_total",
			Help: "Total number of authentication attempts",
		},
		[]string{"status"},
	)

	ActiveTokens = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_tokens",
			Help: "Number of currently active tokens",
		},
	)

	RateLimitHits = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "rate_limit_hits_total",
			Help: "Total number of rate limit hits",
		},
	)
)

func init() {
	prometheus.MustRegister(RequestDuration)
	prometheus.MustRegister(DatabaseOperations)
	prometheus.MustRegister(AuthenticationAttempts)
	prometheus.MustRegister(ActiveTokens)
	prometheus.MustRegister(RateLimitHits)
}

// RecordRequestDuration is middleware that records the duration of HTTP requests
func RecordRequestDuration(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, Status: http.StatusOK}
		next.ServeHTTP(rec, r)
		duration := time.Since(start).Seconds()
		RequestDuration.WithLabelValues(r.URL.Path, r.Method, strconv.Itoa(rec.Status)).Observe(duration)
	})
}

// statusRecorder wraps http.ResponseWriter to capture status code
type statusRecorder struct {
	http.ResponseWriter
	Status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.Status = status
	r.ResponseWriter.WriteHeader(status)
}

// RecordDatabaseOperation records a database operation with its status
func RecordDatabaseOperation(operation, status string) {
	DatabaseOperations.WithLabelValues(operation, status).Inc()
}

// RecordAuthenticationAttempt records an authentication attempt
func RecordAuthenticationAttempt(success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	AuthenticationAttempts.WithLabelValues(status).Inc()
}

// RecordActiveTokens sets the current number of active tokens
func RecordActiveTokens(count int) {
	ActiveTokens.Set(float64(count))
}

// RecordRateLimitHit increments the rate limit hits counter
func RecordRateLimitHit() {
	RateLimitHits.Inc()
}

// Handler returns an HTTP handler for the metrics endpoint
func Handler() http.Handler {
	return promhttp.Handler()
}
