# FunWithFlags System Refinement Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create CLAUDE.md, C4 architecture diagrams, and Hetzner deployment configs — plus the Go backend changes needed to serve the frontend from disk.

**Architecture:** Single Go binary serves both REST API and React SPA static files. Caddy handles TLS in front. PostgreSQL in a separate Podman container. All managed by systemd on a Hetzner Cloud VPS.

**Tech Stack:** Go 1.24, React/Vite, PostgreSQL 16, Podman, Caddy, systemd, Hetzner Object Storage (S3)

**Spec:** `docs/superpowers/specs/2026-03-12-funwithflags-refinement-design.md`

---

## Chunk 1: CLAUDE.md and C4 Diagrams

### Task 1: Create CLAUDE.md

**Files:**
- Create: `CLAUDE.md`

- [ ] **Step 1: Write CLAUDE.md**

```markdown
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
```

- [ ] **Step 2: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: add CLAUDE.md with AI assistant guidance"
```

---

### Task 2: Create C4 Architecture Diagrams

**Files:**
- Create: `docs/architecture.md`

- [ ] **Step 1: Write docs/architecture.md with both L1 and L2 diagrams**

````markdown
# FunWithFlags Architecture

## System Context (C4 Level 1)

High-level view of FunWithFlags and its interactions with users and external systems.

```mermaid
C4Context
    title System Context — FunWithFlags

    Person(admin, "Admin User", "Manages projects, stages, flags, and users")
    System_Ext(client, "Client Application", "Evaluates feature flags via OpenFeature SDK")

    System(fwf, "FunWithFlags", "Feature flag management and evaluation service")

    System_Ext(s3, "Hetzner Object Storage", "S3-compatible backup storage")

    Rel(admin, fwf, "Manages flags", "HTTPS")
    Rel(client, fwf, "Evaluates flags", "REST/HTTPS")
    Rel(fwf, s3, "Backs up database", "S3 API")
```

## Container Diagram (C4 Level 2)

Containers running on a single Hetzner Cloud VPS, managed by Podman + systemd.

```mermaid
C4Container
    title Container Diagram — FunWithFlags (Hetzner VPS)

    Person(admin, "Admin User", "Manages flags via web UI")
    System_Ext(client, "Client Application", "Evaluates flags via OpenFeature")

    Container_Boundary(vps, "Hetzner Cloud VPS") {
        Container(caddy, "Caddy", "Caddy 2.x", "TLS termination, reverse proxy")
        Container(api, "API + Frontend", "Go 1.24", "REST API, flag evaluator, serves React SPA from disk")
        ContainerDb(db, "PostgreSQL", "PostgreSQL 16", "Flags, projects, stages, users, audit logs")
        Container(backup, "Backup Cron", "systemd timer + pg_dump", "Scheduled database dumps")
    }

    System_Ext(s3, "Hetzner Object Storage", "S3-compatible backup storage")

    Rel(admin, caddy, "HTTPS")
    Rel(client, caddy, "REST/HTTPS")
    Rel(caddy, api, "Proxy", "HTTP :8080")
    Rel(api, db, "Reads/writes", "pgx, TCP :5432")
    Rel(backup, db, "pg_dump", "TCP :5432")
    Rel(backup, s3, "Uploads backups", "S3 API")
