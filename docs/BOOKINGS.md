# Booking creation - Phase 5.5

Completed 2026-10-12. Phase 5 remains in progress. This records the creation contract;
[BOOKING_LIFECYCLE.md](BOOKING_LIFECYCLE.md) adds Phase 5.6 cancellation/rescheduling.

## API

`POST /api/v1/businesses/{businessID}/bookings` creates a confirmed tenant-admin
reservation. Current active BUSINESS_OWNER or BUSINESS_ADMIN membership is required.
MANAGER, STAFF and global platform roles have no write bypass. `created_by` records
the authenticated actor; it does not designate a customer or grant a role.
Customer self-service, guest/customer identity capture and booking frontend screens
are outside this milestone.

Send the existing session cookie, exact configured WEB_ORIGIN, application/json,
and one `Idempotency-Key` header (1-128 ASCII letters, digits, hyphens or underscores).
No query parameters are accepted. Example body:

```json
{
  "service_id": "<service UUID>",
  "location_id": "<location UUID>",
  "staff_id": "<optional staff UUID>",
  "resource_id": "<optional resource UUID>",
  "starts_at": "2030-01-15T09:00:00+08:00"
}
```

Omit optional IDs or send null. One selected resource consumes one capacity unit;
a selected staff member is exclusive. Omitted staff/resource reserves neither.
There is no automatic assignment or implicit required resource inferred from links.
Starts must be real, minute-aligned RFC3339 instants in years 2000-2100, on the
location's 15-minute local grid. The server calculates the end and buffered occupancy
from current service configuration; clients cannot supply duration, status or actor.

201 returns id, service_id, location_id, staff_id, resource_id, starts_at, ends_at,
status `confirmed`, and replayed false. Timestamps are UTC; omitted selections are
null. A successful retry returns 200 with the same persisted fields and replayed true.
No database timestamps, actor, hash or idempotency key are returned.

Errors: 400 invalid input/key/JSON, 401 missing/invalid session or disabled account,
403 origin/membership/role denial, 405 unsupported method, 409 BOOKING_CONFLICT for
unavailable/missing/foreign selections or occupancy, 409 IDEMPOTENCY_CONFLICT when a
key is reused with different input, 413 body over 4096 bytes, 415 wrong media type,
429 attempt limit, 503 storage/transaction failure. Database details are not exposed.
Unknown JSON fields, null/nonobject bodies and trailing JSON are rejected.

Responses are no-store. Credentialed CORS allows only the configured origin;
OPTIONS is unauthenticated and allows Content-Type and Idempotency-Key. Mutations
have a ten-second timeout and process-local limit of 60 attempts per transport peer
per 15 minutes. Phase 5.7 adds GET/list routes; Phase 5.6 adds lifecycle POST routes. In-place edits and deletes remain unsupported.

## Transaction and retry semantics

The existing tenant transaction locks the business, then rechecks active membership.
The durable key is scoped to business + authenticated actor. A SHA-256 digest of the
normalized typed request detects mismatches; equivalent timezone offsets produce the
same digest. Keys remain with booking history and have no expiry.
After a lifecycle change, creation retries still acknowledge the original confirmation;
they are not current-state queries. Cancellation/rescheduling have separate durable
response snapshots described in BOOKING_LIFECYCLE.md.

After authorization, an existing matching result is replayed even if schedules,
eligibility or the current time have changed. Revoked/disabled actors cannot replay.
For a new key, active records, service/location/staff/resource links, location hours,
closures, scheduling rules, lead time and full buffered availability windows are
recomputed inside that same transaction using the shared DST-aware slot calculation.
Advisory responses are never accepted as proof of availability.

The insert stores the service interval and before/after occupancy snapshots. Booking,
occupancy and idempotency result are one row and commit together. Any failure rolls
back the reservation and key; no success result escapes a failed commit. Concurrent
same-key retries return one booking. Different keys still undergo conflict checks.

## Database protection

Migration 000013 adds bookings with composite tenant foreign keys, audit actor,
confirmed-only status, bounded finite ordered timestamps, unique durable key and
staff/resource occupancy indexes. Restrictive foreign keys preserve referenced data.

A BEFORE INSERT trigger updates the tenant row to serialize database writers, including
writers that bypass the service. Each subsequent trigger query sees committed changes
under read committed; the row-version change makes stale repeatable-read writers abort
with a serialization failure instead of using stale occupancy. No extension is needed.

Staff occupancy uses half-open buffered intervals [busy_start,busy_end); adjacent
intervals are allowed. Resource protection clips overlapping bookings to the new
interval and sweeps grouped start/end events. Admission uses peak simultaneous
occupancy, not the sum of all overlapping bookings. One booking consumes one unit.
The resource must belong to the selected location. Resource capacity/location/tenant
changes remain rejected once it has bookings. Phase 5.6 migration 14 replaces the
blanket booking UPDATE rejection with guarded terminal transitions; original times
and selections remain immutable. There is no application deletion operation.

Direct SQL protections cover ownership, intervals and occupancy. Scheduling,
eligibility and active-account policy belong to the service; administrative SQL
must not be used as an alternative booking API. Future configuration writers must
continue following the tenant-lock protocol. Privileged database administrators can
still remove data or disable triggers; ordinary application requests cannot.

## Verification and limits

Verified with gofmt, sqlc generation, PostgreSQL-enabled `go test ./... -count=1`,
`go build ./...`, and `go vet ./...`. Migration up/down/up runs only in an isolated
rollback-only schema. Development startup applied migration 13 (dirty=false).

Tests cover concurrent same-key retries, mismatched payloads, staff/buffer conflicts,
resource capacity, adjacent/spanning occupancy, direct SQL lock waiting, stale
repeatable-read rejection, cross-tenant foreign keys, stale advisory eligibility,
closed dates, off-grid/past starts, role/origin/JSON checks, disabled/revoked replay,
real transaction rollback and failed-commit result suppression. Existing DST tests
continue to exercise the shared slot calculation.

Phase 5.7 advisory slots subtract confirmed occupancy; creation remains authoritative.
Same-business operations serialize, and the trigger scans overlapping occupancy;
large-scale performance/load testing remains future work. No customer identity,
notifications, payments, list UI or public booking flow is introduced here. Phase 5.6
adds cancellation and rescheduling under BOOKING_LIFECYCLE.md. Google remains enabled; Apple and earlier browser/race/production
checks remain in DEFERRED_WORK.md. No manual configuration is required.

Phase 5.6 lifecycle and Phase 5.7 reads/occupied slots are implemented. The /bookings
test UI is documented in BOOKING_WORKSPACE.md. Next: Phase 6.1 business/catalog UI.
