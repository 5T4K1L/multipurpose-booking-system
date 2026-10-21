# Business setup clarity and publishing fixes

Completed 2026-10-17 at the user's request before Phase 7.7. Customer cancellation remains the next milestone; it was not implemented in this fix.

## What owners see

- Locations: where customers receive services, including an online location.
- Services: what customers book and how long it takes.
- Staff: the people who provide services. Optional custom staff roles (such as Barber, Tutor or Therapist) group people for assignment. They do not grant login or management permissions.
- Resources: rooms, chairs, vehicles or equipment used by bookings. Capacity is concurrent bookings, not account access or staff count.
- Assignments: people who provide each service and where those people work. Custom role filters use the business's staff labels.
- Service setup: locations and resources for a service, kept separate from people assignments.
- Scheduling: opening hours, closed dates, preparation/cleanup, advance notice and staff/resource availability. Existing UTC dated-window limitations remain clearly labelled.
- Team access: which existing accounts can sign in to the business workspace and what they can change. Only the owner can change access. Custom staff roles never affect this permission system.

## Save and publish

Both **Publish reviewed profile** and **Publish reviewed update** save pending workspace forms before fetching a fresh preview and publishing it with its current review token. The confirmation checkbox explicitly covers pending edits. Included forms are business/catalog edits, pending catalog creation, customization, selected pending assignments, the open scheduling editor, and pending team-access changes (which retain their explicit account-access confirmation).

HTML-required/type/range validation runs for all pending forms before any saves. Saves run sequentially; an API/domain validation failure stops publication and reports which section failed. Earlier saves may already have completed: this is not an atomic batch transaction. Pending failed drafts remain available. Workspace controls are inert throughout saving and publishing. The server still checks owner/admin access and rejects stale publication tokens if another writer changes the preview before publication.

New catalog creation keeps its existing non-idempotent API behavior: if its response is interrupted, inspect the refreshed list before retrying to avoid duplicates. Unpublishing does not save pending edits and remains available independently of incomplete setup. Drafts are local to the open forms, not durable autosaved records.

## A service has no published location

This means no active, linked location is included in that service's reviewed public snapshot. It can happen before setup is finished, after a location is removed/archived, or when old snapshots have not been republished. It is not an authentication problem.

Owner steps:

1. Add a location under **Locations**.
2. In **Service setup**, select **Service locations**, choose the service, then choose its location.
3. Set location hours and service booking rules under **Scheduling**; assigned people/resources also need their location links and availability.
4. Publish the profile/update. Pending assignment selections are saved by Publish, or can be saved separately with Add assignment.

The owner publishing UI stops if any service in the fresh preview lacks a location and identifies those services. The public page now says: "Online booking is not available for this service yet. Please contact the business for help." Existing public APIs retain their snapshot/eligibility rules; this UI guidance does not automatically assign a location or invent hours.

## Data and validation

Migration 20 adds optional staff_profiles.role_name, bounded to 60 characters. Existing rows default to an empty label. Create/update/read/list support the label under the existing tenant and role checks. Labels are trimmed, UTF-8/control-character checked and remain private catalog metadata. Downgrade refuses to discard nonempty labels.

Database-enabled Go tests/build/vet and gofmt passed, including role persistence, normalization, invalid input and existing authorization regression checks. Frontend lint/build and responsive browser checks passed at 320/390/768/1280 pixels. Browser tests covered saving pending person creation, business/service edits, assignments and scheduling through Publish; the update button; invalid input; injected save failure; preserved drafts/publication; missing-location guidance; role persistence and no implicit account access. Test accounts/businesses were removed. Schema 20 is clean.

## Files changed

- `apps/api/internal/db/queries/booking_catalog.sql`
- `apps/api/internal/db/queries/catalog_updates.sql`
- `apps/api/internal/db/booking_catalog.sql.go`
- `apps/api/internal/db/catalog_updates.sql.go`
- `apps/api/internal/db/models.go`
- `apps/api/internal/service/booking_catalog.go`
- `apps/api/internal/service/booking_catalog_test.go`
- `apps/api/internal/service/catalog_updates.go`
- `apps/api/internal/server/catalog.go`
- `apps/api/internal/server/catalog_updates.go`
- `apps/api/internal/server/catalog_updates_integration_test.go`
- `apps/web/app/business/page.tsx`
- `apps/web/app/business/catalog.tsx`
- `apps/web/app/business/drafts.tsx`
- `apps/web/app/business/edit.tsx`
- `apps/web/app/business/assignments.tsx`
- `apps/web/app/business/scheduling.tsx`
- `apps/web/app/business/settings.tsx`
- `apps/web/app/business/publication.tsx`
- `apps/web/app/bookings/api.ts`
- `apps/web/app/businesses/[id]/availability.tsx`
- `migrations/000020_staff_role_labels.up.sql`
- `migrations/000020_staff_role_labels.down.sql`
- `docs/BUSINESS_SETUP_CLARITY.md`
- `docs/CURRENT_STATE.md`
- `docs/PUBLICATION_REVIEW.md`
- `docs/ROADMAP.md`

Local browser verification: `work/booking-ui-check/workspace-clarity-smoke.cjs` (outside the repository).
