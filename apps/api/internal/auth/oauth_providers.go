package auth

import "time"

// GoogleOAuthConfig returns the static Google OIDC configuration.
// Runtime credentials are supplied separately.
func GoogleOAuthConfig(
	clientID string,
	clientSecret string,
	redirectURL string,
) OAuthProviderConfig {
	return OAuthProviderConfig{
		Provider:         OAuthProviderGoogle,
		ClientID:         clientID,
		ClientSecret:     clientSecret,
		RedirectURL:      redirectURL,
		AuthorizationURL: "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:         "https://oauth2.googleapis.com/token",
		Issuer:           "https://accounts.google.com",
		JWKSURL:          "https://www.googleapis.com/oauth2/v3/certs",
		Scopes: []string{
			"openid",
			"email",
			"profile",
		},
	}
}

// AppleOAuthConfig returns the static Apple OIDC configuration.
// Runtime credentials are supplied separately.
func AppleOAuthConfig(
	clientID string,
	teamID string,
	keyID string,
	privateKeyPEM []byte,
	redirectURL string,
) OAuthProviderConfig {
	return OAuthProviderConfig{
		Provider:            OAuthProviderApple,
		ClientID:            clientID,
		RedirectURL:         redirectURL,
		AuthorizationURL:    "https://appleid.apple.com/auth/authorize",
		TokenURL:            "https://appleid.apple.com/auth/token",
		Issuer:              "https://appleid.apple.com",
		JWKSURL:             "https://appleid.apple.com/auth/keys",
		AppleTeamID:         teamID,
		AppleKeyID:          keyID,
		ApplePrivateKeyPEM:  privateKeyPEM,
		AppleSecretValidity: 24 * time.Hour,
		Scopes: []string{
			"openid",
			"email",
			"name",
		},
	}
}
