package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

// OIDCVerifier verifies signed OIDC ID tokens using a provider JWK set.
type OIDCVerifier struct {
	httpClient *http.Client
	jwksURL    string
	issuer     string
	audience   string
	nonce      string
	algorithms []jose.SignatureAlgorithm
}

// NewOIDCVerifier creates an ID-token verifier.
func NewOIDCVerifier(
	httpClient *http.Client,
	jwksURL string,
	issuer string,
	audience string,
	nonce string,
	algorithms []jose.SignatureAlgorithm,
) (*OIDCVerifier, error) {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 10 * time.Second,
		}
	}

	if jwksURL == "" {
		return nil, fmt.Errorf("JWKS URL is required")
	}

	if issuer == "" {
		return nil, fmt.Errorf("OIDC issuer is required")
	}

	if audience == "" {
		return nil, fmt.Errorf("OIDC audience is required")
	}

	if nonce == "" {
		return nil, fmt.Errorf("OIDC nonce is required")
	}

	if len(algorithms) == 0 {
		return nil, fmt.Errorf("at least one signing algorithm is required")
	}

	return &OIDCVerifier{
		httpClient: httpClient,
		jwksURL:    jwksURL,
		issuer:     issuer,
		audience:   audience,
		nonce:      nonce,
		algorithms: algorithms,
	}, nil
}

// Verify validates the signature and OIDC claims of an ID token.
func (v *OIDCVerifier) Verify(
	ctx context.Context,
	rawToken string,
	now time.Time,
) (OIDCClaims, error) {
	if rawToken == "" {
		return OIDCClaims{}, fmt.Errorf("ID token is required")
	}

	token, err := jwt.ParseSigned(rawToken, v.algorithms)
	if err != nil {
		return OIDCClaims{}, fmt.Errorf("parse signed ID token: %w", err)
	}

	if len(token.Headers) != 1 {
		return OIDCClaims{}, fmt.Errorf("unexpected ID token signature count")
	}

	keyID := token.Headers[0].KeyID
	if keyID == "" {
		return OIDCClaims{}, fmt.Errorf("ID token key ID is missing")
	}

	keySet, err := v.fetchJWKS(ctx)
	if err != nil {
		return OIDCClaims{}, fmt.Errorf("fetch OIDC signing keys: %w", err)
	}

	keys := keySet.Key(keyID)
	if len(keys) != 1 {
		return OIDCClaims{}, fmt.Errorf("OIDC signing key not found or ambiguous")
	}

	var claims OIDCClaims
	if err := token.Claims(keys[0], &claims); err != nil {
		return OIDCClaims{}, fmt.Errorf("verify ID token signature and claims: %w", err)
	}

	if err := claims.Validate(
		v.issuer,
		v.audience,
		v.nonce,
		now,
	); err != nil {
		return OIDCClaims{}, fmt.Errorf("validate OIDC claims: %w", err)
	}

	return claims, nil
}

func (v *OIDCVerifier) fetchJWKS(ctx context.Context) (jose.JSONWebKeySet, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return jose.JSONWebKeySet{}, fmt.Errorf("create JWKS request: %w", err)
	}

	response, err := v.httpClient.Do(request)
	if err != nil {
		return jose.JSONWebKeySet{}, fmt.Errorf("request JWKS: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return jose.JSONWebKeySet{}, fmt.Errorf(
			"JWKS endpoint returned HTTP %d",
			response.StatusCode,
		)
	}

	var keySet jose.JSONWebKeySet
	if err := json.NewDecoder(response.Body).Decode(&keySet); err != nil {
		return jose.JSONWebKeySet{}, fmt.Errorf("decode JWKS: %w", err)
	}

	if len(keySet.Keys) == 0 {
		return jose.JSONWebKeySet{}, fmt.Errorf("JWKS contains no keys")
	}

	return keySet, nil
}
