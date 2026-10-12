package server

import (
	"context"
	"encoding/json"
	"errors"
	"multipurpose-booking-system/api/internal/service"
	"net/http"
	"time"
)

func NewBookingHandler(sessions sessionValidator, bookings *service.BookingService, origin string) (http.Handler, error) {
	authn, err := NewAuthenticationMiddleware(sessions, origin, nil)
	if err != nil {
		return nil, err
	}
	limiter := newAttemptLimiter(60)
	protected := authn.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, ok := IdentityFromContext(r.Context())
		if !ok {
			catalogFailure(w, 401, "UNAUTHENTICATED")
			return
		}
		business, ok := catalogUUID(r.PathValue("businessID"))
		if !ok {
			catalogFailure(w, 400, "INVALID_BUSINESS_ID")
			return
		}
		if !limiter.allow(r.RemoteAddr) {
			w.Header().Set("Retry-After", "900")
			catalogFailure(w, 429, "RATE_LIMITED")
			return
		}
		if r.URL.RawQuery != "" || len(r.Header.Values("Idempotency-Key")) != 1 {
			catalogFailure(w, 400, "VALIDATION_ERROR")
			return
		}
		in, ok := decodeCatalogJSON[service.BookingInput](w, r)
		if !ok {
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		result, e := bookings.Create(ctx, actor.UserID, business, r.Header.Get("Idempotency-Key"), in)
		switch {
		case errors.Is(e, service.ErrForbidden):
			catalogFailure(w, 403, "FORBIDDEN")
		case errors.Is(e, service.ErrInvalidBooking):
			catalogFailure(w, 400, "VALIDATION_ERROR")
		case errors.Is(e, service.ErrBookingConflict):
			catalogFailure(w, 409, "BOOKING_CONFLICT")
		case errors.Is(e, service.ErrIdempotencyConflict):
			catalogFailure(w, 409, "IDEMPOTENCY_CONFLICT")
		case e != nil:
			catalogFailure(w, 503, "BOOKING_UNAVAILABLE")
		default:
			status := 201
			if result.Replayed {
				status = 200
			}
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(result)
		}
	}))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			if method := r.Header.Get("Access-Control-Request-Method"); method != "" && method != "POST" {
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
	}), nil
}
