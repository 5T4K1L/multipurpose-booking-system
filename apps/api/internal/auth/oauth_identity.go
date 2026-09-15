package auth

import "fmt"

// NormalizeOIDCIdentity converts already-validated OIDC claims into the
// application's provider-neutral OAuth identity model.
func NormalizeOIDCIdentity(
	provider OAuthProvider,
	claims OIDCClaims,
) (OAuthIdentity, error) {
	switch provider {
	case OAuthProviderGoogle, OAuthProviderApple:
	default:
		return OAuthIdentity{}, fmt.Errorf("unsupported OAuth provider: %q", provider)
	}

	if claims.Subject == "" {
		return OAuthIdentity{}, fmt.Errorf("OAuth provider subject is required")
	}

	return OAuthIdentity{
		Provider:        provider,
		ProviderSubject: claims.Subject,
		Email:           claims.Email,
		EmailVerified:   claims.EmailVerified,
	}, nil
}
