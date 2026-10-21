package server

import (
	"context"
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

func TestCatalogUpdatesIntegration(t *testing.T) {
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
			for _, table := range []string{"bookings", "location_weekly_hours", "location_closed_dates", "service_locations", "service_staff", "service_resources", "staff_locations", "resources", "staff_profiles", "services", "locations"} {
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
	base := "/api/v1/businesses/" + uuid(businesses[0]) + "/catalog/"
	otherBase := "/api/v1/businesses/" + uuid(businesses[1]) + "/catalog/"
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

	profile := "/api/v1/businesses/" + uuid(businesses[0])
	mutate(0, "PUT", profile, `{"name":"Updated business","timezone":"Asia/Manila"}`, 400)
	mutate(0, "PUT", profile, `{"name":"HTTP fixture","timezone":"Asia/Manila"}`, 200)
	if !strings.Contains(get(0, profile, 200).Body.String(), "HTTP fixture") {
		t.Fatal("profile not persisted")
	}
	mutate(1, "PUT", profile, `{"name":"Denied","timezone":"UTC"}`, 403)
	for _, body := range []string{`null`, `{}`, `{"name":"X","timezone":"Local"}`, `{"name":"X","timezone":"UTC","role":"BUSINESS_OWNER"}`} {
		mutate(0, "PUT", profile, body, 400)
	}
	loc, e := catalog.CreateLocation(ctx, users[0], businesses[0], service.LocationInput{Name: "Location", Timezone: "UTC"})
	if e != nil {
		t.Fatal(e)
	}
	svc, e := catalog.CreateService(ctx, users[0], businesses[0], service.ServiceInput{Name: "Service", DurationMinutes: 30})
	if e != nil {
		t.Fatal(e)
	}
	staff, e := catalog.CreateStaffProfile(ctx, users[0], businesses[0], service.StaffProfileInput{Name: "Staff"})
	if e != nil {
		t.Fatal(e)
	}
	res, e := catalog.CreateResource(ctx, users[0], businesses[0], service.ResourceInput{Name: "Resource", LocationID: loc.ID, Capacity: 2})
	if e != nil {
		t.Fatal(e)
	}
	// Whole-day is a persisted fixed 24-hour mode, not a display-only label.
	whole, e := catalog.CreateService(ctx, users[0], businesses[0], service.ServiceInput{Name: "Whole day", AllDay: true})
	if e != nil || !whole.AllDay || whole.DurationMinutes != 1440 {
		t.Fatal("whole-day creation", e)
	}
	if e = catalog.ChangeEligibility(ctx, users[0], businesses[0], whole.ID, loc.ID, service.ServiceLocation, false); e != nil {
		t.Fatal(e)
	}
	hours := []service.WeeklyHours{}
	for d := int32(0); d < 7; d++ {
		hours = append(hours, service.WeeklyHours{Weekday: d, StartMinute: 0, EndMinute: 1440})
	}
	if _, e = catalog.PutLocationSchedule(ctx, users[0], businesses[0], loc.ID, service.LocationScheduleInput{WeeklyHours: hours, ClosedDates: []string{}}); e != nil {
		t.Fatal(e)
	}
	date := time.Now().UTC().Add(72 * time.Hour).Format("2006-01-02")
	slots, e := catalog.CalculateSlots(ctx, users[0], businesses[0], whole.ID, service.SlotRequest{Date: date, LocationID: loc.ID})
	if e != nil || len(slots.Slots) == 0 {
		t.Fatal("whole-day slots", e)
	}
	for _, slot := range slots.Slots {
		if slot.EndsAt.Sub(slot.StartsAt) != 24*time.Hour {
			t.Fatal("not 24 elapsed hours")
		}
	}
	if _, e = catalog.PutBookingOptions(ctx, users[0], businesses[0], whole.ID, service.BookingOptions{Enabled: true, MinimumMinutes: 30, MaximumMinutes: 60}); e != service.ErrInvalidCatalog {
		t.Fatal("whole day allowed variable duration", e)
	}
	for i := range hours {
		hours[i].StartMinute = 540
		hours[i].EndMinute = 1020
	}
	if _, e = catalog.PutLocationSchedule(ctx, users[0], businesses[0], loc.ID, service.LocationScheduleInput{WeeklyHours: hours, ClosedDates: []string{}}); e != nil {
		t.Fatal(e)
	}
	slots, e = catalog.CalculateSlots(ctx, users[0], businesses[0], whole.ID, service.SlotRequest{Date: date, LocationID: loc.ID})
	if e != nil || len(slots.Slots) == 0 {
		t.Fatal("whole day failed to use opening hours", e)
	}
	for _, slot := range slots.Slots {
		if slot.StartsAt.Hour() != 9 || slot.EndsAt.Hour() != 17 || slot.EndsAt.Sub(slot.StartsAt) != 8*time.Hour {
			t.Fatal("invalid whole-day start or duration")
		}
	}
	mutate(0, "PUT", base+"services/"+uuid(whole.ID), `{"name":"Whole day updated","all_day":true}`, 200)
	if !strings.Contains(get(0, base+"services/"+uuid(whole.ID), 200).Body.String(), `"all_day":true`) {
		t.Fatal("whole day missing in API")
	}
	if e := tenants.ChangeMember(ctx, users[0], businesses[0], users[2], service.RoleStaff, false); e != nil {
		t.Fatal(e)
	}
	mutate(0, "PUT", base+"services/"+uuid(svc.ID)+"/locations", `{}`, 405)
	cases := []struct {
		kind string
		id   pgtype.UUID
		body string
	}{
		{"locations", loc.ID, `{"name":"New location","timezone":"Asia/Manila"}`},
		{"services", svc.ID, `{"name":"New service","duration_minutes":45}`},
		{"staff", staff.ID, `{"name":"New staff","role_name":"  Barber  "}`},
		{"resources", res.ID, `{"name":"New resource","capacity":3}`},
	}
	for _, tc := range cases {
		path := base + tc.kind + "/" + uuid(tc.id)
		mutate(-1, "PUT", path, tc.body, 401)
		mutate(2, "PUT", path, tc.body, 403)
		mutate(2, "DELETE", path, "", 403)
		mutate(1, "PUT", otherBase+tc.kind+"/"+uuid(tc.id), tc.body, 404)
		request(0, "PUT", path, tc.body, "https://foreign.invalid", "application/json", 403)
		w := mutate(0, "PUT", path, tc.body, 200)
		if !strings.Contains(w.Body.String(), "New ") {
			t.Fatal("edit response", w.Body.String())
		}
		if get(0, path, 200).Body.String() != w.Body.String() {
			t.Fatal("edit not persisted")
		}
	}

	if !strings.Contains(get(0, base+"staff/"+uuid(staff.ID), 200).Body.String(), `"role_name":"Barber"`) {
		t.Fatal("custom staff role did not persist")
	}
	mutate(0, "POST", base+"staff", `{"name":"Tutor fixture","role_name":"Tutor"}`, 201)
	for _, body := range []string{`{"name":"Person","role_name":"` + strings.Repeat("x", 61) + `"}`, `{"name":"Person","role_name":"bad\nrole"}`, `{"name":"Person","role_name":123}`} {
		mutate(0, "POST", base+"staff", body, 400)
	}
	mutate(2, "PUT", profile, `{"name":"Denied","timezone":"UTC"}`, 403)
	if e := tenants.ChangeMember(ctx, users[0], businesses[0], users[2], service.RoleBusinessAdmin, false); e != nil {
		t.Fatal(e)
	}
	mutate(2, "PUT", profile, `{"name":"HTTP fixture","timezone":"UTC"}`, 200)
	mutate(2, "PUT", base+"staff/"+uuid(staff.ID), `{"name":"Admin staff"}`, 200)
	// A saved booking retains its time snapshot when catalog duration changes.
	start := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Minute)
	end := start.Add(30 * time.Minute)
	_, e = pool.Exec(ctx, `INSERT INTO bookings(business_id,service_id,location_id,resource_id,created_by,idempotency_key,request_hash,starts_at,ends_at,busy_starts_at,busy_ends_at) VALUES($1,$2,$3,$4,$5,'edit-fixture',$6,$7,$8,$7,$8)`, businesses[0], svc.ID, loc.ID, res.ID, users[0], strings.Repeat("a", 64), start, end)
	if e != nil {
		t.Fatal(e)
	}
	mutate(0, "PUT", base+"services/"+uuid(svc.ID), `{"name":"New duration","duration_minutes":60}`, 200)
	var savedEnd time.Time
	if e = pool.QueryRow(ctx, "SELECT ends_at FROM bookings WHERE business_id=$1", businesses[0]).Scan(&savedEnd); e != nil || !savedEnd.Equal(end) {
		t.Fatal("booking changed", e)
	}
	mutate(0, "PUT", base+"resources/"+uuid(res.ID), `{"name":"Must not save","capacity":4}`, 409)
	if !strings.Contains(get(0, base+"resources/"+uuid(res.ID), 200).Body.String(), "New resource") {
		t.Fatal("partial resource update")
	}
	mutate(0, "PUT", base+"resources/"+uuid(res.ID), `{"name":"Rename booked resource","capacity":3}`, 200)
	mutate(0, "PUT", base+"resources/"+uuid(res.ID), `{"name":"Move","capacity":3,"location_id":"`+uuid(loc.ID)+`"}`, 400)
	// Deletion is scoped, permission checked, and preserves booking history.
	for _, tc := range cases {
		path := base + tc.kind + "/" + uuid(tc.id)
		mutate(-1, "DELETE", path, "", 401)
		mutate(1, "DELETE", otherBase+tc.kind+"/"+uuid(tc.id), "", 404)
	}
	for _, tc := range cases {
		if tc.kind != "staff" {
			mutate(0, "DELETE", base+tc.kind+"/"+uuid(tc.id), "", 409)
		}
	}
	if _, e = pool.Exec(ctx, "UPDATE bookings SET status='cancelled',closed_at=now(),closed_by=created_by WHERE business_id=$1", businesses[0]); e != nil {
		t.Fatal(e)
	}
	for _, tc := range cases {
		if tc.kind != "staff" {
			path := base + tc.kind + "/" + uuid(tc.id)
			mutate(2, "DELETE", path, "", 204)
			get(0, path, 404)
			mutate(0, "DELETE", path, "", 404)
		}
	}
	var retained int
	if e = pool.QueryRow(ctx, "SELECT count(*) FROM bookings WHERE business_id=$1", businesses[0]).Scan(&retained); e != nil || retained != 1 {
		t.Fatal("booking history lost", e)
	}
	mutate(0, "DELETE", base+"staff/"+uuid(staff.ID), "", 204)
	mutate(0, "PUT", base+"staff/"+uuid(staff.ID), `{"name":"Archived"}`, 404)
	if e = tenants.ChangeMember(ctx, users[0], businesses[0], users[2], service.RoleStaff, true); e != nil {
		t.Fatal(e)
	}
	mutate(2, "PUT", profile, `{"name":"Revoked","timezone":"UTC"}`, 403)
}
