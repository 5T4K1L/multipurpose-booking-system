package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"multipurpose-booking-system/api/internal/auth"
	"multipurpose-booking-system/api/internal/db"
	"multipurpose-booking-system/api/internal/service"
)

func TestSessionServiceIntegration(t *testing.T) {
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
	queries := db.New(pool)
	email := fmt.Sprintf("session-%d@example.com", time.Now().UnixNano())
	user, err := NewUserRepository(queries).CreateUser(ctx, email, "Session test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		for _, table := range []string{"user_sessions", "user_credentials", "user_oauth_identities"} {
			if _, err := pool.Exec(cleanup, "DELETE FROM "+table+" WHERE user_id=$1", user.ID); err != nil {
				t.Errorf("cleanup %s: %v", table, err)
			}
		}
		if _, err := pool.Exec(cleanup, "DELETE FROM users WHERE id=$1", user.ID); err != nil {
			t.Error(err)
		}
	}()
	hash, err := auth.HashPassword("integration-password")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewAuthRepository(queries).CreateUserCredential(ctx, user.ID, hash); err != nil {
		t.Fatal(err)
	}
	if _, err := NewOAuthRepository(queries).CreateOAuthIdentity(ctx, user.ID, "google", email); err != nil {
		t.Fatal(err)
	}
	authentication := service.NewAuthenticationService(NewAuthenticationTransactions(pool))
	sessions, err := service.NewSessionService(NewSessionTransactions(pool), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	passwordSession, err := sessions.AuthenticatePassword(ctx, authentication, email, "integration-password")
	if err != nil {
		t.Fatal(err)
	}
	oauthSession, err := sessions.AuthenticateOAuth(ctx, authentication, auth.OAuthIdentity{Provider: auth.OAuthProviderGoogle, ProviderSubject: email}, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{passwordSession.Token, oauthSession.Token} {
		got, err := sessions.Validate(ctx, token)
		if err != nil || got.ID != user.ID {
			t.Fatalf("validate: %v", err)
		}
	}
	var plaintext int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM user_sessions WHERE token_hash=$1 OR token_hash=$2", passwordSession.Token, oauthSession.Token).Scan(&plaintext); err != nil || plaintext != 0 {
		t.Fatal("raw token stored or query failed", err)
	}
	if err := sessions.Revoke(ctx, passwordSession.Token); err != nil {
		t.Fatal(err)
	}
	if err := sessions.Revoke(ctx, passwordSession.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := sessions.Validate(ctx, passwordSession.Token); !errors.Is(err, service.ErrInvalidSession) {
		t.Fatal("revoked token accepted", err)
	}
	if _, err := sessions.Validate(ctx, oauthSession.Token); err != nil {
		t.Fatal("single revocation affected another session", err)
	}
	if _, err := pool.Exec(ctx, "UPDATE users SET status='disabled' WHERE id=$1", user.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := sessions.Validate(ctx, oauthSession.Token); !errors.Is(err, service.ErrInvalidSession) {
		t.Fatal("disabled account accepted", err)
	}
	if _, err := sessions.Create(ctx, user.ID); !errors.Is(err, service.ErrAccountDisabled) {
		t.Fatal("disabled account created session", err)
	}
	if _, err := pool.Exec(ctx, "UPDATE users SET status='active' WHERE id=$1", user.ID); err != nil {
		t.Fatal(err)
	}
	if err := sessions.RevokeAll(ctx, user.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := sessions.Validate(ctx, oauthSession.Token); !errors.Is(err, service.ErrInvalidSession) {
		t.Fatal("revoke-all failed", err)
	}
	fresh, err := sessions.Create(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "UPDATE user_sessions SET expires_at=NOW()-INTERVAL '1 second' WHERE user_id=$1", user.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := sessions.Validate(ctx, fresh.Token); !errors.Is(err, service.ErrInvalidSession) {
		t.Fatal("expired session accepted", err)
	}
	if issued, err := sessions.AuthenticatePassword(ctx, authentication, email, "wrong"); !errors.Is(err, service.ErrInvalidCredentials) || issued.Token != "" {
		t.Fatal("failed password created session", err)
	}
}
