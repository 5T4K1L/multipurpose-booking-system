# Configuration Architecture

## Purpose

This document defines how configuration is structured, validated, supplied, and protected across the Universal Booking Platform.

Configuration must be:

- explicit
- environment-aware
- secure
- reproducible
- easy to validate
- separated from application code
- safe for local development
- suitable for CI/CD and production deployment

Secrets must never be committed to source control.

---

# Configuration Principles

The platform follows these principles:

```text id="a7m2q4"
Configuration is external to application code
Secrets are never committed
Environment differences are explicit
Server-only secrets remain server-side
Required configuration fails fast
Safe defaults are preferred
Production configuration is explicit
```

Configuration must not become a hidden source of application behavior.

---

# Environments

The application should distinguish between:

```text id="b4n8s1"
development
test
staging
production
```

Each environment may have different:

- database connection settings
- API URLs
- frontend URLs
- OAuth redirect URLs
- logging levels
- security settings
- external provider configuration

Environment-specific differences must not require source-code changes.

---

# Configuration Sources

Configuration may come from:

- environment variables
- local development configuration
- CI/CD environment configuration
- production secret management
- explicitly supported configuration files for non-secret development values

Environment variables are the primary runtime configuration mechanism.

---

# Secret Management

Secrets must be supplied securely through the runtime environment or an appropriate secret manager.

Never commit:

```text id="c3x7v2"
.env
.env.local
production secrets
OAuth client secrets
private keys
database passwords
payment credentials
API secrets
```

A `.env.example` may contain variable names and safe placeholder values.

It must never contain real credentials.

---

# Configuration Validation

The backend must validate required configuration during startup.

Validation should detect:

- missing required values
- malformed URLs
- invalid durations
- unsupported environment names
- invalid provider settings
- incomplete OAuth configuration
- invalid cryptographic configuration

The application should fail clearly rather than start with unsafe security configuration.

---

# Configuration Categories

Configuration can be grouped into:

```text id="d9h3k1"
application
server
database
frontend
authentication
OAuth
logging
security
external providers
feature flags
```

Grouping improves discoverability and prevents unrelated configuration from becoming mixed together.

---

# Application Configuration

Typical application configuration includes:

```text id="f1j8p5"
APP_ENV
APP_NAME
APP_BASE_URL
```

The exact variables should match the implementation.

`APP_ENV` should identify the runtime environment.

---

# Server Configuration

Typical API configuration includes:

```text id="g5k2r7"
API_HOST
API_PORT
API_BASE_PATH
```

Development currently uses the Go API on:

```text id="0x8q3m"
http://localhost:8080
```

The frontend development application currently uses:

```text id="1n5v6b"
http://localhost:3000
```

Production URLs must come from deployment configuration rather than hard-coded source values.

---

# Database Configuration

Database configuration should include values such as:

```text id="h7y4t9"
DATABASE_URL
```

or equivalent discrete settings when the implementation requires them.

Database credentials must never be hard-coded.

Production database configuration must be injected securely.

---

# Database Security

Database credentials should be least-privileged.

The application runtime should not use an unrestricted database administrator account unless there is an explicit operational requirement.

Migration execution may use a separate controlled credential where appropriate.

---

# Frontend Configuration

The frontend may expose only configuration explicitly intended for the browser.

Examples include:

```text id="j3w8c6"
public API base URL
public application URL
non-secret feature configuration
```

Frontend-exposed environment variables must never contain:

- database passwords
- OAuth client secrets
- private keys
- transaction encryption keys
- payment secrets
- server signing keys

---

# Authentication Configuration

Authentication configuration includes settings for:

- password policy
- session duration
- session cookie behavior
- password reset lifetime
- email verification lifetime
- authentication rate limits

Security-sensitive defaults must be conservative.

---

# OAuth Configuration

The current OAuth foundation requires provider-specific configuration.

## Google

Configuration includes:

```text id="k2a7d9"
GOOGLE_OAUTH_CLIENT_ID
GOOGLE_OAUTH_CLIENT_SECRET
GOOGLE_OAUTH_REDIRECT_URL
```

The client secret is server-only.

The redirect URL must exactly correspond to the provider configuration for the environment.

---

## Apple

Configuration includes:

```text id="m4s9f2"
APPLE_OAUTH_CLIENT_ID
APPLE_TEAM_ID
APPLE_KEY_ID
APPLE_PRIVATE_KEY_PEM
APPLE_OAUTH_REDIRECT_URL
```

The Apple private key is server-only.

The application uses these values to generate the required Apple client secret.

The generated Apple client secret must not be exposed to the browser.

---

# OAuth Transaction Key

The application uses a server-side encryption key for temporary OAuth transaction data.

Conceptually:

```text id="n7c3v8"
OAUTH_TRANSACTION_KEY
```

