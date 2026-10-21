package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"multipurpose-booking-system/api/internal/repository"
	"multipurpose-booking-system/api/internal/service"
)

func TestBusinessSettingsIntegration(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	users := []int64{}
	businesses := []pgtype.UUID{}
	defer func() {
		c, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		for _, id := range businesses {
			for _, table := range []string{"bookings", "service_locations", "service_staff", "service_resources", "staff_locations", "resources", "staff_profiles", "services", "locations"} {
				if _, e := pool.Exec(c, "DELETE FROM "+table+" WHERE business_id=$1", id); e != nil {
					t.Error(e)
				}
			}
			if _, e := pool.Exec(c, "DELETE FROM businesses WHERE id=$1", id); e != nil {
				t.Error(e)
			}
		}
		for _, id := range users {
			for _, table := range []string{"user_sessions", "user_platform_roles"} {
				if _, e := pool.Exec(c, "DELETE FROM "+table+" WHERE user_id=$1", id); e != nil {
					t.Error(e)
				}
			}
			if _, e := pool.Exec(c, "DELETE FROM users WHERE id=$1", id); e != nil {
				t.Error(e)
			}
		}
	}()
	tenants := service.NewBusinessService(repository.NewBusinessTransactions(pool))
	catalog := service.NewBookingCatalogService(tenants)
	sessions, err := service.NewSessionService(repository.NewSessionTransactions(pool), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	tokens := []string{}
	for i := 0; i < 4; i++ {
		var id int64
		if err := pool.QueryRow(ctx, "INSERT INTO users(email,username) VALUES($1,'catalog HTTP fixture') RETURNING id", fmt.Sprintf("catalog-http-%d-%d@example.invalid", time.Now().UnixNano(), i)).Scan(&id); err != nil {
			t.Fatal(err)
		}
		users = append(users, id)
		session, e := sessions.Create(ctx, id)
		if e != nil {
			t.Fatal(e)
		}
		tokens = append(tokens, session.Token)
	}
	for i := 0; i < 2; i++ {
		b, e := tenants.Create(ctx, users[i], service.BusinessInput{Name: "HTTP fixture", Timezone: "UTC"})
		if e != nil {
			t.Fatal(e)
		}
		businesses = append(businesses, b.ID)
	}
	h, err := NewCatalogHandler(sessions, catalog, "http://localhost:3000")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	RegisterCatalogRoutes(mux, h)
	bh, e := NewBusinessHandler(sessions, tenants, "http://localhost:3000")
	if e != nil {
		t.Fatal(e)
	}
	mux.Handle("/api/v1/businesses/", bh)
	uuid := func(id pgtype.UUID) string { v, _ := id.MarshalJSON(); return string(v[1 : len(v)-1]) }

	request := func(actor int, method, path, body, origin, media string, want int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body)).WithContext(ctx)
		if actor >= 0 {
			r.AddCookie(&http.Cookie{Name: "booking_session", Value: tokens[actor]})
		}
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		if media != "" {
			r.Header.Set("Content-Type", media)
		}
		r.Header.Set("X-User-ID", fmt.Sprint(users[0]))
		r.Header.Set("X-Role", "BUSINESS_OWNER")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("%s %s actor %d got %d want %d: %s", method, path, actor, w.Code, want, w.Body.String())
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("missing no-store")
		}
		return w
	}
	const origin = "http://localhost:3000"
	get := func(actor int, path string, want int) *httptest.ResponseRecorder {
		return request(actor, "GET", path, "", origin, "", want)
	}
	mutate := func(actor int, method, path, body string, want int) *httptest.ResponseRecorder {
		return request(actor, method, path, body, origin, "application/json", want)
	}

	path := "/api/v1/businesses/" + uuid(businesses[0]) + "/settings"
	empty := get(0, path, 200)
	if !strings.Contains(empty.Body.String(), `"tagline":""`) {
		t.Fatal("missing defaults")
	}
	input := service.BusinessSettings{Tagline: " Helpful appointments ", Description: "First line\nSecond line", ContactEmail: "business@example.invalid", ContactPhone: "+63 (2) 123-4567", Website: "https://example.invalid/business"}
	body, _ := json.Marshal(input)
	request(-1, "PUT", path, string(body), origin, "application/json", 401)
	request(0, "PUT", path, string(body), "https://foreign.invalid", "application/json", 403)
	mutate(1, "PUT", path, string(body), 403)
	get(1, path, 403)
	mutate(0, "POST", path, string(body), 405)
	get(0, path+"?unknown=true", 400)
	w := mutate(0, "PUT", path, string(body), 200)
	if strings.Contains(w.Body.String(), "business_id") || strings.Contains(w.Body.String(), "updated_at") {
		t.Fatal("private fields")
	}
	if !strings.Contains(w.Body.String(), `"tagline":"Helpful appointments"`) {
		t.Fatal("normalization")
	}
	if get(0, path, 200).Body.String() != w.Body.String() {
		t.Fatal("not persisted")
	}
	for _, bad := range []string{`{}`, `null`, `{"tagline":null}`, string(body) + ` {}`, strings.Replace(string(body), "https://example.invalid/business", "javascript:alert(1)", 1), strings.Replace(string(body), "business@example.invalid", "not-an-email", 1), strings.Replace(string(body), "https://example.invalid/business", "https://user:pass@example.invalid", 1)} {
		mutate(0, "PUT", path, bad, 400)
	}
	mutate(0, "PUT", path, `{"tagline":"`+strings.Repeat("x", 5000)+`"}`, 413)
	for _, role := range []service.Role{service.RoleStaff, service.RoleManager, service.RoleBusinessAdmin} {
		if e := tenants.ChangeMember(ctx, users[0], businesses[0], users[2], role, false); e != nil {
			t.Fatal(e)
		}
		get(2, path, 200)
		status := 403
		if role == service.RoleBusinessAdmin {
			status = 200
		}
		mutate(2, "PUT", path, string(body), status)
	}
	// Settings changes and their response cannot survive a failed transaction.
	failing := service.NewBusinessService(settingsRollback{repository.NewBusinessTransactions(pool)})
	if _, e := failing.SaveSettings(ctx, users[0], businesses[0], service.BusinessSettings{Tagline: "Must roll back"}); e == nil {
		t.Fatal("rollback not reported")
	}
	if !strings.Contains(get(0, path, 200).Body.String(), "Helpful appointments") {
		t.Fatal("rollback failed")
	}
	// Existing membership API: owner only, no owner replacement/removal.
	members := "/api/v1/businesses/" + uuid(businesses[0]) + "/memberships/"
	mutate(2, "PUT", members+fmt.Sprint(users[3]), `{"role":"STAFF"}`, 403)
	mutate(0, "PUT", members+fmt.Sprint(users[0]), `{"role":"STAFF"}`, 403)
	mutate(0, "DELETE", members+fmt.Sprint(users[0]), "", 403)
	mutate(0, "PUT", members+fmt.Sprint(users[3]), `{"role":"STAFF"}`, 204)
	get(3, path, 200)
	mutate(0, "DELETE", members+fmt.Sprint(users[3]), "", 204)
	get(3, path, 403)
	down, e := os.ReadFile("../../../../migrations/000016_business_settings.down.sql")
	if e != nil {
		t.Fatal(e)
	}
	tx, e := pool.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	_, e = tx.Exec(ctx, string(down))
	_ = tx.Rollback(ctx)
	if e == nil {
		t.Fatal("downgrade discarded customization")
	}
	blank, _ := json.Marshal(service.BusinessSettings{})
	mutate(0, "PUT", path, string(blank), 200)
	if get(0, path, 200).Body.String() != empty.Body.String() {
		t.Fatal("clear failed")
	}
	if _, e := pool.Exec(ctx, "UPDATE businesses SET status='disabled' WHERE id=$1", businesses[0]); e != nil {
		t.Fatal(e)
	}
	get(0, path, 403)
}

type settingsRollback struct{ service.BusinessTransactions }

func (t settingsRollback) WithinBusiness(ctx context.Context, fn func(service.BusinessStore) error) error {
	return t.BusinessTransactions.WithinBusiness(ctx, func(q service.BusinessStore) error {
		if e := fn(q); e != nil {
			return e
		}
		return errors.New("forced rollback")
	})
}
