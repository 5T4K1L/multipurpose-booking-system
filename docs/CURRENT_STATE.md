# Current State

## Current Phase

PHASE 3 — Authentication + Authorization

## Current Step

3.5-A — Google + Apple OAuth Add-on

The OAuth foundation has been implemented and locally verified.

The next authorized implementation step is:

3.6 — Authentication Service Layer

---

# Completed Phases

## Phase 0 — Engineering Foundation

Phase 0 has been completed and verified.

Completed:

- Git repository foundation
- Documentation foundation
- Next.js frontend foundation
- Go API foundation
- PostgreSQL with Docker Compose
- SQL migration foundation
- Go-to-PostgreSQL connectivity
- API health endpoint
- Frontend-to-API connectivity
- Basic automated API tests
- Go test verification
- Go vet verification
- Go build verification
- Frontend lint verification
- Frontend production build verification

---

## Phase 1 — System Architecture

Phase 1 has been completed.

Completed architecture documentation:

- Backend architecture
- API architecture and conventions
- Frontend architecture and conventions
- Configuration architecture
- Error handling and logging architecture
- Security architecture baseline
- Database architecture and conventions
- Package architecture
- Application lifecycle and dependency wiring
- Consolidated architecture baseline

---

## Phase 2 — Database Architecture and Foundation

Phase 2 has been completed.

Completed:

- database schema strategy
- database naming conventions
- migration runner
- migration tracking
- core users table
- password credential table
- server-side user session table
- OAuth identity table
- sqlc configuration
- typed SQL query generation
- users repository
- authentication repository foundation
- OAuth repository
- PostgreSQL integration testing
- database verification
- schema documentation

Verified:

- PostgreSQL is available for local development
- migrations apply successfully
- users table exists
- user email uniqueness is enforced
- authentication tables exist
- OAuth identity constraints are enforced
- sqlc generation succeeds
- repository integration succeeds
- database tests pass
- Go tests pass
- Go build passes

---

# Phase 3 — Authentication + Authorization

Phase 3 is currently in progress.

The purpose of Phase 3 is to establish secure platform authentication and the foundation required for later authorization.

Authentication establishes:

```text
Who is this user?
```

Authorization will establish:

```text
What may this user do?
```

These responsibilities remain separate.

## Completed Phase 3 Work

Completed authentication foundation:

- authentication architecture
- authentication data model
- authentication database schema
- password hashing with Argon2id
- authentication repository foundation
- user repository foundation
- server-side authentication session database foundation
- Google OAuth/OIDC foundation
- Apple Sign in with Apple foundation

---

# Phase 3.5-A — Google + Apple OAuth Add-on

Status:

COMPLETE — CODE / FOUNDATION VERIFIED

This add-on extends the existing Phase 3 authentication architecture without replacing the password-based authentication model.

## Completed

### OAuth Identity Model

Implemented:

- dedicated OAuth identity table
- local user relationship
- provider constraint
- stable provider subject identity
- unique `(provider, provider_subject)`
- unique `(user_id, provider)`
- user-based OAuth identity index

Supported providers:

- Google
- Apple

The provider subject is the authoritative external identity key.

Email is supplementary account information and is not used as the permanent external identity key.

### OAuth Security Foundation

Implemented:

- cryptographically random OAuth state
- state validation
- cryptographically random OIDC nonce
- nonce validation
- PKCE
- S256 PKCE challenge
- short-lived OAuth transactions
- encrypted OAuth transaction data
- HttpOnly transaction cookie
- Secure cookie support for production
- SameSite protection
- server-side authorization-code exchange
- OIDC issuer validation
- OIDC audience validation
- authorized-party validation when required
- OIDC issued-at validation
- OIDC expiration validation
- not-before validation where applicable
- JWK-backed ID-token signature verification
- provider-specific signing algorithm restrictions
- stable provider subject normalization

### Google OAuth Foundation

Implemented:

- Google OAuth configuration
- Google authorization request foundation
- Google PKCE support
- Google authorization-code exchange
- Google ID-token verification foundation
- Google identity normalization

### Apple Sign in with Apple Foundation

Implemented:

- Apple OAuth configuration
- Apple authorization request foundation
- Apple PKCE support
- Apple authorization-code exchange foundation
- Apple ID-token verification foundation
- Apple identity normalization
- Apple client-secret generation
- ES256 Apple client-secret signing
- Apple private-key configuration support

### Account Linking Boundary

The authentication architecture does not automatically merge accounts solely because an OAuth email matches an existing local account.

The intended model is:

```text
Authenticated local user
        ↓
Explicit link action
        ↓
Provider authentication
        ↓
External identity attachment
```

OAuth identity creation must remain controlled by the authentication service.

### Provider Token Handling

