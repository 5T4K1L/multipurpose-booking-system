package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"multipurpose-booking-system/api/internal/service"
	"net/http"
	"time"
)

type roleAuthorizer interface {
	RequireAnyRole(context.Context, int64, ...service.Role) error
}

// RequireRoles must run inside AuthenticationMiddleware.Require. allowed is a
// route policy written by the server, never taken from request headers or input.
func RequireRoles(authorizer roleAuthorizer, logger *slog.Logger, allowed ...service.Role) func(http.Handler) http.Handler {
	policy := append([]service.Role(nil), allowed...)
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			fail := func(status int, code, message string) {
				requestID := w.Header().Get("X-Request-ID")
				logger.Info("authorization", "request_id", requestID, "outcome", code)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": message, "request_id": requestID}})
			}
			identity, ok := IdentityFromContext(r.Context())
			if !ok {
				fail(401, "UNAUTHENTICATED", "Authentication is required.")
				return
			}
			if authorizer == nil {
				fail(503, "AUTHORIZATION_UNAVAILABLE", "Authorization is temporarily unavailable.")
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
			err := authorizer.RequireAnyRole(ctx, identity.UserID, policy...)
			cancel()
			if errors.Is(err, service.ErrForbidden) {
				fail(403, "FORBIDDEN", "Access is not allowed.")
				return
			}
			if err != nil {
				fail(503, "AUTHORIZATION_UNAVAILABLE", "Authorization is temporarily unavailable.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
