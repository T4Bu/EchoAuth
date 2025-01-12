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

	// Initialize services
	userRepo := repositories.NewUserRepository(db)
	rateLimiter := services.NewRateLimiter(redisClient, services.RateLimiterConfig{
		MaxAttempts: 5,
		Window:      time.Minute,
	})
	lockoutSvc := services.NewAccountLockoutService(redisClient)
	authService := services.NewAuthService(userRepo, []byte(cfg.JWTSecret), lockoutSvc)
	emailService := services.NewEmailService(services.SMTPConfig{
		Host:     cfg.SMTP.Host,
		Port:     cfg.SMTP.Port,
		Username: cfg.SMTP.Username,
		Password: cfg.SMTP.Password,
		From:     cfg.SMTP.From,
	})
	passwordResetService := services.NewPasswordResetService(userRepo, emailService)

	// Initialize controllers
	authController := controllers.NewAuthController(authService)
	passwordResetController := controllers.NewPasswordResetController(passwordResetService)
	healthController := controllers.NewHealthController(db, redisClient)

	// Initialize router
	router := mux.NewRouter()

	// Add metrics endpoint
	router.Handle("/metrics", promhttp.Handler())

	// Public routes (with rate limiting)
	publicRouter := router.PathPrefix("").Subrouter()
	publicRouter.Use(middlewares.RateLimitMiddleware(rateLimiter))
	publicRouter.HandleFunc("/auth/register", authController.Register).Methods("POST")
	publicRouter.HandleFunc("/auth/login", authController.Login).Methods("POST")
	publicRouter.HandleFunc("/auth/reset-password/request", passwordResetController.RequestReset).Methods("POST")
	publicRouter.HandleFunc("/auth/reset-password/reset", passwordResetController.ResetPassword).Methods("POST")

	// Protected routes
	protectedRouter := router.PathPrefix("").Subrouter()
	protectedRouter.Use(middlewares.AuthMiddleware(authService))
	protectedRouter.HandleFunc("/auth/logout", authController.Logout).Methods("POST")

	// Health check endpoint
	router.HandleFunc("/health", healthController.Check).Methods("GET")

	// Start server
	log.Info().Str("port", cfg.Port).Msg("Starting server")
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatal().Err(err).Msg("Server failed to start")
	}
}
