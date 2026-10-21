# Universal Booking Platform Roadmap

## Purpose

This document defines the phased development roadmap for the Universal Booking Platform.

The project is developed incrementally.

Only the currently authorized phase should be actively implemented unless an explicit architectural dependency requires otherwise.

Each phase should reach a verifiable completion state before the project moves forward.

---

# Product Vision

## Product

Universal Booking Platform

## Core Idea

> Book anything, from one place.

The platform is designed as universal booking infrastructure rather than a single-purpose appointment scheduler.

It should eventually support:

- salons
- barbers
- dentists
- clinics
- gyms
- tutors
- consultants
- photographers
- repair services
- automotive services
- wellness providers
- professional services
- events
- rooms
- equipment
- resource reservations

---

# Customer Journey

The intended customer journey is:

```text id="a1b2c3"
Find
 ↓
Compare
 ↓
Book
 ↓
Pay
 ↓
Manage
 ↓
Message
 ↓
Return
```

The customer experience should remain simple even as the underlying platform becomes more capable.

---

# Business Platform

Each business receives an isolated administrative environment.

Core business navigation is expected to include:

```text id="d4e5f6"
Dashboard
Bookings
Calendar
Inbox
Customers
Services
Team
Resources
Analytics
Automations
Payments
Reviews
Settings
```

Exact navigation may evolve with product development.

---

# Platform Administration

The platform also requires a separate administrative layer for platform-level operations.

Expected roles include:

```text id="g7h8i9"
PLATFORM_ADMIN
PLATFORM_OPERATOR
```

Platform administration must remain separate from business administration.

---

# Development Principles

The roadmap follows these principles:

```text id="j0k1l2"
Incremental development
Small verifiable changes
Security by phase
Server-side enforcement
PostgreSQL as source of truth
Responsive UI
Free-first development
Evidence-based scaling
Minimal unnecessary infrastructure
```

Large features should be broken into logical implementation steps while avoiding unnecessary micro-steps.

---

# Phase Status Summary

```text id="m3n4o5"
PHASE 0   Engineering Foundation              COMPLETE
PHASE 1   System Architecture                 COMPLETE
PHASE 2   Database Architecture               COMPLETE
PHASE 3   Authentication + Authorization      IN PROGRESS
PHASE 4   Multi-Tenancy                       COMPLETED (development scope)
PHASE 5   Core Booking Engine                 DEV COMPLETE
PHASE 6   Business SaaS + Business Admin      DEVELOPMENT COMPLETE
PHASE 7   Customer Platform + Customer UI     DEVELOPMENT COMPLETE
PHASE 8   Customer CRM + Import/Export        NOT STARTED
PHASE 9   Business ↔ Customer Messaging       NOT STARTED
PHASE 10  Background Jobs + Notifications     NOT STARTED
PHASE 11  Search + Marketplace                 NOT STARTED
PHASE 12  Payments + Financial Safety         NOT STARTED
PHASE 13  Security Hardening                  NOT STARTED
PHASE 14  Observability + Reliability         NOT STARTED
PHASE 15  Backups + Disaster Recovery         NOT STARTED
PHASE 16  CI/CD                               PARTIALLY PLANNED
PHASE 17  Infrastructure as Code              NOT STARTED
PHASE 18  Production Deployment               NOT STARTED
PHASE 19  Scaling                             NOT STARTED
PHASE 20  Advanced Infrastructure + API       NOT STARTED
```

CI/CD planning and development workspace practices exist, but the phase is not considered complete until its defined production-ready scope is implemented and verified.

---

# PHASE 0 — Engineering Foundation

## Status

COMPLETE

## Objective

Establish the repository and basic application foundation.

## Completed

```text id="p6q7r8"
Git repository foundation
Next.js frontend
Go API
PostgreSQL
SQL migrations
API health endpoint
Frontend-to-API connectivity
Docker Compose PostgreSQL
Development documentation
Basic verification
```

## Explicitly Out of Scope

At the completion of Phase 0, the following were intentionally not implemented:

