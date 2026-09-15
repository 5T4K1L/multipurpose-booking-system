# API Architecture

## Purpose

This document defines the HTTP API architecture for the Universal Booking Platform.

The API is the controlled boundary between the frontend and backend application logic.

It must provide:

- predictable resource-oriented endpoints
- authenticated request handling
- authorization enforcement
- validation
- consistent errors
- tenant isolation
- idempotency where required
- safe integration with external providers

---

# API Stack

The API is implemented in Go.

The conceptual request flow is:

```text
Browser
   ↓
Next.js
   ↓
HTTP API
   ↓
Handler
   ↓
Service
   ↓
Repository
   ↓
PostgreSQL
```

External systems are accessed through controlled service/provider boundaries.

The frontend must never access PostgreSQL directly.

---

# API Base Path

The current API versioning convention is:

```text
/api/v1
```

Examples:

```text
/api/v1/auth/register
/api/v1/auth/login
/api/v1/auth/logout
/api/v1/auth/me
```

Versioning protects the application from breaking existing clients when future API changes are required.

---

# HTTP Methods

Use standard HTTP methods according to resource semantics.

```text
GET
POST
PUT
PATCH
DELETE
```

Preferred usage:

```text
GET
    retrieve data

POST
    create or trigger an action

PUT
    replace a resource where appropriate

PATCH
    partially update a resource

DELETE
    remove or deactivate a resource where appropriate
```

The project does not need to force every endpoint into a rigid REST pattern when a domain action is clearer.

---

# Resource Naming

Resource paths should use plural nouns where appropriate.

Examples:

```text
/users
/businesses
/bookings
/customers
/services
/messages
```

Action endpoints may be used where they represent a meaningful domain operation.

Examples:

```text
/auth/logout
/bookings/{id}/cancel
```

---

# Identifiers

API resources should normally use stable IDs.

IDs must be validated server-side.

The server must never assume that possession of an object ID grants access to that object.

Authorization must verify ownership, membership, tenant scope, and resource access where required.

---

# Request Validation

All client input is untrusted.

The API must validate:

- request body
- path parameters
- query parameters
- headers where applicable
- content type
- field format
- field length
- enum values
- required fields
- cross-field rules

Validation must occur before sensitive business logic executes.

---

# Handler Responsibility

HTTP handlers should remain thin.

Handlers should primarily:

- parse requests
- validate request structure
- obtain authenticated context
- call the appropriate service
- map service results to HTTP responses
- map expected errors to safe API responses

Handlers should not contain large amounts of business logic.

---

# Service Responsibility

Services own application and domain behavior.

Services may perform:

- authentication
- authorization decisions
- booking rules
- account linking policy
- tenant checks
- workflow orchestration
- transaction coordination
- external-provider coordination

Services should not become generic transport-layer code.

---

# Repository Responsibility

Repositories own database interaction.

Repositories should:

- execute sqlc-generated queries
- load and persist domain data
- handle expected database errors
- provide persistence operations to services

Repositories should not decide application-level authorization policy.

---

# Error Model

The API should return consistent structured errors.

Conceptually:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "The request is invalid.",
    "request_id": "..."
  }
}
```

The exact response structure must remain consistent across API endpoints.

---

# Error Categories

The API should distinguish common classes such as:

```text
400 Bad Request
401 Unauthorized
403 Forbidden
404 Not Found
409 Conflict
422 Unprocessable Entity
429 Too Many Requests
500 Internal Server Error
```

Not every domain needs every status.

The selected response must accurately represent the failure.

---

# Security-Sensitive Errors

Authentication and authorization failures must avoid leaking sensitive information.

Do not return:

- password hashes
- provider tokens
- authorization codes
- internal stack traces
- database errors
- private keys
- server secrets
- sensitive account ownership information

Error details must be safe for client exposure.

---

# Request IDs

Requests should carry or receive a request identifier.

The request ID is useful for:

- debugging
- tracing
- support
- security investigation
- correlating server logs

A client-supplied request ID may be accepted only under controlled validation.

The server must never trust it as an authentication or authorization value.

---

# Authentication

Protected API requests require application authentication.

The request flow is conceptually:

```text
Request
   ↓
Session extraction
   ↓
Session validation
   ↓
Local user resolution
   ↓
Authentication context
   ↓
Authorization
   ↓
Handler / Service
```

Authentication establishes the local user identity.

Authorization determines whether that identity may perform the requested operation.

---

# Authentication Endpoints

The planned authentication API includes:

```text
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/logout
GET  /api/v1/auth/me
```

Additional authentication flows may include:

```text
POST /api/v1/auth/password-reset/request
POST /api/v1/auth/password-reset/complete
```

Exact naming may evolve with the completed Phase 3 implementation.

---

# OAuth Endpoints

OAuth provider flows use:

```text
GET /api/v1/auth/google
GET /api/v1/auth/google/callback

