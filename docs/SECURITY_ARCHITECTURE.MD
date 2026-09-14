# Security Architecture

## Purpose

This document defines the baseline security principles for the Universal Booking Platform.

Security is a system-wide requirement and must be considered during every feature phase.

## Core Security Principles

The system must follow these principles:

- never trust client input
- enforce security on the server
- use least privilege
- minimize sensitive data
- validate all external input
- protect secrets
- fail securely
- prefer explicit authorization
- avoid exposing internal implementation details
- keep dependencies maintained
- log security-relevant events appropriately

## Trust Boundaries

The system has several trust boundaries:

```text
Browser
   │
   │ Untrusted input
   ▼
Go API
   │
   │ Controlled application logic
   ▼
PostgreSQL
```

The browser must never be considered a trusted environment.

## Frontend Security

Frontend validation exists primarily for user experience.

It must never replace backend validation or authorization.

The backend must independently verify:

- identity
- permissions
- resource ownership
- tenant access
- business rules
- financial rules

when those capabilities are implemented.

## Input Validation

Every externally supplied value must be validated before use.

This includes:

- request bodies
- query parameters
- path parameters
- headers
- uploaded data
- imported records
- webhook payloads
- external API responses

Validation must occur on the server.

## Output Handling

User-controlled content must be handled safely.

The application must avoid unsafe HTML rendering unless the content has been appropriately sanitized and the behavior is explicitly required.

Avoid constructing HTML from untrusted values.

## SQL Injection Prevention

Database queries must use parameterized queries or safe query-generation mechanisms.

Do not concatenate user input into SQL statements.

Unsafe:

```text
"SELECT * FROM customers WHERE name = '" + userInput + "'"
```

Preferred:

```text
Parameterized query
```

## Authentication

Authentication is intentionally not implemented yet.

When introduced, authentication must:

- securely verify identity
- protect credentials
- support secure sessions or tokens
- prevent credential enumeration where applicable
- provide appropriate account recovery controls
- invalidate sessions appropriately

Passwords must never be stored in plaintext.

## Password Security

When password authentication is implemented:

- use a modern password-hashing algorithm
- use appropriate work factors
- never store plaintext passwords
- never log passwords
- never return passwords through API responses

The exact password hashing approach will be selected before authentication is implemented.

## Authorization

Authorization must always be enforced server-side.

The system must never rely on:

- hidden frontend buttons
- client-side role values
- route visibility
- browser state

to provide actual security.

## Least Privilege

Users, services, database connections, and infrastructure should receive only the permissions they need.

The application should avoid:

- unnecessary database privileges
- unnecessary administrative permissions
- shared credentials
- broad service access

## Multi-Tenancy Security

Multi-tenancy is implemented in a later phase.

When implemented, every tenant-scoped operation must enforce tenant isolation server-side.

A client must never be able to select another tenant simply by changing an identifier in a request.

## Object-Level Authorization

Every protected resource must verify that the authenticated user is authorized to access that specific resource.

Do not assume that knowing a resource ID grants access.

## ID Enumeration

Resource identifiers should not be treated as authorization mechanisms.

Authorization must be checked independently of the identifier format.

## Session Security

When sessions are implemented, they should consider:

- secure cookies
- HttpOnly
- SameSite
- expiration
- session invalidation
- session rotation
- CSRF protection where applicable

The exact session architecture will be defined during authentication implementation.

## CSRF

Cross-site request protections must be applied where the authentication architecture requires them.

The exact mechanism will be selected when authentication is implemented.

Do not add speculative CSRF infrastructure before the authentication model exists.

## CORS

CORS must be explicitly configured.

The API should allow only the origins required by the application.

Avoid wildcard origins for authenticated production APIs unless there is a documented security reason.

Current local development allows the Next.js development origin:

```text
http://localhost:3000
```

Production CORS configuration will be defined later.

## Secrets

Secrets must never be committed to Git.

Examples:

- database passwords
- API keys
- signing keys
- access tokens
- webhook secrets
- payment credentials

Secrets must also not be placed in:

- source code
- screenshots
- documentation
- logs
- API responses

## Environment Files

`.env.example` may contain safe placeholder values.

Real `.env` files containing secrets must remain outside source control.