```text id="s9t0u1"
Authentication
Authorization
RBAC
Multi-tenancy
Businesses
Services
Staff
Booking engine
CRM
Messaging
Payments
Marketplace
```

---

# PHASE 1 — System Architecture

## Status

COMPLETE

## Objective

Define the platform architecture before deep feature implementation.

## Completed

Architectural documentation established:

```text id="v2w3x4"
modular monolith
frontend/backend boundaries
API architecture
backend architecture
frontend architecture
package architecture
configuration architecture
application lifecycle
security architecture
database strategy
technical terminology
architecture decisions
roadmap
project context
```

The project also established the principle that the application is designed for long-term SaaS evolution without prematurely becoming a distributed microservice system.

---

# PHASE 2 — Database Architecture

## Status

COMPLETE

## Objective

Build the initial relational foundation for the platform.

## Completed

The database phase established:

```text id="y5z6a7"
PostgreSQL schema
versioned migrations
foreign keys
constraints
indexes
database conventions
sqlc integration
repository foundation
authentication-related schema foundations
domain-oriented relational design
```

The current implementation contains the database structures needed to support later authentication and domain phases.

---

# PHASE 3 — Authentication + Authorization

## Status

IN PROGRESS

## Objective

Enable the application to securely establish user identity and enforce application-level authorization.

Phase 3 establishes the identity and security foundation required by all later protected functionality.

---

## Phase 3 Sequence

The current Phase 3 sequence is:

```text id="b8c9d0"
3.1 Authentication Architecture
3.2 Authentication Data Model
3.3 Authentication Database Schema
3.4 Password Hashing
3.5 Authentication Repository
3.5-A Google + Apple OAuth Add-on
3.6 Authentication Service
3.7 Application Session / Token Management
3.8 Registration
3.9 Login
3.10 Logout + Session Revocation
3.11 Authentication Middleware
3.12 /auth/me
3.13 RBAC / Authorization
3.14 Authentication Security Tests
3.15 Phase 3 Closure
```

---

# Phase 3.1 — Authentication Architecture

## Status

COMPLETE

Defined:

```text id="e1f2g3"
authentication vs authorization
local application identity
password authentication
session architecture
OAuth architecture
account linking
security boundaries
authorization model
authentication endpoint concepts
```

---

# Phase 3.2 — Authentication Data Model

## Status

COMPLETE

Defined:

```text id="h4i5j6"
users
user_credentials
user_sessions
user_oauth_identities
```

The OAuth identity model uses:

```text id="k7l8m9"
provider + provider_subject
```

with database uniqueness constraints.

Email is not used as the permanent external OAuth identity key.

---

# Phase 3.3 — Authentication Database Schema

## Status

COMPLETE

The authentication schema foundation has been implemented using versioned migrations.

The OAuth identity table has been added.

Core relationships include:

```text id="n0o1p2"
user_credentials.user_id
        ↓
users.id

user_sessions.user_id
        ↓
users.id

user_oauth_identities.user_id
        ↓
users.id
```

---

# Phase 3.4 — Password Hashing

## Status

COMPLETE

The password architecture uses a secure password hashing strategy based on Argon2id.

Plaintext passwords must never be persisted.

Password verification belongs to the authentication layer.

---

# Phase 3.5 — Authentication Repository

## Status

COMPLETE

The repository foundation provides controlled database access for authentication-related persistence.

The repository layer uses:

```text id="q3r4s5"
Go
pgx
sqlc
PostgreSQL
```

Repository code does not make high-level authorization decisions.

---

# Phase 3.5-A — Google + Apple OAuth Add-on

## Status

COMPLETE AT FOUNDATION / INTEGRATION PENDING

This add-on was introduced before continuing to the main authentication service implementation.

## Implemented

```text id="t6u7v8"
OAuth identity migration
OAuth sqlc queries
OAuth repository
Google provider foundation
Apple provider foundation
state generation
OIDC nonce
PKCE S256
OAuth transaction
encrypted OAuth transaction cookie
OAuth configuration
OAuth authorization handlers
OAuth callbacks
authorization-code exchange foundation
OIDC claim validation
JWK-backed signature verification
OAuth identity normalization
Apple client-secret generation
OAuth tests
```

