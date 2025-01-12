package config

import (
	"os"
)

type Config struct {
	Port            string
	JWTSecret       string
	DatabaseURL     string
	TokenExpiration int64
	SMTP            SMTPConfig
	Redis           RedisConfig
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

func LoadConfig() *Config {
	return &Config{
		Port:            getEnv("PORT", "8080"),
		JWTSecret:       getEnv("JWT_SECRET", "your-super-secret-key-change-this-in-production"),
		DatabaseURL:     getEnv("DATABASE_URL", "host=localhost user=postgres password=postgres dbname=auth_db port=5432 sslmode=disable"),
		TokenExpiration: 24 * 60 * 60, // 24 hours in seconds
		SMTP: SMTPConfig{
			Host:     getEnv("SMTP_HOST", "smtp.gmail.com"),
			Port:     587, // Default TLS port
			Username: getEnv("SMTP_USERNAME", ""),
			Password: getEnv("SMTP_PASSWORD", ""),
			From:     getEnv("SMTP_FROM", "noreply@example.com"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       0, // Default database
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
