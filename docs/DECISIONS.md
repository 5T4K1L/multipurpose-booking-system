# Architecture Decision Records

This document records important architectural and engineering decisions for the Universal Booking Platform.

Decisions should be changed deliberately.

When a major architectural direction changes, create a new decision or supersede the previous decision rather than silently rewriting history.

---

# ADR-0001 — Modular Monolith

## Status

Accepted

## Decision

The platform will begin as a modular monolith rather than a microservice architecture.

The application will have clear internal domain boundaries while remaining deployable as a small number of independently executable components.

The initial application structure is:

```text
Next.js frontend
        ↓
Go API
        ↓
PostgreSQL
```

Background workers and additional infrastructure may be introduced later where justified.

## Rationale

The project is being built by a solo developer and must remain practical to develop, test, deploy, and maintain.

A modular monolith provides:

- clear boundaries
- simpler deployment
- fewer distributed-system failure modes
- easier local development
- lower infrastructure cost
- easier debugging

Microservices may be considered only when measurable product or scaling requirements justify them.

---

# ADR-0002 — PostgreSQL as the System of Record

## Status

Accepted

## Decision

PostgreSQL is the authoritative source of truth for persistent application data.

## Rationale

PostgreSQL provides:

- strong transactional guarantees
- foreign keys
- unique constraints
- check constraints
- concurrency controls
- mature indexing
- reliable tooling
- strong Go support

Caches, search indexes, queues, and frontend state remain derived or transient unless an explicit future decision states otherwise.

---

# ADR-0003 — Go for the Backend API

## Status

Accepted

## Decision

The backend API will be implemented in Go.

The backend stack includes:

```text
Go
pgx
sqlc
PostgreSQL
SQL migrations
```

## Rationale

Go provides:

- simple deployment
- strong performance
- predictable concurrency
- static typing
- good HTTP support
- a small runtime footprint
- mature PostgreSQL tooling

The architecture favors straightforward code over excessive framework abstraction.

---

# ADR-0004 — Next.js for the Frontend

## Status

Accepted

## Decision

The frontend will use:

```text
Next.js
React
TypeScript
Tailwind CSS
shadcn/ui
```

## Rationale

This stack provides:

- strong TypeScript integration
- responsive UI development
- reusable components
- suitable routing
- good developer ergonomics
- a practical path from development to production

The frontend remains a client of the Go API rather than a direct database client.

---

# ADR-0005 — Responsive Web Application

## Status

Accepted

## Decision

The platform will be mobile-first and fully responsive.

Responsive support is required across:

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

This applies to:

- customer interfaces
- business administration
- platform administration
- CRM
- messaging
- calendar
- booking
- authentication

## Rationale

The platform must work across phones, tablets, laptops, desktops, and large displays without maintaining separate applications.

Accessibility, touch interaction, performance, and responsive behavior are considered part of the core architecture.

---

# ADR-0006 — Vertical Feature Slices

## Status

Accepted

## Decision

Features should generally be developed in vertical slices across the relevant layers rather than building every frontend feature first and backend functionality later.

A typical slice is:

```text
database
   ↓
repository
   ↓
service
   ↓
API
   ↓
frontend
   ↓
tests
```

## Rationale

Vertical slices make it easier to:

- verify complete behavior
- identify integration problems early
- keep architecture aligned with actual features
- avoid large unfinished layers
- maintain small, verifiable changes

Phase boundaries remain authoritative even when a feature spans multiple layers.

---

# ADR-0007 — Free-First Development and Infrastructure

## Status

Accepted

## Decision

The project should begin using the lowest-cost practical tooling and infrastructure.

The architecture must not depend on paid AI features, premium development agents, unnecessary managed infrastructure, or expensive services during early development.

Paid infrastructure should be introduced when justified by:

- real users
- revenue
- operational bottlenecks
- security requirements
- measurable scale requirements

## Rationale

The initial objective is to build a production-quality portfolio and SaaS foundation without creating unnecessary financial commitments.

The architecture must remain scalable without requiring early overinvestment.

---

# ADR-0008 — Local Application Identity with External OAuth Identities

## Status

Accepted

## Decision

Google and Apple authentication will authenticate external identities, but the application will maintain its own authoritative local user identity.

The relationship is:

```text
Google / Apple
      ↓
External OAuth Identity
      ↓
Local User
      ↓
Application Session
      ↓
Authorization
```

The application remains authoritative for:

- local users
- account status
- sessions
- roles
- tenant membership
- permissions

## Rationale

External authentication providers should not become the application's identity or authorization database.

A local identity allows the platform to:

- support multiple authentication mechanisms
- change providers later
- preserve application roles
- maintain tenant membership independently
- revoke application sessions
- support future authentication providers

---

# ADR-0009 — OAuth Identity Uses Provider Subject, Not Email

## Status

Accepted

## Decision

External OAuth identities are identified using:

```text
provider + provider_subject
```

The database enforces:

