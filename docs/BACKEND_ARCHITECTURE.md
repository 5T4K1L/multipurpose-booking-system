# Backend Architecture

## Purpose

This document defines the internal architecture and dependency boundaries of the Go backend.

The goal is to keep the backend modular, testable, maintainable, and suitable for the long-term Universal Booking Platform.

## Architectural Style

The backend uses a modular monolith architecture.

The system remains one deployable application while maintaining clear internal module boundaries.

We do not introduce microservices unless future scale or operational requirements justify doing so.

## Dependency Direction

The intended dependency direction is:

```text
HTTP / API
    ↓
Application / Service
    ↓
Repository
    ↓
PostgreSQL
```

Domain concepts should remain independent from HTTP and database implementation details.

Dependencies should point inward toward stable application and domain concepts.

## Main Layers

### HTTP / API Layer

Responsibilities:

- HTTP routing
- request parsing
- response formatting
- HTTP status codes
- transport-level validation
- API error mapping
- authentication context handling when authentication is introduced

The HTTP layer must not contain business rules or direct SQL queries.

### Service Layer

Responsibilities:

- application use cases
- business rules
- orchestration
- transaction coordination
- validation that depends on application rules
- coordination between repositories and domain logic

The service layer should not depend directly on HTTP request or response types.

### Repository Layer

Responsibilities:

- database access
- SQL execution
- persistence operations
- mapping database results into application/domain structures

Repositories must not contain HTTP concerns.

### Domain Layer

Responsibilities:

- core business concepts
- domain models
- domain rules that belong to the business itself

Domain code should avoid dependencies on transport-specific or infrastructure-specific implementation details.

### Database Layer

Responsibilities:

- PostgreSQL connection management
- connection pooling
- transaction support
- migration execution
- database-specific configuration

Database infrastructure should remain isolated from HTTP concerns.

## Current Phase 1 Boundary

Phase 1 defines architecture only.

The following are not being implemented yet:

- authentication
- authorization
- multi-tenancy
- organizations
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
- payments
- subscriptions
- marketplace
- search
- production infrastructure

## Testing Boundaries

Each layer should be testable independently where practical.

Expected direction:

```text
HTTP tests
    ↓
Service tests
    ↓
Repository/database tests
```

Tests should avoid requiring unnecessary infrastructure when a unit test is sufficient.

Integration tests may use PostgreSQL when database behavior itself must be verified.

## Error Handling

Errors should be handled explicitly.

The backend should distinguish between:

- validation errors
- business rule errors
- not-found conditions
- conflict conditions
- authentication/authorization errors when implemented
- infrastructure failures
- unexpected internal errors

Internal implementation details must not be exposed unnecessarily through API responses.

## Configuration

Configuration should come from environment variables or environment-specific configuration.

Secrets must never be hardcoded into source code.

Sensitive configuration must not be committed to Git.

## Database Access

Application code must not construct arbitrary SQL throughout handlers.

Database access should be centralized through repositories or the appropriate database abstraction.

SQL migrations remain version-controlled.

## Transactions

Transactions should be created at the application/service boundary when multiple database operations must succeed or fail together.

Repositories may participate in an existing transaction.

## Modularity

Modules should be organized around business capabilities as the product grows.

Avoid creating abstractions solely for hypothetical future requirements.

Prefer simple implementations that preserve clear boundaries.

## API Layer Rules

Handlers should remain small.

A typical request flow should be:

```text
HTTP Request
    ↓
Handler
    ↓
Service
    ↓
Repository
    ↓
PostgreSQL
    ↓
Response
```

Handlers should delegate application behavior instead of implementing business logic directly.

## Frontend Boundary

The frontend communicates with the backend through defined HTTP APIs.

Frontend code must not access PostgreSQL directly.

```text
Next.js
   ↓
HTTP API
   ↓
Go
   ↓
PostgreSQL
```

## Future Authentication Boundary

Authentication and authorization are intentionally deferred to their authorized phases.

When implemented, authentication context should enter through the HTTP/API boundary and be passed into application logic without coupling business logic directly to HTTP-specific mechanisms.

## Future Background Jobs

Background jobs are intentionally deferred.

When introduced, background work should call application/service logic rather than duplicating business rules.

## Architectural Rule

No layer should bypass the intended dependency direction merely for convenience.

Exceptions require an explicit architectural decision documented in `DECISIONS.md`.

## Phase 1 Principle

Architecture should provide enough structure to support the current product requirements without creating unnecessary complexity for hypothetical future requirements.
