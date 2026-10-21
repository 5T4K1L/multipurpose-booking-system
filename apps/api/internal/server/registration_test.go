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
	"time"
)

type registrationStub struct {
	err   error
	calls int
}

func (s *registrationStub) Register(context.Context, service.RegistrationInput) error {
	s.calls++
	return s.err
}

const registrationBody = `{"email":"private@example.com","username":"Name","password":"private password value"}`

func TestRegistrationHTTP(t *testing.T) {
	for _, tc := range []struct {
		name, method, body, contentType, origin string
		err                                     error
		status                                  int
		calls                                   int
	}{
		{name: "duplicate email", method: "POST", body: registrationBody, contentType: "application/json", err: service.ErrEmailRegistered, status: 409, calls: 1},
		{name: "accepted", method: "POST", body: registrationBody, contentType: "application/json", status: 202, calls: 1},
		{name: "validation", method: "POST", body: registrationBody, contentType: "application/json", err: service.ErrInvalidRegistration, status: 400, calls: 1},
		{name: "internal error", method: "POST", body: registrationBody, contentType: "application/json", err: errors.New("secret DB password"), status: 503, calls: 1},
		{name: "method", method: "GET", status: 405},
		{name: "type", method: "POST", body: registrationBody, contentType: "text/plain", status: 415},
		{name: "bad JSON", method: "POST", body: "{", contentType: "application/json", status: 400},
		{name: "unknown field", method: "POST", body: `{"admin":true}`, contentType: "application/json", status: 400},
		{name: "two objects", method: "POST", body: registrationBody + "{}", contentType: "application/json", status: 400},
		{name: "oversize", method: "POST", body: `{"email":"` + strings.Repeat("a", 5000) + `"}`, contentType: "application/json", status: 413},
		{name: "bad origin", method: "POST", body: registrationBody, origin: "https://evil.example", contentType: "application/json", status: 403},
		{name: "preflight", method: "OPTIONS", origin: "http://localhost:3000", status: 204},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stub := &registrationStub{err: tc.err}
			var logs bytes.Buffer
			handler := NewRegistrationHandler(stub, "http://localhost:3000", slog.New(slog.NewJSONHandler(&logs, nil)))
			req := httptest.NewRequest(tc.method, "/api/v1/auth/register", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", tc.contentType)
			req.Header.Set("Origin", tc.origin)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tc.status || stub.calls != tc.calls {
				t.Fatalf("status %d, calls %d", rec.Code, stub.calls)
			}
			for _, secret := range []string{"private@example.com", "private password value", "secret DB password"} {
				if strings.Contains(rec.Body.String()+logs.String(), secret) {
					t.Fatal("secret leaked")
				}
			}
			if rec.Header().Get("Cache-Control") != "no-store" || len(rec.Header().Get("X-Request-ID")) != 32 {
				t.Fatal("missing response protection")
			}
			if len(rec.Result().Cookies()) != 0 {
				t.Fatal("registration established a session")
			}
		})
	}
}

func TestRegistrationRateLimits(t *testing.T) {
	stub := &registrationStub{}
	handler := NewRegistrationHandler(stub, "", nil)
	now := time.Now()
	handler.now = func() time.Time { return now }
	for attempt := 0; attempt < 6; attempt++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(registrationBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", string(rune('a'+attempt)))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		want := 202
		if attempt == 5 {
			want = 429
		}
		if rec.Code != want {
			t.Fatalf("attempt %d: %d", attempt, rec.Code)
		}
	}
	now = now.Add(15 * time.Minute)
	if !handler.allow("192.0.2.1:1234") {
		t.Fatal("window did not reset")
	}
	handler.slots <- struct{}{}
	handler.slots <- struct{}{}
	req := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(registrationBody))
	req.RemoteAddr = "198.51.100.1:999"
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != 429 {
		t.Fatal("hash concurrency limit not enforced")
	}
}
