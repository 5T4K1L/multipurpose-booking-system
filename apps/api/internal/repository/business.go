package repository

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"multipurpose-booking-system/api/internal/db"
	"multipurpose-booking-system/api/internal/service"
)

type BusinessTransactions struct{ pool *pgxpool.Pool }

func NewBusinessTransactions(pool *pgxpool.Pool) *BusinessTransactions {
	return &BusinessTransactions{pool}
}
func (r *BusinessTransactions) WithinBusiness(ctx context.Context, fn func(service.BusinessStore) error) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error { return fn(db.New(tx)) })
}
