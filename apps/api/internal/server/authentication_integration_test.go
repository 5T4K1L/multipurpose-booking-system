package server

import (
	"context"
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

func TestAuthenticationMiddlewareIntegration(t *testing.T) {
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
	var id int64
	if err := pool.QueryRow(ctx, "INSERT INTO users(email,username) VALUES($1,$2) RETURNING id", fmt.Sprintf("middleware-%d@example.com", time.Now().UnixNano()), "Middleware fixture").Scan(&id); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if _, err := pool.Exec(cleanup, "DELETE FROM user_sessions WHERE user_id=$1", id); err != nil {
			t.Error(err)
		}
		if _, err := pool.Exec(cleanup, "DELETE FROM users WHERE id=$1", id); err != nil {
			t.Error(err)
		}
	}()
	sessions, err := service.NewSessionService(repository.NewSessionTransactions(pool), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	issued, err := sessions.Create(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	middleware, err := NewAuthenticationMiddleware(sessions, "http://localhost:3000", nil)
	if err != nil {
		t.Fatal(err)
	}
	called := 0
	protected := middleware.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, ok := IdentityFromContext(r.Context())
		if !ok || identity.UserID != id {
			t.Fatal("wrong identity")
		}
		called++
		w.WriteHeader(204)
	}))
	check := func(token string, status int) {
		t.Helper()
		before := called
		req := httptest.NewRequest("GET", "/protected?user_id=999", nil).WithContext(ctx)
		req.AddCookie(&http.Cookie{Name: "booking_session", Value: token})
		req.Header.Set("X-User-ID", "999")
		rec := httptest.NewRecorder()
		protected.ServeHTTP(rec, req)
		if rec.Code != status || (called > before) != (status == 204) {
			t.Fatalf("status=%d called=%d", rec.Code, called-before)
		}
	}
	check(issued.Token, 204)
	var seen bool
	var expires time.Time
	if err := pool.QueryRow(ctx, "SELECT last_seen_at IS NOT NULL,expires_at FROM user_sessions WHERE user_id=$1", id).Scan(&seen, &expires); err != nil || !seen || !expires.Equal(issued.ExpiresAt) {
		t.Fatal("last-seen/absolute expiration mismatch", err)
	}
	for _, token := range []string{"bad", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"} {
		check(token, 401)
	}
	if _, err := pool.Exec(ctx, "UPDATE users SET status='disabled' WHERE id=$1", id); err != nil {
		t.Fatal(err)
	}
	check(issued.Token, 401)
	if _, err := pool.Exec(ctx, "UPDATE users SET status='active' WHERE id=$1", id); err != nil {
		t.Fatal(err)
	}
	check(issued.Token, 204)
	logout, err := NewLogoutHandler(sessions, "http://localhost:3000", nil)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil).WithContext(ctx)
	req.Header.Set("Origin", "http://localhost:3000")
	req.AddCookie(&http.Cookie{Name: "booking_session", Value: issued.Token})
	rec := httptest.NewRecorder()
	logout.ServeHTTP(rec, req)
	if rec.Code != 204 {
		t.Fatal("logout failed")
	}
	check(issued.Token, 401)
	fresh, err := sessions.Create(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "UPDATE user_sessions SET expires_at=NOW()-INTERVAL '1 hour' WHERE user_id=$1", id); err != nil {
		t.Fatal(err)
	}
	check(fresh.Token, 401)
}
