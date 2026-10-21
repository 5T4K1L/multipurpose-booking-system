# Phase 7.8: Customer rescheduling

Completed 2026-10-18. No schema migration (schema 21).

## Customer flow

My bookings > View booking details > Reschedule booking. Pick a date at the
branch, select an available time and Confirm reschedule. Time labels use the
customer device timezone and compact same-day formatting. Keep current booking
closes the editor without mutation. Occupied times are omitted; the selection
refreshes every 30 seconds and on focus. A failed availability read clears times.
The original remains reserved until the replacement commits. Both records remain
in history with links. Businesses see the same persisted transition.

## Contract

- GET `/api/v1/customer/bookings/{id}/reschedule?date=YYYY-MM-DD`: signed-in owner
  only, returns the original booking's location timezone and available slots.
  Date is a branch-local date. The booking's own occupancy is excluded, but the
  unchanged original start is not offered. No staff/resource identity is exposed.
- POST the same path with JSON `{ "starts_at": "RFC3339 instant" }` and one
  Idempotency-Key. Exact configured Origin and session required. Unknown JSON
  fields are rejected. No customer/business/staff override is accepted.
- Only confirmed bookings before their saved preparation start can move. Business
  and selected catalog/assignments must still be active and valid. Unpublishing
  alone does not prevent management of an existing booking; membership is not needed.
- Keep original duration, service, branch, staff and resource. Current opening
  hours, closed dates, lead time and buffers are revalidated. A changed service
  default duration does not silently resize the customer's reservation.
- Ownership lookup takes the business lock, serializing against all other writers.
  Close original, create replacement preserving customer ownership, and write audit
  in one transaction. The database occupancy guards prevent double booking. Any
  failure rolls back the release and replacement. Moves overlapping their own
  original interval are allowed if no other booking conflicts.
- Namespaced idempotency hashes bind actor/business, original ID and new instant.
  Matching retries replay the original response, even after the business is disabled.
  Different input with the same key returns 409. Replay still checks ownership.
- Responses contain customer-safe booking and replacement records, plus replay flag.
  400 invalid input, 401 missing authentication, 403 inactive actor/origin, 404
  missing/unowned booking, 409 state/availability/key conflict, 503 storage failure.
  Responses are no-store. Separate limits: 120 reads and 60 writes per peer/15 minutes;
  both use 10-second operation deadlines. OPTIONS supports GET/POST.

## Verification

Database-enabled `go test ./... -count=1`, `go build ./...`, `go vet ./...` and gofmt.
Regression tests cover ownership/forged identity, legacy bookings, malformed input,
origin, occupied target rollback, forced transaction rollback, overlapping original,
concurrent retries, one audit/child, duration preservation after catalog edits,
private-field exclusion, history/ownership links, deadlines and disabled business.
Frontend lint, production build, local-time/format tests. The build needs network
access for the project's existing Google Fonts dependency.
End-to-end temporary fixture verified keep-current, reschedule, simulated lost
response/retry, business visibility, old/new capacity, history and unpublished
business management at 320/390/768/1280px. Fixtures cleaned after execution.

## Boundaries

This moves the existing assignment; it does not switch service, branch, duration,
or automatically reassign another employee/resource. Dates use a simple native
branch-date input, followed by valid times; this is functional UI, not the final
redesign. A stale selection can still lose a race; confirmation revalidates it.
Existing custom-range owner-to-customer end-to-end work remains separately deferred.

## Files changed

- apps/api/internal/service/customer_booking_reschedule.go
- apps/api/internal/service/customer_booking_cancel.go
- apps/api/internal/service/customer_foundation.go
- apps/api/internal/service/booking_occupancy.go
- apps/api/internal/db/queries/booking_reads.sql
- apps/api/internal/db/booking_reads.sql.go (generated)
- apps/api/internal/server/customer_booking_reschedule.go
- apps/api/internal/server/customer_foundation.go
- apps/api/internal/server/customer_booking_reschedule_integration_test.go
- apps/api/internal/server/customer_foundation_integration_test.go
- apps/web/app/my-bookings/reschedule-booking.tsx
- apps/web/app/my-bookings/cancel-booking.tsx
- apps/web/app/my-bookings/workspace.tsx
- docs/CURRENT_STATE.md
- docs/ROADMAP.md
- docs/CUSTOMER_BOOKING_RESCHEDULING.md

TO_REVISE.MD was empty at the start of this task and was left empty.