GET /api/v1/auth/apple
GET /api/v1/auth/apple/callback
```

The provider-start endpoint:

1. validates the configured provider
2. generates OAuth state
3. generates OIDC nonce
4. generates a PKCE verifier/challenge
5. creates the temporary OAuth transaction
6. protects the transaction
7. redirects the browser to the provider

The callback endpoint:

1. retrieves the transaction
2. validates the transaction
3. validates state
4. exchanges the authorization code
5. validates the provider response
6. validates the ID token
7. resolves the external identity
8. passes the trusted identity to the authentication service
9. establishes the normal application session when authentication succeeds

---

# OAuth Callback Boundary

The OAuth callback must not directly create arbitrary users from untrusted callback data.

The trusted flow is:

```text
Provider callback
   ↓
OAuth validation
   ↓
Verified external identity
   ↓
Authentication service
   ↓
Local user resolution
   ↓
Application session
```

The callback is an authentication transport boundary, not the final authority for application identity.

---

# OAuth Security Requirements

OAuth endpoints must enforce:

```text
state validation
nonce validation
PKCE S256
OIDC signature validation
issuer validation
audience validation
subject validation
expiration validation
not-before validation where applicable
authorized-party validation when required
```

Provider signing algorithms must be restricted to expected algorithms.

OAuth secrets must remain server-side.

---

# OAuth Identity Lookup

After successful provider validation, the application resolves:

```text
provider + provider_subject
```

against the OAuth identity table.

The API must not use email as the authoritative provider identity lookup key.

---

# OAuth Account Linking

Account linking is an authenticated operation.

The conceptual API flow is:

```text
Authenticated User
   ↓
Link Provider
   ↓
Provider Authorization
   ↓
Provider Identity Validation
   ↓
Check External Identity Ownership
   ↓
Create OAuth Identity
```

A provider identity already attached to another user must not be silently transferred.

Email equality must not automatically merge accounts.

---

# Session Authentication

After successful authentication, the backend establishes the normal application session.

OAuth does not create a separate permanent session system.

The resulting authenticated request context must identify the same local user regardless of whether the user authenticated through:

```text
password
Google
Apple
```

---

# Authorization

Authorization occurs after authentication.

The server must determine:

```text
Who is the user?
What tenant or business context applies?
What role does the user have?
Does this role permit the operation?
Does the user have access to this specific resource?
```

The client must never be treated as the authority for these decisions.

---

# Role-Based Access Control

The platform uses application roles including:

```text
CUSTOMER
BUSINESS_OWNER
BUSINESS_ADMIN
MANAGER
STAFF
PLATFORM_ADMIN
PLATFORM_OPERATOR
```

The exact role model may evolve as the multi-tenant architecture is implemented.

Authorization must remain server-side.

---

# Object-Level Authorization

Passing a valid object ID is not sufficient.

For example:

```text
GET /api/v1/bookings/{id}
```

must verify that the authenticated user has permission to access that booking.

This protects against insecure direct object reference vulnerabilities.

---

# Tenant Authorization

Tenant-owned API requests must derive trusted tenant context from authenticated membership and authorization state.

A request such as:

```text
GET /api/v1/businesses/{business_id}/bookings
```

must not trust the URL value alone.

The server must verify that the authenticated principal is authorized for that business.

---

# Query Scoping

Repositories should receive trusted ownership or tenant context where required.

Conceptually:

```text
Authenticated User
        ↓
Authorization
        ↓
Trusted Business ID
        ↓
Repository
        ↓
