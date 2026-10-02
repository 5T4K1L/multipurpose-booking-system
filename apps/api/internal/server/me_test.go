package server

import (
	"encoding/json"
	"multipurpose-booking-system/api/internal/db"
	"multipurpose-booking-system/api/internal/service"
	"net/http/httptest"
	"testing"
)

func TestMeContract(t *testing.T) {
	for _, tc := range []struct {
		name, method, origin, cookie string
		err                          error
		status, calls                int
	}{
		{"authenticated", "GET", "http://localhost:3000", "booking_session=secret", nil, 200, 1},
		{"no origin", "GET", "", "booking_session=secret", nil, 200, 1},
		{"anonymous", "GET", "http://localhost:3000", "", nil, 401, 0},
		{"invalid", "GET", "http://localhost:3000", "booking_session=secret", service.ErrInvalidSession, 401, 1},
		{"unavailable", "GET", "http://localhost:3000", "booking_session=secret", service.ErrSessionUnavailable, 503, 1},
		{"foreign", "GET", "https://evil.example", "booking_session=secret", nil, 403, 0},
		{"preflight", "OPTIONS", "http://localhost:3000", "", nil, 204, 0},
		{"post", "POST", "http://localhost:3000", "booking_session=secret", nil, 405, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stub := &validatorStub{user: db.User{ID: 42, Email: "member@example.com", Username: "Member", Status: "active"}, err: tc.err}
			h, err := NewMeHandler(stub, "http://localhost:3000", nil)
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(tc.method, "/api/v1/auth/me?user_id=999", nil)
			req.Header.Set("Origin", tc.origin)
			req.Header.Set("Cookie", tc.cookie)
			req.Header.Set("X-User-ID", "999")
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tc.status || stub.calls != tc.calls {
				t.Fatalf("status=%d calls=%d", rec.Code, stub.calls)
			}
			if rec.Header().Get("Cache-Control") != "no-store" || len(rec.Result().Cookies()) != 0 {
				t.Fatal("cache/cookie contract")
			}
			if tc.origin == "http://localhost:3000" && rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
				t.Fatal("missing CORS on response")
			}
			if tc.status == 200 {
				var body map[string]map[string]any
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				user := body["user"]
				if len(body) != 1 || len(user) != 3 || user["id"] != float64(42) || user["email"] != "member@example.com" || user["username"] != "Member" {
					t.Fatalf("unsafe or wrong payload: %s", rec.Body.String())
				}
			}
		})
	}
}
