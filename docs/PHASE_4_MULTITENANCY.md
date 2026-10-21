# Phase 4 - Multi-tenancy foundation

Completed 2026-10-10. User authorized development progression with live Apple
setup/testing deferred; see [DEFERRED_WORK.md](DEFERRED_WORK.md).
Phase 4.2 and the development closure review completed on 2026-10-10. The scope is
the current business/membership resources; future booking resources need their own isolation tests.

## Implemented behavior

- Migration 000008 creates UUID businesses and unique (business, user) memberships.
- Creating a business atomically grants its authenticated creator BUSINESS_OWNER.
- Business name is trimmed, 1-100 characters, without controls; timezone must load
  as a named timezone (UTC allowed, machine-local `Local` disallowed).
- Active users see only their own active businesses. Every scoped operation resolves
  current database membership; caller-supplied IDs and headers never grant authority.
- Any member can read the business. OWNER and ADMIN can list memberships.
- Only OWNER can add/change/remove ADMIN, MANAGER or STAFF for existing active users.
  OWNER assignment, transfer, demotion and removal are unavailable through these APIs.
- PLATFORM_ADMIN and PLATFORM_OPERATOR confer no automatic tenant access.
- Business/membership locks protect permission checks and membership changes within
  the same transaction. Failed commits return no business result.
- Existing authentication, Google OAuth and global RBAC behavior is preserved.

## API contract

All paths start with `/api/v1/businesses`. Session cookie authentication is required.
The exact configured WEB_ORIGIN header is required for mutations; JSON requests use
Content-Type application/json, reject unknown fields/trailing values, and cap at 4 KiB.
Mutation rate limit: 60 requests per transport peer per 15 minutes per API process.
All responses use Cache-Control: no-store. Preflight does not require a session.

| Method | Path suffix                        | Behavior                                                                    |
| ------ | ---------------------------------- | --------------------------------------------------------------------------- |
| POST   | (none)                             | `{ "name": "Example", "timezone": "Asia/Shanghai" }`; returns business, 201 |
| GET    | (none)                             | Returns caller's businesses, 200                                            |
| GET    | /{businessID}                      | Returns business and caller's scoped role, 200                              |
| GET    | /{businessID}/memberships          | Returns user_id/role entries, 200                                           |
| PUT    | /{businessID}/memberships/{userID} | `{ "role": "STAFF" }`; upserts membership, 204                              |
| DELETE | /{businessID}/memberships/{userID} | Removes a non-owner membership, 204; absent member is idempotent            |

Business response fields: id, name, timezone, role. Lists are arrays (empty is `[]`).
Invalid input returns 400, missing/invalid session 401, denied access 403, and
storage failure 503. Missing and inaccessible businesses share the same 403 response.
Existing-user IDs are required; invitations and email lookup are not implemented.
Both list endpoints accept `limit=1..100` (default 100) and `cursor=<value>`.
Array responses remain compatible. `X-Next-Cursor` appears only when another page
exists and is exposed through CORS. Pass it unchanged as the next cursor. An absent
header means the end. Invalid, duplicated or cross-list parameters return 400.
Businesses sort by (created_at, UUID); memberships by user_id. Cursors bind the actor,
list kind and business where applicable. They are unsigned selectors, not credentials;
changing a cursor cannot bypass the current membership checks. Pages are fresh reads,
not a frozen export snapshot: concurrent additions/removals can change later pages.

## Verification

- gofmt on changed handwritten Go files; sqlc generate for generated queries/models.
- PostgreSQL migration applied successfully through the existing startup runner.
- Database-enabled `go test ./... -count=1`, `go build ./...`, `go vet ./...`.
- Integration coverage: two isolated businesses, spoofed identity/role headers,
  platform-admin non-bypass, scoped staff/admin permissions, immediate revocation,
  owner protection, disabled business/account, strict input and origin checks.
- Actual PostgreSQL rollback coverage plus unit checks that failed membership writes
  or transaction commits expose no created business.

No frontend changes or browser verification are part of this backend milestone.
Race-detector and earlier release QA gaps remain in DEFERRED_WORK.md.

## Phase 4.2 authorization and concurrency contract

WithinTenant locks the active business first, then resolves active membership in
another statement. It requires an explicit exact-role allowlist; empty, unknown,
and platform-role policies fail closed. TenantAccess has private fields with read-only
accessors for actor, business and role. It is valid as an authorization snapshot only
inside the transaction callback, never as a reusable credential or cached session claim.

All future tenant resource repositories must run through this transaction boundary,
use its scoped business ID in every resource query, and avoid external side effects
or publishing results until the transaction commits. Extend the transaction store
with narrow resource methods rather than opening an unrelated transaction. Resource
ownership remains a separate check. All membership writers must lock business first.

PostgreSQL contention tests deliberately hold one transaction open while a second
request waits. Committed revocation/demotion denies the waiting request. Another
business continues independently, and cancelled waiters leave no retained locks.
The per-business lock deliberately serializes scoped operations for this foundation;
load testing and finer locking are future scaling work, not verified here.

Additional pagination tests cover 105 businesses and 105 memberships, equal creation
timestamps, default/custom page sizes, malformed and cross-scope cursors, deleted
boundary rows, and loss of permission between pages. Unit tests cover invalid policies.
The full database-enabled test suite, build and vet passed after these changes.

## Closure and known limits

Phase 4 development scope is complete for businesses/memberships. No new migration
or manual configuration was needed for 4.2. Existing APIs remain compatible; there
are no new frontend screens. This is not production or external security certification.
Apple live setup, automated browser QA and race-detector execution remain explicitly
tracked in DEFERRED_WORK.md. Existing transaction/rate-limit scaling limitations remain.
Owner transfer and invitations are not implemented. Future services, resources and
bookings must prove their own tenant constraints and ownership rules before release.

## Next task

Phase 5.1 - Core Booking Engine foundation: define tenant-owned services, staff,
resources and locations, then implement the first scoped schema/service milestone.
Booking creation, availability and double-booking prevention follow within Phase 5.
Business dashboards remain Phase 6. No Phase 5 implementation is included here.
