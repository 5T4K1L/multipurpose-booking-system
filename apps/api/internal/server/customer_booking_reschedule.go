package server

import (
	"context"
	"encoding/json"
	"errors"
	"multipurpose-booking-system/api/internal/service"
	"net/http"
	"net/url"
	"time"
)

func registerCustomerBookingReschedule(mux *http.ServeMux, sessions sessionValidator, businesses *service.BusinessService, origin string) error {
	authn, e := NewAuthenticationMiddleware(sessions, origin, nil)
	if e != nil {
		return e
	}
	limiter := newAttemptLimiter(60)
	readLimiter := newAttemptLimiter(120)
	protected := authn.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, ok := IdentityFromContext(r.Context())
		if !ok {
			catalogFailure(w, 401, "UNAUTHENTICATED")
			return
		}
		attempts := &limiter
		if r.Method == "GET" {
			attempts = &readLimiter
		}
		if !attempts.allow(r.RemoteAddr) {
			w.Header().Set("Retry-After", "900")
			catalogFailure(w, 429, "RATE_LIMITED")
			return
		}
		id, ok := catalogUUID(r.PathValue("id"))
		if !ok || (r.Method == "POST" && (r.URL.RawQuery != "" || len(r.Header.Values("Idempotency-Key")) != 1)) {
			catalogFailure(w, 400, "VALIDATION_ERROR")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		var result any
		var e error
		if r.Method == "GET" {
			query, err := url.ParseQuery(r.URL.RawQuery)
			if err != nil || len(query) != 1 || len(query["date"]) != 1 {
				catalogFailure(w, 400, "VALIDATION_ERROR")
				return
			}
			result, e = businesses.CustomerRescheduleSlots(ctx, actor.UserID, id, query.Get("date"))
		} else {
			body, ok := decodeCatalogJSON[struct {
				StartsAt time.Time `json:"starts_at"`
			}](w, r)
			if !ok {
				return
			}
			result, e = businesses.RescheduleCustomerBooking(ctx, actor.UserID, id, r.Header.Get("Idempotency-Key"), body.StartsAt)
		}
		switch {
		case errors.Is(e, service.ErrInvalidBooking):
			catalogFailure(w, 400, "VALIDATION_ERROR")
		case errors.Is(e, service.ErrForbidden):
			catalogFailure(w, 403, "FORBIDDEN")
		case errors.Is(e, service.ErrBookingNotFound):
			catalogFailure(w, 404, "NOT_FOUND")
		case errors.Is(e, service.ErrBookingConflict):
			catalogFailure(w, 409, "BOOKING_CONFLICT")
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
	mux.Handle("/api/v1/customer/bookings/{id}/reschedule", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Add("Vary", "Origin")
		if r.Header.Get("Origin") != origin && !(r.Method == "GET" && r.Header.Get("Origin") == "") {
			catalogFailure(w, 403, "ORIGIN_REJECTED")
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Allow", "GET, POST, OPTIONS")
		if r.Method == "OPTIONS" {
			if m := r.Header.Get("Access-Control-Request-Method"); m != "" && m != "POST" && m != "GET" {
				catalogFailure(w, 405, "METHOD_NOT_ALLOWED")
				return
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Idempotency-Key")
			w.WriteHeader(204)
			return
		}
		if r.Method != "POST" && r.Method != "GET" {
			catalogFailure(w, 405, "METHOD_NOT_ALLOWED")
			return
		}
		protected.ServeHTTP(w, r)
	}))
	return nil
}