```text
UNIQUE(provider, provider_subject)
```

The initial model also enforces:

```text
UNIQUE(user_id, provider)
```

Email is treated as account information rather than the permanent external identity key.

## Rationale

Provider subjects are intended to represent the provider-side identity.

Email addresses can:

- change
- be private or relay addresses
- be unavailable
- be represented differently by providers

Using email as the sole external identity key could cause incorrect account association.

---

# ADR-0010 — No Automatic Account Merge by Email

## Status

Accepted

## Decision

The application will not automatically merge or attach an OAuth identity to an existing local account solely because the email addresses match.

Explicit authenticated account-linking is required.

## Rationale

A matching email value is not sufficient authorization to take control of another authentication identity.

Automatic email-based merging creates unnecessary account-takeover risk and ambiguous ownership behavior.

The safer flow is:

```text
Authenticated local user
        ↓
Explicit link action
        ↓
External provider authentication
        ↓
Verified provider identity
        ↓
Identity ownership check
        ↓
Attach to local user
```

---

# ADR-0011 — OAuth Authorization Code Flow with PKCE

## Status

Accepted

## Decision

Google and Apple OAuth use the authorization-code flow with PKCE S256.

The implementation also uses:

```text
state
OIDC nonce
OIDC ID-token validation
provider JWK verification
```

## Rationale

The architecture needs protection against:

- authorization-response tampering
- authorization-code interception
- replay
- token substitution
- forged identity tokens

PKCE, state, nonce, and OIDC validation provide layered protection around the external authentication transaction.

---

# ADR-0012 — Encrypted Short-Lived OAuth Transaction

## Status

Accepted

## Decision

Temporary OAuth transaction state is stored in an encrypted, short-lived browser cookie.

The transaction contains values such as:

```text
provider
state
nonce
PKCE verifier
creation time
```

The cookie is:

```text
HttpOnly
SameSite=Lax
Secure in production
```

The encryption key remains server-side.

## Rationale

The OAuth transaction needs to survive the redirect between the browser and provider without becoming a long-lived database identity record.

Encryption provides confidentiality and integrity while the short lifetime reduces exposure.

The transaction is not an application session.

---

# ADR-0013 — OIDC ID Token Validation Is Server-Side

## Status

Accepted

## Decision

OIDC ID tokens are validated on the backend before external identity data is trusted.

Validation includes:

```text
signature
issuer
audience
subject
nonce
expiration
issued-at
not-before where applicable
authorized party where applicable
```

Provider signing algorithms are restricted to the algorithms expected by each provider.

## Rationale

Claims returned by an external provider must not be trusted solely because they were received through a callback.

Cryptographic verification and claim validation establish a trusted external identity before it enters the local authentication system.

---

# ADR-0014 — Provider Tokens Are Not Application Sessions

## Status

Accepted

## Decision

Google and Apple provider responses do not become the application's permanent authentication session.

After successful provider authentication:

```text
validated provider identity
        ↓
local user
        ↓
normal application session
```

The application uses one session architecture for password and OAuth authentication.

## Rationale

Application authorization must remain independent from provider-specific token behavior.

This gives the platform a consistent session lifecycle for:

- expiration
- logout
- revocation
- account disablement
- authorization
- tenant access

---

# ADR-0015 — Do Not Persist Provider Access Tokens Unless Required

## Status

Accepted

## Decision

The current authentication architecture does not persist Google or Apple access/refresh tokens.

Provider tokens may only be introduced later if a concrete product integration requires ongoing provider API access.

Such a requirement must receive a dedicated security and storage design.

## Rationale

Storing tokens that the product does not need increases:

- attack surface
- secret-management requirements
- breach impact
- operational complexity

Authentication should collect and retain only what the application actually needs.

---

# ADR-0016 — Server-Side Authorization

## Status

Accepted

## Decision

Authorization is enforced by the backend.

The client may display UI according to permissions, but the server remains authoritative.

Authorization must validate:

- local identity
- role
- tenant membership
- object ownership
- resource access
- operation-specific permission

## Rationale

Client-side authorization can always be bypassed by a malicious client.

Server-side enforcement is required for:

- tenant isolation
- business administration
- customer access
- staff permissions
- platform administration
- booking operations

---

# ADR-0017 — Database Constraints Enforce Durable Invariants

## Status

Accepted

## Decision

Important durable invariants should be represented in PostgreSQL where practical.

Examples include:

```text
foreign keys
unique constraints
check constraints
```

Application logic remains responsible for business workflows.

## Rationale

Application-only constraints are vulnerable to:

- race conditions
- bugs
- future code paths
- concurrent requests
- administrative scripts

Database constraints provide a final integrity boundary.

---

# ADR-0018 — Server-Enforced Multi-Tenancy

## Status

Accepted

## Decision

Tenant isolation will be enforced server-side.

The frontend must not be trusted to choose arbitrary tenant context.

The conceptual flow is:

