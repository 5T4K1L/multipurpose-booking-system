package service

import (
	"context"
	"errors"
	"multipurpose-booking-system/api/internal/auth"
	"multipurpose-booking-system/api/internal/db"
	"strings"
	"testing"
)

type registrationFake struct {
	email, name, hash                   string
	createErr, credentialErr, commitErr error
	calls, credentialCalls              int
	committed                           bool
}

func (f *registrationFake) WithinRegistration(_ context.Context, fn func(RegistrationStore) error) error {
	f.calls++
	err := fn(f)
	if err == nil {
		err = f.commitErr
	}
	f.committed = err == nil
	return err
}
func (f *registrationFake) CreateUser(_ context.Context, email, name string) (db.User, error) {
	f.email = email
	f.name = name
	return db.User{ID: 1}, f.createErr
}
func (f *registrationFake) CreateUserCredential(_ context.Context, _ int64, hash string) (db.UserCredential, error) {
	f.credentialCalls++
	f.hash = hash
	return db.UserCredential{}, f.credentialErr
}

func TestRegistrationValidation(t *testing.T) {
	valid := RegistrationInput{Email: "user@example.com", Username: "Person", Password: "a long password 123"}
	for _, tc := range []struct {
		name   string
		change func(*RegistrationInput)
	}{
		{"short password", func(i *RegistrationInput) { i.Password = "short" }},
		{"long password", func(i *RegistrationInput) { i.Password = strings.Repeat("a", 129) }},
		{"bad email", func(i *RegistrationInput) { i.Email = "broken" }},
		{"display email", func(i *RegistrationInput) { i.Email = "Name <user@example.com>" }},
		{"email newline", func(i *RegistrationInput) { i.Email = "user\n@example.com" }},
		{"empty name", func(i *RegistrationInput) { i.Username = "  " }},
		{"control name", func(i *RegistrationInput) { i.Username = "person\x00" }},
		{"long name", func(i *RegistrationInput) { i.Username = strings.Repeat("a", 81) }},
		{"invalid UTF8", func(i *RegistrationInput) { i.Password = strings.Repeat("a", 15) + "\xff" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := valid
			tc.change(&input)
			f := &registrationFake{}
			err := NewRegistrationService(f).Register(context.Background(), input)
			if !errors.Is(err, ErrInvalidRegistration) || f.calls != 0 {
				t.Fatalf("got %v, transaction calls %d", err, f.calls)
			}
		})
	}
}

func TestRegistrationTransactionPolicy(t *testing.T) {
	input := RegistrationInput{Email: " user@example.com ", Username: " Person ", Password: "  a long password  "}
	for _, tc := range []struct {
		name                                      string
		createErr, credentialErr, commitErr, want error
	}{
		{name: "create"},
		{name: "duplicate", createErr: ErrConflict, want: ErrEmailRegistered},
		{name: "insert error", createErr: errors.New("secret detail"), want: ErrRegistrationUnavailable},
		{name: "credential error", credentialErr: errors.New("secret hash"), want: ErrRegistrationUnavailable},
		{name: "credential conflict is not email duplicate", credentialErr: ErrConflict, want: ErrRegistrationUnavailable},
		{name: "commit error", commitErr: errors.New("secret detail"), want: ErrRegistrationUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &registrationFake{createErr: tc.createErr, credentialErr: tc.credentialErr, commitErr: tc.commitErr}
			err := NewRegistrationService(f).Register(context.Background(), input)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v", err)
			}
			if f.email != "user@example.com" || f.name != "Person" {
				t.Fatal("email/name whitespace not trimmed")
			}
			if tc.createErr != nil && f.credentialCalls != 0 {
				t.Fatal("credential written after failed user insertion")
			}
			if tc.name == "create" {
				ok, err := auth.VerifyPassword(input.Password, f.hash)
				if err != nil || !ok || f.hash == input.Password {
					t.Fatal("password not securely preserved")
				}
			}
			if tc.want != nil && f.committed {
				t.Fatal("failed registration committed")
			}
		})
	}
}
