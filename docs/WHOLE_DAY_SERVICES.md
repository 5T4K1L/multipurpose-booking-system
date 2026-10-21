# Whole-day services and simpler scheduling

Whole day means exactly 24 elapsed hours from the customer's selected start time, as requested. It is not opening-to-closing and not midnight-to-midnight. Daylight-saving changes can shift the displayed local ending hour.

## Owner workflow

Create or edit a service and select Whole day. Minutes are hidden and the API normalizes the stored duration to 1,440 minutes. Timed appointments retain the existing duration field. Whole-day services cannot enable customer-selected lengths. Existing custom-duration settings are ignored while whole-day mode is active.

Set branch hours under When customers can book. Whole-day bookings start during branch opening hours and may continue overnight after closing. Explicit closed dates and existing bookings still block the full occupied interval, including buffers. Timed and custom-length bookings retain the full-interval opening-hours requirement. Overnight opening hours contains the explicit shortcut for setting all days to 24 hours; it does not run automatically. Publish the reviewed profile after changing the service mode. No existing service is converted automatically and existing bookings retain their times and their duration when rescheduled.

## UI

Branch and service selection remains highlighted tiles. Opening-hour rows align day, opening time, closing time and Remove. The day selector expands on demand and closes after selection. Refresh/add/remove/reset are quiet secondary controls; Save appears only for changed forms. Booking-mode controls, catalog item actions and spacing use the same workspace styles. Customer service cards, booking summaries and publication previews display Whole day (24 hours).

## Validation

- Migration 23 is additive and enforces a 1,440-minute duration for whole-day services.
- PostgreSQL-backed tests cover persistence, validation, 24-hour slots, starts inside opening hours, overnight continuation, and explicit closed-date protection.
- Live browser/API fixture covers publication, guest booking, exactly 24 hours, overlap rejection, forged shorter-duration rejection, rescheduling, cancellation and restored availability.
- Mocked UI checks cover whole-day creation, weekday selection, deletion controls and 320/768/1280 px light/dark layouts.
- Frontend lint/build and Go tests/build run for this change.

## Files changed

Frontend:
- apps/web/app/business/service-duration.tsx (new)
- apps/web/app/business/catalog.tsx
- apps/web/app/business/edit.tsx
- apps/web/app/business/scheduling.tsx
- apps/web/app/business/booking-options.tsx
- apps/web/app/business/public-profile.tsx
- apps/web/app/business/publication.tsx
- apps/web/app/bookings/api.ts
- apps/web/app/businesses/[id]/page.tsx
- apps/web/app/businesses/[id]/availability.tsx
- apps/web/app/globals.css
- apps/web/tests/workspace-choices.browser.mjs
- apps/web/tests/whole-day.browser.mjs (new)

Backend and schema:
- migrations/000023_whole_day_services.up.sql (new)
- migrations/000023_whole_day_services.down.sql (new)
- apps/api/internal/db/queries/booking_catalog.sql
- apps/api/internal/db/queries/catalog_updates.sql
- apps/api/internal/db/queries/booking_options.sql
- apps/api/internal/db/queries/public_discovery.sql
- apps/api/internal/db/models.go (generated)
- apps/api/internal/db/booking_catalog.sql.go (generated)
- apps/api/internal/db/catalog_updates.sql.go (generated)
- apps/api/internal/db/booking_options.sql.go (generated)
- apps/api/internal/db/public_discovery.sql.go (generated)
- apps/api/internal/service/booking_catalog.go
- apps/api/internal/service/catalog_updates.go
- apps/api/internal/service/booking_options.go
- apps/api/internal/service/business_publication.go
- apps/api/internal/service/public_availability.go
- apps/api/internal/server/catalog.go
- apps/api/internal/server/catalog_updates.go
- apps/api/internal/server/catalog_updates_integration_test.go
- docs/WHOLE_DAY_SERVICES.md (new)
