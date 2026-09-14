# Package Architecture

## Purpose

This document defines the package boundaries and dependency rules for the Go backend.

The goal is to keep the modular monolith internally organized, testable, and maintainable as the platform grows.

## Core Principle

Dependencies should have a clear direction.

Higher-level application behavior may depend on lower-level infrastructure abstractions, but lower-level infrastructure must not depend on HTTP handlers or unrelated application features.

## Initial Package Direction

The intended dependency direction is:

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

Domain types and rules should remain as independent as practical.

A more complete conceptual structure is:

```text
                 ┌──────────────┐
                 │   cmd/api    │
                 └──────┬───────┘
                        ↓
                 ┌──────────────┐
                 │    server    │
                 └──────┬───────┘
                        ↓
                 ┌──────────────┐
                 │   service    │
                 └──────┬───────┘
                        ↓
                 ┌──────────────┐
                 │ repository   │
                 └──────┬───────┘
                        ↓
                 ┌──────────────┐
                 │  database    │
                 └──────────────┘

                 ┌──────────────┐
                 │    domain    │
                 └──────────────┘
```

## `cmd`

The `cmd` package contains application entry points.

Responsibilities:

- application startup
- configuration loading
- dependency wiring
- server startup
- graceful shutdown

`cmd` should not contain business logic.

Example:

```text
cmd/api/main.go
```

## `server`

The `server` package contains HTTP transport concerns.

Responsibilities:

- route registration
- handlers
- HTTP request parsing
- HTTP response formatting
- middleware
- HTTP-specific error mapping

The server package must not contain:

- SQL queries
- database implementation details
- major business rules

## `service`

The `service` package contains application use cases.

Responsibilities:

- application workflows
- business-rule orchestration
- transaction coordination
- interaction between domain logic and repositories

Services must not depend on HTTP request or response objects.

Services should remain usable from other application entry points when appropriate.

## `repository`

The repository layer contains persistence operations.

Responsibilities:

- querying PostgreSQL
- inserting records
- updating records
- deleting records
- mapping database results
- persistence-specific behavior

Repositories must not:

- create HTTP responses
- depend on handlers
- contain unrelated business workflows

## `database`

The database package contains infrastructure for PostgreSQL connectivity.

Responsibilities:

- connection pools
- transaction primitives
- database configuration
- migration execution
- database lifecycle management

The database package should not depend on:

- HTTP handlers
- frontend code
- business-specific services

## `domain`

The domain layer contains core business concepts and rules where appropriate.

Responsibilities:

- domain entities
- value objects
- domain-level invariants
- domain-specific errors
- pure business rules

Domain packages should avoid depending on infrastructure implementations.

Where practical:

```text
domain
   ↓
nothing infrastructure-specific
```

This keeps domain logic testable and reusable.

## `validation`

Validation should be introduced as a package only when shared validation rules justify it.

Validation may include:

- request validation
- reusable field validation
- business-independent format validation

Business rules that require application state belong in the service/application layer rather than generic validation helpers.

## `types`

A shared types package should only be introduced when shared types genuinely need to cross package boundaries.

Avoid creating a giant `types` package containing unrelated models.

Types should have clear ownership.

## `config`

Configuration loading and validation may be centralized into a configuration package when the number of environment settings justifies it.

Configuration should not contain application business logic.

## Package Dependency Rules

The following dependencies are allowed:

```text
cmd
  → server
  → service
  → repository
  → database

server
  → service
  → validation
  → shared types where justified

service
  → domain
  → repository interfaces
  → validation where appropriate

repository
  → database
  → domain
  → generated database code where applicable
```

## Dependency Rules

The following patterns are prohibited unless explicitly justified:

```text
server → PostgreSQL directly
server → raw SQL
domain → PostgreSQL
domain → HTTP
repository → HTTP handlers
repository → frontend
service → HTTP response writers
database → service
```

## Interfaces

Interfaces should be introduced where they provide a real architectural benefit.

Good reasons include:

- isolating a persistence dependency for testing
- defining an application boundary
- supporting multiple implementations where actually required

Do not create interfaces for every struct or function automatically.

## Interface Ownership

Interfaces should generally be owned by the package that consumes the behavior rather than the package that implements it.

This allows the consumer to define the smallest interface it actually needs.

## Dependency Injection

Dependencies should be passed explicitly where practical.

Avoid global mutable dependencies.

Example conceptual flow:

```text
main
  ↓
construct repository
  ↓
construct service
  ↓
construct handler
  ↓
register routes
```

## Global State

Avoid global mutable application state.

Global constants and immutable configuration values may be appropriate.

Connections, repositories, services, and request-specific state should generally be explicitly provided.

## Package Size

Packages should have a focused purpose.

Avoid packages such as:

```text
utils
helpers
common
misc
```

when they become dumping grounds for unrelated functionality.

## Utility Code

Small utility functions are acceptable when they have a clear owner.

Before creating a utility package, determine whether the functionality belongs naturally to an existing package.

## Feature Modules

As the product grows, business capabilities may use feature-oriented package organization.

For example:

```text
internal/
├── bookings/
├── customers/
├── businesses/
└── messaging/
```

Each feature may contain its own:

- domain logic
- service logic
- repository logic
- handlers

The exact organization should be determined when the corresponding feature is authorized.

## Cross-Feature Dependencies

Feature modules should avoid unnecessary direct dependencies on unrelated feature modules.

When communication between features is required, prefer:

- explicit application services
- domain events
- well-defined interfaces

Do not create tightly coupled chains between modules.

## Circular Dependencies

Go package cycles are prohibited by the language.

The architecture should prevent cycles before they appear.

If two packages need each other, reconsider:

- ownership
- interface placement
- shared domain concepts
- package boundaries

Do not solve cycles by creating arbitrary shared packages.

## Generated Code

Generated code should remain in clearly identifiable packages.

Manually editing generated files is prohibited.

Source definitions and generation commands remain the source of truth.

## Testing

Package boundaries should make testing easier.

Examples:

```text
domain
  → unit tests

service
  → unit tests with controlled dependencies

repository
  → integration tests

server
  → HTTP handler tests

cmd
  → minimal startup/integration verification
```

## Dependency Changes

Before adding a third-party dependency:

1. identify the current requirement
2. determine whether the standard library is sufficient
3. evaluate maintenance and security
4. consider dependency size and complexity
5. document major architectural choices when necessary

Do not add dependencies for hypothetical future features.

## Free-Tier Development Principle

The package architecture must not depend on paid development agents, premium tooling, or proprietary development environments.

The project should remain buildable and testable using the local development stack.

## Phase 1 Boundary

This document defines package boundaries only.

It does not implement:

- feature modules
- authentication packages
- booking packages
- customer packages
- payment packages
- messaging packages
- background-job packages

Those belong to their authorized phases.

## Core Principle

Every package should have a clear reason to exist.

Prefer:

```text
Small responsibility
+
Clear ownership
+
One-way dependencies
+
Explicit interfaces
```

over excessive abstraction.
