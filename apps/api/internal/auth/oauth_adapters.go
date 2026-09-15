package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-jose/go-jose/v4"
)

// ValidateGoogleIDToken verifies and normalizes a Google ID token.
func ValidateGoogleIDToken(
	ctx context.Context,
	clientID string,
	rawIDToken string,
	expectedNonce string,
	now time.Time,
) (OAuthIdentity, error) {
	if clientID == "" {
		return OAuthIdentity{}, fmt.Errorf("Google client ID is required")
	}

	verifier, err := NewOIDCVerifier(
		&http.Client{Timeout: 10 * time.Second},
		"https://www.googleapis.com/oauth2/v3/certs",
		"https://accounts.google.com",
		clientID,
		expectedNonce,
		[]jose.SignatureAlgorithm{
			jose.RS256,
		},
	)
	if err != nil {
		return OAuthIdentity{}, fmt.Errorf("create Google OIDC verifier: %w", err)
	}

	claims, err := verifier.Verify(
		ctx,
		rawIDToken,
		now,
	)
	if err != nil {
		return OAuthIdentity{}, fmt.Errorf("verify Google ID token: %w", err)
	}

	identity, err := NormalizeOIDCIdentity(
		OAuthProviderGoogle,
		claims,
	)
	if err != nil {
		return OAuthIdentity{}, fmt.Errorf(
			"normalize Google identity: %w",
			err,
		)
	}

	return identity, nil
}

// ValidateAppleIDToken verifies and normalizes an Apple ID token.
func ValidateAppleIDToken(
	ctx context.Context,
	clientID string,
	rawIDToken string,
	expectedNonce string,
	now time.Time,
) (OAuthIdentity, error) {
	if clientID == "" {
		return OAuthIdentity{}, fmt.Errorf("Apple client ID is required")
	}

	verifier, err := NewOIDCVerifier(
		&http.Client{Timeout: 10 * time.Second},
		"https://appleid.apple.com/auth/keys",
		"https://appleid.apple.com",
		clientID,
		expectedNonce,
		[]jose.SignatureAlgorithm{
			jose.ES256,
		},
	)
	if err != nil {
		return OAuthIdentity{}, fmt.Errorf("create Apple OIDC verifier: %w", err)
	}

	claims, err := verifier.Verify(
		ctx,
		rawIDToken,
		now,
	)
	if err != nil {
		return OAuthIdentity{}, fmt.Errorf("verify Apple ID token: %w", err)
	}

	identity, err := NormalizeOIDCIdentity(
		OAuthProviderApple,
		claims,
	)
	if err != nil {
		return OAuthIdentity{}, fmt.Errorf(
			"normalize Apple identity: %w",
			err,
		)
	}

	return identity, nil
}
