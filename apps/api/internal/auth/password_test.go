package auth

import "testing"

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
