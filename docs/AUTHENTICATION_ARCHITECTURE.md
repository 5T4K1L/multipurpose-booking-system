# Authentication Architecture

## Purpose

This document defines the authentication architecture for the Universal Booking Platform.

Authentication establishes who a user is.

Authorization determines what an authenticated user is allowed to do.

These responsibilities must remain separate.

## Authentication Flow

The intended flow is:

```text
User
   ↓
Login credentials
   ↓
Credential verification
   ↓
Authenticated identity
   ↓
Session
   ↓
Authenticated request
```

## Identity

The `users` table represents the platform identity.

Authentication data must not be mixed unnecessarily into unrelated business entities.

A user may eventually participate in multiple product contexts, but authentication establishes the underlying platform identity first.

## Authentication Responsibilities

Authentication is responsible for:

- establishing identity
- verifying credentials
- creating authenticated sessions
- maintaining session state
- ending sessions
- credential recovery
- credential changes
- protecting authentication endpoints

Authentication is not responsible for:

- business permissions
- tenant access
- staff permissions
- customer/business roles
- booking authorization

Those belong to authorization and application-level access control.

## Password Authentication

The platform will support password-based authentication unless a later architectural decision changes this.

Passwords must never be stored in plaintext.

Only a secure password hash may be persisted.

The exact password-hashing algorithm and configuration will be selected during implementation.

## Password Requirements

Password policy must balance security and usability.

The implementation should prioritize:

- sufficiently long passwords
- secure password hashing
- resistance to brute-force attacks
- protection against credential stuffing
- no unnecessary arbitrary complexity rules

The exact password policy will be defined before password registration is implemented.

## Password Hashing

Password hashing must use a modern password-hashing algorithm designed specifically for passwords.

The implementation must:

- use a unique salt per password
- use an appropriate work factor
- make verification resistant to brute-force attacks
- support future work-factor upgrades

Raw passwords must never be logged or returned by the API.

## Login

A login operation should:

1. validate the request
2. locate the identity
3. verify the password hash
4. apply authentication-abuse protections
5. create an authenticated session
6. return the appropriate authenticated state

Authentication failures should not unnecessarily reveal whether an account exists.

## Account Enumeration

Authentication responses should avoid exposing whether a specific email address is registered when doing so would enable account enumeration.

Where appropriate, externally visible responses should remain generic.

## Session Strategy

The application will use server-controlled authenticated sessions.

The session mechanism should prioritize:

- secure storage
- revocation
- expiration
- rotation where appropriate
- protection against theft
- minimal client exposure

The exact implementation will be finalized during the authentication implementation step.

## Browser Session Storage

For browser authentication, sensitive session credentials should not be exposed unnecessarily to JavaScript.

The preferred browser security model is a secure cookie-based session.

Session cookies should consider:

```text
HttpOnly
Secure
SameSite
appropriate expiration
```

The exact settings depend on the deployment architecture.

## Session Lifetime

Sessions should have controlled lifetimes.

The implementation should distinguish where necessary between:

- normal session lifetime
- inactivity expiration
- explicit logout
- security-triggered invalidation

Long-lived sessions should not be created without a clear reason.

## Session Revocation

The server must be able to invalidate authenticated sessions.

Revocation may be required when:

- the user logs out
- credentials are changed
- a security event occurs
- an administrator disables access
- suspicious activity is detected

## Logout

Logout must invalidate the authenticated session server-side where applicable.

Simply deleting a browser cookie is insufficient if the server still considers the session valid.

## Credential Changes

Changing a password should require appropriate verification of the current authenticated identity.

Credential changes may invalidate existing sessions depending on the security policy.

The exact behavior will be implemented with the password lifecycle.

## Password Reset

Password reset must use a secure recovery flow.

The system should:

- use single-use reset tokens
- limit token lifetime
- invalidate tokens after use
- avoid exposing account existence unnecessarily
- avoid logging reset secrets

Reset tokens must never be stored or transmitted insecurely.

