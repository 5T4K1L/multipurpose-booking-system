package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"multipurpose-booking-system/api/internal/db"
	"multipurpose-booking-system/api/internal/repository"
	"multipurpose-booking-system/api/internal/service"
)

type schedulingFaultTransactions struct{ base service.BusinessTransactions }

func (f schedulingFaultTransactions) WithinBusiness(ctx context.Context, fn func(service.BusinessStore) error) error {
	return f.base.WithinBusiness(ctx, func(q service.BusinessStore) error { return fn(schedulingFaultStore{q}) })
}

type schedulingFaultStore struct{ service.BusinessStore }

func (schedulingFaultStore) CreateLocationClosedDate(context.Context, db.CreateLocationClosedDateParams) error {
	return errors.New("injected write failure")
}

func TestSchedulingHTTPIntegration(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	users := []int64{}
	businesses := []pgtype.UUID{}
	defer func() {
		c, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		for _, id := range businesses {
			for _, table := range []string{"location_weekly_hours", "location_closed_dates", "service_scheduling_rules", "locations", "services"} {
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
	sessions, err := service.NewSessionService(repository.NewSessionTransactions(pool), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	tokens := []string{}
	locations := []pgtype.UUID{}
	services := []pgtype.UUID{}
	for i := 0; i < 4; i++ {
		var id int64
		if err := pool.QueryRow(ctx, "INSERT INTO users(email,username) VALUES($1,'schedule fixture') RETURNING id", fmt.Sprintf("schedule-%d-%d@example.invalid", time.Now().UnixNano(), i)).Scan(&id); err != nil {
			t.Fatal(err)
		}
		users = append(users, id)
		session, e := sessions.Create(ctx, id)
		if e != nil {
			t.Fatal(e)
		}
		tokens = append(tokens, session.Token)
	}
	for i := 0; i < 2; i++ {
		b, e := tenants.Create(ctx, users[i], service.BusinessInput{Name: "Schedule fixture", Timezone: "UTC"})
		if e != nil {
			t.Fatal(e)
		}
		businesses = append(businesses, b.ID)
		loc, e := catalog.CreateLocation(ctx, users[i], b.ID, service.LocationInput{Name: "Main", Timezone: "America/New_York"})
		if e != nil {
			t.Fatal(e)
		}
		locations = append(locations, loc.ID)
		offered, e := catalog.CreateService(ctx, users[i], b.ID, service.ServiceInput{Name: "Consultation", DurationMinutes: 30})
		if e != nil {
			t.Fatal(e)
		}
		services = append(services, offered.ID)
	}
	h, err := NewCatalogHandler(sessions, catalog, "http://localhost:3000")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	RegisterCatalogRoutes(mux, h)
	uuid := func(id pgtype.UUID) string { v, _ := id.MarshalJSON(); return string(v[1 : len(v)-1]) }
	base := "/api/v1/businesses/" + uuid(businesses[0]) + "/catalog/"
	schedulePath := base + "locations/" + uuid(locations[0]) + "/schedule"
	rulesPath := base + "services/" + uuid(services[0]) + "/scheduling-rules"
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
		r.Header.Set("X-User-ID", fmt.Sprint(users[0]))
		r.Header.Set("X-Role", "BUSINESS_OWNER")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("%s %s got %d want %d: %s", method, path, w.Code, want, w.Body.String())
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("missing cache protection")
		}
		return w
	}
	const origin = "http://localhost:3000"
	const schedule = `{"weekly_hours":[{"weekday":1,"start_minute":780,"end_minute":1020},{"weekday":1,"start_minute":540,"end_minute":720}],"closed_dates":["2026-11-01","2026-03-08"]}`
	const rules = `{"buffer_before_minutes":15,"buffer_after_minutes":10,"minimum_lead_minutes":120}`
	request(-1, "OPTIONS", schedulePath, "", origin, 204)
	request(-1, "GET", schedulePath, "", "", 401)
	request(0, "PUT", schedulePath, schedule, "", 403)
	request(0, "GET", schedulePath, "", "https://foreign.invalid", 403)
	request(0, "POST", schedulePath, schedule, origin, 405)
	request(0, "GET", schedulePath+"/"+uuid(locations[1]), "", "", 404)
	w := request(0, "GET", schedulePath, "", "", 200)
	var initial service.LocationSchedule
	if err := json.Unmarshal(w.Body.Bytes(), &initial); err != nil || initial.Timezone != "America/New_York" || len(initial.WeeklyHours) != 0 || initial.WeeklyHours == nil {
		t.Fatal(initial, err)
	}
	var initialRules service.SchedulingRules
	w = request(0, "GET", rulesPath, "", "", 200)
	if err := json.Unmarshal(w.Body.Bytes(), &initialRules); err != nil || initialRules.Configured {
		t.Fatal(initialRules, err)
	}
	for _, body := range []string{`{}`, `null`, `{"weekly_hours":[],"closed_dates":null}`, `{"weekly_hours":[{"weekday":1,"end_minute":720}],"closed_dates":[]}`, `{"weekly_hours":[{"weekday":1,"start_minute":540,"end_minute":720},{"weekday":1,"start_minute":600,"end_minute":800}],"closed_dates":[]}`, `{"weekly_hours":[],"closed_dates":["2026-02-29"]}`, `{"weekly_hours":[],"closed_dates":[],"timezone":"UTC"}`} {
		request(0, "PUT", schedulePath, body, origin, 400)
	}
	w = request(0, "PUT", schedulePath, schedule, origin, 200)
	var expected service.LocationSchedule
	if err := json.Unmarshal(w.Body.Bytes(), &expected); err != nil {
		t.Fatal(err)
	}
	if expected.WeeklyHours[0].StartMinute != 540 || expected.ClosedDates[0] != "2026-03-08" || expected.Timezone != "America/New_York" {
		t.Fatal("local schedule was not preserved", expected)
	}
	request(0, "PUT", rulesPath, `{"buffer_before_minutes":1}`, origin, 400)
	request(0, "PUT", rulesPath, `{"buffer_before_minutes":-1,"buffer_after_minutes":0,"minimum_lead_minutes":0}`, origin, 400)
	w = request(0, "PUT", rulesPath, rules, origin, 200)
	var configured service.SchedulingRules
	if err := json.Unmarshal(w.Body.Bytes(), &configured); err != nil || !configured.Configured || configured.MinimumLeadMinutes != 120 {
		t.Fatal(configured, err)
	}
	request(1, "GET", schedulePath, "", "", 403)
	request(0, "GET", base+"locations/"+uuid(locations[1])+"/schedule", "", "", 404)
	request(0, "PUT", base+"services/"+uuid(services[1])+"/scheduling-rules", rules, origin, 404)
	if _, err := pool.Exec(ctx, "INSERT INTO user_platform_roles(user_id,role) VALUES($1,'PLATFORM_ADMIN')", users[3]); err != nil {
		t.Fatal(err)
	}
	request(3, "GET", schedulePath, "", "", 403)
	for _, role := range []service.Role{service.RoleStaff, service.RoleManager, service.RoleBusinessAdmin} {
		if err := tenants.ChangeMember(ctx, users[0], businesses[0], users[2], role, false); err != nil {
			t.Fatal(err)
		}
		request(2, "GET", schedulePath, "", "", 200)
		want := 403
		if role == service.RoleBusinessAdmin {
			want = 200
		}
		request(2, "PUT", schedulePath, schedule, origin, want)
		request(2, "PUT", rulesPath, rules, origin, want)
	}
	if err := tenants.ChangeMember(ctx, users[0], businesses[0], users[2], "", true); err != nil {
		t.Fatal(err)
	}
	request(2, "GET", schedulePath, "", "", 403)
	// Fail after deleting the previous schedule and inserting new hours; the old
	// hours AND closed dates must survive together.
	faulty := service.NewBookingCatalogService(service.NewBusinessService(schedulingFaultTransactions{repository.NewBusinessTransactions(pool)}))
	replacement := service.LocationScheduleInput{WeeklyHours: []service.WeeklyHours{{Weekday: 2, StartMinute: 60, EndMinute: 120}}, ClosedDates: []string{"2026-12-25"}}
	out, e := faulty.PutLocationSchedule(ctx, users[0], businesses[0], locations[0], replacement)
	if e != service.ErrCatalogUnavailable || out.Timezone != "" {
		t.Fatal("partial failure exposed result", out, e)
	}
	retained, e := catalog.GetLocationSchedule(ctx, users[0], businesses[0], locations[0])
	if e != nil || !reflect.DeepEqual(retained, expected) {
		t.Fatal("failed replacement lost prior schedule", retained, e)
	}
	// Concurrent replacements must produce one complete schedule, never a mixture.
	other := service.LocationScheduleInput{WeeklyHours: []service.WeeklyHours{{Weekday: 3, StartMinute: 300, EndMinute: 400}}, ClosedDates: []string{"2026-12-26"}}
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, in := range []service.LocationScheduleInput{replacement, other} {
		go func(input service.LocationScheduleInput) {
			<-start
			_, e := catalog.PutLocationSchedule(ctx, users[0], businesses[0], locations[0], input)
			results <- e
		}(in)
	}
	close(start)
	for i := 0; i < 2; i++ {
		if e := <-results; e != nil {
			t.Fatal(e)
		}
	}
	final, e := catalog.GetLocationSchedule(ctx, users[0], businesses[0], locations[0])
	if e != nil {
		t.Fatal(e)
	}
	if !(reflect.DeepEqual(final.WeeklyHours, replacement.WeeklyHours) && reflect.DeepEqual(final.ClosedDates, replacement.ClosedDates)) && !(reflect.DeepEqual(final.WeeklyHours, other.WeeklyHours) && reflect.DeepEqual(final.ClosedDates, other.ClosedDates)) {
		t.Fatal("mixed concurrent schedules", final)
	}
	for _, query := range []string{"INSERT INTO location_closed_dates(business_id,location_id,closed_date) VALUES($1,$2,'2026-12-31')", "INSERT INTO location_weekly_hours(business_id,location_id,weekday,start_minute,end_minute) VALUES($1,$2,4,0,60)"} {
		_, err := pool.Exec(ctx, query, businesses[0], locations[1])
		var pgerr *pgconn.PgError
		if !errors.As(err, &pgerr) || pgerr.Code != "23503" {
			t.Fatal("foreign location linked", err)
		}
	}
	_, err = pool.Exec(ctx, "INSERT INTO service_scheduling_rules(business_id,service_id,buffer_before_minutes,buffer_after_minutes,minimum_lead_minutes) VALUES($1,$2,0,0,0)", businesses[0], services[1])
	var pgerr *pgconn.PgError
	if !errors.As(err, &pgerr) || pgerr.Code != "23503" {
		t.Fatal("foreign service linked", err)
	}
	for _, query := range []string{
		"INSERT INTO location_weekly_hours(business_id,location_id,weekday,start_minute,end_minute) VALUES($1,$2,7,0,60)",
		"INSERT INTO location_weekly_hours(business_id,location_id,weekday,start_minute,end_minute) VALUES($1,$2,4,720,540)",
		"INSERT INTO location_weekly_hours(business_id,location_id,weekday,start_minute,end_minute) VALUES($1,$2,4,0,1441)",
	} {
		_, e := pool.Exec(ctx, query, businesses[0], locations[0])
		var constraint *pgconn.PgError
		if !errors.As(e, &constraint) || constraint.Code != "23514" {
			t.Fatal("invalid stored hours accepted", e)
		}
	}
	_, e = pool.Exec(ctx, "UPDATE service_scheduling_rules SET buffer_before_minutes=-1 WHERE business_id=$1 AND service_id=$2", businesses[0], services[0])
	var check *pgconn.PgError
	if !errors.As(e, &check) || check.Code != "23514" {
		t.Fatal("invalid stored buffer accepted", e)
	}
	request(0, "PUT", schedulePath, `{"weekly_hours":[],"closed_dates":[]}`, origin, 200)
	if _, err := pool.Exec(ctx, "UPDATE locations SET status='archived' WHERE id=$1", locations[0]); err != nil {
		t.Fatal(err)
	}
	request(0, "GET", schedulePath, "", "", 404)
	request(0, "PUT", schedulePath, schedule, origin, 404)
	if _, err := pool.Exec(ctx, "UPDATE businesses SET status='disabled' WHERE id=$1", businesses[0]); err != nil {
		t.Fatal(err)
	}
	request(0, "GET", rulesPath, "", "", 403)
}
