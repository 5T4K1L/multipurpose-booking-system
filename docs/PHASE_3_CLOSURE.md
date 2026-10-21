# Phase 3.15 closure review ? 2026-10-07

**Decision: Phase 3 remains open.** The initial review below found missing OAuth
application integration. That integration was subsequently implemented and tested
with signed mock providers; live Google/Apple and browser verification remain open.
See the update at the end and OAUTH_SETUP.md. The initial checklist is historical.
No Phase 4 implementation is authorized by this review.

## Closure checklist

| Requirement                     | Result                                                                             |
| ------------------------------- | ---------------------------------------------------------------------------------- |
| Implementation complete         | Not met: Google/Apple route wiring, UI, and live-provider integration remain       |
| Unit/integration/security tests | Full PostgreSQL-enabled go test ./... -count=1 passed                              |
| Backend static/build checks     | gofmt on changed code, git diff --check, go vet ./..., go build ./... passed       |
| Frontend static/build checks    | ESLint and production build including TypeScript passed                            |
| Current password flow           | Live registration/login/account/logout and fixture cleanup passed                  |
| OAuth foundation security       | Existing component suite and Phase 3.14 tests pass; not live-provider validation   |
| Browser interaction             | Unverified: automation kernel initialization failed again                          |
| Race detector                   | Unverified: CGO/compiler prerequisite unavailable; normal concurrency tests passed |
| Docs/current state/roadmap      | Updated; known gaps retained explicitly                                            |
| Final completion                | Not met; repeat this review after integration and verification                     |

Frontend build initially failed fetching Google Fonts in the sandbox. Retrying with
network access succeeded. This is an existing build-time network requirement.
No new dependency, migration, or frontend code was introduced in this review.

## Review finding fixed

GenerateOAuthTransactionKey outputs raw base64url. loadTransactionKey incorrectly
used RawStdEncoding, rejecting generated keys containing '-' or '_'. It now uses
RawURLEncoding consistently with the generator, existing error contract, and docs.
TestLoadTransactionKeyURLAlphabet uses a deterministic key with URL-only alphabet
characters; it failed before the fix and passes afterward. Existing valid base64url
configuration remains valid. Legacy standard-base64 values containing '+' or '/'
should be re-encoded from the same bytes as unpadded base64url before future use;
this is an encoding correction, not a required key rotation.

## Implementation gaps and next task

main.go currently wires health, registration, password login, logout, and /auth/me.
It does not load OAuthRuntimeConfig or wire Google/Apple start/callback routes.
The callback foundation validates an external identity but is not connected by
an HTTP handler to local-user resolution, session creation, cookie issuance and
safe browser completion. The frontend has no Google/Apple sign-in controls.
ROADMAP.md explicitly lists route wiring, provider verification and OAuth UI as
remaining work; those requirements cannot be marked complete by component tests.

Next work is OAuth application integration: connect each enabled provider to the
shared identity/session boundary, retain explicit linking/no-email-auto-merge,
complete callback/transaction-cookie handling and safe redirects/errors, add
provider buttons and mocked HTTP integration tests, then verify actual providers.
Review Apple callback method and cookie behavior during integration. Existing
password login must continue working when OAuth is not configured. Do not treat
configuration-only work as enough to close these implementation gaps.

## Configuration inventory for the future integration

No secret values were inspected, printed, created, or committed in this review.
The current OAuth loader expects the following names:

- OAUTH_TRANSACTION_KEY: exactly 32 random bytes encoded as unpadded base64url.
- GOOGLE_OAUTH_CLIENT_ID, GOOGLE_OAUTH_CLIENT_SECRET, GOOGLE_OAUTH_REDIRECT_URL.
- APPLE_OAUTH_CLIENT_ID, APPLE_TEAM_ID, APPLE_KEY_ID, APPLE_PRIVATE_KEY_PEM,
  APPLE_OAUTH_REDIRECT_URL.

Current LoadOAuthRuntimeConfig requires both providers. Integration should explicitly
handle enabled/disabled providers before asking for setup. Once exact callback URLs
and deployment origins exist, the user will need to register provider applications
and configure those secrets locally or in deployment secret storage. Do not paste
private keys or client secrets into chat. No manual change is needed for the
working password-based application now.

Existing app configuration remains DATABASE_URL, MIGRATIONS_PATH, WEB_ORIGIN and
NEXT_PUBLIC_API_BASE_URL (see endpoint docs for defaults and origin/cookie policy).
Provider credentials alone will not activate the currently unwired OAuth routes.

## Other documented limits

See SECURITY_TESTS.md and RBAC.md. Per-process rate limits, unverified-email policy,
absence of tenant/resource endpoints, and absence of statistical timing or external
penetration testing remain explicit scope limits. Business-role membership and
actual tenant isolation belong to Phase 4, not this closure review. Admin controls
and automatic platform-role assignment are not implemented or implied.

## Subsequent integration update (2026-10-07)

The implementation gaps above are now addressed: independent provider flags,
start/callback routing, safe session issuance/redirects, provider discovery, and
frontend controls are present. Mock-provider HTTP plus PostgreSQL tests pass for
Google and Apple, including creation-disabled policy and email collision denial.
The loader in use is now LoadEnabledOAuthConfig; the legacy all-provider loader
remains available but is not used by main. No real credentials are enabled.

Live Google is the user's chosen next step. Live Apple and interactive browser QA
remain unverified, as does the previously documented race-detector check. This
update does not close Phase 3 or start Phase 4. OAUTH_SETUP.md is the current setup
contract; the earlier missing-route description above records the original finding.

## Live Google result ? 2026-10-07

The user confirmed successful Google sign-in and the server logged an accepted
callback. Local Google live verification is complete. Windows time synchronization
resolved the earlier future-issued-token failure; no validation bypass was added.
Apple live verification remains pending. This does not claim automated browser QA,
race detection, production readiness, or Phase 3 closure.

## User-authorized deferral - 2026-10-10

The user explicitly deferred Apple setup/live testing and requested the next phase.
Phase 4 development may proceed under that instruction. Apple remains disabled;
this is not unconditional Phase 3 closure or production sign-off. Resume steps and
other outstanding browser/race/deployment checks are in DEFERRED_WORK.md.
