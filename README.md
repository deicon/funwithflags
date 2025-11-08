# FunWithFlags

FunWithFlags is a Go-based feature flag evaluation service that is being built to comply with the [OpenFeature](https://openfeature.dev/) specification. The service exposes a lightweight HTTP API that OpenFeature clients can call today, with roadmap work planned for PostgreSQL-backed persistence and an administrative UI.

## Project Status
- `POST /api/v1/flags/{key}/evaluate` returns typed variation data along with OpenFeature-compatible reasons.
- `/healthz` and `/readyz` endpoints provide basic liveness and readiness probes.
- Flags are stored in an in-memory repository while the PostgreSQL implementation is designed.
- End-to-end coverage includes an OpenFeature Go SDK test that drives the HTTP endpoint.

## Prerequisites
- Go 1.24 or newer (see `go.mod` for the authoritative version).

## Run the Server
```bash
make run
# or
go run ./cmd/server
```
The server listens on `:8080` by default.

## Evaluating a Flag
Send a JSON payload containing the evaluation context:
```bash
curl -X POST http://localhost:8080/api/v1/flags/checkout-flow/evaluate \
  -H 'Content-Type: application/json' \
  -d '{"context": {"country": "DE", "userId": "1234"}}'
```
Sample response (assuming the `checkout-flow` flag exists and is enabled):
```json
{
  "flagKey": "checkout-flow",
  "variationKey": "enabled",
  "variationType": "boolean",
  "value": true,
  "reason": "TARGET_MATCH"
}
```
Admin endpoints are not available yet, so flags must currently be seeded programmatically (e.g., within tests or temporary setup code).

## Development Workflow
- `make fmt` — format the code.
- `make lint` — run `go vet`.
- `make test` — execute the unit and integration tests, including the OpenFeature HTTP client exercise.
- `make build` — build the server binary into `bin/funwithflags`.

## Repository Layout
- `cmd/server` — application entry point and process wiring.
- `internal/app` — lifecycle management for the HTTP server.
- `internal/httpserver` — HTTP router, handlers, and response helpers.
- `internal/flag` — domain model, in-memory repository, evaluation engine, and tests.
- `docs/feature-flag-service-plan.md` — iterative development plan and roadmap.

## Next Steps
- Add PostgreSQL persistence with migrations and data access abstractions.
- Expose flag management APIs that the upcoming UI can consume.
- Expand test coverage for error paths and additional OpenFeature scenarios.

## Fly.io Deployment
- Backend: configure the API using `fly.toml`, then create/attach a managed database (`flyctl postgres create/attach`) so the app receives `DATABASE_URL`. Secrets such as `JWT_SECRET` should be set through `flyctl secrets set` before `flyctl deploy -c fly.toml`.
- Frontend: from `frontend/`, deploy with its own `fly.toml` and Dockerfile. Update `build.args.VITE_API_BASE` to the backend’s public URL before running `flyctl deploy -c fly.toml` so the static build knows where to proxy API calls.
- Full instructions, including recommended app names and scaling tips, live in `docs/fly-deployment.md`.
