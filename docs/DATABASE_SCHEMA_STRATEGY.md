# Database Schema Strategy

## Purpose

This document defines the strategy for designing and evolving the PostgreSQL schema for the Universal Booking Platform.

The schema must support a secure, multi-tenant, universal booking platform without sacrificing:

- data integrity
- clear ownership
- transactional correctness
- extensibility
- maintainability
- predictable performance
- safe migrations

PostgreSQL remains the system of record.

---

# Core Schema Principles

The database should be:

```text id="s1a2b3"
Normalized
Explicit
Constraint-driven
Transaction-safe
Tenant-aware
Migration-driven
Query-oriented
```

The schema should model actual domain relationships rather than encoding important business data into loosely structured blobs.

---

# Source of Truth

PostgreSQL is authoritative for durable application state.

The following must not become independent sources of truth:

- frontend state
- browser storage
- cache
- search indexes
- background-job payloads
- external provider responses

Derived systems may duplicate data for performance, but they must remain reconstructible from authoritative state.

---

# Domain Modeling

The schema should model stable business concepts explicitly.

Examples include:

```text id="c2d3e4"
users
businesses
customers
services
staff
resources
locations
bookings
messages
subscriptions
payments
```

Each entity should have:

- a clear purpose
- a defined owner
- explicit relationships
- appropriate lifecycle behavior

---

# Normalization

Use relational normalization for durable business data.

Do not duplicate data unnecessarily when the duplicated value can become inconsistent.

Denormalization may be introduced when there is a measured performance requirement or when preserving a historical snapshot is necessary.

Any intentional denormalization should have a documented reason.

---

# Identity Model

The application uses a local identity model.

The core relationship is:

```text id="d4e5f6"
users
├── user_credentials
├── user_oauth_identities
└── user_sessions
```

The local `users.id` is the authoritative application identity.

---

# Authentication Identity Separation

Different authentication concepts must remain separate.

```text id="g7h8i9"
users.id
    ↓
local application identity

user_credentials
    ↓
password authentication

user_oauth_identities
    ↓
external provider authentication

user_sessions
    ↓
authenticated application state
```

No external provider becomes the owner of the application's user identity.

---

# OAuth Identity Strategy

External OAuth identities are modeled relationally.

The identity is:

```text id="j0k1l2"
(provider, provider_subject)
```

The schema must enforce:

```text id="m3n4o5"
UNIQUE(provider, provider_subject)
```

The current model also enforces:

```text id="p6q7r8"
UNIQUE(user_id, provider)
```

This creates a clear one-to-one relationship between a local user and each supported provider under the initial product requirements.

---

# OAuth Email Strategy

Provider email is not the external identity key.

The schema must not identify an external account using only:

```text id="s9t0u1"
email
```

Reasons include:

- provider email behavior can differ
- provider email can change
- Apple may provide private relay addresses
- multiple authentication mechanisms may expose the same email
- email matching alone does not prove account ownership

Email remains an account attribute.

---

# Account Linking Strategy

Explicit account linking is modeled by attaching an OAuth identity to an existing local user.

The database ensures that:

```text id="v2w3x4"
one external identity
→ one local user
```

The service layer ensures that:

```text id="y5z6a7"
only an authorized user
→ can link an identity
```

Database constraints and service authorization work together.

---

# Primary Keys

Entities should use stable UUID primary keys.

Primary keys must remain independent from:

- display names
- email addresses
- external provider identifiers
- tenant slugs
- sequential UI numbering

A primary key is an internal identifier, not an authorization mechanism.

---

# External Identifiers

External provider identifiers should be stored separately from local primary keys.

Examples:

```text id="b8c9d0"
provider_subject
payment_customer_id
payment_event_id
calendar_event_id
```

An external ID must only replace a local ID when the domain explicitly requires that behavior.

---

# Ownership

Every major entity should have a clearly defined ownership model.

Examples:

```text id="e1f2g3"
platform-owned
business-owned
user-owned
tenant-owned
booking-owned
conversation-owned
```

