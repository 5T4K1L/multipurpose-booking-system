package auth

import (
	"bytes"
	"encoding/base64"
	"testing"
	"time"
)

func testOAuthTransactionCodec(t *testing.T) *OAuthTransactionCodec {
	t.Helper()

	key := bytes.Repeat([]byte{0x42}, oauthTransactionKeyLen)

	codec, err := NewOAuthTransactionCodec(key)
	if err != nil {
		t.Fatalf("create transaction codec: %v", err)
	}

	return codec
}

func validOAuthTransaction() OAuthTransaction {
	return OAuthTransaction{
		Provider:      OAuthProviderGoogle,
		State:         "state-123",
		Nonce:         "nonce-123",
		PKCEVerifier:  "pkce-verifier-123",
		CreatedAtUnix: time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC).Unix(),
	}
}

func TestNewOAuthTransactionCodecRequires32ByteKey(t *testing.T) {
	_, err := NewOAuthTransactionCodec([]byte("too-short"))

	if err == nil {
		t.Fatal("expected key length error")
	}
}

func TestOAuthTransactionCodecRoundTrip(t *testing.T) {
	codec := testOAuthTransactionCodec(t)

	original := validOAuthTransaction()
	now := time.Date(2026, 9, 15, 12, 5, 0, 0, time.UTC)

	encoded, err := codec.Encode(original)
	if err != nil {
		t.Fatalf("encode transaction: %v", err)
	}

	if encoded == "" {
		t.Fatal("expected encoded transaction")
	}

	decoded, err := codec.Decode(encoded, now)
	if err != nil {
		t.Fatalf("decode transaction: %v", err)
	}

	if decoded.Provider != original.Provider {
		t.Fatalf("unexpected provider: %q", decoded.Provider)
	}

	if decoded.State != original.State {
		t.Fatalf("unexpected state: %q", decoded.State)
	}

	if decoded.Nonce != original.Nonce {
		t.Fatalf("unexpected nonce: %q", decoded.Nonce)
	}

	if decoded.PKCEVerifier != original.PKCEVerifier {
		t.Fatalf("unexpected PKCE verifier: %q", decoded.PKCEVerifier)
	}

	if decoded.CreatedAtUnix != original.CreatedAtUnix {
		t.Fatalf("unexpected creation time: %d", decoded.CreatedAtUnix)
	}
}

func TestOAuthTransactionCannotBeTamperedWith(t *testing.T) {
	codec := testOAuthTransactionCodec(t)

	transaction := validOAuthTransaction()
	now := time.Date(2026, 9, 15, 12, 5, 0, 0, time.UTC)

	encoded, err := codec.Encode(transaction)
	if err != nil {
		t.Fatalf("encode transaction: %v", err)
	}

	combined, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode encoded transaction: %v", err)
	}

	combined[len(combined)-1] ^= 1

	tampered := base64.RawURLEncoding.EncodeToString(combined)

	if _, err := codec.Decode(tampered, now); err == nil {
		t.Fatal("expected tampered transaction to be rejected")
	}
}

func TestOAuthTransactionExpires(t *testing.T) {
	codec := testOAuthTransactionCodec(t)

	transaction := validOAuthTransaction()

	encoded, err := codec.Encode(transaction)
	if err != nil {
		t.Fatalf("encode transaction: %v", err)
	}

	now := time.Unix(
		transaction.CreatedAtUnix,
		0,
	).Add(oauthTransactionMaxAge + time.Second)

	if _, err := codec.Decode(encoded, now); err == nil {
		t.Fatal("expected expired transaction to be rejected")
	}
}

func TestOAuthTransactionRejectsFutureCreationTime(t *testing.T) {
	codec := testOAuthTransactionCodec(t)

	transaction := validOAuthTransaction()

	encoded, err := codec.Encode(transaction)
	if err != nil {
		t.Fatalf("encode transaction: %v", err)
	}

	now := time.Unix(
		transaction.CreatedAtUnix,
		0,
	).Add(-time.Second)

	if _, err := codec.Decode(encoded, now); err == nil {
		t.Fatal("expected future transaction to be rejected")
	}
}

func TestOAuthTransactionRejectsInvalidProvider(t *testing.T) {
	codec := testOAuthTransactionCodec(t)

	transaction := validOAuthTransaction()
	transaction.Provider = OAuthProvider("github")

	if _, err := codec.Encode(transaction); err == nil {
		t.Fatal("expected unsupported provider error")
	}
}
