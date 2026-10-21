# Phase 5 - Core Booking Engine

## Phase 5.1 - Booking catalog foundation

Completed 2026-10-10. Phase 5 remains in progress.
This section records the Phase 5.1 foundation. Phase 5.2 below adds catalog HTTP
endpoints; frontend screens and booking creation remain unimplemented.

### Data model

Migration 000009 adds four tenant-owned tables with UUID IDs, business ownership,
created_at/updated_at timestamps, active/archived status, and unique (business_id,id)
keys for present and future composite relationships:

| Table          | Meaning                               | Additional fields            |
| -------------- | ------------------------------------- | ---------------------------- |
| locations      | Where work or reservations take place | name, explicit IANA timezone |
| services       | What the business offers              | name, duration_minutes       |
| staff_profiles | Bookable people within a business     | name                         |
| resources      | Rooms, equipment or other capacity    | name, location_id, capacity  |

Names are trimmed and must contain 1-100 characters without control characters.
Location timezone must load as a named timezone; UTC is accepted, machine-local
`Local` is rejected. Timezone length is bounded to 100 bytes at the service boundary.
Service duration currently supports 1-10080 minutes (up to seven days); this is a
foundation validation limit, not an implemented multi-day availability guarantee.
Resource capacity is a positive 32-bit integer. Capacity is metadata at this stage;
concurrent capacity allocation and overbooking protection are not yet implemented.

Staff profiles do not create users, grant STAFF roles, or imply login access.
Phase 5.2 adds staff-to-service/location eligibility. Optional identity linkage still
needs an explicit later workflow. Services and staff are business-level records; resources require
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

These are internal service results, not public HTTP schemas. Phase 5.2 handlers map
explicit response fields and errors instead of directly serializing database rows.

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

## Phase 5.2 - Catalog APIs, pagination and eligibility

Completed and verified 2026-10-11. Most implementation arrived in commit afc52db;
completion review preserved it and added HTTP boundary, rate-limit and transaction
failure coverage plus this API contract. Phase 5 remains in progress.

### Catalog endpoints

Base path: `/api/v1/businesses/{businessID}/catalog`.
All endpoints require the existing session cookie and current active membership.
Only OWNER and BUSINESS_ADMIN may mutate; MANAGER and STAFF may read. Global
platform roles have no tenant bypass, and request headers cannot assign an actor/role.

| Method | Suffix       | Request / result                                           |
| ------ | ------------ | ---------------------------------------------------------- |
| POST   | /locations   | `{ "name": "Main", "timezone": "Asia/Shanghai" }`          |
| POST   | /services    | `{ "name": "Consultation", "duration_minutes": 30 }`       |
| POST   | /staff       | `{ "name": "Alex" }`                                       |
| POST   | /resources   | `{ "name": "Room", "location_id": "UUID", "capacity": 2 }` |
| GET    | /{kind}      | Paginated array of active catalog records                  |
| GET    | /{kind}/{id} | One active record in the authorized business               |

`kind` is locations, services, staff or resources. POST returns 201; GET returns 200.
Public fields are id/name, plus timezone for locations, duration_minutes for services,
and location_id/capacity for resources. No database timestamps, internal status or
business_id fields are serialized. Resources at archived locations are hidden.
Item update, delete and archive routes are not implemented and return 405.

### Eligibility endpoints and meaning

The following collection suffixes are supported:

| Collection suffix               | Meaning                                             |
| ------------------------------- | --------------------------------------------------- |
| /services/{serviceID}/locations | Locations eligible to offer a service               |
| /services/{serviceID}/staff     | Staff eligible to deliver a service                 |
| /services/{serviceID}/resources | Resources eligible for a service                    |
| /staff/{staffID}/locations      | Locations where a staff profile is eligible to work |

GET on a collection returns a paginated array of target UUIDs. PUT on
`{collection}/{targetID}` adds a link; DELETE removes it. Both return 204 and are
idempotent. Link mutation requests have no body. Creation requires active same-business
parent and target records. Removal can clean up an archived target's link, while the
parent must still be active. Archived targets are omitted from lists; resource targets
also require an active location.

Migration 000010 creates service_locations, service_staff, service_resources and
staff_locations. Composite foreign keys include business_id at both ends, preventing
cross-tenant links independently of service checks. Primary keys prevent duplicate
links; reverse-lookup indexes support target relationships. Restrictive deletion
preserves referential integrity. Changes run inside the existing tenant transaction.

These links are independent eligibility facts, not appointments, schedules, capacity
reservations or automatic membership grants. Linking a resource does not auto-link its
location; linking staff to a service does not auto-link their locations. The future
availability/booking engine must check the applicable relationships together and
apply hours, absences, lead time and capacity rules. No current booking enforcement
or availability guarantee is implied by an eligibility link.

### Pagination and HTTP safety

All eight catalog/eligibility lists accept `limit=1..100` (default 100) and `cursor`.
Responses remain arrays, including `[]` for empty lists. `X-Next-Cursor` is present
only when another page exists and is exposed through credentialed CORS. Ordering is
ascending UUID; the cursor contains the last returned UUID, actor, business, list kind,
and parent for eligibility lists. It is bounded to 1024 characters. These unsigned
cursors are selectors, not credentials; each page rechecks current membership.
Pages are fresh reads, not a frozen export snapshot, so concurrent edits may change
later pages. Cross-kind/business/actor/parent cursor reuse is rejected.

