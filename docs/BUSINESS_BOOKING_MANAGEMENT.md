# Phase 6.4 - Business booking management and completion

Completed 2026-10-14. Phase 6 remains in progress.

## Access and test

Open http://localhost:3000/business/bookings, or choose Business bookings in business
setup. Sign in and select a business you belong to. Owner/admin may manage bookings;
manager/staff have read-only views. Only that business's records are accessible.

- Details shows current state and links to original/replacement bookings.
- Before preparation starts, Reschedule opens the existing time picker with the same
  service/location/staff/resource choices; Cancel booking asks for confirmation.
- After the reserved interval including cleanup has ended, Mark completed asks you
  to confirm the service was delivered. Confirm completion retains the booking in
  history as completed. Reload the page or Refresh bookings to verify persisted state.
- Terminal bookings cannot be cancelled, moved, completed again with a new key, or
  reopened. Completion is a manual assertion of delivery, never automatic by time.

The dedicated management page has no booking creation/demo controls. /bookings remains
a development operations/test screen. Customer self-service and customer ownership
are not implemented here; those remain Phase 7. Existing records were created through
internal flows, not a completed customer platform. Both future customer and business
views must read the same persisted booking record.

Times use the displayed timezone (business timezone for history; selected location
while rescheduling). Browser time only guides button availability; service and database
checks enforce timing. History is reference-ordered with pagination, not a calendar.
Refresh is manual, not live push. The conservative reschedule picker still hides times
overlapping the original reservation. Catalog loading remains bounded to 1000 items
per kind. Customer identity/details, calendar filters and dashboard summaries are later.

## API and storage

POST /api/v1/businesses/{businessID}/bookings/{bookingID}/complete
Body: {}. Requires session, exact allowed Origin and one Idempotency-Key header.
The existing strict JSON/body limits, rate limiting, deadline and no-store controls apply.

Only active BUSINESS_OWNER/BUSINESS_ADMIN membership can complete, without a platform
role bypass. Current authorization is checked before reading a retry acknowledgement.
Completion requires confirmed status and server time at or after busy_ends_at, so
cleanup occupancy is not released early. Migration 15 enforces this in PostgreSQL
as well as the service, retains immutable booking data and records closed_at/closed_by.
The status change and booking_changes audit/idempotency result commit atomically.

Completion shares the actor/business lifecycle key namespace with cancellation and
rescheduling. Identical retries replay their original acknowledgement; reuse for a
different action/booking conflicts. Clients refresh current state after acknowledgement.
Responses: 200 success/replay, 400 malformed input, 401 unauthenticated, 403 denied,
404 missing/other-business booking after tenant checks, 409 invalid state/time or key
conflict, and existing body/rate/storage errors. Error details do not expose internals.

Migration 000015 adds completed status and complete action and updates the transition
guard. Downgrade refuses to discard completion history. Local schema is 15, dirty=false.
No generated query/model changes were needed; sqlc generation verified compatibility.

## Verification

- gofmt, database-enabled go test ./... -count=1, go build ./... and go vet ./... passed.
- Integration tests cover early/cleanup rejection, independent SQL guard, foreign
  tenants, staff/anonymous/revoked denial, admin completion, audit failure rollback,
  replay/conflicting keys, retained times/actor, no reopening, and downgrade refusal.
- Frontend lint and production build/TypeScript passed.
- Headless Chrome: completion confirmation, persisted completed history after reload,
  no create controls on management page, and no second completion button.
- Separate management flow: reschedule/cancel retained both historical records and
  restored availability from 25 slots to the original 29. Initial-load action race fixed.
- Zero browser page errors; no horizontal overflow at 320, 390 and 768 widths for
  completion. Mobile completion screenshot reviewed; temporary fixtures removed.

Next: Phase 6.5 business calendar and dashboard. Remaining settings/customization and
full-phase review still precede Phase 6 closure. Google credentials, Apple configuration
and earlier release checks remain documented separately; no production sign-off.
