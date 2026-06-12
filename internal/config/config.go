// Package config loads application configuration from environment variables.
package config

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

// ErrMissingDatabaseURL is returned when DATABASE_URL env var is not set.
var ErrMissingDatabaseURL = errors.New("DATABASE_URL environment variable is required")

// ErrMissingSupabaseURL is returned when SUPABASE_URL env var is not set.
var ErrMissingSupabaseURL = errors.New("SUPABASE_URL environment variable is required")

// ErrMissingSupabaseAnonKey is returned when SUPABASE_ANON_KEY env var is not set.
var ErrMissingSupabaseAnonKey = errors.New("SUPABASE_ANON_KEY environment variable is required")

// ErrMissingJWTSecret is returned when SUPABASE_JWT_SECRET env var is not set.
var ErrMissingJWTSecret = errors.New("SUPABASE_JWT_SECRET environment variable is required")

// ErrMissingAppURL is returned when APP_URL env var is not set.
var ErrMissingAppURL = errors.New("APP_URL environment variable is required")

const defaultPort = "8080"

// Config holds application configuration loaded from environment variables.
type Config struct {
	Port            string
	DatabaseURL     string
	SupabaseURL     string
	SupabaseAnonKey string
	JWTSecret       string
	AppURL          string
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

	supabaseURL := os.Getenv("SUPABASE_URL")
	if supabaseURL == "" {
		return Config{}, ErrMissingSupabaseURL
	}

	supabaseAnonKey := os.Getenv("SUPABASE_ANON_KEY")
	if supabaseAnonKey == "" {
		return Config{}, ErrMissingSupabaseAnonKey
	}

	jwtSecret := os.Getenv("SUPABASE_JWT_SECRET")
	if jwtSecret == "" {
		return Config{}, ErrMissingJWTSecret
	}

	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		return Config{}, ErrMissingAppURL
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	return Config{
		Port:            port,
		DatabaseURL:     databaseURL,
		SupabaseURL:     supabaseURL,
		SupabaseAnonKey: supabaseAnonKey,
		JWTSecret:       jwtSecret,
		AppURL:          appURL,
	}, nil
}
