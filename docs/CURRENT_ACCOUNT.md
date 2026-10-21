# Current account (Phase 3.12)

GET `/api/v1/auth/me` returns the currently authenticated local application user:

```json
{ "user": { "id": 42, "email": "member@example.com", "username": "Member" } }
```

The three fields are an explicit allowlist. Status, timestamps, password hashes,
session tokens, OAuth tokens, and private authentication records are not returned.
Email is not a claim of mailbox verification. This endpoint grants no tenant or
resource authorization and makes no role claims. Phase 3.13 applies a CUSTOMER
baseline gate after authentication; every active local account has this baseline.

The Phase 3.11 middleware validates the application session cookie, checks current
account state, and commits last-seen before calling the handler. The safe profile
is attached through a private context key from that same validated user, avoiding
a second lookup. Client headers, query parameters, and IDs do not select the user.
The public IdentityFromContext accessor still returns only UserID.

Missing/invalid/expired/revoked sessions and disabled accounts return generic 401.
Database or commit errors return 503. Authorization denial returns 403;
authorization storage failures return 503. No-store applies to successful and failed
responses; no new session or cookie is issued. Absolute expiration remains unchanged.

Credentialed CORS allows only WEB_ORIGIN. OPTIONS preflight runs before authentication
and permits GET; other methods receive 405. Foreign or null origins receive 403.
GET without Origin is supported for same-origin requests and non-browser clients.
The read-only route does not require a CSRF token. The login/logout exact-origin
mutation rules remain unchanged.

Visit `/account` to load the current account with credentials included and caching
disabled. Loading, signed-out, and failure states are distinct; refresh retries a
failed request. The browser never reads the HttpOnly cookie or stores bearer tokens.
The page is a current snapshot, not a polling session monitor.

No new configuration, migration, or dependency is required. In development, open
http://localhost:3000/login, sign in, then select Your account. After logout the
account request returns 401. The API remains at http://localhost:8080.

Verified with the full database-enabled Go suite, build/vet/gofmt, frontend lint and
production build, and live login/account/logout smoke testing. Browser automation
could not initialize; interactive/visual QA is unverified. See RBAC.md for the Phase 3.13 authorization layer.
