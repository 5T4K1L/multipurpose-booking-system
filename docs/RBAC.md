# RBAC / Authorization (Phase 3.13)

Authentication identifies a local user. Authorization independently checks a
server-defined policy against current database state. The order is CORS/method
handling, AuthenticationMiddleware.Require, RequireRoles, then the handler.
The /auth/me route demonstrates this with the CUSTOMER baseline and keeps its
existing safe profile response. Public registration, login, and logout stay public.

## Policies

| Role | Current behavior |
| --- | --- |
| CUSTOMER | Baseline for every active local account; no stored grant needed |
| PLATFORM_ADMIN | Requires an explicit user_platform_roles grant |
| PLATFORM_OPERATOR | Requires an explicit user_platform_roles grant |
| BUSINESS_OWNER, BUSINESS_ADMIN, MANAGER, STAFF | Denied until Phase 4 membership integration |

RequireAnyRole accepts an authenticated server-derived user ID and a server-owned
list of roles. Any matching supported role permits access. Empty/unknown policies,
unknown grants, missing/inactive accounts, and any policy containing business roles
deny access. There is no implicit role hierarchy: administrator does not imply
operator, resource ownership, or tenant membership. Roles from browser fields,
headers, or OAuth providers are never consulted.

The repository reads account status and grants in one SQL statement for each
decision. Grants are not cached in sessions. Removing a grant affects the next
request; already-running requests are not cancelled. Storage failures fail closed.
An authenticated denial is 403 FORBIDDEN, absent authentication is 401, and lookup
failure is 503 AUTHORIZATION_UNAVAILABLE. Logs omit database details and credentials.
The middleware copies its policy and bounds the lookup with a 10-second timeout.
Business services can call RequireAnyRole directly with trusted identity.

RequireOwner is a separate ownership check: positive authenticatedUserID must equal
ownerID loaded from the persisted resource. Never pass a client-supplied owner ID.
A platform grant does not bypass this check. No booking/resource endpoints exist
yet, so their ownership enforcement remains the responsibility of future services.

## Storage and operation

Migration 000007 adds user_platform_roles with a user foreign key, unique user/role
pair, and a check limiting grants to PLATFORM_ADMIN or PLATFORM_OPERATOR. All
existing accounts retain customer access; none are automatically elevated.
Normal API startup applies the migration. No manual configuration is required.
The down migration drops the grant table and its data; application rollback must
be coordinated before using it. It was not run against the development database.

There is no HTTP role-assignment endpoint or default admin account. Privileged
provisioning remains an explicit trusted administrative operation. Business roles
must be bound to trusted memberships when Phase 4 introduces tenants; this phase
does not create globally privileged owner/admin/manager/staff assignments.

## Verification and boundary

Verified: gofmt, sqlc generation, PostgreSQL-enabled full Go test suite, build, vet,
and live registration/login/current-account/logout regression checks with fixture
cleanup. Unit tests cover supported/unknown policies, role separation, disabled
accounts, ownership mismatch, forged client roles, safe errors, and downstream
context. Integration tests cover grants, immediate removal, role constraints, and
current account status. No frontend changes or browser visual claims.

Next: Phase 3.14 authentication security tests. Tenant membership and role-management
workflows remain deferred; this implementation grants no cross-tenant authority.
