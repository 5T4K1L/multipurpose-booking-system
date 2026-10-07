package server

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"io"
	"log/slog"
	"multipurpose-booking-system/api/internal/auth"
	"multipurpose-booking-system/api/internal/service"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

type oauthLoginStub struct {
	calls    int
	identity auth.OAuthIdentity
	err      error
}

func (s *oauthLoginStub) LoginOAuth(_ context.Context, i auth.OAuthIdentity) (service.IssuedSession, error) {
	s.calls++
	s.identity = i
	return goodSession(), s.err
}

type oauthFixture struct {
	routes                          *OAuthRoutes
	config                          auth.OAuthProviderConfig
	provider                        auth.OAuthProvider
	nonce, verifier, email, subject string
	exchanges                       int
	logs                            bytes.Buffer
}

func newOAuthFixture(t *testing.T, p auth.OAuthProvider, login oauthLogin) *oauthFixture {
	t.Helper()
	f := &oauthFixture{provider: p, email: "oauth@example.invalid", subject: "provider-subject"}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, &jose.SignerOptions{ExtraHeaders: map[jose.HeaderKey]interface{}{"kid": "mock-key"}})
	if err != nil {
		t.Fatal(err)
	}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/keys" {
			_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: key.Public(), KeyID: "mock-key", Algorithm: "RS256", Use: "sig"}}})
			return
		}
		if err := r.ParseForm(); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		f.exchanges++
		if r.Form.Get("code") != "provider-code" || r.Form.Get("code_verifier") != f.verifier {
			t.Error("wrong code or PKCE verifier")
			w.WriteHeader(400)
			return
		}
		claims := map[string]any{"iss": f.config.Issuer, "aud": f.config.ClientID, "sub": f.subject, "nonce": f.nonce, "iat": time.Now().Add(-time.Minute).Unix(), "exp": time.Now().Add(time.Minute).Unix(), "email": f.email, "email_verified": "true"}
		token, err := jwt.Signed(signer).Claims(claims).Serialize()
		if err != nil {
			t.Error(err)
			w.WriteHeader(500)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "private-access-token", "token_type": "Bearer", "id_token": token})
	}))
	t.Cleanup(provider.Close)
	cfg := auth.GoogleOAuthConfig("client", "private-client-secret", "https://api.example/api/v1/auth/google/callback")
	if p == auth.OAuthProviderApple {
		signingKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		der, err := x509.MarshalPKCS8PrivateKey(signingKey)
		if err != nil {
			t.Fatal(err)
		}
		cfg = auth.AppleOAuthConfig("client", "team", "key", pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), "https://api.example/api/v1/auth/apple/callback")
	}
	cfg.TokenURL = provider.URL + "/token"
	cfg.JWKSURL = provider.URL + "/keys"
	f.config = cfg
	runtime := auth.OAuthRuntimeConfig{TransactionKey: bytes.Repeat([]byte{42}, 32)}
	if p == auth.OAuthProviderGoogle {
		runtime.Google = cfg
	} else {
		runtime.Apple = cfg
	}
	f.routes, err = NewOAuthRoutes(runtime, login, "https://web.example", "https://api.example", slog.New(slog.NewTextHandler(&f.logs, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return f
}
func (f *oauthFixture) start(t *testing.T) *http.Cookie {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/v1/auth/"+string(f.provider), nil)
	req.Header.Set("Origin", "https://web.example")
	rec := httptest.NewRecorder()
	f.routes.Start(rec, req, f.provider)
	if rec.Code != 303 || len(rec.Result().Cookies()) != 1 {
		t.Fatalf("start %d", rec.Code)
	}
	cookie := rec.Result().Cookies()[0]
	tx, err := f.routes.codec.Decode(cookie.Value, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	f.nonce = tx.Nonce
	f.verifier = tx.PKCEVerifier
	u, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if u.Query().Get("state") != tx.State || u.Query().Get("nonce") != tx.Nonce {
		t.Fatal("transaction not bound")
	}
	if f.provider == auth.OAuthProviderApple && (u.Query().Get("response_mode") != "form_post" || cookie.SameSite != http.SameSiteNoneMode || !cookie.Secure) {
		t.Fatal("Apple form_post cookie policy")
	}
	return cookie
}
func (f *oauthFixture) callback(t *testing.T, cookie *http.Cookie, change func(url.Values)) *httptest.ResponseRecorder {
	t.Helper()
	tx, err := f.routes.codec.Decode(cookie.Value, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	values := url.Values{"state": {tx.State}, "code": {"provider-code"}}
	if change != nil {
		change(values)
	}
	var req *http.Request
	if f.provider == auth.OAuthProviderApple {
		req = httptest.NewRequest("POST", "/callback", strings.NewReader(values.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest("GET", "/callback?"+values.Encode(), nil)
	}
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	f.routes.Callback(rec, req, f.provider)
	return rec
}
func TestOAuthRoutesSignedProviderFlow(t *testing.T) {
	for _, p := range []auth.OAuthProvider{auth.OAuthProviderGoogle, auth.OAuthProviderApple} {
		t.Run(string(p), func(t *testing.T) {
			login := &oauthLoginStub{}
			f := newOAuthFixture(t, p, login)
			cookie := f.start(t)
			rec := f.callback(t, cookie, nil)
			if rec.Code != 303 || rec.Header().Get("Location") != "https://web.example/account" || login.calls != 1 || !login.identity.EmailVerified || login.identity.Provider != p {
				t.Fatalf("callback %d %s", rec.Code, rec.Header().Get("Location"))
			}
			cookies := rec.Result().Cookies()
			if len(cookies) != 2 || cookies[0].MaxAge != -1 || cookies[1].Name != "__Host-booking_session" || !cookies[1].HttpOnly || !cookies[1].Secure || cookies[1].SameSite != http.SameSiteLaxMode {
				t.Fatal("cookie handoff failed")
			}
			repeated := f.callback(t, cookie, nil)
			if repeated.Header().Get("Location") != "https://web.example/oauth/error" || login.calls != 1 || f.exchanges != 1 {
				t.Fatal("transaction replay accepted")
			}
			for _, secret := range []string{cookie.Value, goodSession().Token, "private-client-secret", "private-access-token", f.email} {
				if strings.Contains(f.logs.String()+rec.Body.String(), secret) {
					t.Fatal("secret leaked")
				}
			}
		})
	}
}
func TestOAuthRoutesRejectInvalidCallbacks(t *testing.T) {
	for _, tc := range []struct {
		name     string
		mutate   func(url.Values)
		badNonce bool
		loginErr error
	}{
		{name: "state", mutate: func(v url.Values) { v.Set("state", "attacker") }},
		{name: "duplicate", mutate: func(v url.Values) { v.Add("code", "other") }},
		{name: "provider error", mutate: func(v url.Values) { v.Set("error", "private-provider-error") }},
		{name: "nonce", badNonce: true},
		{name: "email linking", loginErr: service.ErrLinkRequired},
		{name: "storage", loginErr: errors.New("private-storage-error")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			login := &oauthLoginStub{err: tc.loginErr}
			f := newOAuthFixture(t, auth.OAuthProviderGoogle, login)
			cookie := f.start(t)
			if tc.badNonce {
				f.nonce = "wrong"
			}
			rec := f.callback(t, cookie, tc.mutate)
			if rec.Code != 303 || rec.Header().Get("Location") != "https://web.example/oauth/error" || len(rec.Result().Cookies()) != 1 {
				t.Fatal("unsafe callback response")
			}
			if tc.loginErr == nil && login.calls != 0 {
				t.Fatal("invalid identity reached login")
			}
			if strings.Contains(f.logs.String()+rec.Body.String(), "private-") {
				t.Fatal("private error exposed")
			}
		})
	}
}
func TestOAuthDisabledAndStartCSRF(t *testing.T) {
	h, err := NewOAuthRoutes(auth.OAuthRuntimeConfig{}, nil, "http://localhost:3000", "http://localhost:8080", nil)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	h.Register(mux)
	for _, path := range []string{"/api/v1/auth/google", "/api/v1/auth/apple"} {
		req := httptest.NewRequest("POST", path, nil)
		req.Header.Set("Origin", "http://localhost:3000")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != 404 {
			t.Fatal("disabled provider enabled")
		}
	}
	req := httptest.NewRequest("GET", "/api/v1/auth/providers", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	body, _ := io.ReadAll(rec.Result().Body)
	if !strings.Contains(string(body), `"providers":[]`) {
		t.Fatal("disabled providers advertised")
	}
	f := newOAuthFixture(t, auth.OAuthProviderGoogle, &oauthLoginStub{})
	for _, origin := range []string{"", "https://attacker.example", "null"} {
		req := httptest.NewRequest("POST", "/start", nil)
		req.Header.Set("Origin", origin)
		rec := httptest.NewRecorder()
		f.routes.Start(rec, req, auth.OAuthProviderGoogle)
		if rec.Code != 403 || len(rec.Result().Cookies()) != 0 {
			t.Fatal("start CSRF")
		}
	}
}

func TestOAuthCookieAndCallbackBounds(t *testing.T) {
	for _, provider := range []auth.OAuthProvider{auth.OAuthProviderGoogle, auth.OAuthProviderApple} {
		login := &oauthLoginStub{}
		f := newOAuthFixture(t, provider, login)
		cookie := f.start(t)
		for _, test := range []string{"missing", "duplicate", "tampered", "wrong-method", "oversized"} {
			t.Run(string(provider)+"/"+test, func(t *testing.T) {
				tx, _ := f.routes.codec.Decode(cookie.Value, time.Now())
				values := url.Values{"state": {tx.State}, "code": {"provider-code"}}
				method := "GET"
				target := "/callback?" + values.Encode()
				var body io.Reader
				if provider == auth.OAuthProviderApple {
					method = "POST"
					target = "/callback"
					body = strings.NewReader(values.Encode())
				}
				if test == "wrong-method" {
					method = "PUT"
				}
				if test == "oversized" {
					if provider == auth.OAuthProviderApple {
						body = strings.NewReader(strings.Repeat("x", 9000))
					} else {
						target = "/callback?" + strings.Repeat("x", 9000)
					}
				}
				req := httptest.NewRequest(method, target, body)
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				if test != "missing" {
					copy := *cookie
					if test == "tampered" {
						copy.Value = "bad"
					}
					req.AddCookie(&copy)
					if test == "duplicate" {
						req.AddCookie(&copy)
					}
				}
				rec := httptest.NewRecorder()
				f.routes.Callback(rec, req, provider)
				if login.calls != 0 || f.exchanges != 0 {
					t.Fatal("invalid callback reached provider")
				}
				for _, c := range rec.Result().Cookies() {
					if c.Name == "__Host-booking_session" {
						t.Fatal("invalid callback issued session")
					}
				}
			})
		}
	}
}
func TestOAuthPendingExpiryAndCapacity(t *testing.T) {
	f := newOAuthFixture(t, auth.OAuthProviderGoogle, &oauthLoginStub{})
	cookie := f.start(t)
	key := sha256.Sum256([]byte(cookie.Value))
	f.routes.pending[key] = time.Now().Add(-time.Second)
	rec := f.callback(t, cookie, nil)
	if rec.Header().Get("Location") != "https://web.example/oauth/error" || f.exchanges != 0 {
		t.Fatal("expired pending flow accepted")
	}
	for i := 0; i < 10000; i++ {
		f.routes.pending[sha256.Sum256([]byte(fmt.Sprint(i)))] = time.Now().Add(time.Hour)
	}
	if f.routes.remember("overflow") {
		t.Fatal("unbounded pending transactions")
	}
}

func TestOAuthOriginConfiguration(t *testing.T) {
	google := auth.GoogleOAuthConfig("client", "secret", "http://localhost:8080/api/v1/auth/google/callback")
	cfg := auth.OAuthRuntimeConfig{Google: google, TransactionKey: make([]byte, 32)}
	h, err := NewOAuthRoutes(cfg, &oauthLoginStub{}, "http://localhost:3000", "http://localhost:8080", nil)
	if err != nil {
		t.Fatal(err)
	}
	cookie := h.cookie(auth.OAuthProviderGoogle, "value", false)
	if strings.HasPrefix(cookie.Name, "__Host-") || cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatal("local Google cookie invalid")
	}
	cfg.Google.RedirectURL = "http://attacker.example/callback"
	if _, err := NewOAuthRoutes(cfg, &oauthLoginStub{}, "http://localhost:3000", "http://localhost:8080", nil); err == nil {
		t.Fatal("callback origin mismatch accepted")
	}
	f := newOAuthFixture(t, auth.OAuthProviderApple, &oauthLoginStub{})
	apple := f.config
	apple.RedirectURL = "http://localhost:8080/api/v1/auth/apple/callback"
	if _, err := NewOAuthRoutes(auth.OAuthRuntimeConfig{Apple: apple, TransactionKey: make([]byte, 32)}, &oauthLoginStub{}, "http://localhost:3000", "http://localhost:8080", nil); err == nil {
		t.Fatal("insecure Apple config accepted")
	}
	// Unused OAuth API settings cannot break existing password-only HTTPS startup.
	if _, err := NewOAuthRoutes(auth.OAuthRuntimeConfig{}, nil, "https://web.example", "http://localhost:8080", nil); err != nil {
		t.Fatal(err)
	}
}
