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

// Customer reads require sessions; public profile reads expose confirmed snapshots only.
func RegisterCustomerFoundationRoutes(mux *http.ServeMux, sessions sessionValidator, businesses *service.BusinessService, origin string) error {
	authn, err := NewAuthenticationMiddleware(sessions, origin, nil)
	if err != nil {
		return err
	}
	protected := authn.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, ok := IdentityFromContext(r.Context())
		if !ok {
			catalogFailure(w, 401, "UNAUTHENTICATED")
			return
		}
		id, ok := catalogUUID(r.PathValue("id"))
		if !ok || r.URL.RawQuery != "" {
			catalogFailure(w, 400, "VALIDATION_ERROR")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		result, e := businesses.CustomerBooking(ctx, actor.UserID, id)
		switch {
		case errors.Is(e, service.ErrBookingNotFound):
			catalogFailure(w, 404, "NOT_FOUND")
		case errors.Is(e, service.ErrForbidden):
			catalogFailure(w, 403, "FORBIDDEN")
		case e != nil:
			catalogFailure(w, 503, "SERVICE_UNAVAILABLE")
		default:
			_ = json.NewEncoder(w).Encode(result)
		}
	}))
	mux.Handle("/api/v1/customer/bookings/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Add("Vary", "Origin")
		o := r.Header.Get("Origin")
		if o != "" && o != origin {
			catalogFailure(w, 403, "ORIGIN_REJECTED")
			return
		}
		if o == origin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Allow", "GET, OPTIONS")
		if r.Method == "OPTIONS" {
			if m := r.Header.Get("Access-Control-Request-Method"); m != "" && m != "GET" {
				catalogFailure(w, 405, "METHOD_NOT_ALLOWED")
				return
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.WriteHeader(204)
			return
		}
		if r.Method != "GET" {
			catalogFailure(w, 405, "METHOD_NOT_ALLOWED")
			return
		}
		protected.ServeHTTP(w, r)
	}))
	availabilityLimiter := newAttemptLimiter(60)
	calendarLimiter := newAttemptLimiter(120)
	rangeLimiter := newAttemptLimiter(120)
	publicRead := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Add("Vary", "Origin")
		if o := r.Header.Get("Origin"); o != "" && o != origin {
			catalogFailure(w, 403, "ORIGIN_REJECTED")
			return
		}
		if r.Header.Get("Origin") == origin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Expose-Headers", "X-Next-Cursor")
		}
		w.Header().Set("Allow", "GET, OPTIONS")
		if r.Method == "OPTIONS" {
			if m := r.Header.Get("Access-Control-Request-Method"); m != "" && m != "GET" {
				catalogFailure(w, 405, "METHOD_NOT_ALLOWED")
				return
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.WriteHeader(204)
			return
		}
		if r.Method != "GET" {
			catalogFailure(w, 405, "METHOD_NOT_ALLOWED")
			return
		}
		if rawService := r.PathValue("serviceID"); rawService != "" {
			limiter := &availabilityLimiter
			dateKey := "date"
			if r.Pattern == "/api/v1/public/businesses/{id}/services/{serviceID}/calendar" {
				limiter = &calendarLimiter
				dateKey = "month"
			}
			if r.Pattern == "/api/v1/public/businesses/{id}/services/{serviceID}/range" {
				limiter = &rangeLimiter
			}
			if !limiter.allow(r.RemoteAddr) {
				w.Header().Set("Retry-After", "900")
				catalogFailure(w, 429, "RATE_LIMITED")
				return
			}
			business, ok := catalogUUID(r.PathValue("id"))
			serviceID, serviceOK := catalogUUID(rawService)
			values, e := url.ParseQuery(r.URL.RawQuery)
			isRange := r.Pattern == "/api/v1/public/businesses/{id}/services/{serviceID}/range"
			validQuery := len(values) == 2 && len(values[dateKey]) == 1 && len(values["location_id"]) == 1
			if isRange {
				validQuery = len(values) == 3 && len(values["starts_at"]) == 1 && len(values["ends_at"]) == 1 && len(values["location_id"]) == 1
			}
			if !ok || !serviceOK || e != nil || !validQuery {
				catalogFailure(w, 400, "VALIDATION_ERROR")
				return
			}
			location, ok := catalogUUID(values.Get("location_id"))
			if !ok {
				catalogFailure(w, 400, "VALIDATION_ERROR")
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
			defer cancel()
			var result any
			if isRange {
				start, e1 := time.Parse(time.RFC3339, values.Get("starts_at"))
				end, e2 := time.Parse(time.RFC3339, values.Get("ends_at"))
				if e1 != nil || e2 != nil {
					catalogFailure(w, 400, "VALIDATION_ERROR")
					return
				}
				result, e = businesses.PublicBookingRange(ctx, business, serviceID, location, start, end)
			} else if dateKey == "month" {
				result, e = businesses.PublicAvailabilityCalendar(ctx, business, serviceID, location, values.Get(dateKey))
			} else {
				result, e = businesses.PublicAvailability(ctx, business, serviceID, location, values.Get(dateKey))
			}
			switch {
			case errors.Is(e, service.ErrInvalidCatalog):
				catalogFailure(w, 400, "VALIDATION_ERROR")
			case errors.Is(e, service.ErrPublicProfileNotFound):
				catalogFailure(w, 404, "NOT_FOUND")
			case errors.Is(e, service.ErrPublicationChanged):
				catalogFailure(w, 409, "PUBLIC_AVAILABILITY_CHANGED")
			case e != nil:
				catalogFailure(w, 503, "AVAILABILITY_UNAVAILABLE")
			default:
				_ = json.NewEncoder(w).Encode(result)
			}
			return
		}
		if r.PathValue("id") == "" {
			values, e := url.ParseQuery(r.URL.RawQuery)
			if e != nil || len(r.URL.RawQuery) > 4096 {
				catalogFailure(w, 400, "INVALID_PAGINATION")
				return
			}
			for key, v := range values {
				if len(v) != 1 || (key != "q" && key != "limit" && key != "cursor") {
					catalogFailure(w, 400, "INVALID_PAGINATION")
					return
				}
			}
			input, e := businessPageRequest(r)
			if e != nil {
				catalogFailure(w, 400, "INVALID_PAGINATION")
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
			defer cancel()
			page, e := businesses.Discover(ctx, values.Get("q"), input)
			if errors.Is(e, service.ErrInvalidPage) {
				catalogFailure(w, 400, "INVALID_PAGINATION")
				return
			}
			if e != nil {
				catalogFailure(w, 503, "SERVICE_UNAVAILABLE")
				return
			}
			if page.NextCursor != "" {
				w.Header().Set("X-Next-Cursor", page.NextCursor)
			}
			_ = json.NewEncoder(w).Encode(page.Items)
			return
		}
		id, ok := catalogUUID(r.PathValue("id"))
		if !ok || r.URL.RawQuery != "" {
			catalogFailure(w, 400, "VALIDATION_ERROR")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		profile, e := businesses.PublicProfile(ctx, id)
		switch {
		case errors.Is(e, service.ErrPublicProfileNotFound):
			catalogFailure(w, 404, "NOT_FOUND")
		case e != nil:
			catalogFailure(w, 503, "SERVICE_UNAVAILABLE")
		default:
			_ = json.NewEncoder(w).Encode(profile)
		}
	})
	mux.Handle("/api/v1/public/businesses", publicRead)
	mux.Handle("/api/v1/public/businesses/{id}", publicRead)
	mux.Handle("/api/v1/public/businesses/{id}/services/{serviceID}/availability", publicRead)
	mux.Handle("/api/v1/public/businesses/{id}/services/{serviceID}/calendar", publicRead)
	mux.Handle("/api/v1/public/businesses/{id}/services/{serviceID}/range", publicRead)
	if e := registerCustomerBookingCancel(mux, sessions, businesses, origin); e != nil {
		return e
	}
	if e := registerCustomerBookingReschedule(mux, sessions, businesses, origin); e != nil {
		return e
	}
	return registerCustomerBookingCreate(mux, sessions, businesses, origin)
}
