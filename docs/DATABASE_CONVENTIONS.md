# Database Conventions

## Purpose

This document defines the concrete PostgreSQL conventions used by the Universal Booking Platform.

These conventions exist to keep the schema:

- predictable
- consistent
- secure
- maintainable
- migration-friendly
- compatible with Go and sqlc

When a domain-specific requirement conflicts with a convention, the domain requirement must be documented and the deviation should be intentional.

---

# Naming

## Tables

Use lowercase `snake_case` and plural nouns.

Examples:

```text
users
user_credentials
user_sessions
user_oauth_identities
businesses
services
bookings
messages
```

Avoid:

```text
User
UsersTable
userAccount
user-data
```

---

## Columns

Use lowercase `snake_case`.

Examples:

```text
user_id
business_id
created_at
updated_at
expires_at
provider_subject
```

Do not use camelCase or PascalCase in PostgreSQL identifiers.

---

# Primary Keys

The default primary key name is:

```text
id
```

Primary keys should use the project's standard UUID approach.

A primary key must be:

- unique
- stable
- immutable
- safe to expose as an identifier when appropriate

Do not use database row ordering as an application security boundary.

---

# Foreign Keys

Foreign key columns follow:

```text
<referenced_entity>_id
```

Examples:

```text
user_id
business_id
service_id
staff_id
resource_id
booking_id
```

Foreign keys should reference primary keys unless a documented domain requirement requires another unique key.

---

# Foreign-Key Constraints

Relationships should normally be enforced in PostgreSQL.

Example:

```sql
FOREIGN KEY (user_id) REFERENCES users(id)
```

Foreign-key behavior must be selected deliberately.

Do not automatically apply:

```sql
ON DELETE CASCADE
```

to every relationship.

Consider whether deletion should:

- cascade
- restrict
- set null
- archive the dependent record

Historical domains require particular care.

---

# Timestamps

Use:

```text
timestamptz
```

for persistent timestamps representing instants.

Common fields:

```text
created_at
updated_at
deleted_at
```

Other lifecycle timestamps should describe the event:

```text
expires_at
starts_at
ends_at
confirmed_at
cancelled_at
completed_at
revoked_at
```

Application-generated timestamps should normally originate from trusted server-side time.

---

# UTC

Persistent instants should be represented using UTC-capable timestamps.

The application must support explicit business and location timezones for scheduling.

Do not store a user's local clock time as though it were a universal instant.

---

# Lifecycle Timestamps

Most persistent records should include:

```text
created_at
updated_at
```

`deleted_at` should be used only when soft deletion is appropriate.

Additional lifecycle timestamps should be added when they represent meaningful business events.

Do not create timestamp columns merely for symmetry.

---

# Soft Deletion

Soft deletion uses:

```text
deleted_at
```

when the domain requires historical retention or restoration.

Soft deletion must not automatically be applied to all tables.

Authentication and financial data require specific deletion and retention policies.

Queries must explicitly exclude soft-deleted rows where required.

---

# Boolean Naming

Boolean fields should normally begin with:

```text
is_
has_
can_
```

Examples:

```text
is_active
is_verified
is_primary
has_access
```

Use a status column instead when the domain contains more than two meaningful states.

---

# Status Values

Status values should be:

- explicit
- documented
- finite
- consistently named

Examples:

```text
active
disabled
pending
confirmed
cancelled
completed
```

Do not create inconsistent synonyms such as:

```text
active
enabled
in_use
```

for the same conceptual state.

---

# Status Storage

For state machines, the database may use constrained text values or another explicit mechanism supported by the project.

The important requirement is that invalid states cannot silently enter the database.

Status transitions belong to the service/domain layer unless a simple invariant can safely be enforced directly by PostgreSQL.

---

# NULL

A column should be nullable only when `NULL` has an intentional semantic meaning.

Examples:

```text
deleted_at
revoked_at
completed_at
```

may legitimately be `NULL` when the corresponding event has not occurred.

Do not use `NULL` merely because application code might not always have a value.

Use `NOT NULL` for required domain data.

---

# UNIQUE Constraints

Use database-level unique constraints for values that must be globally unique within their defined scope.

Examples:

```text
UNIQUE(provider, provider_subject)
UNIQUE(user_id, provider)
```

For tenant-scoped data, uniqueness may require the tenant or business identifier.

Do not implement critical uniqueness exclusively in Go application logic.

---

# Authentication Identity Uniqueness

OAuth identities are identified by:

```text
(provider, provider_subject)
```

The database must enforce:

```sql
UNIQUE (provider, provider_subject)
```

The initial authentication design also enforces:

```sql
UNIQUE (user_id, provider)
```

The provider subject is the external identity key.

Provider email is not the database identity key.

---

