package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// OAuthCallbackResult contains a fully validated external identity.
type OAuthCallbackResult struct {
	Identity OAuthIdentity
}

// ProcessOAuthCallback validates and processes an OAuth callback up to the
// point where the external identity is trusted.
func ProcessOAuthCallback(
	ctx context.Context,
	r *http.Request,
	transactionKey []byte,
	config OAuthProviderConfig,
	now time.Time,
) (OAuthCallbackResult, error) {
	if r == nil {
		return OAuthCallbackResult{}, fmt.Errorf("OAuth callback request is required")
	}

	transactionCookie, err := GetOAuthTransactionCookie(r)
	if err != nil {
		return OAuthCallbackResult{}, fmt.Errorf(
			"get OAuth transaction: %w",
			err,
		)
	}

	codec, err := NewOAuthTransactionCodec(transactionKey)
	if err != nil {
		return OAuthCallbackResult{}, fmt.Errorf(
			"create OAuth transaction codec: %w",
			err,
		)
	}

	transaction, err := codec.Decode(
		transactionCookie,
		now,
	)
	if err != nil {
		return OAuthCallbackResult{}, fmt.Errorf(
			"decode OAuth transaction: %w",
			err,
		)
	}

	if transaction.Provider != config.Provider {
		return OAuthCallbackResult{}, fmt.Errorf(
			"OAuth provider mismatch",
		)
	}

	callbackState := r.URL.Query().Get("state")
	if !ValidateOAuthState(
		transaction.State,
		callbackState,
	) {
		return OAuthCallbackResult{}, fmt.Errorf(
			"invalid OAuth state",
		)
	}

	callbackError := r.URL.Query().Get("error")
	if callbackError != "" {
		return OAuthCallbackResult{}, fmt.Errorf(
			"OAuth provider returned an error",
		)
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		return OAuthCallbackResult{}, fmt.Errorf(
			"OAuth authorization code is missing",
		)
	}

	tokenResult, err := ExchangeOAuthCode(
		ctx,
		config,
		code,
		transaction.PKCEVerifier,
	)
	if err != nil {
		return OAuthCallbackResult{}, fmt.Errorf(
			"exchange OAuth authorization code: %w",
			err,
		)
	}

	var identity OAuthIdentity

	switch config.Provider {
	case OAuthProviderGoogle:
		identity, err = ValidateGoogleIDToken(
			ctx,
			config.ClientID,
			tokenResult.IDToken,
			transaction.Nonce,
			now,
		)

	case OAuthProviderApple:
		identity, err = ValidateAppleIDToken(
			ctx,
			config.ClientID,
			tokenResult.IDToken,
			transaction.Nonce,
			now,
		)

	default:
		return OAuthCallbackResult{}, fmt.Errorf(
			"unsupported OAuth provider: %q",
			config.Provider,
		)
	}

	if err != nil {
		return OAuthCallbackResult{}, fmt.Errorf(
			"validate OAuth identity: %w",
			err,
		)
	}

	return OAuthCallbackResult{
		Identity: identity,
	}, nil
}

// ClearOAuthCallbackCookie clears the completed OAuth transaction.
func ClearOAuthCallbackCookie(
	w http.ResponseWriter,
	secure bool,
) {
	ClearOAuthTransactionCookie(w, secure)
}
