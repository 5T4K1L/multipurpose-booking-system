# Architecture

## Purpose

This document defines the high-level application architecture for the Universal Booking Platform.

The system is designed as a modular monolith with clear boundaries between the frontend, HTTP API, application services, repositories, and PostgreSQL.

The architecture should remain simple enough for the current development stage while providing a clean foundation for future growth.

---

# High-Level Architecture

```text
Browser
   ↓
Next.js Web Application
   ↓
HTTP API
   ↓
Go Application
   ↓
PostgreSQL
```

The backend follows this general dependency direction:

```text
cmd/api
   ↓
server / HTTP handlers
   ↓
application services
   ↓
repositories
   ↓
PostgreSQL
```

The application uses explicit dependency wiring.

---

# Architectural Style

The initial system is a modular monolith.

The platform should not be decomposed into microservices unless actual system requirements justify doing so.

The modular monolith provides:

- simpler development
- lower infrastructure requirements
- easier local development
- easier debugging
- lower operational complexity
- clear feature boundaries
- a path toward future service extraction when justified

Modules should communicate through defined application boundaries rather than through uncontrolled cross-module access.

---

# Frontend

The frontend uses:

```text
Next.js
React
TypeScript
```

The frontend is responsible for:

- user interface
- client-side interaction
- presentation
- form handling
- user experience
- API communication
- client-side validation for usability
- responsive behavior

The frontend is never an authority for:

- authentication
- authorization
- tenant access
- ownership
- financial state
- booking availability
- security-sensitive business rules

The backend must independently validate and enforce those rules.

---

# Backend

The backend uses Go.

The backend is responsible for:

- HTTP API handling
- input validation
- authentication
- authorization
- application services
- business rules
- repository/data access
- transaction coordination
- security enforcement
- error handling

Handlers should remain thin.

Business logic should not be placed directly inside HTTP handlers when it belongs in an application service.

---

# Database

PostgreSQL is the primary relational database and source of truth.

The application relies on PostgreSQL for:

- persistent data
- relational integrity
- unique constraints
- foreign-key protection
- transactional consistency
- concurrency protection where applicable

Schema changes must use version-controlled SQL migrations.

Application code accesses PostgreSQL through the repository/data-access layer.

---

# Authentication Architecture

Authentication supports three mechanisms:

```text
Password
Google OAuth / OIDC
Apple Sign in with Apple
        ↓
Local User Account
        ↓
Application Session
        ↓
Authorization / RBAC
```

External OAuth authenticates an external provider identity.

The application remains authoritative for:

- local users
- account linking
- application sessions
- authorization
- roles
- tenant membership

External OAuth providers do not become the authority for application authorization.

---

# Authentication Identity Model

Authentication identities are separated from the core user record where appropriate.

Conceptually:

```text
users
├── user_credentials
├── user_oauth_identities
└── user_sessions
```

Password credentials represent local password authentication.

OAuth identities represent relationships between local users and supported external providers.

Sessions represent authenticated application state.

The core `users` record remains the local platform identity.

---

# OAuth Architecture

The OAuth/OIDC architecture follows:

```text
Google OAuth / OIDC
Apple Sign in with Apple
        ↓
OAuth Provider Validation
        ↓
External OAuth Identity
        ↓
Local User Account
        ↓
Application Authentication
        ↓
Application Session
        ↓
Authorization / RBAC
```

The OAuth system uses:

- authorization-code flow
- state
- nonce
- PKCE with S256
- server-side code exchange
- OIDC ID-token validation
- JWK-backed signature verification
- provider issuer validation
- provider audience validation
- nonce validation
- expiration validation
- stable provider subject identity

OAuth provider email values are supplementary account information.

Email is not treated as the permanent external identity key.

---

# OAuth Identity Persistence

OAuth identities are persisted in a dedicated table:

```text
user_oauth_identities
```

The relationship is:

```text
users
   ↓
user_oauth_identities
```

The external identity is identified by:

```text
provider + provider_subject
```

The database initially supports:

```text
UNIQUE(provider, provider_subject)
UNIQUE(user_id, provider)
```

This prevents the same external provider identity from being linked to multiple users and initially limits a user to one identity per provider.

Provider-specific identifiers are intentionally not placed directly on the `users` table.

The design therefore avoids provider-specific columns such as:

```text
google_id
apple_id
google_email
apple_email
```

---

# OAuth Account Linking

OAuth account linking must remain explicit and secure.

An existing local account must not be automatically merged solely because an OAuth provider returns an email address matching the local account.

The intended model is:

```text
Authenticated local user
        ↓
Explicit link action
        ↓
Provider authentication
        ↓
External identity validation
        ↓
Identity attachment
```

An unauthenticated OAuth attempt must not gain access to an existing local account merely through email matching.

---

# OAuth Transaction Security

An OAuth authorization transaction contains short-lived security state including:

```text
provider
state
nonce
PKCE verifier
creation time
```

The transaction is protected from browser tampering using encrypted server-generated data stored in an HttpOnly cookie.

OAuth transaction data is short-lived.

The application must not log:

- authorization codes
- state values
- nonces
- PKCE verifiers
- ID tokens
- access tokens
- refresh tokens
- private keys
- client secrets

