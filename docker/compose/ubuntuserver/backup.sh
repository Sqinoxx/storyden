#!/usr/bin/env bash
# Sichert Datenbank und hochgeladene Dateien.
#
# Manuell:  bash backup.sh
# Taeglich per cron (crontab -e), 03:30 Uhr:
#   30 3 * * * cd /home/unidentist/storyden && bash backup.sh >> backup.log 2>&1
set -euo pipefail

cd "$(dirname "$0")"

BACKUP_DIR="${BACKUP_DIR:-./backups}"
KEEP_DAYS="${KEEP_DAYS:-14}"
STAMP="$(date +%F-%H%M)"

mkdir -p "$BACKUP_DIR"

# shellcheck disable=SC1091
POSTGRES_USER="$(grep -E '^POSTGRES_USER=' .env | cut -d= -f2- | tr -d '\r')"
POSTGRES_DB="$(grep -E '^POSTGRES_DB=' .env | cut -d= -f2- | tr -d '\r')"

docker compose exec -T postgres \
  pg_dump -U "${POSTGRES_USER:-storyden}" "${POSTGRES_DB:-storyden}" \
  | gzip > "$BACKUP_DIR/db-$STAMP.sql.gz"

tar czf "$BACKUP_DIR/assets-$STAMP.tar.gz" ./data

find "$BACKUP_DIR" -name '*.gz' -mtime "+$KEEP_DAYS" -delete

echo "Backup fertig: $BACKUP_DIR/db-$STAMP.sql.gz, $BACKUP_DIR/assets-$STAMP.tar.gz"
