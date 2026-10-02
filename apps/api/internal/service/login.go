package service

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"
)

var ErrInvalidLoginInput = errors.New("provide an email and password")

// LoginService composes existing authentication and session creation without HTTP.
type LoginService struct {
	authentication *AuthenticationService
	sessions       *SessionService
}

func NewLoginService(authentication *AuthenticationService, sessions *SessionService) *LoginService {
	return &LoginService{authentication: authentication, sessions: sessions}
}

func (s *LoginService) Login(ctx context.Context, email, password string) (IssuedSession, error) {
	email = strings.TrimSpace(email)
	// Do not impose registration's new-password minimum on existing credentials.
	if email == "" || len(email) > 254 || !utf8.ValidString(email) || password == "" || len(password) > 512 || !utf8.ValidString(password) {
		return IssuedSession{}, ErrInvalidLoginInput
	}
	return s.sessions.AuthenticatePassword(ctx, s.authentication, email, password)
}
