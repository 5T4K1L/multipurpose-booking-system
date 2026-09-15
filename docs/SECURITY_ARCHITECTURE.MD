# Security Architecture

## Purpose

This document defines the security architecture for the Universal Booking Platform.

Security is a system-wide requirement rather than a single implementation phase.

The platform must be designed so that:

- untrusted input is never treated as trusted
- authorization is enforced server-side
- tenants cannot access each other's data
- authentication secrets are protected
- sensitive operations are auditable
- failures do not expose internal information
- security controls are layered
- security improvements remain compatible with the free-first development strategy

The project targets OWASP ASVS Level 2 as the baseline security standard, with selective Level 3 controls applied to especially sensitive areas.

This does not represent a claim of ASVS compliance or external security certification.

---

# Core Security Principles

The platform follows these principles:

```text id="s1a9k3"
Never trust the client
Server-side enforcement
Least privilege
Defense in depth
Secure defaults
Minimize sensitive data
Fail securely
Explicit authorization
Strong tenant isolation
Safe dependency usage
Auditable security events
```

Security controls should be implemented as close as possible to the boundary they protect.

---

# Trust Boundaries

The primary trust boundaries are:

```text id="b2c8m4"
Browser
   ↓
Next.js
   ↓
Go API
   ↓
PostgreSQL
```

Additional external boundaries include:

```text id="d4e7p6"
Go API
   ↓
Google
Apple
Payment providers
Email providers
Storage providers
Messaging providers
Search providers
```

Data crossing any trust boundary must be validated and authenticated as appropriate.

---

# Browser Security

The browser is untrusted.

The frontend must never be treated as the authority for:

- identity
- user role
- tenant
- business ownership
- booking ownership
- payment state
- entitlement state

The browser may request an operation.

The backend decides whether that operation is permitted.

---

# Frontend Security

The Next.js application must:

- avoid exposing secrets
- avoid trusting client-controlled permissions
- escape untrusted content
- validate user input before submission
- display safe error messages
- avoid storing sensitive authentication material unnecessarily

Environment variables intended for the browser must never contain server-only secrets.

---

# API Security Boundary

The Go API is the primary enforcement boundary for application operations.

Every protected request must establish:

```text id="e5f6g7"
authentication
    ↓
authorization
    ↓
tenant/resource access
    ↓
service operation
```

A successful frontend check is never sufficient authorization.

---

# Input Validation

All external input must be treated as untrusted.

Validate:

- JSON bodies
- path parameters
- query parameters
- headers
- uploaded files
- webhook payloads
- OAuth callbacks
- pagination values
- filters
- sort values

Validation must occur before the input reaches sensitive business logic.

---

# Output Safety

Responses must expose only information the authenticated principal is authorized to receive.

Do not return:

- password hashes
- session secrets
- OAuth tokens
- OAuth private keys
- client secrets
- internal database errors
- internal stack traces
- sensitive tenant information

---

# SQL Injection Protection

SQL must never be constructed from untrusted strings through direct concatenation.

The application uses:

```text id="h8i9j0"
PostgreSQL
pgx
sqlc
parameterized queries
```

User-controlled values must be passed as parameters.

Dynamic sorting, filtering, or column selection must use explicit allowlists rather than direct SQL interpolation.

---

# Authentication Security

Authentication establishes the local application identity.

Supported mechanisms include:

```text id="k1l2m3"
Password
Google OAuth/OIDC
Apple Sign in with Apple
```

All mechanisms must eventually produce the same local application identity and normal application session.

---

# Password Security

Passwords must never be stored in plaintext.

The application uses a strong password hashing mechanism based on Argon2id.

Passwords must never appear in:

- logs
- API responses
- database dumps intended for developers
- audit records
- error messages

Password verification failures should return safe responses.

---

# Password Reset Security

Password reset flows must use:

- high-entropy random tokens
- short expiration
- single-use behavior
- safe storage
- rate limiting
- generic request responses where appropriate

Password-reset tokens must not be logged.

---

# OAuth Security

OAuth is an external authentication mechanism, not the application's authorization system.

The application remains authoritative for:

- local users
- account status
- application sessions
- roles
- memberships
- authorization

OAuth implementation must use:

```text id="n4o5p6"
authorization code flow
PKCE S256
state
OIDC nonce
OIDC validation
JWK signature verification
```

---

# OAuth State

OAuth state protects the authorization transaction.

State must be:

- cryptographically random
- associated with the authorization attempt
- short-lived
- integrity protected
- validated on callback

A missing, invalid, expired, or mismatched state must terminate the authentication attempt.

---

# OAuth Nonce

