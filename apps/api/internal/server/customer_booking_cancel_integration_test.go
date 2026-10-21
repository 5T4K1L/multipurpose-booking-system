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

func verifyCustomerCancellation(t *testing.T, f publicAvailabilityFixture, mux *http.ServeMux, users []int64, tokens []string) {
	t.Helper()
	ctx := f.ctx
	uuid := func(id pgtype.UUID) string { v, _ := id.MarshalJSON(); return string(v[1 : len(v)-1]) }
	offered, e := f.catalog.CreateService(ctx, f.actor, f.business, service.ServiceInput{Name: "Cancellation fixture", DurationMinutes: 30})
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
	loc, e := f.catalog.CreateLocation(ctx, f.actor, f.business, service.LocationInput{Name: "Cancellation location", Timezone: "UTC"})
	if e != nil {
		t.Fatal(e)
	}
	room, e := f.catalog.CreateResource(ctx, f.actor, f.business, service.ResourceInput{Name: "Cancellation room", LocationID: loc.ID, Capacity: 1})
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
	owned := seed(users[2], start, "cancel-owned")
	other := seed(users[3], start.Add(time.Hour), "cancel-other")
	legacy := seed(nil, start.Add(2*time.Hour), "cancel-legacy")
	send := func(actor int, id pgtype.UUID, key, body, origin string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/api/v1/customer/bookings/"+uuid(id)+"/cancel", strings.NewReader(body)).WithContext(ctx)
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
	for _, actor := range []int{0, 1, 3} {
		check(send(actor, owned, "forged", "{}", origin), 404)
	}
	check(send(-1, owned, "anonymous", "{}", origin), 401)
	for _, id := range []pgtype.UUID{other, legacy, {Valid: true}} {
		check(send(2, id, "not-owned", "{}", origin), 404)
	}
	check(send(2, owned, "", "{}", origin), 400)
	check(send(2, owned, "bad key", "{}", origin), 400)
	check(send(2, owned, "body", `{"customer_id":1}`, origin), 400)
	check(send(2, owned, "origin", "{}", "https://foreign.invalid"), 403)
	check(send(2, owned, "origin", "{}", ""), 403)
	var detail service.CustomerBookingDetail
	if e := json.Unmarshal(f.request(2, "GET", "/api/v1/customer/bookings/"+uuid(owned), "", "", 200).Body.Bytes(), &detail); e != nil {
		t.Fatal(e)
	}
	if !detail.CanCancel || !detail.CancelBefore.Equal(start.Add(-15*time.Minute)) {
		t.Fatal("wrong cancellation deadline", detail)
	}
	available := func() bool {
		t.Helper()
		r, e := f.catalog.CalculateSlots(ctx, f.actor, f.business, offered.ID, service.SlotRequest{Date: f.day.Format("2006-01-02"), LocationID: loc.ID, ResourceID: room.ID})
		if e != nil {
			t.Fatal(e)
		}
		for _, s := range r.Slots {
			if s.StartsAt.Equal(start) {
				return true
			}
		}
		return false
	}
	if available() {
		t.Fatal("occupied slot offered")
	}
	failing := service.NewBusinessService(settingsRollback{repository.NewBusinessTransactions(f.pool)})
	failed, e := failing.CancelCustomerBooking(ctx, users[2], owned, "rollback-cancel")
	if !errors.Is(e, service.ErrBookingUnavailable) || failed.ID.Valid {
		t.Fatal("rollback exposed success", failed, e)
	}
	persisted, e := f.tenants.CustomerBooking(ctx, users[2], owned)
	if e != nil || persisted.Status != "confirmed" || available() {
		t.Fatal("failed cancellation changed booking", persisted, e)
	}
	// Duplicate clicks serialize and produce exactly one cancellation audit.
	responses := make([]*httptest.ResponseRecorder, 2)
	var wg sync.WaitGroup
	for i := range responses {
		wg.Add(1)
		go func(i int) { defer wg.Done(); responses[i] = send(2, owned, "same-cancel", "{}", origin) }(i)
	}
	wg.Wait()
	replayCount := 0
	for _, w := range responses {
		check(w, 200)
		var result service.CustomerCancellation
		if e := json.Unmarshal(w.Body.Bytes(), &result); e != nil {
			t.Fatal(e)
		}
		if result.Status != "cancelled" {
			t.Fatal(result)
		}
		if result.Replayed {
			replayCount++
		}
		for _, private := range []string{"customer_id", "staff_id", "resource_id", "request_hash", "closed_by"} {
			if strings.Contains(w.Body.String(), private) {
				t.Fatal("private cancellation field", private)
			}
		}
	}
	if replayCount != 1 {
		t.Fatal("retry did not replay")
	}
	if !available() {
		t.Fatal("cancellation did not release capacity")
	}
	var count int
	if e := f.pool.QueryRow(ctx, "SELECT count(*) FROM booking_changes WHERE booking_id=$1", owned).Scan(&count); e != nil || count != 1 {
		t.Fatal("duplicate audit", count, e)
	}
	check(send(2, owned, "new-cancel", "{}", origin), 200)
	state, e := f.bookings.Get(ctx, f.actor, f.business, owned)
	if e != nil || state.Status != "cancelled" {
		t.Fatal("business did not see cancellation", state, e)
	}
	// Same key cannot be repurposed for a different owned booking.
	another := seed(users[2], start.Add(3*time.Hour), "cancel-another")
	check(send(2, another, "same-cancel", "{}", origin), 409)
	past := seed(users[2], time.Now().UTC().Truncate(time.Minute).Add(-2*time.Hour), "cancel-past")
	check(send(2, past, "too-late", "{}", origin), 409)
	prep := seed(users[2], time.Now().UTC().Truncate(time.Minute).Add(5*time.Minute), "cancel-preparation")
	check(send(2, prep, "preparation-started", "{}", origin), 409)
	if _, e := f.bookings.Complete(ctx, f.actor, f.business, past, "complete-cancel-fixture"); e != nil {
		t.Fatal(e)
	}
	check(send(2, past, "completed", "{}", origin), 409)
	moved, e := f.bookings.Reschedule(ctx, f.actor, f.business, another, "move-cancel-fixture", start.Add(4*time.Hour))
	if e != nil || moved.Replacement == nil {
		t.Fatal(e)
	}
	check(send(2, another, "rescheduled", "{}", origin), 409)
	// Historical customer control survives publication/catalog/business disablement.
	if _, e := f.pool.Exec(ctx, "UPDATE businesses SET status='disabled' WHERE id=$1", f.business); e != nil {
		t.Fatal(e)
	}
	check(send(2, moved.Replacement.ID, "disabled-business", "{}", origin), 200)
	check(send(2, owned, "same-cancel", "{}", origin), 200)
}