---

## OAuth Security Controls Implemented

The foundation includes:

```text id="w9x0y1"
cryptographically secure state
cryptographically secure nonce
PKCE S256
encrypted transaction
HttpOnly transaction cookie
SameSite=Lax
Secure in production
issuer validation
audience validation
nonce validation
expiration validation
issued-at validation
not-before validation where applicable
authorized-party validation where applicable
provider signing algorithm restriction
JWK signature verification
provider + subject identity model
```

---

## OAuth Identity Rules

The implementation follows:

```text id="a2b3c4"
provider + provider_subject
```

The database enforces:

```text id="d5e6f7"
UNIQUE(provider, provider_subject)
UNIQUE(user_id, provider)
```

Automatic account merging based solely on matching email is not allowed.

---

## OAuth Test Status

The OAuth foundation has been locally verified through:

```text id="g8h9i0"
go test ./internal/auth
go test ./internal/repository
go test ./...
go build ./...
```

Formatting and dependency issues identified during development were resolved.

Real Google and Apple provider credentials have not yet been treated as production-verified configuration.

---

## Remaining Phase 3.5-A Work

The following remains part of later integration:

```text id="j1k2l3"
authentication service integration
application session integration
complete route wiring
live Google verification
live Apple verification
frontend OAuth UI
```

These should not be treated as completed merely because the provider foundation exists.

---

# Phase 3.6 — Authentication Service

## Status

IMPLEMENTED - service layer; session and HTTP integration deferred

The authentication service is the central application layer for:

- password authentication
- OAuth identity resolution
- user creation
- account linking
- account status checks
- authentication policy
- session initiation (implemented by the Phase 3.7 session service)

The service must remain independent from HTTP transport details.

---

# Phase 3.7 — Application Session / Token Management

## Status

IMPLEMENTED AND DATABASE-VERIFIED - session service; HTTP delivery remains deferred.

## Objective

Implement the application's normal authenticated session.

The session system must support:

```text id="m4n5o6"
creation
validation
expiration
revocation
logout
account disablement
```

Password and OAuth authentication must converge on the same application session mechanism.

---

# Phase 3.8 — Registration

Status: IMPLEMENTED AND VERIFIED. See REGISTRATION.md for the API, form, unverified-email policy, and deployment limitations.

## Objective

Implement local account registration.

Requirements include:

```text id="p7q8r9"
input validation
password hashing
safe email handling
account creation
appropriate verification policy
duplicate handling
security logging
rate limiting
```

Registration must not expose unnecessary account-existence information.

---

# Phase 3.9 — Login

## Status

IMPLEMENTED - password login API, secure session-cookie delivery, and login form.
See LOGIN.md for security policy and phase boundaries.

## Objective

Implement password-based login.

The flow is:

```text id="s0t1u2"
request
 ↓
validate
 ↓
resolve local user
 ↓
verify credential
 ↓
check account status
 ↓
create application session
```

OAuth login will resolve through the same application identity/session boundary.

---

# Phase 3.10 — Logout + Session Revocation

## Objective

Implement reliable session termination.

Logout must invalidate the appropriate application session.

Security-sensitive situations may invalidate additional sessions according to policy.

---

# Phase 3.11 — Authentication Middleware

## Objective

Provide trusted authentication context to protected API requests.

The middleware will:

```text id="v3w4x5"
extract session
 ↓
validate session
 ↓
load user
 ↓
check account status
 ↓
attach trusted identity context
 ↓
continue request
```

Client-supplied identity values must not override this context.

---

# Phase 3.12 — `/auth/me`

## Objective

Provide a safe representation of the currently authenticated local user.

The endpoint must not expose:

- password hashes
- session secrets
- OAuth tokens
- private authentication data

The response should represent application identity rather than provider-specific secrets.

---

# Phase 3.13 — RBAC / Authorization

## Objective