---

# OAuth Provider Separation

Google and Apple authentication share the same application-level identity model but retain provider-specific behavior where required.

Google uses:

```text
Google client ID
Google client secret
Google redirect URL
Google OIDC endpoints
Google JWKs
```

Apple uses:

```text
Apple client ID
Apple Team ID
Apple Key ID
Apple private key
Apple redirect URL
Apple OIDC endpoints
Apple JWKs
```

The Apple client secret is generated by the backend from the Apple private key rather than being treated as a permanently stored application secret.

Provider-specific implementation details remain inside the authentication module rather than leaking into unrelated application modules.

---

# Application Authentication Boundary

External provider authentication must eventually resolve to the platform's local authentication model.

The intended flow is:

```text
External Provider
        ↓
Validated External Identity
        ↓
Local User
        ↓
Application Authentication Service
        ↓
Application Session
```

The application does not treat an OAuth access token as its own session.

The application session remains under application control.

Provider access and refresh tokens are not persisted unless a future feature explicitly requires provider API access.

---

# Authorization Boundary

Authentication answers:

```text
Who is this user?
```

Authorization answers:

```text
What may this user do?
```

Authentication and authorization must remain separate.

Authentication establishes trusted identity context.

Authorization evaluates permissions, roles, tenant membership, ownership, and other application rules.

Authorization must always be enforced server-side.

---

# Multi-Tenancy Boundary

Multi-tenancy is a later phase.

Authentication identifies the local user.

Authentication does not automatically grant access to every business, organization, or tenant.

Tenant membership and tenant access must be evaluated separately by authorization logic.

A valid user session is not equivalent to valid access to every tenant-owned resource.

---

# API Boundary

The frontend communicates with the backend through HTTP.

The intended flow is:

```text
Next.js
   ↓
HTTP API
   ↓
HTTP Handler
   ↓
Application Service
   ↓
Repository
   ↓
PostgreSQL
```

Handlers must not access PostgreSQL directly.

Application services must not depend on frontend-specific behavior.

The frontend must not connect directly to PostgreSQL.

---

# Repository Boundary

Repositories provide controlled data access to PostgreSQL.

Repositories should expose operations that are meaningful to the application rather than generic unrestricted SQL access.

SQL remains explicit and reviewable.

sqlc-generated code must not be manually edited.

The SQL definitions and migrations remain the source of truth for generated database access.

---

# Dependency Direction

The preferred dependency direction is:

```text
HTTP / Transport
      ↓
Application Services
      ↓
Domain / Application Logic
      ↓
Repositories
      ↓
Database
```

Lower layers should not depend on higher layers.

Provider-specific OAuth code belongs within the authentication boundary.

---

# Security Boundary

The browser is untrusted.

Security-sensitive decisions must be made by the backend.

The backend must independently verify:

- identity
- authentication state
- permissions
- ownership
- tenant access
- business rules
- financial state
- booking rules

The frontend may improve user experience through client-side validation, but client-side validation is not a security control.

---

# Configuration Boundary

Environment-specific configuration belongs outside application source code.

Sensitive values must remain server-side.

Examples include:

```text
database credentials
OAuth client secrets
Apple private keys
OAuth transaction encryption keys
API credentials
signing secrets
```

Real credentials must never be committed to Git.

---

# Responsive Architecture

Responsive behavior is part of the architecture rather than a final polish step.

The platform should be implemented as:

```text
Mobile-first
+
Responsive
+
Desktop-optimized
```

User-facing features must support appropriate experiences across:

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

Authentication interfaces, business interfaces, customer interfaces, CRM, messaging, calendars, and booking flows must be responsive when implemented.

---

# Current Authentication Status

Phase 3 authentication is currently in progress.

Completed authentication foundation:

- authentication architecture
- authentication data model
- authentication database foundation
- password hashing
- authentication repository
- user repository
- session database foundation
- Google OAuth/OIDC foundation
- Apple Sign in with Apple foundation

Phase 3.5-A Google + Apple OAuth foundation is complete at the code/foundation level.

Real provider-console configuration and live provider verification have not yet been completed.

The next authorized authentication step is:

```text
Phase 3.6 — Authentication Service Layer
```

---

# Phase Boundaries

Only the currently authorized phase may be implemented.

The current architecture does not authorize premature implementation of:

- multi-tenancy
- booking engine
- CRM
- messaging
- payments
- subscriptions
- production deployment
- advanced infrastructure
- microservices
- Kubernetes
- multi-region architecture
- specialized search infrastructure

Future architecture must be introduced only when the corresponding phase is authorized and the requirement is justified.

---

# Core Architectural Principles

The system should prefer:

```text
Clear boundaries
+
Explicit dependencies
+
Strong database integrity
+
Server-side security
+
Small verifiable changes
+
Simple infrastructure
+
Measured scaling
```

over:

```text
Premature microservices
+
Unnecessary abstraction
+
Speculative infrastructure
+
Frontend-enforced security
+
Duplicated sources of truth
```

The platform should be built as a system capable of growing significantly, without paying the complexity cost of large-scale infrastructure before that complexity is actually required.
