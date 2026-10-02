package auth

import (
	"strings"
	"testing"
)

func TestVerifyPasswordRejectsCorruptParameters(t *testing.T) {
	for _, parameters := range []string{"m=65536,t=0,p=4", "m=65536,t=3,p=0", "m=4294967295,t=3,p=4", "m=65536,t=999,p=4", "m=1,t=3,p=4"} {
		hash := "$argon2id$v=19$" + parameters + "$MDEyMzQ1Njc4OWFiY2RlZg$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
		if ok, err := VerifyPassword("password", hash); err == nil || ok {
			t.Fatal("corrupt hash accepted")
		}
	}
	for _, hash := range []string{strings.Repeat("a", 513), "$argon2id$v=19$m=65536,t=3,p=4$$"} {
		if ok, err := VerifyPassword("password", hash); err == nil || ok {
			t.Fatal("malformed hash accepted")
		}
	}
}

func TestHashPassword(t *testing.T) {
	password := "CorrectHorseBatteryStaple"

	// Generate a password hash.
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	if hash == "" {
		t.Fatal("expected password hash")
	}

	if hash == password {
		t.Fatal("password must not be stored as plaintext")
	}

	if hash[:10] != "$argon2id$" {
		t.Fatalf("expected Argon2id hash, got %q", hash[:10])
	}
}

func TestHashPasswordGeneratesDifferentHashes(t *testing.T) {
	password := "CorrectHorseBatteryStaple"

	// A unique salt should produce a different hash each time.
	hash1, err := HashPassword(password)
	if err != nil {
		t.Fatalf("hash password 1: %v", err)
	}

	hash2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("hash password 2: %v", err)
	}

	if hash1 == hash2 {
		t.Fatal("expected different hashes for the same password")
	}
}

func TestVerifyPassword(t *testing.T) {
	password := "CorrectHorseBatteryStaple"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	// Correct passwords must verify.
	valid, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("verify password: %v", err)
	}

	if !valid {
		t.Fatal("expected password verification to succeed")
	}

	// Incorrect passwords must fail.
	invalid, err := VerifyPassword("WrongPassword", hash)
	if err != nil {
		t.Fatalf("verify wrong password: %v", err)
	}

	if invalid {
		t.Fatal("expected wrong password verification to fail")
	}
}
