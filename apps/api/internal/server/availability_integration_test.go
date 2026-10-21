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
	"reflect"
	"strings"
	"testing"
	"time"
)

type availabilityFaultTransactions struct{ base service.BusinessTransactions }

func (f availabilityFaultTransactions) WithinBusiness(ctx context.Context, fn func(service.BusinessStore) error) error {
	return f.base.WithinBusiness(ctx, func(q service.BusinessStore) error { return fn(availabilityFaultStore{q}) })
}

type availabilityFaultStore struct{ service.BusinessStore }

func (availabilityFaultStore) CreateStaffAvailabilityWindow(context.Context, db.CreateStaffAvailabilityWindowParams) error {
	return errors.New("injected window failure")
}

func TestAvailabilityHTTPIntegration(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
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
			for _, table := range []string{"staff_availability_windows", "resource_availability_windows", "location_weekly_hours", "location_closed_dates", "service_scheduling_rules", "service_locations", "service_staff", "service_resources", "staff_locations", "resources", "staff_profiles", "services", "locations"} {
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
		if e := pool.QueryRow(ctx, "INSERT INTO users(email,username) VALUES($1,'availability fixture') RETURNING id", fmt.Sprintf("availability-%d-%d@example.invalid", time.Now().UnixNano(), i)).Scan(&id); e != nil {
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
	resource, e := catalog.CreateResource(ctx, users[0], a, service.ResourceInput{Name: "Room", LocationID: loc.ID, Capacity: 1})
	if e != nil {
		t.Fatal(e)
	}
	foreign, e := catalog.CreateStaffProfile(ctx, users[1], b, service.StaffProfileInput{Name: "Foreign"})
	if e != nil {
		t.Fatal(e)
	}
	h, e := NewCatalogHandler(sessions, catalog, "http://localhost:3000")
	if e != nil {
		t.Fatal(e)
	}
	mux := http.NewServeMux()
	RegisterCatalogRoutes(mux, h)
	uuid := func(id pgtype.UUID) string { v, _ := id.MarshalJSON(); return string(v[1 : len(v)-1]) }
	base := "/api/v1/businesses/" + uuid(a) + "/catalog/"
	day := time.Now().UTC().Truncate(24 * time.Hour).Add(24 * time.Hour)
	slotPath := base + "services/" + uuid(offered.ID) + "/slots?date=" + day.Format("2006-01-02") + "&location_id=" + uuid(loc.ID)
	staffPath := base + "staff/" + uuid(staff.ID) + "/availability-windows"
	resourcePath := base + "resources/" + uuid(resource.ID) + "/availability-windows"
	request := func(actor int, method, path, body, origin string, want int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body)).WithContext(ctx)
		if actor >= 0 {
			r.AddCookie(&http.Cookie{Name: "booking_session", Value: tokens[actor]})
		}
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Role", "BUSINESS_OWNER")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("%s %s got %d want %d %s", method, path, w.Code, want, w.Body.String())
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("missing no-store")
		}
		return w
	}
	const origin = "http://localhost:3000"
	countSlots := func(path string, want int) {
		t.Helper()
		w := request(0, "GET", path, "", "", 200)
		var out service.SlotResult
		if e := json.Unmarshal(w.Body.Bytes(), &out); e != nil || len(out.Slots) != want || !out.Advisory || out.Timezone != "UTC" {
			t.Fatal(out, e)
		}
	}
	request(-1, "OPTIONS", staffPath, "", origin, 204)
	request(-1, "GET", slotPath, "", "", 401)
	request(0, "PUT", staffPath, `{"windows":[]}`, "", 403)
	request(0, "POST", slotPath, "", origin, 405)
	request(0, "GET", slotPath+"&date=2026-10-11", "", "", 400)
	request(0, "GET", slotPath+"&unknown=1", "", "", 400)
	request(0, "GET", slotPath+"&staff_id=invalid", "", "", 400)
	request(1, "GET", slotPath, "", "", 403)
	request(0, "GET", base+"staff/"+uuid(foreign.ID)+"/availability-windows", "", "", 404)
	countSlots(slotPath, 0)
	if e := catalog.ChangeEligibility(ctx, users[0], a, offered.ID, loc.ID, service.ServiceLocation, false); e != nil {
		t.Fatal(e)
	}
	countSlots(slotPath, 0) // Explicit service rules and hours are still absent.
	if _, e := catalog.PutSchedulingRules(ctx, users[0], a, offered.ID, service.SchedulingRulesInput{BufferBeforeMinutes: 15, BufferAfterMinutes: 15}); e != nil {
		t.Fatal(e)
	}
	if _, e := catalog.PutLocationSchedule(ctx, users[0], a, loc.ID, service.LocationScheduleInput{WeeklyHours: []service.WeeklyHours{{Weekday: int32(day.Weekday()), StartMinute: 540, EndMinute: 1020}}, ClosedDates: []string{}}); e != nil {
		t.Fatal(e)
	}
	countSlots(slotPath, 29)
	selected := slotPath + "&staff_id=" + uuid(staff.ID) + "&resource_id=" + uuid(resource.ID)
	countSlots(selected, 0)
	for _, v := range []struct {
		parent, target pgtype.UUID
		kind           service.EligibilityKind
	}{{offered.ID, staff.ID, service.ServiceStaff}, {staff.ID, loc.ID, service.StaffLocation}, {offered.ID, resource.ID, service.ServiceResource}} {
		if e := catalog.ChangeEligibility(ctx, users[0], a, v.parent, v.target, v.kind, false); e != nil {
			t.Fatal(e)
		}
	}
	countSlots(selected, 29) // Assigned staff/resources follow business opening hours.
	staffWindows := []service.AvailabilityWindow{{StartsAt: day.Add(10 * time.Hour), EndsAt: day.Add(12 * time.Hour)}}
	resourceWindows := []service.AvailabilityWindow{{StartsAt: day.Add(10*time.Hour + 30*time.Minute), EndsAt: day.Add(12 * time.Hour)}}
	body := func(windows []service.AvailabilityWindow) string {
		v, e := json.Marshal(map[string]any{"windows": windows})
		if e != nil {
			t.Fatal(e)
		}
		return string(v)
	}
	for _, invalid := range []string{`{}`, `{"windows":null}`, `{"windows":[{"starts_at":"2026-10-11T09:00:00","ends_at":"2026-10-11T10:00:00"}]}`, `{"windows":[],"role":"BUSINESS_OWNER"}`} {
		request(0, "PUT", staffPath, invalid, origin, 400)
	}
	request(0, "PUT", staffPath, body(staffWindows), origin, 200)
	request(0, "PUT", resourcePath, body(resourceWindows), origin, 200)
	countSlots(selected, 29) // Narrow individual windows do not restrict the business.
	if e := tenants.ChangeMember(ctx, users[0], a, users[2], service.RoleStaff, false); e != nil {
		t.Fatal(e)
	}
	request(2, "GET", selected, "", "", 200)
	request(2, "PUT", staffPath, body(staffWindows), origin, 403)
	if e := tenants.ChangeMember(ctx, users[0], a, users[2], "", true); e != nil {
		t.Fatal(e)
	}
	request(2, "GET", selected, "", "", 403)
	fault := service.NewBookingCatalogService(service.NewBusinessService(availabilityFaultTransactions{repository.NewBusinessTransactions(pool)}))
	if out, e := fault.PutAvailabilityWindows(ctx, users[0], a, staff.ID, "staff", resourceWindows); e != service.ErrCatalogUnavailable || out != nil {
		t.Fatal("failure exposed result", out, e)
	}
	retained, e := catalog.GetAvailabilityWindows(ctx, users[0], a, staff.ID, "staff")
	if e != nil || !reflect.DeepEqual(retained, staffWindows) {
		t.Fatal("failed replace lost windows", retained, e)
	}
	_, e = pool.Exec(ctx, "INSERT INTO staff_availability_windows(business_id,staff_id,starts_at,ends_at) VALUES($1,$2,$3,$4)", a, foreign.ID, day, day.Add(time.Hour))
	var constraint *pgconn.PgError
	if !errors.As(e, &constraint) || constraint.Code != "23503" {
		t.Fatal("cross-tenant window accepted", e)
	}
	_, e = pool.Exec(ctx, "INSERT INTO resource_availability_windows(business_id,resource_id,starts_at,ends_at) VALUES($1,$2,$3,$4)", a, resource.ID, day, day)
	if !errors.As(e, &constraint) || constraint.Code != "23514" {
		t.Fatal("invalid window accepted", e)
	}
	if e := catalog.ChangeEligibility(ctx, users[0], a, staff.ID, loc.ID, service.StaffLocation, true); e != nil {
		t.Fatal(e)
	}
	countSlots(selected, 29) // No explicit staff branches means follow the service branches.
	if _, e := catalog.PutLocationSchedule(ctx, users[0], a, loc.ID, service.LocationScheduleInput{WeeklyHours: []service.WeeklyHours{{Weekday: int32(day.Weekday()), StartMinute: 0, EndMinute: 1440}}, ClosedDates: []string{day.Format("2006-01-02")}}); e != nil {
		t.Fatal(e)
	}
	countSlots(slotPath, 0)
	if _, e := pool.Exec(ctx, "UPDATE locations SET status='archived' WHERE id=$1", loc.ID); e != nil {
		t.Fatal(e)
	}
	request(0, "GET", slotPath, "", "", 404)
	request(0, "GET", resourcePath, "", "", 404)
}
