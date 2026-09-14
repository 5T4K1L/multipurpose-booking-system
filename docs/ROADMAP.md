# Roadmap

## PHASE 0 — Engineering Foundation

Current phase.

- Repository foundation
- Documentation foundation
- Next.js foundation
- Go API foundation
- PostgreSQL
- SQL migrations
- Docker Compose
- Health endpoint
- Frontend-to-API connectivity
- Basic tests
- Lint/static checks
- Build verification

## PHASE 1 — System Architecture

- Application boundaries
- API conventions
- error model
- configuration structure
- foundational architectural rules

## PHASE 2 — Database Architecture

- schema conventions
- identifiers
- timestamps
- constraints
- indexes
- transaction conventions

## PHASE 3 — Authentication + Authorization

## PHASE 4 — Multi-Tenancy

## PHASE 5 — Core Booking Engine

## PHASE 6 — Business SaaS + Business Admin

## PHASE 7 — Customer Platform + Customer UI

## PHASE 8 — Customer CRM + Import/Export

## PHASE 9 — Business ↔ Customer Messaging

## PHASE 10 — Background Jobs + Notifications

## PHASE 11 — Search + Marketplace

## PHASE 12 — Payments + Financial Safety / Subscriptions

## PHASE 13 — Security Hardening

## PHASE 14 — Observability + Reliability

## PHASE 15 — Backups + Disaster Recovery

## PHASE 16 — CI/CD

## PHASE 17 — Infrastructure as Code

## PHASE 18 — Production Deployment

## PHASE 19 — Scaling

## PHASE 20 — Advanced Infrastructure + Public API + Integrations

## Phase Rule

Only the currently authorized phase may be implemented.

When a phase is complete, stop and review before proceeding.

## Phase 0 Status

Phase 0 — Engineering Foundation is currently in final verification.

The engineering foundation has been established and the core development flow has been verified:

```text
Next.js
   ↓
Go API
   ↓
PostgreSQL
```

Verified areas include:

- repository foundation
- project documentation
- Next.js frontend foundation
- Go API foundation
- PostgreSQL with Docker Compose
- Go-to-PostgreSQL connectivity
- SQL migration foundation
- API health endpoint
- frontend-to-API connectivity
- basic automated tests
- lint and static analysis
- production build verification

Before Phase 1 begins:

- complete the final Phase 0 review
- verify all Phase 0 completion criteria
- synchronize project documentation
- record known issues
- confirm the repository is clean
- formally close Phase 0

Phase 1 must not begin until Phase 0 is formally completed.

## Phase 1 Status

Phase 1 — System Architecture is complete.

Completed:

- backend architecture
- API conventions
- frontend architecture
- configuration architecture
- error handling and logging architecture
- security baseline
- database conventions
- package dependency architecture
- application lifecycle architecture
- consolidated architecture baseline

Phase 1 exit criteria:

- architecture documented
- dependency boundaries defined
- security boundaries defined
- database conventions defined
- frontend conventions defined
- API conventions defined
- application lifecycle defined
- major architectural decisions recorded
- no future product functionality implemented

Next authorized phase:

PHASE 2 — Database Architecture

## Phase 2 Status

Phase 2 database foundation has been implemented and verified through Step 2.8.

Completed:

- schema strategy
- migration system
- database conventions
- core users table
- sqlc query generation
- repository integration
- database integration testing

The database foundation is ready for formal Phase 2 closure.

The next phase must not begin until Phase 2 is explicitly marked complete.

## Phase 2 Closure

Phase 2 — Database Architecture and Foundation is complete.

Exit criteria completed:

- database strategy documented
- database conventions documented
- migration system implemented
- migration tracking verified
- first domain table implemented
- sqlc query layer implemented
- repository integration implemented
- PostgreSQL integration test implemented
- schema integrity verified
- documentation synchronized

Next authorized phase:

PHASE 3 — Authentication + Authorization
