# Authentication Architecture

## Purpose

This document defines the authentication architecture for the Universal Booking Platform.

Authentication establishes who a user is.

Authorization determines what an authenticated user is allowed to do.

These responsibilities must remain separate.

---

# Authentication Model

The platform supports multiple authentication mechanisms that resolve to one local platform identity:

```text
Password
Google OAuth / OIDC
Apple Sign in with Apple
        ↓
Local User Account
        ↓
Application Session
        ↓
Authorization / RBAC
```

External identity providers authenticate an external account.

The application remains authoritative for:

- local users
- account linking
- application sessions
- authorization
- roles
- tenant membership

An external provider does not become the application's authorization authority.

---

# Local User Identity

The `users` table represents the platform's local identity.

Authentication mechanisms attach to this identity rather than replacing it.

Conceptually:

```text
users
├── user_credentials
├── user_oauth_identities
└── user_sessions
```

The local user remains the application-level identity regardless of whether authentication occurs through a password, Google, or Apple.

---

# Authentication Responsibilities

Authentication is responsible for:

- establishing identity
- verifying authentication credentials
- resolving external authentication identities
- creating authenticated application state
- maintaining authenticated session state
- ending authenticated sessions
- protecting authentication endpoints
- applying authentication-specific security controls

Authentication is not responsible for:

- tenant permissions
- business permissions
- staff permissions
- booking authorization
- resource authorization
- customer/business access rules

Those belong to authorization and application-level access control.

---

# Password Authentication

The platform supports password-based authentication.

Passwords must never be stored in plaintext.

Only a secure password hash may be persisted.

The current implementation uses Argon2id with a unique randomly generated salt per password.

The password hash is stored separately from the `users` table in:

```text
user_credentials
```

---

# Password Authentication Flow

The intended password authentication flow is:

```text
Client
   ↓
Login request
   ↓
Validate request
   ↓
Locate local user
   ↓
Load password credential
   ↓
Verify Argon2id password hash
   ↓
Authentication service
   ↓
Application session
```

Authentication failures should not unnecessarily reveal whether an account exists.

Password hashes must never be exposed through the API.

---

# OAuth / OIDC Authentication

The platform supports:

- Google OAuth/OIDC
- Apple Sign in with Apple

OAuth uses the standard authorization-code flow with OpenID Connect identity validation.

The application does not treat a provider access token as its own authentication session.

The external provider authenticates the external identity.

The local application then resolves that external identity to its own user model.

---

# OAuth Identity Model

OAuth identities are stored in a dedicated table:

```text
user_oauth_identities
```

Each identity contains the conceptual values:

```text
id
user_id
provider
provider_subject
created_at
updated_at
```

The external identity is identified by:

```text
provider + provider_subject
```

The provider subject is the stable external account identifier.

Email is supplementary account information.

Email must not be used as the permanent external identity key.

This is particularly important for Apple because Apple may provide private relay email addresses.

---

# OAuth Provider Identity Rules

Supported providers are initially:

```text
google
apple
```

The database enforces:

```text
UNIQUE(provider, provider_subject)
```

This ensures that the same external provider identity cannot be attached to multiple local users.

The initial model also enforces:

```text
UNIQUE(user_id, provider)
```

This limits each local user to one Google identity and one Apple identity.

A future requirement for multiple identities per provider would require an explicit architectural decision.

---

# OAuth Authorization Flow

The OAuth flow is:

```text
Client
   ↓
OAuth start endpoint
   ↓
Generate state
   ↓
Generate nonce
   ↓
Generate PKCE verifier
   ↓
Build authorization request
   ↓
Create encrypted OAuth transaction
   ↓
Store transaction in protected cookie
   ↓
Redirect to provider
   ↓
Provider authentication
   ↓
OAuth callback
   ↓
Read OAuth transaction
   ↓
Validate transaction
   ↓
Validate state
   ↓
Exchange authorization code
   ↓
Validate ID token signature
   ↓
Validate OIDC claims
   ↓
Normalize external identity
   ↓
Authentication service
   ↓
Local user
   ↓
Application session
```

