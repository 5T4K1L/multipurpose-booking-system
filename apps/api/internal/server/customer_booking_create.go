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

func registerCustomerBookingCreate(mux *http.ServeMux, sessions sessionValidator, businesses *service.BusinessService, origin string) error {
	authn, err := NewAuthenticationMiddleware(sessions, origin, nil)
	if err != nil {
		return err
	}
	limiter := newAttemptLimiter(60)
	protected := authn.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, ok := IdentityFromContext(r.Context())
		if !ok {
			catalogFailure(w, 401, "UNAUTHENTICATED")
			return
		}

		if r.Method == "GET" {
			values, e := url.ParseQuery(r.URL.RawQuery)
			if e != nil || len(r.URL.RawQuery) > 4096 {
				catalogFailure(w, 400, "INVALID_PAGINATION")
				return
			}
			for key, v := range values {
				if len(v) != 1 || (key != "limit" && key != "cursor") {
					catalogFailure(w, 400, "INVALID_PAGINATION")
					return
				}
			}
			in, e := businessPageRequest(r)
			if e != nil {
				catalogFailure(w, 400, "INVALID_PAGINATION")
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
			defer cancel()
			result, e := businesses.CustomerBookings(ctx, actor.UserID, in)
			switch {
			case errors.Is(e, service.ErrInvalidPage):
				catalogFailure(w, 400, "INVALID_PAGINATION")
			case errors.Is(e, service.ErrForbidden):
				catalogFailure(w, 403, "FORBIDDEN")
			case e != nil:
				catalogFailure(w, 503, "SERVICE_UNAVAILABLE")
			default:
				if result.NextCursor != "" {
					w.Header().Set("X-Next-Cursor", result.NextCursor)
				}
				_ = json.NewEncoder(w).Encode(result.Items)
			}
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
		in, ok := decodeCatalogJSON[service.CustomerBookingInput](w, r)
		if !ok {
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		result, e := businesses.CreateCustomerBooking(ctx, actor.UserID, r.Header.Get("Idempotency-Key"), in)
		switch {
		case errors.Is(e, service.ErrForbidden):
			catalogFailure(w, 403, "FORBIDDEN")
		case errors.Is(e, service.ErrInvalidBooking):
			catalogFailure(w, 400, "VALIDATION_ERROR")
		case errors.Is(e, service.ErrPublicProfileNotFound):
			catalogFailure(w, 404, "NOT_FOUND")
		case errors.Is(e, service.ErrPublicationChanged):
			catalogFailure(w, 409, "PUBLIC_AVAILABILITY_CHANGED")
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
	mux.Handle("/api/v1/customer/bookings", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Add("Vary", "Origin")
		if o := r.Header.Get("Origin"); o != "" && o != origin || r.Method != "GET" && o != origin {
			catalogFailure(w, 403, "ORIGIN_REJECTED")
			return
		}
		if r.Header.Get("Origin") == origin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Expose-Headers", "X-Next-Cursor")
		}
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
