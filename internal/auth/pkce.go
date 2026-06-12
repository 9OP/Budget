package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

const verifierBytes = 32

// GenerateVerifier creates a cryptographically random PKCE code verifier.
func GenerateVerifier() (string, error) {
	b := make([]byte, verifierBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate verifier: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}

// ComputeChallenge returns the S256 code_challenge for a given verifier.
func ComputeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}
