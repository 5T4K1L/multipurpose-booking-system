package service

import (
	"context"
	"errors"
	"multipurpose-booking-system/api/internal/auth"
	"multipurpose-booking-system/api/internal/db"
	"strings"
	"testing"
	"time"
)

func TestLoginValidationAndLegacyPassword(t *testing.T) {
	hash, err := auth.HashPassword(" short ")
	if err != nil {
		t.Fatal(err)
	}
	authentication := NewAuthenticationService(&fakeStore{user: db.User{ID: 1, Status: "active"}, credential: db.UserCredential{PasswordHash: hash}})
	sessions, err := NewSessionService(&sessionFake{user: db.User{ID: 1, Status: "active"}}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	login := NewLoginService(authentication, sessions)
	for _, input := range [][2]string{{"", "password"}, {"user@example.com", ""}, {strings.Repeat("a", 255), "pass"}, {"user@example.com", strings.Repeat("a", 513)}, {"user@example.com", "\xff"}} {
		issued, err := login.Login(context.Background(), input[0], input[1])
		if !errors.Is(err, ErrInvalidLoginInput) || issued.Token != "" {
			t.Fatalf("invalid input: %v", err)
		}
	}
	issued, err := login.Login(context.Background(), " user@example.com ", " short ")
	if err != nil || issued.Token == "" {
		t.Fatalf("legacy password rejected: %v", err)
	}
	if _, err := login.Login(context.Background(), "user@example.com", "short"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatal("password whitespace was altered", err)
	}
}
