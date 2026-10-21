# Customer booking creation - Phase 7.5

Implemented 2026-10-16. Schema remains version 19; no migration or new configuration.

## Try it

1. Open `/businesses`, find a published business, and select a service, location and date.
2. Click **Check availability**, choose a time, and review the service/location/time shown above the time list.
3. Click **Confirm booking**. If signed out, use the sign-in or registration link (new tab), then return and confirm. Password and configured Google sign-in use the existing session flow.
4. A successful response shows the saved status and booking reference. Save the reference until customer history UI is delivered.
5. The business owner/admin can find the received booking in `/business/bookings` and use existing business management actions. Other businesses cannot access it.

Publication must already include reviewed service/location details. Existing profiles may need owner/admin review and republishing. Merely choosing a time never reserves capacity.

## API

`POST /api/v1/customer/bookings`

Requires an active authenticated session, exact allowed Origin, JSON body and one `Idempotency-Key` header (1-128 letters, digits, hyphens or underscores). Preflight supports Content-Type and Idempotency-Key. No query parameters or unknown body fields are accepted. Requests are limited to 60 per peer per 15 minutes with a 10-second context.

Body fields:

- `business_id`, `service_id`, `location_id`: UUIDs from the public profile.
- `starts_at`: exact UTC/offset timestamp from public availability.
- `expected_ends_at`: the displayed slot end, used only to detect a stale duration.
- `expected_timezone`: the displayed location timezone, used only to detect a stale timezone.

Customer identity, audit actor, status, staff/resource assignment, stored end and buffered occupancy are server-controlled. Clients cannot submit customer_id, created_by, staff_id, resource_id or ends_at. Current rules determine the actual saved end; stale expectations return 409 rather than changing the requested duration/timezone silently.

201 creates a booking; 200 replays a saved request. The response uses the existing customer-safe booking projection plus `replayed`, with no staff/resource identity, other customer data or internal request keys/hashes. The existing customer item GET remains available by exact ownership.

## Atomicity, allocation and retry policy

Creation locks the active business for the transaction, checks the active customer, then checks the reviewed publication, current service/location eligibility, current schedules, lead time, buffers and occupancy. It chooses the first available eligible staff/resource combination in deterministic ID order, using the same candidate policy as public availability. A configured allocation type requires an eligible active choice; a type with no links is optional. Up to 32 combinations are supported. No staff/resource assignment means no capacity reserved for that type.

The insertion persists both customer_id and created_by as the trusted session actor. Existing database staff-exclusion/resource-capacity constraints remain the final guard. All normal tenant writers share the same business lock. Failed transactions expose no confirmation and leave no booking. Old operator bookings remain unowned.

Customer request keys are hashed under a reserved internal namespace, scoped by business and actor, so they cannot collide with operator creation or reschedule keys. UTC normalization handles offset-equivalent retries. Reusing a key with different input returns 409. Replay returns the saved booking's current lifecycle status and does not recreate cancelled bookings. Replays survive unpublishing; disabled businesses/accounts remain unavailable.

The UI stores a pending random key in sessionStorage, scoped to account and payload. A lost response, timeout or server failure preserves it for retry. It disables selection controls during submission and shows one receipt after success. Retry the same selection after an uncertain response; do not intentionally start a different booking to recover the first one. SessionStorage recovery is limited to that browser tab/session.

## Validation and boundaries

Database-enabled Go tests, build and vet, gofmt, frontend lint/build, and responsive browser verification are recorded with this milestone. Tests cover ownership, cross-business/customer isolation, authentication/origin/input rejection, disabled customers, stale schedules/publication/duration/timezone, concurrent staff/resource conflicts, concurrent duplicate retries, operator/customer key separation, cancellation replay and transaction rollback.

The browser test creates temporary owner/customer accounts, publishes a fixture, checks the signed-out prompt, confirms as a customer, deliberately drops the successful response, retries the same key, verifies the same reference and business visibility, and checks occupied-time removal. It also covers unpublishing and responsive widths 320/390/768/1280; fixture data is removed afterwards.

Phase 7.5 stops at creation/confirmation. Next: Phase 7.6 customer booking history and detail UI. Customer rescheduling/cancellation follow with ownership enforcement. No payments, notifications, customer lifecycle endpoints or production deployment are included.

## Phase 7.6 completed (2026-10-16)

Customer history and detail UI are available at `/my-bookings` and
`/my-bookings/{id}`. Account and booking confirmation links connect the flow.
Pagination and all reads enforce authenticated customer ownership independently
of business publication. See [CUSTOMER_BOOKING_HISTORY.md](CUSTOMER_BOOKING_HISTORY.md).
Next: Phase 7.7 customer cancellation; rescheduling follows. Full Phase 7 remains open.
