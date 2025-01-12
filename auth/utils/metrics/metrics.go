package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	AuthAttempts = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_attempts_total",
			Help: "Total number of authentication attempts by type and result",
		},
		[]string{"type", "result"},
	)

	ActiveSessions = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_sessions",
			Help: "Number of currently active sessions",
		},
	)

	RateLimitHits = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "rate_limit_hits_total",
			Help: "Total number of rate limit hits",
		},
	)

	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path", "method", "status"},
	)

	DatabaseOperations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "database_operations_total",
			Help: "Total number of database operations",
		},
		[]string{"operation", "status"},
	)
)

func init() {
	prometheus.MustRegister(AuthAttempts)
	prometheus.MustRegister(ActiveSessions)
	prometheus.MustRegister(RateLimitHits)
	prometheus.MustRegister(RequestDuration)
	prometheus.MustRegister(DatabaseOperations)
}

func RecordAuthAttempt(attemptType, result string) {
	AuthAttempts.WithLabelValues(attemptType, result).Inc()
}

func UpdateActiveSessions(delta int) {
	ActiveSessions.Add(float64(delta))
}

func RecordRateLimitHit() {
	RateLimitHits.Inc()
}

func RecordRequestDuration(path, method string, status int, duration time.Duration) {
	RequestDuration.WithLabelValues(path, method, strconv.Itoa(status)).Observe(duration.Seconds())
}

func RecordDatabaseOperation(operation, status string) {
	DatabaseOperations.WithLabelValues(operation, status).Inc()
}
