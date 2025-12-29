package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all configuration for the ledger service
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
}

// ServerConfig holds gRPC server configuration
type ServerConfig struct {
	Port int
	Host string
}

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
	MaxConns int
	MinConns int
}

// Load loads configuration from environment variables with defaults
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port: getEnvAsInt("SERVER_PORT", 9090),
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvAsInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			DBName:   getEnv("DB_NAME", "ledger"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
			MaxConns: getEnvAsInt("DB_MAX_CONNS", 25),
			MinConns: getEnvAsInt("DB_MIN_CONNS", 5),
		},
	}

	return cfg, nil
}

// ConnectionString returns the PostgreSQL connection string
func (d *DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt retrieves an environment variable as integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}
