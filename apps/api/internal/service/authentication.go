// Package service contains application workflows independent of HTTP.
package service

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"multipurpose-booking-system/api/internal/auth"
	"multipurpose-booking-system/api/internal/db"
)

var (
	ErrInvalidCredentials        = errors.New("invalid credentials")
	ErrAccountDisabled           = errors.New("account is not active")
	ErrInvalidIdentity           = errors.New("invalid external identity")
	ErrAccountCreationRequired   = errors.New("explicit account creation is required")
	ErrLinkRequired              = errors.New("explicit account linking is required")
	ErrLinkConflict              = errors.New("external identity cannot be linked")
	ErrAuthenticationUnavailable = errors.New("authentication is unavailable")
	// Persistence adapters translate their errors into these boundary errors.
	ErrNotFound = errors.New("authentication record not found")
	ErrConflict = errors.New("authentication record conflict")
)

// AuthenticationStore is scoped to a transaction by AuthenticationTransactions.
type AuthenticationStore interface {
	GetUserByID(context.Context, int64) (db.User, error)
	GetUserByEmail(context.Context, string) (db.User, error)
	GetUserCredential(context.Context, int64) (db.UserCredential, error)
	GetOAuthIdentity(context.Context, string, string) (db.UserOauthIdentity, error)
	GetUserOAuthIdentity(context.Context, int64, string) (db.UserOauthIdentity, error)
	CreateUser(context.Context, string, string) (db.User, error)
	CreateOAuthIdentity(context.Context, int64, string, string) (db.UserOauthIdentity, error)
}

type AuthenticationTransactions interface {
	WithinAuthentication(context.Context, func(AuthenticationStore) error) error
}

type AuthenticationService struct{ transactions AuthenticationTransactions }

func NewAuthenticationService(transactions AuthenticationTransactions) *AuthenticationService {
	return &AuthenticationService{transactions: transactions}
}

// run never returns a user after a failed operation or commit, or exposes storage errors.
func (s *AuthenticationService) run(ctx context.Context, operation func(AuthenticationStore) (db.User, error)) (db.User, error) {
	var user db.User
	err := s.transactions.WithinAuthentication(ctx, func(store AuthenticationStore) error {
		var err error
		user, err = operation(store)
		return err
	})
	if err != nil {
		for _, safe := range []error{ErrInvalidCredentials, ErrAccountDisabled, ErrInvalidIdentity, ErrAccountCreationRequired, ErrLinkRequired, ErrLinkConflict} {
			if errors.Is(err, safe) {
				return db.User{}, safe
			}
		}
		return db.User{}, ErrAuthenticationUnavailable
	}
	return user, nil
}

func active(user db.User, err error) (db.User, error) {
	if errors.Is(err, ErrNotFound) {
		return db.User{}, ErrInvalidCredentials
	}
	if err != nil {
		return db.User{}, err
	}
	if user.Status != "active" {
		return db.User{}, ErrAccountDisabled
	}
	return user, nil
}

// AuthenticatePassword verifies credentials before disclosing account eligibility.
// Email retains the existing repository's exact, case-sensitive lookup semantics.
func (s *AuthenticationService) AuthenticatePassword(ctx context.Context, email, password string) (db.User, error) {
	return s.run(ctx, func(store AuthenticationStore) (db.User, error) {
		if email == "" || password == "" {
			return db.User{}, ErrInvalidCredentials
		}
		user, err := store.GetUserByEmail(ctx, email)
		if errors.Is(err, ErrNotFound) {
			verifyMissingCredential(password)
			return db.User{}, ErrInvalidCredentials
		}
		if err != nil {
			return db.User{}, err
		}
		credential, err := store.GetUserCredential(ctx, user.ID)
		if errors.Is(err, ErrNotFound) {
			verifyMissingCredential(password)
			return db.User{}, ErrInvalidCredentials
		}
		if err != nil {
			return db.User{}, err
		}
		ok, err := auth.VerifyPassword(password, credential.PasswordHash)
		if err != nil {
			return db.User{}, ErrAuthenticationUnavailable
		}
		if !ok {
			return db.User{}, ErrInvalidCredentials
		}
		return active(user, nil)
	})
}

