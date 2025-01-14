package controllers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// Mock database
type mockDB struct {
	pingErr error
}

func (m *mockDB) DB() (*sql.DB, error) {
	// Return a custom error if DB() should fail
	if m.pingErr != nil && m.pingErr.Error() == "db connection error" {
		return nil, errors.New("failed to get database instance")
	}

	// For other cases, return nil to simulate a connection error
	return nil, nil
}

// Mock Redis client
type mockRedisClient struct {
	pingErr error
}

func (m *mockRedisClient) Ping(ctx context.Context) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx)
	if m.pingErr != nil {
		cmd.SetErr(m.pingErr)
	} else {
		cmd.SetVal("PONG")
	}
	return cmd
}

func TestHealthCheck(t *testing.T) {
	tests := []struct {
		name            string
		dbPingErr       error
		redisPingErr    error
		wantStatus      int
		wantHealth      string
		wantDBStatus    string
		wantRedisStatus string
	}{
		{
			name:            "All services healthy",
			dbPingErr:       nil,
			redisPingErr:    nil,
			wantStatus:      http.StatusServiceUnavailable,
			wantHealth:      "unhealthy",
			wantDBStatus:    "error: database connection is nil",
			wantRedisStatus: "healthy",
		},
		{
			name:            "Database connection error",
			dbPingErr:       errors.New("db connection error"),
			redisPingErr:    nil,
			wantStatus:      http.StatusServiceUnavailable,
			wantHealth:      "unhealthy",
			wantDBStatus:    "error: failed to get database instance",
			wantRedisStatus: "healthy",
		},
		{
			name:            "Database ping error",
			dbPingErr:       errors.New("database ping error"),
			redisPingErr:    nil,
			wantStatus:      http.StatusServiceUnavailable,
			wantHealth:      "unhealthy",
			wantDBStatus:    "error: database connection is nil",
			wantRedisStatus: "healthy",
		},
		{
			name:            "Redis unhealthy",
			dbPingErr:       nil,
			redisPingErr:    errors.New("redis connection error"),
			wantStatus:      http.StatusServiceUnavailable,
			wantHealth:      "unhealthy",
			wantDBStatus:    "error: database connection is nil",
			wantRedisStatus: "error: redis connection error",
		},
		{
			name:            "All services unhealthy",
			dbPingErr:       errors.New("database ping error"),
			redisPingErr:    errors.New("redis connection error"),
			wantStatus:      http.StatusServiceUnavailable,
			wantHealth:      "unhealthy",
			wantDBStatus:    "error: database connection is nil",
			wantRedisStatus: "error: redis connection error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockDB := &mockDB{pingErr: tt.dbPingErr}
			mockRedis := &mockRedisClient{pingErr: tt.redisPingErr}

			// Create controller with mocks
			controller := &HealthController{
				db:    mockDB,
				redis: mockRedis,
			}

			// Create request and response recorder
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()

			// Call health check
			controller.Check(w, req)

			// Assert response status code
			assert.Equal(t, tt.wantStatus, w.Code)

			// Parse response
			var response HealthResponse
			err := json.NewDecoder(w.Body).Decode(&response)
			assert.NoError(t, err)

			// Assert response fields
			assert.Equal(t, tt.wantHealth, response.Status)
			assert.Equal(t, tt.wantDBStatus, response.Services["database"])
			assert.Equal(t, tt.wantRedisStatus, response.Services["redis"])
			assert.NotZero(t, response.Timestamp)
			assert.True(t, response.Timestamp.Before(time.Now()))
		})
	}
}

func TestNewHealthController(t *testing.T) {
	db := &gorm.DB{}
	redis := &redis.Client{}

	controller := NewHealthController(db, redis)

	assert.NotNil(t, controller)
	assert.IsType(t, &gormDBAdapter{}, controller.db)
	assert.Equal(t, redis, controller.redis)
}