This key must:

- remain server-side
- have sufficient cryptographic strength
- never be committed
- never appear in logs
- never be returned by an API

The application should fail startup if a production configuration requires this key and it is missing or invalid.

---

# OAuth Redirect URLs

Each provider requires an environment-specific callback URL.

Examples:

```text id="p6j1r4"
/api/v1/auth/google/callback
/api/v1/auth/apple/callback
```

The complete externally reachable URL must be registered with the provider.

Development and production redirect URLs should be treated separately.

Never assume a production redirect URL can safely be reused for local development.

---

# OAuth Configuration Security

Provider configuration must not be accepted from browser input.

The client must not be allowed to submit:

```text id="q8u5y0"
client_id
client_secret
redirect_url
team_id
key_id
private_key
```

The backend reads provider configuration from trusted runtime configuration.

---

# Apple Private Key Handling

`APPLE_PRIVATE_KEY_PEM` is highly sensitive.

It must:

- exist only on trusted backend/runtime environments
- never be committed
- never be logged
- never be returned through API responses
- never be exposed through browser environment variables

The application should validate that the configured key can be parsed before enabling Apple authentication.

---

# Development OAuth Configuration

Real production OAuth credentials should not be inserted into source control.

For local development, developers may configure development-provider credentials through local environment configuration.

Manual setup is required through the provider developer consoles.

The application should clearly distinguish configured-but-unverified development credentials from successfully tested provider integration.

---

# Production OAuth Configuration

Production deployment requires:

```text id="r3v7b1"
registered provider application
correct client IDs
correct redirect URLs
secure client secrets
Apple Team ID
Apple Key ID
Apple private key
OAuth transaction encryption key
```

Real production values must be supplied through deployment secrets management.

They must not be stored in the Git repository.

---

# Configuration and Environment Files

Local developers may use environment files such as:

```text id="s4w8e2"
.env
.env.local
```

only when those files are excluded from source control.

The repository should provide a safe example:

```text id="t7y2k5"
.env.example
```

The example must document variable names without revealing real secrets.

---

# `.gitignore`

Sensitive environment files should be excluded from Git.

The repository should protect against accidental commits of:

```text id="u1c6m9"
.env
.env.*
*.pem
private key files
local secret files
```

Care must be taken to avoid excluding intended example configuration such as `.env.example`.

---

# Configuration Naming

Environment variables should use uppercase `SCREAMING_SNAKE_CASE`.

Examples:

```text id="v4d9p3"
DATABASE_URL
API_PORT
GOOGLE_OAUTH_CLIENT_ID
APPLE_TEAM_ID
OAUTH_TRANSACTION_KEY
```

Names should describe the configuration value clearly.

Avoid ambiguous names such as:

```text id="x6f2q8"
KEY
SECRET
URL
VALUE
```

unless the scope is obvious from a strongly structured configuration namespace.

---

# Configuration Parsing

Configuration should be parsed into typed application configuration structures.

Avoid scattering direct `os.Getenv` calls throughout business logic.

Prefer:

```text id="y8h5r1"
Environment
   ↓
Configuration loader
   ↓
Validated config
   ↓
Application components
```

This keeps configuration behavior predictable and testable.

---

# Required vs Optional Configuration

Each configuration variable should be classified as:

```text id="z2j7k4"
required
optional
development-only
production-only
provider-specific
```

The application should not require disabled provider configuration when that provider is intentionally unavailable in the current environment.

For example, Apple configuration may remain absent while Apple authentication is disabled in a local environment, provided the application explicitly supports this state.

---

# Feature Configuration

Feature flags may be used when functionality needs controlled rollout.

Flags must not become a replacement for authorization.

For example:

```text id="a3e8n6"
ENABLE_GOOGLE_OAUTH
ENABLE_APPLE_OAUTH
```

may control whether a provider is available.

They must not decide whether an authenticated user has permission to access protected resources.

---

# Configuration and Security Defaults

Security-sensitive settings should have safe defaults.

Examples:

```text id="b9f4m2"
Secure cookies enabled in production
Verbose debug logging disabled in production
Broad CORS disabled by default
Authentication rate limits enabled where required
External provider secrets not exposed to clients
```

Unsafe development defaults must never accidentally become production defaults.

---

# Cookie Configuration

Authentication cookie settings may include:

```text id="c5k1v7"
COOKIE_SECURE
COOKIE_DOMAIN
COOKIE_PATH
COOKIE_SAMESITE
```

Production should use:

```text id="d8p3r0"
Secure
HttpOnly
appropriate SameSite
```

The exact values depend on the deployment architecture.

---

# Session Configuration

Session configuration should control values such as:

```text id="e6w2n9"
session lifetime
session idle timeout
session renewal policy
cookie behavior
revocation behavior
```

