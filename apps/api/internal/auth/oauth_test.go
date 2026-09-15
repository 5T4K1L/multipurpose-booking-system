package auth

import (
	"testing"
	"time"
)

func TestOAuthProviderValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  OAuthProviderConfig
		wantErr bool
	}{
		{
			name: "valid Google configuration",
			config: OAuthProviderConfig{
				Provider:         OAuthProviderGoogle,
				ClientID:         "google-client-id",
				ClientSecret:     "google-client-secret",
				RedirectURL:      "https://example.com/api/v1/auth/google/callback",
				AuthorizationURL: "https://accounts.google.com/o/oauth2/v2/auth",
				TokenURL:         "https://oauth2.googleapis.com/token",
				Issuer:           "https://accounts.google.com",
				JWKSURL:          "https://www.googleapis.com/oauth2/v3/certs",
				Scopes: []string{
					"openid",
					"email",
					"profile",
				},
			},
		},
		{
			name: "valid Apple configuration",
			config: OAuthProviderConfig{
				Provider:         OAuthProviderApple,
				ClientID:         "com.example.booking",
				RedirectURL:      "https://example.com/api/v1/auth/apple/callback",
				AuthorizationURL: "https://appleid.apple.com/auth/authorize",
				TokenURL:         "https://appleid.apple.com/auth/token",
				Issuer:           "https://appleid.apple.com",
				JWKSURL:          "https://appleid.apple.com/auth/keys",
				AppleTeamID:      "TEAM123456",
				AppleKeyID:       "KEY1234567",
				ApplePrivateKeyPEM: []byte(`-----BEGIN PRIVATE KEY-----
test-key-material
-----END PRIVATE KEY-----`),
				AppleSecretValidity: time.Hour,
				Scopes: []string{
					"openid",
					"email",
					"name",
				},
			},
		},
		{
			name: "unsupported provider",
			config: OAuthProviderConfig{
				Provider:         OAuthProvider("github"),
				ClientID:         "client-id",
				ClientSecret:     "client-secret",
				RedirectURL:      "https://example.com/callback",
				AuthorizationURL: "https://example.com/authorize",
				TokenURL:         "https://example.com/token",
				Issuer:           "https://example.com",
			},
			wantErr: true,
		},
		{
			name: "missing client ID",
			config: OAuthProviderConfig{
				Provider:         OAuthProviderGoogle,
				ClientSecret:     "google-client-secret",
				RedirectURL:      "https://example.com/callback",
				AuthorizationURL: "https://accounts.google.com/o/oauth2/v2/auth",
				TokenURL:         "https://oauth2.googleapis.com/token",
				Issuer:           "https://accounts.google.com",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantErr && err == nil {
				t.Fatal("expected validation error")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("expected no validation error, got: %v", err)
			}
		})
	}
}
