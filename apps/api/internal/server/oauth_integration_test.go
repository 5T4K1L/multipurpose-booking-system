package server

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"multipurpose-booking-system/api/internal/auth"
	"multipurpose-booking-system/api/internal/repository"
	"multipurpose-booking-system/api/internal/service"
	"os"
	"testing"
	"time"
)

func TestOAuthDatabaseSessionHandoff(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	for _, provider := range []auth.OAuthProvider{auth.OAuthProviderGoogle, auth.OAuthProviderApple} {
		t.Run(string(provider), func(t *testing.T) {
			authentication := service.NewAuthenticationService(repository.NewAuthenticationTransactions(pool))
			sessions, err := service.NewSessionService(repository.NewSessionTransactions(pool), time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			f := newOAuthFixture(t, provider, service.NewOAuthLoginService(authentication, sessions, true))
			f.email = fmt.Sprintf("oauth-flow-%s-%d@example.invalid", provider, time.Now().UnixNano())
			f.subject = f.email
			defer func() {
				cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
				defer stop()
				var id int64
				err := pool.QueryRow(cleanup, "SELECT id FROM users WHERE email=$1", f.email).Scan(&id)
				if err != nil {
					return
				}
				for _, table := range []string{"user_sessions", "user_oauth_identities", "user_credentials"} {
					if _, err := pool.Exec(cleanup, "DELETE FROM "+table+" WHERE user_id=$1", id); err != nil {
						t.Error(err)
					}
				}
				if _, err := pool.Exec(cleanup, "DELETE FROM users WHERE id=$1", id); err != nil {
					t.Error(err)
				}
			}()

			f.routes.login = service.NewOAuthLoginService(authentication, sessions, false)
			deniedCookie := f.start(t)
			denied := f.callback(t, deniedCookie, nil)
			if denied.Header().Get("Location") != "https://web.example/oauth/error" || len(denied.Result().Cookies()) != 1 {
				t.Fatal("creation policy bypassed")
			}
			var before int
			if err := pool.QueryRow(ctx, "SELECT count(*) FROM users WHERE email=$1", f.email).Scan(&before); err != nil || before != 0 {
				t.Fatal("disabled registration created user", err)
			}
			f.routes.login = service.NewOAuthLoginService(authentication, sessions, true)
			cookie := f.start(t)
			rec := f.callback(t, cookie, nil)
			if rec.Header().Get("Location") != "https://web.example/business" {
				t.Fatal("OAuth login did not complete")
			}
			cookies := rec.Result().Cookies()
			if len(cookies) != 2 {
				t.Fatal("session not issued")
			}
			token := cookies[1].Value
			user, err := sessions.Validate(ctx, token)
			if err != nil || user.Email != f.email {
				t.Fatal("cookie not backed by correct local user", err)
			}
			var count int
			if err := pool.QueryRow(ctx, "SELECT count(*) FROM user_oauth_identities WHERE user_id=$1 AND provider=$2 AND provider_subject=$3", user.ID, string(provider), f.subject).Scan(&count); err != nil || count != 1 {
				t.Fatal("identity ownership not persisted", err)
			}
			if err := sessions.Revoke(ctx, token); err != nil {
				t.Fatal(err)
			}
			if _, err := sessions.Validate(ctx, token); err == nil {
				t.Fatal("revoked OAuth session accepted")
			}
			// Reuse the provider with a changed subject and the same email: no auto-link.
			f.subject = "other-" + f.subject
			cookie = f.start(t)
			rec = f.callback(t, cookie, nil)
			if rec.Header().Get("Location") != "https://web.example/oauth/error" || len(rec.Result().Cookies()) != 1 {
				t.Fatal("email collision linked or signed in")
			}
			if err := pool.QueryRow(ctx, "SELECT count(*) FROM user_oauth_identities WHERE user_id=$1", user.ID).Scan(&count); err != nil || count != 1 {
				t.Fatal("ownership changed", err)
			}
		})
	}
}
