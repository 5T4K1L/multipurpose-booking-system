# Customer availability calendar

Implemented 2026-10-17 before Phase 7.7. No migration; schema remains 20.

## Use

Open `/businesses`, open a published business, choose a service and location,
then select an available calendar date and time. Existing confirmation and
customer ownership rules remain in place.

- Green / dot: at least one bookable time remains.
- Amber / Full: scheduling permits times, but booking occupancy removes all of them.
- Gray / dash: no bookable times under current hours, closures, lead time,
  staff/resource windows or eligibility. Past dates are unavailable.
- Unknown availability after an error is disabled, never labeled fully booked.

Dates use the selected location's timezone. A partially occupied day stays
selectable. Changing service, location or month clears the previous selection.
Successful receipts remain visible during automatic refreshes.

## Updates and safety

The visible tab refreshes every 30 seconds, on focus/visibility changes and after
booking submission. Manual refresh is also available. This is polling, not
instant server push; other users' changes can take up to a polling interval plus
request time to appear. Hidden tabs do not poll. Failed refreshes disable new
confirmations until availability can be checked again.

Bookings are still validated atomically against current scheduling and occupancy
under the existing business transaction lock. Stale calendar results cannot
bypass staff exclusivity or resource capacity. Creation, rescheduling and
cancellation affect the next calendar read without republishing the business.
An uncertain submission can retry its existing idempotency key even if the saved
booking has filled the slot, preserving receipt recovery without duplication.

Capacity follows existing business configuration: staff are exclusive and
resources use their configured capacity. Services without either constraint do
not acquire an invented daily booking limit.

## API

`GET /api/v1/public/businesses/{id}/services/{serviceID}/calendar?location_id={uuid}&month=YYYY-MM`

Anonymous, no-store, allowed-origin checks. Exactly one location and month are
accepted; years 2000–2100. Returns month, location timezone, advisory=true, and
every day with status (`available`, `fully_booked`, `unavailable`) and remaining
UTC start/end pairs. No booking, customer, staff or resource identities leak.

One month per request, maximum 32 allocation combinations, ten-second context,
120 calendar requests per transport peer per 15 minutes. Oversized configurations
or calculation failures return errors rather than misleading partial months.
The single-day endpoint remains compatible and has its existing separate limit.
The calendar reuses the same scheduling and occupancy calculation as that endpoint.

## Validation

Database-enabled `go test ./... -count=1`, `go build ./...`, `go vet ./...` passed.
Frontend `npm run lint`, `npm run build` and `git diff --check` passed.
Integration coverage includes partial/full capacity, cancellation, closed dates,
lead time, malformed requests, business/location isolation, stale publication
details, privacy and allocation limits. Browser checks passed at 320, 390, 768 and
1280 pixels with no page errors: booking receipt, double-booking rejection,
rescheduling, cancellation, actual timer polling, month/location isolation,
failed-refresh recovery and lost-response idempotent receipt recovery. The mobile
screenshot was visually inspected. Browser fixtures were removed afterward.

## Files changed

- `apps/api/internal/service/public_availability.go`
- `apps/api/internal/server/customer_foundation.go`
- `apps/api/internal/server/public_availability_integration_test.go`
- `apps/web/app/businesses/[id]/availability-calendar.tsx`
- `apps/web/app/businesses/[id]/availability.tsx`
- `apps/web/app/businesses/[id]/booking-confirmation.tsx`
- `docs/PUBLIC_AVAILABILITY_CALENDAR.md`
- `docs/PUBLIC_AVAILABILITY.md`
- `docs/CURRENT_STATE.md`

Workspace verification script: `work/booking-ui-check/public-calendar-smoke.cjs`.
Phase 7.7 customer cancellation UI remains next; existing operator cancellation
and rescheduling already update this calendar.
