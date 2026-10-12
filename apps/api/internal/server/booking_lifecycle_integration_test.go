package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

type lifecycleAuditFailure struct{ base service.BusinessTransactions }

func (f lifecycleAuditFailure) WithinBusiness(ctx context.Context, fn func(service.BusinessStore) error) error {
	return f.base.WithinBusiness(ctx, func(q service.BusinessStore) error { return fn(lifecycleAuditStore{q}) })
}

type lifecycleAuditStore struct{ service.BusinessStore }

func (lifecycleAuditStore) CreateBookingChange(context.Context, db.CreateBookingChangeParams) error {
	return errors.New("private audit write error")
}

func TestBookingLifecycleIntegration(t *testing.T) {
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

	lookup := func(id pgtype.UUID) db.Booking {
		t.Helper()
		v, e := db.New(pool).GetTenantBooking(ctx, db.GetTenantBookingParams{BusinessID: a, ID: id})
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	checkError := func(v service.BookingChangeResult, e, want error) {
		t.Helper()
		if !errors.Is(e, want) {
			t.Fatalf("got %v want %v", e, want)
		}
		if e != nil && (v.Booking.ID.Valid || v.Replacement != nil) {
			t.Fatal("failure leaked result", v)
		}
	}
	move := func(id pgtype.UUID, key string, start time.Time, want error) service.BookingChangeResult {
		t.Helper()
		v, e := bookings.Reschedule(ctx, users[0], a, id, key, start)
		checkError(v, e, want)
		return v
	}
	cancelBooking := func(id pgtype.UUID, key string, want error) service.BookingChangeResult {
		t.Helper()
		v, e := bookings.Cancel(ctx, users[0], a, id, key)
		checkError(v, e, want)
		return v
	}
	first := create("first", in, nil)
	otherInput := in
	otherInput.StartsAt = day.Add(12 * time.Hour)
	second := create("second", otherInput, nil)
	move(first.ID, "blocked-move", otherInput.StartsAt, service.ErrBookingConflict)
	if v := lookup(first.ID); v.Status != "confirmed" || v.ClosedAt.Valid {
		t.Fatal("failed move released original", v)
	}
	var count int
	if e := pool.QueryRow(ctx, "SELECT count(*) FROM booking_changes WHERE business_id=$1", a).Scan(&count); e != nil || count != 0 {
		t.Fatal(count, e)
	}
	// Moving by 15 minutes overlaps the original occupancy, which must be excluded.
	moved := move(first.ID, "move", in.StartsAt.Add(15*time.Minute), nil)
	if moved.Booking.Status != "rescheduled" || moved.Replacement == nil || moved.Replacement.ID == first.ID || moved.Replacement.Status != "confirmed" {
		t.Fatal(moved)
	}
	replacement := lookup(moved.Replacement.ID)
	closed := lookup(first.ID)
	if replacement.SupersedesBookingID != first.ID || closed.ClosedBy.Int64 != users[0] || !closed.ClosedAt.Valid || !closed.StartsAt.Time.Equal(in.StartsAt) {
		t.Fatal("missing audit/history", replacement, closed)
	}
	retry := move(first.ID, "move", in.StartsAt.Add(15*time.Minute).In(time.FixedZone("offset", 8*3600)), nil)
	if !retry.Replayed || retry.Replacement.ID != moved.Replacement.ID {
		t.Fatal(retry)
	}
	move(first.ID, "move", in.StartsAt.Add(30*time.Minute), service.ErrIdempotencyConflict)
	cancelled := cancelBooking(replacement.ID, "cancel", nil)
	if cancelled.Booking.Status != "cancelled" || cancelled.Replacement != nil || cancelled.Replayed {
		t.Fatal(cancelled)
	}
	if v := cancelBooking(replacement.ID, "cancel", nil); !v.Replayed {
		t.Fatal(v)
	}
	cancelBooking(first.ID, "cancel", service.ErrIdempotencyConflict)
	cancelBooking(first.ID, "terminal", service.ErrBookingState)
	move(replacement.ID, "terminal-move", day.Add(11*time.Hour), service.ErrBookingState)
	// Historical operation and creation retries do not claim current state.
	retry = move(first.ID, "move", in.StartsAt.Add(15*time.Minute), nil)
	if retry.Replacement.Status != "confirmed" {
		t.Fatal("historical retry changed", retry)
	}
	if v := create("first", in, nil); v.ID != first.ID || v.Status != "confirmed" || !v.Replayed {
		t.Fatal(v)
	}
	renewed := create("released-slot", in, nil) // Both staff and resource occupancy released.
	cancelBooking(renewed.ID, "clear-renewed", nil)

	// An audit failure or final transaction rollback restores original occupancy.
	for label, transactions := range map[string]service.BusinessTransactions{
		"audit":    lifecycleAuditFailure{repository.NewBusinessTransactions(pool)},
		"rollback": bookingRollbackTransactions{repository.NewBusinessTransactions(pool)},
	} {
		input := in
		input.StartsAt = day.Add(10 * time.Hour)
		original := create("fault-original-"+label, input, nil)
		failed := service.NewBookingService(service.NewBusinessService(transactions))
		v, e := failed.Reschedule(ctx, users[0], a, original.ID, "fault-"+label, day.Add(11*time.Hour))
		checkError(v, e, service.ErrBookingUnavailable)
		if lookup(original.ID).Status != "confirmed" {
			t.Fatal("failure lost original")
		}
		if e := pool.QueryRow(ctx, "SELECT count(*) FROM bookings WHERE business_id=$1 AND supersedes_booking_id=$2", a, original.ID).Scan(&count); e != nil || count != 0 {
			t.Fatal("replacement leaked", count, e)
		}
		recovered := move(original.ID, "fault-"+label, day.Add(11*time.Hour), nil)
		cancelBooking(recovered.Replacement.ID, "clear-"+label, nil)
	}

	// Cancellation also rolls back if its durable response cannot be saved.
	failedCancel := service.NewBookingService(service.NewBusinessService(lifecycleAuditFailure{repository.NewBusinessTransactions(pool)}))
	failedResult, failedError := failedCancel.Cancel(ctx, users[0], a, second.ID, "failed-cancel")
	checkError(failedResult, failedError, service.ErrBookingUnavailable)
	if lookup(second.ID).Status != "confirmed" {
		t.Fatal("audit failure released capacity")
	}
	// Competing terminal changes: exactly one succeeds, including cancel vs move.
	input := in
	input.StartsAt = day.Add(13 * time.Hour)
	race := create("race-original", input, nil)
	type answer struct {
		v service.BookingChangeResult
		e error
	}
	out := make(chan answer, 2)
	gate := make(chan struct{})
	go func() { <-gate; v, e := bookings.Cancel(ctx, users[0], a, race.ID, "race-cancel"); out <- answer{v, e} }()
	go func() {
		<-gate
		v, e := bookings.Reschedule(ctx, users[0], a, race.ID, "race-move", day.Add(14*time.Hour))
		out <- answer{v, e}
	}()
	close(gate)
	success := 0
	for i := 0; i < 2; i++ {
		v := <-out
		if v.e == nil {
			success++
			if v.v.Replacement != nil {
				cancelBooking(v.v.Replacement.ID, "clear-race", nil)
			}
		} else if v.e != service.ErrBookingState {
			t.Fatal(v.e)
		}
	}
	if success != 1 {
		t.Fatal("competing changes", success)
	}
	// Concurrent retries create one replacement and one durable operation response.
	input.StartsAt = day.Add(13 * time.Hour)
	race = create("retry-original", input, nil)
	many := make(chan answer, 4)
	gate = make(chan struct{})
	for i := 0; i < 4; i++ {
		go func() {
			<-gate
			v, e := bookings.Reschedule(ctx, users[0], a, race.ID, "retry-move", day.Add(14*time.Hour))
			many <- answer{v, e}
		}()
	}
	close(gate)
	var replacementID pgtype.UUID
	replays := 0
	for i := 0; i < 4; i++ {
		v := <-many
		if v.e != nil || v.v.Replacement == nil {
			t.Fatal(v)
		}
		if replacementID.Valid && replacementID != v.v.Replacement.ID {
			t.Fatal("duplicate replacement")
		}
		replacementID = v.v.Replacement.ID
		if v.v.Replayed {
			replays++
		}
	}
	if replays != 3 {
		t.Fatal("retry count", replays)
	}
	cancelBooking(replacementID, "clear-retries", nil)
	// Resource-only capacity is released, too (one remaining confirmed occupant).
	input = in
	input.StaffID = pgtype.UUID{}
	input.StartsAt = otherInput.StartsAt
	resourceOnly := create("resource-only", input, nil)
	create("full-resource", input, service.ErrBookingConflict)
	cancelBooking(resourceOnly.ID, "release-resource", nil)
	resourceOnly = create("resource-reused", input, nil)
	cancelBooking(resourceOnly.ID, "clear-resource", nil)

	// A direct SQL writer cannot consume capacity released only by an uncommitted
	// cancellation. Force it to wait, roll the cancellation back, and check conflict.
	input = in
	input.StartsAt = day.Add(15 * time.Hour)
	locked := create("locked-original", input, nil)
	holding, e := pool.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer holding.Rollback(context.Background())
	_, e = db.New(holding).CloseBooking(ctx, db.CloseBookingParams{BusinessID: a, ID: locked.ID, Status: "cancelled", ClosedBy: pgtype.Int8{Int64: users[0], Valid: true}})
	if e != nil {
		t.Fatal(e)
	}
	conn, e := pool.Acquire(ctx)
	if e != nil {
		t.Fatal(e)
	}
	pid := conn.Conn().PgConn().PID()
	waitingResult := make(chan error, 1)
	oldLocked := lookup(locked.ID)
	go func() {
		_, e := db.New(conn).CreateBooking(ctx, db.CreateBookingParams{BusinessID: a, ServiceID: oldLocked.ServiceID, LocationID: oldLocked.LocationID, StaffID: oldLocked.StaffID, ResourceID: oldLocked.ResourceID, CreatedBy: users[0], IdempotencyKey: "waiting-insert", RequestHash: strings.Repeat("0", 64), StartsAt: oldLocked.StartsAt, EndsAt: oldLocked.EndsAt, BusyStartsAt: oldLocked.BusyStartsAt, BusyEndsAt: oldLocked.BusyEndsAt})
		waitingResult <- e
	}()
	waiting := false
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if e := pool.QueryRow(ctx, "SELECT cardinality(pg_blocking_pids($1))>0", pid).Scan(&waiting); e != nil {
			t.Fatal(e)
		}
		if waiting {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !waiting {
		_ = holding.Rollback(ctx)
		<-waitingResult
		conn.Release()
		t.Fatal("writer did not wait for cancellation")
	}
	if e := holding.Rollback(ctx); e != nil {
		t.Fatal(e)
	}
	waitError := <-waitingResult
	conn.Release()
	var waitPG *pgconn.PgError
	if !errors.As(waitError, &waitPG) || waitPG.ConstraintName != "bookings_staff_conflict" {
		t.Fatal("uncommitted release admitted booking", waitError)
	}
	cancelBooking(locked.ID, "clear-locked", nil)
	// Missing eligibility prevents rescheduling but cannot prevent cancellation.
	input = in
	input.StartsAt = day.Add(15 * time.Hour)
	stale := create("stale-original", input, nil)
	if e := catalog.ChangeEligibility(ctx, users[0], a, offered.ID, staff.ID, service.ServiceStaff, true); e != nil {
		t.Fatal(e)
	}
	move(stale.ID, "stale", day.Add(16*time.Hour), service.ErrBookingConflict)
	cancelBooking(stale.ID, "cancel-stale", nil)
	if e := catalog.ChangeEligibility(ctx, users[0], a, offered.ID, staff.ID, service.ServiceStaff, false); e != nil {
		t.Fatal(e)
	}

	input = in
	input.StartsAt = day.Add(15 * time.Hour)
	archived := create("archive-original", input, nil)
	if _, e := pool.Exec(ctx, "UPDATE services SET status='archived' WHERE id=$1", offered.ID); e != nil {
		t.Fatal(e)
	}
	move(archived.ID, "archive-move", day.Add(16*time.Hour), service.ErrBookingConflict)
	cancelBooking(archived.ID, "archive-cancel", nil)
	if _, e := pool.Exec(ctx, "UPDATE services SET status='active' WHERE id=$1", offered.ID); e != nil {
		t.Fatal(e)
	}
	// Membership is checked even on successful-operation retries.
	for _, role := range []service.Role{service.RoleManager, service.RoleStaff} {
		if e := tenants.ChangeMember(ctx, users[0], a, users[2], role, false); e != nil {
			t.Fatal(e)
		}
		v, e := bookings.Cancel(ctx, users[2], a, second.ID, "denied")
		checkError(v, e, service.ErrForbidden)
	}
	v, e := bookings.Cancel(ctx, users[1], a, second.ID, "outsider")
	checkError(v, e, service.ErrForbidden)
	v, e = bookings.Cancel(ctx, users[1], b, second.ID, "foreign")
	checkError(v, e, service.ErrBookingNotFound)
	move(second.ID, "same-time", otherInput.StartsAt, service.ErrInvalidBooking)
	move(second.ID, "past", day.Add(-time.Hour), service.ErrBookingConflict)
	move(second.ID, "off-grid", day.Add(15*time.Hour+time.Minute), service.ErrBookingConflict)

	// Raw SQL cannot bypass terminal transitions, the cutoff, or replacement linking.
	stamp := func(v time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: v, Valid: true} }
	now := time.Now().UTC().Truncate(time.Minute)
	raw, e := db.New(pool).CreateBooking(ctx, db.CreateBookingParams{BusinessID: a, ServiceID: offered.ID, LocationID: loc.ID, CreatedBy: users[0], IdempotencyKey: "started-buffer", RequestHash: strings.Repeat("0", 64), StartsAt: stamp(now.Add(30 * time.Minute)), EndsAt: stamp(now.Add(time.Hour)), BusyStartsAt: stamp(now.Add(-time.Minute)), BusyEndsAt: stamp(now.Add(time.Hour))})
	if e != nil {
		t.Fatal(e)
	}
	cancelBooking(raw.ID, "too-late", service.ErrBookingState)
	move(raw.ID, "too-late-move", day.Add(16*time.Hour), service.ErrBookingState)
	_, e = db.New(pool).CloseBooking(ctx, db.CloseBookingParams{BusinessID: a, ID: raw.ID, Status: "cancelled", ClosedBy: pgtype.Int8{Int64: users[0], Valid: true}})
	var pg *pgconn.PgError
	if !errors.As(e, &pg) || pg.ConstraintName != "bookings_lifecycle" {
		t.Fatal("direct cutoff bypass", e)
	}
	_, e = pool.Exec(ctx, "UPDATE bookings SET status='confirmed',closed_at=NULL,closed_by=NULL WHERE id=$1", replacement.ID)
	if !errors.As(e, &pg) || pg.ConstraintName != "bookings_lifecycle" {
		t.Fatal("reactivation bypass", e)
	}
	_, e = pool.Exec(ctx, "UPDATE bookings SET starts_at=starts_at+interval '1 minute' WHERE id=$1", second.ID)
	if e == nil {
		t.Fatal("in-place reschedule bypass")
	}
	tx, e := pool.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(context.Background())
	_, e = db.New(tx).CloseBooking(ctx, db.CloseBookingParams{BusinessID: a, ID: second.ID, Status: "rescheduled", ClosedBy: pgtype.Int8{Int64: users[0], Valid: true}})
	if e != nil {
		t.Fatal(e)
	}
	e = tx.Commit(ctx)
	if !errors.As(e, &pg) || pg.ConstraintName != "bookings_lifecycle" {
		t.Fatal("orphan reschedule committed", e)
	}
	if lookup(second.ID).Status != "confirmed" {
		t.Fatal("orphan reschedule not rolled back")
	}
	// A direct insert cannot claim terminal status to evade occupancy.
	_, e = pool.Exec(ctx, `INSERT INTO bookings(business_id,service_id,location_id,created_by,idempotency_key,request_hash,starts_at,ends_at,busy_starts_at,busy_ends_at,status,closed_by,closed_at)
 VALUES($1,$2,$3,$4,'raw-terminal',$5,$6,$7,$6,$7,'cancelled',$4,now())`, a, offered.ID, loc.ID, users[0], strings.Repeat("0", 64), day, day.Add(time.Hour))
	if e == nil {
		t.Fatal("terminal insert bypass")
	}

	h, e := NewBookingHandler(sessions, bookings, "http://localhost:3000")
	if e != nil {
		t.Fatal(e)
	}
	mux := http.NewServeMux()
	RegisterBookingRoutes(mux, h)
	uuid := func(id pgtype.UUID) string { v, _ := id.MarshalJSON(); return strings.Trim(string(v), "\"") }
	base := "/api/v1/businesses/" + uuid(a) + "/bookings/"
	request := func(actor int, method, path, body, key, origin string, want int) *httptest.ResponseRecorder {
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
			t.Fatalf("%s %s: got %d want %d: %s", method, path, w.Code, want, w.Body.String())
		}
		return w
	}
	const origin = "http://localhost:3000"
	path := base + uuid(second.ID) + "/cancel"
	request(-1, "OPTIONS", path, "", "", origin, 204)
	request(-1, "POST", path, "{}", "http", origin, 401)
	request(0, "POST", path, "{}", "http", "", 403)
	request(0, "POST", path, "{}", "http", "https://foreign.invalid", 403)
	request(0, "GET", path, "", "", origin, 405)
	request(0, "POST", path, "{}", "", origin, 400)
	request(1, "POST", path, "{}", "http", origin, 403)
	request(2, "POST", path, "{}", "http", origin, 403)
	for _, body := range []string{"null", "[]", "", `{"status":"cancelled"}`, `{} {}`} {
		request(0, "POST", path, body, "http", origin, 400)
	}
	request(0, "POST", base+"invalid/cancel", "{}", "http", origin, 400)
	request(0, "POST", base+uuid(second.ID)+"/unknown", "{}", "http", origin, 404)
	request(0, "POST", base+"00000000-0000-0000-0000-000000000000/cancel", "{}", "http", origin, 404)
	// Admin may manage an owner's booking; retries need current access.
	if e := tenants.ChangeMember(ctx, users[0], a, users[2], service.RoleBusinessAdmin, false); e != nil {
		t.Fatal(e)
	}
	data, _ := json.Marshal(map[string]time.Time{"starts_at": day.Add(16 * time.Hour)})
	movePath := base + uuid(second.ID) + "/reschedule"
	request(2, "POST", movePath, `{}`, "http", origin, 400)
	request(2, "POST", movePath, `{"starts_at":"bad"}`, "http", origin, 400)
	result := request(2, "POST", movePath, string(data), "http", origin, 200)
	var httpResult service.BookingChangeResult
	if e := json.Unmarshal(result.Body.Bytes(), &httpResult); e != nil || httpResult.Replacement == nil {
		t.Fatal(httpResult, e)
	}
	request(2, "POST", movePath, string(data), "http", origin, 200)
	request(2, "POST", path, "{}", "http", origin, 409) // Action-bound idempotency key.
	path = base + uuid(httpResult.Replacement.ID) + "/cancel"
	request(2, "POST", path, "{}", "http-cancel", origin, 200)
	if e := tenants.ChangeMember(ctx, users[0], a, users[2], service.RoleBusinessAdmin, true); e != nil {
		t.Fatal(e)
	}
	request(2, "POST", path, "{}", "http-cancel", origin, 403)
	if _, e := pool.Exec(ctx, "UPDATE users SET status='disabled' WHERE id=$1", users[0]); e != nil {
		t.Fatal(e)
	}
	request(0, "POST", base+uuid(replacement.ID)+"/cancel", "{}", "cancel", origin, 401)
}
