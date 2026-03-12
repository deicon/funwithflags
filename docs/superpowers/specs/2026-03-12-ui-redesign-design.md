# FunWithFlags UI Redesign — Design Spec

**Date:** 2026-03-12
**Status:** Draft
**Scope:** Full frontend rewrite — redesign all existing views + add flag detail page

## Context

FunWithFlags is a feature flag management service (Go backend + React frontend). The current frontend is functional but visually dated (plain CSS, no component library) and the flag management UX is clunky (modal-based CRUD). The backend API remains unchanged — this spec covers the frontend only.

**Target audience:** Mixed technical and non-technical users — developers, PMs, and QA. Note: all backend admin endpoints require the `admin` role. Non-admin users currently have no accessible views beyond login. Role-based access (read-only views for non-admin users) is out of scope for this redesign — all pages require admin auth.

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Approach | Full rewrite | Current frontend is small (~6 components, ~480 lines CSS). Incremental migration creates more friction than starting fresh. |
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
- Stats footer: stage count, flag count (requires additional API calls per project — fetch stages list and flags count after loading projects)

Actions: search bar, "+ New Project" button. Create/edit via modal dialog.

### Stages Page

Card grid within a project context. Breadcrumb: `Projects > {Project Name}`.

Each stage card shows:
- Colored dot (visual indicator per environment type)
- Stage name + key
- Description
- Stats footer: enabled/disabled flag counts (fetched per stage from the flags list endpoint)

Actions: search bar, "+ New Stage" button, "Edit Project" secondary button. Create/edit via modal dialog.

### Flags Page

List layout with inline toggle switches. Breadcrumb: `Projects > {Project} > {Stage} > Flags`. Breadcrumbs are hand-crafted per page (not auto-derived from URL segments) to keep them concise — "Stages" is omitted as a level since the stage name is shown directly.

Each flag row shows:
- Toggle switch for `enabled` field (optimistic update). The `active` field (temporal validity) is shown on the flag detail page, not the list.
- Flag key (bold) + description
- Type badge — derived from the default variation's `VariationType` (boolean, string, number, object)
- Default variation value
- Chevron indicating clickable → navigates to detail page

Actions: search bar, status filter dropdown, type filter dropdown, "+ New Flag" button. Create via modal (key, name, description, type, default variation).

### Flag Detail Page (new)

Two-column layout. Breadcrumb: `Projects > {Project} > {Stage} > Flags > {Flag Key}`.

**Header:** Flag key as title, toggle switch for `enabled` field, description below. Delete button (secondary/destructive).

**Left column:**
- **Details card** — Name, key (monospace), description. "Edit Details" button opens inline editing.
- **Temporal Validity card** — Shows `active` status badge, `validFrom`/`validTo` dates. "Activate"/"Deactivate" button calls the corresponding backend endpoint. Displays the current validity window clearly.
- **Variations card** — List of variations with key, type, value, description, and "default" badge on the default variation. "+ Add" to create new variations.

**Right column:**
- **Targeting Rules card** — List of rules showing: rule description, and either a target variation (`variationKey`) or a percentage rollout with progress bar (these are mutually exclusive per rule). Conditions displayed as attribute/operator/value in code-style badges. "+ Add Rule" to create new rules.
- **Info card** — Created date, updated date, flag ID.

### Users Page

Table layout with avatar initials, columns: username, role (colored badge), actions. (Note: the backend `User` model only returns `username` and `role` — no `createdAt` field is exposed.)

Actions: search bar, "+ New User" button. Inline actions per row: "Reset PW" (opens a dialog prompting for a new password, calls `PUT /api/v1/admin/users/{username}`), "Delete" (destructive, except for current user). Create via modal (username, password, role). Role change via dropdown in table row.

## Component Patterns

### Shared Components

| Component | Purpose |
|-----------|---------|
| `AppLayout` | Top nav (logo, nav links, theme toggle, user menu) + breadcrumb bar + content slot |
| `ThemeToggle` | Sun/moon icon button, toggles `dark` class on `<html>` |
| `Breadcrumb` | Displays breadcrumb trail; each page provides its own breadcrumb segments via route metadata |
| `SearchInput` | Debounced text input for filtering lists |
| `ConfirmDialog` | Reusable confirmation for destructive actions |
| `EmptyState` | Friendly message + action button when lists are empty |

### Interaction Patterns

