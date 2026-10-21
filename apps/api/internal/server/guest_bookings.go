package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"multipurpose-booking-system/api/internal/db"
	"multipurpose-booking-system/api/internal/service"
	"net/http"
	"strings"
	"time"
)

type guestValidator struct{ businesses *service.BusinessService }

func (g guestValidator) Validate(ctx context.Context, token string) (db.User, error) {
	return g.businesses.GuestVisitor(ctx, token, false)
}

// A separate router reuses customer workflows without granting guest capabilities
// to auth/me, business membership, or any owner endpoint. Browser cookies are ignored.
func RegisterGuestRoutes(mux *http.ServeMux, businesses *service.BusinessService, origin string) error {
	inner := http.NewServeMux()
	if e := RegisterCustomerFoundationRoutes(inner, guestValidator{businesses}, businesses, origin); e != nil {
		return e
	}
	secure, e := sessionCookieSecurity(origin)
	if e != nil {
		return e
	}
	limiter := newAttemptLimiter(120)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Add("Vary", "Origin")
		if o := r.Header.Get("Origin"); o != "" && o != origin || r.Method != "GET" && o != origin {
			catalogFailure(w, 403, "ORIGIN_REJECTED")
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Expose-Headers", "X-Next-Cursor")
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Idempotency-Key, X-Booking-Access")
			w.WriteHeader(204)
			return
		}
		if !limiter.allow(r.RemoteAddr) {
			w.Header().Set("Retry-After", "900")
			catalogFailure(w, 429, "RATE_LIMITED")
			return
		}
		token := r.Header.Get("X-Booking-Access")
		root := r.URL.Path == "/api/v1/guest/bookings"
		if root && r.Method == "GET" && token == "" {
			_, _ = w.Write([]byte("[]"))
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		if root && r.Method == "POST" {
			in, ok := decodeCatalogJSON[service.CustomerBookingInput](w, r)
			if !ok {
				return
			}
			if in.Contact == nil {
				catalogFailure(w, 400, "VALIDATION_ERROR")
				return
			}
			data, _ := json.Marshal(in)
			r.Body = io.NopCloser(bytes.NewReader(data))
			r.ContentLength = int64(len(data))
		}
		user, e := businesses.GuestVisitor(ctx, token, root && r.Method == "POST")
		if e != nil {
			if errors.Is(e, service.ErrInvalidSession) {
				catalogFailure(w, 401, "GUEST_ACCESS_REQUIRED")
			} else {
				catalogFailure(w, 503, "SERVICE_UNAVAILABLE")
			}
			return
		}
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/guest/bookings/"), "/")
		if !root && (r.Method == "DELETE" || (len(parts) == 2 && parts[1] == "contact")) {
			id, ok := catalogUUID(parts[0])
			if !ok || r.URL.RawQuery != "" {
				catalogFailure(w, 400, "VALIDATION_ERROR")
				return
			}
			var result any
			if r.Method == "DELETE" && len(parts) == 1 {
				e = businesses.HideGuestBooking(ctx, user.ID, id)
			} else if len(parts) == 2 && r.Method == "GET" {
				result, e = businesses.GuestContact(ctx, user.ID, id, nil)
			} else if len(parts) == 2 && r.Method == "PATCH" {
				in, ok := decodeCatalogJSON[service.GuestContact](w, r)
				if !ok {
					return
				}
				result, e = businesses.GuestContact(ctx, user.ID, id, &in)
			} else {
				catalogFailure(w, 405, "METHOD_NOT_ALLOWED")
				return
			}
			switch {
			case errors.Is(e, service.ErrBookingNotFound):
				catalogFailure(w, 404, "NOT_FOUND")
			case errors.Is(e, service.ErrBookingState):
				catalogFailure(w, 409, "BOOKING_STATE_CONFLICT")
			case errors.Is(e, service.ErrInvalidBooking):
				catalogFailure(w, 400, "VALIDATION_ERROR")
			case e != nil:
				catalogFailure(w, 503, "SERVICE_UNAVAILABLE")
			default:
				if r.Method == "DELETE" {
					w.WriteHeader(204)
				} else {
					_ = json.NewEncoder(w).Encode(result)
				}
			}
			return
		}
		clone := r.Clone(ctx)
		u := *r.URL
		u.Path = strings.Replace(r.URL.Path, "/api/v1/guest/bookings", "/api/v1/customer/bookings", 1)
		clone.URL = &u
		clone.Header = r.Header.Clone()
		clone.Header.Del("Cookie")
		clone.AddCookie(&http.Cookie{Name: sessionCookieName(secure), Value: token})
		inner.ServeHTTP(w, clone)
	})
	mux.Handle("/api/v1/guest/bookings", handler)
	mux.Handle("/api/v1/guest/bookings/", handler)
	return nil
}
