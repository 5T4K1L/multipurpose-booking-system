package auth

import (
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
