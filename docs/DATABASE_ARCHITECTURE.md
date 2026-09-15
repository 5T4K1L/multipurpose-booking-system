# Database Architecture

## Purpose

This document defines the database architecture for the Universal Booking Platform.

PostgreSQL is the system of record for persistent application data.

The database must prioritize:

- correctness
- consistency
- transactional integrity
- security
- tenant isolation
- predictable performance
- maintainability
- safe schema evolution

---

# Database Technology

The primary database is:

```text
PostgreSQL
```

Development uses PostgreSQL through Docker Compose.

The Go API accesses PostgreSQL through:

```text
pgx
sqlc
versioned SQL migrations
```

The frontend never connects directly to PostgreSQL.

---

# Database Source of Truth

PostgreSQL is authoritative for persistent application state.

The database is the source of truth for:

- users
- authentication state
- sessions
- businesses
- customers
- services
- staff
- resources
- bookings
- messaging
- CRM data
- subscriptions
- payments
- platform configuration
- audit records

Frontend state, caches, search indexes, queues, and derived data must not become independent sources of truth.

---

# Application Data Flow

The standard data flow is:

```text
Next.js
   ↓
HTTP API
   ↓
Go Handler
   ↓
Service
   ↓
Repository
   ↓
sqlc-generated query
   ↓
PostgreSQL
```

The frontend must never bypass the API to access PostgreSQL.

---

# Database Ownership

Database access belongs to the backend.

The Go application owns domain behavior.

The repository layer owns database interaction.

SQL queries are generated and type-safe through sqlc.

The database owns:

- constraints
- foreign keys
- unique rules
- check constraints
- indexes
- transaction enforcement

The service layer owns:

- business rules
- authorization decisions
- workflow orchestration
- domain validation

Both layers are necessary.

---

# PostgreSQL Schema

The application uses the standard PostgreSQL schema unless a future architectural decision introduces an explicit need for additional schemas.

Domain separation should primarily be established through:

- table naming
- repository boundaries
- package structure
- ownership conventions
- foreign-key relationships

Multiple schemas must not be introduced merely for organizational appearance.

---

# Naming Conventions

Database naming follows PostgreSQL-friendly conventions.

Tables use:

```text
snake_case
plural nouns
```

Examples:

```text
users
user_credentials
user_sessions
user_oauth_identities
businesses
bookings
booking_items
```

Columns use:

```text
snake_case
```

Primary keys normally use:

```text
id
```

Foreign keys normally use:

```text
<referenced_table_singular>_id
```

Examples:

```text
user_id
business_id
service_id
staff_id
resource_id
```

---

# Primary Keys

Application entities normally use stable UUID identifiers.

Primary keys must be:

- unique
- immutable
- safe for distributed generation
- independent of user-visible ordering

The API must not rely on sequential database identifiers as a security boundary.

---

# Foreign Keys

Relationships between persistent entities should be enforced using PostgreSQL foreign keys whenever practical.

Foreign keys provide database-level integrity even when an application bug bypasses expected service behavior.

Examples:

```text
user_oauth_identities.user_id
        ↓
users.id
```

```text
user_sessions.user_id
        ↓
users.id
```

Foreign-key behavior must be chosen deliberately.

Cascading deletion must not be used blindly.

---

# Authentication Database Structure

Authentication is represented through related tables:

```text
users
├── user_credentials
├── user_oauth_identities
└── user_sessions
```

The local `users.id` remains the authoritative application identity.

---

# OAuth Identity Architecture

External OAuth identities are stored separately from local users.

The conceptual structure is:

```text
user_oauth_identities
├── id
├── user_id
├── provider
├── provider_subject
├── created_at
└── updated_at
```

The external identity is uniquely determined by:

```text
provider + provider_subject
```

The database must enforce:

```text
UNIQUE(provider, provider_subject)
```

The initial architecture also enforces:

```text
UNIQUE(user_id, provider)
```

This prevents one external identity from being attached to multiple local users and initially limits each local user to one identity per provider.

---

# OAuth Identity Rules

OAuth email must not be the primary database identity key.

The database must not identify a Google or Apple account using only:

```text
email
```

Instead:

```text
(provider, provider_subject)
```

is the stable external identity.

This protects against account association errors caused by provider email behavior.

---

# Timestamps

Persistent entities should use timezone-aware PostgreSQL timestamps.

Preferred type:

```text
timestamptz
```

Common lifecycle fields include:

```text
created_at
updated_at
deleted_at
```

Other time-related fields must be named according to their semantic meaning.

For example:

