# Phase 7.2 - Reviewed public business profiles

Implemented 2026-10-16 after the user instructed proceeding following the explanation of the exact public payload and owner/admin confirmation. Public publishing is opt-in. No existing real business was automatically published.

## Use the feature

1. Sign in at http://localhost:3000/login and open /business.
2. Select your business, save its name/timezone and customization, then scroll to Review your business profile.
3. Click Refresh publishing preview. Review the exact saved business name, timezone, introduction, description, contact email, phone and website. The business ID also appears in its public link/API.
4. An owner/admin checks the confirmation checkbox and clicks Publish reviewed profile. Use View public business profile to open /businesses/{id}; visitors do not need an account.
5. Later edits remain private. Refresh the preview, confirm again and choose Publish reviewed update to replace the public snapshot.
6. Unpublish profile hides subsequent public reads. Already opened pages/copies cannot be retracted; the public page has Refresh profile to load current state.

Manager/staff roles may review but cannot publish. Business membership is checked again on every mutation. This does not expose bookings, customers, membership, staff/resource details or internal audit fields. No customer booking action is provided on the public page yet.

## API and data contracts

GET /api/v1/businesses/{id}/publication is authenticated and tenant-protected. Its profile is the latest saved draft, not necessarily the published snapshot; review_token detects changes to that draft. PUT published=true requires that current token plus owner/admin access and the exact frontend origin. Missing token is 400, changed/foreign-business token is 409. An unchanged retry is safe. PUT published=false unpublishes without needing a review hash.

GET /api/v1/public/businesses/{id} is anonymous, returns only the approved projection, and requires an active business, is_published=true and a confirmed saved snapshot. Hidden, disabled and unknown businesses return 404. Invalid IDs/queries return 400; unsupported methods return 405. Responses are no-store, including errors. Browser reads use credentials=omit; the shared fetch helper now honors that override while preserving authenticated requests' default credentials=include. CORS permits the configured frontend; it is not an authorization boundary for public data.

Snapshot save and publication flag update share a tenant transaction/lock with profile/settings writers. Draft changes are isolated until republished. Both client rendering and the API projection avoid serializing internal models directly. Contact values are displayed as text; markup is not executed.

Migration 19 adds public_confirmed=false for previously prepared snapshots. They remain hidden until a fresh explicit confirmation, even if an old publication flag was true. Migration 19 applied through normal startup; schema 19 is clean. Its down migration refuses to remove the boundary while profile rows exist. No credentials or new manual configuration are required.

## Verification

Database-enabled go test ./... -count=1, go build ./..., go vet ./..., sqlc generation, gofmt and git diff --check passed. Frontend lint/build and isolated Chrome publishing regression passed.

API tests cover anonymous confirmed-profile reads, exact public fields/no-store, unpublished/unknown/disabled/prepared-state denial, stale/missing/foreign preview tokens, draft isolation, transaction rollback, tenant/role/revocation checks and existing customer ownership behavior. Public profile responses do not require or grant customer/business account access.

The browser flow creates temporary fixtures, confirms publishing is initially disabled until the checkbox is checked, publishes, opens the page in a separate unauthenticated context, rejects a stale preview, verifies saved edits stay private, republishes, and verifies unpublishing hides the profile after refresh. No browser page errors. No horizontal overflow at 320/390/768; desktop 1280 and mobile 390 captures, with mobile inspected. Fixtures are cleaned up.

The initial browser run found that the shared fetch helper forced credentials=include, causing the public response's CORS check to fail. Honoring the explicit credentials override fixed it; the flow then passed. Existing authenticated requests keep their prior default.

## Limits and next step

Phase 7.2 is complete; full Phase 7 remains in progress. Next is Phase 7.3 public business discovery and customer-facing service selection, with only active, confirmed published businesses eligible. Customer availability/booking creation and owned history/manage actions follow as separately tested slices. Payments and messaging remain dependent work. No deployment or external hosting was performed; localhost links are for this computer.

Earlier approval-review rejections were resolved by the user's instruction to proceed after the public behavior was explained. No approval request remains pending for this feature. UI polish and broader cross-browser/accessibility, provider setup and production checks remain tracked separately in DEFERRED_WORK.md.

## Phase 7.3 extension - 2026-10-16

PUBLIC_DISCOVERY.md extends the published payload with reviewed service IDs, names and
durations. The confirmation preview explains public directory listing. Existing
snapshots remain link-accessible but require republishing before directory inclusion.
New/edited services require a fresh review; archived services are hidden on the next
request. No automatic publication of existing catalog data was performed.

## Phase 7.4 update (2026-10-16)

Public service/location/date availability and responsive customer UI are complete.
Publication previews now include linked location IDs, names and timezones; existing
snapshots require owner/admin review and republishing. Current scheduling and
occupancy filter advisory times. Selection creates no reservation or booking.
See [PUBLIC_AVAILABILITY.md](PUBLIC_AVAILABILITY.md) for limits and passed checks.
Next is Phase 7.5 authenticated customer-owned booking creation.

## Business setup fix (2026-10-17)

Per the user's request, both Publish actions now save pending workspace forms,
fetch a fresh preview and publish using its token. This supersedes the earlier
requirement to manually save every section and refresh the preview first.
Failed saves stop publication; earlier saves may have completed. Missing locations
produce owner guidance before publishing. Custom staff role labels are private
business vocabulary and do not grant account access. Migration 20 is clean.
See [BUSINESS_SETUP_CLARITY.md](BUSINESS_SETUP_CLARITY.md) for scope, checks and limitations.
Phase 7.7 customer cancellation remains next; no cancellation work was delivered here.
