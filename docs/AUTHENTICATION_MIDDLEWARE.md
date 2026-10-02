# Authentication middleware (Phase 3.11)

`server.NewAuthenticationMiddleware(sessionService, webOrigin, logger)` constructs
route-scoped middleware. Wrap a protected handler using `middleware.Require(handler)`.
The origin configuration selects the same cookie policy as login/logout: HTTP only
on loopback uses booking_session; HTTPS uses __Host-booking_session. It does not
infer policy from untrusted Host or forwarded headers.

The middleware requires exactly one nonempty configured cookie. It delegates
format validation, token hashing, persisted expiry/revocation checks, local user
lookup, active account checks, and transactional last-seen updates to SessionService.
Session validation must commit before the protected handler runs. Absolute expiry
is never extended. Expired/revoked/missing/malformed sessions and disabled accounts
receive the same 401 UNAUTHENTICATED response; storage/commit failures produce
503 AUTHENTICATION_UNAVAILABLE. Errors use structured JSON with a generated request
ID. Authentication logs contain controlled outcome codes, not tokens or user data.
Responses carry Cache-Control: no-store. Middleware does not issue or clear cookies.

Protected handlers use `server.IdentityFromContext(r.Context())` to get the local
UserID and a boolean presence flag. The private context key prevents other packages
from setting identity directly. The returned struct is a value copy containing no
session secret, password data, provider token, role, or tenant claim. An identity
already in context is replaced after validation; headers, query parameters, request
bodies, and Authorization bearer headers are not sources of trusted identity.
Business services should receive the validated ID explicitly, without importing HTTP.

Validation inherits request cancellation and has a 10-second timeout. The temporary
validation context is cancelled after validation; downstream receives the original
request context plus trusted identity, so it is not prematurely cancelled.

Register public health, registration, login and logout handlers without this wrapper.
Logout must remain usable with expired/disabled sessions. Do not wrap the entire mux.
No production protected endpoint is added in Phase 3.11: Phase 3.12 will wire /auth/me.
Unit and PostgreSQL integration tests use a protected handler to verify the boundary.

This middleware provides authentication only. A CORS layer must handle preflight
outside it, and cookie-authenticated mutations require their own CSRF protection
(such as the exact-origin checks already used by login/logout). Resource, tenant,
and role authorization must run after authentication. No global OPTIONS bypass is
provided. Already-running requests are not cancelled by later logout or disablement.

No frontend changes, new dependencies, migrations, or manual configuration are needed.
Verification: gofmt, database-enabled go test ./... -count=1, go build ./..., go vet ./....
