package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/repository"
	"multipurpose-booking-system/api/internal/service"
)

func verifyCustomerBookingCreate(t *testing.T, f publicAvailabilityFixture, mux *http.ServeMux, users []int64, tokens []string) {
	t.Helper()
	ctx := f.ctx
	defer func() {
		if _, e := f.pool.Exec(ctx, "DELETE FROM bookings WHERE business_id=$1 AND idempotency_key LIKE '~customer:%'", f.business); e != nil {
			t.Error(e)
		}
	}()
	input := func(hour int) service.CustomerBookingInput {
		return service.CustomerBookingInput{BusinessID: f.business, ServiceID: f.serviceID, LocationID: f.location, StartsAt: f.day.Add(time.Duration(hour) * time.Hour), ExpectedEndsAt: f.day.Add(time.Duration(hour)*time.Hour + 30*time.Minute), ExpectedTimezone: "UTC"}
	}
	body := func(in service.CustomerBookingInput) string { b, _ := json.Marshal(in); return string(b) }
	send := func(actor int, key, payload, origin string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/api/v1/customer/bookings", strings.NewReader(payload)).WithContext(ctx)
		r.Header.Set("Content-Type", "application/json")
		if key != "" {
			r.Header.Set("Idempotency-Key", key)
		}
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		r.Header.Set("X-User-ID", fmt.Sprint(users[0]))
		r.Header.Set("X-Role", "BUSINESS_OWNER")
		if actor >= 0 {
			r.AddCookie(&http.Cookie{Name: "booking_session", Value: tokens[actor]})
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	check := func(w *httptest.ResponseRecorder, want int) {
		t.Helper()
		if w.Code != want {
			t.Fatalf("got %d want %d: %s", w.Code, want, w.Body.String())
		}
	}
	origin := "http://localhost:3000"
	check(send(-1, "anon", body(input(9)), origin), 401)
	check(send(2, "origin", body(input(9)), ""), 403)
	check(send(2, "origin", body(input(9)), "https://foreign.invalid"), 403)
	check(send(2, "", body(input(9)), origin), 400)
	check(send(2, "bad:key", body(input(9)), origin), 400)
	for _, extra := range []string{`"customer_id":1`, `"created_by":1`, `"staff_id":null`, `"resource_id":null`, `"ends_at":"2099-01-01T00:00:00Z"`} {
		check(send(2, "extra", strings.TrimSuffix(body(input(9)), "}")+","+extra+"}", origin), 400)
	}
	check(send(2, "empty", "{}", origin), 400)
	check(send(2, "null", "null", origin), 400)
	malformed := input(9)
	malformed.StartsAt = malformed.StartsAt.Add(time.Second)
	check(send(2, "second", body(malformed), origin), 400)
	malformed = input(9)
	malformed.LocationID = f.foreign
	check(send(2, "foreign-location", body(malformed), origin), 404)
	malformed = input(9)
	malformed.BusinessID = f.foreign
	check(send(2, "foreign-business", body(malformed), origin), 404)
	check(send(2, "occupied", body(input(10)), origin), 409)
	check(send(2, "outside", body(input(8)), origin), 409)
	f.request(2, "POST", "/api/v1/customer/bookings?customer_id=1", body(input(9)), origin, 400)
	f.request(-1, "OPTIONS", "/api/v1/customer/bookings", "", origin, 204)
	f.request(2, "DELETE", "/api/v1/customer/bookings", "", origin, 405)

	// Availability shown earlier is never trusted at submission time.
	schedule := service.LocationScheduleInput{WeeklyHours: []service.WeeklyHours{{Weekday: int32(f.day.Weekday()), StartMinute: 540, EndMinute: 1020}}, ClosedDates: []string{f.day.Format("2006-01-02")}}
	if _, e := f.catalog.PutLocationSchedule(ctx, f.actor, f.business, f.location, schedule); e != nil {
		t.Fatal(e)
	}
	check(send(2, "closed", body(input(9)), origin), 409)
	schedule.ClosedDates = []string{}
	if _, e := f.catalog.PutLocationSchedule(ctx, f.actor, f.business, f.location, schedule); e != nil {
		t.Fatal(e)
	}
	if _, e := f.pool.Exec(ctx, "UPDATE services SET duration_minutes=45 WHERE id=$1", f.serviceID); e != nil {
		t.Fatal(e)
	}
	check(send(2, "duration", body(input(9)), origin), 409)
	if _, e := f.pool.Exec(ctx, "UPDATE services SET duration_minutes=30 WHERE id=$1", f.serviceID); e != nil {
		t.Fatal(e)
	}
	if _, e := f.pool.Exec(ctx, "UPDATE staff_profiles SET status='archived' WHERE id=$1", f.staff); e != nil {
		t.Fatal(e)
	}
	check(send(2, "archived-staff", body(input(9)), origin), 409)
	if _, e := f.pool.Exec(ctx, "UPDATE staff_profiles SET status='active' WHERE id=$1", f.staff); e != nil {
		t.Fatal(e)
	}
	if e := f.catalog.ChangeEligibility(ctx, f.actor, f.business, f.serviceID, f.location, service.ServiceLocation, true); e != nil {
		t.Fatal(e)
	}
	check(send(2, "unlinked", body(input(9)), origin), 404)
	if e := f.catalog.ChangeEligibility(ctx, f.actor, f.business, f.serviceID, f.location, service.ServiceLocation, false); e != nil {
		t.Fatal(e)
	}
	if _, e := f.catalog.PutSchedulingRules(ctx, f.actor, f.business, f.serviceID, service.SchedulingRulesInput{MinimumLeadMinutes: 525600}); e != nil {
		t.Fatal(e)
	}
	check(send(2, "lead", body(input(9)), origin), 409)
	if _, e := f.catalog.PutSchedulingRules(ctx, f.actor, f.business, f.serviceID, service.SchedulingRulesInput{}); e != nil {
		t.Fatal(e)
	}
	if _, e := f.tenants.CreateCustomerBooking(ctx, -1, "inactive", input(9)); !errors.Is(e, service.ErrForbidden) {
		t.Fatal("invalid actor accepted", e)
	}

	if _, e := f.pool.Exec(ctx, "UPDATE users SET status='disabled' WHERE id=$1", users[2]); e != nil {
		t.Fatal(e)
	}
	if _, e := f.tenants.CreateCustomerBooking(ctx, users[2], "disabled", input(9)); !errors.Is(e, service.ErrForbidden) {
		t.Fatal("disabled customer accepted", e)
	}
	if _, e := f.pool.Exec(ctx, "UPDATE users SET status='active' WHERE id=$1", users[2]); e != nil {
		t.Fatal(e)
	}

	// Even after a new publication, stale client duration/timezone expectations cannot be booked.
	stale := input(9)
	stale.ExpectedEndsAt = stale.ExpectedEndsAt.Add(15 * time.Minute)
	check(send(2, "stale-duration", body(stale), origin), 409)
	stale = input(9)
	stale.ExpectedTimezone = "Asia/Shanghai"
	check(send(2, "stale-zone", body(stale), origin), 409)
	w := send(2, "customer-one", body(input(9)), origin)
	check(w, 201)
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("cacheable response")
	}
	for _, private := range []string{"staff_id", "resource_id", "customer_id", "created_by", "request_hash", "idempotency_key"} {
		if strings.Contains(w.Body.String(), private) {
			t.Fatal("private projection", private)
		}
	}
	var created service.CustomerBookingCreated
	if e := json.Unmarshal(w.Body.Bytes(), &created); e != nil {
		t.Fatal(e)
	}
	if created.Replayed || created.Status != "confirmed" || !created.EndsAt.Equal(input(9).StartsAt.Add(30*time.Minute)) {
		t.Fatal("invalid confirmation", created)
	}
	var customer, creator int64
	var assigned pgtype.UUID
	if e := f.pool.QueryRow(ctx, "SELECT customer_id,created_by,staff_id FROM bookings WHERE id=$1", created.ID).Scan(&customer, &creator, &assigned); e != nil {
		t.Fatal(e)
	}
	if customer != users[2] || creator != users[2] || assigned != f.staff {
		t.Fatal("incorrect ownership/allocation")
	}
	if _, e := f.tenants.CustomerBooking(ctx, users[3], created.ID); !errors.Is(e, service.ErrBookingNotFound) {
		t.Fatal("foreign customer read", e)
	}
	if _, e := f.tenants.CustomerBooking(ctx, users[2], created.ID); e != nil {
		t.Fatal(e)
	}
	if _, e := f.bookings.Get(ctx, users[0], f.business, created.ID); e != nil {
		t.Fatal("not visible to business", e)
	}
	if _, e := f.bookings.Get(ctx, users[1], f.foreign, created.ID); !errors.Is(e, service.ErrBookingNotFound) {
		t.Fatal("cross business read", e)
	}
	equivalent := input(9)
	equivalent.StartsAt = equivalent.StartsAt.In(time.FixedZone("offset", 8*3600))
	replay := send(2, "customer-one", body(equivalent), origin)
	check(replay, 200)
	var repeated service.CustomerBookingCreated
	_ = json.Unmarshal(replay.Body.Bytes(), &repeated)
	if repeated.ID != created.ID || !repeated.Replayed {
		t.Fatal("replay mismatch")
	}
	check(send(2, "customer-one", body(input(13)), origin), 409)
	check(send(3, "customer-one", body(input(9)), origin), 409)
	// Replays survive unpublishing; new requests do not.
	if _, e := f.pool.Exec(ctx, "UPDATE businesses SET is_published=false WHERE id=$1", f.business); e != nil {
		t.Fatal(e)
	}
	check(send(2, "customer-one", body(input(9)), origin), 200)
	check(send(2, "unpublished", body(input(13)), origin), 404)
	if _, e := f.pool.Exec(ctx, "UPDATE businesses SET is_published=true WHERE id=$1", f.business); e != nil {
		t.Fatal(e)
	}
	// Lost-response retries report current lifecycle state, not a false confirmation.
	if _, e := f.bookings.Cancel(ctx, users[0], f.business, created.ID, "cancel-customer-fixture"); e != nil {
		t.Fatal(e)
	}
	replay = send(2, "customer-one", body(input(9)), origin)
	check(replay, 200)
	_ = json.Unmarshal(replay.Body.Bytes(), &repeated)
	if repeated.Status != "cancelled" {
		t.Fatal("replay hid cancellation")
	}
	// Two concurrent customers cannot acquire the same exclusive staff slot.
	var wg sync.WaitGroup
	results := make([]*httptest.ResponseRecorder, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i] = send(2+i, fmt.Sprintf("race-%d", i), body(input(13)), origin)
		}(i)
	}
	wg.Wait()
	successes, conflicts := 0, 0
	for _, r := range results {
		if r.Code == 201 {
			successes++
		} else if r.Code == 409 {
			conflicts++
		} else {
			t.Fatalf("race status %d: %s", r.Code, r.Body.String())
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatal("overbooked exclusive slot", successes, conflicts)
	}
	// Concurrent retries of one request both resolve to exactly one saved booking.
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) { defer wg.Done(); results[i] = send(2, "same-race", body(input(15)), origin) }(i)
	}
	wg.Wait()
	var firstID pgtype.UUID
	for i, r := range results {
		if r.Code != 200 && r.Code != 201 {
			t.Fatalf("retry race %d: %s", r.Code, r.Body.String())
		}
		var v service.CustomerBookingCreated
		_ = json.Unmarshal(r.Body.Bytes(), &v)
		if i == 0 {
			firstID = v.ID
		} else if firstID != v.ID {
			t.Fatal("duplicate replay")
		}
	}
	// Reusing an operator key cannot replay an internal booking as a customer booking.
	if _, e := f.bookings.Create(ctx, users[0], f.business, "shared-key", service.BookingInput{ServiceID: f.serviceID, LocationID: f.location, StaffID: f.staff, StartsAt: input(16).StartsAt}); e != nil {
		t.Fatal(e)
	}
	check(send(0, "shared-key", body(input(16)), origin), 409)
	// A failed transaction must not expose a confirmation or leave a booking behind.
	failing := service.NewBusinessService(settingsRollback{repository.NewBusinessTransactions(f.pool)})
	failed, e := failing.CreateCustomerBooking(ctx, users[2], "rollback", input(14))
	if e == nil || failed.ID.Valid {
		t.Fatal("failed commit exposed receipt", failed, e)
	}
	var count int
	if e := f.pool.QueryRow(ctx, "SELECT count(*) FROM bookings WHERE business_id=$1 AND starts_at=$2", f.business, input(14).StartsAt).Scan(&count); e != nil || count != 0 {
		t.Fatal("rollback left booking", count, e)
	}
	// Restore fixture-only mutations for subsequent availability tests.
	if _, e := f.pool.Exec(ctx, "DELETE FROM booking_changes WHERE business_id=$1 AND idempotency_key='cancel-customer-fixture'", f.business); e != nil {
		t.Fatal(e)
	}
	if _, e := f.pool.Exec(ctx, "DELETE FROM bookings WHERE business_id=$1 AND idempotency_key='shared-key'", f.business); e != nil {
		t.Fatal(e)
	}
	// Resource capacity must still be enforced when multiple staff can serve a time.
	second, e := f.catalog.CreateStaffProfile(ctx, f.actor, f.business, service.StaffProfileInput{Name: "Customer capacity alternate"})
	if e != nil {
		t.Fatal(e)
	}
	resource, e := f.catalog.CreateResource(ctx, f.actor, f.business, service.ResourceInput{Name: "Customer capacity resource", LocationID: f.location, Capacity: 1})
	if e != nil {
		t.Fatal(e)
	}
	for _, link := range []struct {
		parent, target pgtype.UUID
		kind           service.EligibilityKind
	}{{f.serviceID, second.ID, service.ServiceStaff}, {second.ID, f.location, service.StaffLocation}, {f.serviceID, resource.ID, service.ServiceResource}} {
		if e := f.catalog.ChangeEligibility(ctx, f.actor, f.business, link.parent, link.target, link.kind, false); e != nil {
			t.Fatal(e)
		}
	}
	window := []service.AvailabilityWindow{{StartsAt: f.day.Add(9 * time.Hour), EndsAt: f.day.Add(17 * time.Hour)}}
	if _, e := f.catalog.PutAvailabilityWindows(ctx, f.actor, f.business, second.ID, "staff", window); e != nil {
		t.Fatal(e)
	}
	if _, e := f.catalog.PutAvailabilityWindows(ctx, f.actor, f.business, resource.ID, "resources", window); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i] = send(2+i, fmt.Sprintf("resource-race-%d", i), body(input(9)), origin)
		}(i)
	}
	wg.Wait()
	successes, conflicts = 0, 0
	for _, r := range results {
		if r.Code == 201 {
			successes++
		} else if r.Code == 409 {
			conflicts++
		} else {
			t.Fatalf("resource race %d: %s", r.Code, r.Body.String())
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatal("resource overbooked", successes, conflicts)
	}
	if _, e := f.pool.Exec(ctx, "DELETE FROM bookings WHERE business_id=$1 AND idempotency_key LIKE '~customer:%'", f.business); e != nil {
		t.Fatal(e)
	}
	for _, item := range []struct {
		query string
		id    pgtype.UUID
	}{
		{"DELETE FROM staff_availability_windows WHERE staff_id=$1", second.ID},
		{"DELETE FROM service_staff WHERE staff_id=$1", second.ID},
		{"DELETE FROM staff_locations WHERE staff_id=$1", second.ID},
		{"DELETE FROM staff_profiles WHERE id=$1", second.ID},
		{"DELETE FROM resource_availability_windows WHERE resource_id=$1", resource.ID},
		{"DELETE FROM service_resources WHERE resource_id=$1", resource.ID},
		{"DELETE FROM resources WHERE id=$1", resource.ID},
	} {
		if _, e := f.pool.Exec(ctx, item.query, item.id); e != nil {
			t.Fatal(e)
		}
	}

}