## Error Security

Client-facing errors must not expose:

- stack traces
- SQL statements
- database credentials
- internal filesystem paths
- infrastructure details
- secret values

Unexpected server errors should return a safe generic response.

## Logging Security

Logs must not contain secrets or unnecessary sensitive personal information.

Security-relevant events may be logged using safe identifiers.

Examples include:

- authentication failure
- authorization failure
- suspicious access patterns
- administrative actions

## Rate Limiting

Rate limiting will be introduced where required to protect:

- authentication endpoints
- sensitive operations
- public APIs
- resource-intensive endpoints
- abuse-prone functionality

The exact implementation will be selected based on the relevant feature requirements.

## Brute Force Protection

Authentication features must include appropriate protection against repeated credential attacks.

The exact controls will be defined during the authentication phase.

## File Upload Security

Future file-upload functionality must consider:

- file type validation
- file size limits
- content validation
- storage isolation
- filename handling
- malware scanning where appropriate
- unauthorized file access

File uploads are not implemented in the current phase.

## Webhook Security

Future webhooks must verify authenticity before processing.

Possible mechanisms include:

- signatures
- shared secrets
- replay protection
- timestamp validation
- idempotency

Webhook functionality is not implemented in the current phase.

## Payment Security

Payment functionality is intentionally deferred.

When implemented:

- sensitive payment credentials must not be stored unnecessarily
- payment-provider security requirements must be followed
- server-side payment verification is required
- financial state changes must be protected against duplicate processing
- webhook authenticity must be verified

## Dependency Security

Dependencies should be:

- intentionally selected
- kept reasonably current
- reviewed for security issues
- removed when no longer needed

Do not add dependencies solely for hypothetical future requirements.

## Database Security

Database access must:

- use controlled credentials
- use least privilege where practical
- use parameterized queries
- protect connection credentials
- avoid exposing database ports publicly in production

The current PostgreSQL port exposure is for local development only.

## Data Protection

Collect only data required by the product.

Sensitive data should not be retained indefinitely without a legitimate purpose.

Future data-retention rules must consider:

- privacy
- legal requirements
- operational needs
- customer expectations

## Audit Logging

Important security-sensitive and administrative actions should eventually be auditable.

Potential examples:

- permission changes
- account security changes
- business configuration changes
- financial actions
- administrative actions

Audit logging will be designed before the relevant features are implemented.

## Security Headers

The web application should eventually use appropriate security-related HTTP headers.

The exact header set will be established when deployment and production security are implemented.

## Content Security

The application should minimize unsafe script execution and avoid unnecessary third-party script dependencies.

Any external scripts must have a clear business and security reason.

## Dependency and Supply-Chain Risk

Third-party packages should be evaluated before adoption.

Avoid:

- abandoned dependencies
- unnecessary packages
- packages with excessive privileges
- duplicating functionality already provided by the platform

## Secure Defaults

New features should default to the more secure behavior.

Examples:

- deny access unless explicitly allowed
- private by default
- validation enabled
- sensitive information excluded from logs
- secure configuration required

## Security Testing

Security testing will eventually include:

- input validation testing
- authorization testing
- authentication testing
- API abuse testing
- dependency checks
- penetration-oriented testing
- OWASP-focused review

The exact testing strategy will expand as the product becomes more capable.

## Incident and Emergency Controls

Future production architecture should provide mechanisms for:

- disabling dangerous functionality
- restricting access
- revoking credentials
- responding to security incidents
- recovering from compromised components

These capabilities are intentionally deferred until the relevant infrastructure phases.

## Security Documentation

Important security decisions must be documented in:

```text
docs/DECISIONS.md
```

Security-sensitive features should document their assumptions and boundaries.

## Phase 1 Boundary

This phase defines the security baseline only.

It does not implement:

- authentication
- authorization
- RBAC
- multi-tenancy security
- production secrets management
- WAF
- advanced rate limiting
- production security headers
- penetration testing infrastructure
- centralized security monitoring

Those capabilities will be implemented in their authorized phases.

## Core Principle

Security must be built into the architecture from the beginning.

A feature is not considered complete merely because it works functionally.

It must also be:

```text
Validated
Authorized
Protected
Tested
Auditable where required
```
