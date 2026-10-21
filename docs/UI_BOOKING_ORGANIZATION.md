# Booking organization and account design

2026-10-19. Frontend-only; existing API contracts, permissions, deadlines and history policies remain unchanged.

## Delivered

- Customer and business bookings have All, Confirmed, Completed, Cancelled and
  Rescheduled categories, with counts for loaded records. Pagination explicitly
  describes its scope; selections never silently include unloaded records.
- Separate booking cards contain their own details/actions. Business rescheduling,
  cancellation and completion confirmations appear beneath the associated booking.
  Original/replacement details stay anchored to the card that opened them.
- Customers can bulk cancel. Owners/admins can bulk cancel or mark completed.
  Confirmation shows eligible/skipped counts. Requests use the existing idempotency
  helper and backend validation; successful items update and deselect, failed items
  remain selected with per-booking results. Authentication/network/server failures
  stop remaining requests. Batches are not atomic.
- Review selected narrows the list for individual edits. Customer detail reads load
  cancellation/rescheduling eligibility, which the list endpoint does not supply.
  Rescheduling still requires a separate availability choice for each booking.
- No permanent delete operation exists; cancellation retains history and releases
  capacity. This change does not add destructive deletion or account editing APIs.
- Business settings have section shortcuts. Team access now lives with people and
  resources; public profile customization and publishing remain together. Disclosure
  forms stay mounted to preserve drafts and publish-save registration.
- Account page has a profile panel, personal information, booking/workspace actions,
  a secondary account-reference disclosure and a separate sign-out area.

## Validation

- Frontend lint, TypeScript and production build passed.
- Browser fixtures cover categories, selection, confirmation, partial failures,
  eligibility, idempotency keys, inline details/reschedule/cancel and bulk completion.
  Pagination and read-only staff controls also passed.
- Light/dark responsive checks on both booking pages and account at 320/768/1280px;
  no document overflow or browser page errors. Screenshots inspected.
- Live API: two temporary customer bookings cancelled through bulk UI; persisted
  cancellation and business CRM counts verified. Fixtures cleaned up.
- Existing live rescheduling regression passed, including lost-response retry,
  history, capacity transfer and business visibility.
- Existing workspace test passed: settings disclosures preserve unsaved staff input,
  customer/owner navigation and CRM search/count updates remain functional.

Saved browser regression: apps/web/tests/organized-bookings.browser.mjs.
Run against localhost:3000 with Node, playwright-core available (or set
PLAYWRIGHT_MODULE to its absolute module path), and Chrome installed (CHROME_PATH
can override its executable). This test mocks API responses. Screenshots go to
BROWSER_OUTPUT_DIR or the OS temporary directory.

## Edited files

- apps/web/app/components/booking-organizer.tsx (new)
- apps/web/app/my-bookings/workspace.tsx
- apps/web/app/bookings/workspace.tsx
- apps/web/app/account/page.tsx
- apps/web/app/business/catalog.tsx
- apps/web/app/business/settings.tsx
- apps/web/app/globals.css
- apps/web/tests/organized-bookings.browser.mjs (new)
- docs/UI_BOOKING_ORGANIZATION.md (new)
- docs/CURRENT_STATE.md
- TO_REVISE.MD


## Customer booking names correction

Cards show service as the heading, with labeled business and branch names always
visible underneath. Date/time and small reference IDs remain inside Details.
The redundant View booking details link remains removed.

The previous public-profile lookup was insufficient for booking history and has
been removed. Customer list/detail queries now join the booking's own business,
service and location to return business_name, service_name and location_name.
Service/location joins include business_id; customer ownership and active-user
checks remain unchanged. Names remain available for unpublished businesses and
archived catalog entries. These are current names, not historical snapshots.
No migration or booking-policy change is required. Creation and rescheduling
responses also include names for newly returned bookings.

Validation: gofmt, go test ./..., go build ./..., database-backed
TestCustomerFoundationIntegration (names, unpublished business, ownership),
frontend lint/build, browser fixtures using My Business / Wedding Photography /
Brgy. San Roque, and live booking-name assertions with bulk cancellation/CRM
regression. Local API restarted with the tested build. Fixtures cleaned up.

Files changed in this correction:
- apps/api/internal/db/queries/customer_foundation.sql
- apps/api/internal/db/customer_foundation.sql.go (generated)
- apps/api/internal/service/customer_foundation.go
- apps/api/internal/service/customer_booking_cancel.go
- apps/api/internal/service/customer_booking_create.go
- apps/api/internal/service/customer_booking_reschedule.go
- apps/api/internal/server/customer_foundation_integration_test.go
- apps/web/app/my-bookings/workspace.tsx
- apps/web/tests/organized-bookings.browser.mjs
- docs/UI_BOOKING_ORGANIZATION.md
