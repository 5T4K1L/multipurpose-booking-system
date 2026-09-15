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

PHASE 0 — Engineering Foundation

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

Phase 0 — Engineering Foundation is substantially implemented.

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
