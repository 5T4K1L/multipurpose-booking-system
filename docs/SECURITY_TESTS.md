# Phase 3.14 security verification

Verified 2026-10-07 against local PostgreSQL at migration 000007. This is regression
verification of implemented boundaries, not a penetration-test certification or
Phase 3 closure. No real user account was changed by the tests.

## Coverage map

Paths below are relative to apps/api/internal.

| Requirement                             | Evidence                                                                                                                                                                                               |
| --------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Password hashing                        | auth/password_test.go: random salts, correct/incorrect password, corrupt/bounded hash parameters                                                                                                       |
| Login failures and enumeration          | service/authentication_test.go; server/login_test.go, login_integration_test.go, security_test.go: generic response/status, no session on failure, disabled/missing/passwordless accounts              |
| Registration enumeration/linking        | service/registration_test.go; repository/registration_integration_test.go: neutral duplicate handling, concurrent duplicates, no password attachment to existing accounts                              |
| Session expiry/revocation/logout        | service/session_test.go; repository/sessions_integration_test.go; server/logout*_test.go: absolute expiry, disabled users, commit failure, single/all revocation, matching cookie expiration and retry |
| OAuth state/cookie                      | auth/oauth_state_test.go, oauth_transaction_test.go, oauth_callback_test.go, security_test.go: state/provider mismatch, age, AEAD tampering, fuzz decode                                               |
| Nonce and OIDC                          | auth/oidc*_test.go, security_test.go: signature/key, issuer/audience/subject/nonce, temporal claims and azp                                                                                            |
| PKCE                                    | auth/security_test.go and oauth_exchange_test.go: exact S256 digest and exact transaction verifier in exchange, fresh state/nonce/verifier                                                             |
| Identity ownership and explicit linking | service/authentication_test.go; repository/authentication_test.go, oauth_integration_test.go: no email auto-link, provider ownership conflict, rollback                                                |
| RBAC                                    | service/authorization_test.go; server/authorization*_test.go: explicit grants, removal, no hierarchy, spoofed roles/IDs, disabled accounts                                                             |
| Object-level authorization              | service/authorization_test.go: ownership equality, invalid IDs, mismatch denied; no resource endpoints exist yet                                                                                       |
| Tenant assumptions                      | business policies deny pending membership support; platform roles grant no business access; actual cross-tenant data isolation belongs to Phase 4                                                      |
| Authentication middleware               | server/authentication*\_test.go, me*_test.go: trusted identity, private context, invalid cookies, safe profile allowlist, denial after logout                                                          |
| Rate limiting                           | server/security_test.go plus login/registration/logout tests: concurrent count, peer normalization, map capacity/pruning, expiry and proxy spoofing                                                    |
| Safe errors and secret redaction        | login/logout/authentication/authorization handler tests assert tokens, credentials and private storage errors do not appear in output/logs                                                             |

## Finding fixed

The new signed-token single-audience/wrong-azp case initially failed: the verifier
ignored azp unless multiple audiences existed. It now validates every nonempty azp
against the configured client, preserving the existing multiple-audience requirement.
Matching or absent azp for a single audience remains accepted. This follows the
project's client-binding policy and the guidance in
[OIDC Core ID Token Validation](https://openid.net/specs/openid-connect-core-1_0.html#IDTokenValidation).

## Execution

- gofmt and git diff --check.
- DATABASE_URL set: go test ./... -count=1 (PostgreSQL integration tests enabled).
- go build ./... and go vet ./....
- go test ./internal/auth -run TestSecurity -fuzz FuzzOAuthTransactionDecode -fuzztime=10s -parallel=2:
  passed 59,257 executions, 19 additional interesting inputs. Fuzz input is bounded
  at 8192 bytes in the harness. Initial temp-directory access failed; rerun using
  the writable workspace temp directory passed. No fuzz crash corpus was generated.
- go test -race ./internal/server -run TestSecurity -count=1: unavailable because
  CGO is disabled and no C compiler is on PATH. Normal concurrent tests passed;
  this does not substitute for race-detector verification.

## Limits for the closure review

- Google/Apple HTTP callback routing and browser/provider end-to-end login remain
  unwired. Foundation tests do not prove a complete deployed OAuth login flow.
- Browser interaction/visual QA was previously unavailable and was not rerun here.
- No statistical timing-side-channel certification; tests check generic responses
  and existing dummy-password work, not indistinguishable timing distributions.
- No dependency vulnerability scan or external penetration test is claimed.
- Rate limiting is per process and transport peer, with shared-NAT and multi-instance
  limitations. Deployment/proxy policy remains documented in the endpoint docs.
- Object ownership and tenant assumptions are component policies; resource/tenant
  endpoints do not exist yet. Phase 4 requires actual membership and isolation tests.
- Do not mark Phase 3 closed solely because this regression suite passes. Phase 3.15
  must explicitly assess outstanding integration and verification requirements.