Ownership must be represented explicitly when it affects authorization or data isolation.

---

# Multi-Tenancy Strategy

The platform is designed as a multi-tenant system.

Tenant-owned records should have an explicit path back to their tenant or business context.

Conceptually:

```text id="h4i5j6"
Authenticated User
       ↓
Membership / Authorization
       ↓
Business / Tenant
       ↓
Domain Record
```

The application must never rely solely on a client-supplied tenant ID.

---

# Tenant Isolation

Tenant isolation is enforced primarily in server-side application logic and database query scope.

A tenant-owned repository query should operate against trusted tenant context.

For example:

```text id="k7l8m9"
trusted business_id
        ↓
repository query
        ↓
business-owned records
```

Cross-tenant data access must fail securely.

---

# Relationships

Relationships should use foreign keys.

Examples:

```text id="n0o1p2"
booking → customer
booking → business
booking → service
booking → staff
booking → resource
```

Foreign keys protect referential integrity.

---

# One-to-Many Relationships

A one-to-many relationship should normally be represented by a foreign key on the child record.

Example:

```text id="q3r4s5"
users
  ↓
user_sessions.user_id
```

---

# Many-to-Many Relationships

Many-to-many relationships should normally use an explicit join table.

Examples may include:

```text id="t6u7v8"
business_staff
service_staff
booking_resources
user_business_memberships
```

Join tables may contain their own attributes when required.

---

# Historical Snapshots

Some domains require historical copies of data.

Bookings and financial records may need to preserve information as it existed when an event occurred.

For example, a booking may need to retain the price or service name that was valid at booking time even if the service later changes.

This is intentional historical denormalization, not accidental duplication.

---

# Time Modeling

Time-sensitive data must distinguish:

```text id="w9x0y1"
instant
duration
local date
local time
timezone
```

Booking records must preserve enough information to correctly interpret scheduled events across timezones and DST transitions.

The database should store instants using timezone-aware timestamp types.

---

# Business Timezones

Businesses and locations should have explicit timezone configuration where required.

Business hours must not be interpreted using the server's local timezone.

Availability calculations must combine:

- business timezone
- local business hours
- holidays
- staff availability
- resource availability
- existing bookings
- buffers
- lead time

---

# Status and Lifecycle

Entities should model lifecycle explicitly.

Examples:

```text id="z2a3b4"
pending
active
confirmed
cancelled
completed
disabled
archived
```

Statuses should have documented transition rules.

A status column must not become a replacement for an actual relationship when the domain requires separate records.

---

# Soft Deletion Strategy

Soft deletion should be used only where historical context or restoration justifies it.

Potential candidates include:

- businesses
- customers
- services
- staff
- resources

Some data should be immutable or retained instead of soft-deleted, especially financial and audit history.

---

# Security-Critical Data

Authentication data should be modeled separately from ordinary profile data.

Never store:

```text id="c5d6e7"
plaintext passwords
OAuth client secrets
provider private keys
raw authorization codes
unnecessary access tokens
unnecessary refresh tokens
```

Sensitive transient values should remain transient wherever possible.

---

# Authentication Schema

The authentication schema consists conceptually of:

```text id="f8g9h0"
users
user_credentials
user_oauth_identities
user_sessions
```

The database must enforce the relationships between these tables.

---

# OAuth Identity Table

The OAuth table contains:

```text id="i1j2k3"
id
user_id
provider
provider_subject
created_at
updated_at
```

Constraints:

```text id="l4m5n6"
PRIMARY KEY(id)
FOREIGN KEY(user_id)
UNIQUE(provider, provider_subject)
UNIQUE(user_id, provider)
supported provider constraint
```

Indexes should support:

```text id="o7p8q9"
user_id
provider + provider_subject
```

---

# Data Integrity Strategy

Important invariants should exist at multiple layers.

```text id="r0s1t2"
Client validation
        ↓
Application validation
        ↓
Service logic
        ↓
Repository
        ↓
Database constraints
```