```text
Authenticated User
       ↓
Authorized Membership
       ↓
Trusted Tenant Context
       ↓
Tenant-Scoped Query
```

## Rationale

The platform is a multi-tenant SaaS.

Cross-tenant data exposure would be a critical security failure.

Tenant context therefore belongs to trusted server-side authorization state.

---

# ADR-0019 — PostgreSQL First, Scale Infrastructure Later

## Status

Accepted

## Decision

The platform will begin with PostgreSQL as the primary database without prematurely introducing distributed database infrastructure.

Additional infrastructure may later be introduced independently when justified.

Potential future components include:

```text
Redis
background workers
search infrastructure
object storage
read replicas
partitioning
CDN
```

## Rationale

The application should scale components independently rather than assuming every workload requires the same infrastructure.

Premature distribution increases:

- operational complexity
- cost
- failure modes
- development time

---

# ADR-0020 — Security by Phase, Not Security at the End

## Status

Accepted

## Decision

Security controls are introduced throughout development.

Phase 3 addresses authentication and authorization foundations.

Later security hardening will systematically verify and strengthen the entire platform.

The project targets:

```text
OWASP ASVS Level 2 baseline
Selective Level 3 controls for sensitive areas
```

Formal compliance is not claimed until a dedicated verification process is completed.

## Rationale

Deferring all security decisions until the end creates architectural rework and security debt.

Security requirements must influence database, API, authentication, tenancy, payment, and infrastructure design as those systems are built.

---

# ADR-0021 — Provider-Agnostic Integration Boundaries

## Status

Accepted

## Decision

External integrations should use small, practical provider interfaces where multiple providers are reasonably expected.

Potential integration categories include:

```text
payments
email
storage
search
messaging
observability
```

OAuth providers use provider-specific adapters behind a controlled authentication boundary.

## Rationale

The platform should not become permanently coupled to one vendor where a future provider change is realistic.

At the same time, abstraction must remain proportional to actual requirements.

Large generic interfaces without real use cases are discouraged.

---

# ADR-0022 — Application Remains the Booking Authority

## Status

Accepted

## Decision

The platform's booking engine remains authoritative for booking correctness.

Future AI assistants, external clients, integrations, or automation systems must access booking capabilities through controlled application APIs.

They must not directly modify booking tables or bypass availability logic.

## Rationale

Booking correctness depends on:

- staff availability
- resource availability
- business hours
- holidays
- buffers
- lead time
- existing bookings
- concurrency protection

A centralized booking engine reduces the risk of conflicting booking behavior.

---

# ADR-0023 — CRM Uses PostgreSQL as Source of Truth

## Status

Accepted

## Decision

The customer CRM will provide a spreadsheet-like user experience, but PostgreSQL remains authoritative.

The CRM will eventually support:

```text
import
export
preview
mapping
validation
deduplication
warnings
confirmation
audit
```

## Rationale

The product needs spreadsheet usability without creating a second database.

The frontend should provide the familiar interaction model while the backend preserves relational integrity.

---

# ADR-0024 — Messaging Is First-Class Business-Customer Communication

## Status

Accepted

## Decision

Messaging will use individual business-to-customer conversation threads.

The system is not designed as a public group-chat platform.

Messaging will eventually integrate with:

- customers
- bookings
- staff assignment
- internal notes
- notification workflows

## Rationale

The primary use case is business/customer communication around appointments and services.

This structure supports future AI-assisted reception while retaining clear customer-specific context.

---

# ADR-0025 — Free-Tier AI Development Workflow

## Status

Accepted

## Decision

Development assistance should remain compatible with the available ChatGPT Free tier.

The project must not require:

- paid-only AI models
- unlimited context
- premium autonomous agents
- premium repository capabilities
- background AI execution

Development should use incremental, manually verified changes.

## Rationale

The project must remain sustainable without assuming ongoing paid AI tooling.

Engineering correctness must not be sacrificed to work around free-tier limitations.

---

# Decision Maintenance

When making a major architecture change:

1. Identify the affected decision.
2. Determine whether the existing decision is still valid.
3. Create a new ADR when the direction changes materially.
4. Mark the previous decision as superseded when appropriate.
5. Update affected architecture documents.
6. Update `CURRENT_STATE.md`.
7. Update the roadmap if phase scope changes.

Architecture documentation and implementation must remain consistent.

---

# Current Decision Context

The current project state is:

```text
Architecture:
Modular monolith

Frontend:
Next.js + React + TypeScript

Backend:
Go

Database:
PostgreSQL

Data access:
pgx + sqlc

Migrations:
Versioned SQL migrations

Authentication:
Password + Google OAuth/OIDC + Apple Sign in with Apple

External OAuth identity:
(provider, provider_subject)

Session:
Application-controlled

Authorization:
Server-side RBAC

Multi-tenancy:
Server-enforced

Development:
Free-first / incremental / manually verified
```

Phase 3 authentication and authorization implementation is currently in progress.
