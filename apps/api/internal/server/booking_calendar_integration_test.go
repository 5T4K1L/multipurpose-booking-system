package server

import (
	"context"
	"encoding/json"
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

func TestBookingCalendarIntegration(t *testing.T) {
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
	bookings := service.NewBookingService(tenants)
	bookingHandler, e := NewBookingHandler(sessions, bookings, "http://localhost:3000")
	if e != nil {
		t.Fatal(e)
	}
	RegisterBookingRoutes(mux, bookingHandler)
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

	if _, e := tenants.UpdateProfile(ctx, users[0], businesses[0], service.BusinessInput{Name: "HTTP fixture", Timezone: "America/New_York"}); e != nil {
		t.Fatal(e)
	}
	loc, e := catalog.CreateLocation(ctx, users[0], businesses[0], service.LocationInput{Name: "Main", Timezone: "UTC"})
	if e != nil {
		t.Fatal(e)
	}
	svc, e := catalog.CreateService(ctx, users[0], businesses[0], service.ServiceInput{Name: "Visit", DurationMinutes: 30})
	if e != nil {
		t.Fatal(e)
	}
	seed := func(key string, start time.Time) {
		t.Helper()
		_, e := pool.Exec(ctx, `INSERT INTO bookings(business_id,service_id,location_id,created_by,idempotency_key,request_hash,starts_at,ends_at,busy_starts_at,busy_ends_at) VALUES($1,$2,$3,$4,$5,$6,$7,$7::timestamptz+interval '30 minutes',$7,$7::timestamptz+interval '30 minutes')`, businesses[0], svc.ID, loc.ID, users[0], key, strings.Repeat("a", 64), start)
		if e != nil {
			t.Fatal(e)
		}
	}
	// New York's fall-back day is 25 hours. Both repeated 01:30 instants belong to it.
	day := time.Date(2026, 11, 1, 4, 0, 0, 0, time.UTC)
	for i := 0; i < 105; i++ {
		seed(fmt.Sprint(i), day.Add(time.Duration(i/2)*time.Minute))
	}
	seed("fold-first", day.Add(90*time.Minute))
	seed("fold-second", day.Add(150*time.Minute))
	seed("last-minute", day.Add(25*time.Hour-time.Minute))
	seed("before", day.Add(-time.Minute))
	seed("after", day.Add(25*time.Hour))
	path := "/api/v1/businesses/" + uuid(businesses[0]) + "/bookings/calendar"
	query := "?from=2026-11-01&to=2026-11-01&limit=50"
	get(-1, path+query, 401)
	get(1, path+query, 403)
	request(0, "POST", path, `{}`, origin, "application/json", 405)
	request(0, "GET", path+query, "", "https://foreign.invalid", "", 403)
	for _, bad := range []string{"", "?from=bad&to=2026-11-01", "?from=2026-11-01&to=2026-12-02", "?from=2026-11-02&to=2026-11-01", query + "&from=2026-11-01", query + "&status=confirmed", query + "&cursor=bad"} {
		get(0, path+bad, 400)
	}
	seen := map[pgtype.UUID]bool{}
	cursor := ""
	var firstCursor string
	var previous service.CalendarEntry
	for {
		url := path + query
		if cursor != "" {
			url += "&cursor=" + cursor
		}
		w := get(0, url, 200)
		var page service.BookingCalendar
		if e := json.Unmarshal(w.Body.Bytes(), &page); e != nil {
			t.Fatal(e)
		}
		if page.Counts.Total != 108 || page.Counts.Confirmed != 108 || page.Timezone != "America/New_York" {
			t.Fatal("counts/timezone", page.Counts, page.Timezone)
		}
		if strings.Contains(w.Body.String(), "request_hash") || strings.Contains(w.Body.String(), "created_by") {
			t.Fatal("private fields exposed")
		}
		for _, v := range page.Items {
			if seen[v.ID] || v.StartsAt.Before(previous.StartsAt) {
				t.Fatal("pagination order/duplicate")
			}
			seen[v.ID] = true
			previous = v
			if v.ServiceName != "Visit" || v.LocationName != "Main" {
				t.Fatal("labels")
			}
		}
		cursor = page.NextCursor
		if firstCursor == "" {
			firstCursor = cursor
		}
		if cursor == "" {
			break
		}
	}
	if len(seen) != 108 {
		t.Fatal("missing bookings", len(seen))
	}
	get(0, path+"?from=2026-11-02&to=2026-11-02&cursor="+firstCursor, 400)
	if e := tenants.ChangeMember(ctx, users[0], businesses[0], users[2], service.RoleStaff, false); e != nil {
		t.Fatal(e)
	}
	get(2, path+query, 200)
	get(2, path+query+"&cursor="+firstCursor, 400)
	if _, e := pool.Exec(ctx, "UPDATE services SET status='archived' WHERE id=$1", svc.ID); e != nil {
		t.Fatal(e)
	}
	get(0, path+query, 200)
	if e := tenants.ChangeMember(ctx, users[0], businesses[0], users[2], service.RoleStaff, true); e != nil {
		t.Fatal(e)
	}
	get(2, path+query, 403)
	if _, e := tenants.UpdateProfile(ctx, users[0], businesses[0], service.BusinessInput{Name: "HTTP fixture", Timezone: "UTC"}); e != nil {
		t.Fatal(e)
	}
	get(0, path+query+"&cursor="+firstCursor, 400)
}
