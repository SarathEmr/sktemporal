package main

import (
	"fmt"
	"os"
)

// Config holds application configuration loaded from the environment.
type Config struct {
	PostgresUser     string
	PostgresPassword string
	PostgresHost     string
	PostgresPort     string
	AppDBName        string
}

// DBConnectionString returns the PostgreSQL connection string for the app database.
func (c *Config) DBConnectionString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.PostgresUser, c.PostgresPassword, c.PostgresHost, c.PostgresPort, c.AppDBName)
}

// LoadConfigFromEnv loads configuration from environment variables with defaults for development.
func LoadConfigFromEnv() *Config {
	return &Config{
		PostgresUser:     getEnv("POSTGRES_USER", "admin"),
		PostgresPassword: getEnv("POSTGRES_PASSWORD", "admin"),
		PostgresHost:     getEnv("POSTGRES_HOST", "temporal-postgres"),
		PostgresPort:     getEnv("POSTGRES_PORT", "5432"),
		AppDBName:        getEnv("APP_DB_NAME", "appdb"),
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