Implement server-side authorization.

The initial role model includes:

```text id="y6z7a8"
CUSTOMER
BUSINESS_OWNER
BUSINESS_ADMIN
MANAGER
STAFF
PLATFORM_ADMIN
PLATFORM_OPERATOR
```

Authorization must later integrate with the multi-tenant membership model.

---

# Phase 3.14 — Authentication Security Tests

## Objective

Verify authentication and authorization security before Phase 3 closure.

Testing should cover:

```text id="b9c0d1"
password hashing
login failures
account enumeration
session expiration
session revocation
logout
OAuth state
OAuth nonce
PKCE
OIDC validation
OAuth identity ownership
account linking
RBAC
object-level authorization
authentication middleware
tenant-boundary assumptions
rate limiting
safe errors
secret redaction
```

---

# Phase 3.15 — Phase 3 Closure

Phase 3 is complete only after:

```text id="e2f3g4"
implementation complete
unit tests pass
integration tests pass
security tests pass
lint/static checks pass
backend build passes
frontend build passes
documentation updated
CURRENT_STATE.md updated
ROADMAP.md updated
known issues documented
manual configuration documented
final verification completed
```

Then development must stop at the phase boundary until the next phase is explicitly authorized.

---

# PHASE 4 — Multi-Tenancy

## Objective

Implement secure tenant and business isolation.

Expected areas:

```text id="h5i6j7"
businesses
memberships
roles
tenant context
tenant-scoped repositories
server-side tenant enforcement
cross-tenant security tests
```

Authentication from Phase 3 becomes the identity foundation.

---

# PHASE 5 — Core Booking Engine

## Objective

Build the authoritative booking and availability engine.

Expected areas:

```text id="k8l9m0"
services
staff
resources
locations
business hours
availability
holidays
buffers
lead time
booking creation
booking modification
cancellation
double-booking prevention
idempotency
```

PostgreSQL transactions and constraints must protect concurrent booking operations.

---

# PHASE 6 — Business SaaS + Business Admin

## Objective

Build the business-facing SaaS functionality.

Expected areas:

```text id="n1o2p3"
business profile
services
staff
resources
locations
calendar
booking management
business settings
business dashboard
```

The UI must remain responsive across supported screen sizes.

---

# PHASE 7 — Customer Platform + Customer UI

## Objective

Build the customer-facing booking experience.

Expected flow:

```text id="q4r5s6"
Find
Compare
Book
Pay
Manage
Message
Return
```

Expected areas:

```text id="t7u8v9"
customer account
search/discovery
business pages
service selection
availability
booking
booking history
rescheduling
cancellation
profile
```

---

# PHASE 8 — Customer CRM + Import/Export

## Objective

Implement the customer management system.

The CRM must provide a spreadsheet-like experience while maintaining PostgreSQL as the source of truth.

Expected capabilities:

```text id="w0x1y2"
customer table
search
filtering
sorting
bulk operations
import
export
preview
mapping
validation
deduplication
warnings
confirmation
audit
```

---

# PHASE 9 — Business ↔ Customer Messaging

## Objective

Implement first-class business/customer communication.

Expected capabilities:

```text id="a3b4c5"
individual conversation threads
booking context
staff assignment
internal notes
message history
read state
notifications
```

Messaging must remain tenant-scoped.

---

# PHASE 10 — Background Jobs + Notifications

## Objective

Introduce reliable asynchronous processing.

Potential capabilities:

```text id="d6e7f8"
email
SMS
push notifications
reminders
booking notifications
scheduled jobs
retries
dead-letter handling
outbox processing
```

A background queue system should only be introduced when actual workloads justify it.

---

# PHASE 11 — Search + Marketplace

## Objective

Enable platform-wide discovery.

Expected capabilities:

```text id="g9h0i1"
business discovery
service discovery
location-aware search
availability-aware discovery
filtering
sorting
marketplace pages
```

PostGIS or dedicated search infrastructure may be introduced if justified by actual search requirements.

---

# PHASE 12 — Payments + Financial Safety

