# Phase 6.3 - Business profile and catalog editing

Completed 2026-10-13. Phase 6 remains in progress.

## Use

Open http://localhost:3000/business, sign in and select your business.
Owner/admin sees Edit business profile and an Edit button on each catalog item.
Each form loads current values before editing. Change fields, Save changes, then
reload to check persistence. Cancel edit discards the open form.

Editable fields:

- Business: name and default timezone.
- Location: name and timezone.
- Service: name and duration in minutes.
- Staff profile: display name (not a login identity or membership).
- Resource: name and capacity, subject to booking-history protection.

Business timezone changes do not change existing location timezones. Location timezone
changes reinterpret local weekly hours and closed dates; review and reload Scheduling
afterward. Stored booking instants stay fixed. Service duration changes apply to new
bookings and reschedules; existing bookings retain their duration/timestamp snapshot.
Resource location is deliberately immutable in this API to preserve eligibility links.
Any booking history, including terminal bookings, prevents capacity changes; renaming
remains allowed. A denied capacity change returns 409 and saves no partial rename.

## API and security

PUT /api/v1/businesses/{businessID}: {name, timezone}
PUT /api/v1/businesses/{businessID}/catalog/locations/{id}: {name, timezone}
PUT /api/v1/businesses/{businessID}/catalog/services/{id}: {name, duration_minutes}
PUT /api/v1/businesses/{businessID}/catalog/staff/{id}: {name}
PUT /api/v1/businesses/{businessID}/catalog/resources/{id}: {name, capacity}

All return 200 with the existing public view after transaction commit. Names use the
existing 1-100 character validation. Timezones must load as IANA/UTC zones, not Local.
Duration is 1-10080 minutes; capacity is a positive int32. These PUTs replace the listed
fields only; omitted/invalid required values are rejected. Extra fields, null objects,
trailing JSON, oversized payloads and unsupported content types are rejected.

Only active owner/admin membership can edit. Business locks serialize changes with
booking/configuration writers. Catalog queries require business ID and item ID and
active status. Foreign/missing/archived items share 404 behavior after tenant checks;
nonmembers, manager/staff and revoked access are denied. Session, origin, rate limits,
no-store headers and safe error mapping remain in force. Assignment collection routes
remain GET-only. There is no platform-admin bypass or client-supplied identity.

No schema migration or new dependency. Last committed edit wins; optimistic versioning
is not implemented. Reload after an uncertain response before retrying. Save/cancel
before switching business or navigating: edit drafts are not persisted. These are the
existing profile fields; branding, descriptions, contact details, publishing and
archive/restore controls are not included in this milestone.

## Verification

- sqlc generation and gofmt completed.
- PostgreSQL-enabled go test ./... -count=1, go build ./... and go vet ./... passed.
- Integration tests cover persistence, owner/admin edits, staff/unauthenticated/revoked
  denial, foreign tenant IDs, archived items, strict payloads, hostile origins,
  preserved booking times and atomic rejection of booked resource capacity changes.
- Frontend lint, production build and TypeScript checks passed.
- Headless Chrome edited business and all four catalog types, reloaded persisted
  names/timezones/duration/capacity, and verified staff cannot see editing controls.
- No page errors or horizontal overflow at 320/390/768 widths. Mobile screenshot
  reviewed. Temporary browser and API fixtures were cleaned up.

Next: Phase 6.4 business booking management, including business-scoped views and
marking delivered bookings completed. Customer ownership/self-service stays Phase 7.
Full Phase 6 closure still requires remaining dashboard/calendar/settings work.
