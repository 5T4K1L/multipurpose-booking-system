# Architecture Decisions

## ADR-0001 — Use a Modular Monolith

Status: Accepted

The initial product will use a modular monolith rather than microservices.

Reason:

- simpler development
- lower infrastructure requirements
- easier local development
- easier debugging
- lower operational complexity
- appropriate for the current product stage

Future decomposition may occur only when justified by actual system requirements.

---

## ADR-0002 — PostgreSQL as Primary Data Store

Status: Accepted

PostgreSQL will be the primary relational database and source of truth for application data.

---

## ADR-0003 — Go Backend

Status: Accepted

The backend API will be implemented in Go.

Primary database driver:

pgx

SQL generation:

sqlc

---

## ADR-0004 — Next.js Frontend

Status: Accepted

The primary customer-facing web application will use Next.js, React, and TypeScript.

---

## ADR-0005 — Responsive UI from the Beginning

Status: Accepted

Responsive behavior is a requirement for every user-facing feature.

Responsive work must not be postponed until the end of development.

---

## ADR-0006 — Vertical Feature Slices

Status: Accepted

Features should be implemented as small, independently verifiable vertical slices.

Where appropriate, a slice should include:

- database changes
- backend logic
- API contract
- frontend behavior
- validation
- tests
- documentation

Avoid implementing large incomplete subsystems in isolation.
