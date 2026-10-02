package server

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"multipurpose-booking-system/api/internal/repository"
	"multipurpose-booking-system/api/internal/service"
)

func TestPasswordLoginHTTPIntegration(t *testing.T) {
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
	email := fmt.Sprintf("login-%d@example.com", time.Now().UnixNano())
	password := "a dedicated login test password"
	registration := service.NewRegistrationService(repository.NewRegistrationTransactions(pool))
	if err := registration.Register(ctx, service.RegistrationInput{Email: email, Username: "Login integration", Password: password}); err != nil {
		t.Fatal(err)
	}
	var userID int64
	if err := pool.QueryRow(ctx, "SELECT id FROM users WHERE email=$1", email).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		for _, table := range []string{"user_sessions", "user_credentials"} {
			if _, err := pool.Exec(cleanup, "DELETE FROM "+table+" WHERE user_id=$1", userID); err != nil {
				t.Error(err)
			}
		}
		if _, err := pool.Exec(cleanup, "DELETE FROM users WHERE id=$1", userID); err != nil {
			t.Error(err)
		}
	}()
	authentication := service.NewAuthenticationService(repository.NewAuthenticationTransactions(pool))
	sessions, err := service.NewSessionService(repository.NewSessionTransactions(pool), 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewLoginHandler(service.NewLoginService(authentication, sessions), "http://localhost:3000", nil)
	if err != nil {
		t.Fatal(err)
	}
	post := func(email, password string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]string{"email": email, "password": password})
		request := loginRequest(string(body), "http://localhost:3000").WithContext(ctx)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, request)
		return rec
	}
	first := post(email, password)
	if first.Code != 200 || len(first.Result().Cookies()) != 1 {
		t.Fatalf("first login: %d", first.Code)
	}
	token := first.Result().Cookies()[0].Value
	user, err := sessions.Validate(ctx, token)
	if err != nil || user.ID != userID {
		t.Fatal("cookie did not authenticate local user", err)
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(raw)
	var stored string
	if err := pool.QueryRow(ctx, "SELECT token_hash FROM user_sessions WHERE user_id=$1", userID).Scan(&stored); err != nil || stored != hex.EncodeToString(digest[:]) {
		t.Fatal("session hash not persisted", err)
	}
	if strings.Contains(first.Body.String(), token) {
		t.Fatal("bearer token exposed in JSON")
	}
	second := post(email, password)
	if second.Code != 200 || second.Result().Cookies()[0].Value == token {
		t.Fatal("fresh session was not issued")
	}
	for _, attempt := range [][2]string{{email, "wrong password"}, {"missing-" + email, password}} {
		rec := post(attempt[0], attempt[1])
		if rec.Code != 401 || len(rec.Result().Cookies()) != 0 {
			t.Fatal("invalid credentials issued a cookie")
		}
	}
	if _, err := pool.Exec(ctx, "UPDATE users SET status='disabled' WHERE id=$1", userID); err != nil {
		t.Fatal(err)
	}
	rec := post(email, password)
	if rec.Code != 401 || len(rec.Result().Cookies()) != 0 {
		t.Fatal("disabled account signed in")
	}
	if _, err := pool.Exec(ctx, "UPDATE users SET status='active' WHERE id=$1", userID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM user_credentials WHERE user_id=$1", userID); err != nil {
		t.Fatal(err)
	}
	rec = post(email, password)
	if rec.Code != 401 || len(rec.Result().Cookies()) != 0 {
		t.Fatal("account without credential signed in")
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM user_sessions WHERE user_id=$1", userID).Scan(&count); err != nil || count != 2 {
		t.Fatal("failed login created session", err)
	}
}
