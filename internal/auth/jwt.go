package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// ErrInvalidToken is returned when the JWT is missing, expired, or has an invalid signature.
var ErrInvalidToken = errors.New("invalid token")

// Validator validates Supabase JWTs using the project's JWKS endpoint.
// Keys are cached and refreshed automatically in the background.
type Validator struct {
	cache   *jwk.Cache
	jwksURL string
}

// NewValidator creates a Validator that fetches and caches public keys from jwksURL.
// The provided context controls the background key-refresh goroutine lifetime.
func NewValidator(ctx context.Context, jwksURL string) (*Validator, error) {
	cache := jwk.NewCache(ctx)

	if err := cache.Register(jwksURL); err != nil {
		return nil, fmt.Errorf("register jwks url: %w", err)
	}

	// Pre-fetch to fail fast if the endpoint is unreachable.
	if _, err := cache.Refresh(ctx, jwksURL); err != nil {
		return nil, fmt.Errorf("fetch jwks: %w", err)
	}

	return &Validator{cache: cache, jwksURL: jwksURL}, nil
}

// ValidateToken validates a Supabase JWT and returns the user ID (sub claim).
func (v *Validator) ValidateToken(ctx context.Context, tokenStr string) (string, error) {
	keySet, err := v.cache.Get(ctx, v.jwksURL)
	if err != nil {
		return "", fmt.Errorf("get jwks: %w", err)
	}

	token, err := jwt.Parse([]byte(tokenStr), jwt.WithKeySet(keySet))
	if err != nil {
		return "", ErrInvalidToken
	}

	sub := token.Subject()
	if sub == "" {
		return "", ErrInvalidToken
	}

	return sub, nil
}
