package repository

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"multipurpose-booking-system/api/internal/db"
	"multipurpose-booking-system/api/internal/service"
)

type SessionTransactions struct{ pool *pgxpool.Pool }

func NewSessionTransactions(pool *pgxpool.Pool) *SessionTransactions {
	return &SessionTransactions{pool: pool}
}

func (r *SessionTransactions) WithinSession(ctx context.Context, fn func(service.SessionStore) error) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		queries := db.New(tx)
		return fn(sessionStore{AuthRepository: NewAuthRepository(queries), users: NewUserRepository(queries)})
	})
}

type sessionStore struct {
	*AuthRepository
	users *UserRepository
}

func (r sessionStore) GetUserByID(ctx context.Context, id int64) (db.User, error) {
	return authenticationResult(r.users.GetUserByID(ctx, id))
}
func (r sessionStore) GetUserSessionByTokenHash(ctx context.Context, hash string) (db.UserSession, error) {
	return authenticationResult(r.AuthRepository.GetUserSessionByTokenHash(ctx, hash))
}
