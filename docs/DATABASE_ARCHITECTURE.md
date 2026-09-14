# Database Architecture

## Purpose

This document defines the database architecture and data conventions for the Universal Booking Platform.

The goal is to keep PostgreSQL reliable, maintainable, secure, and suitable for the platform's long-term growth.

## Database

PostgreSQL is the primary relational database and source of truth for application data.

The database should preserve strong relational integrity through:

- primary keys
- foreign keys
- unique constraints
- check constraints
- appropriate indexes
- transactions

## Database Ownership

Application data belongs in PostgreSQL unless there is a documented reason to use another storage system.

The application should not maintain a second authoritative copy of relational business data.

## Schema Organization

The initial system should avoid unnecessary PostgreSQL schemas.

Application tables should use the default application schema unless a real requirement justifies separate schemas.

Database organization should primarily come from clear table naming, relationships, and application modules.

## Table Naming

Table names should use:

- lowercase
- snake_case
- plural nouns where appropriate

Examples:

```text
users
businesses
customers
bookings
services
```

Avoid inconsistent naming such as:

```text
User
CustomerData
bookingTable
```

## Primary Keys

Every persistent entity should have a primary key.

IDs must be stable and must not depend on:

- array indexes
- usernames
- email addresses
- mutable business values

The exact ID strategy must be selected before domain tables are introduced.

## Foreign Keys

Relationships between tables should use foreign keys where appropriate.

Foreign keys provide database-level protection against invalid references.

Application-level validation does not replace database constraints.

## Referential Actions

Foreign-key deletion behavior must be deliberate.

Do not use cascading deletes automatically.

Before using:

```text
ON DELETE CASCADE
```

verify that deleting the parent should truly delete all dependent records.

Historical and financial data generally require stronger preservation rules.

## Timestamps

Persistent records should normally track creation time.

Where lifecycle changes matter, records may also track update time.

Use a consistent timestamp representation across the database.

The application should preserve timezone-aware meaning where required.

## Current Timestamp Convention

Database timestamp columns should use a consistent PostgreSQL timestamp type appropriate for UTC-based application storage.

The exact convention should be applied consistently to future tables.

## Soft Deletion

Soft deletion should not be applied automatically to every table.

Use soft deletion only where the business requirement requires:

- restoration
- historical visibility
- auditability
- preservation of references

Examples may include certain user, business, or configuration records.

Do not create `deleted_at` columns solely because they might be useful later.

## Status Fields

Status values should represent explicit business states.

Avoid storing uncontrolled arbitrary strings where a constrained set of states is required.

State-transition rules belong in application logic and should be reinforced by database constraints where practical.

## Money

Financial amounts must not use floating-point database types.

Use an exact representation appropriate to the supported currency model.

The exact financial schema will be finalized before payment and billing tables are introduced.

## Boolean Fields

Use PostgreSQL boolean values for true/false state.

Avoid storing boolean state as strings such as:

```text
"true"
"false"
"yes"
"no"
```

## NULL Usage

NULL should have a deliberate meaning.

Do not use NULL simply because a value is inconvenient to populate.

Document important nullable fields when their meaning is not obvious.

## Required Fields

Fields required for a valid record should normally be declared `NOT NULL`.

Do not depend solely on application validation for required database values.

## Unique Constraints

Values that must be globally unique should have database-level unique constraints.

Examples may include:

- identifiers
- external references
- certain configuration keys

Uniqueness requirements that depend on tenant scope should use the appropriate composite uniqueness strategy when multi-tenancy is implemented.

## Check Constraints

Use database check constraints for simple invariants that should never be violated.

Examples:

```text
amount >= 0
start_time < end_time
```

Do not place complicated application workflows into database check constraints.

## Indexes

Indexes should be created for actual query patterns.

Typical candidates include:

- foreign keys used in queries
- frequently filtered fields
- unique lookup values
- common sorting/filtering combinations

Do not create indexes for every column.

Each index adds storage and write overhead.

## Composite Indexes

Composite indexes should match real query patterns.

Column order matters.

Before creating a composite index, identify:

- equality filters
- range filters
- sort requirements
- expected query frequency

## Database Performance

Database performance should be based on measured query behavior rather than speculation.

Use:

- appropriate indexes
- query plans
- bounded queries
- pagination
- efficient joins

