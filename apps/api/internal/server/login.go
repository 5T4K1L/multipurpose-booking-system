package server

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"time"
	"unicode/utf8"

	"multipurpose-booking-system/api/internal/service"
)

type passwordLogin interface {
	Login(context.Context, string, string) (service.IssuedSession, error)
}

type LoginHandler struct {
	attemptLimiter
	login  passwordLogin
	origin string
	secure bool
	logger *slog.Logger
	slots  chan struct{}
}

// HTTP cookies are permitted only for an explicitly loopback web origin.
// HTTPS deployments always receive a Secure, host-only __Host- cookie.
func NewLoginHandler(login passwordLogin, origin string, logger *slog.Logger) (*LoginHandler, error) {
	secure, err := sessionCookieSecurity(origin)
	if err != nil {
		return nil, err
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &LoginHandler{attemptLimiter: newAttemptLimiter(10), login: login, origin: origin, secure: secure, logger: logger, slots: make(chan struct{}, 2)}, nil
}

func (h *LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Vary", "Origin")
	var randomID [16]byte
	if _, err := rand.Read(randomID[:]); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	requestID := hex.EncodeToString(randomID[:])
	w.Header().Set("X-Request-ID", requestID)
	fail := func(status int, code, message string) {
		h.logger.Info("login", "request_id", requestID, "outcome", code)
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": message, "request_id": requestID}})
	}
	if r.Method != http.MethodPost && r.Method != http.MethodOptions {
		w.Header().Set("Allow", "POST, OPTIONS")
		fail(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Use POST.")
		return
	}
	// Require an exact Origin even for non-browser clients. Reject null/missing
	// origins, form POSTs, and cross-origin login CSRF before credential processing.
	if r.Header.Get("Origin") != h.origin {
		fail(http.StatusForbidden, "ORIGIN_REJECTED", "Request origin is not allowed.")
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", h.origin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if !h.allow(r.RemoteAddr) {
		w.Header().Set("Retry-After", "900")
		fail(429, "RATE_LIMITED", "Please try again later.")
		return
	}
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		fail(415, "UNSUPPORTED_MEDIA_TYPE", "Use application/json.")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 4096))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			fail(413, "BODY_TOO_LARGE", "Request body is too large.")
		} else {
			fail(400, "INVALID_JSON", "Provide one valid JSON object.")
		}
		return
	}
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if !utf8.Valid(body) || decoder.Decode(&input) != nil {
		fail(400, "INVALID_JSON", "Provide one valid JSON object.")
		return
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		fail(400, "INVALID_JSON", "Provide one valid JSON object.")
		return
	}
	select {
	case h.slots <- struct{}{}:
		defer func() { <-h.slots }()
	default:
		w.Header().Set("Retry-After", "1")
		fail(429, "RATE_LIMITED", "Please try again later.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	issued, err := h.login.Login(ctx, input.Email, input.Password)
	switch {
	case errors.Is(err, service.ErrInvalidLoginInput):
		fail(400, "VALIDATION_ERROR", "Provide a valid email and password.")
		return
	case errors.Is(err, service.ErrInvalidCredentials), errors.Is(err, service.ErrAccountDisabled), errors.Is(err, service.ErrInvalidSession):
		fail(401, "INVALID_CREDENTIALS", "Unable to sign in with those credentials.")
		return
	case err != nil:
		fail(503, "LOGIN_UNAVAILABLE", "Sign-in is temporarily unavailable.")
		return
	}
	// A cookie must only be issued for a successful, well-formed, unexpired session.
	raw, err := base64.RawURLEncoding.Strict().DecodeString(issued.Token)
	if err != nil || len(raw) != 32 || len(issued.Token) != 43 || !issued.ExpiresAt.After(time.Now()) {
		fail(503, "LOGIN_UNAVAILABLE", "Sign-in is temporarily unavailable.")
		return
	}
	name := sessionCookieName(h.secure)
	maxAge := int(time.Until(issued.ExpiresAt).Seconds())
	if maxAge < 1 {
		maxAge = 1
	}
	http.SetCookie(w, &http.Cookie{Name: name, Value: issued.Token, Path: "/", HttpOnly: true, Secure: h.secure, SameSite: http.SameSiteLaxMode, Expires: issued.ExpiresAt.UTC(), MaxAge: maxAge})
	h.logger.Info("login", "request_id", requestID, "outcome", "accepted")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Signed in."})
}