The database remains the final integrity boundary for constraints it can reliably enforce.

---

# Check Constraints

Use check constraints for simple rules.

Examples:

```text id="u3v4w5"
amount >= 0
start_at < end_at
provider in supported values
```

Complex business workflows remain in application services.

---

# Unique Constraints

Use database uniqueness wherever duplicate values would violate domain integrity.

Authentication examples:

```text id="x6y7z8"
UNIQUE(provider, provider_subject)
UNIQUE(user_id, provider)
```

Booking and tenant-specific uniqueness must be modeled similarly where the domain requires it.

---

# Index Strategy

Indexes should be driven by actual queries.

Before creating an index, consider:

- filtering
- sorting
- joins
- pagination
- selectivity
- write cost
- table growth

Do not create indexes simply because a column exists.

---

# High-Growth Domains

Potentially high-volume tables include:

```text id="a9b0c1"
bookings
messages
audit_events
notifications
payment_events
CRM activity
```

The initial schema should remain simple.

Partitioning, sharding, or specialized storage should only be introduced after workload evidence justifies it.

---

# Booking Schema Strategy

Booking records must support the concept of:

```text id="d2e3f4"
customer
business
service
staff
resource
location
start
end
status
```

Not every booking requires every relationship.

The schema should allow the universal booking model to support:

- staff-based appointments
- resource reservations
- room bookings
- equipment bookings
- event reservations
- service appointments

without forcing every business to use all scheduling dimensions.

---

# Availability Strategy

Availability is derived from multiple domain constraints.

The database should store authoritative source records such as:

```text id="g5h6i7"
business hours
staff schedules
resource schedules
holidays
bookings
buffers
lead-time rules
```

The booking service combines them to determine actual availability.

Availability must not be permanently stored as a giant generated calendar unless a future performance requirement justifies materialized availability.

---

# Messaging Schema Strategy

The messaging model should support:

```text id="j8k9l0"
business
customer
conversation
messages
staff assignment
booking context
internal notes
```

A conversation is not automatically a public multi-user group.

The schema should support one business communicating with many individual customers while preserving customer-specific conversation history.

---

# CRM Schema Strategy

CRM data remains relational.

Important concepts such as:

- customers
- contact methods
- notes
- tags
- activities
- imports
- deduplication results

should be modeled explicitly where they are core product features.

Spreadsheet-like presentation belongs to the frontend.

PostgreSQL remains the source of truth.

---

# Import Strategy

Import workflows should not immediately mutate production records.

The conceptual workflow is:

```text id="m1n2o3"
Upload
  ↓
Parse
  ↓
Preview
  ↓
Map
  ↓
Validate
  ↓
Detect duplicates
  ↓
Show warnings
  ↓
Confirm
  ↓
Commit
  ↓
Audit
```

Temporary import state may use dedicated tables or controlled temporary storage.

---

# Export Strategy

Exports should be generated from authoritative database records.

Exports must respect:

- tenant boundaries
- authorization
- selected fields
- pagination or streaming requirements
- sensitive-data policies

---

# Financial Schema Strategy

Financial records require stronger historical integrity than ordinary editable entities.

Important data should be recorded as immutable or append-oriented events when appropriate.

The system must retain enough information to reconstruct important financial state.

No raw card data should be stored.

---

# Subscription Schema Strategy

Subscriptions should model:

```text id="p4q5r6"
plan
subscription
entitlements
billing provider references
status
effective dates
```

Entitlements are derived from the subscription/plan model.

The database should not hard-code business limits into arbitrary application checks without a durable entitlement model.

---

# Audit Strategy

Critical actions should be traceable.

Potential audit targets include:

- authentication
- authorization changes
- account linking
- booking changes
- payment state changes
- imports
- administrative actions

Audit records should not contain authentication secrets.

---

# Event and Outbox Strategy

When durable domain state must trigger asynchronous processing, an outbox record may be written in the same database transaction.

Conceptually:

```text id="s7t8u9"
BEGIN
  update domain state
  insert outbox event
COMMIT
```

