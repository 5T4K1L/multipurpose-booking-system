# Start/end booking UI - 2026-10-19

Replaced the start dropdown with a grid of available time buttons. Added quick
booking-length buttons, separate native date/time fields for a custom end, and a
live selection/duration summary. Start changes reset the end to the minimum length.
The section contains no select dropdowns. Native date/time pickers vary by browser.

The existing backend range validation, confirmation, stale-selection clearing and
availability rechecks remain. Quick lengths are suggestions, not guaranteed slots;
the Check availability button verifies the entire interval before confirmation.
Device timezone is stated explicitly. Business duration limits and DST validation
are retained. No backend or business-data changes.

Validation: frontend lint/build and a mocked browser test passed. Browser checks
cover start changes, quick lengths, custom multi-day end, checking availability,
clearing stale confirmation, and 320/768/1280 layouts in light and dark modes.

Changed files:
- apps/web/app/businesses/[id]/booking-range.tsx
- apps/web/app/globals.css
- apps/web/tests/booking-range-design.browser.mjs
- docs/BOOKING_RANGE_UI.md