Tenant-Scoped Query
```

This reduces the risk of accidental cross-tenant access.

---

# Idempotency

Idempotency should be used for operations where clients or infrastructure may safely retry requests.

Important candidates include:

- booking creation
- payment operations
- webhook processing
- imports
- notification operations
- other externally retried commands

Idempotency keys must be scoped appropriately to prevent cross-user or cross-tenant collisions.

---

# Concurrency

The API must assume requests can execute concurrently.

The service and database layers must protect operations such as:

- booking creation
- account linking
- payment transitions
- subscription changes
- import commits

Application-level checks must not be the only protection against race conditions.

---

# Pagination

Collection endpoints should support pagination where datasets may grow significantly.

Examples:

```text
GET /api/v1/bookings
GET /api/v1/customers
GET /api/v1/messages
GET /api/v1/audit-events
```

Pagination should be explicit and predictable.

Cursor pagination should be considered for high-volume or continuously changing collections.

---

# Filtering

Filtering should be represented through validated query parameters.

Example:

```text
GET /api/v1/bookings?status=confirmed
```

The server must validate allowed filter values.

The client must not be allowed to inject raw SQL or arbitrary database expressions through filter parameters.

---

# Sorting

Sorting must use an explicit allowlist.

Example:

```text
?sort=created_at
?sort=-created_at
```

The backend must map permitted sort names to known database columns.

Never directly concatenate arbitrary client-supplied column names into SQL.

---

# Search

Search parameters must be validated and scoped.

Search must not bypass:

- tenant isolation
- authorization
- soft-deletion rules
- data visibility policies

Future dedicated search infrastructure may be introduced when justified by scale.

---

# Time Handling

API requests involving time should use unambiguous representations.

For timestamps, use a standard timezone-aware representation.

The API must preserve enough information to correctly handle:

- timezone differences
- daylight-saving transitions
- booking duration
- local business hours

The backend remains authoritative for booking-time interpretation.

---

# Currency and Money

Monetary API values must be represented without floating-point ambiguity.

Responses should communicate currency explicitly where monetary values are present.

Payment operations must never accept arbitrary currency assumptions from the client.

---

# Authentication Headers and Cookies

The exact browser authentication mechanism is defined by the session architecture.

Where cookies are used, the application must account for:

- HttpOnly
- Secure in production
- SameSite policy
- CSRF protection
- controlled domain/path scope

Sensitive session values must not be exposed to client-side JavaScript unnecessarily.

---

# CORS

CORS must use an explicit allowlist for authenticated application traffic.

The API must not broadly enable arbitrary production origins.

Development configuration may allow the known local frontend origin.

---

# Content Types

Endpoints should explicitly validate supported content types.

JSON APIs should use:

```text
application/json
```

File upload endpoints require separate validation for:

- MIME type
- file extension
- size
- content
- storage destination
- authorization

---

# File Uploads

User-uploaded files must not be trusted based only on their filename or client-provided MIME type.

Upload handling must validate:

- authenticated user
- tenant
- file size
- content type
- file signature where necessary
- storage path
- access policy

Uploads must not be executable through the application server.

---

# Webhooks

Webhook endpoints must be treated as security-sensitive integration boundaries.

They must verify:

- provider signature
- timestamp/replay controls where provided
- event identity
- event authenticity
- idempotency

A webhook must not modify sensitive state before authentication of the webhook itself.

---

# External Provider Boundary

External services must be accessed through controlled provider-specific implementations.

The API must not expose provider-specific credentials or internal transport details to clients.

Examples include:

```text
OAuth
payments
email
SMS
storage
search
messaging
```

---

# Provider-Agnostic Interfaces

Provider abstraction should be used only where multiple providers are realistically expected.

Avoid large generic abstraction layers without a current product requirement.

A practical structure is:

```text
Application Service
        ↓
Provider Interface
        ↓
Provider Adapter
        ↓
External Service
```

---

# Logging

API logs should support operational diagnosis without exposing secrets.

Never log:

- passwords
- session tokens
- OAuth codes
- OAuth state
- OAuth nonce
- PKCE verifier
- OAuth ID tokens
- access tokens
- refresh tokens
- client secrets
- private keys
- payment card data

Request IDs should be used for safe correlation.

---

# Rate Limiting

Rate limiting should be applied to sensitive and abuse-prone endpoints.

High-priority examples include:

- login
- registration
- password reset
- OAuth initiation
- OAuth callbacks
- account linking
- webhook processing
- high-cost search
- public booking endpoints

The exact implementation may use the API layer first and later a shared infrastructure component such as Redis when justified.

---

# API Security Boundary

The API must assume:

```text
All client input is untrusted.
All browser state is untrusted.
All object IDs are untrusted.
All tenant IDs from the client are untrusted.
All provider callback parameters are untrusted until validated.
```

Only validated server-side state may influence privileged operations.

---

# API Documentation

Public or internal API documentation should describe:

- endpoint
- method
- authentication requirement
- request schema
- response schema
- errors
- authorization requirements
- idempotency requirements
- pagination behavior

Documentation must not contain real secrets or production credentials.

---

# Testing

API testing should cover:

```text
authentication
authorization
validation
tenant isolation
object-level access
OAuth callback validation
session behavior
error handling
idempotency
concurrency
pagination
rate limiting
```

Security-sensitive paths require both positive and negative test cases.

---

# Current Phase 3 API Status

Phase 3 authentication API architecture is defined.

The current OAuth foundation includes:

- Google authorization flow foundation
- Apple authorization flow foundation
- OAuth transaction handling
- provider validation
- identity resolution
- OAuth repository operations

The complete application API flow is still in progress.

Remaining Phase 3 integration includes:

- authentication service
- session creation and validation
- registration
- password login
- logout and revocation
- authentication middleware
- `/me`
- RBAC enforcement
- end-to-end OAuth route integration
- authentication UI
- security verification

---

# API Evolution

Backward compatibility should be considered before changing stable API contracts.

Breaking changes should normally result in:

- versioning
- migration strategy
- client update plan
- documentation update

Do not introduce API versioning solely for cosmetic changes.

---

# Core Principles

The API must remain:

```text
Thin at the transport layer
Strong at the service layer
Strict at the authorization boundary
Typed at the data boundary
Transactional where necessary
Tenant-aware
Secure by default
```

The browser is a client.

The API is the enforcement boundary.

The service layer is the application authority.

PostgreSQL remains the durable source of truth.
