package server

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/service"
	"net/http"
	"time"
)

func RegisterBookingRoutes(mux *http.ServeMux, h http.Handler) {
	base := "/api/v1/businesses/{businessID}/bookings"
	mux.Handle(base, h)
	mux.Handle(base+"/{bookingID}", h)
	mux.Handle(base+"/{bookingID}/{action}", h)
}

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
		if r.Method == "GET" {
			serveBookingRead(w, r, bookings, actor.UserID, business)
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
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		var result any
		var e error
		status := 200
		if action := r.PathValue("action"); action != "" {
			id, valid := catalogUUID(r.PathValue("bookingID"))
			if !valid {
				catalogFailure(w, 400, "INVALID_BOOKING_ID")
				return
			}
			result, e = serveBookingChange(w, r.WithContext(ctx), bookings, actor.UserID, business, id, action)
			if result == nil && e == nil {
				return
			}
		} else {
			in, ok := decodeCatalogJSON[service.BookingInput](w, r)
			if !ok {
				return
			}
			var created service.BookingResult
			created, e = bookings.Create(ctx, actor.UserID, business, r.Header.Get("Idempotency-Key"), in)
			result = created
			if !created.Replayed {
				status = 201
			}
		}
		switch {
		case errors.Is(e, service.ErrForbidden):
			catalogFailure(w, 403, "FORBIDDEN")
		case errors.Is(e, service.ErrBookingNotFound):
			catalogFailure(w, 404, "BOOKING_NOT_FOUND")
		case errors.Is(e, service.ErrBookingState):
			catalogFailure(w, 409, "BOOKING_STATE_CONFLICT")
		case errors.Is(e, service.ErrInvalidBooking):
			catalogFailure(w, 400, "VALIDATION_ERROR")
		case errors.Is(e, service.ErrBookingConflict):
			catalogFailure(w, 409, "BOOKING_CONFLICT")
		case errors.Is(e, service.ErrIdempotencyConflict):
			catalogFailure(w, 409, "IDEMPOTENCY_CONFLICT")
		case e != nil:
			catalogFailure(w, 503, "BOOKING_UNAVAILABLE")
		default:
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(result)
		}
	}))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Add("Vary", "Origin")
		if r.Header.Get("Origin") != "" && r.Header.Get("Origin") != origin || r.Method != "GET" && r.Header.Get("Origin") != origin {
			catalogFailure(w, 403, "ORIGIN_REJECTED")
			return
		}
		if r.Header.Get("Origin") == origin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Expose-Headers", "X-Next-Cursor")
		}
		if action := r.PathValue("action"); action != "" && action != "cancel" && action != "reschedule" {
			catalogFailure(w, 404, "NOT_FOUND")
			return
		}
		methods := "GET, POST, OPTIONS"
		if r.PathValue("bookingID") != "" {
			methods = "GET, OPTIONS"
		}
		if r.PathValue("action") != "" {
			methods = "POST, OPTIONS"
		}
		allowed := func(method string) bool {
			return method == "OPTIONS" || method == "GET" && r.PathValue("action") == "" || method == "POST" && (r.PathValue("bookingID") == "" || r.PathValue("action") != "")
		}
		w.Header().Set("Allow", methods)
		if r.Method == "OPTIONS" {
			if method := r.Header.Get("Access-Control-Request-Method"); method != "" && !allowed(method) {
				catalogFailure(w, 405, "METHOD_NOT_ALLOWED")
				return
			}
			w.Header().Set("Access-Control-Allow-Methods", methods)
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Idempotency-Key")
			w.WriteHeader(204)
			return
		}
		if !allowed(r.Method) {
			catalogFailure(w, 405, "METHOD_NOT_ALLOWED")
			return
		}
		protected.ServeHTTP(w, r)
	}), nil
}

func serveBookingChange(w http.ResponseWriter, r *http.Request, s *service.BookingService, actor int64, business, id pgtype.UUID, action string) (any, error) {
	if action == "cancel" {
		if _, ok := decodeCatalogJSON[struct{}](w, r); !ok {
			return nil, nil
		}
		return s.Cancel(r.Context(), actor, business, id, r.Header.Get("Idempotency-Key"))
	}
	in, ok := decodeCatalogJSON[struct {
		StartsAt time.Time `json:"starts_at"`
	}](w, r)
	if !ok {
		return nil, nil
	}
	return s.Reschedule(r.Context(), actor, business, id, r.Header.Get("Idempotency-Key"), in.StartsAt)
}

func serveBookingRead(w http.ResponseWriter, r *http.Request, s *service.BookingService, actor int64, business pgtype.UUID) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	var result any
	var e error
	cursor := ""
	if raw := r.PathValue("bookingID"); raw != "" {
		id, ok := catalogUUID(raw)
		if !ok || r.URL.RawQuery != "" {
			catalogFailure(w, 400, "VALIDATION_ERROR")
			return
		}
		result, e = s.Get(ctx, actor, business, id)
	} else {
		for key := range r.URL.Query() {
			if key != "limit" && key != "cursor" {
				catalogFailure(w, 400, "INVALID_PAGINATION")
				return
			}
		}
		input, err := businessPageRequest(r)
		if err != nil {
			catalogFailure(w, 400, "INVALID_PAGINATION")
			return
		}
		page, err := s.List(ctx, actor, business, input)
		result, e, cursor = page.Items, err, page.NextCursor
	}
	switch {
	case errors.Is(e, service.ErrForbidden):
		catalogFailure(w, 403, "FORBIDDEN")
	case errors.Is(e, service.ErrBookingNotFound):
		catalogFailure(w, 404, "BOOKING_NOT_FOUND")
	case errors.Is(e, service.ErrInvalidPage):
		catalogFailure(w, 400, "INVALID_PAGINATION")
	case e != nil:
		catalogFailure(w, 503, "BOOKING_UNAVAILABLE")
	default:
		if cursor != "" {
			w.Header().Set("X-Next-Cursor", cursor)
		}
		_ = json.NewEncoder(w).Encode(result)
	}
}
