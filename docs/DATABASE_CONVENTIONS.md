# Database Conventions

## Purpose

This document defines the concrete conventions for PostgreSQL tables and columns.

These conventions apply to future schema migrations unless a feature has a documented reason to deviate.

## Naming Convention

Database identifiers use lowercase snake_case.

Examples:

```text
users
businesses
business_id
created_at
updated_at
external_provider_id
```

Avoid:

```text
User
Business
businessId
CreatedAt
```

## Table Names

Use plural nouns for application entity tables.

Examples:

```text
users
customers
bookings
services
resources
messages
```

Join tables should clearly describe the relationship.

Examples:

```text
business_members
service_staff
booking_resources
```

## Column Names

Column names should be descriptive and consistent.

Preferred:

```text
created_at
updated_at
deleted_at
business_id
customer_id
```

Avoid unnecessary abbreviations.

## Primary Key Convention

Every persistent entity must have a primary key.

The project standard is:

```text
id
```

as the primary-key column name.

The exact PostgreSQL ID type must be chosen consistently for domain tables before the first production schema is introduced.

IDs must:

- remain stable
- not depend on mutable business values
- not contain authorization meaning
- be safe to expose through APIs when appropriate

## Foreign Key Convention

Foreign keys use the referenced table's singular entity name followed by `_id`.

Examples:

```text
business_id
customer_id
service_id
booking_id
user_id
```

Foreign keys should reference the corresponding primary key.

## Timestamps

Persistent entities should use:

```text
created_at
```

when creation time has business or operational value.

Use:

```text
updated_at
```

when the record can meaningfully change after creation.

Use PostgreSQL timezone-aware timestamps:

```text
timestamptz
```

Application-level time handling should use UTC as the primary storage convention.

## Lifecycle Timestamps

Additional timestamps should only be added when they represent meaningful domain events.

Examples:

```text
published_at
cancelled_at
completed_at
archived_at
```

Do not add event timestamps speculatively.

## Soft Deletion

When soft deletion is required, use:

```text
deleted_at
```

A NULL value means the record has not been soft-deleted.

Soft deletion is not required for every table.

## Boolean Columns

Boolean columns use PostgreSQL:

```text
boolean
```

Use names that clearly express the state.

Examples:

```text
is_active
is_verified
is_archived
```

Avoid storing boolean values as text.

## Status Columns

Use explicit status values for entities with meaningful lifecycle states.

Example:

```text
status
```

Valid status values must be defined by the relevant feature.

Do not allow arbitrary uncontrolled status values when the domain requires a fixed state machine.

## Nullable Columns

Use NULL only when NULL has a meaningful semantic interpretation.

A nullable column must have a clear application-level meaning.

Do not use NULL simply because the value is currently inconvenient to provide.

## Required Columns

Important required fields should use:

```text
NOT NULL
```

Database constraints must enforce critical invariants instead of relying exclusively on application validation.

## Unique Constraints

True uniqueness requirements must be enforced in PostgreSQL.

Examples:

```text
unique external identifiers
unique provider references
unique scoped names
```

Tenant-scoped uniqueness must include the appropriate tenant/business identifier when multi-tenancy is introduced.

## Foreign Key Constraints

Relationships should normally use foreign keys.

Deletion behavior must be explicitly chosen.

Do not use:

```text
ON DELETE CASCADE
```

automatically.

Historical, financial, audit, and booking records may require preservation.

## Check Constraints

Use check constraints for simple data invariants.

Examples:

```sql
CHECK (amount >= 0)
```

and:

```sql
CHECK (start_time < end_time)
```

Complex workflows belong in application logic.

## Numeric and Financial Values

Never use floating-point types for financial values.

Use an exact numeric representation appropriate to the business requirement.

Financial schema will be finalized during the authorized payments phase.

## Text Values

Use PostgreSQL text types when there is no real business requirement for a fixed length.

Do not create arbitrary limits without a reason.

When a maximum length is a real business rule, enforce it consistently.

## JSONB

Use JSONB only when the data genuinely requires flexible structure.

Appropriate examples may include:

```text
provider_metadata
configuration
external_payload_snapshot
```

Do not use JSONB as a substitute for relational modeling when the data requires:

- relationships
- constraints
- frequent filtering
- independent lifecycle management

## Arrays

Avoid PostgreSQL arrays for independently meaningful relationships.

Prefer join tables when individual elements need:

- querying
- constraints
- relationships
- lifecycle
- ownership

## Indexes

Indexes should support known query patterns.

Common candidates include:

```text
foreign keys
unique lookups
frequent filters
frequent sorting
composite query patterns
```

Do not automatically index every column.

## Index Naming

Use descriptive names.

Example:

```text
idx_bookings_business_id
idx_bookings_customer_id
idx_users_email
```

Unique indexes created by unique constraints may follow PostgreSQL-generated naming unless a named constraint is preferable.

## Composite Indexes

Composite indexes should match real query patterns.

Before creating one, identify:

- equality filters
- range filters
- ordering
- expected selectivity

Do not create composite indexes speculatively.

## Primary Key Indexes

PostgreSQL automatically creates the required index for primary keys.

Do not create duplicate indexes for the same primary key.

## Migration Naming

Migration files use:

```text
NNNNNN_description.up.sql
NNNNNN_description.down.sql
```

Example:

```text
000001_initial.up.sql
000001_initial.down.sql
000002_add_users.up.sql
000002_add_users.down.sql
```

Migration numbers must never be reused after being committed.

## Migration Scope

Each migration should have one clear purpose.

Prefer:

```text
000002_add_users
000003_add_businesses
```

over one large migration containing unrelated features.

## Migration Safety

Migrations must be reviewed before execution.

Pay particular attention to:

- destructive operations
- locking behavior
- large data changes
- indexes on large tables
- foreign-key validation
- backward compatibility

## SQL Formatting

SQL should be readable and consistently formatted.

Prefer explicit column lists.

Avoid:

```sql
SELECT *
```

in application queries where a defined column list is more appropriate.

## Generated Code

sqlc-generated code is generated output.

Do not manually edit generated files.

SQL definitions and queries remain the source of truth.

## Repository Convention

Repositories should operate on well-defined application/domain operations.

Do not expose arbitrary SQL execution to higher application layers.

## Schema Ownership

A table should have a clear owning feature or domain module.

Shared tables should be introduced only when their cross-feature role is clear.

## Domain Separation

Do not create tables merely because a future feature may need them.

A table should correspond to an authorized current requirement.

## Phase 2 Rules

The following are not being created yet:

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
subscriptions
```

Those tables belong to later authorized phases.

## Core Standard

Future schema work should follow:

```text
Consistent names
+
Stable IDs
+
UTC timestamps
+
Explicit constraints
+
Measured indexes
+
Safe migrations
```

The simplest schema that correctly represents the current requirement is preferred.