```text
expires_at
starts_at
ends_at
confirmed_at
cancelled_at
revoked_at
```

All server-side timestamps should be stored in UTC.

Display timezone conversion belongs to the application or client presentation layer.

---

# Time and Timezone

The booking platform must support businesses and customers across time zones.

The database must not assume that all users operate in one local timezone.

Business and location data should explicitly retain the timezone required to interpret local business hours, appointments, availability, and schedules.

Instants are stored as UTC-capable timestamps.

User-facing local times are derived from the appropriate timezone.

---

# Soft Deletion

Soft deletion may be used where historical relationships must be preserved.

A typical pattern is:

```text
deleted_at
```

Soft deletion must not be applied automatically to every table.

The appropriate strategy depends on the domain.

Examples where historical retention may matter:

- customers
- bookings
- messages
- audit records
- financial records

Authentication data requires additional lifecycle consideration because deleting a user must also address credentials, OAuth identities, sessions, and related records.

---

# Status Columns

Status fields should represent a clearly defined finite state machine.

Examples:

```text
active
disabled
pending
confirmed
cancelled
completed
```

Where practical, valid states should be constrained by the database.

Status must not become an arbitrary free-text field when the domain requires a finite set.

---

# Boolean Columns

Boolean columns should be used only when the domain genuinely has two meaningful states.

Examples:

```text
is_active
is_verified
is_primary
```

When a property has more than two meaningful states, use a status or explicit nullable value rather than encoding multiple states into several booleans.

---

# NULL Handling

`NULL` must have a defined meaning.

Possible meanings include:

- unknown
- not provided
- not applicable
- not yet assigned
- not yet occurred

Do not use nullable columns merely to avoid making a schema decision.

Use `NOT NULL` when the domain requires the value.

---

# Unique Constraints

Unique constraints enforce invariants that must remain true regardless of application behavior.

Examples include:

```text
unique email policies where appropriate
unique(provider, provider_subject)
unique(user_id, provider)
unique external identifiers
```

Unique constraints are especially important for concurrency-sensitive workflows.

---

# Check Constraints

Check constraints should be used for simple data invariants that PostgreSQL can enforce reliably.

Examples include:

```text
amount >= 0
end_time > start_time
supported status values
supported OAuth providers
```

Business workflows that require multiple records or complex logic remain service-layer responsibilities.

---

# Money

Financial amounts must not use floating-point database types.

Use fixed-precision decimal or integer minor units according to the financial design of the specific subsystem.

The chosen approach must support:

- exact calculations
- currency awareness
- predictable rounding
- transaction safety

Currency must not be assumed globally.

---

# Text Data

Text columns should use appropriate limits and validation based on domain requirements.

User-provided text must be validated at the application boundary.

Database storage type does not replace application-level validation.

---

# JSONB

`jsonb` may be used for genuinely variable or extensible data.

It must not become a replacement for normal relational modeling.

Use JSONB for data such as:

- provider-specific metadata
- controlled configuration
- extensible integration payloads
- structured snapshots where relational querying is not the primary requirement

Frequently queried business attributes should generally have explicit columns.

---

# Arrays

PostgreSQL arrays may be used where an array is genuinely the correct domain representation.

They should not be used to avoid designing relationships.

For example, many-to-many relationships normally require relational tables rather than arrays of IDs.

---

# Indexing

Indexes must support actual query patterns.

Do not index every column by default.

Important index candidates include:

- foreign keys
- frequently queried status fields
- tenant-scoped lookups
- time-range queries
- unique external identifiers
- pagination/sorting access paths

Indexes must be evaluated against:

- selectivity
- query frequency
- table size
- write cost
- storage cost

---

# Composite Indexes

Composite indexes must follow actual query patterns.

For multi-tenant data, the tenant or ownership column should normally be part of the relevant access path.

For OAuth identities, the uniqueness constraint:

```text
(provider, provider_subject)
```

is the primary lookup path for external identity resolution.

---

# Booking Data and Concurrency

Booking operations are concurrency-sensitive.

The database must help prevent double booking.

Correctness must not depend solely on:

```text
SELECT availability
then
INSERT booking
```

because concurrent requests can observe the same availability.

Booking creation must use appropriate transactions, locking, exclusion constraints, unique constraints, or other database-safe mechanisms as dictated by the booking model.

---

# Transaction Boundaries

Transactions should be used when multiple database operations must succeed or fail together.

Examples include:

- booking creation
- payment state transitions
- account linking
- subscription state changes
- session-related security changes
- import commits
- multi-record domain updates

