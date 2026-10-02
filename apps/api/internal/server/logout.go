package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"multipurpose-booking-system/api/internal/service"
)

type sessionRevoker interface {
	Revoke(context.Context, string) error
}

type LogoutHandler struct {
	attemptLimiter
	sessions sessionRevoker
	origin   string
	secure   bool
	logger   *slog.Logger
}

func NewLogoutHandler(sessions sessionRevoker, origin string, logger *slog.Logger) (*LogoutHandler, error) {
	secure, err := sessionCookieSecurity(origin)
	if err != nil {
		return nil, err
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &LogoutHandler{newAttemptLimiter(60), sessions, origin, secure, logger}, nil
}

func (h *LogoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Vary", "Origin")
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		w.WriteHeader(503)
		return
	}
	requestID := hex.EncodeToString(id[:])
	w.Header().Set("X-Request-ID", requestID)
	fail := func(status int, code, message string) {
		h.logger.Info("logout", "request_id", requestID, "outcome", code)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": message, "request_id": requestID}})
	}
	if r.Method != http.MethodPost && r.Method != http.MethodOptions {
		w.Header().Set("Allow", "POST, OPTIONS")
		fail(405, "METHOD_NOT_ALLOWED", "Use POST.")
		return
	}
	// The exact Origin protects this cookie-authenticated mutation against CSRF.
	if r.Header.Get("Origin") != h.origin {
		fail(403, "ORIGIN_REJECTED", "Request origin is not allowed.")
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", h.origin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(204)
		return
	}
	if !h.allow(r.RemoteAddr) {
		w.Header().Set("Retry-After", "900")
		fail(429, "RATE_LIMITED", "Please try again later.")
		return
	}
	// Logout accepts no caller-selected user/session ID or bearer token in a body.
	body, err := io.ReadAll(io.LimitReader(r.Body, 1))
	if err != nil || len(body) != 0 {
		fail(400, "INVALID_BODY", "Send an empty request body.")
		return
	}
	name := sessionCookieName(h.secure)
	var token string
	count := 0
	for _, cookie := range r.Cookies() {
		if cookie.Name == name {
			count++
			token = cookie.Value
		}
	}
	if count > 1 {
		fail(400, "INVALID_COOKIE", "Multiple session cookies were supplied.")
		return
	}
	if count == 1 {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		// Revoke directly so expired sessions and disabled accounts can also log out.
		err = h.sessions.Revoke(ctx, token)
		if err != nil && !errors.Is(err, service.ErrInvalidSession) {
			fail(503, "LOGOUT_UNAVAILABLE", "Sign-out is temporarily unavailable. Please retry.")
			return
		}
	}
	// Clear only after revocation commits (or no usable session exists).
	http.SetCookie(w, &http.Cookie{Name: name, Value: "", Path: "/", HttpOnly: true, Secure: h.secure, SameSite: http.SameSiteLaxMode, Expires: time.Unix(1, 0).UTC(), MaxAge: -1})
	h.logger.Info("logout", "request_id", requestID, "outcome", "accepted")
	w.WriteHeader(http.StatusNoContent)
}
