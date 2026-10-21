# Customer booking history - Phase 7.6

Completed 2026-10-16. No migration/configuration changes; schema remains 19.

## Access and use

Open `/my-bookings`, or choose **My bookings** from `/account`. A successful customer booking confirmation links directly to `/my-bookings/{id}`.

History is newest-created first, 20 rows per UI page with **Load more bookings**. Details show current status, reference, UTC start/end, business/service/location IDs, and links to original/replacement bookings when rescheduled. **Refresh bookings** reloads current saved state. Returning focus to the tab clears and refreshes displayed data. Signed-out users receive a sign-in link; foreign/missing bookings receive the same unavailable message.

Customer history is independent of business membership and publication. Unpublishing/disabled businesses or archived catalog items do not erase a customer's existing records. Business management remains in `/business/bookings`; business roles never grant access to another customer's history route. Existing operator records with NULL customer_id are excluded, even if created_by matches the viewer.

## API

`GET /api/v1/customer/bookings?limit=20&cursor=...`

Active authenticated session required. Only single limit/cursor query values are accepted. Default/max API limit is 100. Pagination is descending `(created_at,id)` and scoped to the authenticated customer. Cursors are positions, not authorization; every SQL query independently enforces customer_id and active customer status. New insertions are picked up by refreshing from the first page.

Responses are JSON arrays using the existing customer-safe item projection, with no staff/resource assignments, customer IDs, audit actors or idempotency data. `X-Next-Cursor` is exposed through exact-origin credentialed CORS. No-store applies to all responses. Malformed/cross-customer cursors return 400. Missing/invalid sessions return 401; dependency failures return 503 with no partial rows. Failed transactions return no history payload.

Existing `GET /api/v1/customer/bookings/{id}` remains the detail API. POST creation behavior and its exact-Origin requirement remain unchanged. GET accepts an absent Origin for non-browser clients, but rejects foreign origins. The collection supports GET/POST/OPTIONS only.

## Validation

- Database-enabled `go test ./... -count=1`, `go build ./...`, `go vet ./...`, gofmt and diff whitespace checks passed.
- Frontend ESLint and production build passed.
- Integration checks cover pagination order, original/replacement chain, current cancellation state, private projections, unowned/operator exclusion, other-account cursor rejection, malformed inputs, authentication, disabled customers, disabled/unpublished businesses, CORS/no-store and transaction failure.
- Browser verification passed creation-to-details navigation, 22 fixture records across pages (including equal creation timestamps), signed-out and foreign-owner screens, history after unpublish, and widths 320/390/768/1280 with no runtime errors. Temporary accounts/business/bookings were removed.

## Limits and next step

This functional UI deliberately displays UTC and stable IDs. Historical display names/timezones have not been snapshotted, so it does not expose private live catalog edits or imply current names are historical facts. Saved display labels and presentation polish remain future work. Business changes appear on refresh, not via push notifications.

Phase 7.7 is customer cancellation with ownership enforcement and retained history. Customer rescheduling follows. This milestone adds no customer lifecycle mutations, notifications or deployment.

# Phase 7.7 update (2026-10-18)

Customer booking details now include a simple cancellation confirmation, the
server-calculated deadline and current cancellation eligibility. Cancelled records
remain in history. See [CUSTOMER_BOOKING_CANCELLATION.md](CUSTOMER_BOOKING_CANCELLATION.md).
