package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

const (
	oauthTransactionMaxAge = 10 * time.Minute
	oauthTransactionKeyLen = 32
)

// OAuthTransaction binds an OAuth callback to the original authorization
// attempt.
type OAuthTransaction struct {
	Provider      OAuthProvider `json:"provider"`
	State         string        `json:"state"`
	Nonce         string        `json:"nonce"`
	PKCEVerifier  string        `json:"pkce_verifier"`
	CreatedAtUnix int64         `json:"created_at"`
}

// OAuthTransactionCodec protects temporary OAuth transaction data.
type OAuthTransactionCodec struct {
	aead cipher.AEAD
}

// NewOAuthTransactionCodec creates a codec using a 32-byte encryption key.
func NewOAuthTransactionCodec(key []byte) (*OAuthTransactionCodec, error) {
	if len(key) != oauthTransactionKeyLen {
		return nil, fmt.Errorf(
			"OAuth transaction key must be exactly %d bytes",
			oauthTransactionKeyLen,
		)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create OAuth transaction cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create OAuth transaction AEAD: %w", err)
	}

	return &OAuthTransactionCodec{
		aead: aead,
	}, nil
}

// Encode encrypts an OAuth transaction into a URL-safe value.
func (c *OAuthTransactionCodec) Encode(
	transaction OAuthTransaction,
) (string, error) {
	if err := validateOAuthTransaction(transaction); err != nil {
		return "", err
	}

	payload, err := json.Marshal(transaction)
	if err != nil {
		return "", fmt.Errorf("encode OAuth transaction: %w", err)
	}

	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate OAuth transaction nonce: %w", err)
	}

	ciphertext := c.aead.Seal(nil, nonce, payload, nil)
	combined := append(nonce, ciphertext...)

	return base64.RawURLEncoding.EncodeToString(combined), nil
}

// Decode decrypts and validates an OAuth transaction.
func (c *OAuthTransactionCodec) Decode(
	value string,
	now time.Time,
) (OAuthTransaction, error) {
	if value == "" {
		return OAuthTransaction{}, fmt.Errorf("OAuth transaction is required")
	}

	combined, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return OAuthTransaction{}, fmt.Errorf(
			"decode OAuth transaction: %w",
			err,
		)
	}

	if len(combined) < c.aead.NonceSize() {
		return OAuthTransaction{}, fmt.Errorf("invalid OAuth transaction")
	}

	nonce := combined[:c.aead.NonceSize()]
	ciphertext := combined[c.aead.NonceSize():]

	payload, err := c.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return OAuthTransaction{}, fmt.Errorf(
			"decrypt OAuth transaction: %w",
			err,
		)
	}

	var transaction OAuthTransaction
	if err := json.Unmarshal(payload, &transaction); err != nil {
		return OAuthTransaction{}, fmt.Errorf(
			"decode OAuth transaction payload: %w",
			err,
		)
	}

	if err := validateOAuthTransaction(transaction); err != nil {
		return OAuthTransaction{}, err
	}

	createdAt := time.Unix(transaction.CreatedAtUnix, 0)

	if !ValidateOAuthStateAge(
		createdAt,
		now,
		oauthTransactionMaxAge,
	) {
		return OAuthTransaction{}, fmt.Errorf(
			"OAuth transaction has expired",
		)
	}

	return transaction, nil
}

func validateOAuthTransaction(transaction OAuthTransaction) error {
	switch transaction.Provider {
	case OAuthProviderGoogle, OAuthProviderApple:
	default:
		return fmt.Errorf(
			"unsupported OAuth provider: %q",
			transaction.Provider,
		)
	}

	if transaction.State == "" {
		return fmt.Errorf("OAuth transaction state is required")
	}

	if transaction.Nonce == "" {
		return fmt.Errorf("OAuth transaction nonce is required")
	}

	if transaction.PKCEVerifier == "" {
		return fmt.Errorf("OAuth transaction PKCE verifier is required")
	}

	if transaction.CreatedAtUnix <= 0 {
		return fmt.Errorf("OAuth transaction creation time is required")
	}

	return nil
}
