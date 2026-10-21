# Guest bookings and owner onboarding

Completed 2026-10-19 from TO_REVISE_SOON.md. This revision supersedes the old
customer-sign-in requirement; Phase 8.2 and the remaining visual redesign are not implemented here.

## Try it locally

1. Open http://localhost:3000/register to create a business owner account, or use
   Google from /login. Both paths lead to business setup at /business.
2. Enter the business name, timezone, introduction and contact details. Add
   services, locations, opening hours and any capacity resources needed.
3. Under Profile & publishing, upload a JPEG/PNG photo (maximum 2 MB). Review
   and publish the profile. Pending photo edits participate in save-and-publish.
4. Open /businesses in a separate browser/private window. Select a business,
   service, location and available time. Supply a name and phone; email and notes
   are optional. No customer registration or sign-in is needed.
5. Open /my-bookings to edit contact details, reschedule or cancel. Cancelled,
   completed and rescheduled records can be deleted from the guest list.

Business owners retain booking history when a customer removes a record from
My bookings. Active reservations must be cancelled first so capacity is released.
All existing booking availability, deadline and transactional conflict checks remain.

## Private access and recovery

My bookings uses a random 256-bit browser capability, independent of owner
sessions. The browser stores it locally; the database stores only its SHA-256
hash. A private recovery link imports access on another browser. Its URL fragment
is removed after import and is not sent as an HTTP query parameter.

A recovery link grants access to all guest bookings in that browser collection.
Keep it private. Bookings are not recoverable by entering a name or phone number.
Clearing browser storage without keeping the link loses access. Email delivery,
capability rotation and a support-assisted recovery workflow are not implemented.
There is no automatic expiry for this capability in this revision.

The internal guest principal is a users row solely for existing ownership and
booking audit relationships. It is not a sign-in account: database triggers reject
credentials, OAuth identities, owner sessions and business memberships for it.
Reserved guest.invalid addresses cannot register a password account. Guest
contacts are stored per booking and copied on reschedule. CRM groups booking
history by device principal; it does not verify that two supplied names are the
same person. Business booking details show each booking's actual contact fields.

## API and data

Migration 000022_guest_booking_owners adds guest_visitors, guest_booking_contacts,
business_images and a unique partial index limiting each account to one owned
business. Existing staff/admin membership permissions are preserved; this does
not remove team access to another owner's business.

Guest routes use X-Booking-Access and ignore owner cookies:

- GET/POST /api/v1/guest/bookings
- GET/DELETE /api/v1/guest/bookings/{id}
- GET/PATCH /api/v1/guest/bookings/{id}/contact
- Existing /cancel and /reschedule workflows under the guest booking prefix.

Creation, cancellation and rescheduling retain idempotency keys. Mutations check
the configured browser origin. Unknown access capabilities cannot read records;
a valid guest capability cannot read or modify another guest's bookings or gain
owner access. Lists and detail reads exclude guest-hidden records.

Owner/admin GET/PUT/DELETE /api/v1/businesses/{id}/image handles draft images.
GET /api/v1/public/businesses/{id}/image serves only a reviewed, published image
for an active published business. Decoding/re-encoding strips image metadata;
limits are 6000 pixels per dimension and 24 million pixels overall. An image edit
changes the review token. Draft changes/removal do not alter the public image
until publication. Public profile snapshots include image_version.

Password login and Google callback now lead to /business. Unsigned visitors see
Explore, My bookings and Business sign in, with no owner workspace controls.
The owner form creates the profile's introduction, description, contact details
and website alongside the business. No second owned-business creation is allowed.

## Database reset

The requested reset was completed before the revision. Old accounts, identities,
businesses and bookings were removed; migration history was preserved. Schema 22
is clean. Test fixtures were removed; final verification found zero users,
businesses and bookings. Existing local Google provider configuration is retained.
Create a new owner account or sign in through Google and complete setup again.

A verified pre-reset pg_dump backup is at:

`C:\Users\Alfonzo\Documents\Codex\2026-09-29\referenced-chatgpt-conversation-this-is-an\work\database-backups\booking_db-before-guest-revision-20261019-131315.dump`

## Validation

