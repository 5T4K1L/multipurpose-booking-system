# Phase 3.9 Password Login

## API and session delivery

`POST /api/v1/auth/login` accepts `application/json`:

```json
{ "email": "person@example.com", "password": "your existing password" }
```

The request must include `Origin` exactly matching `WEB_ORIGIN`, including scheme and
port. This is required for browser and non-browser clients. Missing, null, and foreign
origins are rejected before authentication. JSON content type, bounded body size, and
strict decoding provide additional login-CSRF protection. Broader authenticated-request
CSRF protection remains part of the later HTTP/middleware work.

Input is limited to 4096 bytes. Unknown fields, trailing JSON, and invalid UTF-8 are
rejected. Email is trimmed and bounded to 254 bytes while preserving exact case.
Passwords are nonempty and bounded to 512 UTF-8 bytes without trimming or truncation.
Registration's minimum password length is not reapplied to legacy credentials.

Successful password verification and an active local account create a fresh persisted
session through the existing authentication/session services. The server returns `200`:

```json
{ "message": "Signed in." }
```

The raw token is delivered only in a cookie, never response JSON, logs, browser storage,
or a URL. Only its SHA-256 hash is stored in PostgreSQL. Cookie settings:

- HttpOnly, Path=/, SameSite=Lax, no Domain attribute (host-only).
- Expiration matches the absolute server-side session expiry; main uses 24 hours.
- HTTPS `WEB_ORIGIN`: Secure cookie named `__Host-booking_session`.
- HTTP loopback development only: cookie named `booking_session` without Secure.
- Non-loopback HTTP origins fail startup. No forwarded headers decide cookie security.

Each login generates a new random token; any supplied cookie is ignored. Other existing
sessions remain valid, supporting independent sessions until expiration or revocation.
No logout/revocation endpoint or session-validation middleware is added in this phase.

## Failures and abuse controls

| Status | Meaning                                                                      |
| ------ | ---------------------------------------------------------------------------- |
| 400    | Invalid JSON or missing/oversized input                                      |
| 401    | Unknown user, wrong password, passwordless account, disabled/deleted account |
| 403    | Origin rejected                                                              |
| 405    | Unsupported method                                                           |
| 413    | Body too large                                                               |
| 415    | Unsupported content type                                                     |
| 429    | Rate/concurrency limit reached                                               |
| 503    | Authentication/session infrastructure unavailable                            |

All authentication rejections use the same `INVALID_CREDENTIALS` code and message.
Failed requests never set or clear session cookies, and no token is returned when
session creation/commit fails. Responses are no-store and have a generated request ID.
Security logs contain only event, request ID, and controlled outcome code.

Unknown and passwordless accounts perform dummy Argon2id verification using the current
hashing parameters to reduce obvious timing differences. This does not claim constant-time
database behavior. Corrupt password hashes reject zero/excessive Argon2 parameters and
invalid component sizes before allocating memory or invoking Argon2id.

Login permits ten attempts per transport peer per 15 minutes, two concurrent password
operations, and at most 10,000 in-memory peer entries. Forwarded headers are not trusted.
Registration retains its separate five-attempt limit and two concurrent operations.
Limits are per process and reset on restart; proxies share the proxy peer limit.
Shared/distributed limits and trusted-proxy handling are future deployment work.

## Frontend and configuration

`/login` provides labeled fields, password visibility, loading/failure states, a generic
credentials error, and sign-in confirmation. Requests use `credentials: include`; the
browser stores the HttpOnly cookie. The homepage and registration form link to login.
Confirmation does not redirect to an unimplemented dashboard, and is not a replacement
for server-side session validation on future protected pages.

- `WEB_ORIGIN` defaults to `http://localhost:3000`. Specify one exact origin without
  a trailing slash or path. Use HTTPS for non-loopback deployment.
- `NEXT_PUBLIC_API_BASE_URL` defaults to `http://localhost:8080` and is set at frontend
  build time. Frontend/API must be same-site for SameSite=Lax cookies; use the same
  hostname locally (`localhost` for both, not a mix of localhost and 127.0.0.1).
- Production must terminate HTTPS for both frontend and API. No wildcard CORS or
  automatic proxy-header trust is introduced.

## Scope and verification

No schema migration or new dependency. Email ownership remains unverified under the
Phase 3.8 local-development policy. OAuth provider callback behavior is unchanged;
its identity resolution already shares the underlying session service.

Tests cover cookie security, origin rejection, bounded JSON, rate/concurrency limits,
uniform credential failures, corrupt hashes, legacy passwords, and real database login
through HTTP, persisted hashed tokens, fresh sessions, disabled/passwordless users,
and absence of new sessions after failed login.

Stop before Phase 3.10 logout, Phase 3.11 middleware, /me, RBAC, and OAuth callback HTTP wiring.
