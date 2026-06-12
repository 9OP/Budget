// Package config loads application configuration from environment variables.
package config

import (
	"errors"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// ErrMissingDatabaseURL is returned when DATABASE_URL env var is not set.
var ErrMissingDatabaseURL = errors.New("DATABASE_URL environment variable is required")

const defaultPort = "8080"

const defaultCacheTTL = 5 * time.Minute

// Config holds application configuration loaded from environment variables.
type Config struct {
	Port        string
	DatabaseURL string
	CacheTTL    time.Duration
}

// Load reads configuration from environment variables and returns a Config.
// It first loads .env from the working directory if present; environment
// variables already set in the shell take priority over .env values.
func Load() (Config, error) {
	_ = godotenv.Load() // silently ignore missing .env

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return Config{}, ErrMissingDatabaseURL
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	cacheTTL := defaultCacheTTL
	if raw := os.Getenv("CACHE_TTL"); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil {
			cacheTTL = d
		}
	}

	return Config{
		Port:        port,
		DatabaseURL: databaseURL,
		CacheTTL:    cacheTTL,
	}, nil
}
