package server

import (
	"context"
	"errors"
	"io"
	"multipurpose-booking-system/api/internal/service"
	"net/http"
	"time"
)

func RegisterBusinessImageRoutes(mux *http.ServeMux, sessions sessionValidator, businesses *service.BusinessService, origin string) error {
	auth, e := NewAuthenticationMiddleware(sessions, origin, nil)
	if e != nil {
		return e
	}
	owner := auth.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, _ := IdentityFromContext(r.Context())
		id, ok := catalogUUID(r.PathValue("id"))
		if !ok {
			catalogFailure(w, 400, "VALIDATION_ERROR")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		var content []byte
		if r.Method == "PUT" {
			var e error
			content, e = io.ReadAll(http.MaxBytesReader(w, r.Body, 2*1024*1024))
			if e != nil || len(content) == 0 {
				catalogFailure(w, 400, "IMAGE_TOO_LARGE_OR_INVALID")
				return
			}
		}
		data, kind, e := businesses.BusinessImage(ctx, actor.UserID, id, content, r.Method != "GET")
		if errors.Is(e, service.ErrForbidden) {
			catalogFailure(w, 403, "FORBIDDEN")
			return
		}
		if errors.Is(e, service.ErrInvalidBusiness) {
			catalogFailure(w, 400, "IMAGE_TOO_LARGE_OR_INVALID")
			return
		}
		if e != nil || r.Method == "GET" && len(data) == 0 {
			catalogFailure(w, 404, "NOT_FOUND")
			return
		}
		if r.Method != "GET" {
			w.WriteHeader(204)
			return
		}
		w.Header().Set("Content-Type", kind)
		_, _ = w.Write(data)
	}))
	mux.Handle("/api/v1/businesses/{id}/image", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if o := r.Header.Get("Origin"); o != "" && o != origin || r.Method != "GET" && o != origin {
			catalogFailure(w, 403, "ORIGIN_REJECTED")
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(204)
			return
		}
		if r.Method != "GET" && r.Method != "PUT" && r.Method != "DELETE" {
			w.WriteHeader(405)
			return
		}
		owner.ServeHTTP(w, r)
	}))
	mux.HandleFunc("GET /api/v1/public/businesses/{id}/image", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		id, ok := catalogUUID(r.PathValue("id"))
		if !ok {
			w.WriteHeader(404)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		data, kind, e := businesses.PublicBusinessImage(ctx, id)
		if e != nil {
			w.WriteHeader(404)
			return
		}
		w.Header().Set("Content-Type", kind)
		_, _ = w.Write(data)
	})
	return nil
}
