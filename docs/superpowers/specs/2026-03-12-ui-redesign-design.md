# FunWithFlags UI Redesign — Design Spec

**Date:** 2026-03-12
**Status:** Draft
**Scope:** Full frontend rewrite — redesign all existing views + add flag detail page

## Context

FunWithFlags is a feature flag management service (Go backend + React frontend). The current frontend is functional but visually dated (plain CSS, no component library) and the flag management UX is clunky (modal-based CRUD). The backend API remains unchanged — this spec covers the frontend only.

**Target audience:** Mixed technical and non-technical users — developers, PMs, and QA.

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Approach | Full rewrite | Current frontend is small (~6 components, ~150 lines CSS). Incremental migration creates more friction than starting fresh. |
| Component library | shadcn/ui (Radix + Tailwind) | Pre-built accessible components with full ownership. Built-in dark mode support. |
| Styling | Tailwind CSS v4 | Utility-first, pairs with shadcn/ui, CSS variable theming. |
| Routing | React Router v7 | URL-based navigation replacing panel switching. Deep links and browser nav work. |
| Navigation | Top nav + breadcrumbs | Clean horizontal nav, no sidebar. Breadcrumbs provide context (Project > Stage > Flags > Flag). |
| Flag editing | Dedicated detail page | Two-column layout with room for variations, targeting rules, and rollout config. Replaces modal-based editing. |
| Themes | Dark + light from day one | Dark: blue accent on charcoal. Light: teal accent on green-tinted gray. Toggle in nav bar. |
| Desktop focus | Desktop-first | Mobile can be usable but is not a priority. |

## Architecture

### Tech Stack

- React 18 + TypeScript + Vite (same build tooling)
- Tailwind CSS v4
- shadcn/ui (Radix UI primitives)
- React Router v7
- No additional state management — React Context + local state

### Route Structure

```
/login                                                    → Login page
/projects                                                 → Projects list
/projects/:projectKey/stages                              → Stages list
/projects/:projectKey/stages/:stageKey/flags              → Flags list
/projects/:projectKey/stages/:stageKey/flags/:flagKey     → Flag detail
/users                                                    → Users management (admin only)
```

### File Structure

```
frontend/src/
├── components/
│   ├── ui/            ← shadcn/ui primitives (button, card, dialog, input, table, switch, etc.)
│   ├── layout/        ← AppLayout, Breadcrumb, ThemeToggle
│   └── shared/        ← SearchInput, ConfirmDialog, EmptyState
├── pages/
│   ├── LoginPage.tsx
│   ├── ProjectsPage.tsx
│   ├── StagesPage.tsx
│   ├── FlagsPage.tsx
│   ├── FlagDetailPage.tsx
│   └── UsersPage.tsx
├── api/
│   └── client.ts      ← Fetch wrapper with JWT auth (carried over from current code)
├── context/
│   └── AuthContext.tsx ← JWT state management (carried over from current code)
├── hooks/
│   └── useAuth.ts
├── lib/
│   └── utils.ts       ← cn() helper for Tailwind class merging
├── types.ts
└── main.tsx
```

## Theme System

Two themes controlled by a `dark` class on `<html>`. CSS variables follow shadcn/ui conventions. Theme preference stored in `localStorage`, defaults to system preference via `prefers-color-scheme`.

### Dark Theme (Blue on Charcoal)

| Token | Value | Usage |
|-------|-------|-------|
| background | `#0f1117` | Page background |
| card | `#161b22` | Cards, nav bar, surfaces |
| border | `#21262d` | All borders |
| foreground | `#e6edf3` | Primary text |
| muted-foreground | `#7d8590` | Secondary text |
| muted | `#484f58` | Tertiary text, placeholders |
| primary | `#2f81f7` | Accent — buttons, active nav, links |
| primary-foreground | `#ffffff` | Text on primary |
| destructive | `#f85149` | Delete actions, disabled status |
| success | `#3fb950` | Enabled status, success states |
| warning | `#f0883e` | Admin badge, caution states |

### Light Theme (Teal on Green-tinted Gray)

| Token | Value | Usage |
|-------|-------|-------|
| background | `#f0fdf4` | Page background (green tint) |
| card | `#ffffff` | Cards, nav bar, surfaces |
| border | `#d1d5db` | All borders |
| foreground | `#111827` | Primary text |
| muted-foreground | `#6b7280` | Secondary text |
| muted | `#9ca3af` | Tertiary text, placeholders |
| primary | `#0d9488` | Accent — buttons, active nav, links |
| primary-foreground | `#ffffff` | Text on primary |
| destructive | `#ef4444` | Delete actions, disabled status |
| success | `#22c55e` | Enabled status, success states |
| warning | `#f59e0b` | Admin badge, caution states |

## Page Designs

### Login Page

