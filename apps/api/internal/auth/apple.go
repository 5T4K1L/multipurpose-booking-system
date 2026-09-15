package auth

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

// AppleClientSecretConfig contains the values required to create Apple's
// server-side client secret.
type AppleClientSecretConfig struct {
	TeamID        string
	KeyID         string
	ClientID      string
	PrivateKeyPEM []byte
	Validity      time.Duration
}

// Validate checks Apple's client-secret configuration.
func (c AppleClientSecretConfig) Validate() error {
	if c.TeamID == "" {
		return fmt.Errorf("Apple Team ID is required")
	}

	if c.KeyID == "" {
		return fmt.Errorf("Apple Key ID is required")
	}

	if c.ClientID == "" {
		return fmt.Errorf("Apple client ID is required")
	}

	if len(c.PrivateKeyPEM) == 0 {
		return fmt.Errorf("Apple private key is required")
	}

	if c.Validity <= 0 {
		return fmt.Errorf("Apple client secret validity must be positive")
	}

	// Apple currently permits client secrets for up to six months.
	if c.Validity > 180*24*time.Hour {
		return fmt.Errorf("Apple client secret validity cannot exceed 180 days")
	}

	return nil
}

// GenerateAppleClientSecret creates a signed client-secret JWT.
func GenerateAppleClientSecret(
	config AppleClientSecretConfig,
	now time.Time,
) (string, error) {
	if err := config.Validate(); err != nil {
		return "", err
	}

	block, _ := pem.Decode(config.PrivateKeyPEM)
	if block == nil {
		return "", fmt.Errorf("invalid Apple private key PEM")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parse Apple private key: %w", err)
	}

	privateKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return "", fmt.Errorf("Apple private key must be an EC private key")
	}

	signer, err := jose.NewSigner(
		jose.SigningKey{
			Algorithm: jose.ES256,
			Key:       privateKey,
		},
		(&jose.SignerOptions{
			ExtraHeaders: map[jose.HeaderKey]interface{}{
				"kid": config.KeyID,
			},
		}).WithType("JWT"),
	)
	if err != nil {
		return "", fmt.Errorf("create Apple client-secret signer: %w", err)
	}

	claims := jwt.Claims{
		Issuer:   config.TeamID,
		Subject:  config.ClientID,
		Audience: jwt.Audience{"https://appleid.apple.com"},
		IssuedAt: jwt.NewNumericDate(now),
		Expiry: jwt.NewNumericDate(
			now.Add(config.Validity),
		),
	}

	token, err := jwt.Signed(signer).
		Claims(claims).
		Serialize()
	if err != nil {
		return "", fmt.Errorf("sign Apple client secret: %w", err)
	}

	return token, nil
}