Workers consume the outbox after commit.

This prevents the common failure where database state succeeds but the corresponding event is lost.

---

# Cache Strategy

Caches are derived state.

A cache must not contain information that cannot be reconstructed from the authoritative source.

Redis, if introduced, should improve:

- performance
- rate limiting
- short-lived coordination
- job infrastructure

rather than replace PostgreSQL as the system of record.

---

# Search Strategy

Search engines, when eventually introduced, should consume authoritative application data.

Search indexes must be rebuildable.

The product must remain correct if a search index is temporarily unavailable.

---

# Migration Strategy

Every schema change must be represented as a versioned migration.

The migration process should be:

```text id="v0w1x2"
Design
  ↓
Migration
  ↓
Apply
  ↓
Regenerate sqlc
  ↓
Test
  ↓
Review
  ↓
Commit
```

Manual production schema edits should be avoided.

---

# Destructive Changes

Destructive changes require explicit review.

Examples:

```text id="y3z4a5"
drop column
drop table
change incompatible type
remove constraint
delete historical data
```

Prefer expand-and-contract migration patterns for live deployments.

---

# Backfills

Large data backfills must consider:

- transaction size
- locking
- write throughput
- application compatibility
- restartability
- progress tracking

A long-running backfill should not unnecessarily block normal application traffic.

---

# Generated Code Strategy

SQL remains the source for sqlc-generated database code.

The workflow is:

```text id="b6c7d8"
modify SQL
    ↓
run sqlc
    ↓
compile
    ↓
test
```

Generated files must not be edited manually.

---

# Schema and Repository Boundaries

The schema should remain independent from UI structure.

For example, a CRM spreadsheet component must not dictate a database table design merely because the UI is grid-based.

The database models business data.

The frontend models interaction.

---

# Performance Strategy

Performance optimization should follow evidence.

Prefer first:

```text id="e9f0g1"
correct schema
good indexes
efficient SQL
appropriate transactions
reasonable pagination
```

Only later consider:

```text id="h2i3j4"
caching
read replicas
partitioning
specialized search
additional infrastructure
```

Premature infrastructure complexity is discouraged.

---

# Scalability Strategy

The database must support growth without assuming that every component scales identically.

Future scaling may independently address:

- API workers
- background workers
- PostgreSQL
- cache
- object storage
- search
- messaging
- notification infrastructure

The schema should avoid coupling all scaling decisions to one provider-specific architecture.

---

# Backup and Recovery

The schema strategy must support:

- regular backups
- restore testing
- point-in-time recovery where available
- migration replay
- disaster recovery procedures

A backup is only useful when restoration has been verified.

---

# Testing Strategy

Schema testing should verify:

```text id="k5l6m7"
migrations
constraints
foreign keys
indexes
repository queries
transaction behavior
concurrency
tenant isolation
authentication identity rules
```

Important security and integrity rules should be tested at the database boundary as well as the service layer.

---

# Current Phase 3 Scope

Current schema-related Phase 3 work focuses on authentication.

Completed schema foundations include:

- users
- credentials
- sessions
- OAuth identities

Current OAuth identity model:

```text id="n8o9p0"
user_oauth_identities
├── user_id
├── provider
├── provider_subject
├── created_at
└── updated_at
```

Current uniqueness:

```text id="q1r2s3"
(provider, provider_subject)
(user_id, provider)
```

The remaining Phase 3 work is primarily service, session, endpoint, UI, authorization, and security integration.

---

# Design Decision Priority

When designing a new table or changing an existing table, use this priority:

```text id="t4u5v6"
1. Correct domain model
2. Security
3. Referential integrity
4. Transactional correctness
5. Tenant isolation
6. Query requirements
7. Performance
8. Operational simplicity
```

Avoid optimizing the schema for hypothetical scale before actual requirements justify it.

---

# Core Principle

The database should make invalid states difficult to represent.

Application code should enforce workflows.

PostgreSQL should enforce durable invariants.

Together they form the authoritative data layer of the Universal Booking Platform.
