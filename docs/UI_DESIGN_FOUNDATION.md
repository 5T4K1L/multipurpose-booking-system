# UI/UX design foundation - 2026-10-19

First bounded part of TO_REVISE.MD. This is not a completed whole-site redesign.
The user reported 63% usage before this task. Remaining usage is unavailable to
this agent. Outstanding work stays in TO_REVISE.MD for a later reset.

## Delivered

- Shared warm-ivory/ink/vermilion visual tokens: surface, canvas, border, ink, accent,
  warning/danger, spacing, typography, radius and shadow conventions. Existing
  Tailwind color utilities resolve to semantic theme values while pages migrate.
  White text on primary actions stays white in both themes.
- Persistent Light/Dark/System appearance control, system-preference updates and
  pre-hydration theme initialization. Storage failures fall back to an in-memory
  choice. Cross-tab changes synchronize through storage events.
- Responsive shared header, active links, mobile disclosure navigation that closes
  on route changes, footer, skip link and keyboard focus indicators.
- Restrained hover/focus feedback and page entrance motion. All animation and
  transitions respect prefers-reduced-motion. Subtle translucency is limited to
  the header; no new animation/icon dependencies were added.
- Rebuilt home page with clear customer and business paths; existing health check
  remains visible as a service-availability message rather than the main headline.
- Refined login/register/logout/account cards and form presentation. Existing
  handlers are unchanged; Google/Apple discovery and password behavior preserved.
- Refined business discovery hierarchy, search form, cards and skeleton loading.
- Shared business form/button/panel styles adopt the design foundation. Existing
  complex workspace structures remain for a later dedicated UX pass.

## Verification

- Frontend lint and production build passed (existing Google Fonts download).
- Seven routes checked in light/dark at 320/390/768/1280px without document overflow
  or page errors: home, login, register, directory, business setup, customers and history.
- Theme persistence, live system preference, reduced motion, mobile navigation,
  password visibility and keyboard skip-link focus verified.
- Real registration and password login through UI, followed by authenticated CRM
  search, empty-state and booking-count checks passed in dark mode.
- Cancellation and rescheduling regression fixtures passed, including lost-response
  retries, history, business visibility and occupancy transfer/release.
- Desktop light home, mobile dark home/login and authenticated dark customer-table
  screenshots inspected. Temporary browser fixture records were removed.

Backend, API, database, booking policies and persisted business configuration were
not edited. No backend tests were rerun for this frontend-only revision. This is
not a full accessibility audit or a claim of complete cross-browser visual coverage.

## Remaining

Use TO_REVISE.MD as the outstanding revision list. Business setup/navigation,
public booking flow, customer history/detail, operator/calendar/CRM page layouts,
complex component states and broader accessibility/browser QA remain. After these
revisions, the user's new preference is to continue Phase 8 without UI first.

## Files edited

- apps/web/app/globals.css
- apps/web/app/layout.tsx
- apps/web/app/components/site-header.tsx (new)
- apps/web/app/page.tsx
- apps/web/app/login/page.tsx
- apps/web/app/login/oauth-buttons.tsx
- apps/web/app/register/page.tsx
- apps/web/app/logout/page.tsx
- apps/web/app/account/page.tsx
- apps/web/app/business/styles.ts
- apps/web/app/businesses/page.tsx
- TO_REVISE.MD
- docs/UI_DESIGN_FOUNDATION.md (new)
- docs/CURRENT_STATE.md


## Design-only refinement - 2026-10-19

The user clarified this is visual design for the eventual public product, not deployment.
The palette now uses ivory, charcoal and vermilion, including warm dark surfaces.
Buttons and cards have restrained hover/press motion with reduced-motion support.

The shared header shows Workspace only for accounts with business memberships.
Business members have a responsive workspace sidebar; nonmembers visiting a workspace
subpage see a personal-bookings/setup entry screen. These are presentation choices,
not replacements for the existing API authorization checks.

Business settings are grouped into Services & branches, People & resources,
Hours & availability, and Profile & publishing. Native disclosure panels keep
forms mounted, preserving draft registration and publish-save behavior. Duplicate
page menus and the embedded duplicate booking workspace were removed; the existing
standalone booking route remains available from the workspace navigation.

Validation: frontend lint/build; seven public/auth routes at 320/390/768/1280px
in both themes; theme persistence/system changes, reduced motion, mobile navigation,
password visibility and keyboard skip link. Authenticated checks cover customer/owner
navigation, grouped settings, CRM search and booking-count changes. No backend edits.
Full booking-flow visual redesign and broader accessibility/browser review remain in
TO_REVISE.MD. This is not a declaration of production launch readiness.

### Files changed in this refinement

- apps/web/app/globals.css
- apps/web/app/layout.tsx
- apps/web/app/components/navigation-session.tsx (new)
- apps/web/app/components/site-header.tsx
- apps/web/app/account/page.tsx
- apps/web/app/login/page.tsx
- apps/web/app/logout/page.tsx
- apps/web/app/business/layout.tsx (new)
- apps/web/app/business/page.tsx
- apps/web/app/business/catalog.tsx
- apps/web/app/business/bookings/page.tsx
- apps/web/app/business/calendar/page.tsx
- apps/web/app/business/customers/page.tsx
- apps/web/app/my-bookings/workspace.tsx
- docs/UI_DESIGN_FOUNDATION.md
- docs/CURRENT_STATE.md
- TO_REVISE.MD
