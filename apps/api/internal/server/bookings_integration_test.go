package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

type bookingRollbackTransactions struct{ base service.BusinessTransactions }

func (f bookingRollbackTransactions) WithinBusiness(ctx context.Context, fn func(service.BusinessStore) error) error {
	return f.base.WithinBusiness(ctx, func(q service.BusinessStore) error {
		if e := fn(q); e != nil {
			return e
		}
		return errors.New("injected rollback")
	})
}
func TestBookingIntegration(t *testing.T) {
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
			for _, table := range []string{"bookings", "staff_availability_windows", "resource_availability_windows", "location_weekly_hours", "location_closed_dates", "service_scheduling_rules", "service_locations", "service_staff", "service_resources", "staff_locations", "resources", "staff_profiles", "services", "locations"} {
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
		if e := pool.QueryRow(ctx, "INSERT INTO users(email,username) VALUES($1,'booking fixture') RETURNING id", fmt.Sprintf("booking-%d-%d@example.invalid", time.Now().UnixNano(), i)).Scan(&id); e != nil {
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
	foreign, e := catalog.CreateStaffProfile(ctx, users[1], b, service.StaffProfileInput{Name: "Foreign"})
	if e != nil {
		t.Fatal(e)
	}

	bookings := service.NewBookingService(tenants)
	day := time.Now().UTC().Truncate(24 * time.Hour).Add(24 * time.Hour)
	in := service.BookingInput{ServiceID: offered.ID, LocationID: loc.ID, StaffID: staff.ID, ResourceID: resource.ID, StartsAt: day.Add(10 * time.Hour)}
	create := func(key string, input service.BookingInput, want error) service.BookingResult {
		t.Helper()
		v, e := bookings.Create(ctx, users[0], a, key, input)
		if !errors.Is(e, want) {
			t.Fatalf("%s: %v want %v", key, e, want)
		}
		if e != nil && v.ID.Valid {
			t.Fatal("failed result leaked")
		}
		return v
	}
	create("missing-config", in, service.ErrBookingConflict)
	for _, v := range []struct {
		parent, target pgtype.UUID
		kind           service.EligibilityKind
	}{{offered.ID, loc.ID, service.ServiceLocation}, {offered.ID, staff.ID, service.ServiceStaff}, {offered.ID, resource.ID, service.ServiceResource}, {staff.ID, loc.ID, service.StaffLocation}} {
		if e := catalog.ChangeEligibility(ctx, users[0], a, v.parent, v.target, v.kind, false); e != nil {
			t.Fatal(e)
		}
	}
	rules := service.SchedulingRulesInput{BufferBeforeMinutes: 15, BufferAfterMinutes: 15}
	if _, e := catalog.PutSchedulingRules(ctx, users[0], a, offered.ID, rules); e != nil {
		t.Fatal(e)
	}
	schedule := service.LocationScheduleInput{WeeklyHours: []service.WeeklyHours{{Weekday: int32(day.Weekday()), StartMinute: 540, EndMinute: 1020}}, ClosedDates: []string{}}
	if _, e := catalog.PutLocationSchedule(ctx, users[0], a, loc.ID, schedule); e != nil {
		t.Fatal(e)
	}
	windows := []service.AvailabilityWindow{{StartsAt: day.Add(9 * time.Hour), EndsAt: day.Add(17 * time.Hour)}}
	for kind, id := range map[string]pgtype.UUID{"staff": staff.ID, "resources": resource.ID} {
		if _, e := catalog.PutAvailabilityWindows(ctx, users[0], a, id, kind, windows); e != nil {
			t.Fatal(e)
		}
	}
	slots, e := catalog.CalculateSlots(ctx, users[0], a, offered.ID, service.SlotRequest{Date: day.Format("2006-01-02"), LocationID: loc.ID, StaffID: staff.ID, ResourceID: resource.ID})
	if e != nil || len(slots.Slots) == 0 {
		t.Fatal(slots, e)
	}
	// A previously returned advisory result is not authorization to book.
	if e := catalog.ChangeEligibility(ctx, users[0], a, offered.ID, staff.ID, service.ServiceStaff, true); e != nil {
		t.Fatal(e)
	}
	create("stale", in, service.ErrBookingConflict)
	if e := catalog.ChangeEligibility(ctx, users[0], a, offered.ID, staff.ID, service.ServiceStaff, false); e != nil {
		t.Fatal(e)
	}
	original := create("first", in, nil)
	if !original.ID.Valid || original.Replayed || original.Status != "confirmed" || !original.EndsAt.Equal(in.StartsAt.Add(30*time.Minute)) {
		t.Fatal(original)
	}
	offset := in
	offset.StartsAt = in.StartsAt.In(time.FixedZone("offset", 8*3600))
	replay := create("first", offset, nil)
	if replay.ID != original.ID || !replay.Replayed {
		t.Fatal(replay)
	}
	changed := in
	changed.StartsAt = changed.StartsAt.Add(time.Hour)
	create("first", changed, service.ErrIdempotencyConflict)
	create("staff-conflict", in, service.ErrBookingConflict)
	changed = in
	changed.StartsAt = changed.StartsAt.Add(30 * time.Minute)
	create("buffer-conflict", changed, service.ErrBookingConflict)
	changed.StartsAt = in.StartsAt.Add(time.Minute)
	create("off-grid", changed, service.ErrBookingConflict)
	changed = in
	changed.StaffID = foreign.ID
	create("foreign", changed, service.ErrBookingConflict)
	changed = in
	changed.StartsAt = day.Add(-time.Hour)
	create("past", changed, service.ErrBookingConflict)
	for _, role := range []service.Role{service.RoleManager, service.RoleStaff} {
		if e := tenants.ChangeMember(ctx, users[0], a, users[2], role, false); e != nil {
			t.Fatal(e)
		}
		if _, e := bookings.Create(ctx, users[2], a, "denied", in); e != service.ErrForbidden {
			t.Fatal(role, e)
		}
	}
	if _, e := bookings.Create(ctx, users[1], a, "other-tenant", in); e != service.ErrForbidden {
		t.Fatal(e)
	}
	// Transaction rollback must remove both occupancy and the idempotency result.
	changed = in
	changed.StartsAt = day.Add(11 * time.Hour)
	faulty := service.NewBookingService(service.NewBusinessService(bookingRollbackTransactions{repository.NewBusinessTransactions(pool)}))
	if v, e := faulty.Create(ctx, users[0], a, "rollback", changed); e != service.ErrBookingUnavailable || v.ID.Valid {
		t.Fatal(v, e)
	}
	create("rollback", changed, nil)
	concurrent := func(label string, n int, input service.BookingInput, sameKey bool, want int) {
		t.Helper()
		start := make(chan struct{})
		type answer struct {
			v service.BookingResult
			e error
		}
		out := make(chan answer, n)
		for i := 0; i < n; i++ {
			go func(i int) {
				<-start
				key := fmt.Sprintf("%s-%d", label, i)
				if sameKey {
					key = label
				}
				v, e := bookings.Create(ctx, users[0], a, key, input)
				out <- answer{v, e}
			}(i)
		}
		close(start)
		successes := 0
		var id pgtype.UUID
		for i := 0; i < n; i++ {
			result := <-out
			if result.e == nil {
				successes++
				if sameKey && id.Valid && result.v.ID != id {
					t.Error("duplicate retry created another booking")
				}
				id = result.v.ID
			} else if result.e != service.ErrBookingConflict {
				t.Error(result.e)
			}
		}
		if successes != want {
			t.Fatalf("%s successes=%d want=%d", label, successes, want)
		}
	}
	changed = in
	changed.StartsAt = day.Add(12 * time.Hour)
	concurrent("retry", 4, changed, true, 4)
	changed.StartsAt = day.Add(13 * time.Hour)
	concurrent("staff-race", 3, changed, false, 1)
	changed.StaffID = pgtype.UUID{}
	changed.StartsAt = day.Add(14 * time.Hour)
	concurrent("capacity-race", 3, changed, false, 2)
	// Direct database writers are protected without the application's tenant lock.
	stamp := func(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }
	raw := db.CreateBookingParams{BusinessID: a, ServiceID: offered.ID, LocationID: loc.ID, ResourceID: resource.ID, CreatedBy: users[0], RequestHash: strings.Repeat("0", 64)}
	rawInsert := func(p db.CreateBookingParams, key string, start, end time.Time) error {
		p.IdempotencyKey = key
		p.StartsAt = stamp(start)
		p.BusyStartsAt = p.StartsAt
		p.EndsAt = stamp(end)
		p.BusyEndsAt = p.EndsAt
		_, e := db.New(pool).CreateBooking(ctx, p)
		return e
	}
	rawStart := day.Add(48 * time.Hour)
	gate := make(chan struct{})
	rawErrors := make(chan error, 3)
	for i := 0; i < 3; i++ {
		go func(i int) {
			<-gate
			rawErrors <- rawInsert(raw, fmt.Sprintf("raw-race-%d", i), rawStart, rawStart.Add(time.Hour))
		}(i)
	}
	close(gate)
	count := 0
	for i := 0; i < 3; i++ {
		e := <-rawErrors
		if e == nil {
			count++
			continue
		}
		var pg *pgconn.PgError
		if !errors.As(e, &pg) || pg.Code != "23P01" {
			t.Fatal(e)
		}
	}
	if count != 2 {
		t.Fatal("direct SQL capacity protection", count)
	}

	// Hold a real direct-SQL booking open and prove a second connection waits on
	// its tenant lock before checking staff occupancy from a fresh snapshot.
	staffRaw := raw
	staffRaw.ResourceID = pgtype.UUID{}
	staffRaw.StaffID = staff.ID
	staffRaw.IdempotencyKey = "raw-staff-first"
	staffRaw.StartsAt = stamp(rawStart)
	staffRaw.EndsAt = stamp(rawStart.Add(time.Hour))
	staffRaw.BusyStartsAt = staffRaw.StartsAt
	staffRaw.BusyEndsAt = staffRaw.EndsAt
	tx, e := pool.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(context.Background())
	if _, e := db.New(tx).CreateBooking(ctx, staffRaw); e != nil {
		t.Fatal(e)
	}
	conn, e := pool.Acquire(ctx)
	if e != nil {
		t.Fatal(e)
	}
	pid := conn.Conn().PgConn().PID()
	blockedResult := make(chan error, 1)
	staffRaw.IdempotencyKey = "raw-staff-second"
	go func() { _, e := db.New(conn).CreateBooking(ctx, staffRaw); blockedResult <- e }()
	deadline := time.Now().Add(5 * time.Second)
	waiting := false
	for time.Now().Before(deadline) {
		if e := pool.QueryRow(ctx, "SELECT cardinality(pg_blocking_pids($1)) > 0", pid).Scan(&waiting); e != nil {
			t.Fatal(e)
		}
		if waiting {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !waiting {
		_ = tx.Rollback(ctx)
		<-blockedResult
		conn.Release()
		t.Fatal("direct SQL writer did not wait")
	}
	if e := tx.Commit(ctx); e != nil {
		t.Fatal(e)
	}
	e = <-blockedResult
	conn.Release()
	var conflict *pgconn.PgError
	if !errors.As(e, &conflict) || conflict.ConstraintName != "bookings_staff_conflict" {
		t.Fatal("staff conflict after wait", e)
	}
	// A stale repeatable-read transaction must abort, never silently overbook.
	stale, e := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if e != nil {
		t.Fatal(e)
	}
	defer stale.Rollback(context.Background())
	var snapshot int
	if e := stale.QueryRow(ctx, "SELECT count(*) FROM bookings WHERE business_id=$1", a).Scan(&snapshot); e != nil {
		t.Fatal(e)
	}
	if e := rawInsert(raw, "advance-tenant-version", rawStart.Add(10*time.Hour), rawStart.Add(11*time.Hour)); e != nil {
		t.Fatal(e)
	}
	staffRaw.IdempotencyKey = "stale-snapshot"
	_, e = db.New(stale).CreateBooking(ctx, staffRaw)
	if !errors.As(e, &conflict) || conflict.Code != "40001" {
		t.Fatal("stale snapshot was not aborted", e)
	}
	_ = stale.Rollback(ctx)
	// Two adjacent occupants plus one spanning both fit capacity 2 (not sum=3).
	rawStart = rawStart.Add(3 * time.Hour)
	for i, interval := range [][2]time.Time{{rawStart, rawStart.Add(time.Hour)}, {rawStart.Add(time.Hour), rawStart.Add(2 * time.Hour)}, {rawStart, rawStart.Add(2 * time.Hour)}} {
		if e := rawInsert(raw, fmt.Sprintf("sweep-%d", i), interval[0], interval[1]); e != nil {
			t.Fatal("capacity sweep rejected valid booking", e)
		}
	}
	if e := rawInsert(raw, "sweep-full", rawStart.Add(30*time.Minute), rawStart.Add(90*time.Minute)); e == nil {
		t.Fatal("capacity exceeded")
	}
	if _, e := pool.Exec(ctx, "UPDATE resources SET capacity=1 WHERE id=$1", resource.ID); e == nil {
		t.Fatal("capacity reduction bypass")
	}
	if _, e := pool.Exec(ctx, "UPDATE bookings SET starts_at=starts_at WHERE id=$1", original.ID); e == nil {
		t.Fatal("unsupported lifecycle update")
	}
	bad := raw
	bad.BusinessID = b
	if e := rawInsert(bad, "foreign-fk", rawStart, rawStart.Add(time.Hour)); e == nil {
		t.Fatal("cross tenant SQL insert")
	}
	// Replay remains stable after schedule changes, but still needs current authorization.
	schedule.ClosedDates = []string{day.Format("2006-01-02")}
	if _, e := catalog.PutLocationSchedule(ctx, users[0], a, loc.ID, schedule); e != nil {
		t.Fatal(e)
	}
	if v := create("first", in, nil); v.ID != original.ID {
		t.Fatal(v)
	}
	changed = in
	changed.StartsAt = day.Add(15 * time.Hour)
	create("closed", changed, service.ErrBookingConflict)

	h, e := NewBookingHandler(sessions, bookings, "http://localhost:3000")
	if e != nil {
		t.Fatal(e)
	}
	mux := http.NewServeMux()
	mux.Handle("/api/v1/businesses/{businessID}/bookings", h)
	idJSON, _ := a.MarshalJSON()
	path := "/api/v1/businesses/" + strings.Trim(string(idJSON), "\"") + "/bookings"
	inputJSON, _ := json.Marshal(in)
	request := func(actor int, method, body, key, origin string, want int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body)).WithContext(ctx)
		r.Header.Set("Content-Type", "application/json")
		if key != "" {
			r.Header.Set("Idempotency-Key", key)
		}
		r.Header.Set("Origin", origin)
		r.Header.Set("X-Role", "BUSINESS_OWNER")
		if actor >= 0 {
			r.AddCookie(&http.Cookie{Name: "booking_session", Value: tokens[actor]})
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("HTTP got %d want %d: %s", w.Code, want, w.Body.String())
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("missing no-store")
		}
		return w
	}
	const origin = "http://localhost:3000"
	request(-1, "OPTIONS", "", "", origin, 204)
	request(-1, "POST", string(inputJSON), "first", origin, 401)
	request(0, "POST", string(inputJSON), "first", "", 403)
	request(0, "DELETE", "", "", origin, 405)
	request(0, "POST", string(inputJSON), "", origin, 400)
	request(0, "POST", string(inputJSON), "bad key", origin, 400)
	request(0, "POST", string(inputJSON), "first", origin, 200)
	request(1, "POST", string(inputJSON), "first", origin, 403)
	request(2, "POST", string(inputJSON), "first", origin, 403)
	for _, body := range []string{`null`, `{}`, `[]`, `{"unknown":true}`, `{} {}`, `{"service_id":"invalid"}`} {
		request(0, "POST", body, "invalid", origin, 400)
	}
	request(0, "POST", `{"pad":"`+strings.Repeat("x", 5000)+`"}`, "too-large", origin, 413)
	changed = in
	changed.StartsAt = day.Add(15 * time.Hour)
	data, _ := json.Marshal(changed)
	request(0, "POST", string(data), "first", origin, 409)
	request(0, "POST", string(data), "closed-http", origin, 409)
	// Allow an admin to create a location-only reservation; no implicit staff/resource.
	schedule.ClosedDates = []string{}
	if _, e := catalog.PutLocationSchedule(ctx, users[0], a, loc.ID, schedule); e != nil {
		t.Fatal(e)
	}
	if e := tenants.ChangeMember(ctx, users[0], a, users[2], service.RoleBusinessAdmin, false); e != nil {
		t.Fatal(e)
	}
	changed.StaffID = pgtype.UUID{}
	changed.ResourceID = pgtype.UUID{}
	data, _ = json.Marshal(changed)
	request(2, "POST", string(data), "admin", origin, 201)
	request(2, "POST", string(data), "admin", origin, 200)
	if e := tenants.ChangeMember(ctx, users[0], a, users[2], service.RoleBusinessAdmin, true); e != nil {
		t.Fatal(e)
	}
	request(2, "POST", string(data), "admin", origin, 403)
	if _, e := pool.Exec(ctx, "UPDATE users SET status='disabled' WHERE id=$1", users[0]); e != nil {
		t.Fatal(e)
	}
	request(0, "POST", string(inputJSON), "first", origin, 401)

}
