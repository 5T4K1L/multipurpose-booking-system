package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

func generateTestApplePrivateKeyPEM(t *testing.T) []byte {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate EC key: %v", err)
	}

	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: der,
	})
}

func TestGenerateAppleClientSecret(t *testing.T) {
	now := time.Date(
		2026,
		9,
		15,
		12,
		0,
		0,
		0,
		time.UTC,
	)

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate EC key: %v", err)
	}

	der, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}

	config := AppleClientSecretConfig{
		TeamID:   "TEAM123456",
		KeyID:    "KEY1234567",
		ClientID: "com.example.booking.web",
		PrivateKeyPEM: pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: der,
		}),
		Validity: time.Hour,
	}

	token, err := GenerateAppleClientSecret(config, now)
	if err != nil {
		t.Fatalf("generate Apple client secret: %v", err)
	}

	parsed, err := jwt.ParseSigned(
		token,
		[]jose.SignatureAlgorithm{jose.ES256},
	)
	if err != nil {
		t.Fatalf("parse Apple client secret: %v", err)
	}

	if len(parsed.Headers) != 1 {
		t.Fatalf("expected one JWT header, got %d", len(parsed.Headers))
	}

	if parsed.Headers[0].KeyID != config.KeyID {
		t.Fatalf(
			"expected key ID %q, got %q",
			config.KeyID,
			parsed.Headers[0].KeyID,
		)
	}

	var claims jwt.Claims
	if err := parsed.Claims(privateKey.Public(), &claims); err != nil {
		t.Fatalf("verify Apple client-secret signature: %v", err)
	}

	if claims.Issuer != config.TeamID {
		t.Fatalf("expected issuer %q, got %q", config.TeamID, claims.Issuer)
	}

	if claims.Subject != config.ClientID {
		t.Fatalf("expected subject %q, got %q", config.ClientID, claims.Subject)
	}
}

func TestGenerateAppleClientSecretRejectsLongValidity(t *testing.T) {
	config := AppleClientSecretConfig{
		TeamID:        "TEAM123456",
		KeyID:         "KEY1234567",
		ClientID:      "com.example.booking.web",
		PrivateKeyPEM: generateTestApplePrivateKeyPEM(t),
		Validity:      181 * 24 * time.Hour,
	}

	_, err := GenerateAppleClientSecret(
		config,
		time.Now(),
	)

	if err == nil {
		t.Fatal("expected maximum-validity error")
	}
}

func TestGenerateAppleClientSecretRejectsInvalidKey(t *testing.T) {
	config := AppleClientSecretConfig{
		TeamID:        "TEAM123456",
		KeyID:         "KEY1234567",
		ClientID:      "com.example.booking.web",
		PrivateKeyPEM: []byte("not-a-private-key"),
		Validity:      time.Hour,
	}

	_, err := GenerateAppleClientSecret(
		config,
		time.Now(),
	)

	if err == nil {
		t.Fatal("expected private-key error")
	}

	if !strings.Contains(err.Error(), "private key") {
		t.Fatalf("unexpected error: %v", err)
	}
}