Provider access and refresh tokens are not persisted by the current authentication model.

Provider tokens should only be stored in a future design if a product feature genuinely requires provider API access.

The application remains authoritative for its own sessions.

---

# Real Provider Verification Status

## Code Verification

Status:

PASS

Locally verified:

- OAuth unit tests
- OAuth security tests
- OAuth repository integration tests
- full Go test suite
- Go build
- Go formatting

## Real Google Provider Verification

Status:

NOT YET VERIFIED

The project has not yet completed:

- Google Cloud OAuth application configuration
- real Google client credentials
- real Google redirect URI configuration
- live Google login
- live Google callback verification

## Real Apple Provider Verification

Status:

NOT YET VERIFIED

The project has not yet completed:

- Apple Developer configuration
- Sign in with Apple Services ID configuration
- real Apple Team ID configuration
- real Apple Key ID configuration
- real Apple private key configuration
- real Apple redirect URL configuration
- live Apple login
- live Apple callback verification

Real provider credentials and private keys must be configured manually and must never be provided in chat or committed to Git.

---

# Current Authentication Architecture

The current conceptual authentication model is:

```text
Password
    │
Google OAuth/OIDC
    │
Apple Sign in with Apple
    │
    ▼
Local User Account
    │
    ▼
Application Session
    │
    ▼
Authorization / RBAC
```

The database relationship is:

```text
users
├── user_credentials
├── user_oauth_identities
└── user_sessions
```

The application remains authoritative for:

- local user identity
- account linking
- sessions
- roles
- authorization
- tenant membership

External OAuth providers do not become the application's authorization authority.

---

# Current OAuth Flow

The implemented OAuth foundation follows:

```text
OAuth Start
    ↓
Generate state
    ↓
Generate nonce
    ↓
Generate PKCE verifier
    ↓
Create short-lived encrypted OAuth transaction
    ↓
Store transaction in protected browser cookie
    ↓
Redirect to provider
    ↓
OAuth Callback
    ↓
Retrieve OAuth transaction
    ↓
Validate transaction age
    ↓
Validate state
    ↓
Exchange authorization code
    ↓
Validate ID-token signature
    ↓
Validate OIDC claims
    ↓
Normalize provider identity
    ↓
Authentication Service
    ↓
Application Session
```

The final authentication-service and application-session portions are not yet implemented.

---

# Current Repository Structure Relevant to Authentication

Backend authentication-related code currently includes:

```text
apps/api/internal/auth/
├── password.go
├── password_test.go
├── oauth.go
├── oauth_test.go
├── oauth_state.go
├── oauth_state_test.go
├── oauth_provider.go
├── oauth_provider_test.go
├── oauth_providers.go
├── oauth_exchange.go
├── oauth_exchange_test.go
├── oidc.go
├── oidc_test.go
├── oidc_verify.go
├── oidc_verify_test.go
├── oauth_identity.go
├── oauth_identity_test.go
├── oauth_transaction.go
├── oauth_transaction_test.go
├── oauth_cookie.go
├── oauth_cookie_test.go
├── oauth_adapters.go
├── oauth_adapters_test.go
├── apple.go
├── apple_test.go
├── oauth_config.go
├── oauth_config_test.go
├── oauth_handlers.go
└── oauth_handlers_test.go
```

Authentication repository code:

```text
apps/api/internal/repository/
├── users.go
├── auth.go
└── oauth.go
```

OAuth database query definitions:

```text
apps/api/internal/db/queries/
└── oauth.sql
```

OAuth database migration:

```text
migrations/
├── 000005_create_user_oauth_identities.up.sql
└── 000005_create_user_oauth_identities.down.sql
```

---

# Current Scope Boundary

The following remain intentionally incomplete:

- authentication service
- application session service integration
- registration
- login
- logout
- current-user endpoint
- authentication middleware
- authorization/RBAC implementation
- user account lifecycle integration
- password reset
- email verification
- authentication frontend UI
- Google live-provider verification
- Apple live-provider verification
- multi-tenancy
- organizations
- businesses
- customers
- services
- staff
- resources
- availability
- booking engine
- CRM
- import/export
- messaging
- notifications
- background jobs
- search
- marketplace
- payments
- subscriptions
- billing
- public API
- production deployment
- advanced infrastructure

---

# Known Issues

- `docs/CURRENT_STATE.md` was previously stale and has now been synchronized with the current Phase 3 state.
- Authentication service integration is not yet implemented.
- Application session creation is not yet integrated with password and OAuth authentication flows.
- Real Google provider configuration has not yet been completed.
- Real Apple provider configuration has not yet been completed.
- Live Google and Apple authentication have not yet been verified.
- Authentication frontend UI/UX has not yet been implemented.
- Multi-tenancy is not implemented.
- Product-domain functionality is not yet implemented.

