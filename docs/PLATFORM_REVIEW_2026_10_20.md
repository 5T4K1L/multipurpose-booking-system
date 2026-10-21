# Platform review - 20 October 2026

## Booking fix

The local API/frontend had stopped. They were restarted. The real business also had a single active 24-hour service with daytime-only branch hours. The previous full-interval opening-hours check therefore returned no available dates.

The owner explicitly approved starting whole-day bookings during branch opening hours and continuing after closing. Whole-day availability now checks the start against weekly hours, retains 24 elapsed hours, and still checks explicitly closed dates, buffers, lead time and allocation occupancy across the full reserved period. Timed/custom-length services retain their existing full-interval hours checks. No opening hours or existing customer records were changed. No migration was required.

Read-only confirmation for 21 October 2026: Brgy. San Roque returned 40 available start times; Camiling returned 32. Chrome reached the real business's contact/confirmation form without submitting a booking.

## Design review

Reviewed home, directory, public business/booking flow, My bookings, login, registration, account, business settings, business bookings, calendar and customers. All 11 pages were exercised at 320, 768 and 1440 px in both light and dark modes (66 combinations). Screenshots were inspected for representative desktop/mobile views.

Changes include separated service-name/duration lines, consistent branch/service card grids, quieter calendar actions, less repeated business-settings information, aligned page headings and panel spacing, mobile customer cards, removal of empty contact fields, and clearer whole-day wording.

## Validation and limits

- PostgreSQL-backed go test ./... passed; go build ./... and go vet ./... passed.
- Frontend lint and production build passed.
- Live fixture: publication, guest booking, 24-hour duration, overlap rejection, shortened-duration rejection, reschedule, cancellation and restored availability passed. Only fixture data was created/deleted.
- Whole-day unit regression includes a daylight-saving transition and explicit closed-date protection.
- Browser review found no horizontal overflow or uncaught page errors in the exercised states.
- Authentication UI checks passed with mocked responses. Live external Google consent was not repeated.
- This review used Chrome; it is not a claim of exhaustive Safari/Firefox or assistive-technology certification.

## Changed files

- apps/api/internal/service/slots.go
- apps/api/internal/service/slots_test.go
- apps/api/internal/server/catalog_updates_integration_test.go
- apps/web/app/business/service-duration.tsx
- apps/web/app/business/scheduling.tsx
- apps/web/app/business/page.tsx
- apps/web/app/business/catalog.tsx
- apps/web/app/business/calendar/page.tsx
- apps/web/app/business/customers/page.tsx
- apps/web/app/business/public-profile.tsx
- apps/web/app/businesses/[id]/page.tsx
- apps/web/app/businesses/[id]/availability.tsx
- apps/web/app/globals.css
- apps/web/tests/whole-day.browser.mjs
- apps/web/tests/platform-review.browser.mjs (new)
- docs/WHOLE_DAY_SERVICES.md
- docs/PLATFORM_REVIEW_2026_10_20.md (new)
- TO_REVISE.md
