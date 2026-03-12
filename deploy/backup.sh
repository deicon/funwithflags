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
