package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"multipurpose-booking-system/api/internal/service"
)

func TestAuthenticationErrorTranslation(t *testing.T) {
	for _, tc := range []struct{ input, want error }{
		{fmt.Errorf("wrapped: %w", pgx.ErrNoRows), service.ErrNotFound},
		{&pgconn.PgError{Code: "23505"}, service.ErrConflict},
		{context.Canceled, context.Canceled},
	} {
		_, err := authenticationResult(0, tc.input)
		if !errors.Is(err, tc.want) {
			t.Fatalf("got %v, want %v", err, tc.want)
		}
	}
}

// Requires the normal repository integration database with migrations applied.
func TestAuthenticationTransactionRollbackIntegration(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	email := fmt.Sprintf("rollback-%d@example.com", time.Now().UnixNano())
	sentinel := errors.New("abort account creation")
	err = NewAuthenticationTransactions(pool).WithinAuthentication(ctx, func(store service.AuthenticationStore) error {
		user, err := store.CreateUser(ctx, email, "Rollback test")
		if err != nil {
			return err
		}
		if user.Status != "active" {
			return errors.New("new account not active")
		}
		if _, err := store.CreateOAuthIdentity(ctx, user.ID, "google", email); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM users WHERE email = $1", email).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("rollback left an orphan account")
	}
}