Mutations require the exact configured WEB_ORIGIN. GET may omit Origin; a supplied
foreign origin is rejected. OPTIONS requires the configured origin and handles
preflight without a session. Responses use Cache-Control: no-store. Catalog mutations
share a process-local limit of 60 attempts per transport peer per 15 minutes; forwarded
IP headers cannot bypass it. Requests receive a ten-second service timeout.

Creation requires application/json, caps the body at 4096 bytes, and rejects unknown
fields, null, non-object input and trailing JSON. Errors are generic:

- 400: invalid input, UUID, pagination or unexpected link body
- 401: missing/invalid session or disabled account
- 403: origin or tenant/role denied
- 404: unknown route, missing/archived/other-business catalog record
- 405: unsupported method; Allow lists permitted methods
- 413: creation body exceeds the size limit
- 415: unsupported creation content type
- 429: mutation rate limit; Retry-After is 900 seconds
- 503: authentication/storage/transaction failure, without database details or partial data

### Verification and current limits

Verified on 2026-10-11: sqlc regeneration (no generated drift), gofmt, database-enabled
`go test ./... -count=1`, `go build ./...`, and `go vet ./...`. API health and provider
discovery confirm normal startup with Google enabled and Apple disabled.

Coverage includes real HTTP/session/database integration for all entity and link
families, 105 entries in every list, current-role checks, forged headers, cross-tenant
reads and foreign keys, archived targets, revoked cursor access, strict JSON, origin
checks and unsupported lifecycle routes. Added tests cover allowed/rejected preflight,
missing/duplicate cookies, transport-peer rate limiting, generic storage errors,
transaction timeouts, link failures, and suppression of data/cursors on failed commits.
Migration 000010 up/down/up is checked in a rollback-only isolated schema.

No manual configuration is needed. There are still no catalog frontend screens,
item edit/archive workflows, booking endpoints or capacity enforcement. Phase 5.3 adds scheduling configuration below.
Apple live setup, browser QA, Go race-detector and production scaling checks remain
in DEFERRED_WORK.md. No production-readiness or external-security-review claim is made.

## Phase 5.3 - Location hours and scheduling rules

Completed 2026-10-11. See [SCHEDULING.md](SCHEDULING.md) for the exact API and semantics.
Location schedule GET/PUT manages weekly local-minute intervals and explicit full-day
closures. Service scheduling-rule GET/PUT manages before/after buffers and minimum
lead time. Both use existing catalog authentication, role/origin checks and transactions.
Migration 000011 enforces tenant foreign keys and value constraints. Schedule replacement
is atomic, bounded, overlap-validated and serialized by the business lock.

Verification passed: sqlc, gofmt, full database-enabled Go tests, build, vet, migration
round trips, cross-tenant/role denial, actual rollback injection and concurrent
replacement tests. No manual setup or frontend changes are required. Google stays
enabled; Apple and prior release checks remain deferred.

## Phase 5.4 - Availability windows and advisory slots

Completed 2026-10-11. [AVAILABILITY.md](AVAILABILITY.md) records the exact APIs,
UTC window semantics, DST behavior, limits and verification. Staff/resources now
have protected availability-window GET/PUT; services expose GET advisory slots for
one local date, optionally filtered by a selected staff profile and resource.

Slot calculation intersects active tenant records, explicit eligibility, location
hours, closed dates, duration, buffers, lead time and selected availability windows.
Missing configuration/eligibility/windows fails closed. Repeated and skipped local
times are handled through real instants. No booking or capacity reservation is made.
Migration 000012 and the full database-enabled tests/build/vet checks passed.

## Phase 5.5 - Transactional booking creation

Completed 2026-10-12. [BOOKINGS.md](BOOKINGS.md) records the protected tenant-admin
POST API, durable actor-scoped idempotency, buffered occupancy and database triggers.
Creation rechecks the shared availability calculation inside the tenant transaction.
Concurrent staff/resource protection applies even to direct SQL inserts; peak resource
occupancy preserves valid adjacent/spanning bookings. Migration 000013 is applied.

Full database-enabled tests, build and vet passed. No frontend or manual configuration
is required. Advisory slots remain non-reserving and do not subtract existing bookings.

## Phase 5.6 - Booking cancellation and rescheduling

Completed 2026-10-12. [BOOKING_LIFECYCLE.md](BOOKING_LIFECYCLE.md) records the exact
POST APIs, owner/admin policy, pre-buffer cutoff, terminal history and durable retries.
Cancellation releases capacity; rescheduling creates a linked replacement in the same
transaction. Failed moves preserve the original. Migration 000014 adds audit metadata,
response snapshots and database transition/replacement guards. Current eligibility and
scheduling are rechecked for moves; historical creation/retry responses stay stable.

Full PostgreSQL-enabled tests, build and vet passed. No manual setup or frontend
changes are required. Customer-facing workflows and outstanding release checks remain
outside this milestone.

## Phase 5.7 - Booking reads, occupied slots and test UI

Completed 2026-10-13. [BOOKING_WORKSPACE.md](BOOKING_WORKSPACE.md) documents the
protected paginated booking/current-history APIs, capacity-aware advisory filtering
and simple /bookings UI. Demo setup, booking, details, linked history, rescheduling and
cancellation are available for user testing. No additional migration is required.

Full database-enabled Go tests/build/vet and frontend lint/build passed. Headless
Chrome verified the complete local flow and capacity release; desktop/mobile captures
were reviewed. Earlier release checks remain explicitly deferred.

## Phase 5 development closure

The bounded Phase 5 core-booking scope is complete, including its required test UI.
This does not imply production readiness, public customer workflows or full business
dashboards. Next: Phase 6.1 business workspace and catalog-management UI. Phase 6
completion must also provide a usable UI and testing instructions.
