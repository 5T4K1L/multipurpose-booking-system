# Phase 4.1 - Business and membership foundation

Completed 2026-10-10. User authorized development progression with live Apple
setup/testing deferred; see [DEFERRED_WORK.md](DEFERRED_WORK.md).
Phase 4 remains in progress; this milestone does not close the entire phase.

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

| Method | Path suffix | Behavior |
|---|---|---|
| POST | (none) | `{ "name": "Example", "timezone": "Asia/Shanghai" }`; returns business, 201 |
| GET | (none) | Returns caller's businesses, 200 |
| GET | /{businessID} | Returns business and caller's scoped role, 200 |
| GET | /{businessID}/memberships | Returns user_id/role entries, 200 |
| PUT | /{businessID}/memberships/{userID} | `{ "role": "STAFF" }`; upserts membership, 204 |
| DELETE | /{businessID}/memberships/{userID} | Removes a non-owner membership, 204; absent member is idempotent |

Business response fields: id, name, timezone, role. Lists are arrays (empty is `[]`).
Invalid input returns 400, missing/invalid session 401, denied access 403, and
storage failure 503. Missing and inaccessible businesses share the same 403 response.
Existing-user IDs are required; invitations and email lookup are not implemented.
Lists currently return at most 100 entries; pagination is required before larger use.

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

## Next task

Phase 4.2: add bounded cursor pagination and a reusable tenant access contract for
future resource repositories; expand concurrent membership-change/isolation tests
and perform the Phase 4 closure review. Bookings (Phase 5) and business dashboards
(Phase 6) remain outside this milestone. Owner-transfer/invitations need their own
explicit workflow before implementation.