The OIDC nonce binds the authorization request to the returned ID token.

The application must:

1. generate a cryptographically secure nonce
2. send it in the authorization request
3. associate it with the temporary transaction
4. validate the nonce returned in the ID token

A nonce mismatch must fail authentication.

---

# OAuth PKCE

PKCE is used with S256.

The server creates a random verifier and derives the challenge.

The verifier must remain protected until code exchange.

The verifier must not be logged or unnecessarily exposed to client-side JavaScript.

---

# OAuth Transaction Protection

The current implementation stores temporary OAuth transaction information in an encrypted browser cookie.

The transaction is short-lived and contains values such as:

```text id="q7r8s9"
provider
state
nonce
PKCE verifier
creation time
```

The transaction cookie uses:

```text id="t0u1v2"
HttpOnly
SameSite=Lax
Secure in production
Path=/
```

The encryption key remains server-side.

The OAuth transaction is not an authenticated session.

---

# OIDC Token Validation

An ID token must never be trusted merely because it came from the provider callback.

The server validates:

```text id="w3x4y5"
signature
issuer
audience
subject
nonce
expiration
issued-at time
not-before where applicable
authorized party when applicable
```

The accepted signing algorithm must be restricted to the provider's expected algorithm.

Google and Apple provider keys are validated using provider JWKs.

---

# JWK Security

Provider signing keys must come from trusted provider JWK endpoints.

The application must not accept arbitrary verification keys from the client.

Key selection must use the provider key identifier where appropriate.

Provider key rotation must be supported without weakening signature verification requirements.

---

# OAuth Identity Security

The authoritative external identity is:

```text id="z6a7b8"
provider + provider_subject
```

Email alone must not identify the external account.

This protects account association from unsafe email-based automatic merging.

---

# Account Linking Security

Account linking is privileged.

A user must already be authenticated to initiate an account-linking operation.

The linking flow must:

- authenticate the current user
- validate the provider transaction
- validate the provider identity
- verify the external identity is not already owned by another user
- attach the identity only to the authorized local user

An OAuth identity belonging to another account must not be silently transferred.

---

# Account Merge Protection

The application must not automatically merge accounts solely because:

```text id="c8d9e0"
OAuth email == local email
```

Email equality alone is not sufficient proof of account ownership.

Account merging, if ever introduced, requires a dedicated security design.

---

# Session Security

After successful authentication, the application creates its own session.

OAuth and password login must use the same application session architecture.

Sessions must support:

- expiration
- revocation
- logout
- account disablement
- security-driven invalidation

Session identifiers must be generated using cryptographically secure randomness.

---

# Session Cookies

Where cookie-based authentication is used, production cookies should use:

```text id="f1g2h3"
HttpOnly
Secure
appropriate SameSite policy
restricted path/domain
```

Session cookies must not expose secrets to JavaScript unnecessarily.

---

# Session Revocation

A session may be revoked because of:

- logout
- password security event
- account disablement
- suspected compromise
- administrative action

Revoked sessions must no longer authorize protected requests.

---

# Authorization

Authorization must occur after authentication.

The backend must determine:

```text id="i4j5k6"
Who is the user?
What roles do they have?
What tenant or business do they belong to?
What resource are they accessing?
Are they allowed to perform this action?
```

Never trust role values supplied by the client.

---

# Role-Based Access Control

The application roles include:

```text id="l7m8n9"
CUSTOMER
BUSINESS_OWNER
BUSINESS_ADMIN
MANAGER
STAFF
PLATFORM_ADMIN
PLATFORM_OPERATOR
```

Role definitions belong to the application authorization layer.

Authentication providers must never assign application roles directly.

---

# Object-Level Authorization

Every resource access must be authorized for the current principal.

Example:

```text id="o0p1q2"
GET /api/v1/bookings/{id}
```

does not mean:

```text id="r3s4t5"
valid ID = valid access
```

The server must verify ownership or authorized membership.

---

# Tenant Isolation

Tenant isolation is a critical security boundary.

The system must prevent:

```text id="u6v7w8"
Business A
    ↓
accessing
Business B data
```

Tenant context must be derived from trusted server-side authentication and authorization state.

A client-supplied tenant ID is never sufficient authorization.

---

# Cross-Tenant Access Protection

Tenant-scoped database queries must use trusted tenant context.

The service should ensure that:

```text id="x9y0z1"
authenticated user
   ↓
authorized membership
   ↓
trusted tenant
   ↓
tenant-scoped repository query
```

Object-level checks remain necessary even after tenant membership is established.

---

# ID Enumeration Protection

Predictable identifiers must not automatically grant access to records.