Transactions should remain as small as practical.

Long-running external network operations should not unnecessarily hold database transactions open.

---

# Isolation and Concurrency

The default PostgreSQL transaction behavior should be used unless the domain requires stronger guarantees.

Where concurrency matters, the implementation must explicitly consider:

- race conditions
- locking
- unique constraints
- serialization failures
- retry behavior
- idempotency

Correctness must be demonstrated rather than assumed.

---

# Idempotency

Operations that may be retried by clients, workers, providers, or infrastructure should support idempotency where required.

Potential examples:

- booking creation
- payment operations
- webhook processing
- import commits
- notification jobs

Idempotency keys and uniqueness constraints must be designed together.

---

# Authentication and Session Data

Authentication tables must remain separate from general profile or business domain tables.

Core relationships:

```text
users
├── user_credentials
├── user_oauth_identities
└── user_sessions
```

Sessions must always resolve to a local user.

OAuth identities must always resolve to a local user.

Passwords must never be stored in the users table as plaintext.

---

# Multi-Tenancy

The platform is multi-tenant.

Tenant isolation must eventually be enforced server-side.

The frontend must never be trusted to choose an arbitrary tenant identifier for authorization.

Domain records that belong to businesses or tenants should normally contain an explicit ownership relationship appropriate to the domain.

Queries must scope tenant-owned data correctly.

---

# Tenant-Scoped Queries

Repositories for tenant-owned resources must receive trusted tenant context from the authenticated application layer.

The repository must not blindly trust arbitrary client-provided tenant IDs.

The authorization flow is conceptually:

```text
Authenticated User
        ↓
Authorized Membership
        ↓
Trusted Tenant Context
        ↓
Repository Query
        ↓
Tenant-Scoped Data
```

---

# Audit Data

Security-sensitive and business-critical actions may require audit records.

Audit data should capture sufficient context for investigation without storing unnecessary secrets.

Potential audit information includes:

```text
actor_user_id
action
entity_type
entity_id
timestamp
request_id
relevant metadata
```

Audit records should generally be append-oriented.

---

# Sensitive Data Protection

Sensitive information must be minimized.

Do not store data merely because it might become useful later.

Never store plaintext:

- passwords
- authentication tokens
- provider private keys
- OAuth client secrets

Do not log sensitive authentication material.

Encryption at rest and infrastructure-level database protection will be addressed as part of deployment and security hardening.

---

# External Provider Data

Provider data must be normalized into the application's data model.

External systems are not authoritative for:

- local roles
- local memberships
- business permissions
- customer permissions
- platform permissions

External IDs may be persisted where needed for integration or identity resolution.

---

# Audit and Historical Data

Records that affect financial, booking, messaging, or security history may require retention even after the related entity is deactivated.

Do not physically delete historical data merely because an active UI entity no longer appears.

Retention and deletion policies will be finalized as the affected domains mature.

---

# Pagination

Large collections must use deliberate pagination strategies.

Offset pagination is acceptable for smaller administrative views where appropriate.

Cursor-based pagination should be considered for:

- high-volume bookings
- message threads
- activity logs
- audit records
- large CRM datasets

Pagination strategy must follow the query and product requirement.

---

# Search and Filtering

Filtering must be executed using database queries appropriate to the dataset size.

The application must avoid loading unnecessarily large datasets into application memory merely to filter them.

Search indexes or PostgreSQL-specific search features may be introduced later when actual product requirements justify them.

---

# Large Tables

Tables expected to grow significantly must be designed with future scale in mind.

Potential high-growth domains include:

- bookings
- messages
- audit events
- CRM activity
- payment events
- notification events

Partitioning must not be introduced prematurely.

It should be considered only after measurable workload characteristics justify it.

---

# Migrations

All schema changes must use versioned migrations.

Migrations must be:

- deterministic
- reviewable
- repeatable in order
- safe for deployment
- compatible with the application's expected state

Never make production schema changes manually without recording the corresponding migration.

---

# Migration Safety

Migrations must carefully consider:

- existing rows
- locks
- table size
- index creation cost
- backward compatibility
- rollback behavior
- deployment ordering

Destructive schema changes require particular care.

When practical, prefer:

```text
expand
migrate
verify
contract
```

rather than immediately removing structures still required by running application versions.

---

# SQL Generation

The project uses sqlc for typed query generation.

The expected flow is:

```text
SQL query
   ↓
sqlc
   ↓
Generated Go types/functions
   ↓
Repository
   ↓
Service
```

Generated sqlc code must not be manually edited.

