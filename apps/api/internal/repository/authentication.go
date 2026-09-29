package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"multipurpose-booking-system/api/internal/db"
	"multipurpose-booking-system/api/internal/service"
)

// AuthenticationTransactions commits user creation and identity attachment together.
type AuthenticationTransactions struct{ pool *pgxpool.Pool }

func NewAuthenticationTransactions(pool *pgxpool.Pool) *AuthenticationTransactions {
	return &AuthenticationTransactions{pool: pool}
}

func (r *AuthenticationTransactions) WithinAuthentication(ctx context.Context, fn func(service.AuthenticationStore) error) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		return fn(authenticationStore{queries: db.New(tx)})
	})
}

type authenticationStore struct{ queries *db.Queries }

func authenticationResult[T any](value T, err error) (T, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return value, service.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return value, service.ErrConflict
	}
	return value, err
}

func (r authenticationStore) GetUserByID(ctx context.Context, id int64) (db.User, error) {
	return authenticationResult(r.queries.GetUserByID(ctx, id))
}
func (r authenticationStore) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	return authenticationResult(r.queries.GetUserByEmail(ctx, email))
}
func (r authenticationStore) GetUserCredential(ctx context.Context, id int64) (db.UserCredential, error) {
	return authenticationResult(r.queries.GetUserCredential(ctx, id))
}
func (r authenticationStore) GetOAuthIdentity(ctx context.Context, provider, subject string) (db.UserOauthIdentity, error) {
	return authenticationResult(r.queries.GetOAuthIdentity(ctx, db.GetOAuthIdentityParams{Provider: provider, ProviderSubject: subject}))
}
func (r authenticationStore) GetUserOAuthIdentity(ctx context.Context, id int64, provider string) (db.UserOauthIdentity, error) {
	return authenticationResult(r.queries.GetUserOAuthIdentity(ctx, db.GetUserOAuthIdentityParams{UserID: id, Provider: provider}))
}
func (r authenticationStore) CreateUser(ctx context.Context, email, name string) (db.User, error) {
	return authenticationResult(r.queries.CreateUser(ctx, db.CreateUserParams{Email: email, Username: name}))
}
func (r authenticationStore) CreateOAuthIdentity(ctx context.Context, id int64, provider, subject string) (db.UserOauthIdentity, error) {
	return authenticationResult(r.queries.CreateOAuthIdentity(ctx, db.CreateOAuthIdentityParams{UserID: id, Provider: provider, ProviderSubject: subject}))
}
