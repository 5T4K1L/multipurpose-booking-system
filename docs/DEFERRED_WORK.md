# Deferred work and resume checklist

Updated 2026-10-10. The user explicitly deferred Apple setup/live testing and
instructed development to continue to the next phase. Apple remains disabled.
Deferral is not a successful live verification or a claim of production readiness.

## Apple sign-in setup and live verification - deferred by user

Implemented code and signed mock-provider/database tests exist. Real Apple login
has not been verified. Resume these steps when Apple Developer access and a public
HTTPS deployment are available:

1. Configure a primary App ID with Sign in with Apple in Apple Developer.
2. Create an associated Services ID for the web application.
3. Choose same-site public HTTPS frontend and API domains. Apple cannot use localhost
   or an IP callback. Register the web domain and the exact return URL:
   https://YOUR_API_DOMAIN/api/v1/auth/apple/callback.
4. Create a Sign in with Apple private key. Keep the .p8 file outside Git and chat.
   Record the Services ID, Team ID and Key ID in local/deployment secret configuration.
5. Set APPLE_OAUTH_CLIENT_ID, APPLE_TEAM_ID, APPLE_KEY_ID, APPLE_PRIVATE_KEY_PEM,
   APPLE_OAUTH_REDIRECT_URL and a secure OAUTH_TRANSACTION_KEY. Set APPLE_OAUTH_ENABLED=true.
6. Set WEB_ORIGIN and API_PUBLIC_ORIGIN to the chosen HTTPS origins; rebuild the web
   app with NEXT_PUBLIC_API_BASE_URL for that API. If Google is enabled there too,
   register/update its exact HTTPS callback. Preserve the working local Google setup.
7. Restart the API and check /api/v1/auth/providers. Verify the Apple button appears.
8. Test first login with permitted new-account creation, returning login, Hide My Email,
   cancellation, expired/replayed state, logout and /auth/me. Confirm no email auto-link.
9. Verify Secure/HttpOnly cookie handoff through Apple's cross-site form POST and
   generic errors. Never log raw codes, tokens, private keys or identity claims.
10. Record the live result in CURRENT_STATE.md and PHASE_3_CLOSURE.md, then revisit
    the remaining release checks below. See OAUTH_SETUP.md for full configuration.

Official setup: https://developer.apple.com/help/account/capabilities/configure-sign-in-with-apple-for-the-web

## Other outstanding verification - recorded, not silently waived

- Automated browser/visual QA: automation previously failed to initialize. Google
  sign-in was manually confirmed by the user and accepted in the API log. Retry
  browser automation or perform documented manual registration/login/logout/account
  checks on mobile and desktop sizes; Apple waits for the setup above.
- Go race detector: previous run could not start because CGO/C compiler support was
  unavailable. Install/configure an appropriate C toolchain or use a suitable CI runner,
  then run database-enabled go test -race ./... and record the result.
- Production deployment: OAuth pending transactions and rate limits are process-local.
  Review shared storage/affinity and rate-limit policy before multi-instance deployment.
  Configure secrets/origins and verify clock synchronization. No deployment is implied.

## Development boundary

Google/password development is verified. The user authorized moving into Phase 4
with Apple deferred; this is a development progression decision, not unconditional
Phase 3 or production security sign-off. Keep these items visible during later reviews.


## Booking UI verification update (2026-10-13)

Phase 5 /bookings passed an isolated headless Chrome flow covering local login,
demo setup, creation, occupied slots, rescheduling, cancellation and history.
Desktop 1280x900 and mobile 390x844 screenshots were inspected; no horizontal mobile
overflow or browser page errors occurred. Interactive computer-use tools still failed
to initialize. This booking-specific coverage does not waive broader authentication,
cross-browser/accessibility, Apple, race-detector or production checks above.
