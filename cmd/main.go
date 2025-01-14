package main

import (
	"EchoAuth/config"
	"EchoAuth/controllers"
	"EchoAuth/database"
	"EchoAuth/middlewares"
	"EchoAuth/models"
	"EchoAuth/repositories"
	"EchoAuth/services"
	"EchoAuth/utils/logger"
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Initialize logger
	logger.Init()
	log := logger.GetLogger("main")
	log.Info().Msg("Starting authentication service")

	// Load configuration
	cfg := config.LoadConfig()
	log.Debug().Interface("config", cfg).Msg("Configuration loaded")

	// Initialize database
	db, err := database.InitDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database")
	}

	// Auto migrate schema
	err = db.AutoMigrate(&models.User{})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to migrate database schema")
	}
	log.Info().Msg("Database schema migrated successfully")

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Test Redis connection
	ctx := context.Background()
	_, err = redisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Redis")
	}
	log.Info().Msg("Connected to Redis successfully")

	// Initialize repositories
	userRepo := repositories.NewUserRepository(db)
	tokenRepo := repositories.NewTokenRepository(db)

	// Initialize services
	lockoutSvc := services.NewAccountLockoutService(redisClient)
	authService := services.NewAuthService(userRepo, tokenRepo, cfg, lockoutSvc)

	// Initialize controllers
	healthController := controllers.NewHealthController(db, redisClient)
	authController := controllers.NewAuthController(authService)

	// Initialize middleware
	authMiddleware := middlewares.NewAuthMiddleware(authService)
	rateLimiter := middlewares.NewRateLimiter(redisClient)
	securityConfig := middlewares.NewSecurityConfig()

	// Setup router
	router := mux.NewRouter()

	// Apply global middleware
	router.Use(rateLimiter.RateLimit)
	router.Use(securityConfig.SecurityMiddleware)

	// Public routes
	router.HandleFunc("/health", healthController.Check).Methods("GET")
	router.HandleFunc("/api/EchoAuth/register", authController.Register).Methods("POST")
	router.HandleFunc("/api/EchoAuth/login", authController.Login).Methods("POST")
	router.HandleFunc("/api/EchoAuth/refresh", authController.RefreshToken).Methods("POST")

	// Protected routes
	protected := router.PathPrefix("/api").Subrouter()
	protected.Use(authMiddleware.Authenticate)
	protected.HandleFunc("/EchoAuth/logout", authController.Logout).Methods("POST")

	// Add metrics endpoint
	router.Handle("/metrics", promhttp.Handler())

	// Create server instance
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// Create context that listens for signals
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start cleanup goroutine for expired tokens
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := tokenRepo.CleanupExpiredTokens(); err != nil {
					log.Error().Err(err).Msg("Failed to cleanup expired tokens")
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Start server in a goroutine
	go func() {
		log.Info().Str("port", cfg.Port).Msg("Starting server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed to start")
		}
	}()

	// Wait for interrupt signal
	<-ctx.Done()
	log.Info().Msg("Shutting down gracefully...")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	// Close database connection
	sqlDB, err := db.DB()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get database instance")
	} else {
		if err := sqlDB.Close(); err != nil {
			log.Error().Err(err).Msg("Error closing database connection")
		}
	}

	// Close Redis connection
	if err := redisClient.Close(); err != nil {
		log.Error().Err(err).Msg("Error closing Redis connection")
	}

	log.Info().Msg("Server exited properly")
}
