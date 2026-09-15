package auth

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"golang.org/x/oauth2"
)

// OAuthAuthorization contains the values needed to begin an OAuth flow.
type OAuthAuthorization struct {
	URL          string
	State        OAuthState
	PKCEVerifier string
}

// BuildOAuthAuthorization creates a secure provider authorization request.
func BuildOAuthAuthorization(
	ctx context.Context,
	config OAuthProviderConfig,
	now time.Time,
) (OAuthAuthorization, error) {
	if err := config.Validate(); err != nil {
		return OAuthAuthorization{}, err
	}

	oauthState, err := GenerateOAuthState(now)
	if err != nil {
		return OAuthAuthorization{}, fmt.Errorf("generate OAuth state: %w", err)
	}

	pkceVerifier := oauth2.GenerateVerifier()

	oauthConfig := oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  config.AuthorizationURL,
			TokenURL: config.TokenURL,
		},
		RedirectURL: config.RedirectURL,
		Scopes:      config.Scopes,
	}

	authURL := oauthConfig.AuthCodeURL(
		oauthState.State,
		oauth2.S256ChallengeOption(pkceVerifier),
		oauth2.SetAuthURLParam("nonce", oauthState.Nonce),
	)

	return OAuthAuthorization{
		URL:          authURL,
		State:        oauthState,
		PKCEVerifier: pkceVerifier,
	}, nil
}

// ValidateAuthorizationURL prevents malformed authorization endpoints from
// being accidentally treated as application redirects.
func ValidateAuthorizationURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parse authorization URL: %w", err)
	}

	if parsed.Scheme != "https" {
		return fmt.Errorf("authorization URL must use HTTPS")
	}

	if parsed.Host == "" {
		return fmt.Errorf("authorization URL host is required")
	}

	return nil
}
