# Repository Guidelines

## Project Structure & Module Organization
FunWithFlags couples a Go API with a lightweight Vite/React UI. Backend code starts in `cmd/server`, fans into `internal/app` for lifecycle wiring, `internal/httpserver` for routing/handlers, and `internal/flag` for the domain model plus evaluators. Docs live in `docs/`, SQL migrations in `migrations/`, deployment manifests (`Dockerfile`, `fly.toml`, `docker-compose.yml`) stay at the root, and all browser assets are isolated inside `frontend/` so Go modules stay node-free.

## Build, Test, and Development Commands
Use `make run` (or `go run ./cmd/server`) for a local API on `:8080`. `make fmt`, `make lint`, `make test`, and `make build` wrap `go fmt`, `go vet`, `go test ./...`, and `go build -o bin/funwithflags ./cmd/server`. Launch the full stack with `docker-compose up app`, which provisions PostgreSQL plus the server using the env values baked into that file. Frontend contributors run `npm install` once in `frontend/`, then `npm run dev` or `npm run build` as needed.

## Coding Style & Naming Conventions
Go files must remain `gofmt`-clean (tabs, 100-column soft wrap) with concise packages (`flag`, `httpserver`) and CamelCase exports. Keep request/response DTOs annotated with explicit `json:""` tags using snake_case keys to match the HTTP contract. Frontend code follows the default Vite + TypeScript style: functional React components, PascalCase component names, and colocated CSS modules.

## Testing Guidelines
Add table-driven `_test.go` files beside every new backend component, mirroring the existing `internal/flag` patterns and covering happy path, default variation, and error reasons. Run `go test ./...` (optionally `-race`) before each push; new evaluation paths should demonstrate OpenFeature-compliant reasons. The frontend is still exploratory, but describe any manual verification or add vitest/screenshot coverage once the UI stabilizes.

## Commit & Pull Request Guidelines
Write short, imperative commits—optionally prefixed like `feat:` or `fix:` as seen in `fix: first draft of UI`—and reference an issue or roadmap item in the body when relevant. Each PR should explain behavior changes, call out config or schema edits (e.g., new files in `migrations/`), and attach evidence such as `curl` output or UI screenshots. CI is light, so run `make fmt lint test` (plus frontend builds when touched) before requesting review.

## Security & Configuration Tips
Prefer environment variables (`STORAGE_TYPE`, `DB_*`, `PORT`, `MIGRATIONS_PATH`) over literals; `docker-compose.yml` supplies sane defaults and mirrors `fly.toml`. Document any new secret or toggle in `README.md` and update both Fly and compose manifests together so local, container, and hosted environments stay aligned.
