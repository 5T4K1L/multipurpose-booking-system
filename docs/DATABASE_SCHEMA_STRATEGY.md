# Database Schema Strategy

## Purpose

This document defines the strategy for designing the application's PostgreSQL schema.

It builds on the database conventions defined during Phase 1 and provides the rules that will be followed when actual domain tables are introduced.

## Schema Design Principles

Database schemas should prioritize:

- data integrity
- clear relationships
- explicit constraints
- predictable naming
- efficient queries
- safe migrations
- maintainability
- minimal duplication
- appropriate normalization

The schema should represent the actual business model rather than hypothetical future requirements.

## Database as Source of Truth

PostgreSQL is the authoritative source of truth for persistent application data.

Application caches, search indexes, external systems, and frontend state must not become competing authoritative data stores.

## Normalization

Relational data should generally be normalized to reduce unnecessary duplication and update anomalies.

Denormalization may be introduced only when:

1. a real performance or architectural requirement exists
2. the access pattern is understood
3. the consistency implications are documented
4. the additional complexity is justified

Do not denormalize preemptively.

## Entities

A persistent business entity should normally have:

- a primary key
- required fields
- creation timestamp
- update timestamp when lifecycle changes matter
- appropriate uniqueness constraints
- appropriate foreign keys

Not every table requires identical columns.

Columns should exist because the entity actually requires them.

## Table Ownership

Each table should have a clear purpose and ownership within the application's domain.

Examples that may eventually exist include:

```text
users
businesses
customers
services
staff
resources
bookings
payments
messages
```

These tables are examples only.

They must not be created until their corresponding feature phase is authorized.

## Relationships

Relationships should reflect real domain ownership and cardinality.

Common relationship types include:

```text
one-to-one
one-to-many
many-to-many
```

Many-to-many relationships should generally use explicit join tables.

Example:

```text
users
   ↓
organization_memberships
   ↓
organizations
```

Do not encode many-to-many relationships through comma-separated strings or JSON arrays when relational integrity is required.

## Foreign Keys

Foreign keys should be used for relationships that require database-level referential integrity.

Every foreign key should have deliberate deletion and update behavior.

Do not automatically cascade deletes.

Historical, financial, and audit-related records generally require preservation.

## Primary Key Strategy

The project must use stable identifiers that are independent of mutable business information.

The exact identifier type will be selected before the first production domain tables are created.

The chosen strategy must be:

- consistent
- index-friendly
- safe for distributed application behavior
- suitable for API exposure
- independent of user-editable values

## External IDs

External provider identifiers must be stored separately from internal primary keys.

Example:

```text
internal_id
external_provider_id
```

An external system must not accidentally become the authority for internal relational identity.

## Tenant Scoping

Multi-tenancy will be introduced in its authorized phase.

When domain tables become tenant-owned, tenant boundaries must be represented explicitly in the schema where required.

Tenant isolation must be enforceable at the database/application access layer and must not depend solely on frontend behavior.

## Timestamps

Domain tables should use a consistent timestamp strategy.

Where appropriate:

```text
created_at
updated_at
```

Lifecycle-specific timestamps may also exist when they carry real business meaning.

Examples:

```text
cancelled_at
completed_at
archived_at
```

Do not create timestamp columns merely for consistency if the event has no meaningful use.

## Status Modeling

Stateful entities should use explicit state definitions.

Examples:

```text
draft
active
cancelled
completed
archived
```

The valid states and transitions must be defined by the relevant domain feature.

The database may enforce simple invariants, while application logic owns workflow transitions.

## Historical Data

Historical records should generally remain understandable after related operational data changes.

Avoid destructive operations that make historical records impossible to interpret.

Financial and booking history require particular care.

## JSON Data

JSON/JSONB should be used only when flexible structure is genuinely required.

Good candidates may include:

- provider-specific metadata
- extensible configuration
- externally defined payload snapshots

Do not use JSON to avoid designing a proper relational structure.

## Arrays

PostgreSQL arrays should not replace relational tables when the data represents an independently meaningful relationship.

Prefer relational modeling when:

- individual elements require querying
- elements have their own lifecycle
- elements have relationships
- elements require constraints

## Unique Constraints

Uniqueness must be enforced at the database level when it represents a true invariant.

Examples:

```text
unique external identifier
unique provider reference
unique scoped business identifier
```

The correct uniqueness scope must be defined explicitly.

For future tenant-scoped values, uniqueness may need to be:

```text
tenant_id + value
```

rather than globally unique.

## Check Constraints

Use check constraints for simple database-level invariants.

Examples:

```text
amount >= 0
```

or:

```text
start_time < end_time
```

Complex business workflows belong in application logic.

## Index Strategy

Indexes should be based on actual query patterns.

Potential index candidates include:

- foreign keys
- unique lookup fields
- frequently filtered columns
- common sort/filter combinations

Do not automatically index every column.

Indexes increase storage and write overhead.

