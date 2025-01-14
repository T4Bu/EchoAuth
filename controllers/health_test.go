package controllers

import (
	"EchoAuth/database"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestHealthController_Check(t *testing.T) {
	// Create a mock DB
	mockDB, err := sql.Open("postgres", "postgres://fake")
	assert.NoError(t, err)
	db := &database.DB{DB: mockDB}

	// Create a mock Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	controller := NewHealthController(db, redisClient)

	// Create a request
	req, err := http.NewRequest("GET", "/health", nil)
	assert.NoError(t, err)

	// Create a ResponseRecorder
	rr := httptest.NewRecorder()

	// Call the handler
	controller.Check(rr, req)

	// Check the status code
	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)

	// Parse the response
	var response HealthResponse
	err = json.NewDecoder(rr.Body).Decode(&response)
	assert.NoError(t, err)

	// Check the response fields
	assert.Equal(t, "unhealthy", response.Status)
	assert.Contains(t, response.Services, "database")
	assert.Contains(t, response.Services, "redis")
	assert.Contains(t, response.Services["database"], "error")
	assert.Contains(t, response.Services["redis"], "error")
	assert.WithinDuration(t, time.Now(), response.Timestamp, 2*time.Second)
}
