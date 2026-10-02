package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"multipurpose-booking-system/api/internal/repository"
	"multipurpose-booking-system/api/internal/service"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestLogoutHTTPIntegration(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var userID int64
	if err := pool.QueryRow(ctx, "INSERT INTO users(email,username) VALUES($1,$2) RETURNING id", fmt.Sprintf("logout-%d@example.com", time.Now().UnixNano()), "Logout test").Scan(&userID); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if _, err := pool.Exec(cleanup, "DELETE FROM user_sessions WHERE user_id=$1", userID); err != nil {
			t.Error(err)
		}
		if _, err := pool.Exec(cleanup, "DELETE FROM users WHERE id=$1", userID); err != nil {
			t.Error(err)
		}
	}()
	sessions, err := service.NewSessionService(repository.NewSessionTransactions(pool), 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	first, err := sessions.Create(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	other, err := sessions.Create(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewLogoutHandler(sessions, "http://localhost:3000", nil)
	if err != nil {
		t.Fatal(err)
	}
	post := func(token string) {
		t.Helper()
		req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil).WithContext(ctx)
		req.Header.Set("Origin", "http://localhost:3000")
		req.AddCookie(&http.Cookie{Name: "booking_session", Value: token})
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != 204 || len(rec.Result().Cookies()) != 1 || rec.Result().Cookies()[0].MaxAge != -1 {
			t.Fatalf("logout failed: %d", rec.Code)
		}
	}
	post(first.Token)
	if _, err := sessions.Validate(ctx, first.Token); !errors.Is(err, service.ErrInvalidSession) {
		t.Fatalf("revoked token accepted: %v", err)
	}
	if user, err := sessions.Validate(ctx, other.Token); err != nil || user.ID != userID {
		t.Fatalf("other session affected: %v", err)
	}
	post(first.Token) // Repeated logout is harmless.
	post("malformed")
	post("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA") // Unknown well-formed token.
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM user_sessions WHERE user_id=$1 AND revoked_at IS NOT NULL", userID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("wrong revocation count: %d %v", count, err)
	}
	if _, err := pool.Exec(ctx, "UPDATE user_sessions SET expires_at=NOW()-INTERVAL '1 hour' WHERE user_id=$1", userID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "UPDATE users SET status='disabled' WHERE id=$1", userID); err != nil {
		t.Fatal(err)
	}
	post(other.Token) // Expired/disabled sessions must still be revocable.
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM user_sessions WHERE user_id=$1 AND revoked_at IS NOT NULL", userID).Scan(&count); err != nil || count != 2 {
		t.Fatalf("expired/disabled session not revoked: %d %v", count, err)
	}
}
