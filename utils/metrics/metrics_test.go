package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestRecordRequestDuration(t *testing.T) {
	// Reset metrics before test
	prometheus.Unregister(RequestDuration)
	prometheus.MustRegister(RequestDuration)

	tests := []struct {
		name       string
		path       string
		method     string
		statusCode int
		sleep      time.Duration
	}{
		{
			name:       "Successful GET request",
			path:       "/api/users",
			method:     "GET",
			statusCode: http.StatusOK,
			sleep:      10 * time.Millisecond,
		},
		{
			name:       "Failed POST request",
			path:       "/api/users",
			method:     "POST",
			statusCode: http.StatusBadRequest,
			sleep:      20 * time.Millisecond,
		},
		{
			name:       "Not found request",
			path:       "/api/unknown",
			method:     "GET",
			statusCode: http.StatusNotFound,
			sleep:      5 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test request
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			// Create handler that sleeps to simulate processing time
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(tt.sleep)
				w.WriteHeader(tt.statusCode)
			})

			// Wrap with metrics middleware
			wrappedHandler := RecordRequestDuration(handler)
			wrappedHandler.ServeHTTP(w, req)

			// Verify metric exists
			if testutil.CollectAndCount(RequestDuration) == 0 {
				t.Error("Expected request duration metric to be recorded")
			}
		})
	}
}

func TestRecordDatabaseOperation(t *testing.T) {
	// Reset metrics before test
	prometheus.Unregister(DatabaseOperations)
	prometheus.MustRegister(DatabaseOperations)

	tests := []struct {
		name       string
		operation  string
		status     string
		wantCount  float64
		wantLabels map[string]string
	}{
		{
			name:      "Successful create operation",
			operation: "create",
			status:    "success",
			wantCount: 1,
			wantLabels: map[string]string{
				"operation": "create",
				"status":    "success",
			},
		},
		{
			name:      "Failed read operation",
			operation: "read",
			status:    "failure",
			wantCount: 1,
			wantLabels: map[string]string{
				"operation": "read",
				"status":    "failure",
			},
		},
		{
			name:      "Successful update operation",
			operation: "update",
			status:    "success",
			wantCount: 1,
			wantLabels: map[string]string{
				"operation": "update",
				"status":    "success",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Record operation
			RecordDatabaseOperation(tt.operation, tt.status)

			// Verify metric was recorded with correct labels and value
			count := testutil.ToFloat64(DatabaseOperations.WithLabelValues(tt.operation, tt.status))
			if count != tt.wantCount {
				t.Errorf("DatabaseOperations count = %v, want %v", count, tt.wantCount)
			}
		})
	}
}

func TestRecordAuthenticationAttempt(t *testing.T) {
	// Reset metrics before test
	prometheus.Unregister(AuthenticationAttempts)
	prometheus.MustRegister(AuthenticationAttempts)

	tests := []struct {
		name       string
		success    bool
		wantCount  float64
		wantLabels map[string]string
	}{
		{
			name:      "Successful authentication",
			success:   true,
			wantCount: 1,
			wantLabels: map[string]string{
				"status": "success",
			},
		},
		{
			name:      "Failed authentication",
			success:   false,
			wantCount: 1,
			wantLabels: map[string]string{
				"status": "failure",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Record authentication attempt
			RecordAuthenticationAttempt(tt.success)

			// Verify metric was recorded with correct labels and value
			status := "success"
			if !tt.success {
				status = "failure"
			}
			count := testutil.ToFloat64(AuthenticationAttempts.WithLabelValues(status))
			if count != tt.wantCount {
				t.Errorf("AuthenticationAttempts count = %v, want %v", count, tt.wantCount)
			}
		})
	}
}

func TestRecordActiveTokens(t *testing.T) {
	// Reset metrics before test
	prometheus.Unregister(ActiveTokens)
	prometheus.MustRegister(ActiveTokens)

	tests := []struct {
		name      string
		count     int
		wantValue float64
	}{
		{
			name:      "Zero active tokens",
			count:     0,
			wantValue: 0,
		},
		{
			name:      "Multiple active tokens",
			count:     5,
			wantValue: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set active tokens count
			RecordActiveTokens(tt.count)

			// Verify metric value
			value := testutil.ToFloat64(ActiveTokens)
			if value != tt.wantValue {
				t.Errorf("ActiveTokens value = %v, want %v", value, tt.wantValue)
			}
		})
	}
}

func TestMetricsEndpoint(t *testing.T) {
	// Create test request
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	// Record some test metrics
	RecordDatabaseOperation("create", "success")
	RecordAuthenticationAttempt(true)
	RecordActiveTokens(3)

	// Call metrics endpoint
	Handler().ServeHTTP(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Metrics endpoint returned wrong status code: got %v want %v",
			w.Code, http.StatusOK)
	}

	body := w.Body.String()

	// Verify metrics are present in response
	expectedMetrics := []string{
		"database_operations_total",
		"authentication_attempts_total",
		"active_tokens",
		"http_request_duration_seconds",
	}

	for _, metric := range expectedMetrics {
		if !strings.Contains(body, metric) {
			t.Errorf("Metric %s not found in response", metric)
		}
	}
}