- **Create/Edit simple entities** (projects, stages, users): Modal dialogs with form fields
- **Edit flag details**: Inline editing sections on the flag detail page
- **Delete anything**: Always requires `ConfirmDialog`
- **Toggle flag `enabled`**: Optimistic update with rollback on error (via `PUT /api/v1/admin/flags/{id}`)
- **Activate/deactivate flag**: On the flag detail page, calls `POST .../flags/{id}/activate` or `.../flags/{id}/deactivate`
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

Each page fetches its own data on mount via `useEffect` + API client. No global store. Pages show a skeleton loader during initial fetch.

| Page | Endpoint |
|------|----------|
| ProjectsPage | `GET /api/v1/admin/projects` |
| StagesPage | `GET /api/v1/admin/projects/{project}/stages` |
| FlagsPage | `GET /api/v1/admin/{project}/{stage}/flags` |
| FlagDetailPage | `GET /api/v1/admin/{project}/{stage}/flags/{key}` |
| UsersPage | `GET /api/v1/admin/users` |

Note: The flag detail route uses `:flagKey` in the URL (`/projects/:projectKey/stages/:stageKey/flags/:flagKey`). The page fetches via the key-based endpoint `GET /api/v1/admin/{project}/{stage}/flags/{key}`, which returns the flag including its numeric ID needed for mutations.

### Mutations

Create, update, and delete operations call the corresponding API endpoint, then refetch the list or update local state on success. Only the flag `enabled` toggle uses optimistic updates.

**Auth endpoints:**

| Action | Endpoint |
|--------|----------|
| Login | `POST /api/v1/auth/login` |
| Refresh token | `POST /api/v1/auth/refresh` |
| Logout | `POST /api/v1/auth/logout` |

**Project endpoints:**

| Action | Endpoint |
|--------|----------|
| Create project | `POST /api/v1/admin/projects` |
| Update project | `PUT /api/v1/admin/projects/{project}` |
| Delete project | `DELETE /api/v1/admin/projects/{project}` |

**Stage endpoints:**

| Action | Endpoint |
|--------|----------|
| Create stage | `POST /api/v1/admin/projects/{project}/stages` |
| Update stage | `PUT /api/v1/admin/projects/{project}/stages/{stage}` |
| Delete stage | `DELETE /api/v1/admin/projects/{project}/stages/{stage}` |

**Flag endpoints:**

| Action | Endpoint |
|--------|----------|
| Create flag | `POST /api/v1/admin/{project}/{stage}/flags` |
| Update flag | `PUT /api/v1/admin/flags/{id}` |
| Delete flag | `DELETE /api/v1/admin/flags/{id}` |
| Activate flag | `POST /api/v1/admin/flags/{id}/activate` |
| Deactivate flag | `POST /api/v1/admin/flags/{id}/deactivate` |

**User endpoints:**

| Action | Endpoint |
|--------|----------|
| Create user | `POST /api/v1/admin/users` |
| Update user | `PUT /api/v1/admin/users/{username}` |
| Delete user | `DELETE /api/v1/admin/users/{username}` |

**Note on variations and rules:** Variations and targeting rules are sub-resources of the flag. They are managed as part of the flag payload via `PUT /api/v1/admin/flags/{id}` — there are no separate endpoints for individual variation or rule CRUD.

### Error Handling

- **Mutation feedback**: Toast notifications via Sonner (shadcn/ui integration) for success and error states
- **Form validation**: Inline error messages below fields
- **Auth errors**: Redirect to `/login` on 401

### Auth Flow

Same as current: `AuthContext` wraps the app, stores JWT in `localStorage`, provides `login`/`logout`/`isAuthenticated`. React Router route guards redirect unauthenticated users to `/login`.

## Loading & Error States

- **Initial page load**: Skeleton loaders matching the page layout (card grid skeletons for projects/stages, row skeletons for flags/users)
- **Mutation success**: Toast notification (green) via Sonner
- **Mutation error**: Toast notification (red) with error message
- **Form validation**: Inline error messages below the invalid field
- **401 Unauthorized**: Redirect to `/login`, clear stored tokens
- **403 Forbidden**: Toast notification explaining insufficient permissions
- **Empty lists**: `EmptyState` component with friendly message and action button (e.g., "No flags yet. Create your first flag.")

## Out of Scope

- Dashboard/overview page (future enhancement)
- Audit log view (future — backend support exists)
- Project settings page (future)
- Role-based access for non-admin users (future — requires backend changes)
- Temporal range management (listing/creating multiple validity ranges per flag — future)
- Mobile-optimized responsive design
- Dark/light theme auto-switching by time of day
- Internationalization
