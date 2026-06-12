package auth

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidToken is returned when the JWT is missing, expired, or has an invalid signature.
var ErrInvalidToken = errors.New("invalid token")

// errUnexpectedSigningMethod is returned when the JWT uses a non-HMAC algorithm.
var errUnexpectedSigningMethod = errors.New("unexpected signing method")

// ValidateToken validates a Supabase HS256 JWT and returns the user ID (sub claim).
func ValidateToken(tokenStr, secret string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errUnexpectedSigningMethod
		}

		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !token.Valid {
		return "", ErrInvalidToken
	}

	sub, err := token.Claims.GetSubject()
	if err != nil || sub == "" {
		return "", ErrInvalidToken
	}

	return sub, nil
}
