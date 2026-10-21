# Booking availability fix - 2026-10-19

## Confirmed causes

Docker Desktop/PostgreSQL and the web server were stopped. They have been restored,
and the API restarted with this fix. Existing data was preserved.

The published Three A's services had no service_scheduling_rules rows. Previously,
missing optional rules silently produced zero available times. The two-hour service
also had a staff assignment without any staff-location links and equipment assigned
only at one of its two published branches. These links hid otherwise valid times.

## Behavior now

- Missing optional rules mean zero preparation, cleanup and advance notice. Saved
  values still apply. Storage failures remain errors rather than becoming defaults.
  Availability, occupancy checks, customer creation and operator booking preparation
  use the same rule resolution.
- Staff without explicit branch assignments follow the service's branches. Staff
  with explicit assignments retain those restrictions. Archived staff remain excluded.
- Equipment is selected from the current branch. Equipment linked only at another
  branch does not block this branch. Existing local equipment capacity and staff
  conflicts remain enforced; archived local equipment does not silently become free.
- Opening hours, closed dates, duration, timezone, publication and tenant checks remain.
- Optional preparation/notice controls are collapsed. Owners do not need to save
  zero values to enable bookings. Publishing instructions point to Available at on
  each service card, rather than the removed Service setup panel.

No migration, database reset, guessed opening hours or automatic publication was
performed. Existing booking history, saved limits and publication choices were preserved.

## Verified results

For October 21, 2026, the real published profile returned 33 available times for the
120-minute service at Brgy. San Roque, 25 at Camiling, Tarlac and 39 for the 30-minute
service at Brgy. San Roque. Availability can change as bookings are made.

Live browser regression used temporary fixtures mirroring all three missing/default
conditions. Customer creation succeeded at both branches; occupied-time rejection,
contact editing, recovery, tenant/guest isolation, rescheduling, cancellation, deletion
and image publication checks passed. Only fixture data was removed afterward.
The user's business was checked through read-only public availability requests.

Validation: gofmt, database-backed go test ./..., go build ./..., go vet ./...,
frontend lint and production build passed. The new default-availability browser test
also checked guest layout at 320/768/1280 pixels.

## Local use

Refresh http://localhost:3000/businesses and choose your business, service, branch
and an open date. No extra zero-value configuration is needed. Docker Desktop must
remain running for PostgreSQL. After a machine restart, start Docker Desktop, run
`docker compose up -d postgres` from the repository, and start the API and web server
using your existing local launch setup. Do not reset the database.

## Changed files

- apps/api/internal/service/scheduling.go
- apps/api/internal/service/slots.go
- apps/api/internal/service/bookings.go
- apps/api/internal/service/booking_occupancy.go
- apps/api/internal/service/customer_booking_create.go
- apps/api/internal/service/scheduling_defaults_test.go
- apps/api/internal/server/availability_integration_test.go
- apps/api/internal/db/queries/availability.sql
- apps/api/internal/db/queries/public_availability.sql
- apps/api/internal/db/queries/booking_readiness.sql
- apps/api/internal/db/availability.sql.go
- apps/api/internal/db/public_availability.sql.go
- apps/api/internal/db/booking_readiness.sql.go
- apps/web/app/business/scheduling.tsx
- apps/web/app/business/publication.tsx
- apps/web/tests/default-availability.browser.mjs
- docs/BOOKING_AVAILABILITY_FIX.md
- docs/CURRENT_STATE.md
