package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"multipurpose-booking-system/api/internal/db"
	"multipurpose-booking-system/api/internal/service"
)

// Identity contains only the authenticated local identity, not authorization claims.
type Identity struct{ UserID int64 }
type identityContextKey struct{}

// IdentityFromContext returns a value copy. Only this package can set the key.
func IdentityFromContext(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(identityContextKey{}).(Identity)
	return identity, ok && identity.UserID > 0
}

type sessionValidator interface {
	Validate(context.Context, string) (db.User, error)
}

type AuthenticationMiddleware struct {
	sessions   sessionValidator
	cookieName string
	logger     *slog.Logger
}

func NewAuthenticationMiddleware(sessions sessionValidator, origin string, logger *slog.Logger) (*AuthenticationMiddleware, error) {
	if sessions == nil {
		return nil, errors.New("session validator is required")
	}
	secure, err := sessionCookieSecurity(origin)
	if err != nil {
		return nil, err
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &AuthenticationMiddleware{sessions: sessions, cookieName: sessionCookieName(secure), logger: logger}, nil
}

// Require wraps protected handlers only. CORS/preflight and mutation CSRF checks
// belong outside this authentication layer; authentication does not grant access
// to a tenant or resource. Public login, registration and logout stay unwrapped.
func (m *AuthenticationMiddleware) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		var id [16]byte
		if _, err := rand.Read(id[:]); err != nil {
			w.WriteHeader(503)
			return
		}
		requestID := hex.EncodeToString(id[:])
		w.Header().Set("X-Request-ID", requestID)
		fail := func(status int, code, message string) {
			m.logger.Info("authentication", "request_id", requestID, "outcome", code)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": message, "request_id": requestID}})
		}
		var token string
		count := 0
		for _, cookie := range r.Cookies() {
			if cookie.Name == m.cookieName {
				count++
				token = cookie.Value
			}
		}
		if count != 1 || token == "" {
			fail(401, "UNAUTHENTICATED", "Authentication is required.")
			return
		}
		// SessionService validates format, persisted expiry/revocation and current
		// account status, and returns no identity unless the transaction commits.
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		user, err := m.sessions.Validate(ctx, token)
		cancel()
		if errors.Is(err, service.ErrInvalidSession) || errors.Is(err, service.ErrAccountDisabled) {
			fail(401, "UNAUTHENTICATED", "Authentication is required.")
			return
		}
		if err != nil {
			fail(503, "AUTHENTICATION_UNAVAILABLE", "Authentication is temporarily unavailable.")
			return
		}
		if user.ID <= 0 || user.Status != "active" {
			fail(401, "UNAUTHENTICATED", "Authentication is required.")
			return
		}
		// Never trust user IDs from headers, query strings, bodies or an earlier context.
		// Use the original request context so the validation timeout doesn't cancel next.
		trusted := context.WithValue(r.Context(), identityContextKey{}, Identity{UserID: user.ID})
		next.ServeHTTP(w, r.WithContext(trusted))
	})
}
