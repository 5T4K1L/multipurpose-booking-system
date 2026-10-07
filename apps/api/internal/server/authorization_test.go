package server

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"multipurpose-booking-system/api/internal/service"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type authorizerStub struct {
	err   error
	id    int64
	roles []service.Role
	calls int
}

func (s *authorizerStub) RequireAnyRole(ctx context.Context, id int64, roles ...service.Role) error {
	s.calls++
	s.id = id
	s.roles = append([]service.Role(nil), roles...)
	if _, ok := ctx.Deadline(); !ok {
		panic("no authorization timeout")
	}
	return s.err
}
func TestRequireRoles(t *testing.T) {
	for _, tc := range []struct {
		name          string
		authenticated bool
		err           error
		status        int
	}{{"allowed", true, nil, 204}, {"anonymous", false, nil, 401}, {"forbidden", true, service.ErrForbidden, 403}, {"unavailable", true, errors.New("secret storage detail"), 503}} {
		t.Run(tc.name, func(t *testing.T) {
			stub := &authorizerStub{err: tc.err}
			var logs bytes.Buffer
			policy := []service.Role{service.RolePlatformAdmin}
			wrap := RequireRoles(stub, slog.New(slog.NewTextHandler(&logs, nil)), policy...)
			policy[0] = service.RoleCustomer
			called := false
			handler := wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				if r.Context().Err() != nil {
					t.Fatal("cancelled downstream")
				}
				w.WriteHeader(204)
			}))
			req := httptest.NewRequest("GET", "/protected?role=PLATFORM_ADMIN&user_id=999", nil)
			req.Header.Set("X-Role", "PLATFORM_ADMIN")
			req.Header.Set("X-User-ID", "999")
			if tc.authenticated {
				req = req.WithContext(context.WithValue(req.Context(), identityContextKey{}, Identity{UserID: 42}))
			}
			rec := httptest.NewRecorder()
			rec.Header().Set("X-Request-ID", "test-id")
			handler.ServeHTTP(rec, req)
			if rec.Code != tc.status || called != (tc.status == 204) {
				t.Fatalf("status %d next %v", rec.Code, called)
			}
			if tc.authenticated && (stub.id != 42 || len(stub.roles) != 1 || stub.roles[0] != service.RolePlatformAdmin) {
				t.Fatal("client or caller mutated policy")
			}
			if !tc.authenticated && stub.calls != 0 {
				t.Fatal("anonymous authorization lookup")
			}
			if strings.Contains(rec.Body.String()+logs.String(), "secret") {
				t.Fatal("secret leaked")
			}
		})
	}
}
