package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"multipurpose-booking-system/api/internal/auth"
	"multipurpose-booking-system/api/internal/db"
	"multipurpose-booking-system/api/internal/service"
	"os"
	"sync"
	"testing"
	"time"
)

func TestRegistrationIntegration(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	email := fmt.Sprintf("register-%d@example.com", time.Now().UnixNano())
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if _, err := pool.Exec(cleanup, "DELETE FROM user_credentials WHERE user_id IN (SELECT id FROM users WHERE email=$1)", email); err != nil {
			t.Error(err)
		}
		if _, err := pool.Exec(cleanup, "DELETE FROM users WHERE email=$1", email); err != nil {
			t.Error(err)
		}
	}()
	transactions := NewRegistrationTransactions(pool)
	registration := service.NewRegistrationService(transactions)
	input := service.RegistrationInput{Email: email, Username: "Registration test", Password: "original long password"}
	// Concurrent requests must yield one account and one credential, without an existence leak.
	var group sync.WaitGroup
	failures := make(chan error, 2)
	for i := 0; i < 2; i++ {
		group.Add(1)
		go func() { defer group.Done(); failures <- registration.Register(ctx, input) }()
	}
	group.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	queries := db.New(pool)
	user, err := queries.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatal(err)
	}
	credential, err := queries.GetUserCredential(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := auth.VerifyPassword(input.Password, credential.PasswordHash)
	if err != nil || !ok {
		t.Fatal("registration password not usable", err)
	}
	input.Password = "replacement password attempt"
	if err := registration.Register(ctx, input); err != nil {
		t.Fatal(err)
	}
	unchanged, err := queries.GetUserCredential(ctx, user.ID)
	if err != nil || unchanged.PasswordHash != credential.PasswordHash {
		t.Fatal("duplicate replaced password", err)
	}
	var sessions int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM user_sessions WHERE user_id=$1", user.ID).Scan(&sessions); err != nil || sessions != 0 {
		t.Fatal("registration created a session", err)
	}
	// Existing accounts without a password must not gain one through registration.
	if _, err := pool.Exec(ctx, "DELETE FROM user_credentials WHERE user_id=$1", user.ID); err != nil {
		t.Fatal(err)
	}
	if err := registration.Register(ctx, input); err != nil {
		t.Fatal(err)
	}
	var credentials int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM user_credentials WHERE user_id=$1", user.ID).Scan(&credentials); err != nil || credentials != 0 {
		t.Fatal("registration attached a password to an existing account", err)
	}
	sentinel := errors.New("abort registration")
	rollbackEmail := "rollback-" + email
	err = transactions.WithinRegistration(ctx, func(store service.RegistrationStore) error {
		created, err := store.CreateUser(ctx, rollbackEmail, "Rollback")
		if err != nil {
			return err
		}
		if _, err := store.CreateUserCredential(ctx, created.ID, credential.PasswordHash); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM users WHERE email=$1", rollbackEmail).Scan(&count); err != nil || count != 0 {
		t.Fatal("transaction did not roll back", err)
	}
}
