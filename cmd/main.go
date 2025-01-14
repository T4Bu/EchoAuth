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
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// Dependencies holds all service dependencies
type Dependencies struct {
	UserRepo    repositories.UserRepository
	TokenRepo   *repositories.TokenRepository
	LockoutSvc  *services.AccountLockoutService
	AuthService controllers.AuthService
	RedisClient *redis.Client
	DB          *gorm.DB
}

// NewDependencies creates a new Dependencies instance
func NewDependencies(db *gorm.DB, redisClient *redis.Client, cfg *config.Config) *Dependencies {
	userRepo := repositories.NewUserRepository(db)
	tokenRepo := repositories.NewTokenRepository(db)
	lockoutSvc := services.NewAccountLockoutService(redisClient)
	authService := services.NewAuthService(userRepo, tokenRepo, cfg, lockoutSvc)

	return &Dependencies{
		UserRepo:    userRepo,
		TokenRepo:   tokenRepo,
		LockoutSvc:  lockoutSvc,
		AuthService: authService,
		RedisClient: redisClient,
		DB:          db,
	}
}

func initLogger() zerolog.Logger {
	logger.Init()
	return logger.GetLogger("main")
}

func initDatabase(cfg *config.Config, log zerolog.Logger) *gorm.DB {
	db, err := database.InitDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database")
	}

	err = db.AutoMigrate(&models.User{})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to migrate database schema")
	}
	log.Info().Msg("Database schema migrated successfully")
	return db
}

func initRedis(cfg *config.Config, log zerolog.Logger) *redis.Client {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := redisClient.Ping(ctx).Result(); err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Redis")
	}
	log.Info().Msg("Connected to Redis successfully")
	return redisClient
}

func setupRouter(deps *Dependencies) *mux.Router {
	healthController := controllers.NewHealthController(deps.DB, deps.RedisClient)
	authController := controllers.NewAuthController(deps.AuthService)

	authMiddleware := middlewares.NewAuthMiddleware(deps.AuthService)
	rateLimiter := middlewares.NewRateLimiter(deps.RedisClient)
	securityConfig := middlewares.NewSecurityConfig()

	router := mux.NewRouter()

	router.Use(rateLimiter.RateLimit)
	router.Use(securityConfig.SecurityMiddleware)

	router.HandleFunc("/health", healthController.Check).Methods("GET")
	router.HandleFunc("/api/EchoAuth/register", authController.Register).Methods("POST")
	router.HandleFunc("/api/EchoAuth/login", authController.Login).Methods("POST")
	router.HandleFunc("/api/EchoAuth/refresh", authController.RefreshToken).Methods("POST")

	protected := router.PathPrefix("/api").Subrouter()
	protected.Use(authMiddleware.Authenticate)
	protected.HandleFunc("/EchoAuth/logout", authController.Logout).Methods("POST")

	router.Handle("/metrics", promhttp.Handler())

	return router
}

func startCleanupRoutine(ctx context.Context, tokenRepo *repositories.TokenRepository, log zerolog.Logger) {
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
}

func startServer(router *mux.Router, cfg *config.Config, log zerolog.Logger, deps *Dependencies) {
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Create context that listens for signals
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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
	sqlDB, err := deps.DB.DB()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get database instance")
	} else {
		if err := sqlDB.Close(); err != nil {
			log.Error().Err(err).Msg("Error closing database connection")
		}
	}

	// Close Redis connection
	if err := deps.RedisClient.Close(); err != nil {
		log.Error().Err(err).Msg("Error closing Redis connection")
	}

	log.Info().Msg("Server exited properly")
}

func main() {
	log := initLogger()
	log.Info().Msg("Starting authentication service")

	cfg := config.LoadConfig()
	log.Debug().Interface("config", cfg).Msg("Configuration loaded")

	db := initDatabase(cfg, log)
	redisClient := initRedis(cfg, log)
	deps := NewDependencies(db, redisClient, cfg)

	router := setupRouter(deps)

	// Create context for cleanup routine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startCleanupRoutine(ctx, deps.TokenRepo, log)
	startServer(router, cfg, log, deps)
}
