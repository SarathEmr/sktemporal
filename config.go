package main

import (
	"fmt"
	"os"
)

const (
	pgUserDefault     = "admin"
	pgPasswordDefault = "admin"
	pgHostDefault     = "temporal-postgres"
	pgPortDefault     = "5432"
	appDBNameDefault  = "appdb"
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
	pgUser := c.PostgresUser
	if pgUser == "" {
		pgUser = pgUserDefault
	}
	pgPassword := c.PostgresPassword
	if pgPassword == "" {
		pgPassword = pgPasswordDefault
	}
	pgHost := c.PostgresHost
	if pgHost == "" {
		pgHost = pgHostDefault
	}
	pgPort := c.PostgresPort
	if pgPort == "" {
		pgPort = pgPortDefault
	}
	appDBName := c.AppDBName
	if appDBName == "" {
		appDBName = appDBNameDefault
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		pgUser, pgPassword, pgHost, pgPort, appDBName)
}

// LoadConfigFromEnv loads configuration from environment variables with defaults for development.
func LoadConfigFromEnv() *Config {
	return &Config{
		PostgresUser:     getEnv("POSTGRES_USER", pgUserDefault),
		PostgresPassword: getEnv("POSTGRES_PASSWORD", pgPasswordDefault),
		PostgresHost:     getEnv("POSTGRES_HOST", pgHostDefault),
		PostgresPort:     getEnv("POSTGRES_PORT", pgPortDefault),
		AppDBName:        getEnv("APP_DB_NAME", appDBNameDefault),
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
