# Availability and slots - Phase 5.4

Completed 2026-10-11. Slot results are advisory: no booking, capacity reservation or
conflict check against bookings is implemented yet. This is a tenant-admin API,
not a public customer booking endpoint.

## Staff/resource availability windows

GET and PUT:

- `/api/v1/businesses/{businessID}/catalog/staff/{staffID}/availability-windows`
- `/api/v1/businesses/{businessID}/catalog/resources/{resourceID}/availability-windows`

PUT accepts one complete replacement, for example:

```json
{
  "windows": [
    { "starts_at": "2026-10-12T09:00:00+08:00", "ends_at": "2026-10-12T17:00:00+08:00" }
  ]
}
```

GET and PUT return 200 and an array of starts_at/ends_at objects normalized to UTC.
An empty array explicitly clears availability. Missing/null windows are invalid.
Limits: at most 32 nonoverlapping windows; minute precision; explicit RFC3339 offset
or Z; years 2000-2100; end after start; each window at most seven elapsed days.
Adjacent windows are allowed and together may cover continuous occupancy.

Windows are absolute intervals, not recurring shifts. Operators must replace them
as future dates are planned; no recurrence or automatic extension is inferred.
Staff windows apply at any location where that staff profile is explicitly eligible.
Resource windows apply to that resource at its assigned location. Missing windows
mean no availability when that entity is selected. No user identity is attached to
a staff profile, and these endpoints never grant membership roles.

Migration 000012 adds staff_availability_windows and resource_availability_windows
with tenant-scoped foreign keys, unique start keys, finite timestamps and duration
checks. Overlap/count/minute-precision policy is enforced by the service. All writers
must use that policy and the existing business lock. Replacements are atomic and
last committed replacement wins; there is no optimistic edit-version check.

## Slot query

GET `/api/v1/businesses/{businessID}/catalog/services/{serviceID}/slots`

Required query parameters: `date=YYYY-MM-DD` and `location_id=UUID`.
Optional: `staff_id=UUID` and `resource_id=UUID`. Only one of each is supported.
Unknown, duplicate, empty and malformed query values are rejected. Date years are
2000-2100; each request calculates one local calendar date, not a date range.

Response shape:

```json
{
  "date": "2026-10-12",
  "timezone": "Asia/Shanghai",
  "advisory": true,
  "slots": [
    { "starts_at": "2026-10-12T01:15:00Z", "ends_at": "2026-10-12T01:45:00Z" }
  ]
}
```

The example illustrates formatting only; actual slots depend on saved rules and
server time. Results use a fixed 15-minute local start grid and UTC instants, sorted
chronologically. No continuation cursor is needed for this bounded one-day result.

Every query rechecks current membership and active catalog records in one tenant
transaction. Computation uses current server time and configured rules:

1. The service must be linked to the requested location and have explicit scheduling
   rules. An unconfigured rule set yields no slots; explicit zero rules are valid.
2. Every elapsed minute of service duration plus both buffers must fit the location's
   weekly opening hours, excluding closed local dates. Adjacent opening intervals can
   form a continuous interval. Overnight and multi-day occupancy checks all dates.
3. Start must meet minimum lead time; preparation/buffer start cannot be in the past.
4. A selected staff profile must be active and linked to both the service and location.
   A selected resource must be active, assigned to this location and linked to the service.
5. Each selected entity's availability windows must cover the full buffered occupancy.

Missing eligibility, closed dates, unconfigured rules, insufficient windows or a
resource assigned elsewhere produce an empty slots array. Missing/archived/foreign
records produce the existing generic catalog-not-found response after tenant checks.
Without staff_id/resource_id, the result checks service/location rules only. It does
not imply that an eligible staff member or resource is available. No automatic entity
assignment, pooled-capacity allocation or multi-resource requirement is inferred.

## Daylight saving and elapsed time

The engine walks real minute instants and converts them to the location's timezone.
It does not manufacture local times with time.Date. Nonexistent spring-forward times
produce no starts. Both occurrences of a repeated fall-back time can appear as
different UTC instants. Durations, buffers and lead time are elapsed minutes, so
local end-clock labels may jump forward or backward across a transition.

Clients must retain the UTC instant/offset instead of using a local clock label as an
identifier. A skipped civil date has no slots. Results have minute granularity and
bounded work; cancellation is checked during calculation. Rules and windows are read
under the existing tenant lock; results may become stale as soon as that lock releases.

## Security and verification

Existing catalog controls apply: active sessions/membership, OWNER/ADMIN writes,
MANAGER/STAFF reads, no platform-admin tenant bypass, exact-origin mutations,
strict 4096-byte JSON bodies, no-store responses and generic errors. Window mutations
share the catalog's per-peer limit. Slots are GET-only and protected; ten-second service
timeouts apply. Inputs cannot choose an authenticated user, role or authorization scope.

Verified: sqlc generation, gofmt, database-enabled `go test ./... -count=1`,
`go build ./...`, `go vet ./...`, and migration up/down/up in a rollback-only schema.

Unit tests cover spring/fall DST, half-hour DST changes, a skipped civil date,
chronological unique instants, closed-day boundaries, multi-day durations, adjacent
windows, buffers, lead time, invalid window ranges and cancellation. HTTP/database
integration covers eligibility intersection, absent windows/rules, staff/resource
selection, role and origin denial, revoked access, archived records, tenant foreign
keys and actual failed-window-replacement rollback preserving previous data.

API health and Google provider discovery passed. No manual setup is required.
Apple, browser QA, race-detector and production checks remain in DEFERRED_WORK.md.

## Booking integration (Phase 5.5)

[BOOKINGS.md](BOOKINGS.md) documents transactional creation, durable idempotency and
database-enforced buffered staff/resource occupancy. Creation reuses the same slot
calculation inside its tenant transaction and rejects stale advisory results.
Phase 5.7 advisory slots now subtract confirmed staff/resource occupancy; they never reserve capacity.
Phase 5.6 cancellation/rescheduling now releases or replaces occupancy atomically;
see BOOKING_LIFECYCLE.md. BOOKING_WORKSPACE.md records Phase 5.7 read APIs and the
simple /bookings test UI. Public customer availability remains unimplemented.
