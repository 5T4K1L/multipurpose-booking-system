# Frontend Architecture

## Purpose

This document defines the architecture and development conventions for the Next.js frontend.

The goal is to keep the frontend maintainable, responsive, accessible, performant, and suitable for the long-term Universal Booking Platform.

## Framework

The frontend uses:

- Next.js
- React
- TypeScript
- Tailwind CSS
- shadcn/ui when appropriate

The Next.js App Router is the primary routing model.

## Frontend Responsibilities

The frontend is responsible for:

- user interface
- navigation
- user interaction
- client-side state where required
- form interaction
- client-side presentation
- API communication
- responsive behavior
- accessibility
- frontend validation where appropriate

The frontend is not responsible for:

- authoritative business rules
- database access
- authorization decisions
- payment authorization
- tenant isolation
- security enforcement

The backend remains authoritative for security and business rules.

## Application Structure

The frontend should evolve toward a structure similar to:

```text
apps/web/
├── app/
├── components/
├── features/
├── lib/
├── hooks/
├── types/
└── public/
```

Exact directories should only be introduced when there is a real need for them.

Avoid creating empty organizational layers solely for hypothetical future features.

## App Router

Next.js App Router is used for:

- routes
- layouts
- loading states
- error states
- server components
- client components

Routes should remain organized around user-facing application areas.

## Server Components

Server Components should be preferred when client-side interactivity is not required.

Use Client Components when the component requires browser-only behavior such as:

- state
- event handlers
- effects
- browser APIs
- interactive UI

Do not mark entire route trees as Client Components unnecessarily.

## Client Components

Client Components should remain as small and focused as practical.

Interactive components should not pull unrelated application state into the client unnecessarily.

## Component Architecture

Components should generally be organized by responsibility.

Examples:

```text
components/
├── ui/
├── layout/
└── shared/
```

Feature-specific components should live with their feature when that improves ownership and maintainability.

Example:

```text
features/
└── bookings/
    ├── components/
    ├── hooks/
    ├── api/
    └── types/
```

Do not create highly generic components until a repeated use case actually exists.

## UI Components

Shared UI components should provide consistent:

- spacing
- typography
- interaction behavior
- states
- accessibility
- responsive behavior

shadcn/ui may be used when it provides a useful foundation.

Components should remain customizable rather than tightly coupling business logic to UI primitives.

## Design System

The product should progressively develop a consistent design system.

The design system should define:

- typography
- spacing
- colors
- borders
- radii
- shadows
- controls
- feedback states
- responsive behavior

Do not introduce a large design system abstraction before the product has enough UI to justify it.

## Responsive Design

Responsive design is mandatory for all user-facing features.

The implementation should be mobile-first.

Representative target widths include:

```text
320px
375px
390px
414px
768px
1024px
1280px
1440px
1920px+
```

Layouts should adapt rather than simply shrinking desktop layouts.

Responsive behavior must consider:

- touch interaction
- readable typography
- navigation
- forms
- tables
- dialogs
- cards
- calendars
- messaging interfaces
- booking flows
- accessibility

## Accessibility

User interfaces should follow accessible web practices.

Consider:

- semantic HTML
- keyboard navigation
- focus states
- appropriate labels
- screen-reader support
- sufficient contrast
- usable touch targets
- error identification

Accessibility should be implemented during feature development rather than postponed.

## API Communication

The frontend communicates with the Go backend through HTTP APIs.

The frontend must not connect directly to PostgreSQL.

Intended flow:

```text
Next.js
   ↓
HTTP API
   ↓
Go
   ↓
PostgreSQL
```

API communication should be centralized where practical rather than duplicating request logic throughout components.

## API Client

A small API client layer may be introduced when multiple frontend features require shared request behavior.

It should eventually centralize concerns such as:

- base URL
- headers
- request handling
- response parsing
- authentication context
- API errors

Do not create a complex API abstraction before it is necessary.

## Data Fetching

Data fetching strategy should depend on the type of data.

Prefer server-side fetching when appropriate.

Use client-side fetching when the interface requires:

- live interaction
- frequent updates
- browser-driven state
- user-triggered requests

Do not introduce a large client-state library without an actual requirement.

## State Management

Prefer the smallest state mechanism that solves the problem.

Use:

- local React state for local UI state
- URL state for shareable/filterable navigation state
- server state for server-provided data
- shared state only when multiple components genuinely need it

Avoid creating global state for values that belong to one component or feature.

## Forms

Forms should provide:

- clear labels
- validation
- useful error messages
- loading states
- success feedback
- disabled states during submission where appropriate

Server-side validation remains authoritative.

Client-side validation improves user experience but must never replace server validation.

## Loading States

User-facing async operations should have appropriate loading states.

Avoid leaving users with apparently frozen interfaces.

Use:

- loading indicators
- skeletons
- disabled controls
- optimistic updates only when safe

## Error States

User-facing errors should be understandable and actionable where possible.

Do not expose:

- stack traces
- SQL errors
- internal infrastructure details
- secrets
- sensitive diagnostics

## Empty States

Pages that can legitimately have no data should provide intentional empty states.

Example:

```text
No bookings yet.
```

Where appropriate, include a clear next action.

## Notifications and Feedback

The frontend should provide feedback for important actions.

Examples:

- success confirmation
- validation errors
- failed requests
- destructive-action warnings

Do not overuse notifications for minor interactions.

## Navigation

Navigation should reflect the user's role and context once role-based product areas are implemented.

Navigation must remain usable across supported screen sizes.

## URL State

Filters, pagination, and other shareable page state should use URL parameters when appropriate.

Example:

```text
/bookings?status=confirmed&page=2
```

URL parameters must be validated by the application.

## Performance

Frontend implementations should consider:

- bundle size
- unnecessary client JavaScript
- image optimization
- rendering cost
- network requests
- caching
- loading performance

Prefer simple implementations before introducing performance infrastructure.

## Security Boundary

The frontend must never be treated as a trusted security boundary.

The backend must independently enforce:

- authentication
- authorization
- ownership
- tenant isolation
- validation
- financial rules

when those capabilities are implemented.

## Feature Development

User-facing features should be developed as vertical slices.

A feature should, where appropriate, include:

- UI
- API integration
- validation
- loading state
- error state
- responsive behavior
- accessibility
- tests

Do not build large frontend subsystems disconnected from their backend requirements.

## Testing

Frontend testing should eventually cover:

- component behavior
- user interactions
- important flows
- API integration behavior
- responsive behavior where practical

End-to-end testing will be introduced when the application has meaningful user flows.

## Code Quality

Frontend code should favor:

- clear naming
- small components
- explicit behavior
- reusable logic where justified
- TypeScript type safety
- minimal duplication
- minimal unnecessary abstraction

Avoid overly clever code.

## Phase 1 Boundary

Phase 1 defines architecture and conventions only.

The following remain outside the current scope:

- authentication UI
- registration
- login
- booking UI
- customer accounts
- business dashboards
- CRM
- messaging
- payments
- marketplace
- search
- notifications

## Architectural Rule

The frontend should remain simple enough to evolve.

Do not introduce libraries, abstractions, state systems, or design-system infrastructure unless a real current requirement justifies them.

## Core Principle

Every feature should be:

```text
Responsive
Accessible
Secure
Testable
API-connected
Maintainable
```

from the beginning.
