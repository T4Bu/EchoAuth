package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type HealthController struct {
	db    *gorm.DB
	redis *redis.Client
}

type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

func NewHealthController(db *gorm.DB, redis *redis.Client) *HealthController {
	return &HealthController{
		db:    db,
		redis: redis,
	}
}

func (h *HealthController) Check(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	services := make(map[string]string)
	overallStatus := "healthy"

	// Check database
	sqlDB, err := h.db.DB()
	if err != nil {
		services["database"] = "error: failed to get database instance"
		overallStatus = "unhealthy"
	} else if err := sqlDB.Ping(); err != nil {
		services["database"] = "error: " + err.Error()
		overallStatus = "unhealthy"
	} else {
		services["database"] = "healthy"
	}

	// Check Redis
	if err := h.redis.Ping(ctx).Err(); err != nil {
		services["redis"] = "error: " + err.Error()
		overallStatus = "unhealthy"
	} else {
		services["redis"] = "healthy"
	}

	// Prepare response
	response := HealthResponse{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Services:  services,
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	if overallStatus != "healthy" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	json.NewEncoder(w).Encode(response)
}
