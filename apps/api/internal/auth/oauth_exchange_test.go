package auth

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestExchangeOAuthCodeRequiresInputs(t *testing.T) {
	config := GoogleOAuthConfig(
		"client-id",
		"client-secret",
		"https://example.com/callback",
	)

	_, err := ExchangeOAuthCode(
		context.Background(),
		config,
		"",
		"verifier",
	)
	if err == nil {
		t.Fatal("expected missing code error")
	}

	_, err = ExchangeOAuthCode(
		context.Background(),
		config,
		"code",
		"",
	)
	if err == nil {
		t.Fatal("expected missing PKCE verifier error")
	}
}

func TestExchangeOAuthCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Fatalf(
				"unexpected content type: %q",
				r.Header.Get("Content-Type"),
			)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}

		values, err := url.ParseQuery(string(body))
		if err != nil {
			t.Fatalf("parse request body: %v", err)
		}

		if values.Get("code") != "authorization-code" {
			t.Fatalf("unexpected code: %q", values.Get("code"))
		}

		clientID, clientSecret, basicAuth := r.BasicAuth()

		if basicAuth {
			if clientID != "client-id" {
				t.Fatalf("unexpected client ID: %q", clientID)
			}

			if clientSecret != "client-secret" {
				t.Fatal("unexpected client secret")
			}
		} else {
			if values.Get("client_id") != "client-id" {
				t.Fatalf("unexpected client ID: %q", values.Get("client_id"))
			}

			if values.Get("client_secret") != "client-secret" {
				t.Fatal("unexpected client secret")
			}
		}

		if values.Get("code_verifier") == "" {
			t.Fatal("expected PKCE verifier")
		}

		if values.Get("redirect_uri") != "https://example.com/callback" {
			t.Fatalf("unexpected redirect URI: %q", values.Get("redirect_uri"))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"access_token": "test-access-token",
			"token_type": "Bearer",
			"expires_in": 3600,
			"id_token": "test-id-token"
		}`))
	}))
	defer server.Close()

	config := GoogleOAuthConfig(
		"client-id",
		"client-secret",
		"https://example.com/callback",
	)

	config.TokenURL = server.URL

	result, err := ExchangeOAuthCode(
		context.Background(),
		config,
		"authorization-code",
		"test-pkce-verifier",
	)
	if err != nil {
		t.Fatalf("exchange OAuth code: %v", err)
	}

	if result.IDToken != "test-id-token" {
		t.Fatalf("expected test ID token, got %q", result.IDToken)
	}
}

func TestExchangeOAuthCodeRejectsMissingIDToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"access_token": "test-access-token",
			"token_type": "Bearer",
			"expires_in": 3600
		}`))
	}))
	defer server.Close()

	config := GoogleOAuthConfig(
		"client-id",
		"client-secret",
		"https://example.com/callback",
	)

	config.TokenURL = server.URL

	_, err := ExchangeOAuthCode(
		context.Background(),
		config,
		"authorization-code",
		"test-pkce-verifier",
	)
	if err == nil {
		t.Fatal("expected missing ID token error")
	}

	if !strings.Contains(err.Error(), "ID token") {
		t.Fatalf("unexpected error: %v", err)
	}
}