# OAuth Provider Constraint

Supported OAuth providers should be restricted by a database check constraint where appropriate.

Initial supported values:

```text
google
apple
```

Example conceptual constraint:

```sql
CHECK (provider IN ('google', 'apple'))
```

Application validation must still be performed before persistence.

---

# Email

Email addresses are account attributes.

The application must define the canonicalization and uniqueness policy for email separately from OAuth identity handling.

Do not assume that:

```text
OAuth email = local user email
```

means that the OAuth identity automatically belongs to that local user.

Email matching alone must not trigger automatic account merging.

---

# Password Data

Passwords must never be stored in plaintext.

The database must store only a secure password hash.

Password credentials belong in:

```text
user_credentials
```

not directly in the general `users` table when the credential architecture separates identity from credentials.

---

# Session Data

Application sessions belong to local users.

Conceptually:

```text
user_sessions.user_id → users.id
```

Session secrets must not be stored or exposed unnecessarily.

The raw session token should not be persisted when the selected session architecture permits secure hashed-token lookup.

---

# OAuth Transaction Data

Temporary OAuth authorization state is not treated as durable relational identity data.

Values such as:

```text
state
nonce
PKCE verifier
```

are short-lived authentication transaction data.

The current architecture protects the OAuth transaction through an encrypted, short-lived browser cookie.

These values must never become long-lived application records without an explicit architectural reason.

---

# Text

Use PostgreSQL text types based on actual domain requirements.

Do not impose arbitrary length limits merely to make the schema appear strict.

Length and content validation should occur at the application boundary when required.

Database constraints should be used when the rule is important to data integrity.

---

# JSONB

Use `jsonb` for genuinely variable structured data.

Good candidates include:

- provider-specific metadata
- controlled configuration
- extensible integration payloads
- structured snapshots

Do not use `jsonb` to avoid modeling a stable relational entity.

Frequently queried or authorization-critical properties should normally receive explicit columns.

---

# Arrays

Use arrays only when the domain is naturally array-shaped.

Do not store related entity IDs in arrays when the relationship should be normalized.

Prefer relationship tables for:

```text
many-to-many
one-to-many collections requiring independent attributes
queryable relationships
```

---

# Monetary Values

Never use floating-point types for money.

Use the financial subsystem's chosen exact representation.

The representation must support:

- exact arithmetic
- explicit currency
- deterministic rounding
- safe persistence

Currency must always be explicit where monetary values are stored.

---

# Foreign-Key Indexes

Foreign keys should be evaluated for indexing based on expected query patterns.

High-volume relationships normally require indexes supporting:

```text
WHERE user_id = ?
WHERE business_id = ?
WHERE booking_id = ?
```

Do not assume the existence of a foreign key automatically means the correct application query is efficient.

---

# Index Naming

Indexes should use a predictable convention.

A typical pattern is:

```text
<short_table_name>_<columns>_idx
```

Examples:

```text
users_email_idx
user_sessions_user_id_idx
user_oauth_identities_user_id_idx
```

Unique indexes created from constraints should follow the project's migration tooling conventions.

---

# Composite Indexes

Composite indexes must reflect actual access patterns.

The column order matters.

Examples:

```text
(provider, provider_subject)
(business_id, starts_at)
(business_id, status)
```

Do not add a composite index solely because several columns often appear in a table.

---

# OAuth Lookup Index

The primary OAuth identity lookup is:

```text
provider + provider_subject
```

The uniqueness constraint on these columns provides the required lookup structure.

A separate duplicate index should not be created unless query analysis demonstrates a need.

---

# Pagination Indexes

Indexes supporting pagination should match the ordering strategy.

For example, if a query uses:

```text
WHERE business_id = ?
ORDER BY created_at DESC
```

an index may be designed around:

```text
business_id
created_at
```

The exact index should follow real query patterns.

---

# Database Check Constraints

Use check constraints for simple invariants.

Examples:

```text
amount >= 0
end_at > start_at
provider in supported values
```

Do not encode complex workflow logic into large database constraints when that logic belongs to the service layer.

---

# SQL Formatting

SQL should remain consistently formatted and readable.

Prefer:

```sql
SELECT
    id,
    user_id,
    provider,
    provider_subject
FROM user_oauth_identities
WHERE provider = $1
  AND provider_subject = $2;
```

Avoid dense one-line SQL for non-trivial queries.

---

# SQL Query Files

SQL queries should be kept in the project's designated sqlc query files.

A query should have:

- a clear name
- predictable parameters
- explicit selected columns
- only the joins actually required

Avoid:

```sql
SELECT *
```

in production application queries.

Explicit columns make schema changes safer and generated code more predictable.

---

# Generated Code

sqlc-generated code must not be manually edited.

