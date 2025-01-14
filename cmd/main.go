package main

import (
	"auth/config"
	"auth/controllers"
	"auth/database"
	"auth/middlewares"
	"auth/models"
	"auth/repositories"
	"auth/services"
	"auth/utils/logger"
	"context"
	"net/http"
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
	router.HandleFunc("/api/auth/register", authController.Register).Methods("POST")
	router.HandleFunc("/api/auth/login", authController.Login).Methods("POST")
	router.HandleFunc("/api/auth/refresh", authController.RefreshToken).Methods("POST")

	// Protected routes
	protected := router.PathPrefix("/api").Subrouter()
	protected.Use(authMiddleware.Authenticate)
	protected.HandleFunc("/auth/logout", authController.Logout).Methods("POST")

	// Start cleanup goroutine for expired tokens
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		for range ticker.C {
			if err := tokenRepo.CleanupExpiredTokens(); err != nil {
				log.Error().Err(err).Msg("Failed to cleanup expired tokens")
			}
		}
	}()

	// Add metrics endpoint
	router.Handle("/metrics", promhttp.Handler())

	// Start server
	log.Info().Str("port", cfg.Port).Msg("Starting server")
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatal().Err(err).Msg("Server failed to start")
	}
}
