package repository

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"multipurpose-booking-system/api/internal/db"
	"multipurpose-booking-system/api/internal/service"
)

type RegistrationTransactions struct{ pool *pgxpool.Pool }

func NewRegistrationTransactions(pool *pgxpool.Pool) *RegistrationTransactions {
	return &RegistrationTransactions{pool: pool}
}
func (r *RegistrationTransactions) WithinRegistration(ctx context.Context, fn func(service.RegistrationStore) error) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		return fn(registrationStore{authenticationStore: authenticationStore{queries: db.New(tx)}})
	})
}

type registrationStore struct{ authenticationStore }

func (r registrationStore) CreateUserCredential(ctx context.Context, id int64, hash string) (db.UserCredential, error) {
	return authenticationResult(r.queries.CreateUserCredential(ctx, db.CreateUserCredentialParams{UserID: id, PasswordHash: hash}))
}