Authorization must protect against insecure direct object reference attacks.

Returning `404` for unauthorized resources may be appropriate where it reduces information disclosure.

---

# CSRF

Cookie-based authenticated operations require CSRF protection.

OAuth `state` protects the OAuth authorization flow specifically.

It does not replace general application CSRF protection.

State-changing APIs must follow the application's CSRF strategy once browser authentication is fully integrated.

---

# CORS

Authenticated cross-origin requests must use a controlled allowlist.

Production should not broadly allow arbitrary origins.

Development may explicitly allow known local development origins.

---

# Secrets Management

Secrets must never be committed to source control.

Examples include:

```text id="a2b3c4"
database passwords
OAuth client secrets
Apple private keys
OAuth transaction encryption keys
payment credentials
email credentials
API keys
signing keys
```

Secrets must be provided through secure runtime configuration.

---

# Server-Only Configuration

Server-only secrets must remain in the Go/backend runtime.

They must never be included in browser bundles.

Frontend-exposed environment variables must contain only values explicitly intended for client exposure.

---

# OAuth Secrets

OAuth credentials must remain server-side.

Never expose:

- Google client secret
- Apple private key
- Apple signing credentials
- generated Apple client secret
- OAuth transaction key

to the frontend.

---

# Logging Security

Logs must support debugging without leaking secrets.

Never log:

```text id="d5e6f7"
passwords
password hashes
session tokens
OAuth authorization codes
OAuth state
OAuth nonce
PKCE verifier
ID tokens
access tokens
refresh tokens
client secrets
private keys
payment card data
```

Structured logs should include safe contextual information such as:

```text id="g8h9i0"
request_id
user_id when appropriate
tenant_id when appropriate
event type
timestamp
result
```

Sensitive identifiers should be minimized according to the event.

---

# Error Security

Production errors must not expose:

- stack traces
- SQL queries
- database schema details
- provider credentials
- internal filesystem paths
- secret values
- implementation details unnecessary for clients

Internal diagnostic information belongs in controlled server-side logs.

---

# Rate Limiting

Rate limiting should protect abuse-prone operations.

Priority targets include:

- login
- registration
- password reset
- OAuth start
- OAuth callback
- account linking
- public booking
- webhook processing
- message sending where abuse is possible
- expensive search operations

A first implementation may use application-level controls.

Shared infrastructure such as Redis can be introduced when scale justifies it.

---

# Brute-Force Protection

Authentication endpoints should eventually support defenses against:

- brute-force password attacks
- credential stuffing
- automated account creation
- OAuth abuse
- password-reset abuse

Responses should avoid helping attackers determine whether an account exists.

---

# File Upload Security

Uploaded files are untrusted.

The platform must validate:

- size
- MIME type
- file signature where necessary
- extension
- destination
- tenant ownership
- authorization

Files must be stored outside executable application paths.

---

# Webhook Security

Webhooks are untrusted until authenticated.

Webhook processing must verify provider signatures and use replay/idempotency controls where available.

A webhook must not modify sensitive state before signature verification.

---

# Payment Security

The application must never store raw card information.

Payment providers should handle sensitive card data.

The platform should store provider references and verified state only.

Payment webhook authenticity is mandatory.

---

# Dependency Security

Dependencies must be monitored for:

- known vulnerabilities
- malicious package changes
- abandoned critical libraries
- unnecessary dependencies

Adding a dependency should have a clear purpose.

The Go module and frontend package lockfiles must remain committed and reviewable.

---

# Supply Chain Security

Generated code and external packages must be treated as potentially influential inputs.

The project should prefer:

- trusted package sources
- pinned or reproducible dependency versions
- dependency review
- minimal dependencies
- regular security updates

---

# Database Security

Database access must use least privilege.

The application database user should receive only the permissions required by the application.

Administrative credentials must not be embedded in application code.

Production database access should be restricted by network and identity controls.

---

# Data Protection

Sensitive customer and business data should be minimized.

The application must avoid collecting information that is not necessary for a product feature.

Future production deployment should provide appropriate encryption at rest, transport encryption, backup protection, and access controls.

---

# HTTPS

Production traffic must use HTTPS.

Sensitive authentication and application traffic must never rely on plaintext HTTP in production.

Secure cookies must be used with HTTPS.

---

# Security Headers

Production frontend and API deployment should use appropriate security headers.

Expected controls may include:

```text id="j1k2l3"
Content-Security-Policy
Strict-Transport-Security
X-Content-Type-Options
Referrer-Policy
Frame-related protections
```

Exact policies will be finalized during production security hardening.

---

# Content Security

User-generated content must be safely rendered.

