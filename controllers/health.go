package controllers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type DBInterface interface {
	DB() (*sql.DB, error)
}

type RedisInterface interface {
	Ping(ctx context.Context) *redis.StatusCmd
}

type HealthController struct {
	db    DBInterface
	redis RedisInterface
}

type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

// gormDBAdapter adapts *gorm.DB to DBInterface
type gormDBAdapter struct {
	gormDB *gorm.DB
}

func (g *gormDBAdapter) DB() (*sql.DB, error) {
	return g.gormDB.DB()
}

func NewHealthController(db *gorm.DB, redis *redis.Client) *HealthController {
	return &HealthController{
		db:    &gormDBAdapter{gormDB: db},
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
		services["database"] = "error: " + err.Error()
		overallStatus = "unhealthy"
	} else if sqlDB == nil {
		services["database"] = "error: database connection is nil"
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
