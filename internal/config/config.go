// Package config loads application configuration from environment variables.
package config

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

// ErrMissingDatabaseURL is returned when DATABASE_URL env var is not set.
var ErrMissingDatabaseURL = errors.New("DATABASE_URL environment variable is required")

const defaultPort = "8080"

// Config holds application configuration loaded from environment variables.
type Config struct {
	Port        string
	DatabaseURL string
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

	return Config{
		Port:        port,
		DatabaseURL: databaseURL,
	}, nil
}
