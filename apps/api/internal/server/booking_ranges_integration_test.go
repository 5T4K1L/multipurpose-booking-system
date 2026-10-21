package server

import (
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5/pgtype"
	"multipurpose-booking-system/api/internal/service"
	"net/url"
	"strings"
	"testing"
	"time"
)

func verifyBookingRanges(t *testing.T, f publicAvailabilityFixture, customer int64) {
	t.Helper()
	ctx := f.ctx
	uuid := func(id pgtype.UUID) string { v, _ := id.MarshalJSON(); return string(v[1 : len(v)-1]) }
	loc, e := f.catalog.CreateLocation(ctx, f.actor, f.business, service.LocationInput{Name: "Range location", Timezone: "Asia/Shanghai"})
	if e != nil {
		t.Fatal(e)
	}
	offered, e := f.catalog.CreateService(ctx, f.actor, f.business, service.ServiceInput{Name: "Range service", DurationMinutes: 30})
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if _, e := f.pool.Exec(ctx, "DELETE FROM booking_changes WHERE business_id=$1 AND booking_id IN (SELECT id FROM bookings WHERE service_id=$2)", f.business, offered.ID); e != nil {
			t.Error(e)
		}
		if _, e := f.pool.Exec(ctx, "DELETE FROM bookings WHERE business_id=$1 AND service_id=$2", f.business, offered.ID); e != nil {
			t.Error(e)
		}
	}()
	resource, e := f.catalog.CreateResource(ctx, f.actor, f.business, service.ResourceInput{Name: "Range room", LocationID: loc.ID, Capacity: 1})
	if e != nil {
		t.Fatal(e)
	}
	for _, link := range []struct {
		target pgtype.UUID
		kind   service.EligibilityKind
	}{{loc.ID, service.ServiceLocation}, {resource.ID, service.ServiceResource}} {
		if e := f.catalog.ChangeEligibility(ctx, f.actor, f.business, offered.ID, link.target, link.kind, false); e != nil {
			t.Fatal(e)
		}
	}
	root := "/api/v1/businesses/" + uuid(f.business) + "/catalog/services/" + uuid(offered.ID)
	readiness := f.request(0, "GET", root+"/booking-readiness", "", "", 200)
	if !strings.Contains(readiness.Body.String(), "Follows location opening hours") || !strings.Contains(readiness.Body.String(), "No opening hours") {
		t.Fatal("Missing setup blockers", readiness.Body.String())
	}
	f.request(-1, "GET", root+"/booking-readiness", "", "", 401)
	f.request(0, "PUT", root+"/booking-options", `{"enabled":true,"minimum_minutes":60,"maximum_minutes":30}`, "http://localhost:3000", 400)
	f.request(0, "PUT", root+"/booking-options", `{"enabled":true,"minimum_minutes":60,"maximum_minutes":10081}`, "http://localhost:3000", 400)
	f.request(0, "PUT", strings.Replace(root, uuid(f.business), uuid(f.foreign), 1)+"/booking-options", `{"enabled":true,"minimum_minutes":60,"maximum_minutes":2880}`, "http://localhost:3000", 403)
	hours := []service.WeeklyHours{}
	for i := 0; i < 7; i++ {
		hours = append(hours, service.WeeklyHours{Weekday: int32(i), StartMinute: 0, EndMinute: 1440})
	}
	schedule := service.LocationScheduleInput{WeeklyHours: hours, ClosedDates: []string{}}
	if _, e := f.catalog.PutLocationSchedule(ctx, f.actor, f.business, loc.ID, schedule); e != nil {
		t.Fatal(e)
	}
	if _, e := f.catalog.PutSchedulingRules(ctx, f.actor, f.business, offered.ID, service.SchedulingRulesInput{}); e != nil {
		t.Fatal(e)
	}
	if _, e := f.catalog.PutAvailabilityWindows(ctx, f.actor, f.business, resource.ID, "resources", []service.AvailabilityWindow{{StartsAt: f.day, EndsAt: f.day.AddDate(0, 0, 6)}}); e != nil {
		t.Fatal(e)
	}
	publish := func() {
		t.Helper()
		path := "/api/v1/businesses/" + uuid(f.business) + "/publication"
		var review service.Publication
		if e := json.Unmarshal(f.request(0, "GET", path, "", "", 200).Body.Bytes(), &review); e != nil {
			t.Fatal(e)
		}
		body, _ := json.Marshal(map[string]any{"published": true, "review_token": review.ReviewToken})
		f.request(0, "PUT", path, string(body), "http://localhost:3000", 200)
	}
	publish()
	// UTC14:00 is 22:00 at this location; end UTC18:00 is next day's 02:00.
	start := f.day.Add(14 * time.Hour)
	end := start.Add(4 * time.Hour)
	path := "/api/v1/public/businesses/" + uuid(f.business) + "/services/" + uuid(offered.ID) + "/range"
	query := func(a, b time.Time) string {
		return "?" + url.Values{"location_id": {uuid(loc.ID)}, "starts_at": {a.Format(time.RFC3339)}, "ends_at": {b.Format(time.RFC3339)}}.Encode()
	}
	f.request(-1, "GET", path+query(start, end), "", "", 409) // fixed service cannot be lengthened
	f.request(0, "PUT", root+"/booking-options", `{"enabled":true,"minimum_minutes":60,"maximum_minutes":2880}`, "http://localhost:3000", 200)
	f.request(-1, "GET", path+query(start, end), "", "", 409) // changed policy needs reviewed publication
	publish()
	read := func(a, b time.Time, want int) {
		t.Helper()
		var result service.SlotResult
		if e := json.Unmarshal(f.request(-1, "GET", path+query(a, b), "", "", 200).Body.Bytes(), &result); e != nil {
			t.Fatal(e)
		}
		if len(result.Slots) != want || result.Timezone != "Asia/Shanghai" {
			t.Fatal("wrong range result", result)
		}
	}
	read(start, end, 1)
	f.request(-1, "GET", path+query(start, start.Add(30*time.Minute)), "", "", 400)
	f.request(-1, "GET", path+query(start, start.Add(49*time.Hour)), "", "", 400)
	f.request(-1, "GET", path+query(start, end)+"&ends_at=bad", "", "", 400)
	f.request(-1, "GET", path+query(start, end), "", "https://foreign.invalid", 403)
	in := service.CustomerBookingInput{BusinessID: f.business, ServiceID: offered.ID, LocationID: loc.ID, StartsAt: start, ExpectedEndsAt: end, ExpectedTimezone: "Asia/Shanghai"}
	created, e := f.tenants.CreateCustomerBooking(ctx, customer, "range-create", in)
	if e != nil {
		t.Fatal(e)
	}
	if !created.EndsAt.Equal(end) {
		t.Fatal("range truncated")
	}
	replay, e := f.tenants.CreateCustomerBooking(ctx, customer, "range-create", in)
	if e != nil || !replay.Replayed || replay.ID != created.ID {
		t.Fatal("range retry failed", replay, e)
	}
	read(start, end, 0)
	overlap := in
	overlap.StartsAt = start.Add(-time.Hour)
	overlap.ExpectedEndsAt = end.Add(time.Hour)
	if _, e := f.tenants.CreateCustomerBooking(ctx, customer, "range-overlap", overlap); !errors.Is(e, service.ErrBookingConflict) {
		t.Fatal("overlap allowed", e)
	}
	moved, e := f.bookings.Reschedule(ctx, f.actor, f.business, created.ID, "range-move", start.AddDate(0, 0, 1))
	if e != nil || moved.Replacement == nil {
		t.Fatal("range move", e)
	}
	if moved.Replacement.EndsAt.Sub(moved.Replacement.StartsAt) != 4*time.Hour {
		t.Fatal("reschedule changed range length")
	}
	read(start, end, 1)
	read(start.AddDate(0, 0, 1), end.AddDate(0, 0, 1), 0)
	if _, e := f.bookings.Cancel(ctx, f.actor, f.business, moved.Replacement.ID, "range-cancel"); e != nil {
		t.Fatal(e)
	}
	read(start.AddDate(0, 0, 1), end.AddDate(0, 0, 1), 1)
	zone, _ := time.LoadLocation("Asia/Shanghai")
	schedule.ClosedDates = []string{end.In(zone).Format("2006-01-02")}
	if _, e := f.catalog.PutLocationSchedule(ctx, f.actor, f.business, loc.ID, schedule); e != nil {
		t.Fatal(e)
	}
	read(start, end, 0) // closed day in the MIDDLE/END of an interval also blocks it
	schedule.ClosedDates = []string{}
	if _, e := f.catalog.PutLocationSchedule(ctx, f.actor, f.business, loc.ID, schedule); e != nil {
		t.Fatal(e)
	}
	if _, e := f.catalog.PutAvailabilityWindows(ctx, f.actor, f.business, resource.ID, "resources", []service.AvailabilityWindow{}); e != nil {
		t.Fatal(e)
	}
	read(start, end, 1) // Missing resource windows do not override business opening hours.
}