## Query-Oriented Design

Schema design should consider how the application will query the data.

Before adding an index or denormalization, identify:

- common filters
- common joins
- sorting requirements
- expected result size
- query frequency

The schema should support predictable access patterns without prematurely optimizing for unknown workloads.

## Large Tables

Tables expected to grow significantly must be designed with future query performance in mind.

However, partitioning should not be introduced until actual scale or retention requirements justify it.

Do not partition tables solely because they might become large someday.

## Soft Delete Strategy

Soft deletion is not universal.

Use it only when the product needs:

- restoration
- historical visibility
- auditability
- retention of references

Permanent deletion may be appropriate when data has no continuing business value and deletion is permitted.

The relevant feature must define which behavior applies.

## Audit Data

Audit requirements should be considered for operations involving:

- security
- permissions
- administrative changes
- financial actions
- sensitive configuration
- important state transitions

Audit structures should be created when the corresponding feature is implemented.

## Financial Data

Financial records require special care.

Financial amounts must use exact numeric representations rather than floating-point values.

Financial records should preserve historical truth rather than being rewritten casually.

Payment-related schema will be designed during the authorized payments phase.

## Booking Data

Booking-related schema must eventually support the booking engine's consistency requirements.

The schema must protect against invalid concurrent states where database constraints can reasonably do so.

Booking tables are intentionally not being created in Phase 2.1.

## Messaging Data

Messaging schema must eventually distinguish between:

- conversation identity
- participants
- messages
- message metadata
- internal business notes
- system events

Messaging tables are intentionally deferred.

## CRM Data

CRM schema must support customer records without unnecessarily duplicating authoritative booking or payment data.

CRM tables are intentionally deferred.

## Import/Export Data

Import processing may eventually require temporary or staging tables.

Such structures should only be created when the import/export phase is authorized.

## Migration Strategy

Every schema change must be introduced through a version-controlled migration.

Example:

```text
000001_initial.sql
000002_add_users.sql
000003_add_businesses.sql
```

Each migration should have one clear purpose.

## Migration Ordering

Migration numbering must remain deterministic.

Never reuse a migration number that has already been committed.

Do not silently rewrite previously shared migration history.

## Migration Review

Before executing a migration, review:

- tables
- columns
- data types
- constraints
- indexes
- foreign keys
- delete behavior
- performance implications
- rollback implications
- security implications

## Destructive Changes

Destructive schema changes require special review.

Examples:

```text
DROP TABLE
DROP COLUMN
TRUNCATE
```

Before destructive changes, determine whether:

- existing data can be migrated
- historical records must be preserved
- dependent code must be updated
- a staged migration is safer

## Migration Transactions

Where PostgreSQL supports the required operation safely, migrations should use transactions to avoid partially applied changes.

Operations that cannot safely run inside a transaction must be identified and handled deliberately.

## Data Backfills

Large or risky data transformations should not be hidden inside ordinary schema-definition migrations when doing so could cause operational problems.

For significant backfills, consider separate controlled migration steps.

## Generated Database Code

The project intends to use sqlc for typed database access.

SQL remains the source of truth.

Generated code must not be manually edited.

## Database Naming

Use:

```text
lowercase_snake_case
```

for database identifiers.

Examples:

```text
created_at
business_id
service_id
external_provider_id
```

Avoid inconsistent abbreviations.

## Reserved Words

Avoid using PostgreSQL reserved words as table or column names.

Prefer clear alternatives when necessary.

## Nullable Columns

Nullable columns must have an intentional semantic meaning.

For every nullable field, the application should know what:

```text
NULL
```

actually means.

Do not use NULL merely as a substitute for unknown schema design.

## Data Types

Choose PostgreSQL data types based on actual semantics.

Examples:

```text
boolean
integer
bigint
numeric
text
timestamp
timestamptz
date
jsonb
```

Avoid storing structured values in text when a suitable native PostgreSQL type exists.

## Text Limits

Use explicit length constraints only when the business rule actually requires them.

Do not add arbitrary limits merely because they seem reasonable.

Application validation and database constraints should agree.

## Schema Safety

Database design must never assume that application code will always behave perfectly.

Important integrity rules should be enforced at the database level wherever practical.

## Phase 2 Scope

Phase 2 will establish the foundation required for future application schema.

Phase 2 does not authorize implementation of:

- authentication tables
- user accounts
- organizations
- businesses
- customers
- services
- staff
- resources
- bookings
- CRM
- messaging
- payments
- subscriptions
- billing
- marketplace
- production database infrastructure

Those belong to their authorized phases.

## Phase 2.1 Boundary

This step defines schema strategy only.

No domain tables should be created as part of this step.

## Core Principle

Design the database for the real product requirements that are authorized now.

Prefer:

```text
Explicit relationships
+
Strong constraints
+
Clear ownership
+
Safe migrations
+
Measured indexes
```

over speculative complexity.
