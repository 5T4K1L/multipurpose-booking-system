# Logout (Phase 3.10)

POST `/api/v1/auth/logout` ends the current application session. The browser sends
its existing session cookie using `credentials: "include"` and no request body.
Visit `/logout` and explicitly press Sign out; navigating or prefetching the page
never ends a session.

The exact configured `WEB_ORIGIN` is mandatory, including for non-browser clients.
Credentialed CORS permits only that origin. OPTIONS is a non-mutating preflight.
GET returns 405. Requests with bodies or duplicate session cookies return 400.
The transport never accepts user IDs, session IDs, or tokens in a JSON body.

On success the response is 204 with Cache-Control: no-store and an expired cookie:
`booking_session` for loopback HTTP; `__Host-booking_session` for HTTPS. Name,
Path=/, host-only scope, HttpOnly, SameSite=Lax, and Secure policy match login.
The cookie has an empty value, Max-Age=0 on the wire, and a past Expires date.
Login and logout share origin validation and cookie naming. See LOGIN.md for
same-site deployment requirements and the current development configuration.

The existing session service hashes the token and commits server-side revocation
before the handler clears the cookie. Missing, malformed, unknown, or previously
revoked tokens are harmless successful logout attempts. Expired sessions and
sessions belonging to disabled users can also be revoked. Other sessions remain
active. Requests already running are not cancelled.

Storage or commit failures return 503 LOGOUT_UNAVAILABLE without clearing the
cookie; the UI asks the user to retry. A lost response can likewise be retried.
The UI claims success only on 204. Logs contain request IDs and controlled outcome
codes, never cookie values or database errors.

The per-process limiter permits 60 attempts per transport peer per 15 minutes,
ignoring untrusted forwarding headers; 429 carries Retry-After. This inherits the
in-memory, shared-NAT, and multi-instance limitations documented for registration.
Database operations have a 10-second context timeout.

No all-session HTTP endpoint is exposed. The existing trusted RevokeAll service
is reserved for future authorized security workflows. Authentication middleware,
/me, RBAC, OAuth callback HTTP wiring, and session cleanup are outside this phase.
No migration or dependency is needed.