## Objective

Implement secure payments, subscriptions, and financial workflows.

Expected areas:

```text id="j2k3l4"
payment providers
checkout
payment state
webhooks
idempotency
refunds
subscriptions
plans
entitlements
invoices
payment methods
```

Raw card information must never be stored by the platform.

Payment providers remain authoritative for externally verified payment events.

---

# PHASE 13 — Security Hardening

## Objective

Perform systematic platform-wide security hardening and verification.

Expected areas:

```text id="m5n6o7"
authentication
authorization
tenant isolation
API security
input validation
session security
OAuth
payment security
secrets
dependency security
logging
abuse prevention
security testing
security headers
```

The project targets OWASP ASVS Level 2 as the baseline, with selective Level 3 controls for sensitive systems.

No compliance claim is made until the required verification has actually been completed.

---

# PHASE 14 — Observability + Reliability

## Objective

Make production behavior measurable and diagnosable.

Expected capabilities:

```text id="p8q9r0"
structured logging
metrics
tracing
health checks
readiness checks
error monitoring
request correlation
performance monitoring
```

Tools such as OpenTelemetry and Sentry may be introduced when operationally justified.

---

# PHASE 15 — Backups + Disaster Recovery

## Objective

Make production data recoverable.

Expected areas:

```text id="s1t2u3"
automated backups
retention
restore testing
recovery procedures
point-in-time recovery where appropriate
off-site protection
recovery objectives
```

A backup strategy is incomplete until restoration has been tested.

---

# PHASE 16 — CI/CD

## Status

PARTIALLY PLANNED

## Objective

Automate reliable build, test, validation, and deployment workflows.

Expected pipeline stages include:

```text id="v4w5x6"
format
lint
test
build
security checks
artifact generation
deployment
```

The existing development workflow also includes a coding workspace/environment strategy.

The phase is not complete until the intended automated pipeline is implemented and verified.

---

# PHASE 17 — Infrastructure as Code

## Objective

Make infrastructure reproducible.

Potential tooling:

```text id="y7z8a9"
OpenTofu
cloud provider infrastructure
networking
databases
secrets integration
storage
observability
```

Infrastructure abstraction must remain proportional to actual deployment requirements.

---

# PHASE 18 — Production Deployment

## Objective

Deploy the platform into a secure production environment.

Expected areas:

```text id="b0c1d2"
HTTPS
domains
production database
secret management
logging
monitoring
backups
deployment process
health checks
rollback process
```

Production deployment should follow completed security and reliability requirements.

---

# PHASE 19 — Scaling

## Objective

Scale individual system components based on measured bottlenecks.

Potential scaling areas:

```text id="e3f4g5"
API workers
background workers
PostgreSQL
cache
storage
search
messaging
notifications
CDN
```

Scaling must be evidence-driven.

The platform must not assume that increasing one provider tier automatically solves every bottleneck.

---

# PHASE 20 — Advanced Infrastructure + Public API + Integrations

## Objective

Introduce advanced capabilities after the core platform is stable.

Potential areas:

```text id="h6i7j8"
public API
API keys
webhooks
third-party integrations
calendar integrations
AI receptionist
advanced messaging
advanced search
Redis/Asynq
PostGIS
advanced observability
developer platform
```

The AI receptionist must interact through controlled application APIs.

It must not bypass booking, authorization, payment, or tenant boundaries.

---

# Cross-Phase Requirements

The following requirements apply throughout the roadmap.

## Responsive Design

All customer, business, and platform interfaces must remain responsive.

Required testing targets include:

```text id="k9l0m1"
320px
375px
390px
414px
768px
1024px
1280px
1440px
1920px
```

---

## Security

Security must be considered during every phase.

The project must continuously address:

```text id="n2o3p4"
authentication
authorization
tenant isolation
input validation
secrets
logging
dependency safety
abuse prevention
data exposure
```

---

## Testing

Every phase must include appropriate:

```text id="q5r6s7"
unit tests
integration tests
security tests
build verification
static checks
```

The exact testing mix depends on the phase.

