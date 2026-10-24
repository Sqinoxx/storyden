#!/usr/bin/env bash
# Kopiert die neuesten Backups auf einen anderen Rechner (z.B. Nextcloud am Pi).
# Dort liegen immer nur die zwei neuesten Datenbank-Dumps und ein Spiegel der
# Uploads, die aeltere Historie bleibt auf dem Server (siehe backup.sh).
#
# Manuell:  bash offsite-sync.sh
# Taeglich nach backup.sh (crontab -e als unidentist):
#   45 3 * * * cd /home/unidentist/storyden && bash offsite-sync.sh >> offsite.log 2>&1
#
# Einrichtung siehe Anleitung.md, Abschnitt "Backups woanders ablegen".
set -euo pipefail

cd "$(dirname "$0")"

OFFSITE_HOST="${OFFSITE_HOST:-pi@192.168.178.30}"
OFFSITE_KEY="${OFFSITE_KEY:-$HOME/.ssh/offsite_backup}"
OFFSITE_KEEP="${OFFSITE_KEEP:-2}"
BACKUP_DIR="${BACKUP_DIR:-./backups}"

SSH="ssh -i $OFFSITE_KEY -o BatchMode=yes -o ConnectTimeout=15"

mapfile -t NEWEST < <(ls -1 "$BACKUP_DIR" | grep -E '^db-.*\.sql\.gz$' | sort -r | head -n "$OFFSITE_KEEP")
if [ "${#NEWEST[@]}" -eq 0 ]; then
  echo "Keine Dumps in $BACKUP_DIR gefunden" >&2
  exit 1
fi

INCLUDES=()
for f in "${NEWEST[@]}"; do INCLUDES+=("--include=$f"); done

rsync -rt --chmod=D755,F644 -e "$SSH" \
  --delete --delete-excluded "${INCLUDES[@]}" --exclude='*' \
  "$BACKUP_DIR/" "$OFFSITE_HOST:db/"

rsync -rt --chmod=D755,F644 -e "$SSH" --delete \
  ./data/ "$OFFSITE_HOST:uploads/"

echo "$(date '+%F %T') Offsite-Kopie fertig: ${NEWEST[*]} + uploads -> $OFFSITE_HOST"
