package main

import (
	"EchoAuth/config"
	"EchoAuth/controllers"
	"EchoAuth/repositories"
	"EchoAuth/services"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) *gorm.DB {
	dsn := "host=localhost user=postgres password=postgres dbname=auth_test_db port=5433 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skipf("Skipping test: could not connect to test database: %v", err)
		return nil
	}
	return db
}

func TestNewDependencies(t *testing.T) {
	// Setup test dependencies
	db := setupTestDB(t)
	if db == nil {
		return
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	cfg := &config.Config{
		JWTSecret: "test-secret",
		JWTExpiry: 24 * time.Hour,
	}

	// Create dependencies
	deps := NewDependencies(db, redisClient, cfg)

	// Assert all dependencies are properly initialized
	assert.NotNil(t, deps)
	assert.NotNil(t, deps.UserRepo)
	assert.NotNil(t, deps.TokenRepo)
	assert.NotNil(t, deps.LockoutSvc)
	assert.NotNil(t, deps.AuthService)
	assert.Equal(t, db, deps.DB)
	assert.Equal(t, redisClient, deps.RedisClient)
}

func TestInitLogger(t *testing.T) {
	// Set test environment variables
	prevLevel := os.Getenv("LOG_LEVEL")
	os.Setenv("LOG_LEVEL", "debug")
	defer os.Setenv("LOG_LEVEL", prevLevel)

	// Initialize logger
	log := initLogger()

	// Assert logger is properly initialized
	assert.NotNil(t, log)
	assert.Equal(t, zerolog.DebugLevel, zerolog.GlobalLevel())
}

func TestSetupRouter(t *testing.T) {
	// Setup test dependencies
	db := setupTestDB(t)
	if db == nil {
		return
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	cfg := &config.Config{
		JWTSecret: "test-secret",
		JWTExpiry: 24 * time.Hour,
	}

	userRepo := repositories.NewUserRepository(db)
	tokenRepo := repositories.NewTokenRepository(db)
	lockoutSvc := services.NewAccountLockoutService(redisClient)
	authService := services.NewAuthService(userRepo, tokenRepo, cfg, lockoutSvc)

	// Create router
	router := mux.NewRouter()
	authController := controllers.NewAuthController(authService)

	router.HandleFunc("/api/EchoAuth/register", authController.Register).Methods("POST")
	router.HandleFunc("/api/EchoAuth/login", authController.Login).Methods("POST")
	router.HandleFunc("/api/EchoAuth/refresh", authController.RefreshToken).Methods("POST")
	router.Handle("/metrics", promhttp.Handler())

	// Test routes
	testCases := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{"Register", "POST", "/api/EchoAuth/register", http.StatusBadRequest},
		{"Login", "POST", "/api/EchoAuth/login", http.StatusBadRequest},
		{"Refresh Token", "POST", "/api/EchoAuth/refresh", http.StatusBadRequest},
		{"Metrics", "GET", "/metrics", http.StatusOK},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// For now, we just verify that the routes exist by checking that we don't get a 404
			assert.NotEqual(t, http.StatusNotFound, w.Code, "Route %s should exist", tc.path)
			assert.Equal(t, tc.expectedStatus, w.Code, "Route %s should return expected status", tc.path)
		})
	}
}

func TestHealthCheck(t *testing.T) {
	// Test cases for health check
	testCases := []struct {
		name           string
		dbError        bool
		redisError     bool
		expectedStatus int
	}{
		{
			name:           "All services healthy",
			dbError:        false,
			redisError:     false,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Database unhealthy",
			dbError:        true,
			redisError:     false,
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "Redis unhealthy",
			dbError:        false,
			redisError:     true,
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "All services unhealthy",
			dbError:        true,
			redisError:     true,
			expectedStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup test dependencies
			db := setupTestDB(t)
			if db == nil {
				return
			}

			// If testing database error, close the connection
			if tc.dbError {
				sqlDB, _ := db.DB()
				sqlDB.Close()
			}

			redisClient := redis.NewClient(&redis.Options{
				Addr: "localhost:6379",
			})

			// If testing Redis error, close the connection
			if tc.redisError {
				redisClient.Close()
			}

			// Create health controller
			healthController := controllers.NewHealthController(db, redisClient)

			// Create request and response recorder
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()

			// Call health check
			healthController.Check(w, req)

			// Assert response status code
			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

func TestStartCleanupRoutine(t *testing.T) {
	// Create test context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Create mock dependencies
	log := zerolog.New(os.Stdout)
	db := setupTestDB(t)
	if db == nil {
		return
	}

	tokenRepo := repositories.NewTokenRepository(db)

	// Start cleanup routine
	startCleanupRoutine(ctx, tokenRepo, log)

	// Wait for context to be done
	<-ctx.Done()
}

func TestStartServer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping server test in short mode")
	}

	// Create test dependencies
	cfg := &config.Config{
		Port: "8081",
		Redis: config.RedisConfig{
			Addr: "localhost:6379",
		},
		DatabaseURL: "host=localhost user=postgres password=postgres dbname=auth_test_db port=5433 sslmode=disable",
	}

	// Initialize real dependencies for integration test
	log := zerolog.New(os.Stdout)
	db := initDatabase(cfg, log)
	redisClient := initRedis(cfg, log)

	deps := &Dependencies{
		DB:          db,
		RedisClient: redisClient,
	}

	router := setupRouter(deps)

	// Start server in a goroutine
	go func() {
		startServer(router, cfg, log, deps)
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Send interrupt signal to trigger shutdown
	p, _ := os.FindProcess(os.Getpid())
	p.Signal(os.Interrupt)

	// Wait for shutdown to complete
	time.Sleep(100 * time.Millisecond)
}

func TestInitDatabase_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cfg := &config.Config{
		DatabaseURL: "host=localhost user=postgres password=postgres dbname=auth_test_db port=5433 sslmode=disable",
	}
	log := zerolog.New(os.Stdout)

	db := initDatabase(cfg, log)
	assert.NotNil(t, db)

	// Verify connection works
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, sqlDB.Ping())
}

func TestInitRedis_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cfg := &config.Config{
		Redis: config.RedisConfig{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		},
	}
	log := zerolog.New(os.Stdout)

	client := initRedis(cfg, log)
	assert.NotNil(t, client)

	// Verify connection works
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Ping(ctx).Result()
	assert.NoError(t, err)
}
