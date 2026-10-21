# Workspace choices and deletion

Branch, service, weekday and booking-length choices use selectable tiles with visible active colors and keyboard focus. Catalog items have individual cards, consistent spacing and inline deletion confirmation. Large timezone lists retain their existing controls.

Owners and business admins can delete branches, services and staff from the active catalog. The API also supports resources. Deletion archives the record, retains historical booking references, and rejects items with confirmed bookings whose occupied interval has not ended. The tenant transaction serializes deletion with booking writes. Deleted staff/resources are detached from service eligibility so stale links do not block future availability. No database migration is needed.

After deleting a branch or service, publish the reviewed profile to update the public snapshot. Use Reload saved list in scheduling to refresh its choices without silently losing unsaved work.

Changed files:
- apps/web/app/business/catalog.tsx
- apps/web/app/business/service-branches.tsx
- apps/web/app/business/scheduling.tsx
- apps/web/app/business/booking-options.tsx
- apps/web/app/globals.css
- apps/web/tests/workspace-choices.browser.mjs
- apps/api/internal/db/queries/catalog_updates.sql
- apps/api/internal/db/catalog_updates.sql.go (generated)
- apps/api/internal/service/catalog_updates.go
- apps/api/internal/server/catalog.go
- apps/api/internal/server/catalog_integration_test.go
- apps/api/internal/server/catalog_updates_integration_test.go
- docs/WORKSPACE_CHOICES_AND_DELETION.md

Validation: PostgreSQL-backed go test ./..., go build ./..., frontend lint/build, and mocked browser checks for branch/weekday selection, delete cancellation/success/conflict, and 320/768/1280 px light/dark layouts. Existing customer booking behavior is unchanged.
