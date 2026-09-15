package auth

import (
	"fmt"
	"net/http"
)

const oauthTransactionCookieName = "__Host-oauth_transaction"

// SetOAuthTransactionCookie stores the encrypted OAuth transaction.
func SetOAuthTransactionCookie(
	w http.ResponseWriter,
	value string,
	secure bool,
) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthTransactionCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   int(oauthTransactionMaxAge.Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// GetOAuthTransactionCookie retrieves the encrypted OAuth transaction.
func GetOAuthTransactionCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(oauthTransactionCookieName)
	if err != nil {
		return "", fmt.Errorf("get OAuth transaction cookie: %w", err)
	}

	if cookie.Value == "" {
		return "", fmt.Errorf("OAuth transaction cookie is empty")
	}

	return cookie.Value, nil
}

// ClearOAuthTransactionCookie removes the OAuth transaction cookie.
func ClearOAuthTransactionCookie(
	w http.ResponseWriter,
	secure bool,
) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthTransactionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}
