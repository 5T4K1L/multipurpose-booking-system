package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"time"
)

const oauthRandomBytes = 32

// OAuthState contains short-lived values associated with one OAuth attempt.
type OAuthState struct {
	State     string
	Nonce     string
	CreatedAt time.Time
}

// GenerateOAuthState creates unpredictable state and nonce values.
func GenerateOAuthState(now time.Time) (OAuthState, error) {
	state, err := generateOAuthRandomValue()
	if err != nil {
		return OAuthState{}, fmt.Errorf("generate OAuth state: %w", err)
	}

	nonce, err := generateOAuthRandomValue()
	if err != nil {
		return OAuthState{}, fmt.Errorf("generate OAuth nonce: %w", err)
	}

	return OAuthState{
		State:     state,
		Nonce:     nonce,
		CreatedAt: now,
	}, nil
}

// ValidateOAuthState compares the callback state with the original state.
func ValidateOAuthState(expected, received string) bool {
	if expected == "" || received == "" {
		return false
	}

	return subtle.ConstantTimeCompare(
		[]byte(expected),
		[]byte(received),
	) == 1
}

// ValidateOAuthStateAge verifies that an OAuth attempt has not expired.
func ValidateOAuthStateAge(createdAt, now time.Time, maxAge time.Duration) bool {
	if maxAge <= 0 || createdAt.IsZero() {
		return false
	}

	if now.Before(createdAt) {
		return false
	}

	return now.Sub(createdAt) <= maxAge
}

func generateOAuthRandomValue() (string, error) {
	buffer := make([]byte, oauthRandomBytes)

	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buffer), nil
}
