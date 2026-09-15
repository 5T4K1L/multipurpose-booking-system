package auth

import "testing"

func TestNormalizeOIDCIdentity(t *testing.T) {
	tests := []struct {
		name     string
		provider OAuthProvider
		claims   OIDCClaims
		wantErr  bool
	}{
		{
			name:     "Google identity",
			provider: OAuthProviderGoogle,
			claims: OIDCClaims{
				Subject:       "google-subject-123",
				Email:         "user@example.com",
				EmailVerified: true,
			},
		},
		{
			name:     "Apple identity",
			provider: OAuthProviderApple,
			claims: OIDCClaims{
				Subject:       "apple-subject-123",
				Email:         "private-relay@example.com",
				EmailVerified: true,
			},
		},
		{
			name:     "missing subject",
			provider: OAuthProviderGoogle,
			claims: OIDCClaims{
				Email:         "user@example.com",
				EmailVerified: true,
			},
			wantErr: true,
		},
		{
			name:     "unsupported provider",
			provider: OAuthProvider("github"),
			claims: OIDCClaims{
				Subject: "github-subject",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identity, err := NormalizeOIDCIdentity(
				tt.provider,
				tt.claims,
			)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if identity.Provider != tt.provider {
				t.Fatalf(
					"expected provider %q, got %q",
					tt.provider,
					identity.Provider,
				)
			}

			if identity.ProviderSubject != tt.claims.Subject {
				t.Fatalf(
					"expected subject %q, got %q",
					tt.claims.Subject,
					identity.ProviderSubject,
				)
			}

			if identity.Email != tt.claims.Email {
				t.Fatalf(
					"expected email %q, got %q",
					tt.claims.Email,
					identity.Email,
				)
			}

			if identity.EmailVerified != tt.claims.EmailVerified {
				t.Fatal("unexpected email verification status")
			}
		})
	}
}