Modify the SQL query files, then regenerate.

Generated files should be treated as build artifacts derived from the source SQL.

---

# Migration Naming

Migrations must use the project's established versioned migration convention.

Migration names should describe the schema change.

Examples:

```text
000005_create_user_oauth_identities
000006_add_booking_indexes
```

Avoid vague names such as:

```text
fix_db
changes
update
misc
```

---

# Migration Safety

Every migration must be evaluated for:

- existing data
- lock duration
- index creation cost
- backward compatibility
- deployment ordering
- rollback requirements

Destructive migrations require explicit consideration.

---

# Expand and Contract

For changes affecting production data, prefer:

```text
expand
migrate
verify
contract
```

when backward compatibility is required between application versions.

Example:

```text
Add new column
   ↓
Deploy code that can use both forms
   ↓
Backfill
   ↓
Verify
   ↓
Remove old structure later
```

---

# Repository Conventions

Database access should normally flow through:

```text
service
  ↓
repository
  ↓
sqlc
  ↓
PostgreSQL
```

Repositories should not contain product-level authorization policy.

Services should not construct arbitrary SQL when a repository/query already represents the required persistence operation.

---

# Transaction Conventions

Use a transaction when multiple related writes form one logical operation.

Examples:

```text
create booking + related records
link OAuth identity + update related state
commit CRM import
record payment state transition
write outbox event with domain state
```

Avoid keeping a transaction open while performing slow external network calls.

---

# Concurrency

When a write can race with another request, correctness must be enforced through appropriate database mechanisms.

Use combinations of:

- unique constraints
- transactions
- row locks
- exclusion constraints
- appropriate isolation
- retries

depending on the domain.

---

# Authentication Concurrency

OAuth identity creation is concurrency-sensitive.

The database uniqueness rule:

```text
UNIQUE(provider, provider_subject)
```

is the final integrity boundary.

The service must safely handle a uniqueness violation if two authentication flows attempt to attach the same provider identity concurrently.

---

# Tenant Scoping

Tenant-owned tables should explicitly represent ownership where required.

Queries against tenant-owned resources must be tenant-scoped.

Do not rely on the frontend to provide a trusted tenant identifier.

The authenticated server-side context must determine the effective tenant.

---

# Audit Columns

When appropriate, domain tables may include fields such as:

```text
created_by
updated_by
```

These should be introduced only when the domain benefits from actor attribution.

Security audit events should use dedicated audit structures rather than overloading every domain table with excessive metadata.

---

# Data Retention

Retention rules must be explicit for:

- authentication records
- bookings
- messages
- customer records
- financial data
- audit records

Do not delete data automatically without understanding downstream relationships and retention requirements.

---

# Sensitive Data

Never store secrets merely because the database can.

Sensitive data should be:

- minimized
- protected
- access-controlled
- excluded from logs
- deleted when no longer required

Never store:

```text
plaintext passwords
OAuth client secrets
OAuth private keys
authorization codes
temporary PKCE verifiers beyond their required lifetime
unnecessary provider tokens
```

---

# Authentication Table Relationships

The expected authentication relationship is:

```text
users
├── user_credentials
├── user_oauth_identities
└── user_sessions
```

All three dependent records reference the local user.

The external provider does not become a foreign-key authority for application identity.

---

# Database Source of Truth

All durable application state must ultimately be reconstructible from PostgreSQL and its explicitly defined durable companions.

Caching, search indexes, queues, and frontend state are derived or transient unless explicitly designated otherwise.

---

# Development Database Rules

Local PostgreSQL is provided through Docker Compose.

Development schema state must be reproducible from migrations.

Local secrets and credentials must not be committed.

Production database credentials must never be copied into development configuration.

---

# Testing Conventions

Database behavior should be validated through appropriate tests.

Important cases include:

- migrations apply successfully
- migrations roll forward correctly
- foreign keys reject invalid relationships
- unique constraints reject duplicates
- OAuth identity uniqueness works
- disabled users cannot establish authenticated state
- concurrent writes preserve invariants
- repository queries return expected results

---

# Current Phase 3 Authentication Conventions

The current authentication data conventions include:

```text
users
user_credentials
user_sessions
user_oauth_identities
```

OAuth identity rules:

```text
provider = google | apple
provider_subject = stable external identity
UNIQUE(provider, provider_subject)
UNIQUE(user_id, provider)
```

The full authentication lifecycle remains under Phase 3 implementation.

---

# Convention Priority

When applying these conventions, the priority is:

```text
1. Security
2. Data integrity
3. Domain correctness
4. Transactional correctness
5. Maintainability
6. Performance
7. Convenience
```

A shorter schema is not automatically a better schema.

The database should enforce important invariants rather than relying exclusively on developer discipline.
