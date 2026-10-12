package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"multipurpose-booking-system/api/internal/db"
	"multipurpose-booking-system/api/internal/service"
	"os"
	"testing"
	"time"
)

func TestBookingCatalogIntegration(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
			for _, table := range []string{"resources", "staff_profiles", "services", "locations"} {
				if _, e := pool.Exec(c, "DELETE FROM "+table+" WHERE business_id=$1", id); e != nil {
					t.Error(e)
				}
			}
			if _, e := pool.Exec(c, "DELETE FROM businesses WHERE id=$1", id); e != nil {
				t.Error(e)
			}
		}
		for _, id := range users {
			if _, e := pool.Exec(c, "DELETE FROM user_platform_roles WHERE user_id=$1", id); e != nil {
				t.Error(e)
			}
			if _, e := pool.Exec(c, "DELETE FROM users WHERE id=$1", id); e != nil {
				t.Error(e)
			}
		}
	}()
	for i := 0; i < 4; i++ {
		var id int64
		if err := pool.QueryRow(ctx, "INSERT INTO users(email,username) VALUES($1,'catalog fixture') RETURNING id", fmt.Sprintf("catalog-%d-%d@example.invalid", time.Now().UnixNano(), i)).Scan(&id); err != nil {
			t.Fatal(err)
		}
		users = append(users, id)
	}
	tenants := service.NewBusinessService(NewBusinessTransactions(pool))
	catalog := service.NewBookingCatalogService(tenants)
	for i := 0; i < 2; i++ {
		b, e := tenants.Create(ctx, users[i], service.BusinessInput{Name: "Catalog fixture", Timezone: "UTC"})
		if e != nil {
			t.Fatal(e)
		}
		businesses = append(businesses, b.ID)
	}
	a, b := businesses[0], businesses[1]
	location, err := catalog.CreateLocation(ctx, users[0], a, service.LocationInput{Name: " Main ", Timezone: "Asia/Shanghai"})
	if err != nil || location.Name != "Main" || location.BusinessID != a {
		t.Fatal(location, err)
	}
	offered, err := catalog.CreateService(ctx, users[0], a, service.ServiceInput{Name: "Consultation", DurationMinutes: 30})
	if err != nil {
		t.Fatal(err)
	}
	staff, err := catalog.CreateStaffProfile(ctx, users[0], a, service.StaffProfileInput{Name: "Alex"})
	if err != nil {
		t.Fatal(err)
	}
	resource, err := catalog.CreateResource(ctx, users[0], a, service.ResourceInput{Name: "Room", LocationID: location.ID, Capacity: 2})
	if err != nil || resource.BusinessID != a || resource.LocationID != location.ID {
		t.Fatal(resource, err)
	}
	var memberships int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM business_memberships WHERE business_id=$1", a).Scan(&memberships); err != nil || memberships != 1 {
		t.Fatal("staff profile granted membership", memberships, err)
	}
	reads := []struct {
		name string
		read func(int64, pgtype.UUID) error
	}{
		{"location", func(actor int64, business pgtype.UUID) error {
			_, e := catalog.GetLocation(ctx, actor, business, location.ID)
			return e
		}},
		{"service", func(actor int64, business pgtype.UUID) error {
			_, e := catalog.GetService(ctx, actor, business, offered.ID)
			return e
		}},
		{"staff", func(actor int64, business pgtype.UUID) error {
			_, e := catalog.GetStaffProfile(ctx, actor, business, staff.ID)
			return e
		}},
		{"resource", func(actor int64, business pgtype.UUID) error {
			_, e := catalog.GetResource(ctx, actor, business, resource.ID)
			return e
		}},
	}
	for _, check := range reads {
		if e := check.read(users[0], a); e != nil {
			t.Fatal(check.name, e)
		}
		if e := check.read(users[1], a); e != service.ErrForbidden {
			t.Fatal("foreign tenant", check.name, e)
		}
		if e := check.read(users[1], b); e != service.ErrCatalogNotFound {
			t.Fatal("foreign resource", check.name, e)
		}
	}
	if _, err := pool.Exec(ctx, "INSERT INTO user_platform_roles(user_id,role) VALUES($1,'PLATFORM_ADMIN')", users[3]); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.GetLocation(ctx, users[3], a, location.ID); err != service.ErrForbidden {
		t.Fatal("platform bypass", err)
	}
	for _, role := range []service.Role{service.RoleStaff, service.RoleManager, service.RoleBusinessAdmin} {
		if err := tenants.ChangeMember(ctx, users[0], a, users[2], role, false); err != nil {
			t.Fatal(err)
		}
		for _, check := range reads {
			if e := check.read(users[2], a); e != nil {
				t.Fatal(role, check.name, e)
			}
		}
		_, err := catalog.CreateService(ctx, users[2], a, service.ServiceInput{Name: "Role check", DurationMinutes: 1})
		if role == service.RoleBusinessAdmin {
			if err != nil {
				t.Fatal(err)
			}
		} else if err != service.ErrForbidden {
			t.Fatal("unauthorized catalog write", role, err)
		}
	}
	if err := tenants.ChangeMember(ctx, users[0], a, users[2], "", true); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.GetService(ctx, users[2], a, offered.ID); err != service.ErrForbidden {
		t.Fatal("revoked member", err)
	}
	if _, err := catalog.CreateResource(ctx, users[1], b, service.ResourceInput{Name: "Cross-tenant", LocationID: location.ID, Capacity: 1}); err != service.ErrCatalogNotFound {
		t.Fatal("cross-tenant location accepted", err)
	}
	assertDBError := func(code, sql string, args ...any) {
		t.Helper()
		_, err := pool.Exec(ctx, sql, args...)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != code {
			t.Fatalf("wanted database error %s got %v", code, err)
		}
	}
	assertDBError("23503", "INSERT INTO resources(business_id,location_id,name,capacity) VALUES($1,$2,'wrong tenant',1)", b, location.ID)
	assertDBError("23514", "INSERT INTO resources(business_id,location_id,name,capacity) VALUES($1,$2,'zero capacity',0)", a, location.ID)
	assertDBError("23514", "INSERT INTO services(business_id,name,duration_minutes) VALUES($1,'zero duration',0)", a)
	assertDBError("23514", "INSERT INTO services(business_id,name,duration_minutes) VALUES($1,'too long',10081)", a)
	assertDBError("23503", "DELETE FROM locations WHERE id=$1", location.ID)
	assertDBError("23503", "DELETE FROM businesses WHERE id=$1", a)
	rollback := errors.New("intentional rollback")
	err = tenants.WithinTenant(ctx, users[0], a, []service.Role{service.RoleBusinessOwner}, func(q service.BusinessStore, scope service.TenantAccess) error {
		if _, e := q.CreateService(ctx, db.CreateServiceParams{BusinessID: scope.BusinessID(), Name: "Rolled back", DurationMinutes: 10}); e != nil {
			return e
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatal(err)
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM services WHERE business_id=$1 AND name='Rolled back'", a).Scan(&count); err != nil || count != 0 {
		t.Fatal("catalog write did not roll back", count, err)
	}
	if _, err := pool.Exec(ctx, "UPDATE locations SET status='archived',updated_at=NOW() WHERE id=$1", location.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.GetResource(ctx, users[0], a, resource.ID); err != service.ErrCatalogNotFound {
		t.Fatal("resource at archived location visible", err)
	}
	if _, err := catalog.CreateResource(ctx, users[0], a, service.ResourceInput{Name: "Archived location", LocationID: location.ID, Capacity: 1}); err != service.ErrCatalogNotFound {
		t.Fatal("archived location accepted", err)
	}
	if _, err := pool.Exec(ctx, "UPDATE businesses SET status='disabled' WHERE id=$1", a); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.GetService(ctx, users[0], a, offered.ID); err != service.ErrForbidden {
		t.Fatal("disabled business", err)
	}
	if _, err := pool.Exec(ctx, "UPDATE users SET status='disabled' WHERE id=$1", users[1]); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.CreateLocation(ctx, users[1], b, service.LocationInput{Name: "Disabled owner", Timezone: "UTC"}); err != service.ErrForbidden {
		t.Fatal("disabled account", err)
	}
}

// Exercise up/down/up only inside an isolated rollback-only schema. Never roll
// back the live development schema or remove existing application data.
func TestBookingCatalogMigrationRoundTrip(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	schema := fmt.Sprintf("catalog_migration_%d", time.Now().UnixNano())
	if _, err := tx.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, "SET LOCAL search_path TO "+schema+",public"); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, "CREATE TABLE businesses(id UUID PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	up, err := os.ReadFile("../../../../migrations/000009_booking_catalog.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../../../migrations/000009_booking_catalog.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range []struct {
		sql    []byte
		exists bool
	}{{up, true}, {down, false}, {up, true}} {
		if _, err := tx.Exec(ctx, string(step.sql)); err != nil {
			t.Fatal(err)
		}
		for _, table := range []string{"locations", "services", "staff_profiles", "resources"} {
			var exists bool
			if err := tx.QueryRow(ctx, "SELECT to_regclass($1) IS NOT NULL", schema+"."+table).Scan(&exists); err != nil || exists != step.exists {
				t.Fatal(table, exists, err)
			}
		}
	}
	// The eligibility migration depends on the catalog schema restored above.
	eligibilityUp, err := os.ReadFile("../../../../migrations/000010_catalog_eligibility.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	eligibilityDown, err := os.ReadFile("../../../../migrations/000010_catalog_eligibility.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range []struct {
		sql    []byte
		exists bool
	}{{eligibilityUp, true}, {eligibilityDown, false}, {eligibilityUp, true}} {
		if _, err := tx.Exec(ctx, string(step.sql)); err != nil {
			t.Fatal(err)
		}
		for _, table := range []string{"service_locations", "service_staff", "service_resources", "staff_locations"} {
			var exists bool
			if err := tx.QueryRow(ctx, "SELECT to_regclass($1) IS NOT NULL", schema+"."+table).Scan(&exists); err != nil || exists != step.exists {
				t.Fatal(table, exists, err)
			}
		}
	}
	schedulingUp, err := os.ReadFile("../../../../migrations/000011_scheduling_rules.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	schedulingDown, err := os.ReadFile("../../../../migrations/000011_scheduling_rules.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range []struct {
		sql    []byte
		exists bool
	}{{schedulingUp, true}, {schedulingDown, false}, {schedulingUp, true}} {
		if _, err := tx.Exec(ctx, string(step.sql)); err != nil {
			t.Fatal(err)
		}
		for _, table := range []string{"location_weekly_hours", "location_closed_dates", "service_scheduling_rules"} {
			var exists bool
			if err := tx.QueryRow(ctx, "SELECT to_regclass($1) IS NOT NULL", schema+"."+table).Scan(&exists); err != nil || exists != step.exists {
				t.Fatal(table, exists, err)
			}
		}
	}
	availabilityUp, err := os.ReadFile("../../../../migrations/000012_availability_windows.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	availabilityDown, err := os.ReadFile("../../../../migrations/000012_availability_windows.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range []struct {
		sql    []byte
		exists bool
	}{{availabilityUp, true}, {availabilityDown, false}, {availabilityUp, true}} {
		if _, err := tx.Exec(ctx, string(step.sql)); err != nil {
			t.Fatal(err)
		}
		for _, table := range []string{"staff_availability_windows", "resource_availability_windows"} {
			var exists bool
			if err := tx.QueryRow(ctx, "SELECT to_regclass($1) IS NOT NULL", schema+"."+table).Scan(&exists); err != nil || exists != step.exists {
				t.Fatal(table, exists, err)
			}
		}
	}
	bookingUp, err := os.ReadFile("../../../../migrations/000013_bookings.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	bookingDown, err := os.ReadFile("../../../../migrations/000013_bookings.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range []struct {
		sql    []byte
		exists bool
	}{{bookingUp, true}, {bookingDown, false}, {bookingUp, true}} {
		if _, err := tx.Exec(ctx, string(step.sql)); err != nil {
			t.Fatal(err)
		}
		var exists bool
		if err := tx.QueryRow(ctx, "SELECT to_regclass($1) IS NOT NULL", schema+".bookings").Scan(&exists); err != nil || exists != step.exists {
			t.Fatal(exists, err)
		}
	}

}
