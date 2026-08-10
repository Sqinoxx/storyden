#!/usr/bin/env bash
# Sichert Datenbank und Uploads auf dem Pi nach ./backups.
#
# Manuell:  bash backup.sh
# Taeglich per cron (crontab -e), 03:30 Uhr:
#   30 3 * * * cd /home/pi/storyden && bash backup.sh >> backup.log 2>&1
#
# Die Sicherungen liegen damit auf derselben SD-Karte wie die Daten. Fuer
# echten Schutz zusaetzlich woanders ablegen, z.B. per rsync auf den PC oder
# eine USB-Platte.
set -euo pipefail

cd "$(dirname "$0")"

BACKUP_DIR="${BACKUP_DIR:-./backups}"
KEEP_DAYS="${KEEP_DAYS:-14}"
STAMP="$(date +%F-%H%M)"

mkdir -p "$BACKUP_DIR"

POSTGRES_USER="$(grep -E '^POSTGRES_USER=' .env | tail -n1 | cut -d= -f2-)"
POSTGRES_DB="$(grep -E '^POSTGRES_DB=' .env | tail -n1 | cut -d= -f2-)"

docker compose exec -T postgres \
  pg_dump -U "${POSTGRES_USER:-storyden}" "${POSTGRES_DB:-storyden}" \
  | gzip > "$BACKUP_DIR/db-$STAMP.sql.gz"

tar czf "$BACKUP_DIR/assets-$STAMP.tar.gz" ./data

find "$BACKUP_DIR" -name '*.gz' -mtime "+$KEEP_DAYS" -delete

echo "Backup fertig: $BACKUP_DIR/db-$STAMP.sql.gz, $BACKUP_DIR/assets-$STAMP.tar.gz"
