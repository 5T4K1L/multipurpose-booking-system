# Authentication Data Model

## Purpose

This document defines the database model used by the authentication system.

The model separates:

- local user identity
- password credentials
- external OAuth identities
- application sessions
- account lifecycle state
- authentication-related metadata

The database is the source of truth for local authentication state.

---

# Identity Model

The core authentication relationship is:

```text id="7q2v8p"
users
├── user_credentials
├── user_oauth_identities
└── user_sessions
```

The `users` table represents the local platform identity.

Authentication mechanisms attach to the local user rather than creating separate application identities.

---

# Users

The `users` table is the authoritative local identity record.

Conceptual fields include:

```text id="7h5x2p"
id
email
status
email_verified_at
created_at
updated_at
deleted_at
```

The exact production schema remains governed by the Phase 2 database implementation and migrations.

The user ID is the stable internal identity used throughout the application.

It must not be replaced by:

- email address
- username
- OAuth provider subject
- provider email

---

# User Identity Rules

The internal user ID is the primary identity reference.

Email is an account attribute.

OAuth provider subject is an external identity attribute.

These values have different responsibilities:

```text id="3hxq1a"
Local User ID
    → internal application identity

Email
    → account/contact attribute

(provider, provider_subject)
    → external authentication identity
```

Email must not be used as the permanent foreign-key relationship between an external provider and a local user.

---

# User Credentials

Password credentials are stored separately from the user record.

Conceptual table:

```text id="9x8z1w"
user_credentials
```

Conceptual fields include:

```text id="0d7z7d"
id
user_id
password_hash
created_at
updated_at
```

The table must never store a plaintext password.

The stored password value is an Argon2id hash.

---

# Password Credential Relationship

A credential belongs to one local user.

Conceptually:

```text id="2w7n8q"
users.id
    ↓
user_credentials.user_id
```

The foreign key must protect referential integrity.

Deleting or deactivating a local user must not leave an authentication credential that can be used independently.

---

# OAuth Identities

External OAuth identities are stored separately from password credentials.

Conceptual table:

```text id="y7jv0r"
user_oauth_identities
```

Conceptual fields:

```text id="m19jz6"
id
user_id
provider
provider_subject
created_at
updated_at
```

This table represents the relationship between a local user and an external authentication provider.

---

# OAuth Provider

The `provider` identifies the external identity system.

Initially supported providers are:

```text id="l8h8u5"
google
apple
```

The database should restrict provider values to supported values.

Application code must also validate provider values before persistence.

Unsupported providers must not be silently accepted.

---

# OAuth Provider Subject

`provider_subject` stores the stable subject identifier returned by the provider.

This is the authoritative external identity key.

The external identity is therefore:

```text id="4uxk6m"
(provider, provider_subject)
```

The provider subject must not be replaced by:

- email
- display name
- profile name
- avatar URL
- a provider-specific temporary authorization code

---

# OAuth Uniqueness

The database must enforce:

```text id="p7w53z"
UNIQUE(provider, provider_subject)
```

This prevents one external provider identity from belonging to multiple local users.

Example:

```text id="4bq9x0"
google + subject_123
```

can reference only one local user.

---

# User-to-Provider Uniqueness

The initial architecture also enforces:

```text id="1c8x9v"
UNIQUE(user_id, provider)
```

This means a local user may initially have at most one Google identity and one Apple identity.

Example:

```text id="v0c8w2"
user A → google account A
user A → apple account A
```

is valid.

But:

```text id="9m5c6h"
user A → google account A
user A → google account B
```

is not valid under the current model.

Supporting multiple external identities from the same provider would require an explicit architecture change.

---

# OAuth Identity Foreign Key

Every OAuth identity must reference an existing local user.

Conceptually:

```text id="v8n1k3"
user_oauth_identities.user_id
        ↓
users.id
```

The foreign key prevents orphaned OAuth identities.

Application code must not create an OAuth identity pointing to a nonexistent user.

---

# OAuth Identity Indexing

At minimum, the database should support efficient lookup by:

```text id="2h6p6t"
provider + provider_subject
```

and:

```text id="9c2n5w"
user_id
```

