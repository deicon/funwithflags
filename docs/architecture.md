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
