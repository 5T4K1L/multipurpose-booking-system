# Simplified business settings - 2026-10-19

Scheduling now presents two direct sections: Opening hours, and Booking length
and notice. The booking-readiness diagnostics and scheduling-category dropdown
are removed from the workspace. Staff/resource schedule categories are not shown.
Branches and services use named buttons when multiple items exist; the sole item
opens automatically. Existing hours, closures, buffers, notice and booking-length
controls still save through the same APIs. Booking length uses radio choices.
Unsaved-change checks and save-on-publish registration remain active.

Assignments, Resources and Service setup are removed as standalone UI panels.
People & access retains editable staff profiles and team access. Service cards
contain Available at branch checkboxes, with immediate per-branch saving, so
owners can still make services bookable without a separate setup screen.
Locations, services, staff, hours, booking rules, profile details and photos retain
their edit controls. The business name remains fixed as previously requested.
Existing assignment/resource data and backend booking capacity checks are retained.
No migration or deletion of user data was performed.

Validation: frontend lint and production build passed. Live browser checks verified
branch unlink/relink, saving opening hours for a selected branch, removed panels
and selectors, booking-mode radio choices and no overflow at 320/768/1280 pixels.
The existing live guest lifecycle test also passed, including availability conflicts,
rescheduling, cancellation, deletion, recovery and photo publication. Fixtures were
cleaned up. No backend code changed in this revision.

Changed files:
- apps/web/app/business/scheduling.tsx
- apps/web/app/business/booking-options.tsx
- apps/web/app/business/catalog.tsx
- apps/web/app/business/service-branches.tsx (new)
- docs/SIMPLE_BUSINESS_SETTINGS.md (new)
