package config

import (
	"os"
	"strconv"
)

// Config holds application configuration
type Config struct {
	Port        int    `json:"port"`
	DatabaseURL string `json:"database_url"`
	LogLevel    string `json:"log_level"`
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	port := 8080
	if portStr := os.Getenv("PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	return &Config{
		Port:        port,
		DatabaseURL: getEnvOrDefault("DATABASE_URL", "sqlite://./app.db"),
		LogLevel:    getEnvOrDefault("LOG_LEVEL", "info"),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}