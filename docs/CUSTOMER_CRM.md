# Phase 8.1 - Business customer table

Implemented 2026-10-19. PostgreSQL remains the source of truth. This foundation is
read-only and derives the business/customer relationship from bookings.customer_id.
No new table or migration; schema remains 21. It does not infer customer identity
from the booking creator, business membership, matching email, or legacy bookings.

## Access and scope

Open `/business/customers` from the Customers link in Business workspace. Owners
and business admins can select a business, search name/email, refresh, and paginate.
Each row shows current account name/email, number of booking records, number still
confirmed, and latest booking start time. Cancelled/rescheduled/completed history
remains in the totals. Rescheduling creates another historical record, so totals
are explicitly labelled booking records, not visits. Latest time can be in the
future and can refer to a cancelled record. Dates use the viewer's device timezone.

GET `/api/v1/businesses/{businessID}/customers?limit=25&q=text&cursor=...` returns an
array and optional X-Next-Cursor header. Default limit 100, maximum 100, stable
ascending customer-ID order. Search is case-insensitive literal substring matching
on name/email; SQL wildcard characters are literal. Search is trimmed, UTF-8 and
at most 200 bytes. Duplicate or unknown query parameters are rejected.

Fresh active tenant membership is checked on every page through WithinTenant.
Only BUSINESS_OWNER/BUSINESS_ADMIN are allowed; manager/staff/customer/platform
roles alone grant no access. Every count and relationship query is scoped to the
selected business. Cursors are bound to actor, business and search; they are not
credentials. Responses are no-store, use existing exact-origin CORS and session
handling, and have a ten-second operation deadline. No customer data is public.

The UI clears data on business change and tab return/account refresh. API failures
clear paginated results. The table scrolls inside its own region on small screens.
Current account contact data is shown, not a separately editable CRM contact copy.

## Validation

- Database-enabled Go tests, build, vet and gofmt.
- Integration coverage: tenant/customer/anonymous denial, origin, unsupported writes,
  owner/admin permissions and revocation, exclusion of legacy records and booking
  creators, pagination, cursor scope, literal/case-insensitive search and no-store.
- Frontend lint/build and isolated browser fixture: booked customer, search, empty
  state, cancellation updates counts while retaining the relationship; widths
  320/390/768/1280, no page errors or document overflow. Fixtures removed afterward.

## Remaining Phase 8 work

Next: Phase 8.2 customer filtering and sorting. Later milestones must define manual
contacts, editable business-specific fields, bulk operations, import/export preview,
column mapping, validation, deduplication, warnings, confirmation and audit. Do not
modify global account identity through CRM editing. Final UI redesign is separate.
This is not completion of the full CRM/import/export phase.

## Files changed

- apps/api/internal/db/queries/customer_crm.sql
- apps/api/internal/db/customer_crm.sql.go (generated)
- apps/api/internal/service/customer_crm.go
- apps/api/internal/service/business.go
- apps/api/internal/server/business.go
- apps/api/internal/server/customer_foundation_integration_test.go
- apps/web/app/business/customers/page.tsx
- apps/web/app/business/page.tsx
- docs/CUSTOMER_CRM.md
- docs/CURRENT_STATE.md
- docs/ROADMAP.md
