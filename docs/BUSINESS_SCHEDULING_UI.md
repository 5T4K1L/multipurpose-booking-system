# Phase 6.2 - Business scheduling UI

Completed 2026-10-13. Phase 6 remains in progress.

## Open and test

Open http://localhost:3000/business, sign in and select your business. Scroll to
Scheduling. Owner/admin can save; manager/staff can inspect saved values only.

1. Select Location hours and closed dates, then a location. Add opening intervals,
   choosing weekday and 24-hour HH:MM times. End-of-day 24:00 is supported. Split
   overnight hours between days. Multiple non-overlapping intervals allow breaks.
2. Add closed dates for holidays or other full-day closures. Save scheduling.
   Hours and closed dates use the location timezone displayed above the form.
3. Select Service buffers and notice, then a service. Save preparation, cleanup and
   minimum advance notice in minutes. Explicitly save zero rules when appropriate;
   a service with no saved rules is not configured for bookings.
4. Select Staff availability or Resource availability. Add dated start/end windows
   in UTC (not your browser timezone). Each window lasts at most seven days; up to
   32 non-overlapping windows are supported. Include preparation and cleanup time.
5. Reload saved configuration to verify persistence. Use /bookings with matching
   assigned entities and dates to inspect available times. Times remain advisory.

No hours means a closed location; no windows means an unavailable staff/resource.
Remove all relevant rows and save to clear configuration. Removal is only a local
edit until saved. Refresh scheduling items after creating new catalog items.

## Behavior and boundaries

Uses existing GET/PUT scheduling APIs and current server-side tenant authorization.
No backend, schema, package or authentication changes. No Go files required gofmt.
Saves replace the full configuration of the selected item; an uncertain response can
be checked by reloading. Concurrent editors use last-committed replacement semantics;
there is no version conflict protection. The UI states this explicitly.

Changing scheduling category/item or reloading warns before discarding dirty edits;
leaving/reloading the document also requests a browser warning. Save before switching
businesses or navigating within the app: drafts are not persisted across those actions.
Read failures block saving; rejected access blocks further edits until a successful
reload. API authorization remains authoritative after membership changes.

Weekly hours are local wall-clock values. Availability windows are explicitly UTC,
with minute precision and years 2000-2100; local timezone conversion and recurring
shift generation are not provided by this version. The API enforces the same window,
rule and schedule limits. A size check prevents sending configurations over 4096 bytes.
Existing confirmed bookings remain unchanged by future scheduling edits.

## Verification

- Frontend ESLint, Next production build and TypeScript passed.
- Database-enabled go test ./... -count=1, go build ./... and go vet ./... passed.
- Headless Chrome: all four scheduling categories saved successfully; midnight
  closing persisted; overlap rejected; edited hours survived row additions/removals.
- A UTC 24-hour day with a 30-minute service and staff/resource windows yielded
  95 advisory slots. Adding that date as closed yielded zero slots.
- Staff read-only controls and HTTP 403 for direct schedule PUT verified.
- No browser page errors or horizontal overflow at 320, 390 and 768 widths.
  Desktop 1280 and mobile 390 screenshots reviewed. Temporary fixtures cleaned up.

Next: Phase 6.3 business profile and catalog editing. Business booking management,
completion status, calendar/dashboard and remaining business settings still belong
within Phase 6. Customer-facing discovery, ownership and self-service are Phase 7.
