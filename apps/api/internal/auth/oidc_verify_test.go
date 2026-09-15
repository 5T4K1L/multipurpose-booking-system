package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

func TestOIDCVerifierAcceptsValidSignedToken(t *testing.T) {
	signerKey := mustGenerateRSAKey(t)
	signer, err := jose.NewSigner(
		jose.SigningKey{
			Algorithm: jose.RS256,
			Key:       signerKey,
		},
		(&jose.SignerOptions{
			ExtraHeaders: map[jose.HeaderKey]interface{}{
				"kid": "test-key",
			},
		}).WithType("JWT"),
	)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}

	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)

	token, err := jwt.Signed(signer).
		Claims(OIDCClaims{
			Issuer:    "https://issuer.example.com",
			Subject:   "provider-user-123",
			Audience:  OIDCAudience{"client-id"},
			Nonce:     "nonce-123",
			IssuedAt:  now.Add(-time.Minute).Unix(),
			ExpiresAt: now.Add(5 * time.Minute).Unix(),
		}).
		Serialize()
	if err != nil {
		t.Fatalf("create signed token: %v", err)
	}

	keySet := jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{
			{
				Key:       signerKey.Public(),
				KeyID:     "test-key",
				Algorithm: string(jose.RS256),
				Use:       "sig",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(keySet); err != nil {
			t.Fatalf("encode JWK set: %v", err)
		}
	}))
	defer server.Close()

	verifier, err := NewOIDCVerifier(
		server.Client(),
		server.URL,
		"https://issuer.example.com",
		"client-id",
		"nonce-123",
		[]jose.SignatureAlgorithm{jose.RS256},
	)
	if err != nil {
		t.Fatalf("create verifier: %v", err)
	}

	claims, err := verifier.Verify(
		context.Background(),
		token,
		now,
	)
	if err != nil {
		t.Fatalf("verify valid token: %v", err)
	}

	if claims.Subject != "provider-user-123" {
		t.Fatalf("unexpected subject: %q", claims.Subject)
	}
}

func TestOIDCVerifierRejectsWrongSigningKey(t *testing.T) {
	signingKey := mustGenerateRSAKey(t)
	differentKey := mustGenerateRSAKey(t)

	signer, err := jose.NewSigner(
		jose.SigningKey{
			Algorithm: jose.RS256,
			Key:       signingKey,
		},
		(&jose.SignerOptions{
			ExtraHeaders: map[jose.HeaderKey]interface{}{
				"kid": "test-key",
			},
		}).WithType("JWT"),
	)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}

	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)

	token, err := jwt.Signed(signer).
		Claims(OIDCClaims{
			Issuer:    "https://issuer.example.com",
			Subject:   "provider-user-123",
			Audience:  OIDCAudience{"client-id"},
			Nonce:     "nonce-123",
			IssuedAt:  now.Add(-time.Minute).Unix(),
			ExpiresAt: now.Add(5 * time.Minute).Unix(),
		}).
		Serialize()
	if err != nil {
		t.Fatalf("create signed token: %v", err)
	}

	keySet := jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{
			{
				Key:       differentKey.Public(),
				KeyID:     "test-key",
				Algorithm: string(jose.RS256),
				Use:       "sig",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(keySet)
	}))
	defer server.Close()

	verifier, err := NewOIDCVerifier(
		server.Client(),
		server.URL,
		"https://issuer.example.com",
		"client-id",
		"nonce-123",
		[]jose.SignatureAlgorithm{jose.RS256},
	)
	if err != nil {
		t.Fatalf("create verifier: %v", err)
	}

	_, err = verifier.Verify(
		context.Background(),
		token,
		now,
	)
	if err == nil {
		t.Fatal("expected signature verification failure")
	}
}

func TestOIDCClaimsAcceptsMultipleAudiencesWithCorrectAuthorizedParty(t *testing.T) {
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)

	claims := OIDCClaims{
		Issuer:          "https://issuer.example.com",
		Subject:         "provider-user-123",
		Audience:        OIDCAudience{"another-client", "client-id"},
		AuthorizedParty: "client-id",
		Nonce:           "nonce-123",
		IssuedAt:        now.Add(-time.Minute).Unix(),
		ExpiresAt:       now.Add(5 * time.Minute).Unix(),
	}

	if err := claims.Validate(
		"https://issuer.example.com",
		"client-id",
		"nonce-123",
		now,
	); err != nil {
		t.Fatalf("expected valid claims, got: %v", err)
	}
}

func TestOIDCClaimsRejectsMultipleAudiencesWithWrongAuthorizedParty(t *testing.T) {
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)

	claims := OIDCClaims{
		Issuer:          "https://issuer.example.com",
		Subject:         "provider-user-123",
		Audience:        OIDCAudience{"another-client", "client-id"},
		AuthorizedParty: "wrong-client",
		Nonce:           "nonce-123",
		IssuedAt:        now.Add(-time.Minute).Unix(),
		ExpiresAt:       now.Add(5 * time.Minute).Unix(),
	}

	if err := claims.Validate(
		"https://issuer.example.com",
		"client-id",
		"nonce-123",
		now,
	); err == nil {
		t.Fatal("expected authorized-party validation failure")
	}
}

func mustGenerateRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}

	return key
}
