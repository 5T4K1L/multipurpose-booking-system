package auth

import (
	"fmt"
	"time"
)

// OAuthProvider identifies a supported external authentication provider.
type OAuthProvider string

const (
	OAuthProviderGoogle OAuthProvider = "google"
	OAuthProviderApple  OAuthProvider = "apple"
)

// OAuthIdentity represents a validated identity returned by an OAuth provider.
type OAuthIdentity struct {
	Provider        OAuthProvider
	ProviderSubject string
	Email           string
	EmailVerified   bool
	DisplayName     string
}

// OAuthProviderConfig contains the provider configuration used by the
// authorization-code flow.
type OAuthProviderConfig struct {
	Provider         OAuthProvider
	ClientID         string
	ClientSecret     string
	RedirectURL      string
	AuthorizationURL string
	TokenURL         string
	Issuer           string
	JWKSURL          string
	Scopes           []string

	// Apple-only signing configuration.
	AppleTeamID         string
	AppleKeyID          string
	ApplePrivateKeyPEM  []byte
	AppleSecretValidity time.Duration
}

// Validate checks whether the provider configuration is usable.
func (c OAuthProviderConfig) Validate() error {
	switch c.Provider {
	case OAuthProviderGoogle, OAuthProviderApple:
	default:
		return fmt.Errorf("unsupported OAuth provider: %q", c.Provider)
	}

	if c.ClientID == "" {
		return fmt.Errorf("%s OAuth client ID is required", c.Provider)
	}

	if c.Provider == OAuthProviderGoogle && c.ClientSecret == "" {
		return fmt.Errorf("google OAuth client secret is required")
	}

	if c.Provider == OAuthProviderApple {
		if c.AppleTeamID == "" {
			return fmt.Errorf("Apple Team ID is required")
		}

		if c.AppleKeyID == "" {
			return fmt.Errorf("Apple Key ID is required")
		}

		if len(c.ApplePrivateKeyPEM) == 0 {
			return fmt.Errorf("Apple private key is required")
		}

		if c.AppleSecretValidity <= 0 {
			return fmt.Errorf("Apple client secret validity is required")
		}
	}

	if c.RedirectURL == "" {
		return fmt.Errorf("%s OAuth redirect URL is required", c.Provider)
	}

	if c.AuthorizationURL == "" {
		return fmt.Errorf("%s OAuth authorization URL is required", c.Provider)
	}

	if c.TokenURL == "" {
		return fmt.Errorf("%s OAuth token URL is required", c.Provider)
	}

	if c.Issuer == "" {
		return fmt.Errorf("%s OAuth issuer is required", c.Provider)
	}

	if c.JWKSURL == "" {
		return fmt.Errorf("%s OAuth JWKS URL is required", c.Provider)
	}

	if len(c.Scopes) == 0 {
		return fmt.Errorf("%s OAuth scopes are required", c.Provider)
	}

	return nil
}