## Email Verification

Email verification may be required for certain product capabilities.

If implemented, verification should use:

- time-limited tokens
- single-use verification
- safe token storage
- clear verification state

Email verification is distinct from authentication itself.

## Brute-Force Protection

Authentication endpoints must be protected against repeated automated attempts.

Controls may include:

- rate limiting
- temporary throttling
- progressive delays
- suspicious-activity detection
- credential-stuffing defenses

The exact implementation will be selected when authentication endpoints are created.

## Authentication Logging

Security-relevant authentication events should be logged appropriately.

Examples:

- successful login
- failed login
- logout
- password change
- password-reset request
- password-reset completion
- session invalidation

Logs must not contain:

- passwords
- session secrets
- reset tokens
- authentication headers
- sensitive credential material

## Authentication Errors

Client-facing authentication errors should be safe and consistent.

Do not expose:

- password-hash details
- database errors
- internal stack traces
- implementation details

## Authorization Boundary

Authentication answers:

```text
Who is this user?
```

Authorization answers:

```text
What may this user do?
```

Authentication must establish identity before authorization is evaluated.

## Future RBAC

Role-based access control is a separate concern.

Authentication should provide a reliable identity context that later authorization logic can use.

Do not hardcode business roles into authentication code.

## Multi-Tenancy Boundary

Multi-tenancy is implemented in a later phase.

Authentication identifies the user.

Tenant membership and tenant access must be evaluated separately.

A valid login must never automatically grant access to every tenant or business.

## User Deactivation

The system should support disabling authentication access for a user without necessarily deleting the user record.

A disabled identity must not be able to establish new authenticated sessions.

Existing-session handling must be defined by the security policy.

## Account Deletion

Account deletion is separate from authentication.

Deleting or anonymizing identity information must consider:

- active sessions
- historical records
- bookings
- financial records
- audit requirements
- legal retention requirements

## Authentication Data Minimization

Store only authentication information that is actually required.

Do not collect or retain unnecessary credential-related data.

## Secret Management

Authentication secrets must never be committed to Git.

Examples:

```text
session-signing secrets
reset-token secrets
email provider credentials
```

Sensitive configuration must remain server-side.

## CSRF

If browser authentication uses cookies, state-changing requests must include appropriate CSRF protection.

The exact mechanism will be selected during implementation.

## CORS

Authenticated API access must use explicitly configured allowed origins.

Wildcard origins must not be used for authenticated production traffic without a documented security reason.

## Authentication API

Authentication endpoints will live under:

```text
/api/v1/auth
```

Potential endpoints include:

```text
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/logout
GET  /api/v1/auth/me
POST /api/v1/auth/password-reset/request
POST /api/v1/auth/password-reset/confirm
POST /api/v1/auth/password/change
```

Only endpoints required by the current implementation step should be created.

## Authentication Middleware

Protected API requests should pass through authentication middleware that establishes the authenticated identity context.

Handlers should not repeatedly implement session parsing logic themselves.

## Identity Context

Authenticated requests should carry a server-controlled identity context.

Application services should receive the authenticated identity through explicit application context rather than trusting user-provided identity fields.

## Client-Supplied Identity

The server must never trust client-provided values such as:

```text
user_id
role
tenant_id
business_id
```

for authorization decisions.

The server must derive trusted identity from the authenticated session and authorization rules.

## Testing

Authentication must eventually have tests for:

- valid login
- invalid credentials
- disabled user
- logout
- expired session
- revoked session
- password change
- reset-token expiration
- reset-token reuse
- account enumeration resistance
- brute-force protections

Security-sensitive failure paths must be tested.

## Phase 3 Boundary

This architecture document defines authentication only.

It does not implement:

- RBAC
- multi-tenancy
- business roles
- tenant membership
- business permissions
- booking authorization

Those belong to later authorized work.

## Core Principle

Authentication must be:

```text
Secure
Revocable
Server-controlled
Testable
Minimal
Independent from authorization
```
