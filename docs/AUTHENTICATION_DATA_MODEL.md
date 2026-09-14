# Authentication Data Model

## Purpose

This document defines the database model required for authentication.

Authentication data is kept separate from the core `users` identity table where practical.

## Identity

The existing `users` table represents the platform identity.

```text
users
```

Authentication-specific data belongs in dedicated tables.

## User Credentials

Password credentials are stored separately from the main user record.

Conceptual structure:

```text
users
   │
   └── user_credentials
```

The credentials table should contain:

- credential ID
- user ID
- password hash
- creation timestamp
- update timestamp

It must never contain plaintext passwords.

## Credential Ownership

Each password credential belongs to exactly one user.

The database should enforce the relationship through a foreign key.

A user should have at most one active password credential in the initial authentication model.

## Password Hash

The password hash is the only password-related value that should be persisted.

The application must never store:

```text
plain_password
password
raw_password
```

## Password Hash Algorithm

The concrete hashing algorithm will be selected during authentication implementation.

The selected algorithm must:

- be designed for passwords
- use a unique salt
- support an appropriate work factor
- allow future work-factor upgrades where practical

## User Sessions

Authenticated browser sessions are stored server-side.

Conceptual structure:

```text
users
   │
   └── user_sessions
```

A session record should contain enough information to:

- identify the authenticated user
- securely identify the session
- determine expiration
- revoke the session
- audit session creation where required

## Session Secret Storage

The raw session credential should not be stored directly when a hashed session-token approach is used.

The preferred model is:

```text
Browser
   ↓
Raw session token
   ↓
Server-side hash lookup
   ↓
user_sessions
```

The exact token and hashing implementation will be selected during authentication implementation.

## Session Fields

The initial session model should conceptually contain:

```text
id
user_id
token_hash
expires_at
created_at
last_seen_at
revoked_at
```

Not every field is mandatory if the implementation can achieve the same security properties with less complexity.

## Session Expiration

Every session must have an expiration boundary.

Potential lifecycle:

```text
active
   ↓
expired
```

or:

```text
active
   ↓
revoked
```

Expired and revoked sessions must not authenticate requests.

## Session Revocation

A session can be revoked without deleting the historical session record.

A non-null:

```text
revoked_at
```

may represent explicit invalidation.

Examples:

- logout
- password change
- security incident
- administrator action

## Session Cleanup

Expired or revoked sessions may eventually be removed by a background cleanup process.

Cleanup is an operational concern and must not be required for authentication correctness.

## Foreign Keys

Both authentication tables must reference `users`.

Conceptual relationship:

```text
users.id
   ↑
   │
user_credentials.user_id

users.id
   ↑
   │
user_sessions.user_id
```

Deletion behavior must be explicitly selected.

Do not automatically cascade deletion of identity data when historical requirements are unknown.

## Unique Constraints

The credential table should enforce one active password credential per user.

The exact constraint should be chosen according to the final credential lifecycle model.

Session identifiers or token hashes must be sufficiently unique.

## Indexes

The session table will require an efficient lookup path for authentication.

Likely indexed fields include:

```text
token_hash
user_id
expires_at
```

Only indexes justified by actual query patterns should be created.

## Authentication Data Security

Authentication tables contain sensitive security data.

They must:

- never expose password hashes through API responses
- never expose session-token hashes
- never be included in logs unnecessarily
- use controlled repository access
- remain server-side

## User Email

The existing `users.email` field remains the platform identity's unique email value.

Authentication may use email to locate a user, but email itself is not a password credential.

## Account Status

Authentication may eventually need to distinguish whether an account can authenticate.

The final model should define an explicit account state rather than relying on deletion.

Possible states may include:

```text
active
disabled
```

The exact implementation should be introduced only when required.

## Verification State

Email verification is separate from password authentication.

If email verification is implemented, its state should be modeled explicitly rather than inferred from authentication behavior.

## Password Reset

Password reset data should not be stored as reusable plaintext secrets.

Reset tokens should be:

- short-lived
- single-use
- securely generated
- securely stored
- invalidated after use

A dedicated reset-token structure may be introduced when password reset is implemented.

## Authentication Auditability

Security-sensitive events may eventually require audit records.

Examples:

- password changed
- session revoked
- account disabled
- reset completed

Audit logging is distinct from normal authentication tables.

## Data Retention

Authentication records should be retained only as long as needed.

Session history may be retained temporarily for security or operational purposes.

Password-reset tokens should not remain valid after their intended lifetime.

## Migration Strategy

Authentication tables must be introduced using new versioned migrations.

Expected pattern:

```text
000003_create_user_credentials.up.sql
000003_create_user_credentials.down.sql

000004_create_user_sessions.up.sql
000004_create_user_sessions.down.sql
```

Each migration should have a focused purpose.

## Phase 3.2 Scope

This step authorizes creation of authentication data structures.

It does not authorize:

- registration endpoint
- login endpoint
- logout endpoint
- session middleware
- password hashing implementation
- password reset API
- email verification API
- RBAC
- authorization
- multi-tenancy

## Core Principle

Keep authentication data:

```text
Separate
Minimal
Server-controlled
Revocable
Securely indexed
```

while preserving the existing `users` identity model.
