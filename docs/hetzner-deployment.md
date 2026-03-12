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
