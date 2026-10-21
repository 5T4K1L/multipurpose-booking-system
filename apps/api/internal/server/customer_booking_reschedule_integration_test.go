package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/repository"
	"multipurpose-booking-system/api/internal/service"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func verifyCustomerRescheduling(t *testing.T, f publicAvailabilityFixture, mux *http.ServeMux, users []int64, tokens []string) {
	t.Helper()
	ctx := f.ctx
	uuid := func(id pgtype.UUID) string { v, _ := id.MarshalJSON(); return string(v[1 : len(v)-1]) }
	offered, e := f.catalog.CreateService(ctx, f.actor, f.business, service.ServiceInput{Name: "Reschedule fixture", DurationMinutes: 30})
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		_, e := f.pool.Exec(ctx, "DELETE FROM booking_changes WHERE business_id=$1 AND booking_id IN(SELECT id FROM bookings WHERE service_id=$2)", f.business, offered.ID)
		if e != nil {
			t.Error(e)
		}
		_, e = f.pool.Exec(ctx, "DELETE FROM bookings WHERE business_id=$1 AND service_id=$2", f.business, offered.ID)
		if e != nil {
			t.Error(e)
		}
		_, e = f.pool.Exec(ctx, "UPDATE businesses SET status='active' WHERE id=$1", f.business)
		if e != nil {
			t.Error(e)
		}
	}()
	loc, e := f.catalog.CreateLocation(ctx, f.actor, f.business, service.LocationInput{Name: "Reschedule location", Timezone: "UTC"})
	if e != nil {
		t.Fatal(e)
	}
	room, e := f.catalog.CreateResource(ctx, f.actor, f.business, service.ResourceInput{Name: "Reschedule room", LocationID: loc.ID, Capacity: 1})
	if e != nil {
		t.Fatal(e)
	}
	for _, link := range []struct {
		id   pgtype.UUID
		kind service.EligibilityKind
	}{{loc.ID, service.ServiceLocation}, {room.ID, service.ServiceResource}} {
		if e := f.catalog.ChangeEligibility(ctx, f.actor, f.business, offered.ID, link.id, link.kind, false); e != nil {
			t.Fatal(e)
		}
	}
	if _, e := f.catalog.PutSchedulingRules(ctx, f.actor, f.business, offered.ID, service.SchedulingRulesInput{BufferBeforeMinutes: 15}); e != nil {
		t.Fatal(e)
	}
	if _, e := f.catalog.PutLocationSchedule(ctx, f.actor, f.business, loc.ID, service.LocationScheduleInput{WeeklyHours: []service.WeeklyHours{{Weekday: int32(f.day.Weekday()), StartMinute: 0, EndMinute: 1440}}, ClosedDates: []string{}}); e != nil {
		t.Fatal(e)
	}
	if _, e := f.catalog.PutAvailabilityWindows(ctx, f.actor, f.business, room.ID, "resources", []service.AvailabilityWindow{{StartsAt: f.day, EndsAt: f.day.Add(24 * time.Hour)}}); e != nil {
		t.Fatal(e)
	}
	seed := func(owner any, start time.Time, key string) pgtype.UUID {
		t.Helper()
		var id pgtype.UUID
		e := f.pool.QueryRow(ctx, `INSERT INTO bookings(business_id,service_id,location_id,resource_id,created_by,customer_id,idempotency_key,request_hash,starts_at,ends_at,busy_starts_at,busy_ends_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$10) RETURNING id`, f.business, offered.ID, loc.ID, room.ID, f.actor, owner, key, strings.Repeat("a", 64), start, start.Add(30*time.Minute), start.Add(-15*time.Minute)).Scan(&id)
		if e != nil {
			t.Fatal(e)
		}
		return id
	}
	start := f.day.Add(9 * time.Hour)
	owned := seed(users[2], start, "move-owned")
	other := seed(users[3], start.Add(time.Hour), "move-other")
	legacy := seed(nil, start.Add(2*time.Hour), "move-legacy")
	send := func(actor int, id pgtype.UUID, key, body, origin string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/api/v1/customer/bookings/"+uuid(id)+"/reschedule", strings.NewReader(body)).WithContext(ctx)
		r.Header.Set("Content-Type", "application/json")
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		if key != "" {
			r.Header.Set("Idempotency-Key", key)
		}
		r.Header.Set("X-User-ID", fmt.Sprint(users[2]))
		r.Header.Set("X-Role", "BUSINESS_OWNER")
		if actor >= 0 {
			r.AddCookie(&http.Cookie{Name: "booking_session", Value: tokens[actor]})
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	check := func(w *httptest.ResponseRecorder, status int) {
		t.Helper()
		if w.Code != status {
			t.Fatalf("got %d want %d: %s", w.Code, status, w.Body.String())
		}
	}

	origin := "http://localhost:3000"
	body := func(start time.Time) string { return fmt.Sprintf(`{"starts_at":%q}`, start.Format(time.RFC3339)) }
	target := start.Add(4 * time.Hour)
	for _, actor := range []int{0, 1, 3} {
		check(send(actor, owned, "forged", body(target), origin), 404)
	}
	check(send(-1, owned, "anon", body(target), origin), 401)
	for _, id := range []pgtype.UUID{other, legacy, {Valid: true}} {
		check(send(2, id, "other", body(target), origin), 404)
	}
	check(send(2, owned, "", body(target), origin), 400)
	check(send(2, owned, "invalid", `{"starts_at":"bad"}`, origin), 400)
	check(send(2, owned, "extra", `{"starts_at":"2026-10-19T10:00:00Z","customer_id":1}`, origin), 400)
	check(send(2, owned, "origin", body(target), "https://foreign.invalid"), 403)
	check(send(2, owned, "same-time", body(start), origin), 400)
	check(send(2, owned, "occupied", body(start.Add(time.Hour)), origin), 409)
	saved, e := f.tenants.CustomerBooking(ctx, users[2], owned)
	if e != nil || saved.Status != "confirmed" {
		t.Fatal("conflict changed original", saved, e)
	}
	// Audit failure rolls back both release and replacement.
	fault := service.NewBusinessService(settingsRollback{repository.NewBusinessTransactions(f.pool)})
	if _, e := fault.RescheduleCustomerBooking(ctx, users[2], owned, "rollback", target); !errors.Is(e, service.ErrBookingUnavailable) {
		t.Fatal("rollback", e)
	}
	saved, e = f.tenants.CustomerBooking(ctx, users[2], owned)
	if e != nil || saved.Status != "confirmed" {
		t.Fatal("rollback lost original", e)
	}
	// Advisory availability excludes only the owned original, not other bookings.
	slots, e := f.tenants.CustomerRescheduleSlots(ctx, users[2], owned, f.day.Format("2006-01-02"))
	if e != nil {
		t.Fatal(e)
	}
	overlapFound := false
	for _, slot := range slots.Slots {
		if slot.StartsAt.Equal(start.Add(15 * time.Minute)) {
			overlapFound = true
		}
		if slot.StartsAt.Equal(start.Add(time.Hour)) {
			t.Fatal("occupied advisory time")
		}
	}
	if !overlapFound {
		t.Fatal("own reservation blocked overlapping move")
	}
	f.request(3, "GET", "/api/v1/customer/bookings/"+uuid(owned)+"/reschedule?date="+f.day.Format("2006-01-02"), "", "", 404)
	f.request(2, "GET", "/api/v1/customer/bookings/"+uuid(owned)+"/reschedule?date="+f.day.Format("2006-01-02"), "", "", 200)
	f.request(2, "GET", "/api/v1/customer/bookings/"+uuid(owned)+"/reschedule?date=bad", "", "", 400)
	// A later catalog duration edit must not silently change an existing booking.
	if _, e := f.pool.Exec(ctx, "UPDATE services SET duration_minutes=60 WHERE id=$1", offered.ID); e != nil {
		t.Fatal(e)
	}
	target = start.Add(15 * time.Minute) // valid move overlapping the original reservation
	var wg sync.WaitGroup
	results := make(chan *httptest.ResponseRecorder, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); results <- send(2, owned, "same-move", body(target), origin) }()
	}
	wg.Wait()
	close(results)
	var result service.CustomerReschedule
	replays := 0
	for w := range results {
		check(w, 200)
		if e := json.Unmarshal(w.Body.Bytes(), &result); e != nil {
			t.Fatal(e)
		}
		if result.Replayed {
			replays++
		}
		for _, private := range []string{"staff_id", "resource_id", "customer_id", "request_hash"} {
			if strings.Contains(w.Body.String(), private) {
				t.Fatal("private field", private)
			}
		}
	}
	if replays != 1 || result.Booking.Status != "rescheduled" || !result.Replacement.StartsAt.Equal(target) || result.Replacement.EndsAt.Sub(result.Replacement.StartsAt) != 30*time.Minute {
		t.Fatal("wrong replacement", result)
	}
	check(send(2, owned, "same-move", body(target.Add(time.Hour)), origin), 409)
	check(send(2, owned, "fresh-move", body(target), origin), 409)
	replacement, e := f.tenants.CustomerBooking(ctx, users[2], result.Replacement.ID)
	if e != nil || replacement.SupersedesID != owned {
		t.Fatal("ownership/link lost", e)
	}
	if _, e := f.tenants.CustomerBooking(ctx, users[3], result.Replacement.ID); !errors.Is(e, service.ErrBookingNotFound) {
		t.Fatal("foreign replacement read", e)
	}
	state, e := f.bookings.Get(ctx, f.actor, f.business, owned)
	if e != nil || state.Status != "rescheduled" || state.ReplacementID != result.Replacement.ID {
		t.Fatal("business did not see update", e)
	}
	var count int
	if e := f.pool.QueryRow(ctx, "SELECT count(*) FROM booking_changes WHERE booking_id=$1", owned).Scan(&count); e != nil || count != 1 {
		t.Fatal("duplicate audit", count, e)
	}
	past := seed(users[2], time.Now().UTC().Truncate(time.Minute).Add(-2*time.Hour), "move-past")
	check(send(2, past, "past", body(target), origin), 409)
	cancelled := seed(users[2], start.Add(6*time.Hour), "move-cancelled")
	if _, e := f.tenants.CancelCustomerBooking(ctx, users[2], cancelled, "cancel-before-move"); e != nil {
		t.Fatal(e)
	}
	check(send(2, cancelled, "cancelled", body(target), origin), 409)
	prep := seed(users[2], time.Now().UTC().Truncate(time.Minute).Add(5*time.Minute), "move-prep")
	check(send(2, prep, "prep", body(target), origin), 409)
	// Disablement blocks new moves, but cannot invalidate a successful retry.
	if _, e := f.pool.Exec(ctx, "UPDATE businesses SET status='disabled' WHERE id=$1", f.business); e != nil {
		t.Fatal(e)
	}
	check(send(2, result.Replacement.ID, "disabled", body(start.Add(5*time.Hour)), origin), 409)
	check(send(2, owned, "same-move", body(target), origin), 200)
}
