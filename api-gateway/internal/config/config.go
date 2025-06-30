package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server      ServerConfig      `yaml:"server"`
	Auth        AuthConfig        `yaml:"auth"`
	Database    DatabaseConfig    `yaml:"database"`
	Services    ServicesConfig    `yaml:"services"`
	Logging     LoggingConfig     `yaml:"logging"`
	Monitoring  MonitoringConfig  `yaml:"monitoring"`
}

type ServerConfig struct {
	Port    int           `yaml:"port"`
	Env     string        `yaml:"env"`
	Timeout time.Duration `yaml:"timeout"`
}

type AuthConfig struct {
	JWTSecret   string        `yaml:"jwt_secret"`
	TokenExpiry time.Duration `yaml:"token_expiry"`
}

type DatabaseConfig struct {
	Postgres PostgresConfig `yaml:"postgres"`
	Redis    RedisConfig    `yaml:"redis"`
}

type PostgresConfig struct {
	Host          string `yaml:"host"`
	Port          int    `yaml:"port"`
	User          string `yaml:"user"`
	Password      string `yaml:"password"`
	Name          string `yaml:"name"`
	MaxConnections int   `yaml:"max_connections"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type ServicesConfig struct {
	BookingService     string `yaml:"booking_service"`
	ScheduleService    string `yaml:"schedule_service"`
	NotificationService string `yaml:"notification_service"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

type MonitoringConfig struct {
	PrometheusPort int  `yaml:"prometheus_port"`
	EnableMetrics  bool `yaml:"enable_metrics"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{}

	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(file, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Валидация обязательных полей
	if cfg.Auth.JWTSecret == "" {
		return nil, fmt.Errorf("auth.jwt_secret is required")
	}

	if cfg.Database.Postgres.Host == "" {
		return nil, fmt.Errorf("database.postgres.host is required")
	}

	return cfg, nil
}