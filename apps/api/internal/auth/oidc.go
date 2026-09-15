package auth

import (
	"encoding/json"
	"fmt"
	"time"
)

// OIDCClaims contains the claims accepted from a validated OIDC ID token.
type OIDCClaims struct {
	Issuer          string       `json:"iss"`
	Subject         string       `json:"sub"`
	Audience        OIDCAudience `json:"aud"`
	AuthorizedParty string       `json:"azp,omitempty"`
	Email           string       `json:"email,omitempty"`
	EmailVerified   bool         `json:"email_verified,omitempty"`
	Nonce           string       `json:"nonce,omitempty"`
	IssuedAt        int64        `json:"iat"`
	ExpiresAt       int64        `json:"exp"`
	NotBefore       *int64       `json:"nbf,omitempty"`
}

// OIDCAudience accepts both the OIDC string and array forms.
type OIDCAudience []string

// UnmarshalJSON supports aud being either a string or an array.
func (a *OIDCAudience) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*a = OIDCAudience{single}
		return nil
	}

	var multiple []string
	if err := json.Unmarshal(data, &multiple); err != nil {
		return fmt.Errorf("invalid OIDC audience: %w", err)
	}

	*a = OIDCAudience(multiple)
	return nil
}

// Validate performs provider-independent OIDC claim validation.
// Signature verification must happen before this method is called.
func (c OIDCClaims) Validate(
	expectedIssuer string,
	expectedAudience string,
	expectedNonce string,
	now time.Time,
) error {
	if c.Issuer == "" || c.Issuer != expectedIssuer {
		return fmt.Errorf("invalid OIDC issuer")
	}

	if c.Subject == "" {
		return fmt.Errorf("missing OIDC subject")
	}

	if !containsString(c.Audience, expectedAudience) {
		return fmt.Errorf("invalid OIDC audience")
	}

	// When multiple audiences are present, azp must identify this client.
	if len(c.Audience) > 1 && c.AuthorizedParty != expectedAudience {
		return fmt.Errorf("invalid OIDC authorized party")
	}

	if expectedNonce == "" || c.Nonce == "" || c.Nonce != expectedNonce {
		return fmt.Errorf("invalid OIDC nonce")
	}

	if c.IssuedAt <= 0 {
		return fmt.Errorf("missing OIDC issued-at timestamp")
	}

	if c.ExpiresAt <= 0 {
		return fmt.Errorf("missing OIDC expiration timestamp")
	}

	nowUnix := now.Unix()

	if c.IssuedAt > nowUnix {
		return fmt.Errorf("OIDC token issued in the future")
	}

	if c.ExpiresAt <= nowUnix {
		return fmt.Errorf("OIDC token has expired")
	}

	if c.NotBefore != nil && *c.NotBefore > nowUnix {
		return fmt.Errorf("OIDC token is not active yet")
	}

	return nil
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}

	return false
}
