package server

import (
	"context"
	"encoding/json"
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

func TestBookingReadsAndOccupancy(t *testing.T) {
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
		if e := pool.QueryRow(ctx, "INSERT INTO users(email,username) VALUES($1,'booking reads fixture') RETURNING id", fmt.Sprintf("booking-reads-%d-%d@example.invalid", time.Now().UnixNano(), i)).Scan(&id); e != nil {
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

	slotRequest := service.SlotRequest{Date: day.Format("2006-01-02"), LocationID: loc.ID, StaffID: staff.ID, ResourceID: resource.ID}
	countSlots := func(request service.SlotRequest, want int) {
		t.Helper()
		result, e := catalog.CalculateSlots(ctx, users[0], a, offered.ID, request)
		if e != nil || len(result.Slots) != want || !result.Advisory {
			t.Fatalf("slots=%d want=%d error=%v", len(result.Slots), want, e)
		}
	}
	countSlots(slotRequest, 29)
	first := create("occupied", in, nil)
	countSlots(slotRequest, 22)
	onlyResource := slotRequest
	onlyResource.StaffID = pgtype.UUID{}
	countSlots(onlyResource, 29)
	noSelection := onlyResource
	noSelection.ResourceID = pgtype.UUID{}
	countSlots(noSelection, 29)
	resourceInput := in
	resourceInput.StaffID = pgtype.UUID{}
	second := create("capacity-two", resourceInput, nil)
	countSlots(onlyResource, 22)
	if _, e := bookings.Cancel(ctx, users[0], a, second.ID, "release-one"); e != nil {
		t.Fatal(e)
	}
	countSlots(onlyResource, 29)
	countSlots(slotRequest, 22)
	if _, e := bookings.Cancel(ctx, users[0], a, first.ID, "release-staff"); e != nil {
		t.Fatal(e)
	}
	countSlots(slotRequest, 29)
	original := create("move-original", in, nil)
	moved, e := bookings.Reschedule(ctx, users[0], a, original.ID, "move", day.Add(14*time.Hour))
	if e != nil {
		t.Fatal(e)
	}
	countSlots(slotRequest, 22)
	view, e := bookings.Get(ctx, users[0], a, original.ID)
	if e != nil || view.Status != "rescheduled" || view.ReplacementID != moved.Replacement.ID {
		t.Fatal(view, e)
	}
	view, e = bookings.Get(ctx, users[0], a, moved.Replacement.ID)
	if e != nil || view.Status != "confirmed" || view.SupersedesID != original.ID {
		t.Fatal(view, e)
	}
	if _, e := bookings.Get(ctx, users[1], b, original.ID); e != service.ErrBookingNotFound {
		t.Fatal(e)
	}
	if _, e := bookings.Get(ctx, users[1], a, original.ID); e != service.ErrForbidden {
		t.Fatal(e)
	}
	// Capacity sweep must allow adjacent occupants plus one spanning both at capacity 2.
	stamp := func(v time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: v, Valid: true} }
	for i, interval := range [][2]time.Time{{day.Add(11 * time.Hour), day.Add(11*time.Hour + 15*time.Minute)}, {day.Add(11*time.Hour + 15*time.Minute), day.Add(11*time.Hour + 30*time.Minute)}} {
		_, e := db.New(pool).CreateBooking(ctx, db.CreateBookingParams{BusinessID: a, ServiceID: offered.ID, LocationID: loc.ID, ResourceID: resource.ID, CreatedBy: users[0], IdempotencyKey: fmt.Sprintf("adjacent-%d", i), RequestHash: strings.Repeat("0", 64), StartsAt: stamp(interval[0]), EndsAt: stamp(interval[1]), BusyStartsAt: stamp(interval[0]), BusyEndsAt: stamp(interval[1])})
		if e != nil {
			t.Fatal(e)
		}
	}
	got, e := catalog.CalculateSlots(ctx, users[0], a, offered.ID, onlyResource)
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, slot := range got.Slots {
		if slot.StartsAt.Equal(day.Add(11 * time.Hour)) {
			found = true
		}
	}
	if !found {
		t.Fatal("adjacent capacity double counted")
	}
	for i := 0; i < 105; i++ {
		input := in
		input.StaffID = pgtype.UUID{}
		input.ResourceID = pgtype.UUID{}
		create(fmt.Sprintf("page-%d", i), input, nil)
	}
	// Fresh membership on every page; cursors are actor/business/kind-bound.
	seen := map[pgtype.UUID]bool{}
	cursor := ""
	var firstCursor string
	for {
		page, e := bookings.List(ctx, users[0], a, service.PageRequest{Limit: 17, Cursor: cursor})
		if e != nil {
			t.Fatal(e)
		}
		for _, v := range page.Items {
			if seen[v.ID] {
				t.Fatal("duplicate page")
			}
			seen[v.ID] = true
		}
		if firstCursor == "" {
			firstCursor = page.NextCursor
		}
		cursor = page.NextCursor
		if cursor == "" {
			break
		}
	}
	if len(seen) != 111 {
		t.Fatal("missing history", len(seen))
	}
	for _, role := range []service.Role{service.RoleBusinessAdmin, service.RoleManager, service.RoleStaff} {
		if e := tenants.ChangeMember(ctx, users[0], a, users[2], role, false); e != nil {
			t.Fatal(e)
		}
		if _, e := bookings.Get(ctx, users[2], a, first.ID); e != nil {
			t.Fatal(role, e)
		}
		if _, e := bookings.List(ctx, users[2], a, service.PageRequest{}); e != nil {
			t.Fatal(role, e)
		}
	}
	if _, e := bookings.List(ctx, users[2], a, service.PageRequest{Cursor: firstCursor}); e != service.ErrInvalidPage {
		t.Fatal("actor cursor reused", e)
	}
	if _, e := bookings.List(ctx, users[0], b, service.PageRequest{Cursor: firstCursor}); e != service.ErrInvalidPage {
		t.Fatal("tenant cursor reused", e)
	}
	staffPage, e := bookings.List(ctx, users[2], a, service.PageRequest{Limit: 1})
	if e != nil {
		t.Fatal(e)
	}
	if e := tenants.ChangeMember(ctx, users[0], a, users[2], service.RoleStaff, true); e != nil {
		t.Fatal(e)
	}
	if _, e := bookings.List(ctx, users[2], a, service.PageRequest{Cursor: staffPage.NextCursor}); e != service.ErrForbidden {
		t.Fatal("revoked cursor access", e)
	}
	// Read history remains visible if its catalog entry is archived.
	if _, e := pool.Exec(ctx, "UPDATE services SET status='archived' WHERE id=$1", offered.ID); e != nil {
		t.Fatal(e)
	}
	if _, e := bookings.Get(ctx, users[0], a, first.ID); e != nil {
		t.Fatal(e)
	}
	h, e := NewBookingHandler(sessions, bookings, "http://localhost:3000")
	if e != nil {
		t.Fatal(e)
	}
	mux := http.NewServeMux()
	RegisterBookingRoutes(mux, h)
	uuid := func(id pgtype.UUID) string { v, _ := id.MarshalJSON(); return strings.Trim(string(v), "\"") }
	base := "/api/v1/businesses/" + uuid(a) + "/bookings"
	request := func(actor int, path, origin string, want int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest("GET", path, nil).WithContext(ctx)
		if actor >= 0 {
			r.AddCookie(&http.Cookie{Name: "booking_session", Value: tokens[actor]})
		}
		r.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatal(w.Code, want, w.Body.String())
		}
		return w
	}
	request(-1, base, "", 401)
	request(1, base, "", 403)
	request(0, base, "https://foreign.invalid", 403)
	response := request(0, base+"?limit=2", "http://localhost:3000", 200)
	if response.Header().Get("X-Next-Cursor") == "" || response.Header().Get("Access-Control-Expose-Headers") != "X-Next-Cursor" || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal(response.Header())
	}
	var list []map[string]any
	if e := json.Unmarshal(response.Body.Bytes(), &list); e != nil || len(list) != 2 {
		t.Fatal(list, e)
	}
	for _, v := range list {
		for _, private := range []string{"request_hash", "created_by", "idempotency_key", "closed_by"} {
			if _, ok := v[private]; ok {
				t.Fatal("private field", private)
			}
		}
	}
	request(0, base+"/"+uuid(first.ID), "", 200)
	request(0, base+"/00000000-0000-0000-0000-000000000000", "", 404)
	for _, query := range []string{"?limit=0", "?limit=101", "?limit=2&limit=3", "?cursor=bad", "?unknown=1"} {
		request(0, base+query, "", 400)
	}
	request(0, base+"/"+uuid(first.ID)+"?limit=1", "", 400)
}
