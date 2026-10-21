# Owner account and workspace revisions - 2026-10-19

- The four business workspace pages display a fixed business name instead of a selector.
  The owned business is preferred over team memberships. Legacy /bookings redirects
  to /business/bookings instead of providing another multi-business setup screen.
- The business name is entered at initial setup and is read-only afterward. The
  API rejects renaming too. Business timezone and other customization remain editable.
- Account displays the business name and its initial. Before setup it shows
  "Set up your business" rather than the authentication username.
- Login hides Your account and Sign out for signed-out visitors. Email/password
  errors appear beneath their fields; service/network failures appear below submit.
  Incorrect passwords and unknown accounts share a clear credentials message.
- Registration has repeat password, live local validation and inline duplicate-email
  feedback after submission. Duplicate registration now returns HTTP 409; existing
  passwords and OAuth identities are preserved. No live email-lookup endpoint was added.
- Google OAuth appears below Sign in/Create account, using existing provider discovery.

Validation: database-backed go test ./..., go build ./..., frontend lint and production
build passed. Browser checks covered inline validation, mismatched passwords blocking
submission, duplicate responses, OAuth placement, signed-out controls, account name,
and 320/768/1280 widths. Live checks verified all four workspace pages, API rename
rejection and duplicate registration, and repeated the guest booking lifecycle test.
Test fixtures were cleaned up; existing user data was not reset. The local API was
restarted with these changes. No new manual configuration or migration is required.

## Changed files

- apps/web/app/components/owner-auth-form.tsx
- apps/web/app/login/page.tsx
- apps/web/app/login/oauth-buttons.tsx
- apps/web/app/register/page.tsx
- apps/web/app/account/page.tsx
- apps/web/app/bookings/page.tsx
- apps/web/app/business/page.tsx
- apps/web/app/business/bookings/page.tsx
- apps/web/app/business/calendar/page.tsx
- apps/web/app/business/customers/page.tsx
- apps/web/app/business/catalog.tsx
- apps/web/app/business/edit.tsx
- apps/web/tests/owner-auth.browser.mjs
- apps/api/internal/service/registration.go
- apps/api/internal/service/registration_test.go
- apps/api/internal/service/catalog_updates.go
- apps/api/internal/server/registration.go
- apps/api/internal/server/registration_test.go
- apps/api/internal/server/catalog_updates_integration_test.go
- apps/api/internal/server/booking_calendar_integration_test.go
- apps/api/internal/repository/registration_integration_test.go
- docs/REGISTRATION.md
- docs/OWNER_ACCOUNT_REVISIONS.md
