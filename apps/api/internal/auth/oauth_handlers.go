package auth

import (
	"fmt"
	"net/http"
	"time"
)

// OAuthHTTPConfig contains the runtime dependencies needed by OAuth handlers.
type OAuthHTTPConfig struct {
	Google         OAuthProviderConfig
	Apple          OAuthProviderConfig
	TransactionKey []byte
	SecureCookies  bool
	Now            func() time.Time
}

// OAuthHandlers handles OAuth authorization endpoints.
type OAuthHandlers struct {
	config OAuthHTTPConfig
}

// NewOAuthHandlers creates OAuth HTTP handlers.
func NewOAuthHandlers(config OAuthHTTPConfig) (*OAuthHandlers, error) {
	if len(config.TransactionKey) != oauthTransactionKeyLen {
		return nil, fmt.Errorf(
			"OAuth transaction key must be exactly %d bytes",
			oauthTransactionKeyLen,
		)
	}

	if err := config.Google.Validate(); err != nil {
		return nil, fmt.Errorf("Google OAuth configuration: %w", err)
	}

	if err := config.Apple.Validate(); err != nil {
		return nil, fmt.Errorf("Apple OAuth configuration: %w", err)
	}

	if config.Now == nil {
		config.Now = time.Now
	}

	return &OAuthHandlers{
		config: config,
	}, nil
}

// GoogleStart begins the Google OAuth flow.
func (h *OAuthHandlers) GoogleStart(
	w http.ResponseWriter,
	r *http.Request,
) {
	h.startProvider(
		w,
		r,
		h.config.Google,
	)
}

// AppleStart begins the Apple OAuth flow.
func (h *OAuthHandlers) AppleStart(
	w http.ResponseWriter,
	r *http.Request,
) {
	h.startProvider(
		w,
		r,
		h.config.Apple,
	)
}

func (h *OAuthHandlers) startProvider(
	w http.ResponseWriter,
	r *http.Request,
	config OAuthProviderConfig,
) {
	now := h.config.Now()

	authorization, err := BuildOAuthAuthorization(
		r.Context(),
		config,
		now,
	)
	if err != nil {
		http.Error(
			w,
			"OAuth authorization could not be started",
			http.StatusInternalServerError,
		)
		return
	}

	if err := ValidateAuthorizationURL(authorization.URL); err != nil {
		http.Error(
			w,
			"OAuth authorization could not be started",
			http.StatusInternalServerError,
		)
		return
	}

	transaction := OAuthTransaction{
		Provider:      config.Provider,
		State:         authorization.State.State,
		Nonce:         authorization.State.Nonce,
		PKCEVerifier:  authorization.PKCEVerifier,
		CreatedAtUnix: now.Unix(),
	}

	codec, err := NewOAuthTransactionCodec(
		h.config.TransactionKey,
	)
	if err != nil {
		http.Error(
			w,
			"OAuth authorization could not be started",
			http.StatusInternalServerError,
		)
		return
	}

	encodedTransaction, err := codec.Encode(transaction)
	if err != nil {
		http.Error(
			w,
			"OAuth authorization could not be started",
			http.StatusInternalServerError,
		)
		return
	}

	SetOAuthTransactionCookie(
		w,
		encodedTransaction,
		h.config.SecureCookies,
	)

	http.Redirect(
		w,
		r,
		authorization.URL,
		http.StatusFound,
	)
}