---

## Documentation

When a full phase ends, update the project documentation to reflect the new architecture and implementation state.

At minimum:

```text id="t8u9v0"
CURRENT_STATE.md
ROADMAP.md
relevant architecture documentation
```

`TECHNICAL_TERMS.md` should be updated when a full phase ends and new terminology has become part of the project's established vocabulary.

---

# Phase Completion Rule

A phase is not complete merely because the main feature works.

The phase must also satisfy:

```text id="x1y2z3"
implementation
tests
security verification
lint/static checks
build verification
documentation
manual configuration
known issues
final review
```

Only then should the phase be marked complete.

User instruction (2026-10-13): every full phase must also include at least a simple,
working UI to test its functionality. Record the access path and testing steps.
Phase 5 must include a booking test UI before completion, ahead of Phase 6 dashboards.

---

# Current Development Position

As of 2026-10-13, Phase 5 core-booking development scope is complete. Phase 5.7 adds
booking read APIs, occupancy-aware advisory slots and the required simple test UI at
http://localhost:3000/bookings. BOOKING_WORKSPACE.md records access, test steps,
contracts, limitations and verification. No new migration is needed (schema 14).

Full database-enabled Go tests/build/vet and frontend lint/build passed. An isolated
headless Chrome flow verified login/demo setup/booking/rescheduling/cancellation and
history; desktop/mobile screenshots were inspected. Earlier release checks remain in
DEFERRED_WORK.md; this is not unconditional Phase 3 closure or production sign-off.

Next: Phase 6.1 business workspace and catalog-management UI. Full Phase 6 must also
include a usable UI and testing instructions under the recorded phase-end rule.

## Phase 6 progress - 2026-10-13

Phase 6.1 completed: /business provides business creation, paginated catalog lists,
catalog creation and eligibility assignment management with current tenant permissions.
See BUSINESS_WORKSPACE.md for testing, verification and limitations.
Next: Phase 6.2 scheduling configuration UI (hours, closures, service rules and
staff/resource availability). Phase 6 remains in progress; its full dashboard scope
is not complete. This entry supersedes the earlier "Next: Phase 6.1" position above.

## Phase 6.2 progress and product clarification - 2026-10-13

Phase 6.2 completed: scheduling forms on /business for hours, closed dates, service
buffers/notice and dated staff/resource windows. See BUSINESS_SCHEDULING_UI.md.
Next is Phase 6.3 business profile and catalog editing; this supersedes earlier next-step
entries. Booking management/completion, calendar/dashboard and remaining settings
must still be delivered before claiming full Phase 6 completion.

User clarification: customers create and manage their own bookings. Businesses manage
only bookings received by their own business, through business-facing pages, and can
mark delivered bookings completed. Both sides share current booking state, so changes
must appear in the other view. Customer deletion means cancellation with retained
history. Customer identity/ownership must be enforced in the API before Phase 7 ships.
The current /bookings screen is an internal operations/test workspace, not customer
self-service. Business configuration will determine the offerings and availability
shown to customers. Booking references, identities and timestamps should be assigned
by the system; automatic available staff/resource allocation is separate future work.
Customer-facing screens, publishing/discovery and ownership are Phase 7. Notifications
and live push updates are not implied by sharing the same persisted booking record.

## Phase 6.3 progress - 2026-10-13

Phase 6.3 completed: edit existing business name/timezone and catalog fields at
/business, with tenant-protected PUT APIs and booked-resource capacity safeguards.
See BUSINESS_EDITING.md. This supersedes earlier next-step entries.
Next: Phase 6.4 business-scoped booking management and completion status. Calendar,
dashboard, remaining settings/customization and full-phase review remain before
Phase 6 closure. Customer ownership and self-service continue to be Phase 7 work.

## Phase 6.4 progress - 2026-10-14

