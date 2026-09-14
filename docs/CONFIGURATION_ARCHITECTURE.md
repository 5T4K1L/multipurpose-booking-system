# Configuration and Environment Architecture

## Purpose

This document defines how application configuration and environment-specific values are managed.

The goal is to keep configuration secure, predictable, portable, and easy to manage across development, testing, staging, and production environments.

## Configuration Principles

Configuration must:

- remain separate from application code
- avoid hardcoded secrets
- support different environments
- be validated at application startup
- use explicit names
- avoid unnecessary duplication
- never expose secrets to the frontend

## Environments

The application is expected to eventually support:

```text
Development
Testing
Staging
Production
```

Only the development environment is currently being implemented.

Do not create production infrastructure during the current phase unless explicitly authorized.

## Environment Variables

Environment variables are the primary mechanism for environment-specific configuration.

Examples include:

```text
DATABASE_URL
API_BASE_URL
PORT
```

Environment variables containing secrets must never be committed to Git.

## `.env.example`

`.env.example` contains the names and safe example values required to configure the application.

It must:

- contain no real secrets
- document required variables
- remain safe to commit
- be updated when required configuration changes

## Secret Handling

Secrets must not be:

- hardcoded into source code
- committed to Git
- placed in documentation as real credentials
- exposed to browser-side JavaScript
- logged unnecessarily

Examples of secrets include:

- database passwords
- API keys
- signing keys
- access tokens
- webhook secrets
- payment provider secrets

## Backend Configuration

The Go backend reads server-side configuration from environment variables.

Backend configuration may include:

```text
DATABASE_URL
PORT
ENVIRONMENT
```

The backend is responsible for validating required configuration before serving requests.

Invalid or missing required configuration should cause a clear startup error.

## Frontend Configuration

Next.js configuration is divided into:

### Server-only variables

These may contain sensitive values and must remain server-side.

### Public variables

Variables intentionally exposed to browser code must use the appropriate Next.js public environment-variable convention.

Public variables must never contain secrets.

## Database Configuration

The database connection is provided through:

```text
DATABASE_URL
```

Example development format:

```text
postgres://booking_user:booking_password@localhost:5432/booking_db
```

Real credentials must not be committed to source control.

## Local Development

Local development configuration should use environment variables loaded by the local development environment.

The current PostgreSQL development configuration is:

```text
Host: localhost
Port: 5432
Database: booking_db
User: booking_user
```

The development password must not be stored in committed documentation.

## Configuration Validation

Configuration should be validated at startup where practical.

Examples:

```text
Missing DATABASE_URL
Invalid PORT
Unsupported environment
Invalid configuration value
```

The application should fail clearly rather than silently using unsafe defaults.

## Defaults

Safe development defaults may be provided for non-sensitive configuration.

Examples:

```text
PORT=8080
ENVIRONMENT=development
```

Sensitive values must not receive insecure hardcoded production defaults.

## Ports

Current local development ports:

```text
Next.js: 3000
Go API: 8080
PostgreSQL: 5432
```

Port values should remain configurable when practical.

## Configuration Ownership

Frontend configuration belongs to the frontend.

Backend configuration belongs to the backend.

Database configuration belongs to the backend/database environment.

Do not duplicate the same configuration across multiple layers unless the duplication is necessary for a clear interface.

## Configuration Naming

Environment variable names should be:

- uppercase
- descriptive
- consistent
- stable

Example:

```text
DATABASE_URL
API_BASE_URL
ENVIRONMENT
```

Avoid ambiguous names such as:

```text
DB
URL1
CONFIG
VALUE
```

## Logging

Configuration values must be handled carefully in logs.

Never log:

- passwords
- API keys
- tokens
- connection strings containing credentials
- signing secrets

Non-sensitive configuration may be logged when it assists debugging.

## Error Messages

Configuration errors should identify the missing or invalid configuration name without exposing secret values.

Good:

```text
DATABASE_URL is not set
```

Bad:

```text
DATABASE_URL=postgres://booking_user:booking_password@...
```

## Testing Configuration

Tests should use isolated configuration where necessary.

Tests must not depend on a developer's personal machine configuration unless explicitly testing environment integration.

Integration tests may require a test database or controlled development database configuration.

## CI/CD

CI/CD configuration will be defined during the authorized CI/CD phase.

Secrets should eventually be provided by the CI/CD secret-management mechanism rather than committed to the repository.

## Production Configuration

Production configuration is intentionally not implemented during the current phase.

Production secrets must eventually be injected through secure infrastructure or deployment configuration.

They must never be stored directly in the repository.

## Configuration Changes

When adding a new required environment variable:

1. add it to `.env.example`
2. document its purpose
3. validate it at startup when appropriate
4. update affected documentation
5. update tests
6. record important architectural decisions when necessary

## Phase 1 Boundary

This phase defines configuration architecture only.

It does not implement:

- production secret management
- cloud secret managers
- deployment environments
- CI/CD secrets
- external configuration services
- production infrastructure

## Core Principle

Configuration should be explicit, secure, environment-aware, and separate from application logic.

No secret belongs in source control.
