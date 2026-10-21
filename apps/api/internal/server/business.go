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
	"net/url"
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
				input, parseErr := businessPageRequest(r)
				if parseErr != nil {
					fail(w, 400, "INVALID_PAGINATION")
					return
				}
				page, pageErr := businesses.ListPage(ctx, actor.UserID, input)
				result, e = page.Items, pageErr
				if e == nil && page.NextCursor != "" {
					w.Header().Set("X-Next-Cursor", page.NextCursor)
				}
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
			case len(parts) == 2 && parts[1] == "customers":
				if r.Method != "GET" {
					w.Header().Set("Allow", "GET, OPTIONS")
					fail(w, 405, "METHOD_NOT_ALLOWED")
					return
				}
				values, err := url.ParseQuery(r.URL.RawQuery)
				if err != nil {
					fail(w, 400, "VALIDATION_ERROR")
					return
				}
				for key, v := range values {
					if (key != "limit" && key != "cursor" && key != "q") || len(v) != 1 {
						fail(w, 400, "VALIDATION_ERROR")
						return
					}
				}
				// Existing pagination parser accepts only pagination keys.
				copyRequest := r.Clone(ctx)
				copyURL := *r.URL
				copyRequest.URL = &copyURL
				search := values.Get("q")
				values.Del("q")
				copyRequest.URL.RawQuery = values.Encode()
				input, err := businessPageRequest(copyRequest)
				if err != nil {
					fail(w, 400, "INVALID_PAGINATION")
					return
				}
				page, pageErr := businesses.Customers(ctx, actor.UserID, id, input, search)
				result, e = page.Items, pageErr
				if e == nil && page.NextCursor != "" {
					w.Header().Set("X-Next-Cursor", page.NextCursor)
				}
			case len(parts) == 2 && parts[1] == "publication":
				if r.URL.RawQuery != "" {
					fail(w, 400, "VALIDATION_ERROR")
					return
				}
				switch r.Method {
				case "GET":
					result, e = businesses.Publication(ctx, actor.UserID, id, nil, "")
				case "PUT":
					in, ok := decodeCatalogJSON[struct {
						Published   *bool  `json:"published"`
						ReviewToken string `json:"review_token"`
					}](w, r)
					if !ok {
						return
					}
					if in.Published == nil {
						fail(w, 400, "VALIDATION_ERROR")
						return
					}
					result, e = businesses.Publication(ctx, actor.UserID, id, in.Published, in.ReviewToken)
				default:
					w.Header().Set("Allow", "GET, PUT, OPTIONS")
					fail(w, 405, "METHOD_NOT_ALLOWED")
					return
				}
			case len(parts) == 2 && parts[1] == "settings":
				if r.URL.RawQuery != "" {
					fail(w, 400, "VALIDATION_ERROR")
					return
				}
				switch r.Method {
				case "GET":
					result, e = businesses.Settings(ctx, actor.UserID, id)
				case "PUT":
					in, ok := decodeCatalogJSON[struct {
						Tagline      *string `json:"tagline"`
						Description  *string `json:"description"`
						ContactEmail *string `json:"contact_email"`
						ContactPhone *string `json:"contact_phone"`
						Website      *string `json:"website"`
					}](w, r)
					if !ok {
						return
					}
					if in.Tagline == nil || in.Description == nil || in.ContactEmail == nil || in.ContactPhone == nil || in.Website == nil {
						fail(w, 400, "VALIDATION_ERROR")
						return
					}
					result, e = businesses.SaveSettings(ctx, actor.UserID, id, service.BusinessSettings{Tagline: *in.Tagline, Description: *in.Description, ContactEmail: *in.ContactEmail, ContactPhone: *in.ContactPhone, Website: *in.Website})
				default:
					w.Header().Set("Allow", "GET, PUT, OPTIONS")
					fail(w, 405, "METHOD_NOT_ALLOWED")
					return
				}
			case len(parts) == 1 && r.Method == "PUT":
				in, ok := decodeCatalogJSON[service.BusinessInput](w, r)
				if !ok {
					return
				}
				result, e = businesses.UpdateProfile(ctx, actor.UserID, id, in)
			case len(parts) == 1 && r.Method == "GET":
				result, e = businesses.Get(ctx, actor.UserID, id)
			case len(parts) == 2 && parts[1] == "memberships" && r.Method == "GET":
				input, parseErr := businessPageRequest(r)
				if parseErr != nil {
					fail(w, 400, "INVALID_PAGINATION")
					return
				}
				page, pageErr := businesses.MembersPage(ctx, actor.UserID, id, input)
				result, e = page.Items, pageErr
				if e == nil && page.NextCursor != "" {
					w.Header().Set("X-Next-Cursor", page.NextCursor)
				}
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
		if errors.Is(e, service.ErrInvalidPage) {
			fail(w, 400, "INVALID_PAGINATION")
			return
		}
		if errors.Is(e, service.ErrPublicationTooLarge) {
			fail(w, 400, "PUBLICATION_TOO_LARGE")
			return
		}
		if errors.Is(e, service.ErrPublicationChanged) {
			fail(w, 409, "PUBLICATION_CHANGED")
			return
		}
		if errors.Is(e, service.ErrBusinessAlreadyOwned) {
			fail(w, 409, "BUSINESS_ALREADY_OWNED")
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
			w.Header().Set("Access-Control-Expose-Headers", "X-Next-Cursor")
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

func businessPageRequest(r *http.Request) (service.PageRequest, error) {
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return service.PageRequest{}, service.ErrInvalidPage
	}
	out := service.PageRequest{}
	if v, ok := values["limit"]; ok {
		if len(v) != 1 {
			return out, service.ErrInvalidPage
		}
		out.Limit, err = strconv.Atoi(v[0])
		if err != nil || out.Limit < 1 || out.Limit > 100 {
			return out, service.ErrInvalidPage
		}
	}
	if v, ok := values["cursor"]; ok {
		if len(v) != 1 || v[0] == "" || len(v[0]) > 1024 {
			return out, service.ErrInvalidPage
		}
		out.Cursor = v[0]
	}
	return out, nil
}
