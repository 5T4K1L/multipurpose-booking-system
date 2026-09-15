package auth

import (
	"context"
	"testing"
	"time"
)

func TestValidateGoogleIDTokenRequiresClientID(t *testing.T) {
	_, err := ValidateGoogleIDToken(
		context.Background(),
		"",
		"test-token",
		"nonce",
		time.Now(),
	)

	if err == nil {
		t.Fatal("expected client ID error")
	}
}

func TestValidateAppleIDTokenRequiresClientID(t *testing.T) {
	_, err := ValidateAppleIDToken(
		context.Background(),
		"",
		"test-token",
		"nonce",
		time.Now(),
	)

	if err == nil {
		t.Fatal("expected client ID error")
	}
}

func TestValidateGoogleIDTokenRejectsInvalidToken(t *testing.T) {
	_, err := ValidateGoogleIDToken(
		context.Background(),
		"google-client-id",
		"not-a-real-token",
		"nonce",
		time.Now(),
	)

	if err == nil {
		t.Fatal("expected invalid token error")
	}
}

func TestValidateAppleIDTokenRejectsInvalidToken(t *testing.T) {
	_, err := ValidateAppleIDToken(
		context.Background(),
		"com.example.booking",
		"not-a-real-token",
		"nonce",
		time.Now(),
	)

	if err == nil {
		t.Fatal("expected invalid token error")
	}
}
