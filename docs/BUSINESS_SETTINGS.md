# Phase 6.6 - Business customization and team settings

Completed 2026-10-14. Phase 6 remains in progress pending its completion review.

## Use

Open http://localhost:3000/business, sign in and select your business. Below Scheduling:

1. Business customization stores a short introduction, description, business contact
   email, phone and HTTPS website. Owner/admin can edit. Save business details and
   reload to verify the Saved business preview. Manager/staff can read saved details.
2. All fields are optional. Saving blank values clears them; every save replaces all
   five fields. Reload warns before discarding unsaved changes. Save before switching
   businesses/navigation; drafts are not retained. Concurrent edits are last-write-wins.
3. Team access is visible to owner/admin. Only the owner can add/update/remove access.
   Ask an existing active account holder for their Account ID from /account, enter it,
   choose a role and confirm the account/role before saving. Removal also confirms.
4. BUSINESS_ADMIN manages catalog, scheduling and bookings, but cannot manage members.
   MANAGER and STAFF currently have read-only business access. Owner replacement,
   demotion, removal and transfer remain prohibited by the existing service.

Membership grants take effect immediately. No invitation/email is sent. Bookable staff
profiles are separate from account membership. Adding a staff profile never grants
login access. The team list is paginated and labels accounts by ID; name/email lookup
and invitation acceptance workflows are not implemented.

## Customization API and storage

GET /api/v1/businesses/{businessID}/settings returns tagline, description, contact_email,
contact_phone and website, defaulting to empty strings before first save.
PUT on the same URL requires all five string fields; blank strings are allowed.
Missing/null fields, extra fields, nonobjects and trailing JSON are rejected. Query
parameters are not accepted. Requests retain the existing exact Origin/session/rate
controls, 4096-byte JSON limit, deadlines, no-store headers and generic error mapping.

Limits: introduction 120 characters, description 2000, email 254, phone 40, website 500.
Values are trimmed, UTF-8 checked and controls rejected (description allows newline/tab).
Email must be a bare valid address; website must be HTTPS with a hostname and no
embedded credentials. No external URL is fetched. Description is plain text, safely
escaped by React; it is not HTML. The total serialized UTF-8 request must still fit
4096 bytes, so long multibyte content may reach the body limit before character limits.

Only owner/admin may PUT. All active tenant roles may GET. The server derives business
identity from the URL and current authorized transaction, not client-supplied identity.
No platform override. Storage/commit failure yields no successful result; transaction
rollback preserves prior settings. Settings do not change scheduling, bookings or
membership. Name/timezone remain in their existing profile editor/API.

Migration 16 adds business_settings keyed by business ID, with server-managed update
time. Settings cascade only if their business is deleted; no business deletion UI is
added. Down migration refuses to discard nonempty customization. Schema is 16, clean.
Membership UI reuses existing membership APIs and owner-only policy without widening
permissions. Account ID is displayed only to the signed-in account holder.

## Boundaries

Saving customization does not publish a customer page. Public discovery and profile
exposure require Phase 7's explicit public contracts. Logo uploads, themes, service
custom-question builders, pricing, custom cancellation policies, invitations and
customer self-service are not included here. UI redesign remains deferred as requested.

## Verification

- sqlc generation and gofmt; database-enabled go test ./... -count=1, go build ./...
  and go vet ./... passed. Frontend lint/build/TypeScript passed.
- Integration checks: default/persisted/cleared settings, normalization, unsafe URLs,
  malformed email/JSON, required fields, body limit, anonymous/foreign/staff rejection,
  admin writes, disabled business denial, rollback preservation and safe public fields.
- Membership checks: owner-only grants/removals, no owner overwrite/removal, immediate
  denial after revocation. Downgrade refuses saved customization.
- Headless Chrome: saved details survive reload, script-like text is displayed as text,
  and owner adds/updates/removes a newly created temporary target account. No browser
  errors or horizontal overflow at 320/390/768 widths. Mobile preview inspected.
- Browser test avoids changing the acting account's role; staff authorization is
  covered by API integration tests. Temporary fixtures cleaned up.

Next: Phase 6.7 completion review and gap closure. Review existing business features
against the agreed Phase 6 scope, preserve deferred work and document the Phase 7
customer ownership boundary before declaring the full phase complete.
