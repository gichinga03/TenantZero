// Package config loads TenantZero server configuration from environment variables.
package config

import (
	"fmt"
	"os"
)

// Config holds all server configuration derived from the environment.
type Config struct {
	// Port is the TCP port the server listens on. Defaults to "8080".
	// Set via SERVER_PORT.
	Port string

	// DBURL is the PostgreSQL connection string. Required.
	// Set via DATABASE_URL.
	DBURL string

	// JWTKey is the HMAC secret for signing JWTs. Required.
	// Set via JWT_SECRET.
	JWTKey string

	// LabsDir is the filesystem path to the labs directory. Defaults to "./labs".
	// Set via LABS_DIR.
	LabsDir string
}



// Load reads configuration from environment variables and validates required fields.
// Returns an error if any required variable is absent.
func Load() (*Config, error) {
	cfg := &Config{
		Port:    envOr("SERVER_PORT", "8080"),
		DBURL:   os.Getenv("DATABASE_URL"),
		JWTKey:  os.Getenv("JWT_SECRET"),
		LabsDir: envOr("LABS_DIR", "./labs"),
	}
	if cfg.DBURL == "" {
		return nil, fmt.Errorf("config: DATABASE_URL is required")
	}
	if cfg.JWTKey == "" {
		return nil, fmt.Errorf("config: JWT_SECRET is required")
	}
	return cfg, nil
}

// envOr returns the value of the environment variable key, or fallback if unset.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}