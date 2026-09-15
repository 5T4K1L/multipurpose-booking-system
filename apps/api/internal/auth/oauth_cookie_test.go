package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetOAuthTransactionCookie(t *testing.T) {
	recorder := httptest.NewRecorder()

	SetOAuthTransactionCookie(
		recorder,
		"encrypted-transaction",
		true,
	)

	response := recorder.Result()
	defer response.Body.Close()

	cookies := response.Cookies()

	if len(cookies) != 1 {
		t.Fatalf("expected one cookie, got %d", len(cookies))
	}

	cookie := cookies[0]

	if cookie.Name != oauthTransactionCookieName {
		t.Fatalf("unexpected cookie name: %q", cookie.Name)
	}

	if cookie.Value != "encrypted-transaction" {
		t.Fatalf("unexpected cookie value: %q", cookie.Value)
	}

	if !cookie.HttpOnly {
		t.Fatal("expected HttpOnly cookie")
	}

	if !cookie.Secure {
		t.Fatal("expected Secure cookie")
	}

	if cookie.Path != "/" {
		t.Fatalf("expected Path=/, got %q", cookie.Path)
	}

	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("expected SameSite=Lax, got %v", cookie.SameSite)
	}
}

func TestGetOAuthTransactionCookie(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodGet,
		"https://example.com/auth/callback",
		nil,
	)

	request.AddCookie(&http.Cookie{
		Name:  oauthTransactionCookieName,
		Value: "encrypted-transaction",
	})

	value, err := GetOAuthTransactionCookie(request)
	if err != nil {
		t.Fatalf("get OAuth transaction cookie: %v", err)
	}

	if value != "encrypted-transaction" {
		t.Fatalf("unexpected transaction value: %q", value)
	}
}

func TestGetOAuthTransactionCookieMissing(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodGet,
		"https://example.com/auth/callback",
		nil,
	)

	if _, err := GetOAuthTransactionCookie(request); err == nil {
		t.Fatal("expected missing cookie error")
	}
}

func TestClearOAuthTransactionCookie(t *testing.T) {
	recorder := httptest.NewRecorder()

	ClearOAuthTransactionCookie(recorder, true)

	response := recorder.Result()
	defer response.Body.Close()

	cookies := response.Cookies()

	if len(cookies) != 1 {
		t.Fatalf("expected one cookie, got %d", len(cookies))
	}

	if cookies[0].MaxAge >= 0 {
		t.Fatalf("expected negative MaxAge, got %d", cookies[0].MaxAge)
	}
}
