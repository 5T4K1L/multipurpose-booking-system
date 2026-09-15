# Application Lifecycle and Dependency Wiring

## Purpose

This document defines how the Go API application starts, constructs dependencies, runs, and shuts down.

The goal is to keep application startup predictable, avoid global state, and ensure resources are properly managed.

## Application Lifecycle

The intended lifecycle is:

```text
Process starts
    ↓
Load configuration
    ↓
Initialize application dependencies
    ↓
Initialize database
    ↓
Initialize repositories
    ↓
Initialize services
    ↓
Initialize HTTP server
    ↓
Register routes
    ↓
Start serving requests
    ↓
Receive shutdown signal
    ↓
Stop accepting new requests
    ↓
Close application resources
    ↓
Process exits
```

## Entry Point

The application entry point is:

```text
apps/api/cmd/api/main.go
```

The entry point is responsible for application composition and lifecycle management.

It should not contain business logic.

## Configuration Initialization

Configuration should be loaded before application dependencies are constructed.

The application should validate required configuration during startup.

If required configuration is invalid or missing:

```text
startup fails
```

The application must not start in a partially configured state.

## Dependency Construction

Dependencies should be created explicitly during startup.

Conceptually:

```text
Configuration
    ↓
Database
    ↓
Repositories
    ↓
Services
    ↓
HTTP Handlers
    ↓
HTTP Server
```

Each dependency receives the dependencies it requires.

## Dependency Ownership

The application entry point owns top-level dependencies.

For example:

```text
main
 ├── database
 ├── repositories
 ├── services
 ├── handlers
 └── HTTP server
```

The entry point is responsible for closing long-lived resources.

## Database Lifecycle

The PostgreSQL connection pool should be created during startup.

The application should verify that the database is reachable before serving requests when the database is required for normal operation.

On shutdown:

```text
HTTP server stops
    ↓
database pool closes
```

Database resources must not be leaked.

## HTTP Server Lifecycle

The HTTP server should:

1. be constructed during startup
2. register all required routes
3. begin listening
4. stop accepting new requests during shutdown
5. allow in-flight requests appropriate time to complete
6. terminate cleanly

## Graceful Shutdown

The API should eventually support graceful shutdown using operating-system termination signals.

The preferred behavior is:

```text
Shutdown signal
    ↓
Stop accepting new requests
    ↓
Allow active requests to finish
    ↓
Close resources
    ↓
Exit
```

The exact timeout will be defined when graceful shutdown is implemented.

## Context Usage

Contexts should be passed through request-scoped operations where appropriate.

Request contexts should not be replaced with unrelated background contexts deep inside application logic.

Long-running operations should respect cancellation.

## Global State

Avoid global mutable state.

Do not use package-level mutable variables as a substitute for dependency injection.

Shared resources should be created during application startup and passed to the components that require them.

## HTTP Handler Construction

Handlers should receive the service dependencies they need.

Conceptually:

```text
Service
    ↓
Handler
```

Example:

```text
bookingService
    ↓
bookingHandler
```

Handlers should not construct their own database connections or repositories.

## Service Construction

Services should receive their dependencies explicitly.

Conceptually:

```text
Repository
    ↓
Service
```

Services should not create repositories internally.

## Repository Construction

Repositories should receive required database dependencies explicitly.

Conceptually:

```text
Database connection
    ↓
Repository
```

Repositories should not create independent connection pools for every request.

## Database Connection Pool

The application should maintain a controlled connection pool.

Connection-pool configuration should remain environment-aware.

Do not create a new PostgreSQL connection for every HTTP request.

## Startup Failures

The application should fail fast when a required dependency cannot be initialized.

Examples:

```text
invalid configuration
database unavailable
invalid migration state
unable to bind server port
```

Startup failures should provide enough information for diagnosis without exposing secrets.

## Shutdown Failures

Shutdown errors should be handled deliberately.

The application should attempt to close important resources even if another shutdown step fails.

Shutdown should not leak:

- database connections
- network listeners
- background workers
- file handles

## Background Work

Background workers are not implemented in the current phase.

When they are introduced, their lifecycle must follow the application's startup and shutdown process.

Workers must be explicitly started and stopped.

They must not be created as unmanaged goroutines.

## Goroutines

Goroutines should have clear ownership.

Every long-running goroutine should have:

- a reason to exist
- a cancellation mechanism
- a shutdown path
- controlled error handling

Avoid unmanaged goroutines started from arbitrary packages.

## Error Propagation During Startup

Startup errors should propagate back to the application entry point.

The entry point should decide whether the application can safely continue.

Do not silently ignore dependency initialization failures.

## Configuration and Environment

The application lifecycle depends on configuration appropriate to the current environment.

Current environment:

```text
development
```

Production lifecycle behavior will be defined later.

## Testing

Application lifecycle behavior should eventually be tested for important cases such as:

- invalid configuration
- unavailable database
- successful startup
- route registration
- graceful shutdown

Not every internal startup detail requires an isolated test.

## Dependency Wiring Pattern

The preferred pattern is explicit composition:

```text
main
 ├── config
 ├── database
 │
 ├── repositories
 │
 ├── services
 │
 ├── handlers
 │
 └── server
```

This makes dependencies visible and easier to test.

## Avoid Service Locators

Do not introduce a global service locator or dependency container solely to avoid explicit wiring.

Explicit construction is preferred while the application remains manageable.

If the dependency graph becomes genuinely complex, the architecture may be revisited through an explicit decision.

## Avoid Global Singletons

Do not introduce global singleton instances for:

- database connections
- repositories
- services
- handlers
- configuration

unless a strong architectural reason exists and the decision is documented.

## Health and Readiness

The platform will eventually distinguish between:

```text
Liveness
Readiness
```

Liveness answers whether the process is alive.

Readiness answers whether the application is ready to serve normal traffic.

The exact endpoint design will be finalized as deployment and observability requirements become relevant.

## Phase 1 Boundary

This document defines lifecycle and dependency-wiring rules only.

It does not implement:

- graceful shutdown
- background workers
- readiness checks
- distributed coordination
- production process management
- container orchestration

Those belong to later implementation phases.

## Core Principle

Application startup and shutdown should be explicit, predictable, and resource-safe.

Prefer:

```text
Explicit construction
+
Explicit ownership
+
Explicit shutdown
```

over hidden global state or unmanaged background behavior.

Only update this if it already documents authentication request lifecycle and OAuth meaningfully changes that lifecycle. Otherwise leave it alone.