- gofmt on changed Go files; database-backed go test ./... passed.
- go build ./... and go vet ./... passed from apps/api.
- npm run lint and npm run build passed from apps/web.
- Live browser test: owner onboarding and one-owned-business enforcement; guest
  creation, occupied-slot rejection, contact editing visible to owners, private
  link recovery, cross-guest read/write denial, owner-access denial, rescheduling
  with copied contacts, cancellation, deletion with retained business history,
  private/public image separation, stale review rejection and invalid-image rejection.
- Guest layout checked at 320, 768 and 1280 pixels, with no overflow/page errors.
- Existing mocked booking organization test passed: categories, bulk cancellation,
  partial failure, eligibility, idempotency, inline actions and business completion;
  My bookings, business bookings and account at three widths in light and dark.

Repeatable tests: apps/web/tests/guest-platform.browser.mjs (live local API/schema22)
and organized-bookings.browser.mjs (mock API). Supply PLAYWRIGHT_MODULE when
playwright-core is external, CHROME_PATH if necessary, BROWSER_OUTPUT_DIR for
screenshots and DOCKER_PATH for live fixture cleanup. The live test creates its
own records and removes only those fixtures from the local booking_db container.

## Files changed for this revision

Existing unrelated workspace changes were preserved.

### Database and backend

- migrations/000022_guest_booking_owners.up.sql
- migrations/000022_guest_booking_owners.down.sql
- apps/api/cmd/api/main.go
- apps/api/internal/db/queries/guest_bookings.sql
- apps/api/internal/db/queries/customer_foundation.sql
- apps/api/internal/db/queries/booking_reads.sql
- apps/api/internal/db/queries/customer_crm.sql
- apps/api/internal/db/guest_bookings.sql.go
- apps/api/internal/db/customer_foundation.sql.go
- apps/api/internal/db/booking_reads.sql.go
- apps/api/internal/db/customer_crm.sql.go
- apps/api/internal/db/models.go
- apps/api/internal/service/guest_bookings.go
- apps/api/internal/service/guest_bookings_test.go
- apps/api/internal/service/business_image.go
- apps/api/internal/service/business.go
- apps/api/internal/service/business_test.go
- apps/api/internal/service/registration.go
- apps/api/internal/service/customer_booking_create.go
- apps/api/internal/service/customer_booking_reschedule.go
- apps/api/internal/service/booking_reads.go
- apps/api/internal/service/business_publication.go
- apps/api/internal/server/guest_bookings.go
- apps/api/internal/server/business_image.go
- apps/api/internal/server/business.go
- apps/api/internal/server/oauth.go
- apps/api/internal/server/oauth_test.go
- apps/api/internal/server/oauth_integration_test.go
- apps/api/internal/server/customer_foundation_integration_test.go
- apps/api/internal/server/business_pagination_integration_test.go
- apps/api/internal/repository/business_integration_test.go

### Frontend

- apps/web/app/bookings/api.ts
- apps/web/app/bookings/workspace.tsx
- apps/web/app/components/booking-organizer.tsx
- apps/web/app/components/site-header.tsx
- apps/web/app/my-bookings/guest-access.ts
- apps/web/app/my-bookings/guest-contact-editor.tsx
- apps/web/app/my-bookings/workspace.tsx
- apps/web/app/my-bookings/cancel-booking.tsx
- apps/web/app/my-bookings/reschedule-booking.tsx
- apps/web/app/businesses/[id]/booking-confirmation.tsx
- apps/web/app/register/page.tsx
- apps/web/app/login/page.tsx
- apps/web/app/page.tsx
- apps/web/app/business/layout.tsx
- apps/web/app/business/page.tsx
- apps/web/app/business/business-image.tsx
- apps/web/app/business/catalog.tsx
- apps/web/app/business/public-profile.tsx
- apps/web/app/business/publication.tsx
- apps/web/tests/guest-platform.browser.mjs
- apps/web/tests/organized-bookings.browser.mjs

### Documentation

- docs/GUEST_BOOKINGS.md
- docs/CURRENT_STATE.md
- TO_REVISE_SOON.md (completed list cleared)

## Next work

Continue the remaining page/interaction revisions in TO_REVISE.MD, then Phase 8.2
customer filtering and sorting. No public deployment was performed. Live Google
sign-in itself was not repeated; callback/session regression tests passed.
Account usage balance is not exposed to this agent.
