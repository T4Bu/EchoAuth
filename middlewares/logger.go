package middlewares

import (
	"auth/utils/logger"
	"auth/utils/metrics"
	"net/http"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	status      int
	written     int64
	wroteHeader bool
}

func (w *responseWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.status = code
		w.wroteHeader = true
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(b)
	w.written += int64(n)
	return n, err
}

func LoggerMiddleware(next http.Handler) http.Handler {
	log := logger.GetLogger("http")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a custom response writer to capture status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			status:         200,
			wroteHeader:    false,
		}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Calculate duration
		duration := time.Since(start)

		// Record metrics
		metrics.RecordRequestDuration(r.URL.Path, r.Method, wrapped.status, duration)

		// Create log event
		event := log.Info().
			Str("method", r.Method).
			Str("url", r.URL.String()).
			Str("path", r.URL.Path).
			Str("remote_ip", r.RemoteAddr).
			Int("status", wrapped.status).
			Int64("size", wrapped.written).
			Dur("duration", duration)

		// Add request ID if present
		if reqID := r.Header.Get("X-Request-ID"); reqID != "" {
			event.Str("request_id", reqID)
		}

		// Add user ID if authenticated
		if userID := r.Context().Value("user_id"); userID != nil {
			event.Interface("user_id", userID)
		}

		// Log the event
		event.Msg("Request processed")
	})
}
