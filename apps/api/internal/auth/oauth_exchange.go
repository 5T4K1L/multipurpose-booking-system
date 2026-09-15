package auth

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/oauth2"
)

// OAuthTokenResult contains the provider credential needed to verify identity.
// Provider access and refresh tokens are intentionally not persisted here.
type OAuthTokenResult struct {
	IDToken string
}

// ExchangeOAuthCode exchanges a provider authorization code for tokens.
func ExchangeOAuthCode(
	ctx context.Context,
	config OAuthProviderConfig,
	code string,
	pkceVerifier string,
) (OAuthTokenResult, error) {
	if err := config.Validate(); err != nil {
		return OAuthTokenResult{}, fmt.Errorf("validate OAuth configuration: %w", err)
	}

	if code == "" {
		return OAuthTokenResult{}, fmt.Errorf("authorization code is required")
	}

	if pkceVerifier == "" {
		return OAuthTokenResult{}, fmt.Errorf("PKCE verifier is required")
	}

	clientSecret := config.ClientSecret

	if config.Provider == OAuthProviderApple {
		clientSecretValue, err := GenerateAppleClientSecret(
			AppleClientSecretConfig{
				TeamID:        config.AppleTeamID,
				KeyID:         config.AppleKeyID,
				ClientID:      config.ClientID,
				PrivateKeyPEM: config.ApplePrivateKeyPEM,
				Validity:      config.AppleSecretValidity,
			},
			time.Now(),
		)
		if err != nil {
			return OAuthTokenResult{}, fmt.Errorf(
				"generate Apple client secret: %w",
				err,
			)
		}

		clientSecret = clientSecretValue
	}

	oauthConfig := oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  config.AuthorizationURL,
			TokenURL: config.TokenURL,
		},
		RedirectURL: config.RedirectURL,
		Scopes:      config.Scopes,
	}

	token, err := oauthConfig.Exchange(
		ctx,
		code,
		oauth2.VerifierOption(pkceVerifier),
	)
	if err != nil {
		return OAuthTokenResult{}, fmt.Errorf("exchange OAuth authorization code: %w", err)
	}

	idToken, ok := token.Extra("id_token").(string)
	if !ok || idToken == "" {
		return OAuthTokenResult{}, fmt.Errorf("OAuth provider did not return an ID token")
	}

	return OAuthTokenResult{
		IDToken: idToken,
	}, nil
}
