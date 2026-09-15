package auth

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestBuildOAuthAuthorization(t *testing.T) {
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)

	config := GoogleOAuthConfig(
		"google-client-id",
		"google-client-secret",
		"https://example.com/api/v1/auth/google/callback",
	)

	authorization, err := BuildOAuthAuthorization(
		context.Background(),
		config,
		now,
	)
	if err != nil {
		t.Fatalf("build OAuth authorization: %v", err)
	}

	if authorization.URL == "" {
		t.Fatal("expected authorization URL")
	}

	if authorization.State.State == "" {
		t.Fatal("expected OAuth state")
	}

	if authorization.State.Nonce == "" {
		t.Fatal("expected OAuth nonce")
	}

	if authorization.PKCEVerifier == "" {
		t.Fatal("expected PKCE verifier")
	}

	parsed, err := url.Parse(authorization.URL)
	if err != nil {
		t.Fatalf("parse authorization URL: %v", err)
	}

	query := parsed.Query()

	if query.Get("state") != authorization.State.State {
		t.Fatal("authorization URL contains incorrect state")
	}

	if query.Get("nonce") != authorization.State.Nonce {
		t.Fatal("authorization URL contains incorrect nonce")
	}

	if query.Get("code_challenge") == "" {
		t.Fatal("authorization URL missing PKCE challenge")
	}

	if query.Get("code_challenge_method") != "S256" {
		t.Fatalf(
			"expected S256 PKCE method, got %q",
			query.Get("code_challenge_method"),
		)
	}

	if !strings.Contains(query.Get("scope"), "openid") {
		t.Fatalf("expected openid scope, got %q", query.Get("scope"))
	}
}

func TestBuildOAuthAuthorizationRejectsInvalidConfig(t *testing.T) {
	_, err := BuildOAuthAuthorization(
		context.Background(),
		OAuthProviderConfig{},
		time.Now(),
	)

	if err == nil {
		t.Fatal("expected invalid configuration error")
	}
}

func TestValidateAuthorizationURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name: "valid HTTPS URL",
			url:  "https://accounts.example.com/authorize",
		},
		{
			name:    "HTTP rejected",
			url:     "http://accounts.example.com/authorize",
			wantErr: true,
		},
		{
			name:    "missing host",
			url:     "https:///authorize",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAuthorizationURL(tt.url)

			if tt.wantErr && err == nil {
				t.Fatal("expected validation error")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
		})
	}
}