Avoid premature optimization.

## Pagination

Queries returning potentially large datasets must avoid unbounded result sets.

The application should introduce pagination for large collections.

Pagination strategy will be finalized when the first large collection feature is implemented.

## Transactions

Transactions should be used when multiple database changes must succeed or fail together.

The transaction boundary should generally be coordinated by the application/service layer.

Example:

```text
Begin transaction
    ↓
Operation A
    ↓
Operation B
    ↓
Commit
```

If any required operation fails:

```text
Rollback
```

## Isolation

Use PostgreSQL's transaction isolation behavior appropriately for the operation.

Higher isolation levels should be introduced only where the business operation requires them.

Do not increase isolation globally without evidence.

## Concurrency

The database must remain the final protection against race conditions that affect data integrity.

Application-level checks alone are insufficient for operations where concurrent requests can create invalid state.

Booking and resource conflicts will receive dedicated concurrency rules when that feature is implemented.

## Migration System

All schema changes must be version-controlled through migrations.

Migration files should:

- have deterministic ordering
- be committed to Git
- be reviewed before execution
- avoid destructive operations unless explicitly justified

## Migration Naming

Migration names should clearly indicate their order and purpose.

Example:

```text
000001_initial.sql
000002_add_example_table.sql
000003_add_example_index.sql
```

## Migration Safety

Migrations should be:

- deterministic
- repeatable through the migration system
- reviewed before execution
- tested when practical

Avoid combining unrelated schema changes into one migration.

## Destructive Migrations

Operations such as:

```text
DROP TABLE
DROP COLUMN
TRUNCATE
```

require explicit justification and careful review.

Do not use destructive migrations casually during development if they may later become part of shared history.

## Data Integrity

Database constraints should protect important invariants.

Application logic should provide the user-facing behavior, while PostgreSQL provides the final persistence-level integrity guarantees.

## Repository Boundary

Application code should access PostgreSQL through the repository/data-access layer.

Handlers should not execute SQL directly.

## SQL

SQL should be:

- parameterized
- readable
- reviewable
- safe from injection

Never concatenate untrusted input into SQL statements.

## sqlc

The project intends to use sqlc for typed SQL access.

SQL should remain explicit and reviewable rather than hiding important queries behind excessive abstraction.

## Generated Code

Generated database code should not be manually edited.

When the SQL schema or query changes, regenerate the code through the project-defined generation process.

## Data Access

Repositories should expose operations meaningful to the application rather than exposing arbitrary SQL execution to higher layers.

Avoid generic repositories that hide important business behavior.

## External Identifiers

External-provider IDs should be stored separately from internal primary keys when external systems are involved.

Examples may eventually include:

- payment-provider customer IDs
- payment-provider transaction IDs
- external calendar IDs
- messaging-provider IDs

Do not use external IDs as internal primary keys unless there is a clear architectural reason.

## Auditability

Records requiring historical traceability may need:

- timestamps
- actor identifiers
- status history
- audit records

Audit requirements should be defined alongside the relevant business feature.

## Data Retention

Data should not be retained indefinitely without a legitimate reason.

Retention policies must consider:

- product requirements
- operational needs
- privacy
- legal requirements

Specific retention rules will be defined for relevant future modules.

## Sensitive Data

Sensitive information stored in PostgreSQL should be minimized.

Do not store information merely because it may become useful in the future.

Sensitive fields may require additional protection depending on their purpose.

## Backup Considerations

Backups are intentionally outside the current implementation scope.

Production backup and disaster-recovery architecture will be defined during the authorized reliability and disaster-recovery phases.

## Local Development Database

The current local PostgreSQL environment is provided through Docker Compose.

Current configuration:

```text
Host: localhost
Port: 5432
Database: booking_db
User: booking_user
```

These credentials are for local development only.

## Phase 1 Boundary

This phase defines database architecture and conventions.

It does not implement:

- user tables
- business tables
- customer tables
- booking tables
- authentication tables
- payment tables
- messaging tables
- CRM tables
- production backups
- advanced partitioning
- database clustering

## Core Principle

The database should enforce important data integrity while remaining simple enough to evolve.

Prefer:

```text
Clear schema
+
Strong constraints
+
Explicit SQL
+
Measured indexes
+
Safe migrations
```

over unnecessary abstraction or speculative optimization.
