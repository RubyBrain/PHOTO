package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

func LoadEnvOverrides(cfg *Config) {
	// Server
	if v := os.Getenv("SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}
	if v := os.Getenv("SERVER_ENV"); v != "" {
		cfg.Server.Env = v
	}

	// Auth
	if v := os.Getenv("AUTH_JWT_SECRET"); v != "" {
		cfg.Auth.JWTSecret = v
	}
	if v := os.Getenv("AUTH_TOKEN_EXPIRY"); v != "" {
		if expiry, err := time.ParseDuration(v); err == nil {
			cfg.Auth.TokenExpiry = expiry
		}
	}

	// Database
	if v := os.Getenv("DB_POSTGRES_HOST"); v != "" {
		cfg.Database.Postgres.Host = v
	}
	// ... остальные переменные аналогично

	log.Println("Environment variables loaded successfully")
}