Business booking management is available at /business/bookings with manual completion
for delivered confirmed bookings after cleanup ends. Migration 15 preserves history
and audit/idempotency semantics. See BUSINESS_BOOKING_MANAGEMENT.md.
Next: Phase 6.5 calendar and dashboard; this supersedes earlier next-step entries.
Remaining settings/customization and full-phase review remain before Phase 6 closure.
Customer self-service remains Phase 7, with ownership enforced server-side before launch.

## Phase 6.5 progress - 2026-10-14

/business/calendar provides a business-local date agenda, chronological pagination
and complete period counts by booking status. See BUSINESS_CALENDAR.md. No new
migration; schema remains 15. Tests include tenant access, multi-page counts and DST.
Next: Phase 6.6 business customization/settings, superseding earlier next-step entries.
User instruction: prioritize functional UI/testing now; beginner-friendly redesign and
polish will be addressed after the whole platform's functional implementation.
This does not waive functional UI or responsive verification at each phase boundary.

## Phase 6.6 progress - 2026-10-14

Business customization now stores description/contact details with a saved preview.
Owner-only team membership controls reuse existing APIs; /account shows the user's ID.
See BUSINESS_SETTINGS.md. Migration 16 is clean; functional/API/browser checks passed.
Next: Phase 6.7 full-phase completion review and gap closure, superseding earlier
next-step entries. No public publishing/customer ownership implementation is claimed;
Phase 7 remains separate. UI polish remains deferred until platform implementation.

## Phase 6.7 closure - 2026-10-14

Phase 6 development scope is complete. PHASE_6_CLOSURE.md maps every Phase 6 area
to its functional UI, records fresh tests and preserves release limitations.
Next: Phase 7.1 customer booking ownership and business publishing boundaries,
superseding earlier next-step entries. Customer self-service must enforce persisted
ownership before exposure; existing internal bookings are not implicitly customer-owned.
No Phase 7 implementation or production approval is included in this closure.

## Phase 7.1 foundation - 2026-10-14

Customer ownership is persisted separately from created_by; old internal bookings
remain unassigned. A protected customer item-read API and private business publication
state are implemented, with database constraints and cross-customer/tenant tests.
See CUSTOMER_FOUNDATION.md. Schema 17 is clean. This does not expose public contacts
or implement customer booking creation/list/manage UI.
Next: Phase 7.2 owner/admin publishing review UI and opt-in public business profiles.
A public route was omitted after automatic approval review flagged contact exposure
without a concrete review flow. Fresh explicit publishing confirmation is required
before exposing data; do not infer it from foundation state alone.
This supersedes previous next-step entries; Phase 7 remains in progress.

## Phase 7.2 preparation - 2026-10-14

Private publishing review and snapshot infrastructure passed backend/frontend/browser
checks; migration 18 is clean. Public page/API and publish/unpublish UI are blocked
pending explicit user approval following automatic approval review. Phase 7.2 remains
incomplete; see PUBLICATION_REVIEW.md for the exact proposed payload and destination.
No public business endpoint is exposed. This supersedes earlier next-step entries.

## Phase 7.2 completed - 2026-10-16

Reviewed opt-in public profiles are implemented after the user's instruction to proceed.
Owner/admin confirmation publishes a snapshot; draft edits require republishing.
Public page /businesses/{id} and anonymous API enforce active/confirmed/published state.
Migration 19 keeps old prepared snapshots private until fresh confirmation.
See PUBLICATION_REVIEW.md for tests and access. Earlier approval blocks are resolved.
Next: Phase 7.3 public discovery and customer-facing service selection, superseding
previous next-step entries. Full Phase 7 and customer booking actions remain pending.

## Phase 7.3 completed - 2026-10-16

Public business search/pagination and reviewed service selection are available at
/businesses. Existing profiles need explicit republishing with services to appear in
the directory. Reviewed service edits remain private; archived services are hidden.
See PUBLIC_DISCOVERY.md for test evidence, limits and access. Schema remains 19.
Next: Phase 7.4 customer-facing availability and location/eligibility resolution for
selected published services. This supersedes earlier next-step entries. Selection is
not booking creation; customer ownership/lifecycle implementation continues separately.

## Phase 7.4 update (2026-10-16)