Standalone layout (no nav bar). Centered card on dark/light background.

- App logo + name + subtitle ("Sign in to manage your feature flags")
- Username and password fields
- Sign in button (primary color)
- Error message displayed inline below form on failure

### Projects Page

Card grid layout. Each project card shows:
- Project name (bold) + key (muted, monospace)
- Description
- Stats footer: stage count, flag count

Actions: search bar, "+ New Project" button. Create/edit via modal dialog.

### Stages Page

Card grid within a project context. Breadcrumb: `Projects > {Project Name}`.

Each stage card shows:
- Colored dot (visual indicator per environment type)
- Stage name + key
- Description
- Stats footer: enabled/disabled flag counts

Actions: search bar, "+ New Stage" button, "Edit Project" secondary button. Create/edit via modal dialog.

### Flags Page

List layout with inline toggle switches. Breadcrumb: `Projects > {Project} > {Stage} > Flags`.

Each flag row shows:
- Toggle switch (enable/disable — optimistic update)
- Flag key (bold) + description
- Type badge (boolean, string, number, object)
- Default variation value
- Chevron indicating clickable → navigates to detail page

Actions: search bar, status filter dropdown, type filter dropdown, "+ New Flag" button. Create via modal (key, name, description, type, default variation).

### Flag Detail Page (new)

Two-column layout. Breadcrumb: `Projects > {Project} > {Stage} > Flags > {Flag Key}`.

**Header:** Flag key as title, toggle switch for enable/disable, description below. Delete button (secondary/destructive).

**Left column:**
- **Details card** — Name, key (monospace), type, valid from/to dates. "Edit Details" button opens inline editing.
- **Variations card** — List of variations with key, description, and "default" badge on the default variation. "+ Add" to create new variations.

**Right column:**
- **Targeting Rules card** — List of rules showing: rule name, target variation, conditions (attribute/operator/value in code-style badges), rollout percentage with progress bar. "+ Add Rule" to create new rules.
- **Info card** — Created date, updated date, flag ID.

### Users Page (Admin only)

Table layout with avatar initials, columns: username, role (colored badge), created date, actions.

Actions: search bar, "+ New User" button. Inline actions per row: "Reset PW" (secondary), "Delete" (destructive, except for current user). Create via modal (username, password, role). Role change via dropdown in table row.

## Component Patterns

### Shared Components

| Component | Purpose |
|-----------|---------|
| `AppLayout` | Top nav (logo, nav links, theme toggle, user menu) + breadcrumb bar + content slot |
| `ThemeToggle` | Sun/moon icon button, toggles `dark` class on `<html>` |
| `Breadcrumb` | Auto-generated from React Router location + route metadata |
| `SearchInput` | Debounced text input for filtering lists |
| `ConfirmDialog` | Reusable confirmation for destructive actions |
| `EmptyState` | Friendly message + action button when lists are empty |

### Interaction Patterns

- **Create/Edit simple entities** (projects, stages, users): Modal dialogs with form fields
- **Edit flag details**: Inline editing sections on the flag detail page
- **Delete anything**: Always requires `ConfirmDialog`
- **Toggle flag enabled/disabled**: Optimistic update with rollback on error
- **Navigation**: Click card/row → React Router navigation → URL changes → breadcrumb updates

## Data Flow & API Integration

### API Client

Carry over the existing fetch wrapper from `api/client.ts`. It handles:
- JWT access/refresh token management
- Auto-refresh 60 seconds before expiry
- Auth headers on all requests
- Token storage in `localStorage`

No changes to the backend API.

### Data Fetching

Each page fetches its own data on mount via `useEffect` + API client. No global store.

| Page | Endpoint |
|------|----------|
| ProjectsPage | `GET /api/v1/projects` |
| StagesPage | `GET /api/v1/projects/:key/stages` |
| FlagsPage | `GET /api/v1/admin/flags?project=X&stage=Y` |
| FlagDetailPage | `GET /api/v1/admin/flags/:id` |
| UsersPage | `GET /api/v1/auth/users` |

### Mutations

Create, update, and delete operations call the corresponding API endpoint, then refetch the list or update local state on success. Only the flag toggle uses optimistic updates.

### Error Handling

- **Mutation feedback**: Toast notifications via Sonner (shadcn/ui integration) for success and error states
- **Form validation**: Inline error messages below fields
- **Auth errors**: Redirect to `/login` on 401

### Auth Flow

Same as current: `AuthContext` wraps the app, stores JWT in `localStorage`, provides `login`/`logout`/`isAuthenticated`. React Router route guards redirect unauthenticated users to `/login`.

## Out of Scope

- Dashboard/overview page (future enhancement)
- Audit log view (future — backend support exists)
- Project settings page (future)
- Mobile-optimized responsive design
- Dark/light theme auto-switching by time of day
- Internationalization
