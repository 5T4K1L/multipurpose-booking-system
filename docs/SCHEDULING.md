# Scheduling rules - Phase 5.3

Completed 2026-10-11. This milestone stores and manages location hours and service
rules. It does not calculate available slots, reserve capacity or create bookings.

## Location schedule API

GET and PUT `/api/v1/businesses/{businessID}/catalog/locations/{locationID}/schedule`.
Both return 200 with the complete schedule and the location's stored timezone.

Example PUT body (Monday 09:00-12:00 and 13:00-17:00, closed on Christmas):

```json
{
  "weekly_hours": [
    { "weekday": 1, "start_minute": 540, "end_minute": 720 },
    { "weekday": 1, "start_minute": 780, "end_minute": 1020 }
  ],
  "closed_dates": ["2026-12-25"]
}
```

- Weekday is 0=Sunday through 6=Saturday, using the location's local calendar.
- Intervals are half-open: start included, end excluded. Start is 0..1439, end is
  1..1440, and start must precede end. Adjacent intervals are allowed; overlaps and
  duplicates are rejected. Split overnight hours across the two affected weekdays.
- At most 28 intervals per location and 128 closed dates are stored by the service.
  Dates must be unique YYYY-MM-DD values with a valid year 0001..9999.
- A closed date represents that entire local calendar day and overrides weekly hours
  when the future availability engine applies the rules. Dates are not UTC instants.
- Every PUT replaces BOTH arrays atomically. Both fields and every interval field
  are required; null/missing values are rejected. Empty arrays explicitly clear the
  schedule. Unconfigured locations return empty arrays and must be treated as closed.
- Responses sort hours by weekday/start and dates chronologically. Timezone is
  read-only here; it comes from the location record. No global business-hours fallback
  or holiday calendar is inferred. Each location must be configured explicitly.

All mutations use the business lock. Overlap validation is enforced by the service;
SQL checks enforce numeric bounds, ordering, duplicate keys and tenant foreign keys.
Future direct writers must preserve the same validation and locking contract.
Concurrent replacements serialize: the last committed complete replacement wins.
There is no optimistic edit-version check; clients must account for full replacement.

## Service scheduling rules API

GET and PUT `/api/v1/businesses/{businessID}/catalog/services/{serviceID}/scheduling-rules`.
Both return 200. Example PUT body:

```json
{
  "buffer_before_minutes": 15,
  "buffer_after_minutes": 10,
  "minimum_lead_minutes": 120
}
```

All three integer fields are required. Each buffer is 0..1440 minutes; minimum lead
is 0..525600 minutes (365 days). PUT replaces the complete rule set. GET returns
`configured: false` and zero values when no rules have been saved; saving explicit
zero rules is valid and returns `configured: true`. Unconfigured rules do not imply
that a service is bookable. The future booking/availability engine must handle this
state explicitly.

These values describe intended occupancy before/after service delivery and the
minimum time between booking and appointment start. Enforcement, UTC instant
conversion, DST gaps/repeated times, staff calendars and booking conflicts are future
work. Storing local wall-clock hours here makes no claim that every local minute
exists or occurs once on a DST transition day.

## Authorization and storage

The existing catalog controls apply: authenticated active membership; OWNER and
BUSINESS_ADMIN can PUT, MANAGER and STAFF can GET. Platform roles cannot bypass
membership. Archived or other-business entities return the same 404 after tenant
policy is checked. A disabled business is denied. No identity or role is taken from
request headers or JSON.

PUT requires exact WEB_ORIGIN, application/json, a maximum 4096-byte body and no
unknown/trailing fields. Catalog mutation rate limiting remains 60 attempts per
transport peer per 15 minutes. OPTIONS uses existing preflight handling without a
session. Responses are no-store and use the existing generic catalog error mapping.
Unsupported methods and extra path segments are rejected.

Migration 000011 adds location_weekly_hours, location_closed_dates and
service_scheduling_rules. Composite foreign keys carry business_id to the location
or service. Restrictive deletion preserves references. Closed dates use PostgreSQL
DATE, not timestamptz. Rules have a server-managed updated_at timestamp. All schedule
replacement statements and rule upserts run inside the existing tenant transaction.
A write or commit failure must expose neither partial persisted changes nor a result.

## Verification

- sqlc regeneration, gofmt, database-enabled `go test ./... -count=1`,
  `go build ./...`, and `go vet ./...` passed.
- HTTP/database checks cover authenticated role policies, origin protection, malformed
  and incomplete input, default/empty schedules, explicit rules, archived locations,
  cross-business reads/writes, revoked membership and disabled businesses.
- Real database failure injection after deleting/reinserting schedule rows verifies
  rollback restores both prior hours and prior dates. Concurrent replacements verify
  the final state is one complete schedule, not mixed rows.
- Database foreign-key/check constraints and isolated migration up/down/up are tested.
- Unit checks cover overlap, adjacency, midnight boundaries, leap/invalid dates,
  duplicate dates, limits, rule bounds and non-mutation of caller-owned input.

The additive migration is applied by normal API startup. No manual configuration
is required. API health and Google provider discovery passed. Apple, browser QA,
race-detector and production checks remain deferred in DEFERRED_WORK.md.

## Next task

Phase 5.4: staff/resource availability and timezone-aware slot calculation, including
DST behavior and the intersection of eligibility, location hours, closed dates,
service duration, buffers and lead time. Booking creation, cancellation, idempotency
and transactional capacity/double-booking protection remain later Phase 5 work.
There are no scheduling frontend screens in this milestone.
