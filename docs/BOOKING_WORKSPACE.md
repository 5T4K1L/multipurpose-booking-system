# Phase 5.7 - Booking reads, occupied slots and test UI

Completed 2026-10-13. Phase 5 core-booking development scope is complete with a
working test UI. This is not production sign-off; DEFERRED_WORK.md remains applicable.

## Open and test

Open http://localhost:3000/bookings with the API running at http://localhost:8080.
The home and account pages link to the booking workspace. Sign in first with your
existing password or Google account. Google remains enabled; Apple stays disabled.

1. Expand **Create a demo workspace**, enter a name and IANA timezone (for example
   Asia/Shanghai or UTC), optionally choose a closed date, and submit.
2. The demo creates a new saved business, location, 30-minute service, specialist,
   capacity-two room, eligibility links, daily 09:00-17:00 hours, 15-minute buffers,
   30-minute notice and seven days of staff/resource availability.
3. Choose a date within that window and click **Find available times**. Select a
   time and **Confirm booking**. Only selected staff/resources are reserved.
4. Refresh times: intervals blocked by staff or full resource capacity disappear.
   **Details** shows current state, preparation/occupancy times and history links.
5. Use **Reschedule**, choose another available time and confirm. The original stays
   in history as Moved; its linked replacement is confirmed.
6. Use **Cancel booking**, then **Confirm cancellation**. Refresh times to see released
   capacity. Terminal history remains visible; new changes require a confirmed booking
   before preparation starts.

Existing business members can choose their workspace and load more businesses or
bookings. OWNER/ADMIN can mutate; MANAGER/STAFF have view-only controls and current
server-side authorization. All displayed times use the selected location's timezone,
shown on screen. Slots expose distinct offset-aware instants during DST repeats.

Demo setup creates real local test data through existing APIs only. It is not one
atomic transaction: partial setup remains saved if a request fails. The screen reports
this and offers a fresh demo; it never deletes existing data. Demo windows expire after
seven days and are absolute intervals, not recurring staff schedules. Arbitrary catalog
editing, availability maintenance and polished business dashboards belong to Phase 6.

Booking mutations retain an actor/path/payload-specific random idempotency key in
sessionStorage while a response is uncertain. Retrying the same action uses that key;
success or a definite client rejection removes it. Session credentials are never stored
there. Creation/lifecycle retry responses are historical acknowledgements; the UI reads
the current booking afterward. Cancellation has an explicit, keyboard-focusable
confirmation. Error and pending states prevent accidental repeated actions.

The simple rescheduling picker uses normal occupied-slot queries, so times overlapping
the original booking are hidden even though the transactional rescheduling API can
accept such moves. This conservative UI limitation is displayed. Lists are UUID-ordered,
not chronological. This is a test workspace, not the complete Phase 6 calendar.

## Booking read APIs

- GET /api/v1/businesses/{businessID}/bookings?limit=25&cursor=...
- GET /api/v1/businesses/{businessID}/bookings/{bookingID}

Both require an active session/business membership: OWNER, ADMIN, MANAGER or STAFF.
No platform role bypass exists. History is readable even when its catalog is archived.

Lists include all confirmed/cancelled/rescheduled records, ordered by ascending UUID.
Limit defaults to 100 and supports 1-100. X-Next-Cursor is exposed through credentialed
CORS. Cursors are version/kind/actor/business-bound selectors and reauthorize every
page; pages are fresh reads, not a snapshot export. Empty results are []. Unknown or
malformed/duplicate query parameters are rejected. Item GET takes no query parameters.

Public fields: id, service_id, location_id, staff_id, resource_id, starts_at, ends_at,
busy_starts_at, busy_ends_at, status, supersedes_booking_id, replacement_id. UTC timestamps
and optional null IDs are explicit. No creator, key, request digest, credentials or
internal audit fields are returned. Current-state reads differ from historical retries.

GET can omit Origin; a supplied foreign Origin is denied. Responses are no-store and
use a ten-second deadline. Errors: 400 validation/pagination, 401 authentication,
403 current access denied, 404 missing/other-business booking, 503 storage/commit
failure. Data and cursor are published only after the tenant transaction succeeds.

## Occupancy-aware advisory slots

The existing slot API now filters current confirmed bookings for the selected staff
and/or resource, across all services and locations for those identities. Buffers apply
to both stored occupancy and proposed service intervals. Staff is exclusive; resource
capacity uses peak simultaneous overlap via grouped start/end events. Half-open
intervals permit adjacent reservations. Cancelled/rescheduled predecessors do not
occupy capacity. With neither staff nor resource selected, location/service availability
is unchanged; no implicit shared resource or staff is inferred.

Candidate instants stay ordered, including DST repeats. One bounded SQL query filters
the calculated candidates inside the same tenant transaction. Existing occupancy
indexes from migration 14 are reused; no new migration is needed. Results remain
advisory: another transaction may reserve them afterward. Booking/rescheduling inserts
retain their database guards and do not depend on advisory results for authorization.

## Verification and boundaries

Passed on 2026-10-13:

- sqlc generation, gofmt, PostgreSQL-enabled go test ./... -count=1, go build ./...,
  and go vet ./... from apps/api.
- Frontend eslint and Next.js production build, including TypeScript checks.
- Real API integration: role/tenant/cursor isolation, 111 historical rows paginated
  without duplicates, archived-history reads, safe JSON fields, buffer conflicts,
  capacity-two filtering, adjacent-overlap sweep and cancellation/move release.
- Isolated headless Chrome: login, demo setup, booking, occupied-slot reduction,
  rescheduling, cancellation and linked history. 29 available slots became 25 after
  booking and returned to 29 after cancellation; zero browser page errors.
- Desktop 1280x900 and mobile 390x844 screenshots inspected; no mobile horizontal
  overflow. Temporary automated accounts/businesses/bookings were cleaned up.
- Live API health and Google discovery. Local schema remains migration 14, clean.

Interactive computer-use tooling could not initialize; the isolated headless test
provided booking-flow and visual coverage. This does not complete all earlier browser,
accessibility, race-detector or production checks. There is no customer self-service,
notifications, payments, public booking frontend or production load-test claim.

Next: Phase 6.1 business workspace and catalog-management UI. As instructed, every
full phase must include a usable UI and testing instructions before completion.
