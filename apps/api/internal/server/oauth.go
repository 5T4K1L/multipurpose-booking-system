package server

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-jose/go-jose/v4"
	"multipurpose-booking-system/api/internal/auth"
	"multipurpose-booking-system/api/internal/service"
)

type oauthLogin interface {
	LoginOAuth(context.Context, auth.OAuthIdentity) (service.IssuedSession, error)
}

// OAuthRoutes owns short-lived, one-use transactions. Restart loses pending flows
// safely. Multi-instance deployments require affinity or a shared transaction store.
type OAuthRoutes struct {
	attemptLimiter
	providers map[auth.OAuthProvider]auth.OAuthProviderConfig
	codec     *auth.OAuthTransactionCodec
	login     oauthLogin
	origin    string
	secure    bool
	logger    *slog.Logger
	mu        sync.Mutex
	pending   map[[32]byte]time.Time
	client    *http.Client
}

func NewOAuthRoutes(config auth.OAuthRuntimeConfig, login oauthLogin, origin, apiOrigin string, logger *slog.Logger) (*OAuthRoutes, error) {
	secure, err := sessionCookieSecurity(origin)
	if err != nil {
		return nil, err
	}
	apiSecure, err := sessionCookieSecurity(apiOrigin)
	if (config.Google.Provider != "" || config.Apple.Provider != "") && (err != nil || apiSecure != secure) {
		return nil, errors.New("API_PUBLIC_ORIGIN must be an exact origin using the same HTTP/HTTPS policy as WEB_ORIGIN")
	}
	if logger == nil {
		logger = slog.Default()
	}
	h := &OAuthRoutes{attemptLimiter: newAttemptLimiter(30), providers: make(map[auth.OAuthProvider]auth.OAuthProviderConfig), login: login, origin: origin, secure: secure, logger: logger, pending: make(map[[32]byte]time.Time), client: &http.Client{Timeout: 10 * time.Second}}
	for _, cfg := range []auth.OAuthProviderConfig{config.Google, config.Apple} {
		if cfg.Provider == "" {
			continue
		}
		if err := cfg.Validate(); err != nil {
			return nil, err
		}
		if cfg.RedirectURL != apiOrigin+"/api/v1/auth/"+string(cfg.Provider)+"/callback" {
			return nil, errors.New("OAuth redirect URL must match API_PUBLIC_ORIGIN and the provider callback path")
		}
		if cfg.Provider == auth.OAuthProviderApple {
			parsed, _ := url.Parse(apiOrigin)
			if !secure || strings.EqualFold(parsed.Hostname(), "localhost") || net.ParseIP(parsed.Hostname()) != nil {
				return nil, errors.New("Apple OAuth requires HTTPS and a public domain callback")
			}
		}
		h.providers[cfg.Provider] = cfg
	}
	if len(h.providers) > 0 {
		if login == nil {
			return nil, errors.New("OAuth login service is required")
		}
		h.codec, err = auth.NewOAuthTransactionCodec(config.TransactionKey)
		if err != nil {
			return nil, err
		}
	}
	return h, nil
}

