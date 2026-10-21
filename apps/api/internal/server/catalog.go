package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/db"
	"multipurpose-booking-system/api/internal/service"
)

// Explicit public fields; database rows are never directly serialized.
type catalogItem struct {
	AllDay          bool         `json:"all_day,omitempty"`
	RoleName        string       `json:"role_name,omitempty"`
	ID              pgtype.UUID  `json:"id"`
	Name            string       `json:"name"`
	Timezone        string       `json:"timezone,omitempty"`
	DurationMinutes int32        `json:"duration_minutes,omitempty"`
	LocationID      *pgtype.UUID `json:"location_id,omitempty"`
	Capacity        int32        `json:"capacity,omitempty"`
}

func publicLocation(v db.Location) catalogItem {
	return catalogItem{ID: v.ID, Name: v.Name, Timezone: v.Timezone}
}
func publicService(v db.Service) catalogItem {
	return catalogItem{ID: v.ID, Name: v.Name, DurationMinutes: v.DurationMinutes, AllDay: v.AllDay}
}
func publicStaff(v db.StaffProfile) catalogItem {
	return catalogItem{ID: v.ID, Name: v.Name, RoleName: v.RoleName}
}
func publicResource(v db.Resource) catalogItem {
	return catalogItem{ID: v.ID, Name: v.Name, LocationID: &v.LocationID, Capacity: v.Capacity}
}
func publicCatalogPage[T any](p service.Page[T], convert func(T) catalogItem) []catalogItem {
	out := []catalogItem{}
	for _, v := range p.Items {
		out = append(out, convert(v))
	}
	return out
}

func RegisterCatalogRoutes(mux *http.ServeMux, h http.Handler) {
	base := "/api/v1/businesses/{businessID}/catalog/{kind}"
	for _, suffix := range []string{"", "/{id}", "/{id}/{relation}", "/{id}/{relation}/{targetID}"} {
		mux.Handle(base+suffix, h)
	}
}
func catalogFailure(w http.ResponseWriter, status int, code string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "request_id": w.Header().Get("X-Request-ID")}})
}
func writeCatalogResult(w http.ResponseWriter, result any, status int, cursor string, err error) {
	switch {
	case errors.Is(err, service.ErrForbidden):
		catalogFailure(w, 403, "FORBIDDEN")
		return
	case errors.Is(err, service.ErrCatalogConflict):
		catalogFailure(w, 409, "CATALOG_CONFLICT")
		return
	case errors.Is(err, service.ErrCatalogNotFound):
		catalogFailure(w, 404, "CATALOG_NOT_FOUND")
		return
	case errors.Is(err, service.ErrInvalidCatalog):
		catalogFailure(w, 400, "VALIDATION_ERROR")
		return
	case errors.Is(err, service.ErrInvalidPage):
		catalogFailure(w, 400, "INVALID_PAGINATION")
		return
	case err != nil:
		catalogFailure(w, 503, "CATALOG_UNAVAILABLE")
		return
	}
	if cursor != "" {
		w.Header().Set("X-Next-Cursor", cursor)
	}
	w.WriteHeader(status)
	if status != 204 {
		_ = json.NewEncoder(w).Encode(result)
	}
}
func catalogUUID(value string) (pgtype.UUID, bool) {
	var id pgtype.UUID
	if len(value) != 36 || id.Scan(value) != nil {
		return id, false
	}
	return id, true
}
func decodeCatalogJSON[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var zero T
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		catalogFailure(w, 415, "UNSUPPORTED_MEDIA_TYPE")
		return zero, false
	}
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	d.DisallowUnknownFields()
	var input *T
	err = d.Decode(&input)
	if err == nil {
		var extra any
		err = d.Decode(&extra)
		if err == io.EOF {
			err = nil
		} else if err == nil {
			err = errors.New("trailing JSON")
		}
	}
	if err != nil || input == nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			catalogFailure(w, 413, "BODY_TOO_LARGE")
		} else {
			catalogFailure(w, 400, "INVALID_JSON")
		}
		return zero, false
	}
	return *input, true
}
func catalogRelation(kind, relation string) (service.EligibilityKind, bool) {
	if kind == "services" {
		switch relation {
		case "locations":
			return service.ServiceLocation, true
		case "staff":
			return service.ServiceStaff, true
		case "resources":
			return service.ServiceResource, true
		}
	}
	if kind == "staff" && relation == "locations" {
		return service.StaffLocation, true
	}
	return "", false
}

