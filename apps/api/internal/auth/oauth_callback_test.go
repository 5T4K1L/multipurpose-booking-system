package auth

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func validOAuthCallbackRequest(
	t *testing.T,
	state string,
) *http.Request {
	t.Helper()

	request := httptest.NewRequest(
		http.MethodGet,
		"https://example.com/api/v1/auth/google/callback?code=test-code&state="+state,
		nil,
	)

	return request
}

func TestProcessOAuthCallbackRequiresTransactionCookie(t *testing.T) {
	request := validOAuthCallbackRequest(
		t,
		"state-123",
	)

	config := GoogleOAuthConfig(
		"google-client-id",
		"google-client-secret",
		"https://example.com/api/v1/auth/google/callback",
	)

	_, err := ProcessOAuthCallback(
		context.Background(),
		request,
		bytes.Repeat([]byte{0x42}, oauthTransactionKeyLen),
		config,
		time.Now(),
	)

	if err == nil {
		t.Fatal("expected missing transaction error")
	}
}

func TestProcessOAuthCallbackRejectsStateMismatch(t *testing.T) {
	now := time.Date(
		2026,
		9,
		15,
		12,
		0,
		0,
		0,
		time.UTC,
	)

	transaction := OAuthTransaction{
		Provider:      OAuthProviderGoogle,
		State:         "expected-state",
		Nonce:         "nonce",
		PKCEVerifier:  "pkce-verifier",
		CreatedAtUnix: now.Unix(),
	}

	key := bytes.Repeat(
		[]byte{0x42},
		oauthTransactionKeyLen,
	)

	codec, err := NewOAuthTransactionCodec(key)
	if err != nil {
		t.Fatalf("create codec: %v", err)
	}

	value, err := codec.Encode(transaction)
	if err != nil {
		t.Fatalf("encode transaction: %v", err)
	}

	request := httptest.NewRequest(
		http.MethodGet,
		"https://example.com/api/v1/auth/google/callback?code=test-code&state=wrong-state",
		nil,
	)

	request.AddCookie(&http.Cookie{
		Name:  oauthTransactionCookieName,
		Value: value,
	})

	config := GoogleOAuthConfig(
		"google-client-id",
		"google-client-secret",
		"https://example.com/api/v1/auth/google/callback",
	)

	_, err = ProcessOAuthCallback(
		context.Background(),
		request,
		key,
		config,
		now.Add(time.Minute),
	)

	if err == nil {
		t.Fatal("expected state mismatch error")
	}
}

func TestProcessOAuthCallbackRejectsProviderMismatch(t *testing.T) {
	now := time.Date(
		2026,
		9,
		15,
		12,
		0,
		0,
		0,
		time.UTC,
	)

	transaction := OAuthTransaction{
		Provider:      OAuthProviderApple,
		State:         "state-123",
		Nonce:         "nonce-123",
		PKCEVerifier:  "pkce-verifier",
		CreatedAtUnix: now.Unix(),
	}

	key := bytes.Repeat(
		[]byte{0x42},
		oauthTransactionKeyLen,
	)

	codec, err := NewOAuthTransactionCodec(key)
	if err != nil {
		t.Fatalf("create codec: %v", err)
	}

	value, err := codec.Encode(transaction)
	if err != nil {
		t.Fatalf("encode transaction: %v", err)
	}

	request := httptest.NewRequest(
		http.MethodGet,
		"https://example.com/api/v1/auth/google/callback?code=test-code&state=state-123",
		nil,
	)

	request.AddCookie(&http.Cookie{
		Name:  oauthTransactionCookieName,
		Value: value,
	})

	config := GoogleOAuthConfig(
		"google-client-id",
		"google-client-secret",
		"https://example.com/api/v1/auth/google/callback",
	)

	_, err = ProcessOAuthCallback(
		context.Background(),
		request,
		key,
		config,
		now.Add(time.Minute),
	)

	if err == nil {
		t.Fatal("expected provider mismatch error")
	}
}

func TestProcessOAuthCallbackRejectsProviderError(t *testing.T) {
	now := time.Date(
		2026,
		9,
		15,
		12,
		0,
		0,
		0,
		time.UTC,
	)

	transaction := OAuthTransaction{
		Provider:      OAuthProviderGoogle,
		State:         "state-123",
		Nonce:         "nonce-123",
		PKCEVerifier:  "pkce-verifier",
		CreatedAtUnix: now.Unix(),
	}

	key := bytes.Repeat(
		[]byte{0x42},
		oauthTransactionKeyLen,
	)

	codec, err := NewOAuthTransactionCodec(key)
	if err != nil {
		t.Fatalf("create codec: %v", err)
	}

	value, err := codec.Encode(transaction)
	if err != nil {
		t.Fatalf("encode transaction: %v", err)
	}

	request := httptest.NewRequest(
		http.MethodGet,
		"https://example.com/api/v1/auth/google/callback?error=access_denied&state=state-123",
		nil,
	)

	request.AddCookie(&http.Cookie{
		Name:  oauthTransactionCookieName,
		Value: value,
	})

	config := GoogleOAuthConfig(
		"google-client-id",
		"google-client-secret",
		"https://example.com/api/v1/auth/google/callback",
	)

	_, err = ProcessOAuthCallback(
		context.Background(),
		request,
		key,
		config,
		now.Add(time.Minute),
	)

	if err == nil {
		t.Fatal("expected provider error")
	}
}
