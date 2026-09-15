package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"time"
)

const OAuthTransactionKeyEnv = "OAUTH_TRANSACTION_KEY"

// OAuthRuntimeConfig contains runtime-only OAuth security configuration.
type OAuthRuntimeConfig struct {
	TransactionKey []byte

	Google OAuthProviderConfig
	Apple  OAuthProviderConfig
}

// LoadOAuthRuntimeConfig reads OAuth configuration from environment variables.
func LoadOAuthRuntimeConfig() (OAuthRuntimeConfig, error) {
	transactionKey, err := loadTransactionKey()
	if err != nil {
		return OAuthRuntimeConfig{}, err
	}

	google, err := loadGoogleConfig()
	if err != nil {
		return OAuthRuntimeConfig{}, err
	}

	apple, err := loadAppleConfig()
	if err != nil {
		return OAuthRuntimeConfig{}, err
	}

	return OAuthRuntimeConfig{
		TransactionKey: transactionKey,
		Google:         google,
		Apple:          apple,
	}, nil
}

func loadTransactionKey() ([]byte, error) {
	value := os.Getenv(OAuthTransactionKeyEnv)

	if value == "" {
		return nil, fmt.Errorf("%s is required", OAuthTransactionKeyEnv)
	}

	key, err := base64.RawStdEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf(
			"%s must be base64url encoded: %w",
			OAuthTransactionKeyEnv,
			err,
		)
	}

	if len(key) != oauthTransactionKeyLen {
		return nil, fmt.Errorf(
			"%s must decode to exactly %d bytes",
			OAuthTransactionKeyEnv,
			oauthTransactionKeyLen,
		)
	}

	return key, nil
}

// GenerateOAuthTransactionKey creates a cryptographically random transaction key.
// Use this locally to generate a value for environment configuration.
func GenerateOAuthTransactionKey() (string, error) {
	key := make([]byte, oauthTransactionKeyLen)

	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("generate OAuth transaction key: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(key), nil
}

func loadGoogleConfig() (OAuthProviderConfig, error) {
	config := GoogleOAuthConfig(
		os.Getenv("GOOGLE_OAUTH_CLIENT_ID"),
		os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"),
		os.Getenv("GOOGLE_OAUTH_REDIRECT_URL"),
	)

	if err := config.Validate(); err != nil {
		return OAuthProviderConfig{}, fmt.Errorf(
			"Google OAuth configuration: %w",
			err,
		)
	}

	return config, nil
}

func loadAppleConfig() (OAuthProviderConfig, error) {
	config := AppleOAuthConfig(
		os.Getenv("APPLE_OAUTH_CLIENT_ID"),
		os.Getenv("APPLE_TEAM_ID"),
		os.Getenv("APPLE_KEY_ID"),
		[]byte(os.Getenv("APPLE_PRIVATE_KEY_PEM")),
		os.Getenv("APPLE_OAUTH_REDIRECT_URL"),
	)

	if err := config.Validate(); err != nil {
		return OAuthProviderConfig{}, fmt.Errorf(
			"Apple OAuth configuration: %w",
			err,
		)
	}

	return config, nil
}

// AppleClientSecretRuntimeConfig contains values used to generate Apple's
// server-side client secret.
type AppleClientSecretRuntimeConfig struct {
	TeamID        string
	KeyID         string
	PrivateKeyPEM []byte
	Validity      time.Duration
}
