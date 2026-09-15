package auth

import (
	"testing"
	"time"
)

func validOIDCClaims() OIDCClaims {
	return OIDCClaims{
		Issuer:        "https://issuer.example.com",
		Subject:       "provider-user-123",
		Audience:      []string{"client-123"},
		Email:         "user@example.com",
		EmailVerified: true,
		Nonce:         "nonce-123",
		IssuedAt:      time.Date(2026, 9, 15, 11, 55, 0, 0, time.UTC).Unix(),
		ExpiresAt:     time.Date(2026, 9, 15, 12, 5, 0, 0, time.UTC).Unix(),
	}
}

func TestOIDCClaimsValidate(t *testing.T) {
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		mutate  func(*OIDCClaims)
		wantErr bool
	}{
		{
			name: "valid claims",
		},
		{
			name: "wrong issuer",
			mutate: func(c *OIDCClaims) {
				c.Issuer = "https://attacker.example.com"
			},
			wantErr: true,
		},
		{
			name: "wrong audience",
			mutate: func(c *OIDCClaims) {
				c.Audience = []string{"different-client"}
			},
			wantErr: true,
		},
		{
			name: "missing subject",
			mutate: func(c *OIDCClaims) {
				c.Subject = ""
			},
			wantErr: true,
		},
		{
			name: "wrong nonce",
			mutate: func(c *OIDCClaims) {
				c.Nonce = "wrong-nonce"
			},
			wantErr: true,
		},
		{
			name: "future issued at",
			mutate: func(c *OIDCClaims) {
				c.IssuedAt = now.Add(time.Minute).Unix()
			},
			wantErr: true,
		},
		{
			name: "expired token",
			mutate: func(c *OIDCClaims) {
				c.ExpiresAt = now.Add(-time.Second).Unix()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := validOIDCClaims()

			if tt.mutate != nil {
				tt.mutate(&claims)
			}

			err := claims.Validate(
				"https://issuer.example.com",
				"client-123",
				"nonce-123",
				now,
			)

			if tt.wantErr && err == nil {
				t.Fatal("expected validation error")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("expected no validation error, got: %v", err)
			}
		})
	}
}
