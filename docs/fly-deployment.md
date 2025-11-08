# Fly.io Deployment Guide

This guide walks through provisioning Fly.io resources for both the Go API and the React frontend while relying on a managed Fly Postgres cluster.

## Prerequisites
- `flyctl` logged in to the correct organization.
- Docker configured locally (Fly builds use it when `fly launch` is run from this repo).
- A unique Fly app name for each service (defaults below: `funwithflags-api`, `funwithflags-frontend`, `funwithflags-db`).

## Backend API + Postgres
1. **Create the apps**
   ```bash
   flyctl apps create funwithflags-api
   flyctl postgres create --name funwithflags-db --region iad --initial-cluster-size 1 --vm-size shared-cpu-1x --volume-size 1
   ```
2. **Attach Postgres** – this provisions `DATABASE_URL` for the app and grants network access.
   ```bash
   flyctl postgres attach funwithflags-db --app funwithflags-api --database-name funwithflags
   ```
3. **Configure secrets** – override anything sensitive; at minimum set JWT/auth secrets.
   ```bash
   flyctl secrets set \
     JWT_SECRET="change-me" \
     AUTH_ADMIN_USERNAME="admin" \
     AUTH_ADMIN_PASSWORD="s3cret" \
     AUTH_USER_USERNAME="user" \
     AUTH_USER_PASSWORD="userpass" \
     --app funwithflags-api
   ```
4. **Deploy** – from the repo root:
   ```bash
   flyctl deploy --config fly.toml
   ```
   The container runs migrations automatically using `/migrations` and the attached `DATABASE_URL`. Health checks poll `/healthz`.

## Frontend
1. **Switch directories and create the app**
   ```bash
   cd frontend
   flyctl apps create funwithflags-frontend
   ```
2. **Point the build at the API** – update `build.args.VITE_API_BASE` inside `frontend/fly.toml` to the public HTTPS URL of the backend (`https://funwithflags-api.fly.dev` if you accept the defaults).
3. **Deploy**
   ```bash
   flyctl deploy --config fly.toml
   ```
   The build stage compiles the Vite app, and Nginx serves it on port 8080 with a `/healthz` endpoint for Fly checks.

## Operational Notes
- Scale machines or regions via `flyctl scale` or `flyctl regions add <code>` as needed.
- Use `flyctl ssh console --app funwithflags-api` to run troubleshooting commands inside the running VM.
- Whenever database schema changes, redeploy the backend; migrations execute on boot before the HTTP server starts.
