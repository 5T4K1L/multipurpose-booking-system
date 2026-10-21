package server

import (
	"context"
	"encoding/json"
	"errors"
	"multipurpose-booking-system/api/internal/service"
	"net/http"
	"time"
)

func registerCustomerBookingCancel(mux *http.ServeMux, sessions sessionValidator, businesses *service.BusinessService, origin string) error {
	authn, e := NewAuthenticationMiddleware(sessions, origin, nil)
	if e != nil {
		return e
	}
	limiter := newAttemptLimiter(60)
	protected := authn.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, ok := IdentityFromContext(r.Context())
		if !ok {
			catalogFailure(w, 401, "UNAUTHENTICATED")
			return
		}
		if !limiter.allow(r.RemoteAddr) {
			w.Header().Set("Retry-After", "900")
			catalogFailure(w, 429, "RATE_LIMITED")
			return
		}
		id, ok := catalogUUID(r.PathValue("id"))
		if !ok || r.URL.RawQuery != "" || len(r.Header.Values("Idempotency-Key")) != 1 {
			catalogFailure(w, 400, "VALIDATION_ERROR")
			return
		}
		if _, ok := decodeCatalogJSON[struct{}](w, r); !ok {
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		result, e := businesses.CancelCustomerBooking(ctx, actor.UserID, id, r.Header.Get("Idempotency-Key"))
		switch {
		case errors.Is(e, service.ErrInvalidBooking):
			catalogFailure(w, 400, "VALIDATION_ERROR")
		case errors.Is(e, service.ErrForbidden):
			catalogFailure(w, 403, "FORBIDDEN")
		case errors.Is(e, service.ErrBookingNotFound):
			catalogFailure(w, 404, "NOT_FOUND")
		case errors.Is(e, service.ErrBookingState):
			catalogFailure(w, 409, "BOOKING_STATE_CONFLICT")
		case errors.Is(e, service.ErrIdempotencyConflict):
			catalogFailure(w, 409, "IDEMPOTENCY_CONFLICT")
		case e != nil:
			catalogFailure(w, 503, "BOOKING_UNAVAILABLE")
		default:
			_ = json.NewEncoder(w).Encode(result)
		}
	}))
	mux.Handle("/api/v1/customer/bookings/{id}/cancel", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Add("Vary", "Origin")
		if r.Header.Get("Origin") != origin {
			catalogFailure(w, 403, "ORIGIN_REJECTED")
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Allow", "POST, OPTIONS")
		if r.Method == "OPTIONS" {
			if m := r.Header.Get("Access-Control-Request-Method"); m != "" && m != "POST" {
				catalogFailure(w, 405, "METHOD_NOT_ALLOWED")
				return
			}
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Idempotency-Key")
			w.WriteHeader(204)
			return
		}
		if r.Method != "POST" {
			catalogFailure(w, 405, "METHOD_NOT_ALLOWED")
			return
		}
		protected.ServeHTTP(w, r)
	}))
	return nil
}
