# Phase 5 - Core Booking Engine

## Phase 5.1 - Booking catalog foundation

Completed 2026-10-10. Phase 5 remains in progress.
This milestone supplies database and application-service foundations only. There
are no catalog HTTP endpoints or frontend screens yet, and no booking is created.

### Data model

Migration 000009 adds four tenant-owned tables with UUID IDs, business ownership,
created_at/updated_at timestamps, active/archived status, and unique (business_id,id)
keys for present and future composite relationships:

| Table | Meaning | Additional fields |
|---|---|---|
| locations | Where work or reservations take place | name, explicit IANA timezone |
| services | What the business offers | name, duration_minutes |
| staff_profiles | Bookable people within a business | name |
| resources | Rooms, equipment or other capacity | name, location_id, capacity |

Names are trimmed and must contain 1-100 characters without control characters.
Location timezone must load as a named timezone; UTC is accepted, machine-local
`Local` is rejected. Timezone length is bounded to 100 bytes at the service boundary.
Service duration currently supports 1-10080 minutes (up to seven days); this is a
foundation validation limit, not an implemented multi-day availability guarantee.
Resource capacity is a positive 32-bit integer. Capacity is metadata at this stage;
concurrent capacity allocation and overbooking protection are not yet implemented.

Staff profiles do not create users, grant STAFF roles, or imply login access.
Staff-to-service/location eligibility and optional identity linkage need explicit
later workflows. Services and staff are business-level records; resources require
an active location belonging to that same business.

A composite foreign key enforces (resource.business_id, resource.location_id) against
locations(business_id,id), preventing cross-business links even through direct SQL.
Restrictive foreign keys prevent deleting a business/location underneath its catalog.
Archiving is reserved by the schema; mutation endpoints and lifecycle workflows are
not implemented yet. Future lifecycle writers must use the tenant transaction lock.

### Application service contract

BookingCatalogService exposes typed CreateLocation, CreateService, CreateStaffProfile,
CreateResource and corresponding Get methods. Its narrow BookingCatalogStore is
embedded in the existing transaction store, using sqlc queries in the same transaction.
No independent database connection or unscoped resource lookup is introduced.

All operations use BusinessService.WithinTenant with current active membership:

- OWNER and BUSINESS_ADMIN may create and read catalog records.
- MANAGER and STAFF may read records; catalog writes are denied.
- Platform roles do not bypass tenant membership.
- Both business and resource IDs are required for lookups; missing, archived and
  other-business resources return ErrCatalogNotFound after tenant authorization.
- An archived location also hides its resources and cannot receive new resources.
- Invalid input returns ErrInvalidCatalog; denied membership/policy returns ErrForbidden;
  storage or commit failures return ErrCatalogUnavailable without partial results.

These are internal service results, not public HTTP schemas. Future handlers must
map explicit response fields and errors instead of directly serializing database rows.

### Files and verification

- migrations/000009_booking_catalog.{up,down}.sql
- apps/api/internal/db/queries/booking_catalog.sql, generated bindings/models
- apps/api/internal/service/booking_catalog.go and BusinessStore extension
- unit tests and repository-level PostgreSQL integration tests

Verified: sqlc generation, gofmt, database-enabled `go test ./... -count=1`,
`go build ./...`, and `go vet ./...`.

Tests cover owner/admin vs manager/staff policy, global-admin non-bypass, revoked
membership, disabled accounts/businesses, cross-business reads and foreign keys,
archived locations, invalid duration/capacity, deletion restrictions, rollback and
failed-commit result suppression. Migration up/down/up is tested only in an isolated
rollback-only schema; existing development data is never rolled back for this check.
The additive migration was applied through the existing API startup runner.

No manual configuration is needed. Google remains enabled and Apple disabled.
Earlier Apple, browser QA and race-detector checks remain in DEFERRED_WORK.md.

## Next milestone

Phase 5.2: protected catalog management APIs with bounded paginated reads and
explicit same-business service/staff/location/resource eligibility relationships.
Availability calendars, hours/holidays, buffers and lead time follow before booking
creation, cancellation, idempotency and double-booking/capacity enforcement.
Business dashboards remain Phase 6.