The unique constraint on `(provider, provider_subject)` also provides the required lookup structure for external identity resolution.

An index on `user_id` supports retrieving all linked external identities for a local user.

---

# OAuth Identity Email

The initial OAuth identity table does not need to treat provider email as the identity key.

Email can be obtained from the validated provider response and used by application logic when appropriate.

The application must not assume:

```text id="h3x8cr"
same email = same person = same local account
```

Email-based account association requires explicit application policy.

---

# Account Linking Model

Account linking is represented by attaching an OAuth identity to an existing local user.

Conceptually:

```text id="v9w4qr"
users
  │
  └── user_oauth_identities
          │
          ├── google
          └── apple
```

Linking must happen only after authenticated and validated application flow.

The database uniqueness constraints prevent the same provider identity from being attached to multiple users.

---

# Account Merge Protection

The database model must support the rule that accounts are not automatically merged solely because their emails match.

Example:

```text id="8k9v5r"
Existing local user:
user_id = 100
email = person@example.com

OAuth identity:
provider = google
provider_subject = abc123
email = person@example.com
```

The matching email alone does not authorize:

```text
google identity → user 100
```

The authentication service must decide whether the user is signing into an existing linked identity, creating a new account, or explicitly linking an identity.

---

# User Sessions

Application sessions are stored separately from user identity and credentials.

Conceptual table:

```text id="b0l7h8"
user_sessions
```

Conceptual fields include:

```text id="2w7q8y"
id
user_id
token_hash
created_at
expires_at
last_seen_at
revoked_at
```

The exact schema is governed by the implemented Phase 3 migrations.

---

# Session Ownership

Every application session belongs to one local user.

Conceptually:

```text id="2k7d3x"
user_sessions.user_id
        ↓
users.id
```

An application session must never belong directly to:

- an OAuth provider
- a provider subject
- an email address

OAuth authentication resolves to a local user before an application session is created.

---

# Session Revocation

Sessions must support explicit invalidation.

A session may be revoked because of:

- logout
- password security event
- administrator action
- account disablement
- suspicious authentication activity
- other security policy

A revoked session must not authenticate protected requests.

---

# Session Token Storage

The raw session token should not be stored directly when the session architecture uses hashed token lookup.

The server stores a secure representation suitable for verification.

The exact token generation and storage mechanism belongs to the session implementation.

Authentication secrets must never be returned through normal API responses.

---

# Account Lifecycle

The local user model must support account lifecycle states.

Examples include:

```text id="7b8z6m"
active
disabled
```

Additional states may be introduced when required by product behavior.

Authentication must respect the local account status.

A disabled user must not receive a new authenticated session.

---

# Email Verification

Email verification is a local account property.

The system may record:

```text id="2b0n4h"
email_verified_at
```

A provider may supply a verified email claim, but the application must decide how provider verification maps to its own email verification policy.

OAuth validation alone does not automatically define every application-level email policy.

---

# Password Reset Metadata

Password reset functionality may require temporary reset records or token metadata.

Any future reset-token data must be:

- short-lived
- single-use
- securely generated
- stored safely
- revocable
- excluded from logs

Reset tokens must not be stored or exposed as plaintext when the design permits secure hashing.

---

# Authentication Audit Data

Security-sensitive authentication events may require audit records or structured security logs.

Potential events include:

```text id="2t6w0c"
login_success
login_failure
logout
oauth_login
oauth_link
oauth_unlink
password_change
password_reset_request
password_reset_completion
session_revoked
account_disabled
```

Audit data must minimize sensitive information.

Authentication secrets must never be written to the audit trail.

---

# Sensitive Data

The authentication data model must protect highly sensitive values.

Never persist or log unnecessary copies of:

- plaintext passwords
- OAuth authorization codes
- OAuth state
- OAuth nonce
- PKCE verifier
- OAuth client secrets
- provider private keys
- access tokens when persistence is not required
- refresh tokens when persistence is not required
- raw session secrets

The current OAuth architecture does not require persistent provider access or refresh tokens.

---

# OAuth Transaction Data

Temporary OAuth transaction state is intentionally not modeled as a long-lived relational identity record.

The transaction contains short-lived authorization-flow information such as:

