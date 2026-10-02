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

type logoutStub struct {
	calls int
	token string
	err   error
}

func (s *logoutStub) Revoke(_ context.Context, token string) error {
	s.calls++
	s.token = token
	return s.err
}

func TestLogoutHandler(t *testing.T) {
	for _, tc := range []struct {
		name, origin, method, body, cookie string
		err                                error
		status, calls                      int
		clear                              bool
	}{
		{"current", "http://localhost:3000", "POST", "", "booking_session=secret", nil, 204, 1, true},
		{"missing", "http://localhost:3000", "POST", "", "", nil, 204, 0, true},
		{"malformed", "http://localhost:3000", "POST", "", "booking_session=bad", service.ErrInvalidSession, 204, 1, true},
		{"storage failure", "http://localhost:3000", "POST", "", "booking_session=secret", service.ErrSessionUnavailable, 503, 1, false},
		{"unexpected failure", "http://localhost:3000", "POST", "", "booking_session=secret", errors.New("secret database error"), 503, 1, false},
		{"foreign origin", "https://evil.example", "POST", "", "booking_session=secret", nil, 403, 0, false},
		{"missing origin", "", "POST", "", "booking_session=secret", nil, 403, 0, false},
		{"get", "http://localhost:3000", "GET", "", "booking_session=secret", nil, 405, 0, false},
		{"preflight", "http://localhost:3000", "OPTIONS", "", "booking_session=secret", nil, 204, 0, false},
		{"body", "http://localhost:3000", "POST", "{}", "booking_session=secret", nil, 400, 0, false},
		{"duplicates", "http://localhost:3000", "POST", "", "booking_session=secret; booking_session=other", nil, 400, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stub := &logoutStub{err: tc.err}
			var logs bytes.Buffer
			h, err := NewLogoutHandler(stub, "http://localhost:3000", slog.New(slog.NewJSONHandler(&logs, nil)))
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(tc.method, "/api/v1/auth/logout", strings.NewReader(tc.body))
			req.Header.Set("Origin", tc.origin)
			req.Header.Set("Cookie", tc.cookie)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tc.status || stub.calls != tc.calls {
				t.Fatalf("status=%d calls=%d", rec.Code, stub.calls)
			}
			cookies := rec.Result().Cookies()
			if (len(cookies) == 1) != tc.clear {
				t.Fatal("incorrect cookie clearing")
			}
			if tc.clear {
				c := cookies[0]
				if c.Name != "booking_session" || c.Value != "" || c.Path != "/" || c.Domain != "" || !c.HttpOnly || c.Secure || c.SameSite != http.SameSiteLaxMode || c.MaxAge != -1 || !c.Expires.Before(time.Now()) {
					t.Fatalf("bad deletion cookie: %+v", c)
				}
			}
			if strings.Contains(rec.Body.String()+logs.String(), "secret") {
				t.Fatal("secret leaked")
			}
			if rec.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("cacheable logout")
			}
			if tc.origin == "http://localhost:3000" && tc.method != "GET" && rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
				t.Fatal("missing credentialed CORS")
			}
		})
	}
}

func TestLogoutSecureCookieAndLimits(t *testing.T) {
	stub := &logoutStub{}
	h, err := NewLogoutHandler(stub, "https://booking.example", slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 61; i++ {
		req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
		req.Header.Set("Origin", "https://booking.example")
		req.AddCookie(&http.Cookie{Name: "__Host-booking_session", Value: "secure-token"})
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if i == 60 {
			if rec.Code != 429 || len(rec.Result().Cookies()) != 0 || stub.calls != 60 {
				t.Fatal("rate limit failed")
			}
			continue
		}
		c := rec.Result().Cookies()[0]
		if rec.Code != 204 || c.Name != "__Host-booking_session" || !c.Secure || c.Path != "/" || c.Domain != "" || c.MaxAge != -1 || stub.token != "secure-token" {
			t.Fatal("HTTPS cookie mismatch")
		}
	}
	for _, origin := range []string{"http://example.com", "https://example.com/path", ""} {
		if _, err := NewLogoutHandler(stub, origin, nil); err == nil {
			t.Fatal("invalid origin accepted")
		}
	}
}
