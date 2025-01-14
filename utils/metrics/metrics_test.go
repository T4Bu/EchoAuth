package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestRecordAuthAttempt(t *testing.T) {
	// Reset metrics before test
	prometheus.Unregister(AuthAttempts)
	prometheus.MustRegister(AuthAttempts)

	// Record some auth attempts
	RecordAuthAttempt("login", "success")
	RecordAuthAttempt("login", "failure")
	RecordAuthAttempt("register", "success")

	// Check the values
	success := testutil.ToFloat64(AuthAttempts.WithLabelValues("login", "success"))
	if success != 1 {
		t.Errorf("Expected 1 successful login attempt, got %f", success)
	}

	failure := testutil.ToFloat64(AuthAttempts.WithLabelValues("login", "failure"))
	if failure != 1 {
		t.Errorf("Expected 1 failed login attempt, got %f", failure)
	}

	regSuccess := testutil.ToFloat64(AuthAttempts.WithLabelValues("register", "success"))
	if regSuccess != 1 {
		t.Errorf("Expected 1 successful registration, got %f", regSuccess)
	}
}

func TestUpdateActiveSessions(t *testing.T) {
	// Reset metrics before test
	prometheus.Unregister(ActiveSessions)
	prometheus.MustRegister(ActiveSessions)

	// Test incrementing and decrementing
	UpdateActiveSessions(1)
	if sessions := testutil.ToFloat64(ActiveSessions); sessions != 1 {
		t.Errorf("Expected 1 active session, got %f", sessions)
	}

	UpdateActiveSessions(-1)
	if sessions := testutil.ToFloat64(ActiveSessions); sessions != 0 {
		t.Errorf("Expected 0 active sessions, got %f", sessions)
	}
}

func TestRecordRateLimitHit(t *testing.T) {
	// Reset metrics before test
	prometheus.Unregister(RateLimitHits)
	prometheus.MustRegister(RateLimitHits)

	// Record some hits
	RecordRateLimitHit()
	RecordRateLimitHit()

	// Check the value
	hits := testutil.ToFloat64(RateLimitHits)
	if hits != 2 {
		t.Errorf("Expected 2 rate limit hits, got %f", hits)
	}
}

func TestRecordRequestDuration(t *testing.T) {
	// Reset metrics before test
	prometheus.Unregister(RequestDuration)
	prometheus.MustRegister(RequestDuration)

	// Record some durations
	RecordRequestDuration("/auth/login", "POST", 200, 100*time.Millisecond)
	RecordRequestDuration("/auth/login", "POST", 401, 50*time.Millisecond)

	// Check that metrics were recorded (we can't easily check exact values for histograms)
	if testutil.CollectAndCount(RequestDuration) == 0 {
		t.Error("Expected request duration metrics to be recorded")
	}
}

func TestRecordDatabaseOperation(t *testing.T) {
	// Reset metrics before test
	prometheus.Unregister(DatabaseOperations)
	prometheus.MustRegister(DatabaseOperations)

	// Record some operations
	RecordDatabaseOperation("create", "success")
	RecordDatabaseOperation("read", "success")
	RecordDatabaseOperation("read", "failure")

	// Check the values
	createSuccess := testutil.ToFloat64(DatabaseOperations.WithLabelValues("create", "success"))
	if createSuccess != 1 {
		t.Errorf("Expected 1 successful create operation, got %f", createSuccess)
	}

	readSuccess := testutil.ToFloat64(DatabaseOperations.WithLabelValues("read", "success"))
	if readSuccess != 1 {
		t.Errorf("Expected 1 successful read operation, got %f", readSuccess)
	}

	readFailure := testutil.ToFloat64(DatabaseOperations.WithLabelValues("read", "failure"))
	if readFailure != 1 {
		t.Errorf("Expected 1 failed read operation, got %f", readFailure)
	}
}
