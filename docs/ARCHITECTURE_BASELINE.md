# Architecture Baseline

## Purpose

This document consolidates the architectural decisions established during Phase 1.

It serves as the high-level architectural reference for future implementation work.

Detailed rules remain in their respective architecture documents.

## Architecture Style

The platform uses a modular monolith.

The application remains a single deployable system while maintaining clear internal boundaries between:

- HTTP transport
- application services
- domain logic
- repositories
- database infrastructure

Microservices are not part of the current architecture.

## System Overview

```text
┌─────────────────────────────┐
│         Next.js Web         │
│       React + TypeScript    │
└──────────────┬──────────────┘
               │ HTTP
               ▼
┌─────────────────────────────┐
│          Go API             │
│                             │
│  HTTP → Service → Repo      │
│             ↓               │
│          Domain             │
└──────────────┬──────────────┘
               │
               ▼
┌─────────────────────────────┐
│         PostgreSQL          │
└─────────────────────────────┘
```

## Backend Dependency Direction

The preferred dependency direction is:

```text
cmd/api
   ↓
server
   ↓
service
   ↓
repository
   ↓
database
```

Domain concepts remain as independent as practical and should not depend on infrastructure-specific implementations.

## Frontend Boundary

The frontend communicates with the backend through HTTP.

```text
Next.js
   ↓
HTTP API
   ↓
Go
   ↓
PostgreSQL
```

The browser must never connect directly to PostgreSQL.

## API Architecture

The API is a JSON HTTP API.

The intended application namespace is:

```text
/api/v1
```

API conventions include:

- standard HTTP methods
- predictable resource naming
- consistent JSON responses
- consistent error responses
- explicit validation
- server-side authorization
- bounded collection responses
- safe error messages

## Configuration Architecture

Configuration comes from environment-specific configuration.

Rules:

- secrets are never committed
- `.env.example` contains safe templates
- required configuration should be validated
- frontend public variables must never contain secrets
- sensitive configuration remains server-side

Current local development services:

```text
Next.js      3000
Go API       8080
PostgreSQL   5432
```

## Database Architecture

PostgreSQL is the primary source of truth.

Database design prioritizes:

- relational integrity
- foreign keys
- unique constraints
- appropriate indexes
- transactions
- explicit SQL
- version-controlled migrations

The project intends to use:

```text
pgx
sqlc
SQL migrations
```

## Error Architecture

Errors move through the system as:

```text
Database / Infrastructure Error
            ↓
       Repository
            ↓
         Service
            ↓
      HTTP Error Mapping
            ↓
       Safe JSON Response
```

Clients receive safe, stable error codes.

Internal implementation details remain internal.

## Logging Architecture

Logs should provide useful operational information without exposing:

- passwords
- API keys
- tokens
- database credentials
- unnecessary personal information
- sensitive request contents

Future observability infrastructure will expand this model.

## Security Baseline

Security is enforced primarily on the server.

Required principles include:

- never trust client input
- validate external input
- authorize server-side
- use least privilege
- use parameterized SQL
- protect secrets
- avoid information disclosure
- use secure defaults
- minimize sensitive data

Authentication, authorization, and tenant isolation will be implemented in their authorized phases.

## Application Lifecycle

The application lifecycle follows:

```text
Start
  ↓
Load configuration
  ↓
Initialize dependencies
  ↓
Connect database
  ↓
Construct repositories
  ↓
Construct services
  ↓
Construct handlers
  ↓
Start HTTP server
  ↓
Serve requests
  ↓
Graceful shutdown
  ↓
Close resources
  ↓
Exit
```

Dependencies are explicitly constructed and owned by the application entry point.

Global mutable service state is avoided.

## Testing Architecture

Testing is aligned with system boundaries.

```text
Domain
  → unit tests

Service
  → unit tests

Repository
  → database/integration tests

HTTP
  → handler/API tests

Application
  → startup/integration verification
```

The project should prefer the smallest test that reliably verifies the behavior.

## Responsive Frontend Requirement

All user-facing features must be responsive from the beginning.

Target widths include:

```text
320px
375px
390px
414px
768px
1024px
1280px
1440px
1920px+
```

Responsive implementation must consider:

- touch
- accessibility
- performance
- mobile usability
- desktop usability

## Feature Development Model

Future product features should be implemented as vertical slices.

A meaningful feature should, where appropriate, include:

```text
Database
   ↓
Backend
   ↓
API
   ↓
Frontend
   ↓
Validation
   ↓
Tests
   ↓
Documentation
```

Do not build large disconnected subsystems merely because they will be needed later.

## Architectural Change Rule

A major architectural change requires:

1. identify the decision
2. document the alternatives
3. evaluate trade-offs
4. choose the preferred approach
5. record the decision
6. update affected documentation

Do not silently introduce major architectural changes.

## Current Scope

The architecture is being established for the Universal Booking Platform.

The following are intentionally not implemented during Phase 1:

- authentication
- registration
- login
- RBAC
- multi-tenancy
- businesses
- customers
- services
- staff
- resources
- availability
- bookings
- CRM
- messaging
- notifications
- search
- marketplace
- payments
- billing
- subscriptions
- public API integrations
- production infrastructure

## Source of Truth

Architecture documentation is part of the project's persistent development context.

Important documents include:

```text
PROJECT_CONTEXT.md
CURRENT_STATE.md
ARCHITECTURE.md
BACKEND_ARCHITECTURE.md
API_ARCHITECTURE.md
FRONTEND_ARCHITECTURE.md
CONFIGURATION_ARCHITECTURE.md
ERROR_HANDLING_AND_LOGGING.md
SECURITY_ARCHITECTURE.md
DATABASE_ARCHITECTURE.md
PACKAGE_ARCHITECTURE.md
APPLICATION_LIFECYCLE.md
DECISIONS.md
ROADMAP.md
```

Future development should consult these documents before making architectural changes.

## Phase 1 Completion Principle

Phase 1 is complete when the architecture is sufficiently defined to begin implementing the next authorized system capability without repeatedly redesigning the foundation.

The architecture should provide structure without unnecessary complexity.

Only update if this file is your baseline/source-of-truth architecture document. If it is intended to remain a historical baseline, do not modify it.
