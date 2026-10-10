package server

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5/pgtype"
	"io"
	"mime"
	"multipurpose-booking-system/api/internal/service"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// NewBusinessHandler handles only the business foundation API. Authentication
// supplies the actor; the service resolves fresh tenant membership per operation.
func NewBusinessHandler(sessions sessionValidator, businesses *service.BusinessService, origin string) (http.Handler, error) {
	authn, err := NewAuthenticationMiddleware(sessions, origin, nil)
	if err != nil {
		return nil, err
	}
	limiter := newAttemptLimiter(60)
	fail := func(w http.ResponseWriter, status int, code string) {
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "request_id": w.Header().Get("X-Request-ID")}})
	}
	protected := authn.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, ok := IdentityFromContext(r.Context())
		if !ok {
			fail(w, 401, "UNAUTHENTICATED")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		if r.Method != "GET" && !limiter.allow(r.RemoteAddr) {
			w.Header().Set("Retry-After", "900")
			fail(w, 429, "RATE_LIMITED")
			return
		}
		decode := func(v any) bool {
			media, _, e := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if e != nil || media != "application/json" {
				fail(w, 415, "UNSUPPORTED_MEDIA_TYPE")
				return false
			}
			d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
			d.DisallowUnknownFields()
			if e = d.Decode(v); e == nil {
				var extra any
				if d.Decode(&extra) != io.EOF {
					e = errors.New("trailing JSON")
				}
			}
			if e != nil {
				fail(w, 400, "INVALID_JSON")
				return false
			}
			return true
		}
		var result any
		var e error
		status := 200
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/businesses")
		parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
		if path == "" {
			switch r.Method {
			case "GET":
				result, e = businesses.List(ctx, actor.UserID)
			case "POST":
				var in service.BusinessInput
				if !decode(&in) {
					return
				}
				result, e = businesses.Create(ctx, actor.UserID, in)
				status = 201
			default:
				w.Header().Set("Allow", "GET, POST, OPTIONS")
				fail(w, 405, "METHOD_NOT_ALLOWED")
				return
			}
		} else {
			var id pgtype.UUID
			if len(parts[0]) != 36 || id.Scan(parts[0]) != nil {
				fail(w, 400, "INVALID_BUSINESS_ID")
				return
			}
			switch {
			case len(parts) == 1 && r.Method == "GET":
				result, e = businesses.Get(ctx, actor.UserID, id)
			case len(parts) == 2 && parts[1] == "memberships" && r.Method == "GET":
				result, e = businesses.Members(ctx, actor.UserID, id)
			case len(parts) == 3 && parts[1] == "memberships" && (r.Method == "PUT" || r.Method == "DELETE"):
				target, parseErr := strconv.ParseInt(parts[2], 10, 64)
				if parseErr != nil || target <= 0 {
					fail(w, 400, "INVALID_USER_ID")
					return
				}
				var in struct {
					Role service.Role `json:"role"`
				}
				if r.Method == "PUT" && !decode(&in) {
					return
				}
				e = businesses.ChangeMember(ctx, actor.UserID, id, target, in.Role, r.Method == "DELETE")
				status = 204
			default:
				fail(w, 404, "NOT_FOUND")
				return
			}
		}
		if errors.Is(e, service.ErrForbidden) {
			fail(w, 403, "FORBIDDEN")
			return
		}
		if errors.Is(e, service.ErrInvalidBusiness) {
			fail(w, 400, "VALIDATION_ERROR")
			return
		}
		if e != nil {
			fail(w, 503, "BUSINESS_UNAVAILABLE")
			return
		}
		w.WriteHeader(status)
		if status != 204 {
			_ = json.NewEncoder(w).Encode(result)
		}
	}))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Add("Vary", "Origin")
		requestOrigin := r.Header.Get("Origin")
		if requestOrigin != "" && requestOrigin != origin || r.Method != "GET" && requestOrigin != origin {
			fail(w, 403, "ORIGIN_REJECTED")
			return
		}
		if requestOrigin == origin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(204)
			return
		}
		protected.ServeHTTP(w, r)
	}), nil
}