```

## Container Summary

| Container | Technology | Responsibility |
|-----------|-----------|----------------|
| Caddy | Caddy 2.x (Podman) | TLS termination, reverse proxy to Go backend |
| API + Frontend | Go 1.24 (Podman) | REST API, flag evaluation engine, serves React SPA from disk |
| PostgreSQL | PostgreSQL 16 (Podman) | Persistent storage for flags, projects, stages, users, audit logs |
| Backup Cron | systemd timer + pg_dump | Scheduled database dumps to Hetzner Object Storage |

All Podman containers are managed by systemd with auto-update labels.
Containers communicate over a shared Podman network (`funwithflags`).
````

- [ ] **Step 2: Commit**

```bash
git add docs/architecture.md
git commit -m "docs: add C4 architecture diagrams (context + container)"
```

---

## Chunk 2: Go Backend Static File Serving

### Task 3: Add SPA file server handler with test

**Files:**
- Create: `internal/httpserver/spa.go`
- Create: `internal/httpserver/spa_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/httpserver/spa_test.go`:

```go
package httpserver

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestSPAHandler(t *testing.T) {
	// Set up a temporary directory with test files
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>spa</html>"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte("console.log('app')"), 0644); err != nil {
		t.Fatal(err)
	}

	handler := NewSPAHandler(dir)

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "serves index.html at root",
			path:       "/",
			wantStatus: http.StatusOK,
			wantBody:   "<html>spa</html>",
		},
		{
			name:       "serves static file",
			path:       "/assets/app.js",
			wantStatus: http.StatusOK,
			wantBody:   "console.log('app')",
		},
		{
			name:       "falls back to index.html for unknown paths",
			path:       "/some/client/route",
			wantStatus: http.StatusOK,
			wantBody:   "<html>spa</html>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if body := rec.Body.String(); body != tt.wantBody {
				t.Errorf("body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/httpserver/ -run TestSPAHandler -v`
Expected: FAIL — `NewSPAHandler` not defined

- [ ] **Step 3: Write the SPA handler implementation**

Create `internal/httpserver/spa.go`:

```go
package httpserver

import (
	"net/http"
	"os"
	"path/filepath"
)

// NewSPAHandler returns an http.Handler that serves static files from dir.
// If the requested file does not exist, it falls back to dir/index.html
// to support client-side routing.
func NewSPAHandler(dir string) http.Handler {
	return &spaHandler{dir: dir}
}

type spaHandler struct {
	dir string
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Clean the path and resolve to filesystem
	path := filepath.Join(h.dir, filepath.Clean(r.URL.Path))

	// Check if the file exists
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		// Fall back to index.html for SPA client-side routing
		http.ServeFile(w, r, filepath.Join(h.dir, "index.html"))
		return
	}

	http.ServeFile(w, r, path)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/httpserver/ -run TestSPAHandler -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/httpserver/spa.go internal/httpserver/spa_test.go
git commit -m "feat(httpserver): add SPA file server handler"
```

---

### Task 4: Wire SPA handler into router and app config

**Files:**
- Modify: `internal/httpserver/router.go:13-19` (Config struct)
- Modify: `internal/httpserver/router.go:107-112` (after mux setup, before CORS)
- Modify: `internal/app/app.go:27-31` (read FRONTEND_DIR env)
- Modify: `internal/app/app.go:157-163` (pass FrontendDir to router config)

- [ ] **Step 1: Add FrontendDir to httpserver.Config**

In `internal/httpserver/router.go`, add `FrontendDir` to the Config struct:

```go
type Config struct {
	FlagService    *flag.Service
	ProjectService *project.Service
	AuthManager    *auth.Manager
	AuthService    *auth.Service
	AllowedOrigins []string
	FrontendDir    string
}
```

- [ ] **Step 2: Register SPA handler as catch-all in NewRouter**

In `internal/httpserver/router.go`, add the SPA handler after all API routes are registered (before the CORS handler wrapping at line 107):

```go
	// Serve frontend SPA if configured
	if cfg.FrontendDir != "" {
		mux.Handle("GET /", NewSPAHandler(cfg.FrontendDir))
	}

	handler := http.Handler(mux)
```

- [ ] **Step 3: Read FRONTEND_DIR env in app.go**

In `internal/app/app.go`, inside the `New()` function, after line 33 (authSecret), add:

```go
	frontendDir := getEnv("FRONTEND_DIR", "")
	if frontendDir != "" {
		log.Printf("Serving frontend from %s", frontendDir)
	}
```

- [ ] **Step 4: Pass FrontendDir to router config**

In `internal/app/app.go`, add `FrontendDir` to the `httpserver.Config` struct literal (around line 157):

```go
	router, err := httpserver.NewRouter(httpserver.Config{
		FlagService:    flagService,
		ProjectService: projectService,
		AuthManager:    authManager,
		AuthService:    authService,
		AllowedOrigins: allowedOrigins,
		FrontendDir:    frontendDir,
	})
```

- [ ] **Step 5: Run all tests**

Run: `go test ./... -v`
Expected: All existing tests pass (SPA handler is opt-in, empty string disables it)

- [ ] **Step 6: Commit**

```bash
git add internal/httpserver/router.go internal/app/app.go
git commit -m "feat: wire SPA file server into router via FRONTEND_DIR env"
```

---

## Chunk 3: Multi-Stage Dockerfile

### Task 5: Replace Dockerfile with multi-stage build

**Files:**
- Modify: `Dockerfile` (full rewrite)

- [ ] **Step 1: Rewrite Dockerfile**

```dockerfile
# Stage 1: Build frontend
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.24 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/funwithflags ./cmd/server

# Stage 3: Final image
FROM gcr.io/distroless/base-debian12
COPY --from=builder /bin/funwithflags /bin/funwithflags
COPY --from=builder /app/migrations /migrations
COPY --from=frontend /app/frontend/dist /app/frontend/dist
EXPOSE 8080
ENV FRONTEND_DIR=/app/frontend/dist
ENTRYPOINT ["/bin/funwithflags"]
```

- [ ] **Step 2: Build the image to verify it works**

Run: `docker build -t funwithflags:test .`
Expected: Build completes successfully

- [ ] **Step 3: Commit**

```bash
git add Dockerfile
git commit -m "feat(docker): multi-stage build with frontend bundled"
```

---

### Task 6: Update docker-compose.yml for local dev

**Files:**
- Modify: `docker-compose.yml`

- [ ] **Step 1: Remove frontend service, add FRONTEND_DIR to app**

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    container_name: funwithflags-postgres
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: funwithflags
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 5s
      retries: 5

  app:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: funwithflags-app
    environment:
      STORAGE_TYPE: postgres
      DB_HOST: postgres
      DB_PORT: 5432
      DB_USER: postgres
      DB_PASSWORD: postgres
      DB_NAME: funwithflags
      MIGRATIONS_PATH: /migrations
      PORT: 8080
      FRONTEND_DIR: /app/frontend/dist
      CORS_ALLOWED_ORIGINS: "http://localhost:8080,http://localhost:5173,http://localhost:3000"
    ports:
      - "8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
    restart: unless-stopped

volumes:
  postgres_data:
```

- [ ] **Step 2: Test with docker-compose**

Run: `docker-compose up --build -d && docker-compose ps`
Expected: Both `postgres` and `app` services running, frontend accessible at http://localhost:8080

- [ ] **Step 3: Tear down**

Run: `docker-compose down`

- [ ] **Step 4: Commit**

```bash
git add docker-compose.yml
git commit -m "feat(docker): consolidate frontend into app service"
```

---

## Chunk 4: Hetzner Deployment Configs

### Task 7: Create deploy directory with Caddyfile

**Files:**
- Create: `deploy/Caddyfile`

- [ ] **Step 1: Write Caddyfile**

```
# Replace funwithflags.example.com with your actual domain
funwithflags.example.com {
    reverse_proxy funwithflags:8080
}
```

Note: `funwithflags` is the container name on the shared Podman network.

- [ ] **Step 2: Commit**

```bash
git add deploy/Caddyfile
git commit -m "deploy: add Caddyfile for Caddy reverse proxy"
```

---

### Task 8: Create systemd unit for PostgreSQL

**Files:**
- Create: `deploy/postgres.service`

- [ ] **Step 1: Write postgres.service**

```ini
[Unit]
Description=FunWithFlags PostgreSQL
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
Restart=always
RestartSec=10
EnvironmentFile=%h/.config/funwithflags/postgres.env

ExecStartPre=-/usr/bin/podman rm -f funwithflags-postgres
ExecStart=/usr/bin/podman run \
    --name funwithflags-postgres \
    --network funwithflags \
    --volume funwithflags-pgdata:/var/lib/postgresql/data:Z \
    --env-file %h/.config/funwithflags/postgres.env \
    --health-cmd "pg_isready -U postgres" \
    --health-interval 10s \
    --health-timeout 5s \
    --health-retries 5 \
    docker.io/library/postgres:16-alpine

ExecStop=/usr/bin/podman stop -t 30 funwithflags-postgres

[Install]
WantedBy=default.target
```

Create `deploy/postgres.env.example`:
```ini
POSTGRES_USER=postgres
POSTGRES_PASSWORD=changeme
POSTGRES_DB=funwithflags
```

Copy to `~/.config/funwithflags/postgres.env` on the server and set a real password.

- [ ] **Step 2: Commit**

```bash
git add deploy/postgres.service deploy/postgres.env.example
git commit -m "deploy: add systemd unit for PostgreSQL container"
```

---

### Task 9: Create systemd unit for FunWithFlags app

**Files:**
- Create: `deploy/funwithflags.service`
- Create: `deploy/funwithflags.env.example`

- [ ] **Step 1: Write funwithflags.env.example**

```ini
# Database
DATABASE_URL=postgres://postgres:changeme@funwithflags-postgres:5432/funwithflags?sslmode=disable

# Auth
JWT_SECRET=changeme-generate-a-real-secret
AUTH_ADMIN_USERNAME=admin
AUTH_ADMIN_PASSWORD=changeme
AUTH_USER_USERNAME=user
AUTH_USER_PASSWORD=changeme

# Server
PORT=8080
STORAGE_TYPE=postgres
MIGRATIONS_PATH=/migrations
FRONTEND_DIR=/app/frontend/dist
CORS_ALLOWED_ORIGINS=https://funwithflags.example.com
```

- [ ] **Step 2: Write funwithflags.service**

```ini
[Unit]
Description=FunWithFlags API + Frontend
After=funwithflags-postgres.service
Requires=funwithflags-postgres.service

[Service]
Type=simple
Restart=always
RestartSec=10

ExecStartPre=-/usr/bin/podman rm -f funwithflags
ExecStart=/usr/bin/podman run \
    --name funwithflags \
    --network funwithflags \
    --label io.containers.autoupdate=registry \
    --env-file %h/.config/funwithflags/funwithflags.env \
    ghcr.io/deicon/funwithflags:latest

ExecStop=/usr/bin/podman stop -t 15 funwithflags

[Install]
WantedBy=default.target
```

Note: No `--publish` needed — Caddy reaches the app via the shared Podman network by container name. Replace `ghcr.io/deicon/funwithflags:latest` with your actual registry path.

- [ ] **Step 3: Commit**

```bash
git add deploy/funwithflags.service deploy/funwithflags.env.example
git commit -m "deploy: add systemd unit and env template for app container"
```

---

### Task 10: Create systemd unit for Caddy

**Files:**
- Create: `deploy/caddy.service`

- [ ] **Step 1: Write caddy.service**

```ini
[Unit]
Description=FunWithFlags Caddy Reverse Proxy
After=funwithflags.service
Requires=funwithflags.service

[Service]
Type=simple
Restart=always
RestartSec=10

ExecStartPre=-/usr/bin/podman rm -f funwithflags-caddy
ExecStart=/usr/bin/podman run \
    --name funwithflags-caddy \
    --network funwithflags \
    --publish 80:80 \
    --publish 443:443 \
    --volume %h/.config/funwithflags/Caddyfile:/etc/caddy/Caddyfile:Z \
    --volume funwithflags-caddy-data:/data:Z \
    --volume funwithflags-caddy-config:/config:Z \
    docker.io/library/caddy:2-alpine

ExecStop=/usr/bin/podman stop -t 15 funwithflags-caddy

[Install]
WantedBy=default.target
```

- [ ] **Step 2: Commit**

```bash
git add deploy/caddy.service
git commit -m "deploy: add systemd unit for Caddy reverse proxy"
```

---

### Task 11: Create backup script and systemd timer

**Files:**
- Create: `deploy/backup.sh`
- Create: `deploy/backup.service`
- Create: `deploy/backup.timer`

- [ ] **Step 1: Write backup.sh**

```bash
#!/usr/bin/env bash
set -euo pipefail

# Required environment variables:
#   S3_ENDPOINT    - Hetzner S3 endpoint (e.g., https://fsn1.your-objectstorage.com)
#   S3_BUCKET      - Bucket name
#   S3_ACCESS_KEY  - Access key
#   S3_SECRET_KEY  - Secret key
#   BACKUP_RETENTION_DAYS - Number of days to keep backups (default: 30)

RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-30}"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
BACKUP_FILE="funwithflags-${TIMESTAMP}.sql.gz"
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

echo "Starting backup: ${BACKUP_FILE}"

# Dump database
podman exec funwithflags-postgres pg_dump -U postgres funwithflags | gzip > "${TMPDIR}/${BACKUP_FILE}"

# Upload to S3
export AWS_ACCESS_KEY_ID="${S3_ACCESS_KEY}"
export AWS_SECRET_ACCESS_KEY="${S3_SECRET_KEY}"

aws s3 cp "${TMPDIR}/${BACKUP_FILE}" "s3://${S3_BUCKET}/backups/${BACKUP_FILE}" \
    --endpoint-url "${S3_ENDPOINT}"

echo "Uploaded ${BACKUP_FILE} to s3://${S3_BUCKET}/backups/"

# Clean up old backups
CUTOFF_DATE=$(date -d "-${RETENTION_DAYS} days" +%Y%m%d 2>/dev/null || date -v-${RETENTION_DAYS}d +%Y%m%d)
aws s3 ls "s3://${S3_BUCKET}/backups/" --endpoint-url "${S3_ENDPOINT}" | while read -r line; do
    FILE=$(echo "$line" | awk '{print $4}')
    FILE_DATE=$(echo "$FILE" | grep -oE '[0-9]{8}' | head -1)
    if [[ -n "$FILE_DATE" && "$FILE_DATE" < "$CUTOFF_DATE" ]]; then
        echo "Deleting old backup: ${FILE}"
        aws s3 rm "s3://${S3_BUCKET}/backups/${FILE}" --endpoint-url "${S3_ENDPOINT}"
    fi
done

echo "Backup complete"
```

- [ ] **Step 2: Make backup.sh executable**

Run: `chmod +x deploy/backup.sh`

- [ ] **Step 3: Write backup.service**

```ini
[Unit]
Description=FunWithFlags Database Backup

[Service]
Type=oneshot
EnvironmentFile=%h/.config/funwithflags/backup.env
ExecStart=%h/.config/funwithflags/backup.sh
```

- [ ] **Step 4: Write backup.timer**

```ini
[Unit]
Description=FunWithFlags Daily Database Backup

[Timer]
OnCalendar=*-*-* 03:00:00
Persistent=true
RandomizedDelaySec=900

[Install]
WantedBy=timers.target
```

- [ ] **Step 5: Commit**

```bash
git add deploy/backup.sh deploy/backup.service deploy/backup.timer
git commit -m "deploy: add database backup script and systemd timer"
```

---

### Task 12: Write Hetzner deployment guide

**Files:**
- Create: `docs/hetzner-deployment.md`

- [ ] **Step 1: Write docs/hetzner-deployment.md**

````markdown
# Hetzner Deployment Guide

Deploys FunWithFlags on a Hetzner Cloud VPS using Podman, Caddy, and systemd.

## Prerequisites

- Hetzner Cloud VPS (CX22 or similar)
- Domain name pointed to the VPS IP
- Hetzner Object Storage bucket for backups
- Container image pushed to a registry (e.g., ghcr.io/deicon/funwithflags)

## 1. Install Podman

```bash
sudo apt update && sudo apt install -y podman
```

Enable lingering so user services run without a login session:

```bash
sudo loginctl enable-linger $USER
```

## 2. Create Podman Network

```bash
podman network create funwithflags
```

## 3. Create Config Directory

```bash
mkdir -p ~/.config/funwithflags
```

Copy and customize the config files:

```bash
# Copy from the deploy/ directory in the repo
cp deploy/Caddyfile ~/.config/funwithflags/Caddyfile
cp deploy/funwithflags.env.example ~/.config/funwithflags/funwithflags.env
cp deploy/backup.sh ~/.config/funwithflags/backup.sh
chmod +x ~/.config/funwithflags/backup.sh
```

Edit `~/.config/funwithflags/funwithflags.env` with your actual values:
- Set a strong `JWT_SECRET`
- Set real passwords for `AUTH_ADMIN_PASSWORD` and `AUTH_USER_PASSWORD`
- Set `DATABASE_URL` with your PostgreSQL password
- Set `CORS_ALLOWED_ORIGINS` to your domain

Edit `~/.config/funwithflags/Caddyfile`:
- Replace `funwithflags.example.com` with your domain

Create `~/.config/funwithflags/backup.env`:

```ini
S3_ENDPOINT=https://fsn1.your-objectstorage.com
S3_BUCKET=your-bucket-name
S3_ACCESS_KEY=your-access-key
S3_SECRET_KEY=your-secret-key
BACKUP_RETENTION_DAYS=30
```

## 4. Install systemd Units

```bash
mkdir -p ~/.config/systemd/user

cp deploy/postgres.service ~/.config/systemd/user/funwithflags-postgres.service
cp deploy/funwithflags.service ~/.config/systemd/user/funwithflags.service
cp deploy/caddy.service ~/.config/systemd/user/funwithflags-caddy.service
cp deploy/backup.service ~/.config/systemd/user/funwithflags-backup.service
cp deploy/backup.timer ~/.config/systemd/user/funwithflags-backup.timer

systemctl --user daemon-reload
```

## 5. Start Services

Start in dependency order:

```bash
systemctl --user enable --now funwithflags-postgres.service
systemctl --user enable --now funwithflags.service
systemctl --user enable --now funwithflags-caddy.service
```

## 6. Verify

```bash
# Check services are running
systemctl --user status funwithflags-postgres
systemctl --user status funwithflags
systemctl --user status funwithflags-caddy

# Test frontend is served (no auth required for static files)
curl -s -o /dev/null -w "%{http_code}" https://funwithflags.example.com/

# Test health endpoint (requires auth token — get one via login first)
# Note: /healthz requires a valid Bearer token
TOKEN=$(curl -s -X POST https://funwithflags.example.com/api/v1/auth/login \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"your-password"}' | jq -r '.access_token')
curl -s -H "Authorization: Bearer $TOKEN" https://funwithflags.example.com/healthz
```

## 7. Enable Auto-Update

```bash
systemctl --user enable --now podman-auto-update.timer
```

This checks for new images with the `io.containers.autoupdate=registry` label and restarts containers when updates are available.

## 8. Enable Backups

```bash
# Install AWS CLI for S3 uploads
sudo apt install -y awscli

systemctl --user enable --now funwithflags-backup.timer

# Verify timer is scheduled
systemctl --user list-timers funwithflags-backup.timer

# Run a manual backup to test
systemctl --user start funwithflags-backup.service
journalctl --user -u funwithflags-backup.service --no-pager -n 20
```

## Operations

### View logs

```bash
journalctl --user -u funwithflags -f
journalctl --user -u funwithflags-postgres -f
journalctl --user -u funwithflags-caddy -f
```

### Restart a service

```bash
systemctl --user restart funwithflags
```

### Manual image update

```bash
podman pull ghcr.io/deicon/funwithflags:latest
systemctl --user restart funwithflags
```

### Database shell

```bash
podman exec -it funwithflags-postgres psql -U postgres funwithflags
```
````

- [ ] **Step 2: Commit**

```bash
git add docs/hetzner-deployment.md
git commit -m "docs: add Hetzner deployment guide"
```

---

## Chunk 5: Final Cleanup

### Task 13: Update .env.example with FRONTEND_DIR

**Files:**
- Modify: `.env.example`

- [ ] **Step 1: Add FRONTEND_DIR to .env.example**

Add this line to the existing `.env.example`:

```
# Frontend (optional - set to serve SPA from Go backend)
# FRONTEND_DIR=./frontend/dist
```

- [ ] **Step 2: Commit**

```bash
git add .env.example
git commit -m "docs: add FRONTEND_DIR to .env.example"
```

---

### Task 14: Final verification

- [ ] **Step 1: Run all Go tests**

Run: `go test ./... -v`
Expected: All tests pass

- [ ] **Step 2: Run gofmt check**

Run: `gofmt -l ./internal/`
Expected: No output (all files formatted)

- [ ] **Step 3: Run go vet**

Run: `go vet ./...`
Expected: No issues

- [ ] **Step 4: Verify Docker build**

Run: `docker build -t funwithflags:verify .`
Expected: Build completes successfully
