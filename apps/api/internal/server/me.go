package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// Explicit allowlist: never serialize a database user or authentication record.
type currentUser struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}
type currentUserContextKey struct{}

// NewMeHandler puts method/CORS handling outside authentication so preflight
// requires no session. Missing Origin is allowed for same-origin/non-browser GETs.
func NewMeHandler(sessions sessionValidator, origin string, logger *slog.Logger) (http.Handler, error) {
	middleware, err := NewAuthenticationMiddleware(sessions, origin, logger)
	if err != nil {
		return nil, err
	}
	protected := middleware.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, ok := IdentityFromContext(r.Context())
		user, present := r.Context().Value(currentUserContextKey{}).(currentUser)
		w.Header().Set("Content-Type", "application/json")
		if !ok || !present || user.ID != identity.UserID {
			w.WriteHeader(401)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": "UNAUTHENTICATED", "message": "Authentication is required.", "request_id": w.Header().Get("X-Request-ID")}})
			return
		}
		_ = json.NewEncoder(w).Encode(struct {
			User currentUser `json:"user"`
		}{User: user})
	}))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Add("Vary", "Origin")
		w.Header().Set("Content-Type", "application/json")
		fail := func(status int, code, message string) {
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": message}})
		}
		requestOrigin := r.Header.Get("Origin")
		if requestOrigin != "" && requestOrigin != origin {
			fail(403, "ORIGIN_REJECTED", "Request origin is not allowed.")
			return
		}
		if requestOrigin == origin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		if r.Method == http.MethodOptions {
			if requestOrigin != origin {
				fail(403, "ORIGIN_REJECTED", "Request origin is required for preflight.")
				return
			}
			if method := r.Header.Get("Access-Control-Request-Method"); method != "" && method != http.MethodGet {
				fail(405, "METHOD_NOT_ALLOWED", "Use GET.")
				return
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.WriteHeader(204)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET, OPTIONS")
			fail(405, "METHOD_NOT_ALLOWED", "Use GET.")
			return
		}
		protected.ServeHTTP(w, r)
	}), nil
}
