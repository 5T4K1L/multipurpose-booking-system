package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"multipurpose-booking-system/api/internal/db"
	"multipurpose-booking-system/api/internal/repository"
	"multipurpose-booking-system/api/internal/service"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestBookingCompletionIntegration(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	pool, e := pgxpool.New(ctx, url)
	if e != nil {
		t.Fatal(e)
	}
	defer pool.Close()
	users := []int64{}
	businesses := []pgtype.UUID{}
	defer func() {
		c, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		for _, id := range businesses {
			for _, table := range []string{"booking_changes", "bookings", "staff_availability_windows", "resource_availability_windows", "location_weekly_hours", "location_closed_dates", "service_scheduling_rules", "service_locations", "service_staff", "service_resources", "staff_locations", "resources", "staff_profiles", "services", "locations"} {
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
	sessions, e := service.NewSessionService(repository.NewSessionTransactions(pool), time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	tokens := []string{}
	for i := 0; i < 3; i++ {
		var id int64
		if e := pool.QueryRow(ctx, "INSERT INTO users(email,username) VALUES($1,'lifecycle fixture') RETURNING id", fmt.Sprintf("lifecycle-%d-%d@example.invalid", time.Now().UnixNano(), i)).Scan(&id); e != nil {
			t.Fatal(e)
		}
		users = append(users, id)
		issued, e := sessions.Create(ctx, id)
		if e != nil {
			t.Fatal(e)
		}
		tokens = append(tokens, issued.Token)
	}
	for i := 0; i < 2; i++ {
		b, e := tenants.Create(ctx, users[i], service.BusinessInput{Name: "Availability fixture", Timezone: "UTC"})
		if e != nil {
			t.Fatal(e)
		}
		businesses = append(businesses, b.ID)
	}
	a, b := businesses[0], businesses[1]
	loc, e := catalog.CreateLocation(ctx, users[0], a, service.LocationInput{Name: "Main", Timezone: "UTC"})
	if e != nil {
		t.Fatal(e)
	}
	offered, e := catalog.CreateService(ctx, users[0], a, service.ServiceInput{Name: "Consultation", DurationMinutes: 30})
	if e != nil {
		t.Fatal(e)
	}
	staff, e := catalog.CreateStaffProfile(ctx, users[0], a, service.StaffProfileInput{Name: "Alex"})
	if e != nil {
		t.Fatal(e)
	}
	resource, e := catalog.CreateResource(ctx, users[0], a, service.ResourceInput{Name: "Room", LocationID: loc.ID, Capacity: 2})
	if e != nil {
		t.Fatal(e)
	}

	bookings := service.NewBookingService(tenants)

	seed := func(key string, start, end, busyEnd time.Time) pgtype.UUID {
		t.Helper()
		var id pgtype.UUID
		e := pool.QueryRow(ctx, `INSERT INTO bookings(business_id,service_id,location_id,staff_id,resource_id,created_by,idempotency_key,request_hash,starts_at,ends_at,busy_starts_at,busy_ends_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$9,$11) RETURNING id`, a, offered.ID, loc.ID, staff.ID, resource.ID, users[0], key, strings.Repeat("a", 64), start, end, busyEnd).Scan(&id)
		if e != nil {
			t.Fatal(e)
		}
		return id
	}
	now := time.Now().UTC().Truncate(time.Minute)
	past := seed("past", now.Add(-2*time.Hour), now.Add(-time.Hour), now.Add(-45*time.Minute))
	future := seed("future", now.Add(time.Hour), now.Add(2*time.Hour), now.Add(3*time.Hour))
	cleanup := seed("cleanup", now.Add(-30*time.Minute), now.Add(-10*time.Minute), now.Add(10*time.Minute))
	check := func(id pgtype.UUID, key string, want error) service.BookingChangeResult {
		t.Helper()
		v, e := bookings.Complete(ctx, users[0], a, id, key)
		if !errors.Is(e, want) {
			t.Fatalf("completion got %v want %v", e, want)
		}
		if e != nil && v.Booking.ID.Valid {
			t.Fatal("failed result leaked")
		}
		return v
	}
	check(future, "early", service.ErrBookingState)
	check(cleanup, "cleanup", service.ErrBookingState)
	// Audit failure rolls back the state transition.
	failing := service.NewBookingService(service.NewBusinessService(lifecycleAuditFailure{repository.NewBusinessTransactions(pool)}))
	if _, e = failing.Complete(ctx, users[0], a, past, "audit-fail"); !errors.Is(e, service.ErrBookingUnavailable) {
		t.Fatal("audit failure", e)
	}
	row, e := db.New(pool).GetTenantBooking(ctx, db.GetTenantBookingParams{BusinessID: a, ID: past})
	if e != nil || row.Status != "confirmed" {
		t.Fatal("audit rollback", e)
	}
	// PostgreSQL independently prevents early completion, including cleanup.
	if _, e = pool.Exec(ctx, "UPDATE bookings SET status='completed',closed_by=$1 WHERE id=$2", users[0], cleanup); e == nil {
		t.Fatal("database allowed early completion")
	}
	h, e := NewBookingHandler(sessions, bookings, "http://localhost:3000")
	if e != nil {
		t.Fatal(e)
	}
	mux := http.NewServeMux()
	RegisterBookingRoutes(mux, h)
	uuid := func(id pgtype.UUID) string { v, _ := id.MarshalJSON(); return string(v[1 : len(v)-1]) }
	request := func(actor int, business, id pgtype.UUID, key, body, origin string, want int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest("POST", "/api/v1/businesses/"+uuid(business)+"/bookings/"+uuid(id)+"/complete", strings.NewReader(body)).WithContext(ctx)
		if actor >= 0 {
			r.AddCookie(&http.Cookie{Name: "booking_session", Value: tokens[actor]})
		}
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", origin)
		r.Header.Set("Idempotency-Key", key)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("completion HTTP got %d want %d: %s", w.Code, want, w.Body.String())
		}
		return w
	}
	const origin = "http://localhost:3000"
	request(-1, a, past, "anon", "{}", origin, 401)
	request(1, a, past, "foreign", "{}", origin, 403)
	request(1, b, past, "other", "{}", origin, 404)
	if e = tenants.ChangeMember(ctx, users[0], a, users[2], service.RoleStaff, false); e != nil {
		t.Fatal(e)
	}
	request(2, a, past, "staff", "{}", origin, 403)
	request(0, a, past, "origin", "{}", "https://foreign.invalid", 403)
	request(0, a, past, "body", `{"status":"completed"}`, origin, 400)
	request(0, a, future, "early-http", "{}", origin, 409)
	if e = tenants.ChangeMember(ctx, users[0], a, users[2], service.RoleBusinessAdmin, false); e != nil {
		t.Fatal(e)
	}
	request(2, a, past, "done", "{}", origin, 200)
	w := request(2, a, past, "done", "{}", origin, 200)
	if !strings.Contains(w.Body.String(), `"replayed":true`) {
		t.Fatal("missing replay")
	}
	request(2, a, past, "done-again", "{}", origin, 409)
	request(2, a, future, "done", "{}", origin, 409)
	row, e = db.New(pool).GetTenantBooking(ctx, db.GetTenantBookingParams{BusinessID: a, ID: past})
	if e != nil || row.Status != "completed" || !row.ClosedAt.Valid || row.ClosedBy.Int64 != users[2] || !row.EndsAt.Time.Equal(now.Add(-time.Hour)) {
		t.Fatal("completion audit", e)
	}
	if _, e = bookings.Cancel(ctx, users[0], a, past, "cancel-completed"); !errors.Is(e, service.ErrBookingState) {
		t.Fatal("completed cancel", e)
	}
	if _, e = pool.Exec(ctx, "UPDATE bookings SET status='confirmed',closed_at=NULL,closed_by=NULL WHERE id=$1", past); e == nil {
		t.Fatal("reopened completed booking")
	}
	if e = tenants.ChangeMember(ctx, users[0], a, users[2], service.RoleStaff, true); e != nil {
		t.Fatal(e)
	}
	request(2, a, past, "done", "{}", origin, 403)
	// Downgrade refuses to discard completed history.
	down, e := os.ReadFile("../../../../migrations/000015_booking_completion.down.sql")
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
		t.Fatal("downgrade accepted completed history")
	}
}