Session security configuration must be centralized rather than duplicated across handlers.

---

# Rate-Limit Configuration

Rate limits should be configurable for environments where infrastructure capacity differs.

Potential settings include:

```text id="f0m5q8"
login rate limit
registration rate limit
password reset rate limit
OAuth initiation rate limit
account-linking rate limit
public booking rate limit
```

Defaults should protect the application rather than maximize raw request throughput.

---

# Logging Configuration

Logging configuration may include:

```text id="g7r1s4"
LOG_LEVEL
LOG_FORMAT
```

Production logs should generally be structured.

Debug logging should not expose sensitive request or authentication data.

---

# Error Configuration

Production error responses should not expose development diagnostics.

Configuration should distinguish safe client errors from internal diagnostics.

The environment may influence logging detail, but it must not weaken authorization or authentication rules.

---

# CORS Configuration

CORS origins should be explicit.

Possible configuration:

```text id="h3v9c2"
CORS_ALLOWED_ORIGINS
```

Production must use known origins.

Do not use a wildcard origin for authenticated traffic merely for convenience.

---

# URL Configuration

Important URLs should be configuration-driven.

Examples:

```text id="j0n6t5"
APP_BASE_URL
WEB_BASE_URL
API_BASE_URL
GOOGLE_OAUTH_REDIRECT_URL
APPLE_OAUTH_REDIRECT_URL
```

The backend should not silently construct security-sensitive redirect URLs from arbitrary request headers.

---

# Configuration Validation Rules

Examples of startup checks:

```text id="k4p8w1"
DATABASE_URL exists
API port is valid
production base URL uses HTTPS
OAuth transaction key is valid
enabled provider has required configuration
OAuth redirect URL is valid
Apple private key parses successfully when Apple is enabled
```

Configuration errors should produce clear startup failures.

---

# Secrets and Logs

Configuration loaders must never print complete secret values during startup.

Safe logging may indicate:

```text id="m2x7q6"
Google OAuth configured: yes
Apple OAuth configured: no
```

It must not output:

```text id="n5c3v9"
GOOGLE_OAUTH_CLIENT_SECRET=...
APPLE_PRIVATE_KEY_PEM=...
```

Even partial secret values should generally be avoided.

---

# Configuration Testing

Configuration code should be tested for:

- missing required values
- malformed values
- unsupported providers
- invalid URLs
- invalid durations
- invalid cryptographic keys
- incomplete provider configuration
- production security requirements

Tests should not use real provider credentials.

---

# Configuration and CI/CD

CI/CD must inject required configuration through secure mechanisms.

Secrets must not appear in:

- workflow files
- source code
- build output
- test logs
- artifacts
- pull-request comments

Non-secret configuration may be explicitly defined in pipeline configuration when appropriate.

---

# Configuration and Docker

Docker Compose may provide local development configuration.

Local Compose files must not embed production secrets.

For example, development PostgreSQL credentials may exist as local non-production values when needed.

Production deployment configuration is separate.

---

# Configuration and Infrastructure

As infrastructure evolves, configuration may eventually be supplied through:

- container environment
- managed secret stores
- infrastructure variables
- deployment platform secrets

The application configuration interface should remain stable where practical.

---

# Configuration Changes

Configuration changes must be reviewed when they affect:

- security
- authentication
- OAuth
- sessions
- payments
- database access
- external integrations
- tenant isolation

A configuration change can be a security change even when no application source code changes.

---

# Manual OAuth Configuration Boundary

Real Google and Apple credentials require manual developer-console setup.

The developer must configure:

```text id="p8j4d7"
Google application
Google redirect URL
Apple service/application identifier
Apple redirect URL
Apple Team ID
Apple Key ID
Apple private key
```

The application repository must never contain the real secret values.

---

# Current OAuth Configuration Status

OAuth configuration support exists at the code/foundation level.

The provider configuration model supports:

```text id="q1v6m3"
Google client ID
Google client secret
Google redirect URL
Apple client ID
Apple Team ID
Apple Key ID
Apple private key
Apple redirect URL
OAuth transaction key
```

Real provider credentials have not yet been verified as production-ready.

Live Google and Apple provider verification remains part of the authentication integration work.

---

# Current Phase 3 Configuration Boundary

Phase 3 configuration includes the values required for:

- password authentication
- sessions
- Google OAuth
- Apple OAuth
- authentication security

Phase 3 does not finalize production infrastructure secret management.

Production secret storage and deployment controls will be strengthened during the deployment and security phases.

---

# Core Principles

Configuration must remain:

```text id="r5w9h2"
Explicit
Validated
Environment-aware
Server-controlled
Secret-safe
Reproducible
Minimal
```

Application logic should depend on validated configuration rather than directly reading raw environment variables throughout the codebase.