```text id="5v9l2a"
provider
state
nonce
PKCE verifier
creation time
```

The current implementation protects this transaction using an encrypted browser cookie.

This transaction is not a local user session.

---

# Migration Relationship

Authentication-related schema changes must be implemented through versioned PostgreSQL migrations.

The current OAuth identity table was introduced as a dedicated migration.

Conceptually:

```text id="h8q8b4"
users
    ↓
user_credentials
user_oauth_identities
user_sessions
```

Migration ordering must guarantee that referenced tables exist before dependent foreign keys are created.

---

# Referential Integrity

Authentication relationships must be enforced at the database level whenever practical.

The database must protect:

- OAuth identity → user
- credential → user
- session → user

Application validation remains necessary, but database constraints provide the final integrity boundary.

---

# Deletion and Retention

Authentication records must follow the platform's account lifecycle and retention policy.

Deleting or anonymizing a user must account for:

- credentials
- OAuth identities
- sessions
- authentication audit records
- related business or customer records

Authentication data must not be deleted in isolation when doing so would violate referential integrity or compliance requirements.

---

# Query and Repository Boundary

Authentication database access must follow the project's standard data-access architecture:

```text id="4a8w0w"
Authentication Service
        ↓
Repository
        ↓
sqlc-generated queries
        ↓
PostgreSQL
```

Authentication logic must not build arbitrary SQL strings from client input.

SQL remains the database source of truth.

---

# OAuth Repository Operations

The OAuth repository requires operations conceptually equivalent to:

```text id="c7x4p8"
CreateOAuthIdentity
GetOAuthIdentity
GetUserOAuthIdentity
DeleteOAuthIdentity
```

The repository must not decide whether an OAuth identity should be created.

Business and authentication policy belongs in the authentication service.

The repository performs data access and persistence only.

---

# Authentication Data Access Rules

Application code must not:

- trust client-supplied user IDs for identity
- update OAuth identities without authorization checks
- bypass repository boundaries
- construct provider identity keys from unvalidated user input
- use email as an implicit OAuth identity lookup key

Identity resolution must be deterministic and server-controlled.

---

# Concurrency and Uniqueness

Concurrent OAuth callbacks or account-linking attempts must not be allowed to create duplicate identities.

Database uniqueness constraints are the final protection.

Application code must correctly handle unique-constraint conflicts.

Example:

```text id="9p3k7v"
Request A → creates google/provider_subject X
Request B → creates google/provider_subject X
```

Only one operation may succeed.

The other must resolve the conflict safely rather than creating another identity.

---

# Authentication Data Model and Authorization

Authentication tables establish identity.

They must not become the primary source of business authorization.

Roles, organization membership, tenant membership, staff permissions, and business access belong to authorization and future multi-tenant domain models.

For example:

```text id="9x5q2s"
user_sessions
    ↓
local user
    ↓
authorization
    ↓
tenant / business / resource access
```

Authentication must not embed tenant IDs into external provider identity records as an authorization shortcut.

---

# Current Phase 3 Scope

The authentication data model is part of Phase 3.

Current work includes:

- local user identity
- password credential model
- session model
- OAuth identity model
- account status
- authentication-related lifecycle metadata

OAuth identity persistence and repository foundation are implemented.

The full authentication service and session flow remain in progress.

---

# Current OAuth Schema Status

The OAuth identity migration is implemented.

The current structure includes:

```text id="3m0z5j"
user_oauth_identities
├── id
├── user_id
├── provider
├── provider_subject
├── created_at
└── updated_at
```

Constraints include:

```text id="2p5k7n"
PRIMARY KEY (id)
FOREIGN KEY (user_id) REFERENCES users(id)
UNIQUE (provider, provider_subject)
UNIQUE (user_id, provider)
provider restricted to supported values
```

The final schema remains governed by the actual committed migration and generated sqlc model.

---

# Core Principles

The authentication data model must remain:

```text id="3n5z0j"
Normalized
Explicit
Server-controlled
Referentially consistent
Provider-independent
Secure by default
```

The local user is the application's identity.

Passwords are credentials.

OAuth identities are external authentication links.

Sessions represent authenticated application state.

Authorization remains a separate concern.
