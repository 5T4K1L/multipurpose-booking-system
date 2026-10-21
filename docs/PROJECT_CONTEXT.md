# Booking Platform — Project Context

## Product

Universal Booking Platform

Vision:

> Book anything, from one place.

The platform is intended to become a universal booking infrastructure and marketplace supporting customers, businesses, staff, resources, scheduling, payments, messaging, CRM, search, and marketplace functionality.

## Development Principles

- Modular monolith architecture.
- Free-first development and infrastructure.
- Security-first engineering.
- Responsive web design from the beginning.
- Mobile-first implementation.
- Backend and frontend are developed together for each feature.
- Features are implemented as small, independently verifiable vertical slices.
- Avoid premature abstraction.
- Do not build future functionality before its authorized phase.
- AI-generated code is untrusted until reviewed and tested.
- PostgreSQL is the primary source of truth for application data.

## Phase-end UI requirement (user instruction, 2026-10-13)

At the end of every full phase, build at least a simple working UI so the user can
test that phase's functionality. Include a usable test flow and access instructions;
API-only completion is insufficient. For Phase 5, provide a booking test UI before
closing the phase. This does not require completing all Phase 6 dashboard features.

## Technology Stack

Frontend:

- Next.js
- React
- TypeScript
- Tailwind CSS
- shadcn/ui

Backend:

- Go
- pgx
- sqlc
- SQL migrations

Database:

- PostgreSQL

Development:

- Docker
- Docker Compose
- Git

## Architecture Direction

The system will begin as a modular monolith.

Future infrastructure may be introduced only when required by the roadmap and justified by actual system needs.

## Responsive Requirement

All user-facing interfaces must work across:

- 320px
- 375px
- 390px
- 414px
- 768px
- 1024px
- 1280px
- 1440px
- 1920px+

Interfaces must consider:

- touch interaction
- accessibility
- performance
- desktop usability
- mobile usability

## Current Authorization

Current authorized phase:

PHASE 5 - Core Booking Engine (development complete; 2026-10-13)

No functionality belonging to later phases may be implemented unless explicitly authorized.

## Current Phase Goal

Create a clean, testable development foundation consisting of:

- Git repository
- project documentation
- Next.js frontend foundation
- Go API foundation
- PostgreSQL
- SQL migration system
- Docker Compose
- API health endpoint
- frontend-to-API connectivity
- basic verification

## Future Development Rule

At the completion of every authorized phase:

1. Verify implementation.
2. Run tests.
3. Run lint/static checks.
4. Run builds.
5. Review security and architecture.
6. Update documentation.
7. Record known issues.
8. Record required manual actions.
9. Stop before the next phase.

## Current Implementation Status

Phase 5 core-booking development is complete with the /bookings test UI. See CURRENT_STATE.md and BOOKING_WORKSPACE.md; Phase 6 is next.

The current verified development stack is:

```text
Next.js
   ↓
Go API
   ↓
PostgreSQL
```

The system currently supports:

- Next.js frontend foundation
- Go API foundation
- PostgreSQL development database
- PostgreSQL connection from Go
- API health endpoint
- Frontend-to-API connectivity
- Basic automated API tests
- Go test verification
- Go vet verification
- Go build verification
- Frontend lint verification
- Frontend production build verification

No product-domain functionality has been implemented yet.

Authentication, authorization, multi-tenancy, businesses, customers, services, staff, resources, availability, bookings, CRM, messaging, notifications, marketplace functionality, payments, subscriptions, billing, search, and production infrastructure remain outside the current authorized scope.

No OAuth implementation details needed unless this file is explicitly your product/project overview.

## UI priority clarification - 2026-10-14

User requests functional testable UI throughout implementation. Beginner-friendly
redesign and visual polish are deferred until the whole platform is functionally
implemented. Continue responsive and workflow verification; do not treat cosmetic
polish as a blocker for Phase 6. Phase 6.5 calendar/dashboard is complete; next is 6.6
business customization/settings. CURRENT_STATE.md and milestone docs track progress.

## Phase 6 development closure (2026-10-14)

Phase 6.7 review is recorded in PHASE_6_CLOSURE.md. Business-facing functional scope
is complete, with remaining production checks preserved in DEFERRED_WORK.md.
Next is Phase 7.1 customer ownership/publishing boundaries; internal operator booking
records must not be inferred to belong to customers. UI redesign remains deferred.

## Phase 7.1 ownership boundary (2026-10-14)

Customer identity is bookings.customer_id, independently of the operator in created_by.
Legacy records remain unassigned. Customer reads require exact ownership; business
reads/writes retain tenant membership authorization. Rescheduling preserves ownership.
Publication state defaults false and remains private until an explicit review/publish
UI and safe public projection are delivered. See CUSTOMER_FOUNDATION.md.


## Current development handoff (2026-10-18)

Phase 7.9 closes the customer-booking development scope; see CURRENT_STATE.md and
PHASE_7_CLOSURE.md for the current status, superseding historical next-step entries.
Next is Phase 8.1 tenant-scoped CRM foundation and its customer table UI. Account
identity/profile is currently read-only. Messaging, payments, final UI redesign
and production release checks remain separate. Read TO_REVISE.MD before phase work
and clear its contents only after its requested fixes have been verified.
