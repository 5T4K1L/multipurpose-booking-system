# Phase 7 completion review

Reviewed 2026-10-18. Phase 7.9 closes the customer-booking development scope.
This is not production approval or closure of unrelated deferred release checks.
Schema remains 21, dirty=false; this review adds no migration.

## Roadmap coverage

| Area | Implemented flow | Evidence |
| --- | --- | --- |
| Customer account/profile | Registration, password/Google sign-in, read-only name/email/account view, logout | /register, /login, /account; authentication tests and existing milestone guides |
| Search/discovery | Public published-business search and pagination | PUBLIC_DISCOVERY.md |
| Business pages | Owner-reviewed published snapshot with contact and service/location details; unpublish | PUBLICATION_REVIEW.md |
| Service selection | Customer selects a published service and branch | PUBLIC_DISCOVERY.md, PUBLIC_AVAILABILITY.md |
| Availability | Location hours/closures, full-duration checks, staff conflicts and resource capacity; selectable month calendar | PUBLIC_AVAILABILITY_CALENDAR.md |
| Booking | Authenticated customer ownership, atomic validation/allocation, idempotent confirmation | CUSTOMER_BOOKING_CREATION.md |
| Custom duration | Per-service owner opt-in/min/max, reviewed publishing, customer start/end selection across dates | CUSTOMER_BOOKING_RANGES.md |
| History | Owned list/detail; original/replacement links retained independently of publication | CUSTOMER_BOOKING_HISTORY.md |
| Cancellation | Deadline checked, history retained, capacity released, retries safe | CUSTOMER_BOOKING_CANCELLATION.md |
| Rescheduling | Available new time, unchanged service/branch/duration/assignment, atomic replacement and safe retry | CUSTOMER_BOOKING_RESCHEDULING.md |

These areas have functional responsive UI. Profile here means the authenticated
account identity view; editable identity/email, avatars, saved preferences and richer
customer profiles are not delivered. The aspirational Find/Compare/Book/Pay/Manage/
Message/Return sequence is not a claim that payments or messaging are complete:
those belong to Phases 12 and 9. Search/marketplace expansion remains Phase 11.

## Gaps resolved in this review

The deferred owner-to-customer range flow now passes. Explicit accessible names
were added to Booking mode, Start date and time, and End date and time controls.
Owner guidance now describes location opening hours and existing capacity instead
of requiring separate staff/resource windows. Customer time wording was corrected.
Historical notes saying that the real business still requires individual availability
windows are superseded by the approved 2026-10-18 opening-hours policy.

## Fresh verification

- Database-enabled go test ./... -count=1; go build ./...; go vet ./... passed.
- Frontend lint and production build (with access to existing Google Fonts) passed.
- Local-time/DST and compact date-range formatting tests: 4 passed.
- Range browser flow: owner checklist, opening hours, service rules, custom duration,
  saving pending changes during publication, customer 25-hour overnight booking,
  duplicate occupied-range rejection and cancellation passed. No individual resource
  windows were needed. Tested widths 320, 390, 768 and 1280; no page errors/overflow.
- Cancellation browser regression: keep-current, lost-response retry, business state,
  released capacity and retained history passed at the same four widths.
- Rescheduling browser regression: keep-current, lost-response retry, business state,
  old/new capacity, linked history and unpublished-business management passed at the
  same four widths.
- Inspected the 320px range screenshot; controls and confirmation fit the viewport.
- All browser-created fixtures were removed. Real business settings were not changed.

## Carry-forward limitations

- Availability refresh is polling/focus/manual plus final server validation, not push.
- Profile editing, final visual redesign, broad cross-browser/accessibility checks,
  load testing and production readiness are not part of this closure.
- Apple live setup, the Go race-detector toolchain, renewed live Google consent test,
  shared production OAuth/limiter state and deployment checks remain deferred.
- Local native datetime input rejects ambiguous/nonexistent daylight-saving wall
  times rather than guessing. Custom durations are 1 minute to 7 days within the
  owner-configured min/max and must fit the entire schedule including overnight time.
- Customer rescheduling retains existing assignments and duration; it does not
  switch service or branch. See individual guides for pagination/query limits.

## Next boundary

Phase 8.1: define tenant-scoped CRM customer records and their relationship to
platform accounts and bookings, then deliver a basic customer table and UI.
Cross-business data must never leak; bulk operations and import/export follow their
own validated, previewed workflows. No Phase 8 implementation was added here.

## Files edited in this review

- apps/web/app/business/booking-options.tsx
- apps/web/app/businesses/[id]/booking-range.tsx
- apps/web/app/businesses/[id]/availability.tsx
- docs/PHASE_7_CLOSURE.md
- docs/CUSTOMER_BOOKING_RANGES.md
- docs/CURRENT_STATE.md
- docs/ROADMAP.md
- docs/DEFERRED_WORK.md
- docs/PROJECT_CONTEXT.md

TO_REVISE.MD was empty and remains empty. Account usage cannot be read by the agent;
the most recent user report was a reset before Phase 7.8.
