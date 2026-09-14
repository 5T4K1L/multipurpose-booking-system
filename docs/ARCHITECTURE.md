# Architecture

## Current Architecture Status

Architecture foundation only.

The actual application architecture will be implemented during Phase 0.

## Target Initial Architecture

    Browser
       │
       ▼
    Next.js Web Application
       │
       ▼
    Go HTTP API
       │
       ▼
    PostgreSQL

## Architectural Style

The initial system is a modular monolith.

The codebase must maintain clear boundaries between:

- frontend
- HTTP/API layer
- application/domain logic
- data access
- database

## Frontend

Responsibilities:

- user interface
- client-side interaction
- form handling
- presentation
- API communication

## Backend

Responsibilities:

- HTTP API
- validation
- application logic
- database access
- error handling

## Database

PostgreSQL is the primary source of truth.

Database changes must use versioned SQL migrations.

## Phase 0 Boundary

Phase 0 must not introduce:

- authentication
- authorization
- businesses
- organizations
- customers
- bookings
- services
- staff
- payments
- messaging
- marketplace functionality
- background jobs
- search
- production infrastructure
