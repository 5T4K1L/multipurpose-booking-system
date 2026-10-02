package server

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"multipurpose-booking-system/api/internal/db"
	"multipurpose-booking-system/api/internal/service"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type validatorStub struct {
	user  db.User
	err   error
	calls int
	token string
}

func (s *validatorStub) Validate(ctx context.Context, token string) (db.User, error) {
	s.calls++
	s.token = token
	if _, ok := ctx.Deadline(); !ok {
		panic("missing validation timeout")
	}
	return s.user, s.err
}

func TestAuthenticationMiddleware(t *testing.T) {
	for _, tc := range []struct {
		name, cookie  string
		user          db.User
		err           error
		status, calls int
	}{
		{"valid", "booking_session=secret", db.User{ID: 42, Status: "active"}, nil, 204, 1},
		{"missing", "", db.User{}, nil, 401, 0},
		{"empty", "booking_session=", db.User{}, nil, 401, 0},
		{"duplicate", "booking_session=secret; booking_session=other", db.User{}, nil, 401, 0},
		{"wrong cookie", "__Host-booking_session=secret", db.User{}, nil, 401, 0},
		{"invalid session", "booking_session=secret", db.User{}, service.ErrInvalidSession, 401, 1},
		{"disabled error", "booking_session=secret", db.User{}, service.ErrAccountDisabled, 401, 1},
		{"storage failure", "booking_session=secret", db.User{ID: 42, Status: "active"}, service.ErrSessionUnavailable, 503, 1},
		{"unexpected error", "booking_session=secret", db.User{}, errors.New("private secret"), 503, 1},
		{"invalid user", "booking_session=secret", db.User{}, nil, 401, 1},
		{"disabled user", "booking_session=secret", db.User{ID: 42, Status: "disabled"}, nil, 401, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stub := &validatorStub{user: tc.user, err: tc.err}
			var logs bytes.Buffer
			m, err := NewAuthenticationMiddleware(stub, "http://localhost:3000", slog.New(slog.NewJSONHandler(&logs, nil)))
			if err != nil {
				t.Fatal(err)
			}
			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				identity, ok := IdentityFromContext(r.Context())
				if !ok || identity.UserID != 42 {
					t.Fatal("untrusted identity")
				}
				if r.Context().Err() != nil {
					t.Fatal("validation timeout cancelled handler")
				}
				identity.UserID = 99
				again, _ := IdentityFromContext(r.Context())
				if again.UserID != 42 {
					t.Fatal("identity was mutable")
				}
				w.WriteHeader(204)
			})
			req := httptest.NewRequest("POST", "/protected?user_id=99", strings.NewReader(`{"user_id":99}`))
			req.Header.Set("Cookie", tc.cookie)
			req.Header.Set("X-User-ID", "99")
			req.Header.Set("Authorization", "Bearer secret")
			req = req.WithContext(context.WithValue(req.Context(), identityContextKey{}, Identity{UserID: 99}))
			rec := httptest.NewRecorder()
			m.Require(next).ServeHTTP(rec, req)
			if rec.Code != tc.status || stub.calls != tc.calls || called != (tc.status == 204) {
				t.Fatalf("status=%d calls=%d next=%v", rec.Code, stub.calls, called)
			}
			if tc.calls == 1 && stub.token != "secret" {
				t.Fatal("wrong credential used")
			}
			if len(rec.Result().Cookies()) != 0 {
				t.Fatal("middleware changed session cookie")
			}
			if rec.Header().Get("Cache-Control") != "no-store" || rec.Header().Get("X-Request-ID") == "" {
				t.Fatal("missing security headers")
			}
			if strings.Contains(rec.Body.String()+logs.String(), "secret") {
				t.Fatal("secret leaked")
			}
		})
	}
}

func TestAuthenticationConfigurationAndContext(t *testing.T) {
	if _, ok := IdentityFromContext(context.Background()); ok {
		t.Fatal("anonymous context authenticated")
	}
	ctx := context.WithValue(context.Background(), "user_id", int64(42))
	if _, ok := IdentityFromContext(ctx); ok {
		t.Fatal("string key trusted")
	}
	if _, err := NewAuthenticationMiddleware(nil, "http://localhost:3000", nil); err == nil {
		t.Fatal("nil validator accepted")
	}
	stub := &validatorStub{user: db.User{ID: 42, Status: "active"}}
	if _, err := NewAuthenticationMiddleware(stub, "http://example.com", nil); err == nil {
		t.Fatal("insecure origin accepted")
	}
	m, err := NewAuthenticationMiddleware(stub, "https://example.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		cookie string
		status int
	}{{"booking_session=secret", 401}, {"__Host-booking_session=secret", 204}} {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Cookie", tc.cookie)
		rec := httptest.NewRecorder()
		m.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })).ServeHTTP(rec, req)
		if rec.Code != tc.status {
			t.Fatalf("HTTPS cookie policy: %d", rec.Code)
		}
	}
}
