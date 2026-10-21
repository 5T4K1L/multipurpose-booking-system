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

## Local Google launcher credential - 2026-10-13

The credential JSON referenced by work/start-google-api.ps1 is no longer present at
its previous D: path. Restore it or update that local launcher's path, then restart
the Google-enabled API and verify provider discovery/sign-in. Do not commit secrets.
Phase 6.1 was verified using password sign-in through a separate temporary launcher;
Google is disabled in that running process. Project OAuth code was not changed.

## Phase 7 publishing follow-up - 2026-10-14

The proposed public business profile route was rejected by automatic approval review
because it could expose contact details without a concrete publishing review flow.
Phase 7.1 safely implements only private publication state and customer ownership.
Next, build an owner/admin UI showing exactly what becomes public, with explicit
publish/unpublish actions; require fresh confirmation before public exposure. Only
then enable a safe public projection gated on active/published state. Customer UI,
creation and owned history/manage flows remain subsequent Phase 7 work.

## Publishing approval status - Phase 7.2, 2026-10-14

Private review UI and snapshot checks are ready; PUBLICATION_REVIEW.md records them.
Automatic approval review again rejected the proposed public page because anonymous
contact exposure needs explicit approval for its fields and destination. Public page,
public API and publish/unpublish controls remain unfinished pending that approval.
The proposed exposure is the reviewed business name/timezone/introduction/description,
contact email/phone/website and business ID on this app's /businesses/{id} page and
/api/v1/public/businesses/{id} API, only after an owner/admin explicitly publishes.
Later edits require republishing; no real business has been made public by this work.

## Google local configuration restored - 2026-10-14

The user supplied a replacement web-client JSON outside the repository. Updated the
local work/start-google-api.ps1 launcher to read it without logging credentials and
restarted the API. Provider discovery reports google, and isolated Chrome verified
Continue with Google is visible on /login. This resolves the missing-file/disabled
provider issue above. The user must complete a fresh Google sign-in to verify the
replacement client's consent/callback. No credentials were copied into the repository.

## Publishing approval resolved - 2026-10-16

The user instructed proceeding after the proposed public payload and confirmation
behavior were explained. Phase 7.2 now provides reviewed publish/unpublish controls,
a public page and anonymous API. No existing business was automatically published;
migration 19 requires fresh confirmation for old prepared snapshots. See
PUBLICATION_REVIEW.md. This supersedes the publishing approval blocks above.
Customer discovery, service selection and booking actions remain upcoming Phase 7 work.

# Booking-range and owner setup work — unfinished (2026-10-18)

The user moved to Phase 7.7 after finding the setup too complicated. Preserve the
in-progress per-service custom start/end booking controls and owner readiness
checklist, but do not count them as completed. Migration 21 was applied; backend
range tests passed. The end-to-end owner-to-customer range browser flow has not
passed: the last run stopped locating the booking-mode control. Review and simplify
this flow, finish its browser verification and document it before declaring it done.

The real published Wedding Photography service had no bookings when investigated
on 2026-10-17. Its assigned staff window was expired and Sony A30 had no availability
window. No actual owner hours, assignments or availability windows were changed.
The owner still needs to choose the intended availability. Phase 7.7 cancellation
does not fix that configuration. Temporary verification fixtures were removed.


## Booking-range and availability follow-up resolved (2026-10-18)

This supersedes the unfinished range/setup and real-business window notes above.
The user approved staff/resources following business location hours. The matching
staff-location assignment was repaired in the earlier availability task and the real
calendar was verified to offer slots. No individual dated windows are now required.
Phase 7.9 completed the owner-to-customer range browser flow: accessible controls,
opening-hours setup, pending-save publication, 25-hour booking, overlap rejection
and cancellation passed at 320/390/768/1280px. See CUSTOMER_BOOKING_RANGES.md and
PHASE_7_CLOSURE.md. This gap is resolved; Apple/race/production and broader browser
coverage remain deferred as documented above.
