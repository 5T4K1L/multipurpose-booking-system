# Booking lifecycle - Phase 5.6

Completed 2026-10-12. Phase 5 remains in progress.

## Rules

Only a current active BUSINESS_OWNER or BUSINESS_ADMIN in the booking's business
may cancel or reschedule it. Administrators may manage bookings created by another
tenant administrator. Platform roles, MANAGER and STAFF do not bypass this policy.
There is no customer-facing cancellation/rescheduling workflow yet.

A new operation requires a confirmed booking and must happen strictly before its
stored buffered start (`busy_starts_at`). This prevents changing a booking whose
preparation has begun. The database rechecks the cutoff after obtaining its lock.
No fee, refund, configurable cancellation notice, completion or no-show policy is
introduced. These require separate product rules.

| Operation | Original record | Capacity | Result |
|---|---|---|---|
| Cancel | confirmed -> cancelled | Released on commit | Retained terminal record |
| Reschedule | confirmed -> rescheduled | Old interval released and new interval reserved atomically | New confirmed record linked to original |

Terminal records cannot be reactivated or changed again under a new key. Same-key
retries replay their original operation. A later change must target the replacement
ID returned by rescheduling. A chain can have multiple generations, but each original
has at most one direct replacement. Booking times and selections remain immutable.

Rescheduling changes only the start time. Service, location, staff and resource are
retained; duration/buffers/lead time use current service configuration. The new start
must differ from the old one and satisfy the existing UTC instant/local-grid rules.
Active records, eligibility, hours, closed dates and selected availability windows are
revalidated inside the tenant transaction. New staff/resource occupancy is enforced
by the database. Any failure preserves the original confirmed booking and occupancy.
Cancellation still works when catalog records are archived or eligibility/schedules
have changed, provided the booking has not reached its buffered start.

## API

Both endpoints require the existing session cookie, exact WEB_ORIGIN,
application/json and one valid Idempotency-Key header. UUIDs are tenant scoped.

| Method and path | Body |
|---|---|
| POST /api/v1/businesses/{businessID}/bookings/{bookingID}/cancel | `{}` |
| POST /api/v1/businesses/{businessID}/bookings/{bookingID}/reschedule | `{"starts_at":"2030-01-15T10:00:00+08:00"}` |

Both return 200 for first success and retry. Response fields:

- `booking`: the original booking response fields, with its new terminal status.
- `replacement`: only for rescheduling; the newly confirmed booking and its ID.
- `replayed`: false initially, true on a matching retry.

Nested booking objects retain the Phase 5.5 schema (including replayed false).
No internal audit actor, hash or key is returned. Preserve replacement IDs for later
operations. Phase 5.7 provides GET/list APIs and a simple /bookings test screen.

Additional errors are 404 BOOKING_NOT_FOUND (missing or another business's booking)
and 409 BOOKING_STATE_CONFLICT (terminal booking or buffered start reached).
400 covers invalid UUID/key/time, unchanged reschedule time and invalid JSON.
409 BOOKING_CONFLICT covers new-slot availability/occupancy; 409 IDEMPOTENCY_CONFLICT
covers key reuse for different targets/actions/times. Existing 401/403/405/413/415/
429/503 semantics remain. Storage details and partial results are never exposed.

The 4096-byte strict JSON limit, no-store responses, credentialed exact-origin CORS,
unauthenticated OPTIONS, ten-second service timeout and shared booking-handler limit
of 60 attempts per transport peer per 15 minutes apply to creation and lifecycle
operations. Unknown fields, missing/null/nonobject bodies, trailing JSON, duplicate
Idempotency-Key headers and query parameters are rejected.

## Transactions, audit and retries

Every operation uses the existing tenant transaction and current membership check.
Migration 000014 adds booking_changes with a durable response snapshot and a unique
(business_id,actor_id,idempotency_key). This namespace is shared by cancellation and
rescheduling and is separate from initial booking creation keys. Its request digest
binds action, target booking ID and UTC-normalized new start time. Keys do not expire.

Authorization runs before replay. Matching retries return the stored operation result
even after later changes; disabled/revoked actors cannot replay. Initial creation
retries likewise preserve their original confirmation fields, rather than reporting
later terminal state. Retry responses are historical acknowledgements, not a current
booking-status query.

Closing stores closed_by and a database-generated closed_at. A reschedule inserts a
new booking with supersedes_booking_id, preserving the original creator, selections
and interval on the old record. The new record's creator is the rescheduling actor.
Its internal creation key uses a reserved prefix that public creation keys reject.
The state change, replacement, occupancy and durable response commit together. Audit
write or commit failures roll back everything and suppress success responses.

## Database guarantees

Only confirmed records participate in staff and resource occupancy checks/indexes.
The existing tenant-row version update still serializes direct SQL insert writers
and rejects stale repeatable-read snapshots. Transition updates also acquire that
lock. Uncommitted capacity releases cannot admit another booking; rollback restores
occupancy before a waiting writer checks it.

Database triggers reject terminal inserts, in-place identity/time edits, reactivation,
invalid transitions and changes after the stored buffered start. Composite tenant
foreign keys and a unique predecessor link protect replacement identity. A deferred
constraint trigger requires a replacement by commit for a reschedule; its selections
must match its predecessor. Booked-resource capacity/location edits remain restricted,
including records with terminal history. No deletion API is added; privileged database
administration remains outside the application's lifecycle workflow.

Migration 14 is additive for existing confirmed bookings. Its down migration refuses
to proceed when lifecycle history exists, instead of discarding that history. All
up/down/up verification uses an isolated rollback-only schema.

## Verification and remaining work

Passed: sqlc generation, gofmt, PostgreSQL-enabled `go test ./... -count=1`,
`go build ./...`, and `go vet ./...`. Development schema is version 14, dirty=false.

Coverage includes failed moves retaining originals, self-overlapping moves, released
staff/resource capacity, simultaneous cancel versus move, concurrent retry deduplication,
response stability after later changes, audit/transaction/commit failure suppression,
direct SQL waiting on an uncommitted cancellation, terminal/cutoff/deferred-replacement
guards, cross-tenant and role denial, archived/changed eligibility, HTTP validation,
revoked retries and disabled sessions. Existing creation, OAuth and DST tests pass.

No manual configuration or frontend changes are needed. Google remains enabled.
Apple live setup, browser QA, race detector and production checks remain in
DEFERRED_WORK.md. Phase 5.7 advisory slots now filter confirmed occupancy.

Phase 5.7 read APIs and the booking test UI are complete; see BOOKING_WORKSPACE.md.
Customer self-service, notifications, payments and frontend dashboards remain separate.
