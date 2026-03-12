# CLAUDE.md

## Role
Reference AGENTS.md for build/test commands, coding style, naming, and PR conventions.
This file covers architecture awareness and AI-specific guidance.

## Architecture Awareness
- Single Go binary serves REST API (/api/v1/*) and frontend static files (React SPA from disk)
- Entry point: cmd/server/main.go → internal/app/app.go wires all dependencies
- Repository pattern: interfaces in domain packages, implementations in same package (memory + postgres)
- Service layer wraps repositories with business logic
- HTTP handlers in internal/httpserver/, grouped by domain
- Frontend in frontend/ — built separately, copied into Go container image

## Key Domain Concepts
- Projects contain Stages; Flags are scoped to a Project+Stage pair
- Flags support temporal ranges: multiple validity windows with active/inactive state
- Flag evaluation: rules → target conditions → percentage rollouts → default variation
- Audit trail captures all flag changes with before/after JSONB values

## Patterns to Follow
- Repository pattern for any new storage concern (define interface, implement memory + postgres)
- Table-driven tests beside every package
- Composite keys for project/stage scoping
- HTTP handlers follow: parse request → call service → write JSON response
- Error types defined per package in errors.go
- Environment variables for all configuration (see internal/app/app.go)

## Things to Avoid
- Do not add an ORM — the project uses pgx directly, keep it that way
- Do not bypass or weaken auth middleware
- Do not add external HTTP client libraries — use net/http
- Do not add frontend dependencies without discussion (keep the SPA lightweight)
- Do not use embed.FS for frontend — files are served from disk at runtime
- Do not create Fly.io configs — deployment target is Hetzner/Podman

## Deployment Context
- Production: Hetzner Cloud VPS
- Containers: Podman with systemd units and auto-update
- Reverse proxy: Caddy (TLS termination)
- Database: PostgreSQL 16 in Podman container
- Backups: pg_dump to Hetzner Object Storage (S3-compatible)
- Local dev: docker-compose.yml (still uses Docker for convenience)

## Frontend
- React + Vite + TypeScript SPA in frontend/
- UI redesign planned separately (using Stitch)
- API calls go through frontend/src/api/client.ts (fetch wrapper with Bearer token)
- Auth state managed via React Context (AuthContext)
- Styling: CSS

## Notes
- Go version is 1.24 (per go.mod) — CI workflow (.github/workflows/ci.yml) still references 1.22
- AGENTS.md references Fly.io — will be updated as part of Hetzner migration
- CORS middleware may still be needed for external OpenFeature SDK clients, but SPA no longer needs it
- Repository interfaces live in different files per package (e.g., flag: model.go, project: repository.go)
