package auth

import (
	"bytes"
	"encoding/base64"
	"os"
	"testing"
)

func TestLoadTransactionKey(t *testing.T) {
	key := make([]byte, oauthTransactionKeyLen)
	encoded := base64.RawURLEncoding.EncodeToString(key)

	t.Setenv(OAuthTransactionKeyEnv, encoded)

	loaded, err := loadTransactionKey()
	if err != nil {
		t.Fatalf("load transaction key: %v", err)
	}

	if len(loaded) != oauthTransactionKeyLen {
		t.Fatalf("expected %d bytes, got %d", oauthTransactionKeyLen, len(loaded))
	}
}

func TestLoadTransactionKeyRejectsMissingValue(t *testing.T) {
	_ = os.Unsetenv(OAuthTransactionKeyEnv)

	if _, err := loadTransactionKey(); err == nil {
		t.Fatal("expected missing transaction key error")
	}
}

func TestLoadTransactionKeyRejectsWrongLength(t *testing.T) {
	encoded := base64.RawURLEncoding.EncodeToString(
		[]byte("too-short"),
	)

	t.Setenv(OAuthTransactionKeyEnv, encoded)

	if _, err := loadTransactionKey(); err == nil {
		t.Fatal("expected invalid key length error")
	}
}

func TestGenerateOAuthTransactionKey(t *testing.T) {
	first, err := GenerateOAuthTransactionKey()
	if err != nil {
		t.Fatalf("generate first key: %v", err)
	}

	second, err := GenerateOAuthTransactionKey()
	if err != nil {
		t.Fatalf("generate second key: %v", err)
	}

	if first == "" || second == "" {
		t.Fatal("expected generated keys")
	}

	if first == second {
		t.Fatal("expected generated keys to differ")
	}

	decoded, err := base64.RawURLEncoding.DecodeString(first)
	if err != nil {
		t.Fatalf("decode generated key: %v", err)
	}

	if len(decoded) != oauthTransactionKeyLen {
		t.Fatalf("expected %d bytes, got %d", oauthTransactionKeyLen, len(decoded))
	}
}

func TestLoadTransactionKeyURLAlphabet(t *testing.T) {
	key := bytes.Repeat([]byte{255}, oauthTransactionKeyLen)
	t.Setenv(OAuthTransactionKeyEnv, base64.RawURLEncoding.EncodeToString(key))
	loaded, err := loadTransactionKey()
	if err != nil || !bytes.Equal(loaded, key) {
		t.Fatal("generated key alphabet is not accepted", err)
	}
}

func TestEnabledOAuthConfiguration(t *testing.T) {
	for _, name := range []string{"GOOGLE_OAUTH_ENABLED", "APPLE_OAUTH_ENABLED", OAuthTransactionKeyEnv} {
		t.Setenv(name, "")
	}
	cfg, err := LoadEnabledOAuthConfig()
	if err != nil || cfg.Google.Provider != "" || cfg.Apple.Provider != "" {
		t.Fatal("password-only startup failed", err)
	}
	t.Setenv("GOOGLE_OAUTH_ENABLED", "yes")
	if _, err := LoadEnabledOAuthConfig(); err == nil {
		t.Fatal("ambiguous flag accepted")
	}
	t.Setenv("GOOGLE_OAUTH_ENABLED", "true")
	t.Setenv("GOOGLE_OAUTH_CLIENT_ID", "")
	if _, err := LoadEnabledOAuthConfig(); err == nil {
		t.Fatal("incomplete enabled provider accepted")
	}
	t.Setenv("GOOGLE_OAUTH_CLIENT_ID", "client")
	t.Setenv("GOOGLE_OAUTH_CLIENT_SECRET", "secret")
	t.Setenv("GOOGLE_OAUTH_REDIRECT_URL", "http://localhost:8080/api/v1/auth/google/callback")
	t.Setenv(OAuthTransactionKeyEnv, base64.RawURLEncoding.EncodeToString(make([]byte, 32)))
	cfg, err = LoadEnabledOAuthConfig()
	if err != nil || cfg.Google.Provider != OAuthProviderGoogle || cfg.Apple.Provider != "" {
		t.Fatal("independent Google configuration failed", err)
	}
}