Changes belong in SQL query files followed by regeneration.

---

# Repository Layer

Repositories provide the database-facing boundary for domain code.

Repositories should:

- execute prepared/generated queries
- map database results
- handle expected persistence errors
- expose domain-appropriate persistence operations

Repositories should not decide high-level authorization policy.

---

# Query Ownership

SQL queries should live in the appropriate domain query files.

For example:

```text
auth queries
booking queries
customer queries
messaging queries
payment queries
```

The exact package organization follows the existing Go project structure.

---

# External IDs

External systems may expose identifiers that differ from the local primary key.

Where required, store external IDs explicitly.

Examples include:

```text
oauth provider subject
payment provider customer ID
payment provider transaction ID
calendar provider event ID
messaging provider ID
```

External identifiers must not replace internal primary keys.

---

# Authentication External Identity

OAuth identity lookup uses:

```text
provider
provider_subject
```

not provider email.

This allows the local platform to remain stable even when provider profile information changes.

---

# Booking Data Integrity

The booking domain must preserve:

- time correctness
- timezone semantics
- resource constraints
- staff assignment rules
- booking conflicts
- status transitions
- cancellation rules

Database constraints should enforce simple invariants.

More complex scheduling decisions belong to the booking service.

---

# Messaging Data Integrity

Messaging should preserve:

- customer identity
- business identity
- conversation ownership
- message ordering
- timestamps
- delivery state where required

Messages must not be modeled as public multi-user groups when the intended domain is business-to-customer conversations.

---

# CRM Data Integrity

The CRM remains relational and database-backed.

Import and export features must not create a second source of truth outside PostgreSQL.

Import workflows should support:

```text
upload
preview
mapping
validation
deduplication
warnings
confirmation
commit
audit
```

Large imports should be transactional where practical and recoverable when full atomicity is not practical.

---

# Financial Data Integrity

Financial operations must preserve a clear audit trail.

Amounts, currencies, statuses, and provider identifiers must be stored in a way that avoids ambiguity.

Raw card data must never be stored by the platform.

Payment provider interactions must be modeled around provider references and verified state transitions.

---

# Outbox and Events

Reliable asynchronous processing may use an outbox pattern when required.

The outbox allows a database transaction to persist:

```text
domain state
+
event/outbox record
```

atomically.

A worker can then process the event after the transaction commits.

This prevents common dual-write failures.

---

# Caching

Caches are not authoritative.

If Redis or another cache is introduced later, cached values must always be reconstructible from the database or another explicitly authoritative source.

Cache invalidation must be treated as part of the affected domain design.

---

# Search Indexes

External search systems, if introduced later, are derived from PostgreSQL data.

Search indexes must not become the only durable copy of important domain information.

Rebuild procedures must exist before an external search system becomes operationally important.

---

# Backup

PostgreSQL backups are part of the platform's disaster-recovery strategy.

The backup architecture must eventually support:

- automated backups
- retention
- restore verification
- recovery objectives
- off-site protection
- documented recovery procedures

Backups must be treated as production infrastructure rather than an afterthought.

---

# Development Database

Local development uses PostgreSQL through Docker Compose.

Developers should be able to initialize the database from:

- migrations
- seed data where appropriate
- documented local configuration

Development data must not contain real customer credentials or production secrets.

---

# Testing

Database-related behavior must be tested at appropriate levels.

Potential layers include:

```text
unit tests
repository tests
integration tests
migration verification
concurrency tests
security tests
```

Important invariants should be tested at the database boundary rather than relying only on mocks.

---

# Current Authentication Schema

The current authentication architecture includes:

```text
users
user_credentials
user_sessions
user_oauth_identities
```

The OAuth identity model is:

```text
user_oauth_identities
├── user_id
├── provider
├── provider_subject
├── created_at
└── updated_at
```

The database enforces the identity relationship through foreign-key and uniqueness constraints.

---

# Current Phase 3 Status

Phase 3 is currently in progress.

Completed database-related authentication work includes:

- authentication schema foundation
- password credential model
- session data model
- OAuth identity migration
- OAuth identity repository integration

The remaining authentication work includes:

- authentication service
- session integration
- registration
- login
- logout
- authentication middleware
- `/me`
- RBAC enforcement
- security verification
- Phase 3 closure

---

# Architectural Principles

The database must remain:

```text
The source of truth
Consistent
Transactionally safe
Tenant-aware
Security-conscious
Migration-driven
Query-efficient
```

Application architecture must work with PostgreSQL rather than attempting to replace relational integrity with application-only conventions.
