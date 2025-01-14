package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Save original environment variables
	origEnv := map[string]string{
		"PORT":         os.Getenv("PORT"),
		"JWT_SECRET":   os.Getenv("JWT_SECRET"),
		"JWT_EXPIRY":   os.Getenv("JWT_EXPIRY"),
		"DATABASE_URL": os.Getenv("DATABASE_URL"),
		"REDIS_ADDR":   os.Getenv("REDIS_ADDR"),
		"REDIS_PASS":   os.Getenv("REDIS_PASS"),
		"REDIS_DB":     os.Getenv("REDIS_DB"),
		"ENV":          os.Getenv("ENV"),
	}

	// Cleanup function to restore original environment
	defer func() {
		for k, v := range origEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	tests := []struct {
		name        string
		envVars     map[string]string
		want        Config
		description string
	}{
		{
			name: "Default values",
			envVars: map[string]string{
				"PORT":         "",
				"JWT_SECRET":   "",
				"JWT_EXPIRY":   "",
				"DATABASE_URL": "",
				"REDIS_ADDR":   "",
				"REDIS_PASS":   "",
				"REDIS_DB":     "",
				"ENV":          "",
			},
			want: Config{
				Port:        "8080",
				JWTSecret:   "your-secret-key",
				JWTExpiry:   24 * time.Hour,
				DatabaseURL: "host=localhost user=postgres password=postgres dbname=auth_db port=5432 sslmode=disable",
				Redis: RedisConfig{
					Addr:     "localhost:6379",
					Password: "",
					DB:       0,
				},
				Environment: "development",
			},
			description: "Should use default values when environment variables are not set",
		},
		{
			name: "Custom values",
			envVars: map[string]string{
				"PORT":         "3000",
				"JWT_SECRET":   "custom-secret",
				"JWT_EXPIRY":   "12h",
				"DATABASE_URL": "custom-db-url",
				"REDIS_ADDR":   "redis:6379",
				"REDIS_PASS":   "redis-pass",
				"REDIS_DB":     "1",
				"ENV":          "production",
			},
			want: Config{
				Port:        "3000",
				JWTSecret:   "custom-secret",
				JWTExpiry:   12 * time.Hour,
				DatabaseURL: "custom-db-url",
				Redis: RedisConfig{
					Addr:     "redis:6379",
					Password: "redis-pass",
					DB:       1,
				},
				Environment: "production",
			},
			description: "Should use environment variable values when set",
		},
		{
			name: "Invalid JWT expiry",
			envVars: map[string]string{
				"PORT":         "",
				"JWT_SECRET":   "",
				"JWT_EXPIRY":   "invalid",
				"DATABASE_URL": "",
				"REDIS_ADDR":   "",
				"REDIS_PASS":   "",
				"REDIS_DB":     "",
				"ENV":          "",
			},
			want: Config{
				Port:        "8080",
				JWTSecret:   "your-secret-key",
				JWTExpiry:   24 * time.Hour, // Should fall back to default
				DatabaseURL: "host=localhost user=postgres password=postgres dbname=auth_db port=5432 sslmode=disable",
				Redis: RedisConfig{
					Addr:     "localhost:6379",
					Password: "",
					DB:       0,
				},
				Environment: "development",
			},
			description: "Should use default JWT expiry when invalid duration is provided",
		},
		{
			name: "Invalid Redis DB",
			envVars: map[string]string{
				"PORT":         "",
				"JWT_SECRET":   "",
				"JWT_EXPIRY":   "",
				"DATABASE_URL": "",
				"REDIS_ADDR":   "",
				"REDIS_PASS":   "",
				"REDIS_DB":     "invalid",
				"ENV":          "",
			},
			want: Config{
				Port:        "8080",
				JWTSecret:   "your-secret-key",
				JWTExpiry:   24 * time.Hour,
				DatabaseURL: "host=localhost user=postgres password=postgres dbname=auth_db port=5432 sslmode=disable",
				Redis: RedisConfig{
					Addr:     "localhost:6379",
					Password: "",
					DB:       0, // Should fall back to default
				},
				Environment: "development",
			},
			description: "Should use default Redis DB when invalid number is provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all environment variables first
			for k := range origEnv {
				os.Unsetenv(k)
			}

			// Set environment variables for the test
			for k, v := range tt.envVars {
				if v != "" {
					os.Setenv(k, v)
				}
			}

			got := LoadConfig()

			// Compare fields
			if got.Port != tt.want.Port {
				t.Errorf("LoadConfig().Port = %v, want %v", got.Port, tt.want.Port)
			}
			if got.JWTSecret != tt.want.JWTSecret {
				t.Errorf("LoadConfig().JWTSecret = %v, want %v", got.JWTSecret, tt.want.JWTSecret)
			}
			if got.JWTExpiry != tt.want.JWTExpiry {
				t.Errorf("LoadConfig().JWTExpiry = %v, want %v", got.JWTExpiry, tt.want.JWTExpiry)
			}
			if got.DatabaseURL != tt.want.DatabaseURL {
				t.Errorf("LoadConfig().DatabaseURL = %v, want %v", got.DatabaseURL, tt.want.DatabaseURL)
			}
			if got.Redis.Addr != tt.want.Redis.Addr {
				t.Errorf("LoadConfig().Redis.Addr = %v, want %v", got.Redis.Addr, tt.want.Redis.Addr)
			}
			if got.Redis.Password != tt.want.Redis.Password {
				t.Errorf("LoadConfig().Redis.Password = %v, want %v", got.Redis.Password, tt.want.Redis.Password)
			}
			if got.Redis.DB != tt.want.Redis.DB {
				t.Errorf("LoadConfig().Redis.DB = %v, want %v", got.Redis.DB, tt.want.Redis.DB)
			}
			if got.Environment != tt.want.Environment {
				t.Errorf("LoadConfig().Environment = %v, want %v", got.Environment, tt.want.Environment)
			}
		})
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		want         string
		description  string
	}{
		{
			name:         "Existing environment variable",
			key:          "TEST_KEY",
			defaultValue: "default",
			envValue:     "test-value",
			want:         "test-value",
			description:  "Should return environment variable value when set",
		},
		{
			name:         "Non-existent environment variable",
			key:          "NON_EXISTENT_KEY",
			defaultValue: "default",
			envValue:     "",
			want:         "default",
			description:  "Should return default value when environment variable is not set",
		},
		{
			name:         "Empty environment variable",
			key:          "EMPTY_KEY",
			defaultValue: "default",
			envValue:     "",
			want:         "default",
			description:  "Should return default value when environment variable is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			if got := getEnv(tt.key, tt.defaultValue); got != tt.want {
				t.Errorf("getEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}