func NewCatalogHandler(sessions sessionValidator, catalog *service.BookingCatalogService, origin string) (http.Handler, error) {
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
		var id, target pgtype.UUID
		if r.PathValue("id") != "" {
			id, ok = catalogUUID(r.PathValue("id"))
			if !ok {
				catalogFailure(w, 400, "INVALID_CATALOG_ID")
				return
			}
		}
		if r.PathValue("targetID") != "" {
			target, ok = catalogUUID(r.PathValue("targetID"))
			if !ok {
				catalogFailure(w, 400, "INVALID_CATALOG_ID")
				return
			}
		}
		if r.Method != "GET" && !limiter.allow(r.RemoteAddr) {
			w.Header().Set("Retry-After", "900")
			catalogFailure(w, 429, "RATE_LIMITED")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		if catalogAvailabilityRoute(r.PathValue("kind"), r.PathValue("relation")) {
			serveAvailability(w, r.WithContext(ctx), catalog, actor.UserID, business, id)
			return
		}
		if catalogSchedulingRoute(r.PathValue("kind"), r.PathValue("relation")) {
			serveScheduling(w, r.WithContext(ctx), catalog, actor.UserID, business, id)
			return
		}
		if relation := r.PathValue("relation"); relation != "" {
			kind, _ := catalogRelation(r.PathValue("kind"), relation)
			if !target.Valid {
				input, err := businessPageRequest(r)
				if err != nil {
					writeCatalogResult(w, nil, 200, "", err)
					return
				}
				p, err := catalog.ListEligibility(ctx, actor.UserID, business, id, kind, input)
				writeCatalogResult(w, p.Items, 200, p.NextCursor, err)
				return
			}
			// Link endpoints take their complete identity from the URL and accept no body.
			body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1))
			if err != nil || len(body) != 0 {
				catalogFailure(w, 400, "UNEXPECTED_BODY")
				return
			}
			err = catalog.ChangeEligibility(ctx, actor.UserID, business, id, target, kind, r.Method == "DELETE")
			writeCatalogResult(w, nil, 204, "", err)
			return
		}
		serveCatalogEntity(w, r.WithContext(ctx), catalog, actor.UserID, business, id, r.PathValue("kind"))
	}))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Add("Vary", "Origin")
		requestOrigin := r.Header.Get("Origin")
		if requestOrigin != "" && requestOrigin != origin || r.Method != "GET" && requestOrigin != origin {
			catalogFailure(w, 403, "ORIGIN_REJECTED")
			return
		}
		if requestOrigin == origin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Expose-Headers", "X-Next-Cursor")
		}
		kind := r.PathValue("kind")
		if kind != "locations" && kind != "services" && kind != "staff" && kind != "resources" {
			catalogFailure(w, 404, "NOT_FOUND")
			return
		}
		methods := "GET, POST, OPTIONS"
		if r.PathValue("id") != "" {
			methods = "GET, PUT, DELETE, OPTIONS"
		}
		if rel := r.PathValue("relation"); rel != "" {
			methods = "GET, OPTIONS"
			if catalogAvailabilityRoute(kind, rel) {
				if r.PathValue("targetID") != "" {
					catalogFailure(w, 404, "NOT_FOUND")
					return
				}
				methods = "GET, PUT, OPTIONS"
				if rel == "slots" {
					methods = "GET, OPTIONS"
				}
			} else if catalogSchedulingRoute(kind, rel) {
				if r.PathValue("targetID") != "" {
					catalogFailure(w, 404, "NOT_FOUND")
					return
				}
				methods = "GET, PUT, OPTIONS"
				if rel == "booking-readiness" {
					methods = "GET, OPTIONS"
				}
			} else if _, ok := catalogRelation(kind, rel); !ok {
				catalogFailure(w, 404, "NOT_FOUND")
				return
			}
			if r.PathValue("targetID") != "" {
				methods = "PUT, DELETE, OPTIONS"
			}
		}
		allowed := func(method string) bool {
			for _, m := range strings.Split(methods, ", ") {
				if m == method {
					return true
				}
			}
			return false
		}
		w.Header().Set("Allow", methods)
		if r.Method == "OPTIONS" {
			requested := r.Header.Get("Access-Control-Request-Method")
			if requested != "" && !allowed(requested) {
				catalogFailure(w, 405, "METHOD_NOT_ALLOWED")
				return
			}
			w.Header().Set("Access-Control-Allow-Methods", methods)
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
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
func serveCatalogEntity(w http.ResponseWriter, r *http.Request, s *service.BookingCatalogService, actor int64, business, id pgtype.UUID, kind string) {
	ctx := r.Context()
	if r.Method == "DELETE" {
		writeCatalogResult(w, nil, 204, "", s.Archive(ctx, actor, business, id, kind))
		return
	}
	if r.Method == "PUT" {
		serveCatalogUpdate(w, r, s, actor, business, id, kind)
		return
	}
	if r.Method == "POST" {
		switch kind {
		case "locations":
			in, ok := decodeCatalogJSON[struct {
				Name     string `json:"name"`
				Timezone string `json:"timezone"`
			}](w, r)
			if !ok {
				return
			}
			v, err := s.CreateLocation(ctx, actor, business, service.LocationInput{Name: in.Name, Timezone: in.Timezone})
			writeCatalogResult(w, publicLocation(v), 201, "", err)
		case "services":
			in, ok := decodeCatalogJSON[struct {
				Name            string `json:"name"`
				DurationMinutes int32  `json:"duration_minutes"`
				AllDay          bool   `json:"all_day"`
			}](w, r)
			if !ok {
				return
			}
			v, err := s.CreateService(ctx, actor, business, service.ServiceInput{Name: in.Name, DurationMinutes: in.DurationMinutes, AllDay: in.AllDay})
			writeCatalogResult(w, publicService(v), 201, "", err)
		case "staff":
			in, ok := decodeCatalogJSON[struct {
				Name     string `json:"name"`
				RoleName string `json:"role_name"`
			}](w, r)
			if !ok {
				return
			}
			v, err := s.CreateStaffProfile(ctx, actor, business, service.StaffProfileInput{Name: in.Name, RoleName: in.RoleName})
			writeCatalogResult(w, publicStaff(v), 201, "", err)
		case "resources":
			in, ok := decodeCatalogJSON[struct {
				Name       string `json:"name"`
				LocationID string `json:"location_id"`
				Capacity   int32  `json:"capacity"`
			}](w, r)
			if !ok {
				return
			}
			location, ok := catalogUUID(in.LocationID)
			if !ok {
				catalogFailure(w, 400, "INVALID_LOCATION_ID")
				return
			}
			v, err := s.CreateResource(ctx, actor, business, service.ResourceInput{Name: in.Name, LocationID: location, Capacity: in.Capacity})
			writeCatalogResult(w, publicResource(v), 201, "", err)
		}
		return
	}
	if id.Valid {
		switch kind {
		case "locations":
			v, err := s.GetLocation(ctx, actor, business, id)
			writeCatalogResult(w, publicLocation(v), 200, "", err)
		case "services":
			v, err := s.GetService(ctx, actor, business, id)
			writeCatalogResult(w, publicService(v), 200, "", err)
		case "staff":
			v, err := s.GetStaffProfile(ctx, actor, business, id)
			writeCatalogResult(w, publicStaff(v), 200, "", err)
		case "resources":
			v, err := s.GetResource(ctx, actor, business, id)
			writeCatalogResult(w, publicResource(v), 200, "", err)
		}
		return
	}
	input, err := businessPageRequest(r)
	if err != nil {
		writeCatalogResult(w, nil, 200, "", err)
		return
	}
	switch kind {
	case "locations":
		p, err := s.ListLocations(ctx, actor, business, input)
		writeCatalogResult(w, publicCatalogPage(p, publicLocation), 200, p.NextCursor, err)
	case "services":
		p, err := s.ListServices(ctx, actor, business, input)
		writeCatalogResult(w, publicCatalogPage(p, publicService), 200, p.NextCursor, err)
	case "staff":
		p, err := s.ListStaffProfiles(ctx, actor, business, input)
		writeCatalogResult(w, publicCatalogPage(p, publicStaff), 200, p.NextCursor, err)
	case "resources":
		p, err := s.ListResources(ctx, actor, business, input)
		writeCatalogResult(w, publicCatalogPage(p, publicResource), 200, p.NextCursor, err)
	}
}