No part of the browser-supplied OAuth response is trusted without server-side validation.

---

# OAuth State

The OAuth state value protects the authorization flow against forged callbacks and cross-request confusion.

State must be:

- generated using a cryptographically secure random source
- associated with the original authorization attempt
- short-lived
- validated during the callback
- protected against tampering

A callback with missing, incorrect, expired, or otherwise invalid state must fail safely.

---

# OAuth Nonce

OIDC nonce protects the relationship between the authorization request and the returned identity token.

Nonce must be:

- generated using a cryptographically secure random source
- associated with the OAuth transaction
- sent with the provider authorization request
- validated against the returned ID token

A missing or incorrect nonce must cause authentication to fail.

---

# PKCE

The authorization-code flow uses PKCE with the S256 challenge method.

The flow is:

```text
Generate verifier
     ↓
Create S256 challenge
     ↓
Send challenge to provider
     ↓
Retain verifier server-side
     ↓
Receive authorization code
     ↓
Exchange code using verifier
```

The PKCE verifier must not be exposed unnecessarily to the browser or logged.

---

# OAuth Transaction

A short-lived OAuth transaction binds the authorization request to the callback.

The transaction contains:

```text
provider
state
nonce
PKCE verifier
creation time
```

The transaction is encrypted before being stored in the browser.

The transaction is:

- integrity protected
- confidentiality protected
- short-lived
- provider-bound
- validated during the callback

The transaction does not represent an authenticated application session.

---

# OAuth Browser Cookie

The temporary OAuth transaction is stored in a protected browser cookie.

The cookie uses:

```text
HttpOnly
SameSite=Lax
Secure=true in production
Path=/
```

The transaction cookie does not contain a local authenticated user identity.

The OAuth transaction key remains server-side.

---

# OAuth Authorization Endpoints

Authentication OAuth endpoints are under:

```text
/api/v1/auth
```

The planned provider endpoints are:

```text
GET /api/v1/auth/google
GET /api/v1/auth/google/callback

GET /api/v1/auth/apple
GET /api/v1/auth/apple/callback
```

The start endpoint creates the authorization transaction and redirects to the provider.

The callback validates the transaction and provider response before handing the trusted external identity to the authentication service.

---

# Google Authentication

Google authentication uses:

```text
Google OAuth 2.0
OpenID Connect
Authorization Code
PKCE
ID Token
Google JWKs
```

The Google flow validates:

- token signature
- expected signing algorithm
- issuer
- audience
- authorized party where required
- nonce
- issued-at time
- expiration
- not-before where applicable
- provider subject

The Google provider subject is the stable identity key.

Google email may be retained as account information but is not the permanent external identity key.

---

# Apple Authentication

Apple authentication uses:

```text
Sign in with Apple
OpenID Connect
Authorization Code
PKCE
ID Token
Apple JWKs
```

The Apple flow validates:

- token signature
- expected ES256 signing algorithm
- issuer
- audience
- authorized party where required
- nonce
- issued-at time
- expiration
- not-before where applicable
- provider subject

The Apple provider subject is the stable identity key.

Apple email may be a private relay address and must not be treated as the permanent external account identifier.

---

# Apple Client Secret

Apple requires a server-generated client secret for the token exchange.

The application generates this secret using:

```text
Apple Team ID
Apple Key ID
Apple Client ID
Apple private key
```

The signing algorithm is:

```text
ES256
```

The client secret contains the appropriate issuer, subject, audience, issued-at time, and expiration claims.

The Apple private key must remain server-side.

The private key must never be:

- sent to the frontend
- returned by an API
- committed to Git
- included in documentation
- written to logs

Real Apple credentials are runtime configuration and require manual developer-console setup.

---

# OIDC ID Token Validation

A returned ID token must not be trusted merely because the provider returned it.

