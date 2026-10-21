# Public availability - Phase 7.4

Update 2026-10-17: the customer date input is now a monthly availability calendar
with automatic refresh and remaining-capacity indicators. The single-day API
below remains supported. See [PUBLIC_AVAILABILITY_CALENDAR.md](PUBLIC_AVAILABILITY_CALENDAR.md).

Completed 2026-10-16. Schema remains version 19; no migration required.

## Try it

1. In `/business`, configure an active service, link an active location, save its weekly hours and the service scheduling rules. Configure availability windows for any linked staff/resources.
2. Refresh the publishing preview, review the service and linked location names, IDs and timezones, then publish. Older published snapshots require this explicit review and republish before availability appears.
3. Open `/businesses` without signing in, find the business, select a service, location and date, and click **Check availability**.
4. Select a time to inspect it. This does not reserve capacity or create a booking.

## Contract and policy

`GET /api/v1/public/businesses/{id}/services/{serviceID}/availability?location_id={uuid}&date=YYYY-MM-DD`

Anonymous read, no-store, exact allowed browser origin. Only these two query keys are accepted, once each. Dates span 2000-2100. The endpoint allows 60 requests per peer per 15 minutes and uses a 10-second context.

The business must be active, published and explicitly confirmed. Service and location must be in its reviewed snapshot and remain active and linked. Unpublished, foreign, removed or archived selections return 404. A changed live service duration or location timezone returns 409 until the owner republishes.

The response contains date, location timezone, advisory=true, and sorted unique UTC starts_at/ends_at pairs. It reuses the existing 15-minute local grid, DST handling, weekly hours, closures, availability windows, buffers, lead time and booking occupancy calculations.

For each configured allocation type, at least one eligible active staff/resource choice is required. A type is optional only when the service has no links for that type. Available times are the union across valid combinations; staff exclusivity and resource capacity still apply. Internal allocation IDs/names, capacity and customer/booking data are not returned. Reads share the business transaction lock with mutations for a consistent result.

At most 32 allocation combinations are evaluated. Larger configurations return 503 rather than incomplete availability. Publication reviews support 1,000 services and 2,000 service-location links; oversized drafts can still be unpublished. No background publishing or automatic location disclosure occurs.

## Validation

Database-enabled `go test ./... -count=1`, `go build ./...`, `go vet ./...`, gofmt, frontend lint and production build passed. Integration coverage includes tenant/publication boundaries, malformed queries, assignment unions/deduplication, resource capacity, cancellation, closures, buffers, lead time, archived staff, removed location links, stale duration/timezone and configuration bounds.

Browser verification covered anonymous discovery, location/date selection, 31 expected fixture times with optional allocations, selection without reservation, reviewed publication, private drafts, unpublish, and no overflow at 320/390/768/1280 pixels. No browser runtime errors occurred; temporary test data was removed.

## Next

Phase 7.5: authenticated customer-owned booking creation from a published business. Revalidate publication, eligibility and occupancy and resolve an allocation atomically at submission. Never trust a previously displayed available time as a reservation. Customer history and management UI remain later work.

## Phase 7.5 completed (2026-10-16)

Customers can now confirm a booking from a published business after signing in.
Ownership, assignment and current availability are enforced atomically; idempotent
retries recover the saved booking without duplication. The responsive confirmation
UI includes a booking reference. Schema stays 19. See
[CUSTOMER_BOOKING_CREATION.md](CUSTOMER_BOOKING_CREATION.md) for the contract and tests.
Next: Phase 7.6 customer booking history and detail UI; full Phase 7 remains open.


## Business opening hours policy (2026-10-18)

Customer calendars and booking validation now use the selected location's weekly opening hours and closed dates. Assigned staff and resources follow those hours; saved individual availability windows no longer constrain slots. Existing window data is retained for compatibility, but the business scheduling UI no longer presents it as a booking requirement. This supersedes earlier instructions requiring staff/resource windows.

Service duration, preparation/cleanup, advance notice, active assignments, staff booking conflicts and resource capacity remain enforced. Creation and rescheduling use the same rules as the public calendar. Owners manage availability under Business > Scheduling > Location hours and closed dates. No migration or change to saved business hours is needed.
