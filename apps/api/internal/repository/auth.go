package repository

import (
	"context"
	"fmt"

	"multipurpose-booking-system/api/internal/db"

	"github.com/jackc/pgx/v5/pgtype"
)

// AuthRepository provides database access for authentication data.
type AuthRepository struct {
	queries *db.Queries
}

// NewAuthRepository creates an authentication repository.
func NewAuthRepository(queries *db.Queries) *AuthRepository {
	return &AuthRepository{
		queries: queries,
	}
}

// GetUserCredential retrieves a user's password credential.
func (r *AuthRepository) GetUserCredential(
	ctx context.Context,
	userID int64,
) (db.UserCredential, error) {
	credential, err := r.queries.GetUserCredential(ctx, userID)
	if err != nil {
		return db.UserCredential{}, fmt.Errorf("get user credential: %w", err)
	}

	return credential, nil
}

// CreateUserCredential stores a user's password hash.
func (r *AuthRepository) CreateUserCredential(
	ctx context.Context,
	userID int64,
	passwordHash string,
) (db.UserCredential, error) {
	credential, err := r.queries.CreateUserCredential(ctx, db.CreateUserCredentialParams{
		UserID:       userID,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return db.UserCredential{}, fmt.Errorf("create user credential: %w", err)
	}

	return credential, nil
}

// UpdateUserPasswordHash changes a user's stored password hash.
func (r *AuthRepository) UpdateUserPasswordHash(
	ctx context.Context,
	userID int64,
	passwordHash string,
) error {
	if err := r.queries.UpdateUserPasswordHash(ctx, db.UpdateUserPasswordHashParams{
		UserID:       userID,
		PasswordHash: passwordHash,
	}); err != nil {
		return fmt.Errorf("update user password hash: %w", err)
	}

	return nil
}

// CreateUserSession stores a new authentication session.
func (r *AuthRepository) CreateUserSession(
	ctx context.Context,
	userID int64,
	tokenHash string,
	expiresAt pgtype.Timestamp,
) (db.UserSession, error) {
	session, err := r.queries.CreateUserSession(ctx, db.CreateUserSessionParams{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return db.UserSession{}, fmt.Errorf("create user session: %w", err)
	}

	return session, nil
}

// GetUserSessionByTokenHash retrieves a session by its hashed token.
func (r *AuthRepository) GetUserSessionByTokenHash(
	ctx context.Context,
	tokenHash string,
) (db.UserSession, error) {
	session, err := r.queries.GetUserSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		return db.UserSession{}, fmt.Errorf("get user session: %w", err)
	}

	return session, nil
}

// UpdateUserSessionLastSeen updates session activity.
func (r *AuthRepository) UpdateUserSessionLastSeen(
	ctx context.Context,
	sessionID int64,
) error {
	if err := r.queries.UpdateUserSessionLastSeen(ctx, sessionID); err != nil {
		return fmt.Errorf("update user session activity: %w", err)
	}

	return nil
}

// RevokeUserSession invalidates one session.
func (r *AuthRepository) RevokeUserSession(
	ctx context.Context,
	sessionID int64,
) error {
	if err := r.queries.RevokeUserSession(ctx, sessionID); err != nil {
		return fmt.Errorf("revoke user session: %w", err)
	}

	return nil
}

// RevokeAllUserSessions invalidates all active sessions for a user.
func (r *AuthRepository) RevokeAllUserSessions(
	ctx context.Context,
	userID int64,
) error {
	if err := r.queries.RevokeAllUserSessions(ctx, userID); err != nil {
		return fmt.Errorf("revoke all user sessions: %w", err)
	}

	return nil
}
