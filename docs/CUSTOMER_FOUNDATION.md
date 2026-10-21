# Phase 7.1 - Customer ownership and private publication foundation

Implemented 2026-10-14. This is a backend foundation, not completed customer self-service or full Phase 7. No public business profile route is exposed yet.

## Ownership contract

Migration 17 adds nullable bookings.customer_id, independently of created_by (the operation's actor). Existing internal/test/operator bookings remain NULL; there is no email-based, creator-based or membership-based backfill. Existing internal creation continues to produce unassigned bookings. No HTTP client can assign or transfer customer ownership in this milestone.

The customer read route is GET /api/v1/customer/bookings/{id}. It requires an active session and matches the persisted customer_id to the authenticated actor. It does not require business membership. Unknown and differently owned bookings both return 404; anonymous/disabled sessions return 401. Query parameters, including customer_id, are rejected. Spoofed identity/role headers do not grant access. Responses are no-store and omit customer/creator identifiers, request hashes, idempotency keys and internal staff/resource identifiers.

Business routes retain their existing tenant membership rules. A business can read/manage bookings belonging to that business, including customer-owned bookings, but cannot access another business's records. Business membership does not grant access through customer routes. No global role bypass is introduced.

Customer ownership is immutable under the existing database transition guard. Rescheduling explicitly copies it to the replacement and a new insert trigger rejects mismatched replacements, including attempts to turn an owned record into an unassigned record. Cancellation/completion retain ownership/history. Customer reads use current persisted state, so business changes are visible on the next request. This does not implement live push.

Historical customer reads remain available if a business is unpublished or disabled. The customer's own account must remain active. A future customer creation route must derive ownership from authentication and enforce eligibility, published/active business state, occupancy, idempotency and transaction rules. Do not infer ownership for existing records.

## Private publication state

Businesses default to is_published=false. GET /api/v1/businesses/{id}/publication is readable by current active business members. PUT on that route accepts exactly a required, non-null boolean published field and requires current owner/admin membership, exact origin and existing mutation limits. Foreign tenants, staff/manager writes, revoked access and disabled businesses are denied. Writes use the existing tenant transaction/lock.

This is stored state only: there is NO /api/v1/public/businesses/{id} implementation or public profile UI, so changing it does not currently expose a business. No existing real business was published by this work. Automatic approval review rejected an earlier proposed public route because contact details could be exposed before a concrete publishing review flow. The safer implementation omits that route.

Before introducing public reads, build the owner/admin review UI showing the exact public fields and a clear publish/unpublish action. Require fresh explicit confirmation through that UI; do not treat a pre-existing foundation flag as approval to newly expose contacts. Public reads must require active AND published state, use a safe projection, and return 404 for hidden businesses. Explain that later edits to public fields become visible while published. Customer discovery and booking creation must share this boundary.

## Migration and compatibility

Migration 000017 applied locally through the normal startup runner; schema 17 is clean. Regenerated sqlc code reads the added columns while existing response DTOs remain unchanged. The down migration refuses to discard assigned customer ownership or true publication state. Do not delete real history merely to force a downgrade.

No frontend changes or new credentials/configuration are required. The existing /business, /business/bookings and /business/calendar remain usable; customer creation/list/manage UI is still pending. Full Phase 7 must include a responsive customer UI under the phase-end rule.

## Verification

- gofmt on changed Go sources and sqlc generation.
- Database-enabled go test ./internal/server -run TestCustomerFoundationIntegration -count=1 -v.
- Database-enabled go test ./... -count=1, go build ./..., go vet ./... from apps/api.
- git diff --check.
- Existing isolated Chrome calendar/completion regression: completed count 1, empty-period count 0, date selection, no page errors; responsive overflow checks at 320, 390 and 768 with 1280 desktop capture.
- API health 200 and schema version 17, dirty=false.

The new integration test uses temporary accounts/businesses, cleans fixtures and covers cross-customer/cross-business denial, legacy NULL ownership despite matching created_by, safe response projection, origin/auth/method/input checks, publication permissions and revocation, reschedule ownership continuity, business cancellation visibility, direct SQL reassignment/replacement denial, disabled accounts/businesses and downgrade refusal. The public path is explicitly tested as absent. Existing Go lifecycle tests also cover unassigned booking behavior.

Frontend source was unchanged, so lint/build results remain those recorded at Phase 6.7; the browser regression above was rerun after this backend change. Race-detector, Apple/Google configuration and production limitations remain in DEFERRED_WORK.md.

## Next

Phase 7.2: business publishing review UI and explicitly opt-in public business profiles. Then add customer discovery, availability/booking creation and owned history/manage flows in separate tested slices. Payments and messaging remain dependent phases. Phase 7 is still in progress.

## Phase 7.2 preparation update

PUBLICATION_REVIEW.md supersedes the Phase 7.1 publication-write contract above:
published=true now requires a current review_token, and reviewed snapshots are stored
separately. Draft edits do not become public automatically. The private preview is at
/business; public routes and publish/unpublish controls await approval. Schema is 18.

## Phase 7.2 completed - 2026-10-16

The public page/API and explicit owner/admin publish controls are now enabled.
PUBLICATION_REVIEW.md supersedes the earlier private-only and pending-approval notes.
Schema 19 keeps prior prepared snapshots private until newly confirmed; customer
ownership contracts are unchanged. Next is public discovery/service selection.

## Phase 7.5 completed (2026-10-16)

Customers can now confirm a booking from a published business after signing in.
Ownership, assignment and current availability are enforced atomically; idempotent
retries recover the saved booking without duplication. The responsive confirmation
UI includes a booking reference. Schema stays 19. See
[CUSTOMER_BOOKING_CREATION.md](CUSTOMER_BOOKING_CREATION.md) for the contract and tests.
Next: Phase 7.6 customer booking history and detail UI; full Phase 7 remains open.
