package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"multipurpose-booking-system/api/internal/service"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

type publicAvailabilityFixture struct {
	ctx                                           context.Context
	pool                                          *pgxpool.Pool
	tenants                                       *service.BusinessService
	catalog                                       *service.BookingCatalogService
	bookings                                      *service.BookingService
	actor                                         int64
	business, foreign, serviceID, location, staff pgtype.UUID
	day                                           time.Time
	request                                       func(int, string, string, string, string, int) *httptest.ResponseRecorder
}

func verifyPublicAvailability(t *testing.T, f publicAvailabilityFixture) {
	t.Helper()
	ctx := f.ctx
	uuid := func(id pgtype.UUID) string { v, _ := id.MarshalJSON(); return string(v[1 : len(v)-1]) }
	date := f.day.Format("2006-01-02")
	path := "/api/v1/public/businesses/" + uuid(f.business) + "/services/" + uuid(f.serviceID) + "/availability"
	query := "?location_id=" + uuid(f.location) + "&date=" + date
	read := func() service.SlotResult {
		t.Helper()
		w := f.request(-1, "GET", path+query, "", "", 200)
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("cacheable availability")
		}
		var result service.SlotResult
		if e := json.Unmarshal(w.Body.Bytes(), &result); e != nil {
			t.Fatal(e)
		}
		if !result.Advisory || result.Date != date || result.Timezone != "UTC" {
			t.Fatal("wrong advisory envelope", result)
		}
		for _, key := range []string{"staff_id", "resource_id", "capacity", "booking_id", "customer_id"} {
			if strings.Contains(w.Body.String(), key) {
				t.Fatal("internal allocation leaked", key)
			}
		}
		return result
	}
	calendarPath := strings.TrimSuffix(path, "availability") + "calendar"
	calendarQuery := "?location_id=" + uuid(f.location) + "&month=" + f.day.Format("2006-01")
	calendarDay := func(want string) service.PublicCalendarDay {
		t.Helper()
		w := f.request(-1, "GET", calendarPath+calendarQuery, "", "", 200)
		var result service.PublicCalendar
		if e := json.Unmarshal(w.Body.Bytes(), &result); e != nil {
			t.Fatal(e)
		}
		days := time.Date(f.day.Year(), f.day.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
		if len(result.Days) != days || result.Timezone != "UTC" || !result.Advisory || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("bad calendar", result)
		}
		for _, private := range []string{"staff_id", "resource_id", "booking_id", "customer_id", "capacity"} {
			if strings.Contains(w.Body.String(), private) {
				t.Fatal("calendar leaked", private)
			}
		}
		for _, day := range result.Days {
			if day.Date == date {
				if day.Status != want {
					t.Fatalf("calendar status %s want %s", day.Status, want)
				}
				return day
			}
		}
		t.Fatal("missing calendar date")
		return service.PublicCalendarDay{}
	}
	calendarDay("available")
	for _, query := range []string{"&month=bad", "&month=2026-13", "&month=1999-12", "&month=2101-01", "&date=" + date, "&month=" + f.day.Format("2006-01") + "&date=" + date} {
		f.request(-1, "GET", calendarPath+"?location_id="+uuid(f.location)+query, "", "", 400)
	}
	f.request(-1, "GET", calendarPath+calendarQuery+"&month=2026-01", "", "", 400)
	f.request(-1, "GET", calendarPath+strings.Replace(calendarQuery, uuid(f.location), uuid(f.foreign), 1), "", "", 404)
	f.request(-1, "GET", strings.Replace(calendarPath, uuid(f.business), uuid(f.foreign), 1)+calendarQuery, "", "", 404)
	f.request(-1, "GET", calendarPath+calendarQuery, "", "https://foreign.invalid", 403)
	f.request(-1, "POST", calendarPath+calendarQuery, "{}", "http://localhost:3000", 405)
	base := read()
	internal, e := f.catalog.CalculateSlots(ctx, f.actor, f.business, f.serviceID, service.SlotRequest{Date: date, LocationID: f.location, StaffID: f.staff})
	if e != nil {
		t.Fatal(e)
	}
	if len(base.Slots) == 0 || !reflect.DeepEqual(base, internal) {
		t.Fatal("public read differs from existing scheduling/occupancy", base, internal)
	}
	for _, suffix := range []string{"&date=" + date, "&staff_id=" + uuid(f.staff), "&unknown=1"} {
		f.request(-1, "GET", path+query+suffix, "", "", 400)
	}
	f.request(-1, "GET", path+"?location_id="+uuid(f.location)+"&date=bad", "", "", 400)
	f.request(-1, "GET", path+"?location_id="+uuid(f.location)+"&date=1999-12-31", "", "", 400)
	f.request(-1, "GET", path+"?location_id="+uuid(f.foreign)+"&date="+date, "", "", 404)
	f.request(-1, "GET", strings.Replace(path, uuid(f.business), uuid(f.foreign), 1)+query, "", "", 404)
	f.request(-1, "POST", path+query, "{}", "http://localhost:3000", 405)
	f.request(-1, "GET", path+query, "", "https://foreign.invalid", 403)
	// A second eligible staff member can cover occupied times without exposing their identity.
	second, e := f.catalog.CreateStaffProfile(ctx, f.actor, f.business, service.StaffProfileInput{Name: "Private alternate provider"})
	if e != nil {
		t.Fatal(e)
	}
	for _, link := range []struct {
		parent, target pgtype.UUID
		kind           service.EligibilityKind
	}{{f.serviceID, second.ID, service.ServiceStaff}, {second.ID, f.location, service.StaffLocation}} {
		if e := f.catalog.ChangeEligibility(ctx, f.actor, f.business, link.parent, link.target, link.kind, false); e != nil {
			t.Fatal(e)
		}
	}
	window := []service.AvailabilityWindow{{StartsAt: f.day.Add(9 * time.Hour), EndsAt: f.day.Add(17 * time.Hour)}}
	if _, e := f.catalog.PutAvailabilityWindows(ctx, f.actor, f.business, second.ID, "staff", []service.AvailabilityWindow{}); e != nil {
		t.Fatal(e)
	}
	expanded := read()
	if len(expanded.Slots) <= len(base.Slots) {
		t.Fatal("alternative staff did not restore availability")
	}
	seen := map[time.Time]bool{}
	for _, slot := range expanded.Slots {
		if seen[slot.StartsAt] {
			t.Fatal("duplicate slot")
		}
		seen[slot.StartsAt] = true
	}
	resource, e := f.catalog.CreateResource(ctx, f.actor, f.business, service.ResourceInput{Name: "Private room", LocationID: f.location, Capacity: 2})
	if e != nil {
		t.Fatal(e)
	}
	if e := f.catalog.ChangeEligibility(ctx, f.actor, f.business, f.serviceID, resource.ID, service.ServiceResource, false); e != nil {
		t.Fatal(e)
	}
	if _, e := f.catalog.PutAvailabilityWindows(ctx, f.actor, f.business, resource.ID, "resources", window); e != nil {
		t.Fatal(e)
	}
	start := f.day.Add(15 * time.Hour)
	contains := func(result service.SlotResult) bool {
		for _, slot := range result.Slots {
			if slot.StartsAt.Equal(start) {
				return true
			}
		}
		return false
	}
	if !contains(read()) {
		t.Fatal("test time missing")
	}
	makeBooking := func(staff pgtype.UUID, key string) pgtype.UUID {
		t.Helper()
		v, e := f.bookings.Create(ctx, f.actor, f.business, key, service.BookingInput{ServiceID: f.serviceID, LocationID: f.location, StaffID: staff, ResourceID: resource.ID, StartsAt: start})
		if e != nil {
			t.Fatal(e)
		}
		return v.ID
	}
	// Only one possible start, with two staff and capacity two: one booking
	// must leave the date selectable; only the second exhausts the date.
	if _, e := f.catalog.PutLocationSchedule(ctx, f.actor, f.business, f.location, service.LocationScheduleInput{WeeklyHours: []service.WeeklyHours{{Weekday: int32(f.day.Weekday()), StartMinute: 900, EndMinute: 930}}, ClosedDates: []string{}}); e != nil {
		t.Fatal(e)
	}
	calendarDay("available")
	first := makeBooking(f.staff, "availability-capacity-one")
	if !contains(read()) {
		t.Fatal("spare resource capacity hidden")
	}
	calendarDay("available")
	next := makeBooking(second.ID, "availability-capacity-two")
	if contains(read()) {
		t.Fatal("full resource time shown")
	}
	calendarDay("fully_booked")
	if _, e := f.bookings.Cancel(ctx, f.actor, f.business, first, "availability-release-one"); e != nil {
		t.Fatal(e)
	}
	if !contains(read()) {
		t.Fatal("cancelled occupancy not released")
	}
	calendarDay("available")
	if _, e := f.bookings.Cancel(ctx, f.actor, f.business, next, "availability-release-two"); e != nil {
		t.Fatal(e)
	}
	schedule := service.LocationScheduleInput{WeeklyHours: []service.WeeklyHours{{Weekday: int32(f.day.Weekday()), StartMinute: 540, EndMinute: 1020}}, ClosedDates: []string{date}}
	if _, e := f.catalog.PutLocationSchedule(ctx, f.actor, f.business, f.location, schedule); e != nil {
		t.Fatal(e)
	}
	calendarDay("unavailable")
	if len(read().Slots) != 0 {
		t.Fatal("closed day offered")
	}
	schedule.ClosedDates = []string{}
	if _, e := f.catalog.PutLocationSchedule(ctx, f.actor, f.business, f.location, schedule); e != nil {
		t.Fatal(e)
	}
	if _, e := f.catalog.PutSchedulingRules(ctx, f.actor, f.business, f.serviceID, service.SchedulingRulesInput{BufferBeforeMinutes: 15, BufferAfterMinutes: 15}); e != nil {
		t.Fatal(e)
	}
	if len(read().Slots) >= len(expanded.Slots) {
		t.Fatal("buffers ignored")
	}
	if _, e := f.catalog.PutSchedulingRules(ctx, f.actor, f.business, f.serviceID, service.SchedulingRulesInput{MinimumLeadMinutes: 525600}); e != nil {
		t.Fatal(e)
	}
	calendarDay("unavailable")
	if len(read().Slots) != 0 {
		t.Fatal("lead time ignored")
	}
	if _, e := f.catalog.PutSchedulingRules(ctx, f.actor, f.business, f.serviceID, service.SchedulingRulesInput{}); e != nil {
		t.Fatal(e)
	}
	if _, e := f.pool.Exec(ctx, "UPDATE staff_profiles SET status='archived' WHERE business_id=$1", f.business); e != nil {
		t.Fatal(e)
	}
	if len(read().Slots) != 0 {
		t.Fatal("archived required staff bypassed")
	}
	if _, e := f.pool.Exec(ctx, "UPDATE staff_profiles SET status='active' WHERE business_id=$1", f.business); e != nil {
		t.Fatal(e)
	}
	if e := f.catalog.ChangeEligibility(ctx, f.actor, f.business, f.serviceID, f.location, service.ServiceLocation, true); e != nil {
		t.Fatal(e)
	}
	f.request(-1, "GET", path+query, "", "", 404)
	if e := f.catalog.ChangeEligibility(ctx, f.actor, f.business, f.serviceID, f.location, service.ServiceLocation, false); e != nil {
		t.Fatal(e)
	}
	if _, e := f.pool.Exec(ctx, "UPDATE services SET duration_minutes=45 WHERE id=$1", f.serviceID); e != nil {
		t.Fatal(e)
	}
	f.request(-1, "GET", calendarPath+calendarQuery, "", "", 409)
	f.request(-1, "GET", path+query, "", "", 409)
	if _, e := f.pool.Exec(ctx, "UPDATE services SET duration_minutes=30 WHERE id=$1", f.serviceID); e != nil {
		t.Fatal(e)
	}
	if _, e := f.pool.Exec(ctx, "UPDATE locations SET timezone='Asia/Shanghai' WHERE id=$1", f.location); e != nil {
		t.Fatal(e)
	}
	f.request(-1, "GET", calendarPath+calendarQuery, "", "", 409)
	f.request(-1, "GET", path+query, "", "", 409)
	if _, e := f.pool.Exec(ctx, "UPDATE locations SET timezone='UTC' WHERE id=$1", f.location); e != nil {
		t.Fatal(e)
	}
	if _, e := f.pool.Exec(ctx, "UPDATE businesses SET is_published=false WHERE id=$1", f.business); e != nil {
		t.Fatal(e)
	}
	f.request(-1, "GET", path+query, "", "", 404)
	if _, e := f.pool.Exec(ctx, "UPDATE businesses SET is_published=true WHERE id=$1", f.business); e != nil {
		t.Fatal(e)
	}
	// Do not silently truncate candidate sets into a misleading partial answer.
	extras := []pgtype.UUID{}
	for i := 0; i < 31; i++ {
		st, e := f.catalog.CreateStaffProfile(ctx, f.actor, f.business, service.StaffProfileInput{Name: fmt.Sprintf("Limit fixture %d", i)})
		if e != nil {
			t.Fatal(e)
		}
		extras = append(extras, st.ID)
		for _, link := range []struct {
			parent, target pgtype.UUID
			kind           service.EligibilityKind
		}{{f.serviceID, st.ID, service.ServiceStaff}, {st.ID, f.location, service.StaffLocation}} {
			if e := f.catalog.ChangeEligibility(ctx, f.actor, f.business, link.parent, link.target, link.kind, false); e != nil {
				t.Fatal(e)
			}
		}
	}
	f.request(-1, "GET", calendarPath+calendarQuery, "", "", 503)
	f.request(-1, "GET", path+query, "", "", 503)
	if _, e := f.pool.Exec(ctx, "UPDATE staff_profiles SET status='archived' WHERE id=ANY($1::uuid[])", extras); e != nil {
		t.Fatal(e)
	}
	if len(read().Slots) == 0 {
		t.Fatal("availability did not recover")
	}
}