The server validates the cryptographic signature against the provider's JWK set.

The server must then validate the appropriate claims.

Required validation includes:

```text
signature
issuer
audience
authorized party when applicable
nonce
issued-at time
expiration
not-before when applicable
subject
```

Provider signing algorithms are restricted to the algorithms expected by that provider.

Google uses its expected RSA signing configuration.

Apple uses its expected ES256 signing configuration.

---

# JWK Verification

Provider signing keys are obtained from the provider's JWK endpoint.

The JWK key identifier is used to select the appropriate signing key.

The application must not accept an arbitrary signing key supplied by the client.

The signature must be verified before the claims are treated as trusted authentication information.

Production implementation must account for provider signing-key rotation and appropriate key caching behavior.

---

# OAuth Identity Normalization

After successful OIDC validation, provider-specific claims are normalized into the application identity model:

```text
OAuthIdentity
├── Provider
├── ProviderSubject
├── Email
├── EmailVerified
└── DisplayName
```

The `ProviderSubject` is the authoritative external identity value.

The normalized identity is then passed to the authentication service.

---

# Account Linking

Account linking is separate from first-time authentication.

The safe linking flow is:

```text
Authenticated local user
        ↓
Explicit link action
        ↓
Provider authorization
        ↓
Provider identity validation
        ↓
Check identity ownership
        ↓
Attach external identity
```

The system must not automatically merge accounts solely because:

```text
OAuth email == local user email
```

Email matching alone is insufficient authorization to take control of or merge an existing account.

A provider identity already belonging to another local user must not be attached to the current user.

An unauthenticated request must not attach an OAuth identity to an existing account.

---

# New OAuth Account

A first-time OAuth identity may eventually create a local user through the authentication service.

The conceptual flow is:

```text
Validated external identity
        ↓
Search user_oauth_identities
        ↓
Identity exists?
   ├── Yes → existing local user
   └── No  → controlled account-creation/linking flow
```

Local user creation must happen through the authentication service.

The OAuth provider must not directly create arbitrary users through repository calls.

---

# OAuth Provider Tokens

The current authentication architecture does not persist provider access or refresh tokens.

The provider's authentication response is used to establish or resolve the local authentication identity.

Provider access to external APIs is a separate future requirement.

If future product functionality needs provider API access, token storage and encryption must receive a separate security and architecture review.

---

# Application Session Integration

All authentication mechanisms eventually use the same application session model.

Conceptually:

```text
Password
Google OAuth
Apple OAuth
        ↓
Authentication Service
        ↓
Application Session
```

There must not be separate permanent session systems for password, Google, and Apple authentication.

The application session is the authoritative authenticated state for protected application requests.

---

# Authentication Service Boundary

The authentication service is responsible for combining authentication mechanisms with the local user model.

It will eventually handle:

- password authentication
- OAuth identity resolution
- new-user creation where appropriate
- account linking
- account status checks
- authentication policy
- application-session initiation

It must not silently grant authorization or tenant access.

---

# Session Boundary

Session creation and session lifecycle belong to the application session mechanism.

A validated OAuth identity alone does not constitute a fully authenticated application session.

The application must establish its own authenticated session after successful identity resolution.

---

# Authentication and Authorization Boundary

Authentication answers:

```text
Who is this user?
```

Authorization answers:

```text
What can this user do?
```

Authentication must establish the trusted identity context required by authorization.

Authorization remains a separate layer.

Business roles, tenant membership, staff permissions, customer permissions, and resource access must not be embedded in provider-specific OAuth logic.

---

# Account Status

Authentication must verify that the local account is eligible to authenticate.

A disabled local account must not establish a new authenticated application session.

External provider authentication does not override local account status.

---

# Authentication Errors

Client-facing authentication errors should be safe and consistent.

Do not expose:

- provider access tokens
- provider refresh tokens
- ID tokens
- authorization codes
- client secrets
- private keys
- password hashes
- database errors
- internal stack traces
- provider-specific sensitive internals

