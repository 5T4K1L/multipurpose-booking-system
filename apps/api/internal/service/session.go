package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/auth"
	"multipurpose-booking-system/api/internal/db"
)

var (
	ErrInvalidSession     = errors.New("invalid session")
	ErrSessionUnavailable = errors.New("session service is unavailable")
)

// SessionStore is scoped to a transaction; adapters translate missing rows to ErrNotFound.
type SessionStore interface {
	GetUserByID(context.Context, int64) (db.User, error)
	CreateUserSession(context.Context, int64, string, pgtype.Timestamp) (db.UserSession, error)
	GetUserSessionByTokenHash(context.Context, string) (db.UserSession, error)
	UpdateUserSessionLastSeen(context.Context, int64) error
	RevokeUserSession(context.Context, int64) error
	RevokeAllUserSessions(context.Context, int64) error
}
type SessionTransactions interface {
	WithinSession(context.Context, func(SessionStore) error) error
}

// IssuedSession carries the bearer secret only at creation, for later cookie delivery.
// Token is deliberately excluded from ordinary JSON serialization.
type IssuedSession struct {
	Token     string `json:"-"`
	ExpiresAt time.Time
}

type SessionService struct {
	transactions SessionTransactions
	lifetime     time.Duration
	now          func() time.Time
}

// NewSessionService requires an explicit absolute lifetime. Sessions never slide or refresh.
func NewSessionService(transactions SessionTransactions, lifetime time.Duration) (*SessionService, error) {
	if transactions == nil || lifetime < time.Second {
		return nil, errors.New("session store and lifetime of at least one second are required")
	}
	return &SessionService{transactions: transactions, lifetime: lifetime, now: time.Now}, nil
}

func sessionHash(token string) (string, error) {
	if len(token) != 43 {
		return "", ErrInvalidSession
	}
	raw, err := base64.RawURLEncoding.Strict().DecodeString(token)
	if err != nil || len(raw) != 32 {
		return "", ErrInvalidSession
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func sessionError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrInvalidSession) || errors.Is(err, ErrNotFound) {
		return ErrInvalidSession
	}
	if errors.Is(err, ErrAccountDisabled) {
		return ErrAccountDisabled
	}
	return ErrSessionUnavailable
}

// Create requires a user ID obtained from successful server-side authentication,
// never an arbitrary client-supplied ID. Account eligibility is checked again.
func (s *SessionService) Create(ctx context.Context, authenticatedUserID int64) (IssuedSession, error) {
	if authenticatedUserID <= 0 {
		return IssuedSession{}, ErrInvalidSession
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return IssuedSession{}, ErrSessionUnavailable
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash, _ := sessionHash(token)
	expires := s.now().UTC().Add(s.lifetime).Truncate(time.Microsecond)
	err := s.transactions.WithinSession(ctx, func(store SessionStore) error {
		user, err := store.GetUserByID(ctx, authenticatedUserID)
		if err != nil {
			return err
		}
		if user.Status != "active" {
			return ErrAccountDisabled
		}
		_, err = store.CreateUserSession(ctx, user.ID, hash, pgtype.Timestamp{Time: expires, Valid: true})
		return err
	})
	if err != nil {
		return IssuedSession{}, sessionError(err)
	}
	return IssuedSession{Token: token, ExpiresAt: expires}, nil
}

// Validate checks persisted revocation, absolute expiry, and current account status.
// Last-seen updates do not extend expiration. No identity is returned on commit failure.
func (s *SessionService) Validate(ctx context.Context, token string) (db.User, error) {
	hash, err := sessionHash(token)
	if err != nil {
		return db.User{}, err
	}
	var user db.User
	err = s.transactions.WithinSession(ctx, func(store SessionStore) error {
		session, err := store.GetUserSessionByTokenHash(ctx, hash)
		if err != nil {
			return err
		}
		if session.RevokedAt.Valid || !session.ExpiresAt.Valid || session.ExpiresAt.InfinityModifier != pgtype.Finite || !s.now().UTC().Before(session.ExpiresAt.Time) {
			return ErrInvalidSession
		}
		user, err = store.GetUserByID(ctx, session.UserID)
		if err != nil {
			return err
		}
		if user.Status != "active" {
			return ErrInvalidSession
		}
		return store.UpdateUserSessionLastSeen(ctx, session.ID)
	})
	if err != nil {
		return db.User{}, sessionError(err)
	}
	return user, nil
}

// Revoke is idempotent for unknown/already-revoked valid-format bearer tokens.
func (s *SessionService) Revoke(ctx context.Context, token string) error {
	hash, err := sessionHash(token)
	if err != nil {
		return err
	}
	return sessionError(s.transactions.WithinSession(ctx, func(store SessionStore) error {
		session, err := store.GetUserSessionByTokenHash(ctx, hash)
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		return store.RevokeUserSession(ctx, session.ID)
	}))
}

// RevokeAll is a trusted server operation for an authenticated owner or authorized
// account-security workflow. Authorization and HTTP endpoints belong to later phases.
func (s *SessionService) RevokeAll(ctx context.Context, userID int64) error {
	if userID <= 0 {
		return ErrInvalidSession
	}
	return sessionError(s.transactions.WithinSession(ctx, func(store SessionStore) error { return store.RevokeAllUserSessions(ctx, userID) }))
}

// AuthenticatePassword creates a fresh application session only after password verification.
func (s *SessionService) AuthenticatePassword(ctx context.Context, authentication *AuthenticationService, email, password string) (IssuedSession, error) {
	user, err := authentication.AuthenticatePassword(ctx, email, password)
	if err != nil {
		return IssuedSession{}, err
	}
	return s.Create(ctx, user.ID)
}

// AuthenticateOAuth requires an identity verified by the existing OAuth foundation.
func (s *SessionService) AuthenticateOAuth(ctx context.Context, authentication *AuthenticationService, identity auth.OAuthIdentity, allowCreation bool) (IssuedSession, error) {
	user, err := authentication.ResolveOAuth(ctx, identity, allowCreation)
	if err != nil {
		return IssuedSession{}, err
	}
	return s.Create(ctx, user.ID)
}
