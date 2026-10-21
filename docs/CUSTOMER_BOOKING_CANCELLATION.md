# Phase 7.7 — Customer cancellation

Completed 2026-10-18. No migration for this phase; schema remains 21 from the
earlier, unfinished booking-range work.

## Customer flow

Open `/my-bookings`, open a booking's details, choose **Cancel booking**, then
**Confirm cancellation**. **Keep booking** dismisses the confirmation without
changing anything. Cancelled bookings remain in customer and business history.

The cancellation deadline is the saved preparation/start time (`busy_starts_at`),
consistent with the existing operator lifecycle. Details show this deadline in
UTC. The server rechecks it at submission; a stale button never bypasses it.

Cancellation releases staff/resource occupancy immediately on commit. The public
calendar sees the change on its next refresh (normally within 30 seconds while
visible). Business booking lists show the shared saved status on refresh.

## API and authorization

`POST /api/v1/customer/bookings/{id}/cancel`

Requires an active session, exact allowed Origin, JSON body `{}` and one valid
`Idempotency-Key`. No customer ID, business ID, role or status can be submitted.
Unknown, legacy/unowned and other customers' records all return 404. Business
membership does not grant access to another customer's cancellation endpoint.

Cancellation remains possible if a business is unpublished or disabled, or its
catalog changes. It never depends on a currently published profile. Confirmed
bookings can be cancelled before preparation begins; started, completed and
rescheduled records return 409. Customers must open the replacement booking to
act on a rescheduled appointment.

The operation locks the business in the same order as operator changes, rereads
ownership/state, performs the transition and writes its audit in one transaction.
Existing database lifecycle/occupancy guards remain in force. A failed transaction
does not cancel the booking or release capacity.

Keys use a separate customer-cancellation namespace. Identical retries return the
saved receipt; reusing a key for a different owned booking in the same business
returns 409. Cancelling an already cancelled owned booking with a fresh key returns
success without another state transition or audit. Responses expose the safe
customer booking projection plus `replayed` and `already_cancelled`.

The existing detail GET adds `can_cancel` and `cancel_before`. List responses are
unchanged. Requests are no-store, have a ten-second context and allow 60 attempts
per transport peer per 15 minutes.

## Validation

- Database-enabled `go test ./... -count=1`, `go build ./...`, `go vet ./...`.
- Frontend lint and production build.
- Integration coverage: ownership, forged identity headers, legacy/missing records,
  origin/body/key validation, duplicate concurrent requests, idempotency conflicts,
  preparation deadlines, terminal states, retained history, disabled businesses,
  capacity release and forced transaction rollback.
- Browser checks: keep/cancel flow, lost-response retry, business visibility,
  released calendar capacity, reload/history after unpublishing, and no overflow
  at 320/390/768/1280 pixels. No page errors. Mobile screenshot inspected.

## Files changed in Phase 7.7

- `apps/api/internal/db/queries/customer_foundation.sql`
- `apps/api/internal/db/customer_foundation.sql.go` (generated)
- `apps/api/internal/service/customer_booking_cancel.go`
- `apps/api/internal/service/customer_foundation.go`
- `apps/api/internal/server/customer_booking_cancel.go`
- `apps/api/internal/server/customer_foundation.go`
- `apps/api/internal/server/customer_booking_cancel_integration_test.go`
- `apps/api/internal/server/customer_foundation_integration_test.go`
- `apps/web/app/my-bookings/cancel-booking.tsx`
- `apps/web/app/my-bookings/workspace.tsx`
- `docs/CUSTOMER_BOOKING_CANCELLATION.md`
- `docs/CUSTOMER_BOOKING_HISTORY.md`
- `docs/CURRENT_STATE.md`
- `docs/ROADMAP.md`
- `docs/DEFERRED_WORK.md`

Workspace browser script: `work/booking-ui-check/customer-cancel-smoke.cjs`.
Next planned phase: 7.8 customer rescheduling. The earlier booking-range/owner
setup work remains a separate unfinished task recorded in DEFERRED_WORK.md.