The platform must defend against:

- XSS
- HTML injection
- unsafe URL handling
- script injection

User content should be escaped or sanitized according to the rendering context.

Never assume database-stored content is safe merely because it came through the application.

---

# Authentication Audit Events

Security-relevant events should be auditable.

Examples:

```text id="m4n5o6"
login_success
login_failure
oauth_login
oauth_link
oauth_unlink
logout
session_revoked
password_changed
password_reset_completed
account_disabled
role_changed
```

Audit records must never contain credentials or authentication secrets.

---

# Security Testing

Security testing should include:

```text id="p7q8r9"
authentication tests
authorization tests
tenant-isolation tests
session tests
OAuth validation tests
input-validation tests
SQL-injection tests
CSRF tests
rate-limit tests
file-upload tests
webhook tests
dependency checks
```

Security tests should include both accepted and rejected cases.

---

# OAuth Security Tests

OAuth tests should verify rejection of:

```text id="s0t1u2"
incorrect state
missing state
expired transaction
tampered transaction
incorrect nonce
missing nonce
invalid PKCE verifier
invalid authorization code
invalid issuer
invalid audience
invalid subject
expired ID token
not-yet-valid ID token
invalid signature
unexpected signing algorithm
unknown provider
identity already belonging to another user
```

The existing OAuth foundation includes dedicated tests for these categories.

---

# Security Headers and Cookies Testing

Browser-facing authentication flows must verify:

- HttpOnly
- Secure in production
- SameSite behavior
- path/domain scope
- appropriate CORS behavior
- CSRF protections

These controls should be tested through integration/browser tests once the authentication UI is integrated.

---

# Observability Security

Observability systems must not become a secret-leak mechanism.

Tracing, metrics, error reporting, and logs must use redaction rules.

Sensitive request fields must not automatically be captured.

---

# Security Incident Handling

Production security incidents should have a documented response process.

The process should include:

```text id="v3w4x5"
identify
contain
investigate
recover
rotate credentials
restore affected services
document
prevent recurrence
```

Compromised authentication or integration credentials should be rotated immediately.

---

# Emergency Access

Administrative access must be limited and auditable.

Emergency privileges should not become a normal mechanism for application operations.

Platform administrator access requires stronger protection than ordinary customer or business access.

---

# Security and Development Workflow

Security controls must be implemented during normal development rather than postponed entirely to the final phase.

Each phase should consider:

```text id="y6z7a8"
authentication
authorization
tenant isolation
input validation
secrets
logging
error handling
abuse
data exposure
```

---

# OWASP ASVS Target

The security baseline is:

```text id="b9c0d1"
OWASP ASVS Level 2
```

Selective Level 3 controls may be applied to especially sensitive systems such as:

- authentication
- session management
- payment flows
- administrative capabilities
- tenant isolation

The project must not claim formal ASVS compliance until a dedicated verification process has actually been completed.

---

# Phase 13 Security Hardening

Phase 13 is responsible for systematic security hardening and verification.

Expected areas include:

```text id="e2f3g4"
authentication
authorization
tenant isolation
API security
input validation
session security
OAuth
payment security
secrets
dependency security
logging
abuse prevention
security testing
```

Phase 13 should verify controls introduced earlier rather than postponing all security engineering until then.

---

# Current Phase 3 Security Status

Phase 3 security foundations currently include:

- password-hashing architecture
- application session architecture
- OAuth state
- OIDC nonce
- PKCE S256
- encrypted OAuth transaction
- protected OAuth cookie
- OIDC claim validation
- JWK signature verification
- provider-specific signing algorithm restrictions
- OAuth identity uniqueness
- no email-only account merge
- server-side identity ownership

Not yet complete:

- complete application session integration
- authentication middleware
- production OAuth configuration
- complete frontend authentication flow
- complete RBAC enforcement
- full security hardening

---

# Security Architecture Boundaries

Security architecture must not be confused with product authorization architecture.

Security establishes the controls.

Authentication establishes identity.

Authorization determines permissions.

Database constraints protect durable integrity.

The booking engine remains authoritative for booking correctness.

Payment providers remain authoritative for their externally verified payment events.

---

# Core Principle

The platform must assume that attackers can control:

```text id="h5i6j7"
the browser
request bodies
query parameters
path parameters
headers
object IDs
tenant IDs
OAuth callback parameters
uploaded files
webhook payloads
```

Security therefore depends on:

```text id="k8l9m0"
validated input
authenticated identity
server-side authorization
database constraints
secure session handling
tenant isolation
minimal sensitive data
layered defenses
```

No single security control is sufficient on its own.
