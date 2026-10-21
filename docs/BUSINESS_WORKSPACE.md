# Phase 6.1 - Business workspace and catalog UI

Completed 2026-10-13. Phase 6 remains in progress.

Open http://localhost:3000/business with the API on localhost:8080. Sign in, then:

1. Create a business with its name and IANA timezone, or select an existing business.
2. Add a location, a service with its duration, and a bookable staff profile.
3. Under Resources, load locations, choose one and create a resource with its capacity.
4. Use Assignments to connect services to locations, staff and resources, and staff
   to locations. Select the parent, select an item and add the assignment. Remove
   unneeded assignments with Remove. Existing booking history is retained.
5. Reload to verify persistence. Catalog and business lists provide pagination.

BUSINESS_OWNER and BUSINESS_ADMIN can create catalog items and manage assignments.
MANAGER and STAFF see read-only catalog controls. Every request still checks current
server-side tenant access. Staff profiles are bookable entities, not login accounts.
The home, account and booking screens link to this workspace.

## Scope and limitations

This UI uses existing APIs; no backend, schema, dependency or authentication changes.
It supports catalog creation, reading and assignment changes. Phase 6.3 adds catalog
and business profile editing; see BUSINESS_EDITING.md. Archiving, membership
administration UI and calendars remain later work.
Creation endpoints are not idempotent: after an interrupted response, refresh the list
before retrying to avoid duplicates. Each operation saves independently.
Catalog setup alone does not make times bookable. Location hours, closed dates, service
rules and staff/resource availability must also be configured. Phase 6.2 now provides those
scheduling forms; see BUSINESS_SCHEDULING_UI.md. Existing /bookings demo setup remains available for booking tests.

## Verification

- Frontend ESLint and Next production build with TypeScript passed.
- PostgreSQL-enabled go test ./... -count=1, go build ./... and go vet ./... passed.
- Isolated Chrome tested business creation, all four catalog types, four assignment
  types, removal, reload persistence, staff read-only UI and HTTP 403 on staff writes.
- No browser page errors. No horizontal overflow at 320, 390 or 768 pixels; desktop
  1280 and mobile 390 screenshots reviewed. Temporary test fixtures removed.
- No Go files changed, so gofmt was not needed.

The old Google development launcher references a credential JSON file that is now
missing. Current local verification used password sign-in and a separate temporary
launcher with Google disabled. Restore the credential file or update its launcher path
to resume Google testing. No secrets were written into the repository. Apple and other
release checks remain deferred in DEFERRED_WORK.md.

Phase 6.4: use the Business bookings navigation link for business-scoped history,
rescheduling/cancellation and marking delivered bookings completed. See
BUSINESS_BOOKING_MANAGEMENT.md. Creation/demo controls remain on the test workspace.

Phase 6.6 adds Business customization and Team access below Scheduling. Owner/admin
can save business descriptions/contact details; owner alone manages existing-account
memberships. See BUSINESS_SETTINGS.md. Saved preview does not publish a public page.
