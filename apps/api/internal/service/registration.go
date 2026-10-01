package service

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"unicode"
	"unicode/utf8"

	"multipurpose-booking-system/api/internal/auth"
	"multipurpose-booking-system/api/internal/db"
)

var (
	ErrInvalidRegistration     = errors.New("provide a valid email, a name of 1–80 characters, and a password of 15–128 characters")
	ErrRegistrationUnavailable = errors.New("registration is temporarily unavailable")
)

type RegistrationInput struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type RegistrationStore interface {
	CreateUser(context.Context, string, string) (db.User, error)
	CreateUserCredential(context.Context, int64, string) (db.UserCredential, error)
}
type RegistrationTransactions interface {
	WithinRegistration(context.Context, func(RegistrationStore) error) error
}
type RegistrationService struct{ transactions RegistrationTransactions }

func NewRegistrationService(transactions RegistrationTransactions) *RegistrationService {
	return &RegistrationService{transactions: transactions}
}

func validRegistration(input RegistrationInput) bool {
	if !utf8.ValidString(input.Username) || !utf8.ValidString(input.Password) {
		return false
	}
	nameLength, passwordLength := utf8.RuneCountInString(input.Username), utf8.RuneCountInString(input.Password)
	if nameLength < 1 || nameLength > 80 || passwordLength < 15 || passwordLength > 128 || len(input.Password) > 512 {
		return false
	}
	if strings.IndexFunc(input.Username, unicode.IsControl) >= 0 {
		return false
	}
	if len(input.Email) > 254 {
		return false
	}
	// Use a plain ASCII mailbox, preserving the existing exact-email identity policy.
	for _, r := range input.Email {
		if r < 33 || r > 126 {
			return false
		}
	}
	address, err := mail.ParseAddress(input.Email)
	if err != nil || address.Address != input.Email {
		return false
	}
	at := strings.LastIndexByte(input.Email, '@')
	return at > 0 && at <= 64 && at < len(input.Email)-1
}

// Register creates a local user and credential together, without a session or
// verification claim. Duplicate emails intentionally return the same result as creation.
func (s *RegistrationService) Register(ctx context.Context, input RegistrationInput) error {
	input.Email = strings.TrimSpace(input.Email)
	input.Username = strings.TrimSpace(input.Username)
	if !validRegistration(input) {
		return ErrInvalidRegistration
	}
	if err := ctx.Err(); err != nil {
		return ErrRegistrationUnavailable
	}
	// Hash before attempting insertion for both new and duplicate accounts.
	// Passwords are neither trimmed nor truncated.
	hash, err := auth.HashPassword(input.Password)
	if err != nil {
		return ErrRegistrationUnavailable
	}
	duplicate := false
	err = s.transactions.WithinRegistration(ctx, func(store RegistrationStore) error {
		user, err := store.CreateUser(ctx, input.Email, input.Username)
		if errors.Is(err, ErrConflict) {
			duplicate = true
			return err
		}
		if err != nil {
			return err
		}
		_, err = store.CreateUserCredential(ctx, user.ID, hash)
		return err
	})
	if duplicate && errors.Is(err, ErrConflict) {
		return nil
	}
	if err != nil {
		return ErrRegistrationUnavailable
	}
	return nil
}
