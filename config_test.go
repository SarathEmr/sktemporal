package main

import (
	"os"
	"testing"
)

func TestConfig_DBConnectionString(t *testing.T) {
	tests := []struct {
		name string
		c    *Config
		want string
	}{
		{
			name: "default style values",
			c: &Config{
				PostgresUser:     "admin",
				PostgresPassword: "admin",
				PostgresHost:     "temporal-postgres",
				PostgresPort:     "5432",
				AppDBName:        "appdb",
			},
			want: "postgres://admin:admin@temporal-postgres:5432/appdb?sslmode=disable",
		},
		{
			name: "custom values",
			c: &Config{
				PostgresUser:     "user",
				PostgresPassword: "secret",
				PostgresHost:     "localhost",
				PostgresPort:     "5433",
				AppDBName:        "mydb",
			},
			want: "postgres://user:secret@localhost:5433/mydb?sslmode=disable",
		},
		{
			name: "empty values use defaults",
			c: &Config{
				PostgresUser:     "",
				PostgresPassword: "",
				PostgresHost:     "",
				PostgresPort:     "",
				AppDBName:        "",
			},
			want: "postgres://admin:admin@temporal-postgres:5432/appdb?sslmode=disable",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.c.DBConnectionString()
			if got != tt.want {
				t.Errorf("DBConnectionString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoadConfigFromEnv_Defaults(t *testing.T) {
	// Clear any relevant env vars so we get defaults
	keys := []string{"POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_HOST", "POSTGRES_PORT", "APP_DB_NAME"}
	restore := clearEnv(keys)
	defer restore()

	got := LoadConfigFromEnv()

	if got.PostgresUser != "admin" {
		t.Errorf("PostgresUser = %q, want admin", got.PostgresUser)
	}
	if got.PostgresPassword != "admin" {
		t.Errorf("PostgresPassword = %q, want admin", got.PostgresPassword)
	}
	if got.PostgresHost != "temporal-postgres" {
		t.Errorf("PostgresHost = %q, want temporal-postgres", got.PostgresHost)
	}
	if got.PostgresPort != "5432" {
		t.Errorf("PostgresPort = %q, want 5432", got.PostgresPort)
	}
	if got.AppDBName != "appdb" {
		t.Errorf("AppDBName = %q, want appdb", got.AppDBName)
	}
}

func TestLoadConfigFromEnv_Overrides(t *testing.T) {
	restore := setEnv(map[string]string{
		"POSTGRES_USER":     "myuser",
		"POSTGRES_PASSWORD": "mypass",
		"POSTGRES_HOST":     "db.example.com",
		"POSTGRES_PORT":     "5434",
		"APP_DB_NAME":       "testdb",
	})
	defer restore()

	got := LoadConfigFromEnv()

	if got.PostgresUser != "myuser" {
		t.Errorf("PostgresUser = %q, want myuser", got.PostgresUser)
	}
	if got.PostgresPassword != "mypass" {
		t.Errorf("PostgresPassword = %q, want mypass", got.PostgresPassword)
	}
	if got.PostgresHost != "db.example.com" {
		t.Errorf("PostgresHost = %q, want db.example.com", got.PostgresHost)
	}
	if got.PostgresPort != "5434" {
		t.Errorf("PostgresPort = %q, want 5434", got.PostgresPort)
	}
	if got.AppDBName != "testdb" {
		t.Errorf("AppDBName = %q, want testdb", got.AppDBName)
	}
}

func TestLoadConfigFromEnv_EmptyEnvUsesDefault(t *testing.T) {
	restore := setEnv(map[string]string{"POSTGRES_USER": ""})
	defer restore()

	got := LoadConfigFromEnv()
	// getEnv returns default when value is empty
	if got.PostgresUser != "admin" {
		t.Errorf("PostgresUser with empty env = %q, want default admin", got.PostgresUser)
	}
}

// clearEnv unsets the given keys and returns a function to restore their previous values.
func clearEnv(keys []string) func() {
	prev := make(map[string]string)
	for _, k := range keys {
		prev[k] = os.Getenv(k)
		os.Unsetenv(k)
	}
	return func() {
		for _, k := range keys {
			if v, ok := prev[k]; ok {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
	}
}

// setEnv sets the given env vars and returns a function to restore the previous state.
func setEnv(env map[string]string) func() {
	prev := make(map[string]string)
	for k, v := range env {
		prev[k] = os.Getenv(k)
		os.Setenv(k, v)
	}
	return func() {
		for k := range env {
			if v, ok := prev[k]; ok {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
	}
}
