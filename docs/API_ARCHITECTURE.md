# API Architecture

## Purpose

This document defines the conventions for the Go HTTP API.

All future API endpoints should follow these rules unless an explicit architectural decision changes them.

## API Style

The backend exposes an HTTP JSON API.

Primary format:

```text
Request
    ↓
HTTP Handler
    ↓
Application Service
    ↓
Repository
    ↓
PostgreSQL
```

The API should remain predictable and consistent across the platform.

## Base Path

Application API routes use:

```text
/api
```

Example:

```text
GET /api/health
```

Infrastructure or server-level endpoints may exist outside the application API namespace when justified.

## Versioning

The initial API version is:

```text
/api/v1
```

Example:

```text
GET /api/v1/health
```

Versioning is introduced to make future breaking API changes manageable.

Do not create multiple API versions without a documented reason.

## HTTP Methods

Use standard HTTP methods according to their intended purpose.

```text
GET     Read data
POST    Create a resource or execute an action
PUT     Replace a resource
PATCH   Partially update a resource
DELETE  Remove a resource
```

Do not use `POST` for ordinary updates when another appropriate HTTP method exists.

## Resource Naming

Use plural nouns for resource collections.

Examples:

```text
/api/v1/businesses
/api/v1/customers
/api/v1/bookings
/api/v1/services
```

Nested resources may be used when the relationship is meaningful.

Example:

```text
/api/v1/businesses/{businessID}/services
```

Avoid deeply nested routes.

## Resource IDs

API resources should use stable identifiers.

Identifiers should not rely on array positions, UI indexes, or mutable business values.

## Request Format

JSON is the default request format for API endpoints that accept structured data.

Example:

```json
{
  "name": "Example Service"
}
```

Content type:

```text
application/json
```

## Response Format

Successful responses should use JSON unless the endpoint explicitly requires another representation.

Example:

```json
{
  "id": "resource-id",
  "name": "Example"
}
```

Collection responses should provide a predictable structure.

Example:

```json
{
  "items": [],
  "total": 0
}
```

Pagination conventions will be finalized when the relevant feature is implemented.

## HTTP Status Codes

Use appropriate HTTP status codes.

Common statuses:

```text
200 OK
201 Created
202 Accepted
204 No Content
400 Bad Request
401 Unauthorized
403 Forbidden
404 Not Found
409 Conflict
422 Unprocessable Entity
429 Too Many Requests
500 Internal Server Error
```

Do not return `200 OK` for failed application operations merely to simplify frontend handling.

## Error Responses

API errors should use a consistent JSON structure.

Example:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "The request contains invalid data."
  }
}
```

The `code` should be stable enough for client-side handling.

Human-readable messages may change.

Internal implementation details, stack traces, SQL errors, secrets, and sensitive infrastructure information must not be exposed to clients.

## Validation Errors

Validation failures should identify the affected field when practical.

Example:

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

## Request IDs

The API should eventually support a request or correlation ID for tracing requests across logs and services.

The exact implementation will be defined during the observability and reliability work.

Do not introduce unnecessary tracing infrastructure in the current phase.

## Authentication

Authentication is not implemented in the current phase.

When authentication is introduced, authenticated identity and authorization context should be established before application services execute protected operations.

Do not place authentication logic directly inside individual business handlers.

## Authorization

Authorization is not implemented in the current phase.

When introduced, authorization decisions must be explicit and centrally understandable.

Do not rely on frontend-only authorization.

## Idempotency

Idempotency requirements will apply to operations where duplicate execution could cause harmful or financially significant effects.

Examples include:

- creating bookings
- processing payments
- financial operations
- external side effects

The exact mechanism will be defined before those features are implemented.

Do not add an idempotency system to unrelated Phase 1 endpoints without a current requirement.

## Pagination

Endpoints returning potentially large collections should use pagination.

The pagination contract will be standardized before the first large collection endpoint is implemented.

Do not load unbounded datasets into memory.

## Filtering

Filtering should use explicit query parameters.

Example:

```text
GET /api/v1/bookings?status=confirmed
```

Do not create ad-hoc request formats for each endpoint.

## Sorting

Sorting should use explicit query parameters.

Example:

```text
GET /api/v1/customers?sort=created_at
```

Allowed sortable fields should be controlled by the backend.

Clients must not be able to inject arbitrary SQL expressions.

## Search

Search behavior will be defined when the search and marketplace phase is implemented.

Search must not be confused with unrestricted database querying.

## Date and Time

API date/time values should use a standardized representation.

The backend must preserve timezone information when timezone context matters.

Booking and availability functionality will define detailed time handling during their authorized phases.

## Money

Financial values must not use floating-point representations for financial calculations.

The exact money representation will be defined before payments and billing are implemented.

## Idempotent Reads

GET requests should not create or modify application state.

## API Documentation

API contracts should be documented as the API becomes more substantial.

OpenAPI may be introduced when the API contract becomes complex enough to justify it.

The API implementation and contract must remain synchronized.

## Security Rules

The API must:

- validate incoming input
- enforce authorization server-side when authorization exists
- avoid exposing sensitive information
- avoid trusting client-provided ownership information
- use parameterized database access
- apply appropriate request limits
- return safe error messages

## Database Boundary

Handlers must never access PostgreSQL directly.

The intended flow remains:

```text
HTTP Handler
    ↓
Service
    ↓
Repository
    ↓
PostgreSQL
```

## Frontend Boundary

The frontend communicates with the API through HTTP.

The frontend must not connect directly to PostgreSQL.

## API Stability

Avoid unnecessary breaking changes.

When an API contract must change:

1. determine whether the change is breaking
2. document the decision
3. update affected clients
4. update tests
5. update API documentation

## Phase 1 Rule

API conventions should remain simple until actual product requirements require additional complexity.

Do not implement speculative API infrastructure.
