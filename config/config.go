package config

import (
	"os"
	"strconv"
	"time"
)

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type Config struct {
	Port        string
	JWTSecret   string
	JWTExpiry   time.Duration
	DatabaseURL string
	Redis       RedisConfig
	Environment string
}

func LoadConfig() *Config {
	jwtExpiry := 24 * time.Hour
	if expStr := getEnv("JWT_EXPIRY", "24h"); expStr != "" {
		if exp, err := time.ParseDuration(expStr); err == nil {
			jwtExpiry = exp
		}
	}

	redisDB := 0
	if dbStr := getEnv("REDIS_DB", "0"); dbStr != "" {
		if db, err := strconv.Atoi(dbStr); err == nil {
			redisDB = db
		}
	}

	return &Config{
		Port:        getEnv("PORT", "8080"),
		JWTSecret:   getEnv("JWT_SECRET", "your-secret-key"),
		JWTExpiry:   jwtExpiry,
		DatabaseURL: getEnv("DATABASE_URL", "host=localhost user=postgres password=postgres dbname=auth_db port=5432 sslmode=disable"),
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASS", ""),
			DB:       redisDB,
		},
		Environment: getEnv("ENV", "development"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
