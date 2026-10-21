# Phase 6.5 - Business calendar and dashboard

Completed 2026-10-14. Phase 6 remains in progress. UI polish is deferred by user
instruction; this milestone provides a functional agenda and booking totals.

## Open and test

Open http://localhost:3000/business/calendar, or choose Calendar and dashboard from
business setup or business bookings. Sign in and select a business you belong to.

1. Choose From date and Through date, then Apply dates (1-31 inclusive days).
2. Today selects the current day in the business timezone. Next 7 days includes
   today and the following six days. Refresh dashboard reloads the applied range.
3. Inspect All records, Confirmed, Completed, Cancelled and Moved totals.
4. The calendar agenda groups records by starting date and orders by start time,
   breaking equal-time ties by reference. Expand Booking details for the reference
   and buffered interval. Load more bookings retrieves subsequent pages.
5. Manage bookings opens the existing business management page for changes.

All active business roles (owner/admin/manager/staff) can read this page. It makes no
booking mutations. Nonmembers, inactive/revoked memberships and other businesses are
not accessible; platform status never bypasses membership.

## Date and count semantics

Dates and displayed times use the current business timezone, not the browser timezone
or each location's timezone. A record belongs to the local date it starts; a booking
that starts before the range and overlaps it is not included. Both repeated hours on
DST fall-back days are included correctly; dates are not assumed to last 24 hours.
Cross-midnight entries display full start and end dates.

Summary counts cover the entire applied range regardless of loaded pages. Counts are
historical booking records, not unique customer visits or revenue: a moved predecessor
and its replacement count separately if both start in range. All five status totals
are returned by the server. Labels reflect current catalog names, including archived
entities; they are not snapshots of names at booking time.

## API

GET /api/v1/businesses/{businessID}/bookings/calendar?from=YYYY-MM-DD&to=YYYY-MM-DD
Optional limit=1..100 (default 100) and cursor; UI requests 50 records per page.
Range must be ordered, at most 31 inclusive days, with years 2000-2100. Unknown,
duplicate, malformed or incomplete parameters are rejected with 400. GET and OPTIONS
only; existing authentication, Origin checks, no-store headers and deadline apply.

Response: from, to, timezone, counts, items and next_cursor. Items contain the existing
safe booking view plus service_name/location_name/staff_name/resource_name. No creator,
request hashes or idempotency keys are exposed. Empty results are items:[] and zero
counts. The response is published only after the authorized tenant transaction commits.
Counts and page rows share the tenant lock with supported business writers.

Cursors bind actor, business, date range and timezone and resume after immutable
(start timestamp, reference). They are selectors, not authorization. Every page checks
current membership. Changing business timezone or date range requires a fresh page.
Pages are fresh reads, not an export snapshot; refresh after changes to loaded records.

## Verification and limits

- sqlc generation and gofmt completed; no new schema migration (version 15).
- Database-enabled go test ./... -count=1, go build ./... and go vet ./... passed.
- Frontend ESLint, production build and TypeScript passed.
- API integration: 108 records over three pages without duplicates, equal-time ties,
  New York's 25-hour day including both repeated 01:30 instants and day boundaries,
  full totals on every page, role/tenant/cursor isolation, archived labels, safe fields,
  strict query validation and timezone-change cursor rejection.
- Headless Chrome: selected date range showed the completed fixture with total=1,
  completed=1; an empty range showed total=0. No page errors or horizontal overflow at
  320/390/768 widths. Mobile screenshot inspected. Temporary fixtures removed.

This is a date-grouped agenda, not a drag/drop month grid. No live push, exports,
revenue metrics, staff/location filters or customer identity details in this milestone.
Business-local date filtering scans that tenant's booking history; large-history query
optimization and production load verification remain later work. UI redesign stays
deferred until the platform's functional implementation is complete, as requested.
Next: Phase 6.6 business customization/settings, then remaining Phase 6 review.
