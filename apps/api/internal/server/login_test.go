package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"multipurpose-booking-system/api/internal/service"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type loginStub struct {
	err    error
	calls  int
	issued service.IssuedSession
}

func (s *loginStub) Login(context.Context, string, string) (service.IssuedSession, error) {
	s.calls++
	return s.issued, s.err
}

const loginBody = `{"email":"private@example.com","password":"private password"}`

func goodSession() service.IssuedSession {
	return service.IssuedSession{Token: base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32)), ExpiresAt: time.Now().Add(time.Hour)}
}
func loginRequest(body, origin string) *http.Request {
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", origin)
	return req
}

func TestLoginCookiePolicy(t *testing.T) {
	for _, origin := range []string{"http://localhost:3000", "https://booking.example"} {
		t.Run(origin, func(t *testing.T) {
			stub := &loginStub{issued: goodSession()}
			var logs bytes.Buffer
			handler, err := NewLoginHandler(stub, origin, slog.New(slog.NewTextHandler(&logs, nil)))
			if err != nil {
				t.Fatal(err)
			}
			request := loginRequest(loginBody, origin)
			request.AddCookie(&http.Cookie{Name: "booking_session", Value: "attacker-fixed-token"})
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, request)
			if rec.Code != 200 {
				t.Fatal(rec.Code)
			}
			cookies := rec.Result().Cookies()
			if len(cookies) != 1 {
				t.Fatal("missing session cookie")
			}
			cookie := cookies[0]
			secure := strings.HasPrefix(origin, "https:")
			if cookie.Value != stub.issued.Token || !cookie.HttpOnly || cookie.Secure != secure || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" || cookie.Domain != "" || cookie.MaxAge < 3590 || cookie.MaxAge > 3600 || cookie.Expires.IsZero() {
				t.Fatal("incorrect cookie policy")
			}
			if secure && cookie.Name != "__Host-booking_session" {
				t.Fatal("production cookie missing host prefix")
			}
			if !secure && cookie.Name != "booking_session" {
				t.Fatal("unexpected development cookie")
			}
			for _, secret := range []string{stub.issued.Token, "private@example.com", "private password", "attacker-fixed-token"} {
				if strings.Contains(rec.Body.String()+logs.String(), secret) {
					t.Fatal("secret exposed")
				}
			}
			if rec.Header().Get("Access-Control-Allow-Credentials") != "true" || rec.Header().Get("Access-Control-Allow-Origin") != origin || rec.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("incorrect headers")
			}
		})
	}
	for _, origin := range []string{"", "null", "*", "http://booking.example", "https://user:secret@booking.example", "https://booking.example/path", "https://booking.example?x=1"} {
		if _, err := NewLoginHandler(&loginStub{}, origin, nil); err == nil {
			t.Fatalf("unsafe origin accepted: %s", origin)
		}
	}
}

func TestLoginFailures(t *testing.T) {
	for _, tc := range []struct {
		name, body, origin, contentType string
		err                             error
		status                          int
	}{
		{name: "wrong password", err: service.ErrInvalidCredentials, status: 401},
		{name: "disabled", err: service.ErrAccountDisabled, status: 401},
		{name: "account removed", err: service.ErrInvalidSession, status: 401},
		{name: "invalid input", err: service.ErrInvalidLoginInput, status: 400},
		{name: "storage failure", err: errors.New("database secret"), status: 503},
		{name: "bad JSON", body: "{", status: 400},
		{name: "unknown field", body: `{"admin":true}`, status: 400},
		{name: "trailing object", body: loginBody + "{}", status: 400},
		{name: "large body", body: strings.Repeat(" ", 4097), status: 413},
		{name: "form post", contentType: "application/x-www-form-urlencoded", status: 415},
		{name: "foreign origin", origin: "https://evil.example", status: 403},
		{name: "missing origin", origin: "none", status: 403},
		{name: "invalid UTF8", body: "{\"email\":\"\xff\"}", status: 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stub := &loginStub{err: tc.err, issued: goodSession()}
			handler, _ := NewLoginHandler(stub, "http://localhost:3000", nil)
			body := tc.body
			if body == "" {
				body = loginBody
			}
			origin := tc.origin
			if origin == "" {
				origin = "http://localhost:3000"
			}
			if origin == "none" {
				origin = ""
			}
			req := loginRequest(body, origin)
			if tc.contentType != "" {
				req.Header.Set("Content-Type", tc.contentType)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tc.status || len(rec.Result().Cookies()) != 0 {
				t.Fatalf("status %d, cookies %d", rec.Code, len(rec.Result().Cookies()))
			}
			if strings.Contains(rec.Body.String(), "database secret") {
				t.Fatal("internal details exposed")
			}
			if tc.err == nil && stub.calls != 0 {
				t.Fatal("invalid request reached authentication")
			}
			if tc.status == 401 {
				var response struct {
					Error struct{ Code, Message string }
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				if response.Error.Code != "INVALID_CREDENTIALS" || response.Error.Message != "Unable to sign in with those credentials." {
					t.Fatal("account state disclosed")
				}
			}
		})
	}
}

func TestLoginRateLimitAndPreflight(t *testing.T) {
	stub := &loginStub{err: service.ErrInvalidCredentials}
	handler, _ := NewLoginHandler(stub, "http://localhost:3000", nil)
	req := loginRequest("", "http://localhost:3000")
	req.Method = "OPTIONS"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != 204 || stub.calls != 0 {
		t.Fatal("preflight invoked authentication")
	}
	for i := 0; i < 11; i++ {
		req := loginRequest(loginBody, "http://localhost:3000")
		req.Header.Set("X-Forwarded-For", string(rune('a'+i)))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		want := 401
		if i == 10 {
			want = 429
		}
		if rec.Code != want {
			t.Fatalf("attempt %d: %d", i, rec.Code)
		}
	}
	handler.slots <- struct{}{}
	handler.slots <- struct{}{}
	req = loginRequest(loginBody, "http://localhost:3000")
	req.RemoteAddr = "198.51.100.4:1234"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != 429 {
		t.Fatal("concurrency limit ignored")
	}
}
