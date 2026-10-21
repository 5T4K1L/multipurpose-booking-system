# Phase 6 completion review

Reviewed 2026-10-14. Phase 6.7 closes the Phase 6 development scope. This is not production approval or unconditional closure of previously deferred verification.

## Delivered scope

| Roadmap area                          | Functional implementation                                                              | Detail and test instructions               |
| ------------------------------------- | -------------------------------------------------------------------------------------- | ------------------------------------------ |
| Business profile                      | Create/edit name and timezone; timezone choices                                        | BUSINESS_WORKSPACE.md, BUSINESS_EDITING.md |
| Services, staff, resources, locations | Create/read/edit catalog and eligibility assignments; protect booked resource capacity | BUSINESS_WORKSPACE.md, BUSINESS_EDITING.md |
| Scheduling                            | Weekly hours, closures, buffers/notice, dated availability                             | BUSINESS_SCHEDULING_UI.md                  |
| Booking management                    | Business-scoped history, reschedule, cancel, complete after cleanup ends               | BUSINESS_BOOKING_MANAGEMENT.md             |
| Calendar and dashboard                | Business-local date agenda, pagination and full-period status counts                   | BUSINESS_CALENDAR.md                       |
| Business settings                     | Saved introduction/contact details, escaped preview, owner-managed account membership  | BUSINESS_SETTINGS.md                       |

All areas have a functional responsive UI. Manager/staff access is currently read-only; owner/admin can manage business data and bookings, and only the owner changes account membership. Bookable staff profiles are separate from account access. API authorization is authoritative on every request.

## Verification

Fresh Phase 6.7 checks passed:

- Database-enabled `go test ./... -count=1`, `go build ./...`, and `go vet ./...` from apps/api.
- `gofmt -l` over service Go files returned no files; `git diff --check` passed.
- `npm run lint` and `npm run build` from apps/web, including TypeScript compilation.
- Local schema version 16, dirty=false.
- Isolated Chrome settings flow: save/reload persistence, markup displayed as text, separate temporary member add/update/remove; no browser page errors.
- Isolated Chrome calendar flow: complete a fixture booking, completed count 1, empty-period count 0, date selection; no browser page errors.
- Browser flows checked horizontal overflow at widths 320, 390 and 768, with desktop 1280 captures. Earlier milestone documents record catalog/editing/scheduling and reschedule/cancel checks; those were not all rerun during this closure review.

Browser fixtures were removed. No application code or schema changes were needed for this review. Existing uncommitted work was preserved.

## Try the phase

1. Open http://localhost:3000/login and sign in with a password account.
2. Open http://localhost:3000/business. Create/select a business, then configure its catalog, assignments and scheduling. Save each section before navigating away.
3. Edit business customization and reload to verify persistence. Team access grants existing accounts access immediately; use only an intended account ID from /account.
4. The existing /bookings screen is an internal operations/test tool for creating test bookings, not customer self-service. Use test data when trying it.
5. At /business/bookings, inspect, reschedule or cancel an eligible future booking. Complete only a delivered booking after its cleanup period ends.
6. At /business/calendar, select the business and dates to inspect the agenda and status counts. Refresh after mutations to see current persisted state.

No new manual configuration is required for the password-based local flow.

## Limits carried forward

- Customer ownership and self-service are not implemented. Operator-created test records must not be treated as customer-owned records by inference.
- Settings are not published publicly. Publishing/discovery, customer pages and customer booking permissions belong to Phase 7.
- Calendar counts describe booking history records, not unique visits or revenue. The agenda is not a drag-and-drop month calendar; date queries need scale review before production.
- No invitations, owner transfer, branding uploads, pricing/forms customization, notifications or live push are implied by this phase. Existing catalog/UI loading limits and whole-form save semantics remain documented in the feature guides.
- Beginner-friendly UI redesign remains deferred by user instruction until platform functionality is implemented. Cross-browser/accessibility, load and production review remain outstanding.
- Google local credentials must be restored to enable its local launcher. Apple live setup and Go race-detector toolchain checks remain in DEFERRED_WORK.md.

## Next boundary

Phase 7.1: define and implement customer booking ownership and business publishing boundaries, with API tests for cross-customer and cross-business denial, before exposing customer booking CRUD. Customer deletion is cancellation with history retained; business and customer views must share persisted booking state. Existing internal records need an explicit compatibility policy. Deliver the corresponding customer UI incrementally and a usable responsive UI at the end of Phase 7. Payment and messaging work must follow their planned dependencies rather than being implied by this first slice.

Stop here: no Phase 7 implementation was included in Phase 6.7.
