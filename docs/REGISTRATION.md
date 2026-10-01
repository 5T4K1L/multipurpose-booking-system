# Phase 3.8 Registration

## API

`POST /api/v1/auth/register` accepts `application/json` with exactly the fields:

```json
{"email":"person@example.com","username":"Person","password":"a long unique passphrase"}
```

- Body limit: 4096 bytes. Unknown fields and trailing JSON values are rejected.
- Email: trim surrounding whitespace; accept a plain ASCII mailbox up to 254 bytes,
  with local part up to 64 bytes. Preserve case and the existing exact-email uniqueness
  policy. Do not normalize dots, plus tags, or provider aliases, or merge accounts.
- Username: trim surrounding whitespace, 1–80 Unicode characters, no control characters.
- Password: 15–128 Unicode characters, at most 512 UTF-8 bytes. No trimming, truncation,
  or composition rules. Invalid UTF-8 is rejected by the service.
- Password hashing uses the existing salted Argon2id implementation. Hashing runs
  before insertion for both new and duplicate emails.
- User and credential creation share a PostgreSQL transaction. Database uniqueness
  handles concurrent duplicate requests. Existing credentials/identities are never changed.

New and duplicate requests both return `202` with:

```json
{"message":"Registration request received."}
```

No user ID, email-existence flag, session, cookie, or verification claim is returned.
Duplicate accounts include password-only and OAuth-only local accounts. This endpoint
cannot attach a password to an existing OAuth account.

Errors use `{ "error": { "code", "message", "request_id" } }`:

| Status | Meaning |
| --- | --- |
| 400 | Invalid JSON or invalid registration fields |
| 403 | Browser origin is not allowed |
| 405 | Unsupported method |
| 413 | Body exceeds the limit |
| 415 | Content type is not application/json |
| 429 | Request rate or concurrent hashing limit reached |
| 503 | Registration temporarily unavailable |

Responses use `Cache-Control: no-store` and a generated `X-Request-ID`.

## Abuse controls and logging

The handler allows five attempts per transport peer IP per 15 minutes and at most
two concurrent registration operations. The in-memory limiter holds at most 10,000
peer entries and denies new peers at capacity until expired entries can be reclaimed.
It never trusts X-Forwarded-For or other forwarding headers. These limits are per process
and reset on restart; multiple replicas require a later shared limiter. Deployments
behind a proxy currently share that proxy's IP limit, intentionally failing conservatively.

Logs include only event name, generated request ID, and a controlled outcome code.
New and duplicate accounts share the accepted outcome. Passwords, email addresses,
request bodies, token values, database errors, and identity information are not logged.
Response equivalence and hashing reduce enumeration signals but do not claim constant-time
behavior across database paths.

## Verification and session policy

The current local-development policy creates an active account without a verified-email
claim. No email is sent and mailbox ownership is not established. No registration-time
session is issued. Email verification delivery/tokens and production verification gating
require a separately authorized design. Never use an unverified email as proof of identity
or authority to link an account. Password login remains Phase 3.9.

## Frontend and configuration

The `/register` form provides labeled fields, password visibility, client validation,
pending/error states, and the same neutral accepted message for duplicates. The home
page links to it; the existing health display is preserved.

- Go `WEB_ORIGIN`: exact permitted browser origin; defaults to `http://localhost:3000`.
- Web `NEXT_PUBLIC_API_BASE_URL`: public API base URL without a trailing slash;
  defaults to `http://localhost:8080`. Set in `apps/web/.env.local` before building.
- No credentials are included in registration fetches. HTTPS and explicit deployment
  origins are required when this local development flow is later deployed.

## Scope

No schema migration or new dependency. Existing password/OAuth/session behavior is
preserved. No login/logout endpoint, auth middleware, /me, RBAC, automatic linking,
email delivery, or registration session cookie is implemented.

Unit and integration tests cover validation, safe errors/logs, HTTP bounds, rate limiting,
concurrent duplicates, password preservation, and atomic rollback. See CURRENT_STATE.md
for the actual verification results and remaining limitations.
