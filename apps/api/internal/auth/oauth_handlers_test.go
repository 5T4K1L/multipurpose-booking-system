package auth

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func testOAuthHTTPConfig(t *testing.T) OAuthHTTPConfig {
	return OAuthHTTPConfig{
		Google: GoogleOAuthConfig(
			"google-client-id",
			"google-client-secret",
			"https://example.com/api/v1/auth/google/callback",
		),
		Apple: AppleOAuthConfig(
			"com.example.booking",
			"TEAM123456",
			"KEY1234567",
			generateTestApplePrivateKeyPEM(t),
			"https://example.com/api/v1/auth/apple/callback",
		),
		TransactionKey: bytes.Repeat(
			[]byte{0x42},
			oauthTransactionKeyLen,
		),
		SecureCookies: true,
		Now: func() time.Time {
			return time.Date(
				2026,
				9,
				15,
				12,
				0,
				0,
				0,
				time.UTC,
			)
		},
	}
}

func TestNewOAuthHandlers(t *testing.T) {
	handlers, err := NewOAuthHandlers(testOAuthHTTPConfig(t))
	if err != nil {
		t.Fatalf("create OAuth handlers: %v", err)
	}

	if handlers == nil {
		t.Fatal("expected OAuth handlers")
	}
}

func TestNewOAuthHandlersRejectsInvalidTransactionKey(t *testing.T) {
	config := testOAuthHTTPConfig(t)
	config.TransactionKey = []byte("too-short")

	if _, err := NewOAuthHandlers(config); err == nil {
		t.Fatal("expected transaction key error")
	}
}

func TestGoogleStart(t *testing.T) {
	handlers, err := NewOAuthHandlers(testOAuthHTTPConfig(t))
	if err != nil {
		t.Fatalf("create OAuth handlers: %v", err)
	}

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/auth/google",
		nil,
	)

	recorder := httptest.NewRecorder()

	handlers.GoogleStart(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	if response.StatusCode != http.StatusFound {
		t.Fatalf(
			"expected HTTP %d, got %d",
			http.StatusFound,
			response.StatusCode,
		)
	}

	location := response.Header.Get("Location")
	if location == "" {
		t.Fatal("expected redirect location")
	}

	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse redirect location: %v", err)
	}

	if parsed.Scheme != "https" {
		t.Fatalf("expected HTTPS redirect, got %q", parsed.Scheme)
	}

	query := parsed.Query()

	if query.Get("state") == "" {
		t.Fatal("expected state")
	}

	if query.Get("nonce") == "" {
		t.Fatal("expected nonce")
	}

	if query.Get("code_challenge") == "" {
		t.Fatal("expected PKCE challenge")
	}

	cookies := response.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one OAuth transaction cookie, got %d", len(cookies))
	}

	if cookies[0].Name != oauthTransactionCookieName {
		t.Fatalf(
			"unexpected cookie name: %q",
			cookies[0].Name,
		)
	}

	if !cookies[0].HttpOnly {
		t.Fatal("expected HttpOnly cookie")
	}

	if !cookies[0].Secure {
		t.Fatal("expected Secure cookie")
	}
}

func TestAppleStart(t *testing.T) {
	handlers, err := NewOAuthHandlers(testOAuthHTTPConfig(t))
	if err != nil {
		t.Fatalf("create OAuth handlers: %v", err)
	}

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/auth/apple",
		nil,
	)

	recorder := httptest.NewRecorder()

	handlers.AppleStart(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	if response.StatusCode != http.StatusFound {
		t.Fatalf(
			"expected HTTP %d, got %d",
			http.StatusFound,
			response.StatusCode,
		)
	}

	location := response.Header.Get("Location")
	if location == "" {
		t.Fatal("expected redirect location")
	}

	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse redirect location: %v", err)
	}

	if parsed.Host != "appleid.apple.com" {
		t.Fatalf(
			"unexpected Apple redirect host: %q",
			parsed.Host,
		)
	}

	query := parsed.Query()

	if query.Get("state") == "" {
		t.Fatal("expected state")
	}

	if query.Get("nonce") == "" {
		t.Fatal("expected nonce")
	}

	if query.Get("code_challenge") == "" {
		t.Fatal("expected PKCE challenge")
	}
}
