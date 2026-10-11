package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"multipurpose-booking-system/api/internal/db"
	"multipurpose-booking-system/api/internal/service"
)

// Failure occurs at the transaction boundary, so HTTP must never publish a
// partial result, internal error text or continuation cursor.
type catalogFailedTransactions struct{ calls int }

func (tx *catalogFailedTransactions) WithinBusiness(ctx context.Context, _ func(service.BusinessStore) error) error {
	tx.calls++
	if _, ok := ctx.Deadline(); !ok {
		panic("catalog operation has no timeout")
	}
	return errors.New("private database details")
}

func TestCatalogHTTPBoundary(t *testing.T) {
	const origin = "http://localhost:3000"
	const base = "/api/v1/businesses/00000000-0000-0000-0000-000000000001/catalog/"
	for _, tc := range []struct {
		name, method, path, origin, preflight, cookie string
		status, calls                                 int
	}{
		{name: "preflight", method: "OPTIONS", path: "locations", origin: origin, preflight: "POST", status: 204},
		{name: "link preflight", method: "OPTIONS", path: "services/00000000-0000-0000-0000-000000000002/locations/00000000-0000-0000-0000-000000000003", origin: origin, preflight: "PUT", status: 204},
		{name: "unsupported preflight", method: "OPTIONS", path: "locations", origin: origin, preflight: "PATCH", status: 405},
		{name: "missing preflight origin", method: "OPTIONS", path: "locations", status: 403},
		{name: "foreign preflight origin", method: "OPTIONS", path: "locations", origin: "https://foreign.invalid", status: 403},
		{name: "unknown kind", method: "GET", path: "users", status: 404},
		{name: "unknown relation", method: "GET", path: "locations/00000000-0000-0000-0000-000000000002/staff", status: 404},
		{name: "unsupported method", method: "PATCH", path: "locations", origin: origin, status: 405},
		{name: "missing cookie", method: "GET", path: "locations", status: 401},
		{name: "duplicate cookie", method: "GET", path: "locations", cookie: "booking_session=one; booking_session=two", status: 401},
		{name: "bad UUID", method: "GET", path: "services/bad", cookie: "booking_session=token", status: 400, calls: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			validator := &validatorStub{user: db.User{ID: 1, Status: "active"}}
			h, err := NewCatalogHandler(validator, nil, origin)
			if err != nil {
				t.Fatal(err)
			}
			mux := http.NewServeMux()
			RegisterCatalogRoutes(mux, h)
			r := httptest.NewRequest(tc.method, base+tc.path, nil)
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}
			if tc.cookie != "" {
				r.Header.Set("Cookie", tc.cookie)
			}
			r.Header.Set("Access-Control-Request-Method", tc.preflight)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			if w.Code != tc.status || validator.calls != tc.calls {
				t.Fatalf("status %d calls %d", w.Code, validator.calls)
			}
			if w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("response can be cached")
			}
			if tc.origin == origin {
				if w.Header().Get("Access-Control-Allow-Origin") != origin || w.Header().Get("Access-Control-Allow-Credentials") != "true" {
					t.Fatal("credential CORS missing")
				}
			}
			if tc.origin == "https://foreign.invalid" && w.Header().Get("Access-Control-Allow-Origin") != "" {
				t.Fatal("foreign origin allowed")
			}
		})
	}
	tx := &catalogFailedTransactions{}
	catalog := service.NewBookingCatalogService(service.NewBusinessService(tx))
	h, err := NewCatalogHandler(&validatorStub{user: db.User{ID: 1, Status: "active"}}, catalog, origin)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	RegisterCatalogRoutes(mux, h)
	for _, path := range []string{"locations", "services", "staff", "resources", "services/00000000-0000-0000-0000-000000000002/locations"} {
		r := httptest.NewRequest("GET", base+path, nil)
		r.Header.Set("Cookie", "booking_session=token")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != 503 || strings.Contains(w.Body.String(), "private") || w.Header().Get("X-Next-Cursor") != "" || w.Header().Get("X-Request-ID") == "" {
			t.Fatal("unsafe storage failure", w.Code, w.Body.String())
		}
	}
	if tx.calls != 5 {
		t.Fatal("requests did not reach service", tx.calls)
	}
}

func TestCatalogMutationRateLimit(t *testing.T) {
	h, err := NewCatalogHandler(&validatorStub{user: db.User{ID: 1, Status: "active"}}, nil, "http://localhost:3000")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	RegisterCatalogRoutes(mux, h)
	check := func(peer string, n, want int) {
		t.Helper()
		r := httptest.NewRequest("POST", "/api/v1/businesses/00000000-0000-0000-0000-000000000001/catalog/locations", strings.NewReader("{"))
		r.RemoteAddr = peer
		r.Header.Set("Cookie", "booking_session=token")
		r.Header.Set("Origin", "http://localhost:3000")
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Forwarded-For", fmt.Sprintf("203.0.113.%d", n))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("attempt %d status %d want %d", n, w.Code, want)
		}
		if want == 429 && w.Header().Get("Retry-After") != "900" {
			t.Fatal("missing retry guidance")
		}
	}
	for i := 1; i <= 60; i++ {
		check("192.0.2.1:1234", i, 400)
	}
	check("192.0.2.1:5678", 61, 429)
	check("192.0.2.2:1234", 62, 400)
}
