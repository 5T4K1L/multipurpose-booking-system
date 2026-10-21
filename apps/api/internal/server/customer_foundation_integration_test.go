package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"multipurpose-booking-system/api/internal/repository"
	"multipurpose-booking-system/api/internal/service"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCustomerFoundationIntegration(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
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
			for _, table := range []string{"booking_changes", "bookings", "staff_availability_windows", "resource_availability_windows", "service_resources", "resources", "location_weekly_hours", "location_closed_dates", "service_scheduling_rules", "service_locations", "service_staff", "staff_locations", "staff_profiles", "services", "locations"} {
				if _, e := pool.Exec(c, "DELETE FROM "+table+" WHERE business_id=$1", id); e != nil {
					t.Error(e)
				}
			}
			if _, e := pool.Exec(c, "DELETE FROM businesses WHERE id=$1", id); e != nil {
				t.Error(e)
			}
		}
		for _, id := range users {
			for _, table := range []string{"user_sessions", "user_platform_roles", "users"} {
				column := "user_id"
				if table == "users" {
					column = "id"
				}
				if _, e := pool.Exec(c, "DELETE FROM "+table+" WHERE "+column+"=$1", id); e != nil {
					t.Error(e)
				}
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
	for i := 0; i < 4; i++ {
		var id int64
		if e := pool.QueryRow(ctx, "INSERT INTO users(email,username) VALUES($1,'customer foundation fixture') RETURNING id", fmt.Sprintf("customer-foundation-%d-%d@example.invalid", time.Now().UnixNano(), i)).Scan(&id); e != nil {
			t.Fatal(e)
		}
		users = append(users, id)
		v, e := sessions.Create(ctx, id)
		if e != nil {
			t.Fatal(e)
		}
		tokens = append(tokens, v.Token)
	}
	for i := 0; i < 2; i++ {
		b, e := tenants.Create(ctx, users[i], service.BusinessInput{Name: "Customer foundation fixture", Timezone: "UTC"})
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
	staff, e := catalog.CreateStaffProfile(ctx, users[0], a, service.StaffProfileInput{Name: "Provider"})
	if e != nil {
		t.Fatal(e)
	}
	day := time.Now().UTC().Truncate(24 * time.Hour).Add(48 * time.Hour)
	for _, link := range []struct {
		parent, target pgtype.UUID
		kind           service.EligibilityKind
	}{{offered.ID, loc.ID, service.ServiceLocation}, {offered.ID, staff.ID, service.ServiceStaff}, {staff.ID, loc.ID, service.StaffLocation}} {
		if e := catalog.ChangeEligibility(ctx, users[0], a, link.parent, link.target, link.kind, false); e != nil {
			t.Fatal(e)
		}
	}
	if _, e := catalog.PutSchedulingRules(ctx, users[0], a, offered.ID, service.SchedulingRulesInput{}); e != nil {
		t.Fatal(e)
	}
	if _, e := catalog.PutLocationSchedule(ctx, users[0], a, loc.ID, service.LocationScheduleInput{WeeklyHours: []service.WeeklyHours{{Weekday: int32(day.Weekday()), StartMinute: 540, EndMinute: 1020}}, ClosedDates: []string{}}); e != nil {
		t.Fatal(e)
	}
	if _, e := catalog.PutAvailabilityWindows(ctx, users[0], a, staff.ID, "staff", []service.AvailabilityWindow{{StartsAt: day.Add(9 * time.Hour), EndsAt: day.Add(17 * time.Hour)}}); e != nil {
		t.Fatal(e)
	}
	seed := func(key string, owner any, hour int) pgtype.UUID {
		t.Helper()
		var id pgtype.UUID
		start := day.Add(time.Duration(hour) * time.Hour)
		e := pool.QueryRow(ctx, `INSERT INTO bookings(business_id,service_id,location_id,staff_id,created_by,idempotency_key,request_hash,starts_at,ends_at,busy_starts_at,busy_ends_at,customer_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$8,$9,$10) RETURNING id`, a, offered.ID, loc.ID, staff.ID, users[2], key, strings.Repeat("a", 64), start, start.Add(30*time.Minute), owner).Scan(&id)
		if e != nil {
			t.Fatal(e)
		}
		return id
	}
	owned := seed("owned", users[2], 10)
	other := seed("other", users[3], 11)
	legacy := seed("legacy", nil, 12)
	uuid := func(id pgtype.UUID) string { v, _ := id.MarshalJSON(); return string(v[1 : len(v)-1]) }
	mux := http.NewServeMux()
	bh, e := NewBusinessHandler(sessions, tenants, "http://localhost:3000")
	if e != nil {
		t.Fatal(e)
	}
	mux.Handle("/api/v1/businesses/", bh)
	bookings := service.NewBookingService(tenants)
	h, e := NewBookingHandler(sessions, bookings, "http://localhost:3000")
	if e != nil {
		t.Fatal(e)
	}
	RegisterBookingRoutes(mux, h)
	catalogHandler, e := NewCatalogHandler(sessions, catalog, "http://localhost:3000")
	if e != nil {
		t.Fatal(e)
	}
	RegisterCatalogRoutes(mux, catalogHandler)
	if e := RegisterCustomerFoundationRoutes(mux, sessions, tenants, "http://localhost:3000"); e != nil {
		t.Fatal(e)
	}
	request := func(actor int, method, path, body, origin string, want int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body)).WithContext(ctx)
		if actor >= 0 {
			r.AddCookie(&http.Cookie{Name: "booking_session", Value: tokens[actor]})
		}
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		if body != "" {
			r.Header.Set("Content-Type", "application/json")
		}
		r.Header.Set("X-User-ID", fmt.Sprint(users[2]))
		r.Header.Set("X-Role", "BUSINESS_OWNER")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("%s %s actor %d got %d want %d: %s", method, path, actor, w.Code, want, w.Body.String())
		}
		return w
	}
	t.Run("customer CRM", func(t *testing.T) {
		path := "/api/v1/businesses/" + uuid(a) + "/customers"
		for _, actor := range []int{1, 2, 3} {
			request(actor, "GET", path, "", "", 403)
		}
		request(-1, "GET", path, "", "", 401)
		request(0, "GET", path, "", "https://foreign.invalid", 403)
		request(0, "POST", path, "{}", "http://localhost:3000", 405)
		request(0, "GET", path+"?customer_id=1", "", "", 400)
		request(0, "GET", path+"?q=a&q=b", "", "", 400)
		first := request(0, "GET", path+"?limit=1", "", "", 200)
		if first.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("cacheable CRM")
		}
		var rows []service.BusinessCustomer
		if e := json.Unmarshal(first.Body.Bytes(), &rows); e != nil || len(rows) != 1 || rows[0].ID != users[2] || rows[0].BookingRecords != 1 {
			t.Fatal("wrong customer projection", rows, e)
		}
		cursor := first.Header().Get("X-Next-Cursor")
		if cursor == "" {
			t.Fatal("missing CRM cursor")
		}
		next := request(0, "GET", path+"?limit=1&cursor="+cursor, "", "", 200)
		if e := json.Unmarshal(next.Body.Bytes(), &rows); e != nil || len(rows) != 1 || rows[0].ID != users[3] {
			t.Fatal("CRM pagination", rows, e)
		}
		request(0, "GET", path+"?q=changed&cursor="+cursor, "", "", 400)
		request(1, "GET", "/api/v1/businesses/"+uuid(b)+"/customers?cursor="+cursor, "", "", 400)
		empty := request(1, "GET", "/api/v1/businesses/"+uuid(b)+"/customers", "", "", 200)
		if strings.TrimSpace(empty.Body.String()) != "[]" {
			t.Fatal("tenant leakage", empty.Body.String())
		}
		literal := request(0, "GET", path+"?q=%25", "", "", 200)
		if strings.TrimSpace(literal.Body.String()) != "[]" {
			t.Fatal("wildcard search leaked all rows")
		}
		found := request(0, "GET", path+"?q=FOUNDATION", "", "", 200)
		if e := json.Unmarshal(found.Body.Bytes(), &rows); e != nil || len(rows) != 2 {
			t.Fatal("case-insensitive search", rows, e)
		}
		if e := tenants.ChangeMember(ctx, users[0], a, users[1], service.RoleStaff, false); e != nil {
			t.Fatal(e)
		}
		request(1, "GET", path, "", "", 403)
		if e := tenants.ChangeMember(ctx, users[0], a, users[1], service.RoleBusinessAdmin, false); e != nil {
			t.Fatal(e)
		}
		request(1, "GET", path, "", "", 200)
		if e := tenants.ChangeMember(ctx, users[0], a, users[1], "", true); e != nil {
			t.Fatal(e)
		}
		request(1, "GET", path, "", "", 403)
	})
	ownPath := "/api/v1/customer/bookings/" + uuid(owned)
	origin := "http://localhost:3000"
	response := request(2, "GET", ownPath, "", "", 200)
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("cacheable ownership response")
	}
	var fields map[string]any
	if e := json.Unmarshal(response.Body.Bytes(), &fields); e != nil {
		t.Fatal(e)
	}
	for key, want := range map[string]string{"business_name": "Customer foundation fixture", "service_name": "Consultation", "location_name": "Main"} {
		if fields[key] != want {
			t.Fatalf("booking %s = %v, want %s", key, fields[key], want)
		}
	}
	namesList := request(2, "GET", "/api/v1/customer/bookings", "", "", 200)
	var namedRows []service.CustomerBooking
	if err := json.Unmarshal(namesList.Body.Bytes(), &namedRows); err != nil || len(namedRows) != 1 || namedRows[0].BusinessName != "Customer foundation fixture" || namedRows[0].ServiceName != "Consultation" || namedRows[0].LocationName != "Main" {
		t.Fatalf("unpublished customer booking names missing: %s", namesList.Body.String())
	}
	for _, field := range []string{"customer_id", "created_by", "idempotency_key", "request_hash", "closed_by", "staff_id", "resource_id"} {
		if _, ok := fields[field]; ok {
			t.Fatal("private field exposed", field)
		}
	}
	for _, actor := range []int{0, 1, 3} {
		request(actor, "GET", ownPath, "", "", 404)
	}
	request(-1, "GET", ownPath, "", "", 401)
	request(2, "GET", "/api/v1/customer/bookings/"+uuid(other), "", "", 404)
	request(2, "GET", "/api/v1/customer/bookings/"+uuid(legacy), "", "", 404)
	request(2, "GET", "/api/v1/customer/bookings/00000000-0000-0000-0000-000000000000", "", "", 404)
	request(2, "GET", ownPath+"?customer_id="+fmt.Sprint(users[3]), "", "", 400)
	request(2, "GET", ownPath, "", "https://foreign.invalid", 403)
	request(2, "POST", ownPath, "{}", origin, 405)
	request(2, "GET", "/api/v1/businesses/"+uuid(a)+"/bookings/"+uuid(owned), "", "", 403)
	request(1, "GET", "/api/v1/businesses/"+uuid(b)+"/bookings/"+uuid(owned), "", "", 404)
	request(0, "GET", "/api/v1/businesses/"+uuid(a)+"/bookings/"+uuid(owned), "", "", 200)
	publication := "/api/v1/businesses/" + uuid(a) + "/publication"
	publicPath := "/api/v1/public/businesses/" + uuid(a)
	request(-1, "GET", publicPath, "", "", 404)
	request(-1, "GET", publicPath+"?include=memberships", "", "", 400)
	request(-1, "GET", "/api/v1/public/businesses/not-a-uuid", "", "", 400)
	request(-1, "POST", publicPath, "{}", origin, 405)
	request(-1, "GET", publicPath, "", "https://foreign.invalid", 403)
	request(-1, "OPTIONS", publicPath, "", origin, 204)

	// Foundation flags cannot expose a profile without a newly reviewed snapshot.
	if _, e := pool.Exec(ctx, "UPDATE businesses SET is_published=true WHERE id=$1", a); e != nil {
		t.Fatal(e)
	}
	if _, e := tenants.PublicProfile(ctx, a); !errors.Is(e, service.ErrPublicProfileNotFound) {
		t.Fatal("legacy flag exposed profile", e)
	}
	var preview service.Publication
	if e := json.Unmarshal(request(0, "GET", publication, "", "", 200).Body.Bytes(), &preview); e != nil {
		t.Fatal(e)
	}
	reviewedBody, _ := json.Marshal(map[string]any{"published": true, "review_token": preview.ReviewToken})
	request(0, "PUT", publication, `{"published":true}`, origin, 400)
	request(0, "PUT", publication, `{"published":true,"review_token":"stale"}`, origin, 409)

	if !strings.Contains(request(0, "GET", publication, "", "", 200).Body.String(), `"published":false`) {
		t.Fatal("business published by default")
	}
	request(-1, "PUT", publication, `{"published":true}`, origin, 401)
	request(1, "PUT", publication, `{"published":true}`, origin, 403)
	request(0, "PUT", publication, `{"published":true}`, "https://foreign.invalid", 403)
	for _, body := range []string{`{}`, `{"published":null}`, `{"published":"true"}`, `{"published":true,"extra":1}`} {
		request(0, "PUT", publication, body, origin, 400)
	}
	if e := tenants.ChangeMember(ctx, users[0], a, users[3], service.RoleStaff, false); e != nil {
		t.Fatal(e)
	}
	request(3, "GET", publication, "", "", 200)
	request(3, "PUT", publication, `{"published":true}`, origin, 403)
	if e := tenants.ChangeMember(ctx, users[0], a, users[3], service.RoleBusinessAdmin, false); e != nil {
		t.Fatal(e)
	}
	request(3, "PUT", publication, string(reviewedBody), origin, 200)
	publicResponse := request(-1, "GET", publicPath, "", "", 200)
	if publicResponse.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("public profile cached")
	}
	var publicFields map[string]any
	if e := json.Unmarshal(publicResponse.Body.Bytes(), &publicFields); e != nil {
		t.Fatal(e)
	}
	allowed := map[string]bool{"id": true, "name": true, "timezone": true, "tagline": true, "description": true, "contact_email": true, "contact_phone": true, "website": true, "services": true}
	if len(publicFields) != len(allowed) {
		t.Fatal("unexpected public projection", publicFields)
	}
	for field := range publicFields {
		if !allowed[field] {
			t.Fatal("private field exposed", field)
		}
	}
	// Pre-release prepared snapshots need fresh explicit confirmation.
	if _, e := pool.Exec(ctx, "UPDATE business_public_profiles SET public_confirmed=false WHERE business_id=$1", a); e != nil {
		t.Fatal(e)
	}
	request(-1, "GET", publicPath, "", "", 404)
	request(0, "PUT", publication, string(reviewedBody), origin, 200)
	request(-1, "GET", publicPath, "", "", 200)

	if profile, e := tenants.PublicProfile(ctx, a); e != nil || profile.ID != a || profile.Name != preview.Profile.Name {
		t.Fatal("reviewed profile missing", profile, e)
	}
	t.Run("customer booking creation", func(t *testing.T) {
		verifyCustomerBookingCreate(t, publicAvailabilityFixture{ctx: ctx, pool: pool, tenants: tenants, catalog: catalog, bookings: bookings, actor: users[0], business: a, foreign: b, serviceID: offered.ID, location: loc.ID, staff: staff.ID, day: day, request: request}, mux, users, tokens)
	})
	t.Run("public availability", func(t *testing.T) {
		verifyPublicAvailability(t, publicAvailabilityFixture{ctx: ctx, pool: pool, tenants: tenants, catalog: catalog, bookings: bookings, actor: users[0], business: a, foreign: b, serviceID: offered.ID, location: loc.ID, staff: staff.ID, day: day, request: request})
	})
	t.Run("booking ranges", func(t *testing.T) {
		verifyBookingRanges(t, publicAvailabilityFixture{ctx: ctx, pool: pool, tenants: tenants, catalog: catalog, bookings: bookings, actor: users[0], business: a, foreign: b, day: day, request: request}, users[2])
	})
	t.Run("customer cancellation", func(t *testing.T) {
		verifyCustomerCancellation(t, publicAvailabilityFixture{ctx: ctx, pool: pool, tenants: tenants, catalog: catalog, bookings: bookings, actor: users[0], business: a, foreign: b, day: day, request: request}, mux, users, tokens)
		verifyCustomerRescheduling(t, publicAvailabilityFixture{ctx: ctx, pool: pool, tenants: tenants, catalog: catalog, bookings: bookings, actor: users[0], business: a, foreign: b, day: day, request: request}, mux, users, tokens)
	})
	// Settings changes stay private until a fresh preview is confirmed.
	if _, e := tenants.SaveSettings(ctx, users[0], a, service.BusinessSettings{Description: "Private draft update", ContactEmail: "business@example.invalid"}); e != nil {
		t.Fatal(e)
	}
	if profile, e := tenants.PublicProfile(ctx, a); e != nil || profile.Description != "" || profile.ContactEmail != "" {
		t.Fatal("draft leaked into snapshot", profile, e)
	}
	if strings.Contains(request(-1, "GET", publicPath, "", "", 200).Body.String(), "Private draft update") {
		t.Fatal("anonymous draft leak")
	}
	request(0, "PUT", publication, string(reviewedBody), origin, 409)
	if e := json.Unmarshal(request(0, "GET", publication, "", "", 200).Body.Bytes(), &preview); e != nil {
		t.Fatal(e)
	}
	reviewedBody, _ = json.Marshal(map[string]any{"published": true, "review_token": preview.ReviewToken})
	publish := true
	failing := service.NewBusinessService(settingsRollback{repository.NewBusinessTransactions(pool)})
	if result, e := failing.Publication(ctx, users[0], a, &publish, preview.ReviewToken); e == nil || result.Published || result.Profile.ID.Valid {
		t.Fatal("failed transaction exposed publication", result, e)
	}
	if profile, e := tenants.PublicProfile(ctx, a); e != nil || profile.Description != "" {
		t.Fatal("failed publication changed snapshot", profile, e)
	}

	request(0, "PUT", publication, string(reviewedBody), origin, 200)
	if profile, e := tenants.PublicProfile(ctx, a); e != nil || profile.Description != "Private draft update" {
		t.Fatal("reviewed update missing", profile, e)
	}
	// Tenant identity is part of the reviewed hash.
	request(1, "PUT", "/api/v1/businesses/"+uuid(b)+"/publication", string(reviewedBody), origin, 409)

	if e := tenants.ChangeMember(ctx, users[0], a, users[3], "", true); e != nil {
		t.Fatal(e)
	}
	request(3, "PUT", publication, `{"published":false}`, origin, 403)
	request(0, "PUT", publication, `{"published":false}`, origin, 200)
	if _, e := tenants.PublicProfile(ctx, a); !errors.Is(e, service.ErrPublicProfileNotFound) {
		t.Fatal("unpublished profile readable", e)
	}

	// Unpublished profiles are unavailable to anonymous visitors.
	request(-1, "GET", "/api/v1/public/businesses/"+uuid(a), "", "", 404)
	// No UPDATE can reassign ownership, including an otherwise valid close.
	if _, e := pool.Exec(ctx, "UPDATE bookings SET status='cancelled',closed_by=$2,closed_at=now(),customer_id=$3 WHERE id=$1", owned, users[0], users[3]); e == nil {
		t.Fatal("ownership reassignment accepted")
	}
	moved, e := bookings.Reschedule(ctx, users[0], a, owned, "customer-move", day.Add(14*time.Hour))
	if e != nil || moved.Replacement == nil {
		t.Fatal(moved, e)
	}
	request(2, "GET", ownPath, "", "", 200)
	newPath := "/api/v1/customer/bookings/" + uuid(moved.Replacement.ID)
	request(2, "GET", newPath, "", "", 200)
	request(3, "GET", newPath, "", "", 404)
	if _, e := bookings.Cancel(ctx, users[0], a, moved.Replacement.ID, "customer-cancel"); e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(request(2, "GET", newPath, "", "", 200).Body.String(), `"status":"cancelled"`) {
		t.Fatal("customer did not see business cancellation")
	}
	// Direct insertion cannot detach a replacement from its customer's history.
	tx, e := pool.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = tx.Exec(ctx, "UPDATE bookings SET status='rescheduled',closed_by=$2,closed_at=now() WHERE id=$1", other, users[0]); e != nil {
		t.Fatal(e)
	}
	_, e = tx.Exec(ctx, `INSERT INTO bookings(business_id,service_id,location_id,staff_id,created_by,idempotency_key,request_hash,starts_at,ends_at,busy_starts_at,busy_ends_at,supersedes_booking_id,customer_id) SELECT business_id,service_id,location_id,staff_id,created_by,'wrong-owner',request_hash,starts_at,ends_at,busy_starts_at,busy_ends_at,id,NULL FROM bookings WHERE id=$1`, other)
	if e == nil {
		t.Error("replacement lost customer")
	}
	if e := tx.Rollback(ctx); e != nil {
		t.Fatal(e)
	}
	if _, e := pool.Exec(ctx, "UPDATE businesses SET status='disabled' WHERE id=$1", a); e != nil {
		t.Fatal(e)
	}
	request(0, "PUT", publication, `{"published":true}`, origin, 403)
	if _, e := pool.Exec(ctx, "UPDATE businesses SET is_published=true WHERE id=$1", a); e != nil {
		t.Fatal(e)
	}
	if _, e := tenants.PublicProfile(ctx, a); !errors.Is(e, service.ErrPublicProfileNotFound) {
		t.Fatal("disabled business exposed", e)
	}
	request(-1, "GET", publicPath, "", "", 404)

	request(2, "GET", ownPath, "", "", 200) // Historical ownership survives unpublishing/disablement.

	t.Run("customer history", func(t *testing.T) {
		path := "/api/v1/customer/bookings?limit=1"
		first := request(2, "GET", path, "", origin, 200)
		if first.Header().Get("Cache-Control") != "no-store" || first.Header().Get("Access-Control-Expose-Headers") != "X-Next-Cursor" {
			t.Fatal("history cache/CORS contract")
		}
		var one, two []service.CustomerBooking
		if e := json.Unmarshal(first.Body.Bytes(), &one); e != nil || len(one) != 1 {
			t.Fatal("first page", one, e)
		}
		cursor := first.Header().Get("X-Next-Cursor")
		if cursor == "" {
			t.Fatal("missing next cursor")
		}
		second := request(2, "GET", path+"&cursor="+neturl.QueryEscape(cursor), "", "", 200)
		if e := json.Unmarshal(second.Body.Bytes(), &two); e != nil || len(two) != 1 || two[0].ID == one[0].ID || second.Header().Get("X-Next-Cursor") != "" {
			t.Fatal("second page", two, e)
		}
		if one[0].ID != moved.Replacement.ID || two[0].ID != owned || one[0].Status != "cancelled" || two[0].Status != "rescheduled" {
			t.Fatal("order/lifecycle history incorrect", one, two)
		}
		if one[0].SupersedesID != owned || two[0].ReplacementID != moved.Replacement.ID {
			t.Fatal("chain missing")
		}
		for _, field := range []string{"staff_id", "resource_id", "customer_id", "created_by", "request_hash", "idempotency_key"} {
			if strings.Contains(first.Body.String(), field) {
				t.Fatal("private history field", field)
			}
		}
		empty := request(0, "GET", "/api/v1/customer/bookings", "", "", 200)
		if strings.TrimSpace(empty.Body.String()) != "[]" {
			t.Fatal("operator saw customer history", empty.Body.String())
		}
		request(3, "GET", path+"&cursor="+neturl.QueryEscape(cursor), "", "", 400)
		for _, query := range []string{"?customer_id=1", "?limit=1&limit=2", "?cursor=bad", "?limit=101", "?limit=0", "?unknown=1"} {
			request(2, "GET", "/api/v1/customer/bookings"+query, "", "", 400)
		}
		request(-1, "GET", path, "", "", 401)
		request(2, "GET", path, "", "https://foreign.invalid", 403)
		failed := service.NewBusinessService(settingsRollback{repository.NewBusinessTransactions(pool)})
		if result, e := failed.CustomerBookings(ctx, users[2], service.PageRequest{}); e == nil || len(result.Items) > 0 {
			t.Fatal("failed transaction leaked history", result, e)
		}
	})
	if _, e := pool.Exec(ctx, "UPDATE users SET status='disabled' WHERE id=$1", users[2]); e != nil {
		t.Fatal(e)
	}
	request(2, "GET", ownPath, "", "", 401)
	request(2, "GET", "/api/v1/customer/bookings", "", "", 401)

	t.Run("public discovery and service snapshots", func(t *testing.T) {
		marker := fmt.Sprintf("Discovery-%d", users[0])
		ids := []pgtype.UUID{}
		serviceIDs := []pgtype.UUID{}
		for i := 0; i < 4; i++ {
			var discoveryOwner int64
			if e := pool.QueryRow(ctx, "INSERT INTO users(email,username) VALUES($1,'Discovery owner') RETURNING id", fmt.Sprintf("discovery-owner-%d-%d@example.invalid", time.Now().UnixNano(), i)).Scan(&discoveryOwner); e != nil {
				t.Fatal(e)
			}
			users = append(users, discoveryOwner)
			name := fmt.Sprintf("%s %d", marker, i)
			if i == 0 {
				name = marker + "% literal"
			}
			business, e := tenants.Create(ctx, discoveryOwner, service.BusinessInput{Name: name, Timezone: "UTC"})
			if e != nil {
				t.Fatal(e)
			}
			businesses = append(businesses, business.ID)
			ids = append(ids, business.ID)
			// The original test actor can review each fixture without owning them all.
			if _, e := pool.Exec(ctx, "INSERT INTO business_memberships(business_id,user_id,role) VALUES($1,$2,'BUSINESS_ADMIN')", business.ID, users[0]); e != nil {
				t.Fatal(e)
			}
			offered, e := catalog.CreateService(ctx, discoveryOwner, business.ID, service.ServiceInput{Name: "Reviewed service", DurationMinutes: 30})
			if e != nil {
				t.Fatal(e)
			}
			serviceIDs = append(serviceIDs, offered.ID)
			p, e := tenants.Publication(ctx, discoveryOwner, business.ID, nil, "")
			if e != nil {
				t.Fatal(e)
			}
			yes := true
			if _, e := tenants.Publication(ctx, discoveryOwner, business.ID, &yes, p.ReviewToken); e != nil {
				t.Fatal(e)
			}
		}
		// An older published snapshot remains link-accessible, but is not silently listed.
		if _, e := pool.Exec(ctx, "UPDATE business_public_profiles SET profile=profile-'services' WHERE business_id=$1", ids[3]); e != nil {
			t.Fatal(e)
		}
		listPath := "/api/v1/public/businesses?q=" + neturl.QueryEscape(marker)
		var firstCursor string
		seen := map[pgtype.UUID]bool{}
		cursor := ""
		for i := 0; i < 3; i++ {
			path := listPath + "&limit=1"
			if cursor != "" {
				path += "&cursor=" + neturl.QueryEscape(cursor)
			}
			w := request(-1, "GET", path, "", origin, 200)
			if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Access-Control-Expose-Headers") != "X-Next-Cursor" {
				t.Fatal("missing public pagination headers")
			}
			var items []service.PublicBusinessSummary
			if e := json.Unmarshal(w.Body.Bytes(), &items); e != nil || len(items) != 1 {
				t.Fatal("pagination", items, e)
			}
			if seen[items[0].ID] {
				t.Fatal("duplicate page")
			}
			seen[items[0].ID] = true
			cursor = w.Header().Get("X-Next-Cursor")
			if i == 0 {
				firstCursor = cursor
			}
		}
		if cursor != "" || len(seen) != 3 || seen[ids[3]] {
			t.Fatal("wrong directory boundary", seen, cursor)
		}
		request(-1, "GET", listPath+"changed&cursor="+neturl.QueryEscape(firstCursor), "", "", 400)
		for _, suffix := range []string{"&q=duplicate", "&limit=0", "&limit=101", "&cursor=bad", "&unknown=true"} {
			request(-1, "GET", listPath+suffix, "", "", 400)
		}
		request(-1, "GET", "/api/v1/public/businesses?q="+strings.Repeat("x", 101), "", "", 400)
		literal := request(-1, "GET", "/api/v1/public/businesses?q="+neturl.QueryEscape(marker+"%"), "", "", 200)
		var literalItems []service.PublicBusinessSummary
		if e := json.Unmarshal(literal.Body.Bytes(), &literalItems); e != nil || len(literalItems) != 1 || literalItems[0].ID != ids[0] {
			t.Fatal("search treated percent as wildcard", literal.Body.String(), e)
		}
		profilePath := "/api/v1/public/businesses/" + uuid(ids[0])
		var public service.PublicBusinessProfile
		get := func() {
			t.Helper()
			if e := json.Unmarshal(request(-1, "GET", profilePath, "", "", 200).Body.Bytes(), &public); e != nil {
				t.Fatal(e)
			}
		}
		get()
		if len(public.Services) != 1 || public.Services[0].Name != "Reviewed service" {
			t.Fatal(public)
		}
		preview, e := tenants.Publication(ctx, users[0], ids[0], nil, "")
		if e != nil {
			t.Fatal(e)
		}
		if _, e := catalog.UpdateService(ctx, users[0], ids[0], serviceIDs[0], service.ServiceInput{Name: "Unpublished edit", DurationMinutes: 45}); e != nil {
			t.Fatal(e)
		}
		get()
		if public.Services[0].Name != "Reviewed service" || public.Services[0].DurationMinutes != 30 {
			t.Fatal("draft service leaked")
		}
		stale, _ := json.Marshal(map[string]any{"published": true, "review_token": preview.ReviewToken})
		request(0, "PUT", "/api/v1/businesses/"+uuid(ids[0])+"/publication", string(stale), origin, 409)
		preview, e = tenants.Publication(ctx, users[0], ids[0], nil, "")
		if e != nil {
			t.Fatal(e)
		}
		yes := true
		if _, e := tenants.Publication(ctx, users[0], ids[0], &yes, preview.ReviewToken); e != nil {
			t.Fatal(e)
		}
		get()
		if public.Services[0].Name != "Unpublished edit" || public.Services[0].DurationMinutes != 45 {
			t.Fatal("republished service missing")
		}
		if _, e := pool.Exec(ctx, "UPDATE services SET status='archived' WHERE id=$1", serviceIDs[0]); e != nil {
			t.Fatal(e)
		}
		get()
		if len(public.Services) != 0 {
			t.Fatal("archived service visible")
		}
		// Even a corrupt snapshot cannot use another tenant's active service.
		if _, e := pool.Exec(ctx, "UPDATE business_public_profiles SET profile=jsonb_set(profile,'{services,0,id}',to_jsonb($2::text)) WHERE business_id=$1", ids[0], uuid(serviceIDs[1])); e != nil {
			t.Fatal(e)
		}
		get()
		if len(public.Services) != 0 {
			t.Fatal("foreign service visible")
		}
		no := false
		if _, e := tenants.Publication(ctx, users[0], ids[1], &no, ""); e != nil {
			t.Fatal(e)
		}
		if _, e := pool.Exec(ctx, "UPDATE businesses SET status='disabled' WHERE id=$1", ids[2]); e != nil {
			t.Fatal(e)
		}
		var remaining []service.PublicBusinessSummary
		if e := json.Unmarshal(request(-1, "GET", listPath, "", "", 200).Body.Bytes(), &remaining); e != nil || len(remaining) != 1 || remaining[0].ID != ids[0] {
			t.Fatal("hidden businesses listed", remaining, e)
		}
		// Publication bounds do not prevent taking a business offline.
		if _, e := pool.Exec(ctx, "INSERT INTO services(business_id,name,duration_minutes) SELECT $1,'Limit fixture',30 FROM generate_series(1,1001)", ids[0]); e != nil {
			t.Fatal(e)
		}
		request(0, "GET", "/api/v1/businesses/"+uuid(ids[0])+"/publication", "", "", 400)
		request(0, "PUT", "/api/v1/businesses/"+uuid(ids[0])+"/publication", `{"published":false}`, origin, 200)
		request(-1, "GET", profilePath, "", "", 404)
	})
	// A down migration must not silently discard ownership.
	down, e := os.ReadFile("../../../../migrations/000017_customer_foundation.down.sql")
	if e != nil {
		t.Fatal(e)
	}
	tx, e = pool.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if _, e := tx.Exec(ctx, string(down)); e == nil {
		t.Error("downgrade discarded ownership")
	}
	_ = tx.Rollback(ctx)
}