// ResolveLocalUser checks eligibility; it is not proof of authentication.
func (s *AuthenticationService) ResolveLocalUser(ctx context.Context, userID int64) (db.User, error) {
	return s.run(ctx, func(store AuthenticationStore) (db.User, error) { return active(store.GetUserByID(ctx, userID)) })
}

func validIdentity(identity auth.OAuthIdentity) bool {
	return (identity.Provider == auth.OAuthProviderGoogle || identity.Provider == auth.OAuthProviderApple) && strings.TrimSpace(identity.ProviderSubject) != ""
}

// ResolveOAuth accepts only identities already verified by the OAuth foundation.
// allowCreation must come from server policy, never an untrusted callback parameter.
func (s *AuthenticationService) ResolveOAuth(ctx context.Context, identity auth.OAuthIdentity, allowCreation bool) (db.User, error) {
	return s.run(ctx, func(store AuthenticationStore) (db.User, error) {
		if !validIdentity(identity) {
			return db.User{}, ErrInvalidIdentity
		}
		linked, err := store.GetOAuthIdentity(ctx, string(identity.Provider), identity.ProviderSubject)
		if err == nil {
			return active(store.GetUserByID(ctx, linked.UserID))
		}
		if !errors.Is(err, ErrNotFound) {
			return db.User{}, err
		}
		if !allowCreation {
			return db.User{}, ErrAccountCreationRequired
		}
		address, err := mail.ParseAddress(identity.Email)
		if !identity.EmailVerified || err != nil || address.Address != identity.Email {
			return db.User{}, ErrInvalidIdentity
		}
		// Preserve the existing exact-email uniqueness policy; never merge by email.
		_, err = store.GetUserByEmail(ctx, identity.Email)
		if err == nil {
			return db.User{}, ErrLinkRequired
		}
		if !errors.Is(err, ErrNotFound) {
			return db.User{}, err
		}
		name := strings.TrimSpace(identity.DisplayName)
		if name == "" {
			name = "User"
		}
		user, err := store.CreateUser(ctx, identity.Email, name)
		if errors.Is(err, ErrConflict) {
			return db.User{}, ErrLinkRequired
		}
		if err != nil {
			return db.User{}, err
		}
		if _, err = active(user, nil); err != nil {
			return db.User{}, err
		}
		_, err = store.CreateOAuthIdentity(ctx, user.ID, string(identity.Provider), identity.ProviderSubject)
		if errors.Is(err, ErrConflict) {
			return db.User{}, ErrLinkConflict
		}
		return user, err
	})
}

// LinkOAuth requires a server-authenticated user ID and an explicit link action
// bound to the validated OAuth flow. A user ID supplied by a client is insufficient.
// Session and HTTP enforcement belong to later phases.
func (s *AuthenticationService) LinkOAuth(ctx context.Context, authenticatedUserID int64, identity auth.OAuthIdentity) (db.User, error) {
	return s.run(ctx, func(store AuthenticationStore) (db.User, error) {
		if authenticatedUserID <= 0 {
			return db.User{}, ErrInvalidCredentials
		}
		if !validIdentity(identity) {
			return db.User{}, ErrInvalidIdentity
		}
		user, err := active(store.GetUserByID(ctx, authenticatedUserID))
		if err != nil {
			return db.User{}, err
		}
		linked, err := store.GetOAuthIdentity(ctx, string(identity.Provider), identity.ProviderSubject)
		if err == nil {
			if linked.UserID != user.ID {
				return db.User{}, ErrLinkConflict
			}
			return user, nil
		}
		if !errors.Is(err, ErrNotFound) {
			return db.User{}, err
		}
		_, err = store.GetUserOAuthIdentity(ctx, user.ID, string(identity.Provider))
		if err == nil {
			return db.User{}, ErrLinkConflict
		}
		if !errors.Is(err, ErrNotFound) {
			return db.User{}, err
		}
		_, err = store.CreateOAuthIdentity(ctx, user.ID, string(identity.Provider), identity.ProviderSubject)
		if errors.Is(err, ErrConflict) {
			return db.User{}, ErrLinkConflict
		}
		return user, err
	})
}
