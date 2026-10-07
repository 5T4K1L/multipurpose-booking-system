# Google and Apple application sign-in

Implemented 2026-10-07. Both providers are disabled by default. Password sign-in,
registration, logout and current-account access continue to work without credentials.
Provider configuration is read from the Go process environment at startup. The tracked placeholder template is docs/oauth.env.example. Copying
that template into a file alone does not load that file into the Go process.

## Google first: manual provider step

The selected next step is Google. Create an OAuth client of type **Web application**
in your Google Cloud project's Google Auth Platform configuration. Configure the
consent/audience settings and include your Google account as a test user if the app
is in testing. Register this exact authorized redirect URI for local development:

`http://localhost:8080/api/v1/auth/google/callback`

Save the downloaded client JSON outside the Git repository. Tell the coding agent
its local file path; do not paste the client secret or upload it to chat. Once the
file exists, local process configuration and API restart can be completed without
printing secrets. No Google Cloud project, consent screen or client was created by
the implementation task. Live-provider verification is not yet claimed.

Reference: [Google's server-side OpenID Connect flow](https://developers.google.com/identity/openid-connect/openid-connect).

## Runtime settings

| Setting | Meaning |
| --- | --- |
| GOOGLE_OAUTH_ENABLED | Exact true enables Google; false/empty disables it |
| APPLE_OAUTH_ENABLED | Exact true enables Apple independently |
| OAUTH_TRANSACTION_KEY | 32 random bytes, unpadded base64url; kept server-side |
| OAUTH_ALLOW_REGISTRATION | true permits creation of a new local OAuth user; default false allows existing linked identities only |
| WEB_ORIGIN | Browser origin, default http://localhost:3000 |
| API_PUBLIC_ORIGIN | API origin, default http://localhost:8080 |
| NEXT_PUBLIC_API_BASE_URL | Next build-time API address; must point to the same API origin |

Google additionally requires GOOGLE_OAUTH_CLIENT_ID, GOOGLE_OAUTH_CLIENT_SECRET,
and GOOGLE_OAUTH_REDIRECT_URL. For the first new-account test, enable
OAUTH_ALLOW_REGISTRATION=true deliberately. Use a verified Google email that is not
already a password account: existing emails require explicit linking, and this
sign-in route never links by email. No HTTP linking workflow is added here.

Callbacks must exactly equal API_PUBLIC_ORIGIN plus the provider callback path.
API and web origins must share the HTTP/HTTPS policy. Deploy web and API same-site
(for example sibling HTTPS subdomains); the application session remains SameSite=Lax
and is not intended for unrelated cross-site frontend/API deployments. Never infer
public origin from untrusted forwarded headers. Restart the API after changing its
environment; rebuild Next when changing its public API address.

## Apple setup after Google

Apple requires a public HTTPS domain callback; IP addresses and localhost are not
accepted. Set HTTPS web/API origins and register:

`https://YOUR_API_DOMAIN/api/v1/auth/apple/callback`

Configure the Apple Services ID and Sign in with Apple web/domain settings, then
supply APPLE_OAUTH_CLIENT_ID, APPLE_TEAM_ID, APPLE_KEY_ID, APPLE_PRIVATE_KEY_PEM and
APPLE_OAUTH_REDIRECT_URL through local/deployment secret storage. Do not commit keys.
Apple returns form_post when requesting name/email. Only verified token claims are
used for identity; the untrusted posted user/name blob is not used for account linking.

References: [Apple authorization request](https://developer.apple.com/documentation/signinwithapplerestapi/request-an-authorization-to-the-sign-in-with-apple-server),
[Apple signing keys](https://appleid.apple.com/auth/keys). ID tokens use RS256; the
application-generated Apple client secret uses ES256. These are distinct credentials.

## HTTP and security contract

- GET /api/v1/auth/providers exposes only enabled provider names, with no-store and
  exact configured-origin CORS. No secrets or client configuration are returned.
- POST /api/v1/auth/google or /apple requires exact WEB_ORIGIN. The login page uses
  native form buttons so the browser supplies Origin; GET cannot begin a flow.
- Google callback is GET; Apple callback is bounded form POST. State, encrypted
  browser cookie, provider, nonce, signature, issuer/audience/azp/time claims, PKCE,
  and a one-use server transaction must all validate before local-user resolution.
- Google transaction cookie is SameSite=Lax. Apple's is Secure SameSite=None for
  the cross-site POST. HTTPS cookies use __Host- names; local Google HTTP uses an
  unprefixed cookie. Cookies are HttpOnly, host-only, Path=/ and cleared on callback.
- Success creates a fresh normal session after commit and redirects only to the
  configured /account. Failure redirects only to /oauth/error. No client return URL
  is accepted, and no token, authorization code, email or raw provider error is logged.
- Existing provider+subject identity wins over changed email. New-user creation is
  controlled by server policy and requires verified email. Matching email never
  attaches a provider or password credential. No platform privileges are granted.
- Start/callback share a per-peer 30-attempt/15-minute limiter. Transactions expire
  after 10 minutes and are capped at 10,000. A callback consumes its transaction
  before exchange, so retries require a new start and cannot mint multiple sessions.

The pending transaction store is process-local. Restart safely invalidates pending
flows; multiple replicas need sticky routing to the originating instance or a
shared atomic transaction store. This is not multi-instance production verification.
Session persistence remains PostgreSQL. No schema or new package is required.

## Verification and remaining work

Passed: full database-enabled Go tests, build/vet/gofmt, frontend lint/production
build, signed mock Google and Apple exchange/JWKS validation, PostgreSQL session
handoff/revocation, replay prevention, bad cookies/nonce/state/duplicate parameters,
Apple form handling, bounded inputs/transaction map, disabled providers, CSRF,
creation policy and no automatic email linking. Live password flow regression and
provider discovery with both disabled also passed.

Real Google/Apple consent and callback round trips need credentials and manual
provider setup. Browser automation remains unavailable from the prior closure
review, so interactive visual QA is not claimed. Phase 3 stays open until the
remaining live verification and closure review are resolved. Phase 4 is not started.

## Live verification update ? 2026-10-07

Google sign-in was confirmed working by the user and the API logged an accepted
Google callback. An initial future-issued-token rejection was traced to Windows
clock skew. Enabling/configuring Windows Time and synchronizing the clock resolved
it without relaxing token validation. If this recurs, inspect the server's safe
outcome code and Windows synchronization status before changing authentication rules.
Apple remains disabled and has not been live-verified. Earlier statements above
about pending Google verification describe the setup stage, now completed locally.
