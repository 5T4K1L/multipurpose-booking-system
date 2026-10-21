package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"multipurpose-booking-system/api/internal/repository"
	"multipurpose-booking-system/api/internal/service"
)

func TestCatalogHTTPIntegration(t *testing.T) {
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
			for _, table := range []string{"service_locations", "service_staff", "service_resources", "staff_locations", "resources", "staff_profiles", "services", "locations"} {
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
	for i := 0; i < 4; i++ {
		var id int64
		if err := pool.QueryRow(ctx, "INSERT INTO users(email,username) VALUES($1,'catalog HTTP fixture') RETURNING id", fmt.Sprintf("catalog-http-%d-%d@example.invalid", time.Now().UnixNano(), i)).Scan(&id); err != nil {
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
		b, e := tenants.Create(ctx, users[i], service.BusinessInput{Name: "HTTP fixture", Timezone: "UTC"})
		if e != nil {
			t.Fatal(e)
		}
		businesses = append(businesses, b.ID)
	}
	h, err := NewCatalogHandler(sessions, catalog, "http://localhost:3000")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	RegisterCatalogRoutes(mux, h)
	uuid := func(id pgtype.UUID) string { v, _ := id.MarshalJSON(); return string(v[1 : len(v)-1]) }
	base := "/api/v1/businesses/" + uuid(businesses[0]) + "/catalog/"
	otherBase := "/api/v1/businesses/" + uuid(businesses[1]) + "/catalog/"
	request := func(actor int, method, path, body, origin, media string, want int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body)).WithContext(ctx)
		if actor >= 0 {
			r.AddCookie(&http.Cookie{Name: "booking_session", Value: tokens[actor]})
		}
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		if media != "" {
			r.Header.Set("Content-Type", media)
		}
		r.Header.Set("X-User-ID", fmt.Sprint(users[0]))
		r.Header.Set("X-Role", "BUSINESS_OWNER")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("%s %s actor %d got %d want %d: %s", method, path, actor, w.Code, want, w.Body.String())
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("missing no-store")
		}
		return w
	}
	const origin = "http://localhost:3000"
	get := func(actor int, path string, want int) *httptest.ResponseRecorder {
		return request(actor, "GET", path, "", origin, "", want)
	}
	mutate := func(actor int, method, path, body string, want int) *httptest.ResponseRecorder {
		return request(actor, method, path, body, origin, "application/json", want)
	}
	request(-1, "OPTIONS", base+"locations", "", origin, "", 204)
	get(-1, base+"locations", 401)
	request(0, "POST", base+"locations", `{"name":"X","timezone":"UTC"}`, "", "application/json", 403)
	request(0, "GET", base+"locations", "", "https://foreign.invalid", "", 403)
	request(0, "POST", base+"locations", `{}`, origin, "text/plain", 415)
	for _, body := range []string{"null", "[]", `{"name":"X","timezone":"UTC","business_id":"forged"}`, `{} {}`} {
		mutate(0, "POST", base+"locations", body, 400)
	}
	mutate(0, "POST", base+"locations", `{"name":"`+strings.Repeat("x", 5000)+`","timezone":"UTC"}`, 413)
	mutate(0, "POST", base+"locations", `{"name":"X","timezone":"Local"}`, 400)
	get(0, base+"locations?limit=0", 400)
	get(0, base+"locations?cursor=invalid", 400)
	created := map[string]catalogItem{}
	for _, tc := range []struct{ kind, body string }{{"locations", `{"name":"Main","timezone":"Asia/Shanghai"}`}, {"services", `{"name":"Consultation","duration_minutes":30}`}, {"staff", `{"name":"Alex"}`}} {
		w := mutate(0, "POST", base+tc.kind, tc.body, 201)
		var v catalogItem
		if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
			t.Fatal(err)
		}
		created[tc.kind] = v
		if strings.Contains(w.Body.String(), "business_id") || strings.Contains(w.Body.String(), "CreatedAt") {
			t.Fatal("database fields exposed")
		}
	}
	w := mutate(0, "POST", base+"resources", fmt.Sprintf(`{"name":"Room","location_id":"%s","capacity":2}`, uuid(created["locations"].ID)), 201)
	var resource catalogItem
	if err := json.Unmarshal(w.Body.Bytes(), &resource); err != nil {
		t.Fatal(err)
	}
	created["resources"] = resource
	for kind, item := range created {
		get(0, base+kind+"/"+uuid(item.ID), 200)
		get(1, base+kind+"/"+uuid(item.ID), 403)
		get(1, otherBase+kind+"/"+uuid(item.ID), 404)
		get(0, base+kind, 200)
	}
	if _, err := pool.Exec(ctx, "INSERT INTO user_platform_roles(user_id,role) VALUES($1,'PLATFORM_ADMIN')", users[3]); err != nil {
		t.Fatal(err)
	}
	get(3, base+"locations", 403)
	if err := tenants.ChangeMember(ctx, users[0], businesses[0], users[2], service.RoleStaff, false); err != nil {
		t.Fatal(err)
	}
	get(2, base+"services", 200)
	mutate(2, "POST", base+"staff", `{"name":"Denied"}`, 403)
	foreign, err := catalog.CreateLocation(ctx, users[1], businesses[1], service.LocationInput{Name: "Foreign", Timezone: "UTC"})
	if err != nil {
		t.Fatal(err)
	}
	foreignStaff, err := catalog.CreateStaffProfile(ctx, users[1], businesses[1], service.StaffProfileInput{Name: "Foreign"})
	if err != nil {
		t.Fatal(err)
	}
	foreignService, err := catalog.CreateService(ctx, users[1], businesses[1], service.ServiceInput{Name: "Foreign", DurationMinutes: 5})
	if err != nil {
		t.Fatal(err)
	}
	foreignResource, err := catalog.CreateResource(ctx, users[1], businesses[1], service.ResourceInput{Name: "Foreign", LocationID: foreign.ID, Capacity: 1})
	if err != nil {
		t.Fatal(err)
	}
	foreignIDs := map[string]pgtype.UUID{"locations": foreign.ID, "staff": foreignStaff.ID, "services": foreignService.ID, "resources": foreignResource.ID}
	relations := []struct{ kind, relation, table, parentColumn, targetColumn string }{
		{"services", "locations", "service_locations", "service_id", "location_id"},
		{"services", "staff", "service_staff", "service_id", "staff_id"},
		{"services", "resources", "service_resources", "service_id", "resource_id"},
		{"staff", "locations", "staff_locations", "staff_id", "location_id"},
	}
	for _, rel := range relations {
		parent, target := created[rel.kind].ID, created[rel.relation].ID
		path := base + rel.kind + "/" + uuid(parent) + "/" + rel.relation
		link := path + "/" + uuid(target)
		mutate(2, "PUT", link, "", 403)
		mutate(0, "PUT", link, `{}`, 400)
		mutate(0, "PUT", link, "", 204)
		mutate(0, "PUT", link, "", 204)
		w := get(2, path, 200)
		var ids []pgtype.UUID
		if err := json.Unmarshal(w.Body.Bytes(), &ids); err != nil || len(ids) != 1 || ids[0] != target {
			t.Fatal("link response", w.Body.String(), err)
		}
		// PostgreSQL independently rejects either endpoint from a different tenant.
		_, err := pool.Exec(ctx, "INSERT INTO "+rel.table+"(business_id,"+rel.parentColumn+","+rel.targetColumn+") VALUES($1,$2,$3)", businesses[1], parent, target)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23503" {
			t.Fatal("cross-tenant FK", rel.table, err)
		}
		for _, pair := range [][2]pgtype.UUID{{parent, foreignIDs[rel.relation]}, {foreignIDs[rel.kind], target}} {
			_, e := pool.Exec(ctx, "INSERT INTO "+rel.table+"(business_id,"+rel.parentColumn+","+rel.targetColumn+") VALUES($1,$2,$3)", businesses[0], pair[0], pair[1])
			var constraint *pgconn.PgError
			if !errors.As(e, &constraint) || constraint.Code != "23503" {
				t.Fatal("individual foreign endpoint accepted", rel.table, e)
			}
		}
		mutate(0, "DELETE", link, "", 204)
		mutate(0, "DELETE", link, "", 204)
		if body := get(0, path, 200).Body.String(); strings.TrimSpace(body) != "[]" {
			t.Fatal("link remained", body)
		}
		mutate(1, "PUT", link, "", 403)
	}
	// Foreign target in an otherwise authorized business gives the same 404 as missing.
	parentPath := base + "services/" + uuid(created["services"].ID) + "/locations"
	mutate(0, "PUT", parentPath+"/"+uuid(foreign.ID), "", 404)
	if err := tenants.ChangeMember(ctx, users[0], businesses[0], users[2], service.RoleBusinessAdmin, false); err != nil {
		t.Fatal(err)
	}
	mutate(2, "PUT", parentPath+"/"+uuid(created["locations"].ID), "", 204)
	// Archive hides active eligibility; deletion still permits cleanup of that target.
	if _, err := pool.Exec(ctx, "UPDATE locations SET status='archived' WHERE id=$1", created["locations"].ID); err != nil {
		t.Fatal(err)
	}
	get(0, base+"resources/"+uuid(resource.ID), 404)
	if strings.TrimSpace(get(0, parentPath, 200).Body.String()) != "[]" {
		t.Fatal("archived target listed")
	}
	mutate(0, "PUT", parentPath+"/"+uuid(created["locations"].ID), "", 404)
	mutate(0, "DELETE", parentPath+"/"+uuid(created["locations"].ID), "", 204)
	if err := tenants.ChangeMember(ctx, users[0], businesses[0], users[2], "", true); err != nil {
		t.Fatal(err)
	}
	get(2, parentPath, 403)
	// Restore only this fixture for the pagination scenarios below.
	if _, err := pool.Exec(ctx, "UPDATE locations SET status='active' WHERE id=$1", created["locations"].ID); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ table, columns, values string }{
		{"locations", "business_id,name,timezone", "$1,'Page location','UTC'"},
		{"services", "business_id,name,duration_minutes", "$1,'Page service',15"},
		{"staff_profiles", "business_id,name", "$1,'Page staff'"},
	} {
		if _, err := pool.Exec(ctx, "INSERT INTO "+tc.table+"("+tc.columns+") SELECT "+tc.values+" FROM generate_series(1,104)", businesses[0]); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, "INSERT INTO resources(business_id,location_id,name,capacity) SELECT $1,$2,'Page resource',1 FROM generate_series(1,104)", businesses[0], created["locations"].ID); err != nil {
		t.Fatal(err)
	}
	for _, rel := range relations {
		targetTable := rel.relation
		if targetTable == "staff" {
			targetTable = "staff_profiles"
		}
		if _, err := pool.Exec(ctx, "INSERT INTO "+rel.table+"(business_id,"+rel.parentColumn+","+rel.targetColumn+") SELECT $1,$2,id FROM "+targetTable+" WHERE business_id=$1", businesses[0], created[rel.kind].ID); err != nil {
			t.Fatal(err)
		}
	}
	var locationCursor, linkCursor string
	for _, kind := range []string{"locations", "services", "staff", "resources"} {
		path := base + kind
		first := get(0, path, 200)
		var firstItems []catalogItem
		if err := json.Unmarshal(first.Body.Bytes(), &firstItems); err != nil || len(firstItems) != 100 || first.Header().Get("X-Next-Cursor") == "" {
			t.Fatal("default page", kind, err)
		}
		if kind == "locations" {
			locationCursor = first.Header().Get("X-Next-Cursor")
		}
		if first.Header().Get("Access-Control-Expose-Headers") != "X-Next-Cursor" {
			t.Fatal("cursor not exposed to browser")
		}
		seen := map[pgtype.UUID]bool{}
		cursor := ""
		for page := 0; page < 10; page++ {
			query := path + "?limit=17"
			if cursor != "" {
				query += "&cursor=" + cursor
			}
			w := get(0, query, 200)
			var items []catalogItem
			if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
				t.Fatal(err)
			}
			if len(items) > 17 {
				t.Fatal("oversized page")
			}
			for _, item := range items {
				if seen[item.ID] {
					t.Fatal("duplicate page item")
				}
				seen[item.ID] = true
			}
			cursor = w.Header().Get("X-Next-Cursor")
			if cursor == "" {
				break
			}
		}
		if len(seen) != 105 || cursor != "" {
			t.Fatal("catalog pagination truncated", kind, len(seen))
		}
	}
	for _, rel := range relations {
		path := base + rel.kind + "/" + uuid(created[rel.kind].ID) + "/" + rel.relation
		seen := map[pgtype.UUID]bool{}
		cursor := ""
		for page := 0; page < 10; page++ {
			query := path + "?limit=17"
			if cursor != "" {
				query += "&cursor=" + cursor
			}
			w := get(0, query, 200)
			var ids []pgtype.UUID
			if err := json.Unmarshal(w.Body.Bytes(), &ids); err != nil {
				t.Fatal(err)
			}
			for _, id := range ids {
				if seen[id] {
					t.Fatal("duplicate eligibility")
				}
				seen[id] = true
			}
			cursor = w.Header().Get("X-Next-Cursor")
			if rel.table == "service_locations" && page == 0 {
				linkCursor = cursor
			}
			if cursor == "" {
				break
			}
		}
		if len(seen) != 105 || cursor != "" {
			t.Fatal("link pagination truncated", rel.table, len(seen))
		}
	}
	get(0, base+"services?cursor="+locationCursor, 400)
	get(0, parentPath+"?cursor="+locationCursor, 400)
	anotherService, err := catalog.CreateService(ctx, users[0], businesses[0], service.ServiceInput{Name: "Other parent", DurationMinutes: 5})
	if err != nil {
		t.Fatal(err)
	}
	get(0, base+"services/"+uuid(anotherService.ID)+"/locations?cursor="+linkCursor, 400)
	if err := tenants.ChangeMember(ctx, users[0], businesses[0], users[2], service.RoleBusinessAdmin, false); err != nil {
		t.Fatal(err)
	}
	adminCursor := get(2, parentPath+"?limit=1", 200).Header().Get("X-Next-Cursor")
	if err := tenants.ChangeMember(ctx, users[0], businesses[0], users[2], "", true); err != nil {
		t.Fatal(err)
	}
	get(2, parentPath+"?cursor="+adminCursor, 403)
	mutate(0, "DELETE", base+"locations/"+uuid(created["locations"].ID), "", 204)
	// The next phase owns lifecycle mutations; unsupported item updates are rejected.
	mutate(0, "PUT", base+"services/"+uuid(created["services"].ID), `{"name":"Changed"}`, 400)
	if _, err := pool.Exec(ctx, "UPDATE businesses SET status='disabled' WHERE id=$1", businesses[0]); err != nil {
		t.Fatal(err)
	}
	get(0, base+"services", 403)
	if _, err := pool.Exec(ctx, "UPDATE users SET status='disabled' WHERE id=$1", users[1]); err != nil {
		t.Fatal(err)
	}
	get(1, otherBase+"services", 401)
}
