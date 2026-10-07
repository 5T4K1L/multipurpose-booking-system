package server

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"multipurpose-booking-system/api/internal/db"
	"multipurpose-booking-system/api/internal/repository"
	"multipurpose-booking-system/api/internal/service"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestAuthorizationIntegration(t *testing.T) {
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
	if err := pool.QueryRow(ctx, "INSERT INTO users(email,username) VALUES($1,$2) RETURNING id", fmt.Sprintf("rbac-%d@example.invalid", time.Now().UnixNano()), "RBAC fixture").Scan(&id); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		for _, table := range []string{"user_platform_roles", "user_sessions"} {
			if _, err := pool.Exec(cleanup, "DELETE FROM "+table+" WHERE user_id=$1", id); err != nil {
				t.Error(err)
			}
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
	authn, err := NewAuthenticationMiddleware(sessions, "http://localhost:3000", nil)
	if err != nil {
		t.Fatal(err)
	}
	authz := service.NewAuthorizationService(repository.NewAuthorizationRepository(db.New(pool)))
	check := func(role service.Role, status int) {
		t.Helper()
		called := false
		handler := authn.Require(RequireRoles(authz, nil, role)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true; w.WriteHeader(204) })))
		req := httptest.NewRequest("GET", "/protected?role=PLATFORM_ADMIN&user_id=999", nil).WithContext(ctx)
		req.AddCookie(&http.Cookie{Name: "booking_session", Value: issued.Token})
		req.Header.Set("X-Role", "PLATFORM_ADMIN")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != status || called != (status == 204) {
			t.Fatalf("role %s status %d want %d", role, rec.Code, status)
		}
	}
	check(service.RoleCustomer, 204)
	check(service.RolePlatformAdmin, 403)
	if _, err := pool.Exec(ctx, "INSERT INTO user_platform_roles(user_id,role) VALUES($1,'PLATFORM_ADMIN')", id); err != nil {
		t.Fatal(err)
	}
	check(service.RolePlatformAdmin, 204)
	check(service.RolePlatformOperator, 403)
	check(service.RoleBusinessOwner, 403)
	if _, err := pool.Exec(ctx, "INSERT INTO user_platform_roles(user_id,role) VALUES($1,'PLATFORM_ADMIN')", id); err == nil {
		t.Fatal("duplicate role allowed")
	}
	if _, err := pool.Exec(ctx, "INSERT INTO user_platform_roles(user_id,role) VALUES($1,'BUSINESS_OWNER')", id); err == nil {
		t.Fatal("business role granted globally")
	}
	if _, err := pool.Exec(ctx, "DELETE FROM user_platform_roles WHERE user_id=$1", id); err != nil {
		t.Fatal(err)
	}
	check(service.RolePlatformAdmin, 403)
	if _, err := pool.Exec(ctx, "INSERT INTO user_platform_roles(user_id,role) VALUES($1,'PLATFORM_OPERATOR')", id); err != nil {
		t.Fatal(err)
	}
	check(service.RolePlatformOperator, 204)
	if _, err := pool.Exec(ctx, "UPDATE users SET status='disabled' WHERE id=$1", id); err != nil {
		t.Fatal(err)
	}
	check(service.RoleCustomer, 401)
	if err := authz.RequireAnyRole(ctx, id, service.RolePlatformOperator); err != service.ErrForbidden {
		t.Fatal("service allowed disabled account", err)
	}
	missing, err := repository.NewAuthorizationRepository(db.New(pool)).GetAuthorizationState(ctx, -1)
	if err != nil || missing.Active || len(missing.PlatformRoles) != 0 {
		t.Fatal("missing user authorized", err)
	}
}