---

# Security Status

The Phase 3 OAuth foundation includes security controls for:

- state protection
- nonce protection
- PKCE S256
- encrypted OAuth transactions
- protected browser transaction cookies
- OIDC signature verification
- provider JWK verification
- issuer validation
- audience validation
- nonce validation
- expiration validation
- provider subject identity
- account-linking safety
- runtime secret protection
- avoidance of secret/token logging

The project does not yet claim:

- full OWASP ASVS verification
- penetration testing
- production security readiness
- formal security compliance
- complete Phase 13 security hardening

Those remain part of their authorized security work.

---

# Current Architecture

```text
Next.js
   ↓
HTTP API
   ↓
Go
   ↓
Application / Authentication Services
   ↓
Repository
   ↓
PostgreSQL
```

Authentication-specific flow:

```text
Password
Google OAuth/OIDC
Apple Sign in with Apple
        ↓
Authentication Service
        ↓
Application Session
        ↓
Authorization
```

The system remains a modular monolith.

---

# Important Technical Decisions

- Modular monolith rather than microservices.
- Next.js + React + TypeScript for the frontend.
- Go for the backend.
- PostgreSQL as the primary data store.
- pgx for PostgreSQL connectivity.
- sqlc for typed SQL access.
- SQL migrations for schema changes.
- Explicit dependency wiring.
- Server-side security enforcement.
- Responsive UI from the beginning.
- Vertical feature slices.
- Free-first development workflow.
- Small, verifiable implementation steps.
- Password authentication remains separate from external OAuth identities.
- OAuth identities are stored separately from the core `users` table.
- Provider subject is the stable external OAuth identity key.
- Email is not used as the permanent OAuth identity key.
- OAuth does not create a separate permanent application session system.
- The application remains authoritative for authorization and sessions.
- Real provider credentials remain runtime configuration and are never stored in source control.

---

# Documentation Source of Truth

The current architecture and development rules are defined by:

```text
PROJECT_CONTEXT.md
CURRENT_STATE.md
ARCHITECTURE.md
ARCHITECTURE_BASELINE.md
BACKEND_ARCHITECTURE.md
API_ARCHITECTURE.md
FRONTEND_ARCHITECTURE.md
CONFIGURATION_ARCHITECTURE.md
ERROR_HANDLING_AND_LOGGING.md
SECURITY_ARCHITECTURE.MD
DATABASE_ARCHITECTURE.md
DATABASE_CONVENTIONS.md
DATABASE_SCHEMA_STRATEGY.md
AUTHENTICATION_ARCHITECTURE.md
AUTHENTICATION_DATA_MODEL.md
PACKAGE_ARCHITECTURE.md
APPLICATION_LIFECYCLE.md
DECISIONS.md
ROADMAP.md
TECHNICAL_TERMS.md
```

Documentation must remain synchronized with implementation.

---

# Phase 3 Progress

Current Phase 3 sequence:

```text
3.1  Authentication Architecture                    COMPLETE
3.2  Authentication Data Model                      COMPLETE
3.3  Authentication Database Schema                 COMPLETE
3.4  Password Hashing                               COMPLETE
3.5  Authentication Repository Layer                COMPLETE
3.5-A Google + Apple OAuth Add-on                  COMPLETE
3.6  Authentication Service Layer                   NEXT
3.7  Session / Token Management                     PENDING
3.8  Registration                                   PENDING
3.9  Login                                          PENDING
3.10 Logout + Session Revocation                    PENDING
3.11 Authentication Middleware                      PENDING
3.12 /me / Current User                             PENDING
3.13 Authorization / RBAC Foundation                PENDING
3.14 Authentication Security Tests                  PENDING
3.15 Phase 3 Review + Closure                       PENDING
```

---

# Next Step

The next authorized implementation step is:

## PHASE 3.6 — Authentication Service Layer

The authentication service will become the application-level boundary that unifies:

```text
Password authentication
Google OAuth/OIDC
Apple Sign in with Apple
        ↓
Local User Identity
        ↓
Application Authentication State
```

It will remain separate from:

- authorization
- tenant access
- business permissions
- booking permissions
- RBAC implementation

The application session mechanism will be integrated through the authorized session-management step.

---

# Phase Boundary Rule

Only the currently authorized Phase 3 step may be implemented.

Do not begin Phase 4 work.

Do not implement unrelated booking, CRM, payments, messaging, marketplace, or advanced infrastructure functionality.

After Phase 3 completion:

- run all required tests
- run formatting and static checks
- review security
- synchronize documentation
- report manual configuration requirements
- report known issues
- formally close Phase 3
- hard stop before Phase 4