Public service/location/date availability and responsive customer UI are complete.
Publication previews now include linked location IDs, names and timezones; existing
snapshots require owner/admin review and republishing. Current scheduling and
occupancy filter advisory times. Selection creates no reservation or booking.
See [PUBLIC_AVAILABILITY.md](PUBLIC_AVAILABILITY.md) for limits and passed checks.
Next is Phase 7.5 authenticated customer-owned booking creation.

## Phase 7.5 completed (2026-10-16)

Customers can now confirm a booking from a published business after signing in.
Ownership, assignment and current availability are enforced atomically; idempotent
retries recover the saved booking without duplication. The responsive confirmation
UI includes a booking reference. Schema stays 19. See
[CUSTOMER_BOOKING_CREATION.md](CUSTOMER_BOOKING_CREATION.md) for the contract and tests.
Next: Phase 7.6 customer booking history and detail UI; full Phase 7 remains open.

## Phase 7.6 completed (2026-10-16)

Customer history and detail UI are available at `/my-bookings` and
`/my-bookings/{id}`. Account and booking confirmation links connect the flow.
Pagination and all reads enforce authenticated customer ownership independently
of business publication. See [CUSTOMER_BOOKING_HISTORY.md](CUSTOMER_BOOKING_HISTORY.md).
Next: Phase 7.7 customer cancellation; rescheduling follows. Full Phase 7 remains open.

## Business setup fix (2026-10-17)

Per the user's request, both Publish actions now save pending workspace forms,
fetch a fresh preview and publish using its token. This supersedes the earlier
requirement to manually save every section and refresh the preview first.
Failed saves stop publication; earlier saves may have completed. Missing locations
produce owner guidance before publishing. Custom staff role labels are private
business vocabulary and do not grant account access. Migration 20 is clean.
See [BUSINESS_SETUP_CLARITY.md](BUSINESS_SETUP_CLARITY.md) for scope, checks and limitations.
Phase 7.7 customer cancellation remains next; no cancellation work was delivered here.

# Phase 7.7 completed — 2026-10-18

Customers can cancel their own future bookings from booking details, with explicit
confirmation, retained history, atomic capacity release and safe retries. Existing
operator lifecycle rules and ownership boundaries are enforced. See
[CUSTOMER_BOOKING_CANCELLATION.md](CUSTOMER_BOOKING_CANCELLATION.md).
Next: Phase 7.8 customer rescheduling. The earlier range-booking/setup simplification
work is unfinished and recorded in DEFERRED_WORK.md. Full Phase 7 remains open.


## Phase 7.8 completed (2026-10-18)

Customer-owned future bookings can be rescheduled through their detail page.
A protected availability read and atomic replacement preserve duration, branch,
service, assignments and history. Occupancy and the preparation deadline are
rechecked on confirmation. Retrying a committed move is safe after a lost response.
See CUSTOMER_BOOKING_RESCHEDULING.md for the API, tests, limitations and file list.
Next: Phase 7 completion review and gap closure, including previously deferred
custom range workflow verification. Full Phase 7 and final UI redesign remain open.


## Phase 7.9 completion review (2026-10-18)

Phase 7 customer-booking development scope is complete. The custom-range workflow
passed owner setup, save-before-publish, overnight creation and overlap rejection.
Fresh backend/frontend checks and cancellation/rescheduling browser regressions
passed. PHASE_7_CLOSURE.md maps scope and records limits; account profile is read-only.
No payment, messaging, final redesign or production approval is implied.
Next: Phase 8.1 tenant-scoped CRM foundation and basic customer table/UI.


## Phase 8.1 completed - 2026-10-19

Tenant-scoped read-only customer table, literal name/email search and pagination
are implemented at /business/customers. Customer relationships derive from owned
bookings; no account-wide directory is exposed. Only business owners/admins can
read contact details and business-local counts. See CUSTOMER_CRM.md.
Next: Phase 8.2 filtering and sorting. Manual CRM editing, bulk actions and the
import/export lifecycle remain later work. No migration; schema remains 21.
