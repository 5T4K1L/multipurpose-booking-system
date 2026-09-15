# Error Handling and Logging Architecture

## Purpose

This document defines the application's error-handling and logging conventions.

The goal is to make failures:

- predictable
- safe
- debuggable
- consistent
- useful to developers and operators

## Core Principles

Errors must be handled explicitly.

The application must:

- return appropriate HTTP status codes
- provide safe client-facing error messages
- preserve useful internal error context
- avoid exposing sensitive implementation details
- log unexpected failures appropriately
- avoid logging secrets
- distinguish expected errors from unexpected failures

## Error Categories

The backend should distinguish between the following categories.

### Validation Errors

The request contains invalid or incomplete input.

Examples:

- missing required field
- invalid format
- invalid value
- invalid query parameter

Typical HTTP status:

```text
400 Bad Request
```

### Not Found Errors

The requested resource does not exist or is not available to the caller.

Typical HTTP status:

```text
404 Not Found
```

### Conflict Errors

The requested operation conflicts with the current state of the resource.

Examples:

- duplicate resource
- conflicting booking
- invalid state transition

Typical HTTP status:

```text
409 Conflict
```

### Unauthorized Errors

The request requires authentication that is missing or invalid.

Typical HTTP status:

```text
401 Unauthorized
```

Authentication is not implemented yet.

### Forbidden Errors

The authenticated caller does not have permission to perform the requested operation.

Typical HTTP status:

```text
403 Forbidden
```

Authorization is not implemented yet.

### Internal Errors

An unexpected server-side failure occurred.

Typical HTTP status:

```text
500 Internal Server Error
```

Internal errors must not expose implementation details to clients.

## API Error Format

API errors should use a consistent JSON structure:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "The request contains invalid data."
  }
}
```

Field-specific validation information may be included:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "The request contains invalid data.",
    "fields": {
      "email": "must be a valid email address"
    }
  }
}
```

## Error Codes

Error codes should be:

- stable
- descriptive
- machine-readable
- independent of human-readable messages

Examples:

```text
VALIDATION_ERROR
NOT_FOUND
CONFLICT
UNAUTHORIZED
FORBIDDEN
INTERNAL_ERROR
```

Avoid exposing internal Go error names as API error codes.

## Internal Error Wrapping

Go errors should preserve useful context through error wrapping.

Example pattern:

```go
fmt.Errorf("load customer: %w", err)
```

The wrapped error should remain available for internal diagnosis.

Do not expose the complete wrapped error directly to the client.

## Error Translation

The API layer is responsible for translating application errors into HTTP responses.

Intended flow:

```text
Repository Error
      ↓
Service Error
      ↓
HTTP Error Mapping
      ↓
Safe JSON Response
```

Business logic should not construct HTTP responses directly.

## Logging Principles

Logs should help answer:

- what happened?
- when did it happen?
- where did it happen?
- what operation was being performed?
- did it succeed or fail?

Logs should remain concise and structured enough to be useful later.

## Log Levels

The application should conceptually support:

```text
DEBUG
INFO
WARN
ERROR
```

### DEBUG

Detailed information useful during development.

Do not rely on DEBUG logging for essential production diagnostics.

### INFO

Normal important application events.

Examples:

- server started
- configuration loaded
- background process started

### WARN

Unexpected conditions that do not necessarily stop an operation.

### ERROR

Unexpected failures requiring investigation.

## Sensitive Data

Never log:

- passwords
- API keys
- access tokens
- refresh tokens
- session secrets
- database passwords
- payment credentials
- webhook secrets
- authentication headers
- complete database connection strings containing credentials

## Personal Data

Sensitive customer information should not be logged unnecessarily.

Avoid logging complete:

- addresses
- phone numbers
- private messages
- payment details
- personal notes

Use stable internal IDs when possible.

## Request Logging

The API should eventually record useful request metadata such as:

- HTTP method
- route
- status code
- duration
- request ID

Do not log complete request bodies by default.

Request-body logging should only be introduced for carefully controlled debugging scenarios.

## Request IDs

The backend should eventually assign or propagate a request/correlation ID.

This will make it possible to connect:

```text
HTTP request
    ↓
application operation
    ↓
database operation
    ↓
background job
```

The exact implementation will be established during observability work.

## Startup Logging

The API may log safe startup information such as:

```text
Server is running on http://localhost:8080
PostgreSQL connection verified
```

Startup logs must not expose credentials.

## Database Errors

Database errors should be wrapped with useful internal context.

Example:

```text
load customer
create booking
update payment
```

The API response must not directly expose raw PostgreSQL errors.

## Panic Handling

Unexpected panics must not terminate the entire server unnecessarily.

A recovery mechanism may eventually be introduced at the HTTP boundary.

The exact implementation should be added when the API architecture requires it.

## Client Error Messages

Client-facing messages should be:

- understandable
- concise
- safe
- actionable when practical

Do not return messages such as:

```text
pq: duplicate key value violates unique constraint ...
```

Prefer:

```text
A resource with that value already exists.
```

## Logging and Privacy

Logging must follow data-minimization principles.

Only collect information required for:

- debugging
- operational visibility
- security investigation
- reliability analysis

Do not collect sensitive information simply because it may be useful someday.

## Development vs Production

Development may use more verbose logs.

Production should prioritize:

- useful operational information
- structured logs
- security
- privacy
- low noise

Production logging configuration will be defined during the observability phase.

## Frontend Errors

The frontend should display safe, user-friendly error states.

It must not expose:

- backend stack traces
- SQL errors
- internal server paths
- secrets
- infrastructure details

Frontend error handling must treat the backend as the authoritative source for business errors.

## Error Handling in Services

Services should return meaningful application-level errors.

Services should not depend on HTTP response types.

Example conceptual flow:

```text
Service
    ↓
ErrNotFound
    ↓
HTTP layer
    ↓
404 Not Found
```

## Error Handling in Repositories

Repositories should return errors describing persistence failures.

They should not convert database errors into HTTP responses.

## Testing Error Behavior

Important error paths should be tested.

Tests should verify:

- correct status code
- correct error code
- safe response body
- expected behavior for invalid input
- expected behavior for missing resources
- expected behavior for conflicts

## Security Requirement

Error handling must never become an information-disclosure mechanism.

Unexpected internal errors should produce a generic client response while preserving useful internal diagnostics.

## Phase 1 Boundary

This phase defines error and logging architecture only.

It does not implement:

- centralized production logging
- distributed tracing
- external log aggregation
- Sentry
- OpenTelemetry
- advanced observability infrastructure

Those belong to later authorized phases.

## Core Principle

Clients receive safe and consistent errors.

Developers and operators receive enough internal information to diagnose problems.

Sensitive information remains protected.

Add that OAuth errors must not expose authorization codes, tokens, private keys, client secrets, or sensitive provider response details.
