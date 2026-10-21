# Phase 7.3 - Public discovery and service selection

Implemented 2026-10-16. Visitors can search published businesses at /businesses,
open a business page and select a published service without signing in. Selection
is local UI state; it does not reserve a slot, create a booking or persist a draft.

## Try it

1. A business owner/admin opens /business, creates the desired services and saves
   profile/customization changes.
2. Refresh the publishing preview. It now explicitly includes active service names,
   IDs and durations and explains inclusion in the public searchable directory.
3. Review, confirm and publish (or republish). Previously published profiles remain
   accessible through their existing link, but need this new review before listing.
4. Open http://localhost:3000/businesses in a signed-out browser. Search by published
   business name or introduction, open a result and choose a service.
5. Edit a service: visitors still see the reviewed name/duration until republished.
   Archived services disappear on the next public profile request. Unpublishing or
   disabling a business removes it from subsequent directory/profile requests.

There is a Find a business link on the home page and public business pages. Empty
results, request errors, pagination and refresh are supported. Earlier previews
containing no services show an empty service list until republished.

## API and publication boundary

GET /api/v1/public/businesses accepts q (optional), limit (1..100, default 100), and
cursor. Search trims surrounding whitespace and uses a case-insensitive literal
substring of the published business name or tagline; percent/underscore characters
are not wildcards. q is limited to 100 Unicode characters and cannot contain control
characters. Unknown/duplicate query keys, malformed or mismatched cursors and invalid
limits return 400. The UI requests 20 results per page.

Results contain only id, name, timezone and tagline. X-Next-Cursor is exposed through
CORS for pagination. Ordering is ascending UUID; cursor scope includes a hash of the
search. Cursors are selectors, not authorization. Every query independently enforces
active business, confirmed snapshot and published state. Queries project only summary
fields rather than fetching full service/contact snapshots. Responses are no-store.

Directory eligibility also requires the stored snapshot's services field, introduced
in this milestone. This makes republishing the explicit opt-in for existing profiles.
Old private flags or prior profile publication alone do not silently add listings.

Publication snapshots now include services: [{id,name,duration_minutes}]. The owner's
preview and confirmation cover all active services, bounded to 1,000 services per
review. A larger active catalog returns PUBLICATION_TOO_LARGE; unpublishing remains
available even then. The UI offers that action if the review cannot load. No staff,
resources, memberships, customer data or booking details are exposed.

GET /api/v1/public/businesses/{id} returns reviewed service fields and filters out
currently archived or foreign-tenant service IDs. Adding or editing services requires
republishing; archived services are removed immediately on the next request. Review
hashes include service data, so a catalog change invalidates a stale preview. Services
without configured availability may still be listed; their display is not a promise
of bookable times. Public availability must validate current rules in Phase 7.4.

No schema migration or new credentials are required; schema stays 19. No real business
was published or republished by this work. Google remains enabled in the local launcher.

## Verification

- Database-enabled full go test ./... -count=1, go build ./..., go vet ./....
- Targeted customer-foundation/discovery integration test and build/vet rerun after
  reducing the directory SQL projection to summary fields.
- sqlc generation, gofmt and git diff --check.
- Frontend lint/build, including the new /businesses route.
- Tests cover multi-page search, literal percent search, cursor/search mismatch,
  invalid query input, legacy snapshot exclusion, unpublished/disabled exclusion,
  service snapshot edits and stale confirmation, archive filtering, foreign-tenant
  service denial, and unpublishing above the publishing review limit.
- Isolated Chrome flow: create temporary business/service fixtures, publish, search
  anonymously, select service, reject stale review, retain private drafts, republish,
  unpublish and verify disappearance from search; empty search also checked. No page
  errors. Directory, profile and owner workspace overflow checks at 320/390/768;
  desktop 1280 and mobile 390 captures, with the mobile service picker inspected.
- The first browser run identified an ambiguous select label; an explicit accessible
  label fixed it and the complete flow passed. Fixtures were cleaned up.

## Limits and next

Next: Phase 7.4 customer-facing availability for a selected published service, with
location/eligibility resolution and current scheduling rules. Customer booking creation,
owned history/rescheduling/cancellation remain subsequent slices. No payment, map,
geographic ranking, price filtering or full-text search is included here.

Search is a bounded simple scan of published snapshots and needs scale evaluation
before production. Pagination is not a frozen snapshot; newly published/removed data
can change between requests. An already open page is refreshed manually, not by live
push. Full Phase 7 and its end-to-end customer UI remain in progress; UI polish and
broader release checks remain deferred as previously recorded.

## Phase 7.4 update (2026-10-16)

Public service/location/date availability and responsive customer UI are complete.
Publication previews now include linked location IDs, names and timezones; existing
snapshots require owner/admin review and republishing. Current scheduling and
occupancy filter advisory times. Selection creates no reservation or booking.
See [PUBLIC_AVAILABILITY.md](PUBLIC_AVAILABILITY.md) for limits and passed checks.
Next is Phase 7.5 authenticated customer-owned booking creation.