Provider failure details may be recorded in controlled server logs only when safe to do so.

---

# Authentication Logging

Security-relevant authentication events should be logged appropriately.

Potential events include:

- successful login
- failed login
- logout
- OAuth authentication failure
- OAuth identity linking
- OAuth identity unlinking
- password change
- session invalidation
- account disablement

Logs must never contain:

- passwords
- session secrets
- authorization codes
- OAuth state
- OAuth nonce
- PKCE verifier
- ID tokens
- access tokens
- refresh tokens
- client secrets
- private keys

---

# Account Enumeration

Authentication responses should avoid exposing whether a particular email address is already registered when doing so would enable account enumeration.

OAuth account resolution must not expose sensitive information about another local user's identity ownership.

---

# Brute-Force and Abuse Protection

Authentication endpoints must eventually be protected against abuse.

Potential controls include:

- rate limiting
- throttling
- progressive delays
- suspicious-activity detection
- credential-stuffing defenses

The final mechanisms will be implemented with the authentication service and security hardening work.

---

# CSRF

Cookie-based authentication requires appropriate protection against cross-site request forgery.

The exact application-wide CSRF mechanism will be selected as the authenticated browser API is implemented.

OAuth `state` protects the OAuth authorization transaction specifically and does not replace general CSRF protections for unrelated state-changing application requests.

---

# CORS

Authenticated API access must use explicitly configured allowed origins.

Wildcard production origins must not be used for authenticated traffic without a documented security reason.

Local development currently uses the Next.js development origin:

```text
http://localhost:3000
```

---

# Frontend Authentication UI

The authentication system will eventually provide responsive user interfaces for:

- registration
- sign in
- password authentication
- Continue with Google
- Continue with Apple
- password visibility
- validation errors
- authentication loading states
- OAuth errors
- logout
- session expiration

The authentication UI belongs to Phase 3.

It must remain:

```text
mobile-first
responsive
accessible
touch-friendly
```

The frontend must never contain:

- OAuth client secrets
- Apple private keys
- server-side OAuth transaction keys
- provider access tokens that do not need browser exposure
- password hashes

The frontend is a client of the authentication API, not the authority for authentication.

---

# Current OAuth Implementation Status

Phase 3.5-A is complete at the code/foundation level.

Implemented:

- OAuth identity persistence
- Google OAuth/OIDC foundation
- Apple Sign in with Apple foundation
- state
- nonce
- PKCE S256
- encrypted OAuth transaction
- protected transaction cookie
- code exchange foundation
- OIDC claims validation
- JWK-backed signature verification
- identity normalization
- Apple client-secret generation
- OAuth repository integration
- OAuth security testing

Not yet completed:

- authentication-service integration
- application-session integration
- complete production callback integration
- live Google provider verification
- live Apple provider verification
- frontend OAuth login UI
- complete registration/login integration

---

# Manual Provider Configuration Boundary

Real Google and Apple authentication require human configuration.

The application must not fabricate:

- OAuth client IDs
- client secrets
- Apple Team IDs
- Apple Key IDs
- Apple private keys
- provider redirect registrations

Real provider configuration must be performed through the appropriate provider developer consoles.

Secrets must remain outside source control and must never be provided through chat.

---

# Phase 3 Boundary

Phase 3 covers authentication and the foundation required for authorization.

It does not implement:

- multi-tenancy
- business authorization
- customer authorization
- booking authorization
- payment authorization
- messaging permissions
- production security hardening
- full OWASP ASVS verification

Those capabilities belong to their authorized phases.

---

# Core Principles

Authentication must be:

```text
Secure
Server-controlled
Revocable
Minimal
Testable
Provider-aware
Independent from authorization
```

OAuth must be:

```text
Validated
Bound to the authorization request
Protected against tampering
Protected against replay
Based on stable provider identity
Integrated into the local application identity model
```

The application must always remain the authority for its own user accounts, sessions, and authorization.
