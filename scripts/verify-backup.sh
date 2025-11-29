#!/bin/bash
# Verify the latest backup can be restored
# Usage: ./scripts/verify-backup.sh
#
# Runs daily after backup to verify integrity.
# On failure, writes error details to R2 bucket.
#
# Required environment variables (or in /opt/yearofbingo/.env):
#   BACKUP_ENCRYPTION_KEY - GPG passphrase used when creating backup

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${ENV_FILE:-/opt/yearofbingo/.env}"

# Load environment if available
if [[ -f "$ENV_FILE" ]]; then
    set -a
    source "$ENV_FILE"
    set +a
fi

# Configuration
BACKUP_DIR="/tmp/yearofbingo-backups"
R2_BUCKET="${R2_BUCKET:-yearofbingo-backups}"
TEST_CONTAINER="yearofbingo-backup-verify"
TEST_DB_PASSWORD="verify_password_$(date +%s)"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
ERROR_FILE="BACKUP_VERIFICATION_FAILED_${TIMESTAMP}.txt"

# Validation
if [[ -z "${BACKUP_ENCRYPTION_KEY:-}" ]]; then
    echo "ERROR: BACKUP_ENCRYPTION_KEY is required"
    exit 1
fi

if ! command -v rclone &> /dev/null; then
    echo "ERROR: rclone is not installed"
    exit 1
fi

if ! command -v podman &> /dev/null; then
    echo "ERROR: podman is not installed"
    exit 1
fi

# Write error to R2
write_error() {
    local error_message="$1"
    local error_file="/tmp/${ERROR_FILE}"

    cat > "$error_file" << EOF
BACKUP VERIFICATION FAILED
==========================
Timestamp: $(date '+%Y-%m-%d %H:%M:%S %Z')
Hostname: $(hostname)
Backup file: ${BACKUP_FILE:-unknown}

Error:
${error_message}

Action required: Check backup system immediately.
EOF

    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Writing error report to R2..."
    rclone copy "$error_file" "r2:${R2_BUCKET}/" 2>/dev/null || true
    rm -f "$error_file"

    echo "ERROR: $error_message"
    exit 1
}

# Cleanup function
cleanup() {
    podman rm -f "$TEST_CONTAINER" 2>/dev/null || true
    rm -f "${BACKUP_DIR}/${BACKUP_FILE}" 2>/dev/null || true
    rm -f "${BACKUP_DIR}/verify_restore.sql" 2>/dev/null || true
}
trap cleanup EXIT

# Get latest backup
BACKUP_FILE=$(rclone ls "r2:${R2_BUCKET}/" 2>/dev/null | grep -E '\.sql\.gz\.gpg$' | sort -k2 | tail -1 | awk '{print $2}')

if [[ -z "$BACKUP_FILE" ]]; then
    write_error "No backup files found in r2:${R2_BUCKET}/"
fi

echo "[$(date '+%Y-%m-%d %H:%M:%S')] Verifying backup: ${BACKUP_FILE}"

mkdir -p "$BACKUP_DIR"

# Step 1: Download
echo "[$(date '+%Y-%m-%d %H:%M:%S')] Downloading backup..."
if ! rclone copy "r2:${R2_BUCKET}/${BACKUP_FILE}" "${BACKUP_DIR}/" 2>&1; then
    write_error "Failed to download backup from R2"
fi

# Step 2: Decrypt
echo "[$(date '+%Y-%m-%d %H:%M:%S')] Decrypting backup..."
if ! gpg --decrypt --batch --passphrase "$BACKUP_ENCRYPTION_KEY" "${BACKUP_DIR}/${BACKUP_FILE}" 2>/dev/null | gunzip > "${BACKUP_DIR}/verify_restore.sql" 2>&1; then
    write_error "Failed to decrypt/decompress backup. Encryption key may be wrong or file corrupted."
fi

SQL_SIZE=$(stat -f%z "${BACKUP_DIR}/verify_restore.sql" 2>/dev/null || stat -c%s "${BACKUP_DIR}/verify_restore.sql" 2>/dev/null)
if [[ "$SQL_SIZE" -lt 1000 ]]; then
    write_error "Decrypted SQL file too small (${SQL_SIZE} bytes). Backup may be corrupted."
fi

# Step 3: Start test container
echo "[$(date '+%Y-%m-%d %H:%M:%S')] Starting test PostgreSQL container..."
if ! podman run -d \
    --name "$TEST_CONTAINER" \
    -e POSTGRES_USER=bingo \
    -e POSTGRES_PASSWORD="$TEST_DB_PASSWORD" \
    -e POSTGRES_DB=yearofbingo \
    docker.io/library/postgres:16-alpine 2>&1; then
    write_error "Failed to start test PostgreSQL container"
fi

# Wait for PostgreSQL
echo "[$(date '+%Y-%m-%d %H:%M:%S')] Waiting for PostgreSQL..."
for i in {1..30}; do
    if podman exec "$TEST_CONTAINER" pg_isready -U bingo -d yearofbingo &>/dev/null; then
        break
    fi
    if [[ $i -eq 30 ]]; then
        write_error "Test PostgreSQL container failed to become ready"
    fi
    sleep 1
done

# Step 4: Restore
echo "[$(date '+%Y-%m-%d %H:%M:%S')] Restoring to test container..."
if ! podman exec -i -e PGPASSWORD="$TEST_DB_PASSWORD" "$TEST_CONTAINER" \
    psql -U bingo -d yearofbingo < "${BACKUP_DIR}/verify_restore.sql" &>/dev/null; then
    write_error "Failed to restore backup to test database"
fi

# Step 5: Validate
echo "[$(date '+%Y-%m-%d %H:%M:%S')] Validating data..."
USER_COUNT=$(podman exec -e PGPASSWORD="$TEST_DB_PASSWORD" "$TEST_CONTAINER" \
    psql -U bingo -d yearofbingo -t -c "SELECT COUNT(*) FROM users;" 2>/dev/null | tr -d ' ')

if [[ -z "$USER_COUNT" ]] || [[ "$USER_COUNT" -lt 0 ]]; then
    write_error "Failed to query users table - restore may have failed"
fi

CARD_COUNT=$(podman exec -e PGPASSWORD="$TEST_DB_PASSWORD" "$TEST_CONTAINER" \
    psql -U bingo -d yearofbingo -t -c "SELECT COUNT(*) FROM bingo_cards;" 2>/dev/null | tr -d ' ')

echo ""
echo "=========================================="
echo "BACKUP VERIFICATION PASSED"
echo "=========================================="
echo "Backup: ${BACKUP_FILE}"
echo "Users: ${USER_COUNT}"
echo "Cards: ${CARD_COUNT}"
echo "Verified: $(date '+%Y-%m-%d %H:%M:%S %Z')"
echo "=========================================="