func (h *OAuthRoutes) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/auth/providers", h.Providers)
	for _, p := range []auth.OAuthProvider{auth.OAuthProviderGoogle, auth.OAuthProviderApple} {
		mux.HandleFunc("/api/v1/auth/"+string(p), func(w http.ResponseWriter, r *http.Request) { h.Start(w, r, p) })
		mux.HandleFunc("/api/v1/auth/"+string(p)+"/callback", func(w http.ResponseWriter, r *http.Request) { h.Callback(w, r, p) })
	}
}
func oauthHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}
func (h *OAuthRoutes) Providers(w http.ResponseWriter, r *http.Request) {
	oauthHeaders(w)
	w.Header().Set("Vary", "Origin")
	if origin := r.Header.Get("Origin"); origin != "" && origin != h.origin {
		http.Error(w, "Origin not allowed", 403)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", h.origin)
	if r.Method != "GET" {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(405)
		return
	}
	names := []string{}
	for _, p := range []auth.OAuthProvider{auth.OAuthProviderGoogle, auth.OAuthProviderApple} {
		if _, ok := h.providers[p]; ok {
			names = append(names, string(p))
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"providers": names})
}
func (h *OAuthRoutes) cookie(p auth.OAuthProvider, value string, clear bool) *http.Cookie {
	name := "oauth_" + string(p)
	if h.secure {
		name = "__Host-" + name
	}
	sameSite := http.SameSiteLaxMode
	if p == auth.OAuthProviderApple {
		sameSite = http.SameSiteNoneMode
	}
	c := &http.Cookie{Name: name, Value: value, Path: "/", HttpOnly: true, Secure: h.secure, SameSite: sameSite, MaxAge: 600}
	if clear {
		c.MaxAge = -1
		c.Expires = time.Unix(1, 0).UTC()
	}
	return c
}
func (h *OAuthRoutes) remember(value string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := time.Now()
	for key, expiry := range h.pending {
		if !now.Before(expiry) {
			delete(h.pending, key)
		}
	}
	if len(h.pending) >= 10000 {
		return false
	}
	h.pending[sha256.Sum256([]byte(value))] = now.Add(10 * time.Minute)
	return true
}
func (h *OAuthRoutes) consume(value string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	key := sha256.Sum256([]byte(value))
	expiry, ok := h.pending[key]
	delete(h.pending, key)
	return ok && time.Now().Before(expiry)
}
func (h *OAuthRoutes) Start(w http.ResponseWriter, r *http.Request, p auth.OAuthProvider) {
	oauthHeaders(w)
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(405)
		return
	}
	if r.Header.Get("Origin") != h.origin {
		http.Error(w, "Origin not allowed", 403)
		return
	}
	cfg, ok := h.providers[p]
	if !ok {
		http.NotFound(w, r)
		return
	}
	if !h.allow(r.RemoteAddr) {
		w.Header().Set("Retry-After", "900")
		w.WriteHeader(429)
		return
	}
	a, err := auth.BuildOAuthAuthorization(r.Context(), cfg, time.Now())
	if err != nil {
		w.WriteHeader(503)
		return
	}
	if p == auth.OAuthProviderApple {
		u, _ := url.Parse(a.URL)
		q := u.Query()
		q.Set("response_mode", "form_post")
		u.RawQuery = q.Encode()
		a.URL = u.String()
	}
	if auth.ValidateAuthorizationURL(a.URL) != nil {
		w.WriteHeader(503)
		return
	}
	value, err := h.codec.Encode(auth.OAuthTransaction{Provider: p, State: a.State.State, Nonce: a.State.Nonce, PKCEVerifier: a.PKCEVerifier, CreatedAtUnix: time.Now().Unix()})
	if err != nil || !h.remember(value) {
		w.WriteHeader(503)
		return
	}
	// Superseding a browser's pending flow cannot leave its earlier cookie usable.
	if previous, err := r.Cookie(h.cookie(p, "", false).Name); err == nil {
		h.consume(previous.Value)
	}
	http.SetCookie(w, h.cookie(p, value, false))
	http.Redirect(w, r, a.URL, http.StatusSeeOther)
}
func (h *OAuthRoutes) Callback(w http.ResponseWriter, r *http.Request, p auth.OAuthProvider) {
	oauthHeaders(w)
	cfg, ok := h.providers[p]
	if !ok {
		http.NotFound(w, r)
		return
	}
	expected := "GET"
	if p == auth.OAuthProviderApple {
		expected = "POST"
	}
	if r.Method != expected {
		w.Header().Set("Allow", expected)
		w.WriteHeader(405)
		return
	}
	fail := func(code string) {
		h.logger.Info("oauth_callback", "provider", p, "outcome", code)
		http.Redirect(w, r, h.origin+"/oauth/error", http.StatusSeeOther)
	}
	http.SetCookie(w, h.cookie(p, "", true))
	if !h.allow(r.RemoteAddr) {
		fail("unavailable")
		return
	}
	var values url.Values
	if p == auth.OAuthProviderApple {
		media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || media != "application/x-www-form-urlencoded" {
			fail("failed")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 8192)
		if r.ParseForm() != nil {
			fail("failed")
			return
		}
		values = r.PostForm
	} else {
		if len(r.URL.RawQuery) > 8192 {
			fail("failed")
			return
		}
		var err error
		values, err = url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			fail("failed")
			return
		}
	}
	for _, key := range []string{"state", "code", "error"} {
		if len(values[key]) > 1 {
			fail("failed")
			return
		}
	}
	count := 0
	value := ""
	for _, c := range r.Cookies() {
		if c.Name == h.cookie(p, "", false).Name {
			count++
			value = c.Value
		}
	}
	if count != 1 || len(value) > 4096 {
		fail("failed")
		return
	}
	tx, err := h.codec.Decode(value, time.Now())
	if err != nil || tx.Provider != p || !auth.ValidateOAuthState(tx.State, values.Get("state")) {
		fail("failed")
		return
	}
	if !h.consume(value) {
		fail("failed")
		return
	}
	if values.Get("error") != "" || values.Get("code") == "" {
		fail("failed")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	identity, err := h.verify(ctx, cfg, values.Get("code"), tx)
	if err != nil {
		fail("failed")
		return
	}
	issued, err := h.login.LoginOAuth(ctx, identity)
	if err != nil {
		// Do not disclose whether an email matches an existing account or is disabled.
		fail("failed")
		return
	}
	raw, err := base64.RawURLEncoding.Strict().DecodeString(issued.Token)
	if err != nil || len(raw) != 32 || len(issued.Token) != 43 || !issued.ExpiresAt.After(time.Now()) {
		fail("unavailable")
		return
	}
	age := int(time.Until(issued.ExpiresAt).Seconds())
	if age < 1 {
		age = 1
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName(h.secure), Value: issued.Token, Path: "/", HttpOnly: true, Secure: h.secure, SameSite: http.SameSiteLaxMode, Expires: issued.ExpiresAt.UTC(), MaxAge: age})
	h.logger.Info("oauth_callback", "provider", p, "outcome", "accepted")
	http.Redirect(w, r, h.origin+"/account", http.StatusSeeOther)
}
func (h *OAuthRoutes) verify(ctx context.Context, cfg auth.OAuthProviderConfig, code string, tx auth.OAuthTransaction) (auth.OAuthIdentity, error) {
	token, err := auth.ExchangeOAuthCode(ctx, cfg, code, tx.PKCEVerifier)
	if err != nil {
		return auth.OAuthIdentity{}, err
	}
	verifier, err := auth.NewOIDCVerifier(h.client, cfg.JWKSURL, cfg.Issuer, cfg.ClientID, tx.Nonce, []jose.SignatureAlgorithm{jose.RS256})
	if err != nil {
		return auth.OAuthIdentity{}, err
	}
	claims, err := verifier.Verify(ctx, token.IDToken, time.Now())
	if err != nil {
		return auth.OAuthIdentity{}, err
	}
	return auth.NormalizeOIDCIdentity(cfg.Provider, claims)
}
